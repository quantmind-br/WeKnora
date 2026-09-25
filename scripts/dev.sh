#!/bin/bash
# Development startup script — only starts infrastructure; app and frontend must be run locally by hand

# Set colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No color

# Get project root directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Logging function
log_info() {
    printf "%b\n" "${BLUE}[INFO]${NC} $1"
}

log_success() {
    printf "%b\n" "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    printf "%b\n" "${RED}[ERROR]${NC} $1"
}

log_warning() {
    printf "%b\n" "${YELLOW}[WARNING]${NC} $1"
}

# Select an available Docker Compose command
DOCKER_COMPOSE_BIN=""
DOCKER_COMPOSE_SUBCMD=""

detect_compose_cmd() {
    if docker compose version &> /dev/null; then
        DOCKER_COMPOSE_BIN="docker"
        DOCKER_COMPOSE_SUBCMD="compose"
        return 0
    fi
    if command -v docker-compose &> /dev/null; then
        if docker-compose version &> /dev/null; then
            DOCKER_COMPOSE_BIN="docker-compose"
            DOCKER_COMPOSE_SUBCMD=""
            return 0
        fi
    fi
    return 1
}

# Show help message
show_help() {
    printf "%b\n" "${GREEN}WeKnora Development Environment Script${NC}"
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "start      Start infrastructure services (postgres, redis, docreader, langfuse)"
    echo "stop       Stop all services"
    echo "restart    Restart all services"
    echo "logs       View service logs"
    echo "status     View service status"
    echo "app        Start the backend app (run locally)"
    echo "frontend   Start the frontend dev server (run locally)"
    echo "help       Show this help message"
    echo ""
    echo "Optional profiles (for the start command):"
    echo "--minio       Start MinIO object storage"
    echo "--qdrant      Start the Qdrant vector database"
    echo "--neo4j       Start the Neo4j graph database"
    echo "--dex         Start Dex (OIDC authentication)"
    echo "--langfuse    Start Langfuse (enabled by default)"
    echo "--no-langfuse Don't start Langfuse"
    echo "--odl-hybrid  Start OpenDataLoader hybrid (Docling, larger image, enable as needed)"
    echo "--full        Start all optional services (excludes odl-hybrid, add --odl-hybrid separately)"
    echo ""
    echo "Examples:"
    echo "  $0 start                    # Start base services"
    echo "  $0 start --qdrant           # Start base services + Qdrant"
    echo "  $0 start --dex             # Start base services + Dex"
    echo "  $0 start --odl-hybrid       # Start base services + OpenDataLoader hybrid"
    echo "  $0 start --full             # Start all services"
    echo "  make dev-start DEV_ARGS=--odl-hybrid   # Same as above (Makefile argument passing)"
    echo "  $0 app                      # Start the backend in another terminal"
    echo "  $0 frontend                 # Start the frontend in another terminal"
}

# Load .env and optional .env.local (the latter overrides the former)
# Strip trailing \r on read, for compatibility with Windows-style (CRLF) line endings,
# otherwise bash source would treat a stray \r as part of a command, causing "...: $'\r': command not found".
# Note: cannot use source <(sed ...) — macOS's built-in Bash 3.2 doesn't support process substitution
# Sourcing does not import variables into the current shell; must be redirected to a seekable temp file before sourcing.
_source_env_file() {
    local src="$1"
    local tmp
    tmp="$(mktemp)" || return 1
    sed -e 's/\r$//' "$src" > "$tmp"
    set -a
    # shellcheck source=/dev/null
    source "$tmp"
    set +a
    rm -f "$tmp"
}

load_env_files() {
    if [ -f ".env" ]; then
        _source_env_file .env || return 1
    else
        return 1
    fi

    if [ -f ".env.local" ]; then
        log_info "Loading .env.local overrides..."
        _source_env_file .env.local || return 1
    fi
    return 0
}

# Checking Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed; please install Docker first"
        return 1
    fi
    
    if ! detect_compose_cmd; then
        log_error "Docker Compose not detected"
        return 1
    fi
    
    if ! docker info &> /dev/null; then
        log_error "Docker service is not running"
        return 1
    fi
    
    return 0
}

