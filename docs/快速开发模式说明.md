# Quick Development Mode Guide

Solves the problem where every code change to `app` (backend) or `frontend` requires rebuilding a Docker image, enabling hot reload for both modules.

## 🚀 Usage

### Method 1: Using Make Commands (Recommended)

```bash
# Terminal 1: Start infrastructure
make dev-start

# Terminal 2: Start backend
make dev-app

# Terminal 3: Start frontend
make dev-frontend
```

### Method 2: Using the Development Script

```bash
# Terminal 1
./scripts/dev.sh start

# Terminal 2
./scripts/dev.sh app

# Terminal 3
./scripts/dev.sh frontend
```

### Method 3: One-Click Start (Interactive)

```bash
./scripts/quick-dev.sh
```

### Using Air for Backend Hot Reload

After installing Air, backend code changes will automatically recompile and restart:

```bash
# Install Air
go install github.com/air-verse/air@latest

# Make sure it's in PATH
export PATH=$PATH:$(go env GOPATH)/bin

# Start with Air (auto-detected)
make dev-app
```

## 🔄 Architecture Overview

### Development Mode Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Local Development Environment          │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────┐         ┌──────────┐                     │
│  │ Backend App│◄────────┤ Frontend UI│                     │
│  │ (running   │         │ (running   │                     │
│  │  locally)  │         │  locally)  │                     │
│  │  :8080   │         │  :5173   │                     │
│  └────┬─────┘         └──────────┘                     │
│       │                                                 │
│       │ Connects to infrastructure services              │
│       ▼                                                 │
│  ┌─────────────────────────────────────────────────┐   │
│  │        Docker Infrastructure Containers            │   │
│  ├─────────────────────────────────────────────────┤   │
│  │ PostgreSQL │ Redis │ MinIO │ Neo4j │ DocReader │   │
│  │   :5432    │ :6379 │ :9000 │ :7687 │  :50051   │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

### Production Mode Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Docker Compose Environment              │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  ┌──────────┐         ┌──────────┐                     │
│  │ Backend App│◄────────┤ Frontend UI│                     │
│  │ (running   │         │ (running   │                     │
│  │ in container)│         │ in container)│                     │
│  │  :8080   │         │   :80    │                     │
│  └────┬─────┘         └──────────┘                     │
│       │                                                 │
│       ▼                                                 │
│  ┌─────────────────────────────────────────────────┐   │
│  │              Infrastructure Containers              │   │
│  ├─────────────────────────────────────────────────┤   │
│  │ PostgreSQL │ Redis │ MinIO │ Neo4j │ DocReader │   │
│  └─────────────────────────────────────────────────┘   │
│                                                         │
└─────────────────────────────────────────────────────────┘
```
