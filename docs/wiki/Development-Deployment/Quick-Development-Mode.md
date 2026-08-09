---
title: Quick Development Mode
tags: [Development & Deployment, Development, Hot Reload, Air]
aliases: [Quick Dev, QuickDev, dev mode]
source: Quick-Development-Mode.md
---

# Quick Development Mode

Solves the problem where every change to the `app` (backend) or `frontend` (frontend) code during development requires rebuilding the Docker image, by enabling hot reload for both modules.

> This page supplements the [Development Guide](../Development-Deployment/Development-Guide.md), providing more detailed architectural explanations

## Usage

### Method 1: Make Commands (Recommended)

```bash
make dev-start       # Terminal 1: start infrastructure
make dev-app         # Terminal 2: start backend
make dev-frontend    # Terminal 3: start frontend
```

### Method 2: Development Script

```bash
./scripts/dev.sh start      # Terminal 1
./scripts/dev.sh app         # Terminal 2
./scripts/dev.sh frontend    # Terminal 3
```

### Method 3: One-Click Start

```bash
./scripts/quick-dev.sh
```

### Using Air for Backend Hot Reload

```bash
go install github.com/air-verse/air@latest
make dev-app  # Automatically detects and uses Air
```

## Architecture Comparison

### Development Mode

```
Local Backend App (:8080) ← Local Frontend UI (:5173) → Docker Infrastructure Containers
                                              (PostgreSQL/Redis/MinIO/Neo4j/DocReader)
```

- Backend/frontend run directly on the local machine, changes take effect within seconds
- Infrastructure still uses Docker containers

### Production Mode

```
Containerized Backend App (:8080) ← Containerized Frontend UI (:80) → Infrastructure Containers
```

- All services run inside Docker containers
- Every change requires rebuilding the image

## Efficiency Comparison

| Method | Time per Change |
|------|------------|
| Old Method (rebuild image) | 2-5 minutes |
| New Method (local development) | Backend 5-10 seconds, frontend real-time hot reload |

## Related Topics

- [Development Guide](../Development-Deployment/Development-Guide.md) — Complete guide to setting up the development environment
- [Frequently Asked Questions](../Operations-Troubleshooting/FAQ.md) — Troubleshooting for development mode-related issues

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Development Guide](../Development-Deployment/Development-Guide.md) — Complete development guide (this page's supplementary explanation)
- [Frequently Asked Questions](../Operations-Troubleshooting/FAQ.md) — Troubleshooting for development mode
