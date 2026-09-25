#!/bin/bash
# This script builds all of WeKnora's Docker images from source

# Set colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No color

# Get the project root directory (parent of the script's directory)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

# Enable BuildKit
export DOCKER_BUILDKIT=1

# Version information
VERSION="1.0.0"
SCRIPT_NAME=$(basename "$0")

# Show help information
show_help() {
    echo -e "${GREEN}WeKnora Image Build Script v${VERSION}${NC}"
    echo -e "${GREEN}Usage:${NC} $0 [options]"
    echo "Options:"
    echo "-h, --help     Show help information"
    echo "-a, --all      Build all images (default)"
    echo "-p, --app      Build only the app image"
    echo "-d, --docreader Build only the docreader image"
    echo "-f, --frontend Build only the frontend image"
    echo "-s, --sandbox  Build only the sandbox image"
    echo "-c, --clean    Clean up all local images"
    echo "-v, --version  Show version information"
    exit 0
}

# Show version information
show_version() {
    echo -e "${GREEN}WeKnora Image Build Script v${VERSION}${NC}"
    exit 0
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if Docker is installed
check_docker() {
    log_info "Checking Docker environment..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed, please install Docker first"
        return 1
    fi
    
    # Check Docker service running status
    if ! docker info &> /dev/null; then
        log_error "Docker service is not running, please start Docker service"
        return 1
    fi
    
    log_success "Docker environment check passed"
    return 0
}

# Detect platform
check_platform() {
    log_info "Detecting system platform information..."
    if [ "$(uname -m)" = "x86_64" ]; then
        export PLATFORM="linux/amd64"
        export TARGETARCH="amd64"
    elif [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then
        export PLATFORM="linux/arm64"
        export TARGETARCH="arm64"
    else
        log_warning "Unrecognized platform type: $(uname -m), using default platform linux/amd64"
        export PLATFORM="linux/amd64"
        export TARGETARCH="amd64"
    fi
    log_info "Current platform: $PLATFORM"
    log_info "Current architecture: $TARGETARCH"
}

# Get version information
get_version_info() {
    # Get version number from VERSION file
    if [ -f "VERSION" ]; then
        VERSION=$(cat VERSION | tr -d '\n\r')
    else
        VERSION="unknown"
    fi
    
    # Get commit ID
    if command -v git >/dev/null 2>&1; then
        COMMIT_ID=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    else
        COMMIT_ID="unknown"
    fi
    
    # Get build time
    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
    
    # Get Go version
    if command -v go >/dev/null 2>&1; then
        GO_VERSION=$(go version 2>/dev/null || echo "unknown")
    else
        GO_VERSION="unknown"
    fi
    
    log_info "Version: $VERSION"
    log_info "Commit ID: $COMMIT_ID"
    log_info "Build time: $BUILD_TIME"
    log_info "Go version: $GO_VERSION"
}

# Build app image
build_app_image() {
    log_info "Build the app image (weknora-app)..."
    
    cd "$PROJECT_ROOT"
    
    # Get version information
    get_version_info
    
    docker build \
        --platform $PLATFORM \
        --build-arg GOPRIVATE_ARG=${GOPRIVATE:-""} \
        --build-arg GOPROXY_ARG=${GOPROXY:-"https://goproxy.cn,direct"} \
        --build-arg GOSUMDB_ARG=${GOSUMDB:-"off"} \
        --build-arg VERSION_ARG="$VERSION" \
        --build-arg COMMIT_ID_ARG="$COMMIT_ID" \
        --build-arg BUILD_TIME_ARG="$BUILD_TIME" \
        --build-arg GO_VERSION_ARG="$GO_VERSION" \
        --build-arg WITH_ANYDOC=${WITH_ANYDOC:-1} \
        -f docker/Dockerfile.app \
        -t wechatopenai/weknora-app:latest \
        .
    
    if [ $? -eq 0 ]; then
        log_success "Application image built successfully"
        return 0
    else
        log_error "Application image build failed"
        return 1
    fi
}

# Build docreader image
build_docreader_image() {
    log_info "Build docreader image (weknora-docreader)..."
    
    cd "$PROJECT_ROOT"
    
    docker build \
        --platform $PLATFORM \
        --build-arg PLATFORM=$PLATFORM \
        --build-arg TARGETARCH=$TARGETARCH \
        --build-arg APT_MIRROR=${APT_MIRROR:-} \
        -f docker/Dockerfile.docreader \
        -t wechatopenai/weknora-docreader:latest \
        .
    
    if [ $? -eq 0 ]; then
        log_success "Docreader image built successfully"
        return 0
    else
        log_error "Docreader image build failed"
        return 1
    fi
}

# Build frontend image (multi-stage: npm runs inside the builder, no host-side prebuilt dist needed)
build_frontend_image() {
    log_info "Build frontend image (weknora-ui)..."
    
    cd "$PROJECT_ROOT"
    
    # Get version information (for injecting frontend commit hash)
    get_version_info

    docker build \
        --platform $PLATFORM \
        --build-arg VITE_FRONTEND_COMMIT="$COMMIT_ID" \
        ${NPM_REGISTRY:+--build-arg NPM_REGISTRY="$NPM_REGISTRY"} \
        ${NODE_MAX_OLD_SPACE_SIZE:+--build-arg NODE_MAX_OLD_SPACE_SIZE="$NODE_MAX_OLD_SPACE_SIZE"} \
        -f frontend/Dockerfile \
        -t wechatopenai/weknora-ui:latest \
        frontend/
    
    if [ $? -eq 0 ]; then
        log_success "Frontend image built successfully"
        return 0
    else
        log_error "Frontend image build failed"
        return 1
    fi
}

# Build sandbox image
build_sandbox_image() {
    log_info "Build sandbox image (weknora-sandbox)..."

    cd "$PROJECT_ROOT"

    # Also tag main: the default image tracks main (see DefaultDockerImage in sandbox.go),
    # and the Docker backend only pulls when the image is missing locally, so building without this tag is wasted work.
    docker build \
        --platform $PLATFORM \
        --build-arg TARGETPLATFORM=$PLATFORM \
        -f docker/Dockerfile.sandbox \
        --target sandbox \
        -t wechatopenai/weknora-sandbox:latest \
        -t wechatopenai/weknora-sandbox:main \
        .

    if [ $? -ne 0 ]; then
        log_error "Sandbox image build failed"
        return 1
    fi

    # Cube builds templates directly from the image and probes :49983/health, which always fails without envd,
    # so Cube uses a variant image with envd injected. See website-docs/06-development/04-sandbox-deployment.md.
    # Pinned to linux/amd64: cubesandbox-base, the image envd comes from, does not publish arm64.
    log_info "Build sandbox image Cube variant (weknora-sandbox:main-cube)..."

    docker build \
        --platform linux/amd64 \
        --build-arg TARGETPLATFORM=linux/amd64 \
        --build-arg TARGETARCH=amd64 \
        -f docker/Dockerfile.sandbox \
        --target cube \
        -t wechatopenai/weknora-sandbox:latest-cube \
        -t wechatopenai/weknora-sandbox:main-cube \
        .

    if [ $? -ne 0 ]; then
        log_error "Sandbox image Cube variant build failed"
        return 1
    fi

    # Desktop variant: XFCE + x11vnc + websockify. Tagged for E2B template
    # builds; the Docker backend does not consume this image yet.
    log_info "Build sandbox image desktop variant (weknora-sandbox:main-desktop)..."

    docker build \
        --platform $PLATFORM \
        --build-arg TARGETPLATFORM=$PLATFORM \
        -f docker/Dockerfile.sandbox \
        --target desktop \
        -t wechatopenai/weknora-sandbox:latest-desktop \
        -t wechatopenai/weknora-sandbox:main-desktop \
        .

    if [ $? -ne 0 ]; then
        log_error "Sandbox image desktop variant build failed"
        return 1
    fi

    log_info "Build sandbox image desktop Cube variant (weknora-sandbox:main-desktop-cube)..."

    docker build \
        --platform linux/amd64 \
        --build-arg TARGETPLATFORM=linux/amd64 \
        --build-arg TARGETARCH=amd64 \
        -f docker/Dockerfile.sandbox \
        --target desktop-cube \
        -t wechatopenai/weknora-sandbox:latest-desktop-cube \
        -t wechatopenai/weknora-sandbox:main-desktop-cube \
        .

    if [ $? -eq 0 ]; then
        log_success "Sandbox image built successfully"
        return 0
    else
        log_error "Sandbox image desktop Cube variant build failed"
        return 1
    fi
}

# Build all images
build_all_images() {
    log_info "Starting build of all images..."

    local app_result=0
    local docreader_result=0
    local frontend_result=0
    local sandbox_result=0

    # Build application image
    build_app_image
    app_result=$?

    # Build docreader image
    build_docreader_image
    docreader_result=$?

    # Build frontend image
    build_frontend_image
    frontend_result=$?

    # Build sandbox image
    build_sandbox_image
    sandbox_result=$?

    # Show build results
    echo ""
    log_info "=== Build Results ==="
    if [ $app_result -eq 0 ]; then
        log_success "✓ Application image built successfully"
    else
        log_error "✗ Application image build failed"
    fi

    if [ $docreader_result -eq 0 ]; then
        log_success "✓ Docreader image built successfully"
    else
        log_error "✗ Docreader image build failed"
    fi

    if [ $frontend_result -eq 0 ]; then
        log_success "✓ Frontend image built successfully"
    else
        log_error "✗ Frontend image build failed"
    fi

    if [ $sandbox_result -eq 0 ]; then
        log_success "✓ Sandbox image built successfully"
    else
        log_error "✗ Sandbox image build failed"
    fi

    if [ $app_result -eq 0 ] && [ $docreader_result -eq 0 ] && [ $frontend_result -eq 0 ] && [ $sandbox_result -eq 0 ]; then
        log_success "All images built successfully!"
        return 0
    else
        log_error "Some image builds failed"
        return 1
    fi
}

# Clean up local images
clean_images() {
    log_info "Cleaning up local WeKnora images..."
    
    # Stop related containers
    log_info "Stopping related containers..."
    docker stop $(docker ps -q --filter "ancestor=wechatopenai/weknora-app:latest" 2>/dev/null) 2>/dev/null || true
    docker stop $(docker ps -q --filter "ancestor=wechatopenai/weknora-docreader:latest" 2>/dev/null) 2>/dev/null || true
    docker stop $(docker ps -q --filter "ancestor=wechatopenai/weknora-ui:latest" 2>/dev/null) 2>/dev/null || true
    
    # Delete related containers
    log_info "Deleting related containers..."
    docker rm $(docker ps -aq --filter "ancestor=wechatopenai/weknora-app:latest" 2>/dev/null) 2>/dev/null || true
    docker rm $(docker ps -aq --filter "ancestor=wechatopenai/weknora-docreader:latest" 2>/dev/null) 2>/dev/null || true
    docker rm $(docker ps -aq --filter "ancestor=wechatopenai/weknora-ui:latest" 2>/dev/null) 2>/dev/null || true
    
    # Delete image
    log_info "Deleting local image..."
    docker rmi wechatopenai/weknora-app:latest 2>/dev/null || true
    docker rmi wechatopenai/weknora-docreader:latest 2>/dev/null || true
    docker rmi wechatopenai/weknora-ui:latest 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:latest 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:latest-cube 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:latest-desktop 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:latest-desktop-cube 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:main 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:main-cube 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:main-desktop 2>/dev/null || true
    docker rmi wechatopenai/weknora-sandbox:main-desktop-cube 2>/dev/null || true
    
    docker image prune -f
    
    log_success "Image cleanup complete"
    return 0
}

# Parse command-line arguments
BUILD_ALL=false
BUILD_APP=false
BUILD_DOCREADER=false
BUILD_FRONTEND=false
BUILD_SANDBOX=false
CLEAN_IMAGES=false

# Default to building all images when no arguments are given
if [ $# -eq 0 ]; then
    BUILD_ALL=true
fi

while [ "$1" != "" ]; do
    case $1 in
        -h | --help )       show_help
                            ;;
        -a | --all )        BUILD_ALL=true
                            ;;
        -p | --app )        BUILD_APP=true
                            ;;
        -d | --docreader )  BUILD_DOCREADER=true
                            ;;
        -f | --frontend )   BUILD_FRONTEND=true
                            ;;
        -s | --sandbox )    BUILD_SANDBOX=true
                            ;;
        -c | --clean )      CLEAN_IMAGES=true
                            ;;
        -v | --version )    show_version
                            ;;
        * )                 log_error "Unknown option: $1"
                            show_help
                            ;;
    esac
    shift
done

# Check Docker environment
check_docker
if [ $? -ne 0 ]; then
    exit 1
fi

# Detect platform
check_platform

# Perform cleanup operation
if [ "$CLEAN_IMAGES" = true ]; then
    clean_images
    exit $?
fi

# Perform build operation
if [ "$BUILD_ALL" = true ]; then
    build_all_images
    exit $?
fi

if [ "$BUILD_APP" = true ]; then
    build_app_image
    exit $?
fi

if [ "$BUILD_DOCREADER" = true ]; then
    build_docreader_image
    exit $?
fi

if [ "$BUILD_FRONTEND" = true ]; then
    build_frontend_image
    exit $?
fi

if [ "$BUILD_SANDBOX" = true ]; then
    build_sandbox_image
    exit $?
fi

exit 0