# Checking whether .env has hybrid mode enabled (used to rebuild docreader after --odl-hybrid startup)
_should_enable_odl_hybrid_from_env() {
    local hybrid="${DOCREADER_ODL_HYBRID:-off}"
    hybrid=$(printf '%s' "$hybrid" | tr '[:upper:]' '[:lower:]' | tr -d '[:space:]')
    case "$hybrid" in
        off|"") return 1 ;;
        *) return 0 ;;
    esac
}

_enable_odl_hybrid_profile() {
    PROFILES="$PROFILES --profile odl-hybrid"
    ENABLED_SERVICES="$ENABLED_SERVICES odl-hybrid"
}

# Waiting for odl-hybrid HTTP health check to pass (services may still be pulling dependencies after compose startup)
_wait_odl_hybrid_ready() {
    local port="${ODL_HYBRID_PORT:-5002}"
    local max_wait="${ODL_HYBRID_STARTUP_WAIT_SEC:-180}"
    local waited=0
    local interval=5

    if ! command -v curl &> /dev/null; then
        log_warning "curl is not installed, skipping odl-hybrid readiness wait; please check http://localhost:${port}/health manually"
        return 0
    fi

    log_info "Waiting for odl-hybrid to become ready (up to ${max_wait}s, image build required on first run: docker compose ... build odl-hybrid)..."
    while [ "$waited" -lt "$max_wait" ]; do
        if curl -sf "http://127.0.0.1:${port}/health" >/dev/null 2>&1; then
            log_success "odl-hybrid is ready (http://localhost:${port}/health)"
            return 0
        fi
        sleep "$interval"
        waited=$((waited + interval))
    done
    log_warning "odl-hybrid did not become ready within ${max_wait}s; please check: docker logs WeKnora-odl-hybrid"
    return 1
}

