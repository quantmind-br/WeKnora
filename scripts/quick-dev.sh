#!/bin/bash
# A one-click script to quickly start the development environment
# This script starts all required services in a single terminal

# Set colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No color

# Get project root directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

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

echo ""
printf "%b\n" "${GREEN}========================================${NC}"
printf "%b\n" "${GREEN}  WeKnora Quick Dev Environment Startup${NC}"
printf "%b\n" "${GREEN}========================================${NC}"
echo ""

# Check whether we're in the project root directory
cd "$PROJECT_ROOT"

# Start infrastructure
log_info "Step 1/3: Starting infrastructure services..."
./scripts/dev.sh start
if [ $? -ne 0 ]; then
    log_error "Failed to start infrastructure"
    exit 1
fi

# Wait for services to be ready
log_info "Waiting for services to finish starting..."
sleep 5

# Ask whether to start the backend
echo ""
log_info "Step 2/3: Starting backend application"
printf "%b" "${YELLOW}Start the backend in this terminal? (y/N): ${NC}"
read -r start_backend

if [ "$start_backend" = "y" ] || [ "$start_backend" = "Y" ]; then
    log_info "Starting backend..."
    # Start the backend in the background
    nohup bash -c 'cd "'$PROJECT_ROOT'" && ./scripts/dev.sh app' > "$PROJECT_ROOT/logs/backend.log" 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > "$PROJECT_ROOT/tmp/backend.pid"
    log_success "Backend started in the background (PID: $BACKEND_PID)"
    log_info "View backend logs: tail -f $PROJECT_ROOT/logs/backend.log"
else
    log_warning "Skipping backend startup"
    log_info "Later, run in a new terminal: make dev-app or ./scripts/dev.sh app"
fi

# Ask whether to start the frontend
echo ""
log_info "Step 3/3: Starting frontend application"
printf "%b" "${YELLOW}Start the frontend in this terminal? (y/N): ${NC}"
read -r start_frontend

if [ "$start_frontend" = "y" ] || [ "$start_frontend" = "Y" ]; then
    log_info "Starting frontend..."
    # Start the frontend in the background
    nohup bash -c 'cd "'$PROJECT_ROOT'/frontend" && npm run dev' > "$PROJECT_ROOT/logs/frontend.log" 2>&1 &
    FRONTEND_PID=$!
    echo $FRONTEND_PID > "$PROJECT_ROOT/tmp/frontend.pid"
    log_success "Frontend started in the background (PID: $FRONTEND_PID)"
    log_info "View frontend logs: tail -f $PROJECT_ROOT/logs/frontend.log"
else
    log_warning "Skipping frontend startup"
    log_info "Later, run in a new terminal: make dev-frontend or ./scripts/dev.sh frontend"
fi

# Show summary
echo ""
printf "%b\n" "${GREEN}========================================${NC}"
printf "%b\n" "${GREEN}  Startup complete!${NC}"
printf "%b\n" "${GREEN}========================================${NC}"
echo ""

log_info "Access URLs:"
echo "- Frontend: http://localhost:5173"
echo "- Backend API: http://localhost:8080"
echo "  - MinIO Console: http://localhost:9001"
echo ""

log_info "Management commands:"
echo "- Check service status: make dev-status"
echo "- View logs: make dev-logs"
echo "- Stop all services: make dev-stop"
echo ""

if [ -f "$PROJECT_ROOT/tmp/backend.pid" ] || [ -f "$PROJECT_ROOT/tmp/frontend.pid" ]; then
    log_warning "Stop background processes:"
    if [ -f "$PROJECT_ROOT/tmp/backend.pid" ]; then
        echo "- Stop backend: kill \$(cat $PROJECT_ROOT/tmp/backend.pid)"
    fi
    if [ -f "$PROJECT_ROOT/tmp/frontend.pid" ]; then
        echo "- Stop frontend: kill \$(cat $PROJECT_ROOT/tmp/frontend.pid)"
    fi
fi

echo ""
log_success "Dev environment is ready, start coding!"
echo ""

