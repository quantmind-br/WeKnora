---
title: Development Guide
tags: [Development & Deployment, Development, Local Development, Environment Setup]
aliases: [Development Environment, DevGuide, Development Guide]
source: Development-Guide.md
---

# Development Guide

## Quick Development Mode (Recommended)

If you need to frequently modify the `app` or `frontend` code, you don't need to rebuild the Docker image every time — you can use local development mode instead.

> For detailed information on development mode, see [Quick Development Mode](../Development-Deployment/Quick-Development-Mode.md)

### Option 1: Using Make Commands (Recommended)

```bash
# Terminal 1: Start infrastructure
make dev-start

# Terminal 2: Start backend
make dev-app

# Terminal 3: Start frontend
make dev-frontend
```

### Option 2: Using Script Commands

```bash
./scripts/dev.sh start    # Start infrastructure
./scripts/dev.sh app       # Start backend
./scripts/dev.sh frontend  # Start frontend
```

## Access URLs

| Service | URL |
|------|------|
| Frontend Dev Server | http://localhost:5173 |
| Backend API | http://localhost:8080 |
| PostgreSQL | localhost:5432 |
| Redis | localhost:6379 |
| MinIO Console | http://localhost:9001 |
| Neo4j Browser | http://localhost:7474 |

> For information on using Neo4j, see [Knowledge Graph](../Core-Features/Knowledge-Graph.md) and [Enabling the Knowledge Graph Feature](../Core-Features/Enabling-Knowledge-Graph.md)

## Backend Hot Reload with Air

```bash
go install github.com/air-verse/air@latest
make dev-app  # Hot reload is enabled automatically when Air is detected
```

## Standalone Frontend Development

```bash
cd frontend
npm run dev  # Connects to the backend API at http://localhost:8080
```

## Production Deployment

```bash
sh scripts/build_images.sh    # Build all images
sh scripts/start_all.sh       # Start the production environment
```

## FAQ

- **Getting a database connection error when starting dev-app**: Make sure you've run `make dev-start` first
- **Frontend CORS errors**: Check the proxy configuration in `vite.config.ts`
- **DocReader needs to be rebuilt**: `sh scripts/build_images.sh -d`

> For more troubleshooting tips, see [FAQ](../Operations-Troubleshooting/FAQ.md)

## Related Topics

- [Quick Development Mode](../Development-Deployment/Quick-Development-Mode.md) — Architecture explanation of development mode
- [Knowledge Graph](../Core-Features/Knowledge-Graph.md) — Neo4j configuration in the development environment
- [Agent Skills System](../Core-Features/Agent-Skills-System.md) — Building the sandbox image

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Quick Development Mode](../Development-Deployment/Quick-Development-Mode.md) — Architecture comparison for development mode
- [Knowledge Graph](../Core-Features/Knowledge-Graph.md) — Starting Neo4j in the development environment
- [Enabling the Knowledge Graph Feature](../Core-Features/Enabling-Knowledge-Graph.md) — Enabling the graph feature in the development environment
- [FAQ](../Operations-Troubleshooting/FAQ.md) — Development-related troubleshooting