# Starting infrastructure services
start_services() {
    log_info "Starting dev environment infrastructure services..."
    
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"
    
    # Checking .env file
    if [ ! -f ".env" ]; then
        log_error ".env file does not exist; please create it first"
        return 1
    fi

    load_env_files
    if [ $? -ne 0 ]; then
        log_error ".env file does not exist; please create it first"
        return 1
    fi

    if [ -n "${DEV_REMOTE_HOST:-}" ]; then
        log_warning "DEV_REMOTE_HOST=${DEV_REMOTE_HOST} is configured, skipping local Docker infrastructure startup"
        log_info "Remote services: PostgreSQL/Redis/DocReader/Langfuse → ${DEV_REMOTE_HOST}"
        log_info "Next: make dev-app (local backend) or make dev-frontend (frontend)"
        return 0
    fi
    
    # Parsing profile arguments
    shift  # Removing the "start" command itself
    # By default, start infrastructure (postgres / redis / docreader) + langfuse,
    # other optional services are enabled on demand via --minio / --qdrant / --neo4j / --dex / --full.
    PROFILES="--profile langfuse"
    ENABLED_SERVICES="langfuse"
    while [ $# -gt 0 ]; do
        case "$1" in
            --minio)
                PROFILES="$PROFILES --profile minio"
                ENABLED_SERVICES="$ENABLED_SERVICES minio"
                ;;
            --qdrant)
                PROFILES="$PROFILES --profile qdrant"
                ENABLED_SERVICES="$ENABLED_SERVICES qdrant"
                ;;
            --neo4j)
                PROFILES="$PROFILES --profile neo4j"
                ENABLED_SERVICES="$ENABLED_SERVICES neo4j"
                ;;
            --dex)
                PROFILES="$PROFILES --profile dex"
                ENABLED_SERVICES="$ENABLED_SERVICES dex"
                ;;
            --langfuse)
                PROFILES="$PROFILES --profile langfuse"
                ENABLED_SERVICES="$ENABLED_SERVICES langfuse"
                ;;
            --no-langfuse)
                PROFILES="${PROFILES//--profile langfuse/}"
                ENABLED_SERVICES="${ENABLED_SERVICES//langfuse/}"
                ;;
            --odl-hybrid)
                if [[ "$ENABLED_SERVICES" != *"odl-hybrid"* ]]; then
                    _enable_odl_hybrid_profile
                fi
                ;;
            --full)
                PROFILES="--profile full"
                ENABLED_SERVICES="minio qdrant neo4j dex"
                break
                ;;
            *)
                log_warning "Unknown argument: $1"
                ;;
        esac
        shift
    done

    # Starting services (odl-hybrid built separately with --build, to avoid rebuilding docreader every time)
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml $PROFILES up -d
    local compose_rc=$?
    if [ "$compose_rc" -eq 0 ] && [[ "$ENABLED_SERVICES" == *"odl-hybrid"* ]]; then
        log_info "Building/updating odl-hybrid image..."
        "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml $PROFILES up -d --build odl-hybrid
        _wait_odl_hybrid_ready || true
        # docreader needs to read DOCREADER_ODL_HYBRID; if .env was just changed, force a rebuild to inject the environment variable
        if _should_enable_odl_hybrid_from_env; then
            log_info "Rebuilding docreader to apply DOCREADER_ODL_HYBRID=${DOCREADER_ODL_HYBRID} ..."
            "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml up -d --force-recreate docreader
        fi
    fi

    if [ "$compose_rc" -eq 0 ]; then
        log_success "Infrastructure services started"
        echo ""
        log_info "Service access URLs:"
        echo "  - PostgreSQL:    localhost:5432"
        echo "  - Redis:         localhost:6379"
        echo "  - DocReader:     localhost:50051"
        
        # Show additional services based on enabled profiles
        if [[ "$ENABLED_SERVICES" == *"minio"* ]]; then
            echo "  - MinIO:         localhost:9000 (Console: localhost:9001)"
        fi
        if [[ "$ENABLED_SERVICES" == *"qdrant"* ]]; then
            echo "  - Qdrant:        localhost:6333 (gRPC: localhost:6334)"
        fi
        if [[ "$ENABLED_SERVICES" == *"neo4j"* ]]; then
            echo "  - Neo4j:         localhost:7474 (Bolt: localhost:7687)"
        fi
        if [[ "$ENABLED_SERVICES" == *"dex"* ]]; then
            echo "  - Dex:           localhost:5556"
        fi
        if [[ "$ENABLED_SERVICES" == *"langfuse"* ]]; then
            echo "  - Langfuse:      http://localhost:${LANGFUSE_WEB_PORT:-3000}"
        fi
        if [[ "$ENABLED_SERVICES" == *"odl-hybrid"* ]]; then
            echo "  - ODL Hybrid:    http://localhost:${ODL_HYBRID_PORT:-5002} (health: /health)"
            echo "docreader requires DOCREADER_ODL_HYBRID=docling-fast"
        fi
        
        echo ""
        log_info "Next steps:"
        printf "%b\n" "${YELLOW}1. Run the backend in a new terminal:${NC} make dev-app"
        printf "%b\n" "${YELLOW}2. Run the frontend in a new terminal:${NC} make dev-frontend"
        return 0
    else
        log_error "Service startup failed"
        return 1
    fi
}

# Stopping services
stop_services() {
    log_info "Stopping dev environment services..."
    
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi
    
    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml down
    
    if [ $? -eq 0 ]; then
        log_success "All services stopped"
        return 0
    else
        log_error "Service stop failed"
        return 1
    fi
}

# Restart service
restart_services() {
    stop_services
    sleep 2
    start_services
}

# View logs
show_logs() {
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml logs -f
}

# View status
show_status() {
    check_docker
    if [ $? -ne 0 ]; then
        return 1
    fi

    cd "$PROJECT_ROOT"
    "$DOCKER_COMPOSE_BIN" $DOCKER_COMPOSE_SUBCMD -f docker-compose.dev.yml ps
}

