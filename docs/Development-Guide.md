# WeKnora Development Guide

## Fast Development Mode (Recommended)

If you need to frequently modify `app` or `frontend` code, **you don't need to rebuild the Docker image every time** — you can use local development mode instead.

### Option 1: Using Make Commands (Recommended)

#### 1. Start Infrastructure Services

```bash
make dev-start
```

This will start Docker containers for the following services:
- PostgreSQL (database)
- Redis (cache)
- MinIO (object storage)
- Neo4j (graph database)
- DocReader (document reading service)

#### 2. Start the Backend Application (new terminal)

```bash
make dev-app
```

This runs the Go application directly on your local machine. After modifying the code, press Ctrl+C to stop and rerun to apply changes.

#### 3. Start the Frontend (new terminal)

```bash
make dev-frontend
```

This starts the Vite dev server, which supports hot reload and automatically refreshes after code changes.

#### 4. Check Service Status

```bash
make dev-status
```

#### 5. Stop All Services

```bash
make dev-stop
```

### Option 2: Using Script Commands

If you prefer to use scripts directly:

```bash
# Start infrastructure
./scripts/dev.sh start

# Start the backend (new terminal)
./scripts/dev.sh app

# Start the frontend (new terminal)
./scripts/dev.sh frontend

# View logs
./scripts/dev.sh logs

# Stop all services
./scripts/dev.sh stop
```

## Access URLs

### Development Environment

- **Frontend dev server**: http://localhost:5173
- **Backend API**: http://localhost:8080
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379
- **MinIO Console**: http://localhost:9001
- **Neo4j Browser**: http://localhost:7474

## Development Workflow Comparison

### ❌ Old Way (Slow)

```bash
# Every time you modify code, you need to:
sh scripts/build_images.sh -p      # Rebuild the image (very slow)
sh scripts/start_all.sh --no-pull  # Restart containers
```

**Time cost**: 2-5 minutes per change

### ✅ New Way (Fast)

```bash
# First-time startup (only needed once):
make dev-start

# In two other terminals, run respectively:
make dev-app       # After modifying Go code, Ctrl+C and restart (seconds)
make dev-frontend  # Frontend code changes hot-reload automatically (no restart needed)
```

**Time cost**:
- First startup: 1-2 minutes
- Subsequent backend changes: 5-10 seconds (restart the Go app)
- Subsequent frontend changes: real-time hot reload

## Using Air for Backend Hot Reload (Optional)

If you want the backend to automatically restart after code changes, you can install `air`:

```bash
go install github.com/air-verse/air@latest
```

The project root already includes a built-in `.air.toml`, so there's no need to manually create a config file. `make dev-app` automatically detects whether Air is installed on your machine: if Air is detected, it uses hot reload mode; if not, it falls back to plain `go run` mode.

## Other Development Tips

### Modifying Only the Frontend

If you're only modifying the frontend, you just need:

```bash
cd frontend
npm run dev
```

The frontend will connect to the backend API at http://localhost:8080.

### Modifying Only the Backend

If you're only modifying the backend, you just need:

```bash
# Start infrastructure
make dev-start

# Run the backend
make dev-app
```

### Debug Mode

#### Backend Debugging

Use the debugging features of VS Code or GoLand, configured to connect to the locally running Go application.

Example VS Code configuration (`.vscode/launch.json`):

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Launch Server",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}/cmd/server",
            "env": {
                "DB_HOST": "localhost",
                "DOCREADER_ADDR": "localhost:50051",
                "MINIO_ENDPOINT": "localhost:9000",
                "REDIS_ADDR": "localhost:6379",
                "NEO4J_URI": "bolt://localhost:7687"
            },
            "args": []
        }
    ]
}
```

#### Frontend Debugging

Simply use the browser's developer tools — Vite provides source maps.

## Production Deployment

Once you've finished development and need to deploy, that's when you build the images:

```bash
# Build all images
sh scripts/build_images.sh

# Or build a specific image only
sh scripts/build_images.sh -p  # Build only the backend
sh scripts/build_images.sh -f  # Build only the frontend
sh scripts/build_images.sh -d  # Build only the doc reader
sh scripts/build_images.sh -s  # Build only the sandbox image (Agent Skills execution environment)

# Start the production environment
sh scripts/start_all.sh
```

## FAQ

### Q: I get a database connection error when starting dev-app

A: Make sure you've run `make dev-start` first, and wait for all services to finish starting (about 30 seconds).

### Q: I get a CORS error when the frontend accesses the API

A: Check the frontend's proxy configuration and make sure `vite.config.ts` has the correct proxy settings.

### Q: What if the DocReader service needs to be rebuilt?

A: DocReader still uses a Docker image, so if you need to make changes, you'll need to rebuild it:

```bash
sh scripts/build_images.sh -d
make dev-restart
```

## Summary

- **Day-to-day development**: use the `make dev-*` commands for fast iteration
- **Integration testing**: use `sh scripts/start_all.sh --no-pull` to test the full environment
- **Production deployment**: use `sh scripts/build_images.sh` + `sh scripts/start_all.sh`