# Check whether infrastructure ports are reachable in remote dev mode
check_remote_dev_connectivity() {
    local host="${DEV_REMOTE_HOST:-}"
    if [ -z "$host" ]; then
        return 0
    fi

    local db_port="${DB_PORT:-5432}"
    local redis_port
    redis_port="${REDIS_ADDR#*:}"
    if [ "$redis_port" = "$REDIS_ADDR" ]; then
        redis_port=6379
    fi
    local docreader_port="${DOCREADER_PORT:-50051}"

    log_info "Checking remote infrastructure connectivity (${host})..."
    local failed=0
    for spec in "PostgreSQL:${host}:${db_port}" "Redis:${host}:${redis_port}" "DocReader:${host}:${docreader_port}"; do
        local name="${spec%%:*}"
        local rest="${spec#*:}"
        local h="${rest%%:*}"
        local p="${rest##*:}"
        if command -v nc &> /dev/null; then
            if nc -z -w 3 "$h" "$p" 2>/dev/null; then
                log_success "${name} ${h}:${p} reachable"
            else
                log_error "${name} ${h}:${p} unreachable (no route / connection refused)"
                failed=1
            fi
        else
            log_warning "nc not installed, skipping ${name} connectivity check"
        fi
    done

    if [ "$failed" -ne 0 ]; then
        echo ""
        log_error "Unable to connect to remote dev environment ${host}"
        log_info "Troubleshooting suggestions:"
        echo "Confirm the Docker containers are running on the remote machine (postgres/redis/docreader)"
        echo "Confirm this machine and ${host} are on the same LAN (local: $(ipconfig getifaddr en0 2>/dev/null || echo 'unknown'))"
        echo "Check port mappings on the remote: docker ps --format 'table {{.Names}}\t{{.Ports}}'"
        echo "Check whether the remote firewall allows 5432/6379/50051"
        return 1
    fi
    return 0
}

# Host-platform path of the anydoc static archive (built by `make anydoc-lib`).
anydoc_host_archive() {
    case "$(uname -s)-$(uname -m)" in
        Darwin-arm64) echo "$PROJECT_ROOT/third_party/anydoc-go/lib/darwin_arm64/libanydoc_go.a" ;;
        Darwin-x86_64) echo "$PROJECT_ROOT/third_party/anydoc-go/lib/darwin_amd64/libanydoc_go.a" ;;
        Linux-x86_64)
            if [ -f "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_amd64_gnu/libanydoc_go.a" ]; then
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_amd64_gnu/libanydoc_go.a"
            else
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_amd64_musl/libanydoc_go.a"
            fi
            ;;
        Linux-aarch64)
            if [ -f "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_arm64_gnu/libanydoc_go.a" ]; then
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_arm64_gnu/libanydoc_go.a"
            else
                echo "$PROJECT_ROOT/third_party/anydoc-go/lib/linux_arm64_musl/libanydoc_go.a"
            fi
            ;;
        *) echo "" ;;
    esac
}

# Enable the in-process anydoc engine when the archive is present, unless the
# caller already set GO_BUILD_TAGS (including empty, which opts out).
enable_anydoc_build_tag() {
    if [ -n "${GO_BUILD_TAGS+x}" ]; then
        export GO_BUILD_TAGS
        return
    fi
    local archive
    archive="$(anydoc_host_archive)"
    if [ -n "$archive" ] && [ -f "$archive" ]; then
        export GO_BUILD_TAGS=anydoc
        log_info "anydoc static library detected, enabled -tags anydoc"
    else
        log_info "anydoc static library not detected, parsing engine unavailable. Run first when needed: make anydoc-lib"
    fi
}

# Start backend app (local)
start_app() {
    log_info "Starting backend app (local dev mode)..."
    
    cd "$PROJECT_ROOT"
    
    # Check whether Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        return 1
    fi
    
    log_info "Loading environment configuration..."
    if ! load_env_files; then
        log_error ".env file not found, please create the config file first"
        return 1
    fi
    
    # Local docker-compose.dev mode: map container service names to the host loopback address
    # Remote dev mode (DEV_REMOTE_HOST or .env.local set): keep values from .env/.env.local
    if [ -n "${DEV_REMOTE_HOST:-}" ]; then
        log_info "Remote dev mode: infrastructure → ${DEV_REMOTE_HOST}"
        export DB_HOST="${DB_HOST:-$DEV_REMOTE_HOST}"
        export REDIS_ADDR="${REDIS_ADDR:-$DEV_REMOTE_HOST:6379}"
        export DOCREADER_ADDR="${DOCREADER_ADDR:-$DEV_REMOTE_HOST:50051}"
        export MINIO_ENDPOINT="${MINIO_ENDPOINT:-$DEV_REMOTE_HOST:9000}"
        export MILVUS_ADDRESS="${MILVUS_ADDRESS:-$DEV_REMOTE_HOST:19530}"
        export NEO4J_URI="${NEO4J_URI:-bolt://$DEV_REMOTE_HOST:7687}"
        export QDRANT_HOST="${QDRANT_HOST:-$DEV_REMOTE_HOST}"
        if [ -z "${LANGFUSE_HOST:-}" ] || [ "$LANGFUSE_HOST" = "http://langfuse-web:3000" ]; then
            export LANGFUSE_HOST="http://${DEV_REMOTE_HOST}:3000"
        fi
    else
        export DB_HOST=127.0.0.1
        export DOCREADER_ADDR=127.0.0.1:50051
        export MINIO_ENDPOINT=127.0.0.1:9000
        export REDIS_ADDR=127.0.0.1:6379
        export MILVUS_ADDRESS=127.0.0.1:19530
        export NEO4J_URI=bolt://127.0.0.1:7687
        export QDRANT_HOST=127.0.0.1
    fi
    export DOCREADER_TRANSPORT="${DOCREADER_TRANSPORT:-grpc}"

    if ! check_remote_dev_connectivity; then
        return 1
    fi

    # .env.example uses /data/files for the Docker app container, where a
    # volume is mounted at that path. When the backend runs directly on the
    # host via dev-app, /data is often read-only or missing, so use a repo-local
    # writable directory unless the developer explicitly configured another
    # local storage path.
    if [ -z "${LOCAL_STORAGE_BASE_DIR:-}" ] || [ "$LOCAL_STORAGE_BASE_DIR" = "/data/files" ]; then
        export LOCAL_STORAGE_BASE_DIR="$PROJECT_ROOT/.local-data/files"
    fi
    mkdir -p "$LOCAL_STORAGE_BASE_DIR"
    
    # Ensure required environment variables are set
    if [ -z "$DB_DRIVER" ]; then
        log_error "DB_DRIVER environment variable not set, please check the .env file"
        return 1
    fi
    
    log_info "Environment variables set, starting app..."
    log_info "Database address: $DB_HOST:${DB_PORT:-5432}"
    
    export CGO_CFLAGS="-Wno-deprecated-declarations -Wno-gnu-folding-constant"
    if [[ "$(uname)" == "Darwin" ]]; then
      export CGO_LDFLAGS="-Wl,-no_warn_duplicate_libraries"
    fi

    enable_anydoc_build_tag

    # Check whether Air (hot reload tool) is installed
    if command -v air &> /dev/null; then
        log_success "Air detected, starting in hot reload mode..."
        log_info "Will automatically recompile and restart after Go code changes"
        air
    else
        log_info "Air not detected, starting in normal mode"
        log_warning "Tip: install Air to enable automatic restart on code changes"
        log_info "Install command: go install github.com/air-verse/air@latest"
        LDFLAGS="$(./scripts/get_version.sh ldflags) -X 'google.golang.org/protobuf/reflect/protoregistry.conflictPolicy=warn'"
        go run -tags "${GO_BUILD_TAGS:-}" -ldflags="$LDFLAGS" ./cmd/server
    fi
}

# Start frontend (local)
start_frontend() {
    log_info "Starting frontend dev server..."

    cd "$PROJECT_ROOT"
    if [ -f ".env" ] || [ -f ".env.local" ]; then
        load_env_files >/dev/null 2>&1 || true
    fi
    
    cd "$PROJECT_ROOT/frontend"
    
    # Check whether npm is installed
    if ! command -v npm &> /dev/null; then
        log_error "npm is not installed"
        return 1
    fi
    
    # Check whether dependencies are installed
    if [ ! -d "node_modules" ]; then
        log_warning "node_modules not found, installing dependencies..."
        npm install
    fi
    
    log_info "Starting Vite dev server..."
    log_info "Frontend will run at http://localhost:5173"
    log_info "Frontend API proxy target: ${VITE_DEV_PROXY_TARGET:-${FRONTEND_BACKEND_URL:-http://localhost:8080}}"
    
    # Run dev server
    npm run dev
}

# Parse command
CMD="${1:-help}"
case "$CMD" in
    start)
        start_services "$@"
        ;;
    stop)
        stop_services
        ;;
    restart)
        restart_services
        ;;
    logs)
        show_logs
        ;;
    status)
        show_status
        ;;
    app)
        start_app
        ;;
    frontend)
        start_frontend
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $CMD"
        show_help
        exit 1
        ;;
esac

exit 0
