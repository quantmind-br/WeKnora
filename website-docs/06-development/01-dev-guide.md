# Development Guide

The development environment uses `docker-compose.dev.yml` to start the dependency services, and supports running the Go backend, the web frontend, and docreader independently. After changing code, build and test the module it belongs to, then check the related integration paths.

## Tech Stack and Environment Requirements {#_1-tech-stack-and-environment-requirements}

WeKnora consists of three independently developable processes:

| Component | Directory | Language / Runtime | Version requirement (source) |
| --- | --- | --- | --- |
| Main backend `app` | `cmd/server` + `internal/` | Go | **Go 1.26.0** (`go 1.26.0` in `go.mod`), requires CGO (DuckDB, sqlite-vec bindings) |
| Document parsing service `docreader` | `docreader/` | Python + gRPC | **Python >= 3.10.18** (`requires-python` in `docreader/pyproject.toml`), dependencies managed with **uv** (the repo includes `uv.lock`; `uv sync --locked` inside Docker) |
| Frontend `frontend` | `frontend/` | Node.js + Vue 3 | Node 22 series (`devDependencies` include `@tsconfig/node22`, `@types/node ^22`), Vite 7 + TypeScript ~6.0 + Vue 3.5 + TDesign, version number `0.8.2` |
| CLI | `cli/` (independent Go module) | Go | Go 1.26 (`.github/workflows/cli.yml` matrix `go: ['1.26']`) |

Additional development tools recommended for installation:

```bash
# Database migration CLI (dependency of scripts/migrate.sh)
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Code linting (invoked by make lint)
# See https://golangci-lint.run for install instructions; config is at .golangci.yml in the repo root

# Swagger doc generation (invoked by make docs)
make install-swagger    # go install github.com/swaggo/swag/cmd/swag@latest

# Python dependency management
pip install uv          # docreader uses uv sync to install dependencies

# Docker + Docker Compose (either the v2 plugin or the standalone docker-compose works; scripts/dev.sh auto-detects)
```

## Quick Start: Development Mode (Recommended) {#_2-quick-start-development-mode-recommended}

Development mode runs the infrastructure in Docker and runs `app` and `frontend` locally. After changing application code you can simply restart the process, with no need to rebuild dependency images. The entry point is `scripts/dev.sh`, and the Makefile's `dev-*` targets wrap the corresponding operations.

```bash
# 1. Prepare environment variables: dev.sh loads .env (must exist), then overrides with .env.local (optional)
cp .env.example .env

# 2. Start infrastructure (ParadeDB/Postgres + Redis + docreader, and by default Langfuse as well)
make dev-start                      # equivalent to ./scripts/dev.sh start
make dev-start DEV_ARGS=--qdrant    # attach an optional profile

# 3. In another terminal: run the backend locally (internally runs go run -ldflags=... ./cmd/server)
make dev-app                        # equivalent to ./scripts/dev.sh app

# 4. In yet another terminal: run the frontend locally (cd frontend && npm install && npm run dev)
make dev-frontend                   # equivalent to ./scripts/dev.sh frontend

# Others
make dev-status   # view container status
make dev-logs     # view logs
make dev-stop     # stop
make dev-restart  # restart
```

The frontend dev server listens on `5173` (`server.port: 5173` in `frontend/vite.config.ts`), and proxies `/api` and `/files` to the local backend (`DEV_PROXY_TARGET`). `vite preview` (port `4173`) serves the production build artifacts, making it the environment that most closely resembles the release image for verification.

### docker-compose.dev.yml Service List {#_2-1-docker-compose-dev-yml-service-list}

`docker-compose.dev.yml` contains only dependency services, not `app`/`frontend`. The services started by default and the optional profile services are as follows (profiles are enabled via arguments to `dev.sh start`):

| Service | Image | Port (default) | Startup condition |
| --- | --- | --- | --- |
| `postgres` | `paradedb/paradedb:v0.22.6-pg17` (bundled with pg_search/BM25) | `5432` | Started by default |
| `redis` | `redis:7.0-alpine` (`--requirepass`) | `6379` | Started by default |
| `docreader` | Built locally from `docker/Dockerfile.docreader` | `50051` (gRPC) | Started by default |
| `searxng` (+ `searxng-init`) | `searxng/searxng:latest` | `127.0.0.1:8888` | `--searxng` / `--full` (compose profile `searxng`) |
| `minio` | `quay.io/minio/minio:latest` | `9000` / console `9001` | `--minio` / `--full` |
| `qdrant` | `qdrant/qdrant:v1.16.2` | `6333` / `6334` | `--qdrant` / `--full` |
| `opensearch` | `opensearchproject/opensearch:3.3.2` (security disabled, plain HTTP) | `9200` | profile `opensearch` / `full` |
| `opensearch-dashboards` | `opensearchproject/opensearch-dashboards:3.3.0` | `5601` | profile `opensearch-ui` (started separately as needed) |
| `milvus` | `milvusdb/milvus:v2.6.11` (standalone, embedded etcd) | `19530` / `9091` | profile `milvus` / `full` |
| `neo4j` | `neo4j:latest` (APOC plugin) | `7474` / `7687` | `--neo4j` / `--full` |
| `dex` | `dexidp/dex:latest` (OIDC test identity provider, config `misc/dex-config.yaml`) | `5556` | `--dex` / `--full` |
| `langfuse-web` / `langfuse-worker` / `langfuse-clickhouse` / `langfuse-minio` / `langfuse-db-init` | Self-hosted Langfuse v3 stack, reusing dev's postgres (separate `langfuse` database) and redis (DB 1) | web `3000`, minio `9100/9101` | `--langfuse` (enabled by default in `dev.sh`, disable with `--no-langfuse`) |
| `odl-hybrid` | Built locally from `docker/Dockerfile.odl-hybrid` (Docling PDF backend) | `5002` | `--odl-hybrid` (large image, use as needed) |
| `sandbox` | `wechatopenai/weknora-sandbox` (Skills script execution sandbox, build/pull only, not long-running) | - | profile `full` |

Optional arguments to `dev.sh start`: `--minio`, `--qdrant`, `--neo4j`, `--dex`, `--langfuse` (on by default), `--no-langfuse`, `--odl-hybrid`, `--full` (all optional services, excluding odl-hybrid). Pass these through the Makefile: `make dev-start DEV_ARGS=--odl-hybrid`.

### Running docreader Locally on Its Own {#_2-2-running-docreader-locally-on-its-own}

By default, `dev-start` runs docreader in a container. To debug the Python code locally instead:

```bash
cd docreader
uv sync                       # install dependencies per uv.lock (in-container it's uv sync --locked --no-dev)
uv run -m docreader.main      # start the gRPC service (matches the Dockerfile CMD), listening on DOCREADER_GRPC_PORT (default 50051)
```

docreader has a large number of tuning parameters (PDF rendering DPI, scanned-document detection, SSRF allowlist, gRPC TLS, etc.) injected via `DOCREADER_*` environment variables; see the `docreader.environment` section of `docker-compose.dev.yml` for the complete list.

### Lite Mode (Zero External Dependencies) {#_2-3-lite-mode-zero-external-dependencies}

Lite mode compiles SQLite (+sqlite-vec) and an in-memory queue into a single binary, suitable for quick trials and the desktop client:

```bash
make build-lite     # first builds the frontend into web/, then builds Go with CGO (tags: sqlite_fts5); SKIP_FRONTEND=1 skips the frontend
make run-lite       # depends on .env.lite; builds and starts WeKnora-lite
make package-lite   # packages a tarball release (scripts/package-lite.sh)
make package-mac-app  # packages a macOS .app (scripts/package-mac-app.sh)
```

## Full Overview of Makefile Targets {#_3-full-overview-of-makefile-targets}

The following targets are defined in the root `Makefile`; `make help` also provides a help text in Chinese.

### Basic Build and Run {#_3-1-basic-build-and-run}

| Target | Function |
| --- | --- |
| `build` | `go build -o WeKnora ./cmd/server` |
| `run` | Runs `build` first, then executes `./WeKnora` |
| `test` | `go test -v ./...` |
| `clean` | `go clean` and removes the binary |
| `build-prod` | Production build: CGO_ENABLED=1, `-ldflags "-w -s"` injects Version/CommitID/BuildTime/GoVersion (package variables in `internal/handler`), and sets protobuf `conflictPolicy=warn` (to work around the qdrant/milvus proto conflict) |
| `fmt` | `go fmt ./...` |
| `lint` | `golangci-lint run` |
| `deps` | `go mod download` |
| `docs` | `swag init -g ./cmd/server/main.go -o ./docs --parseDependency --parseInternal` generates the Swagger documentation |
| `install-swagger` | Installs the `swag` CLI |

### Docker Images and Service Management {#_3-2-docker-images-and-service-management}

| Target | Function |
| --- | --- |
| `docker-build-app` | Builds `wechatopenai/weknora-app` (`docker/Dockerfile.app`, injecting version info from `scripts/get_version.sh`) |
| `docker-build-docreader` | Builds `wechatopenai/weknora-docreader` (`docker/Dockerfile.docreader`) |
| `docker-build-frontend` | Multi-stage build of `wechatopenai/weknora-ui` (`npm ci` + `npm run build` inside the builder, so no dist needs to be prebuilt on the host; `VITE_FRONTEND_COMMIT` is injected from git automatically) |
| `docker-build-all` | Builds all three of the images above |
| `docker-run` | Ensures `.env` exists (copies from `.env.example` or touches it if missing), then runs `docker-compose up` |
| `docker-stop` / `docker-restart` | `docker-compose down` / `stop -t 60` + `up` |
| `start-all` / `stop-all` | `scripts/start_all.sh` (one-click start/stop of all services) |
| `start-ollama` / `start-docker` | `start_all.sh --ollama` / `--docker` |
| `build-images` / `build-images-app` / `build-images-docreader` / `build-images-frontend` / `clean-images` | `scripts/build_images.sh` builds/cleans images from source |
| `check-env` / `list-containers` / `pull-images` | `start_all.sh --check / --list / --pull` |
| `show-platform` | Displays `uname -m` and the Docker build platform (amd64/arm64 auto-detected) |
| `clean-db` | Deletes the three Docker volumes `weknora_postgres-data` / `weknora_minio_data` / `weknora_redis_data` (**wipes data**) |

### Database Migrations (see the "Database and Migrations" chapter for details) {#_3-3-database-migrations-see-the-database-and-migrations-chapter-for-details}

| Target | Function |
| --- | --- |
| `migrate-up` / `migrate-down` | `scripts/migrate.sh up / down` |
| `migrate-version` | View the current migration version |
| `migrate-create name=xxx` | Create a new pair of migration files |
| `migrate-force version=N` | Force-set the version (recovery from a dirty state) |
| `migrate-goto version=N` | Migrate to a specific version |

### Development Mode and Lite {#_3-4-development-mode-and-lite}

| Target | Function |
| --- | --- |
| `dev-start` / `dev-stop` / `dev-restart` / `dev-logs` / `dev-status` | `scripts/dev.sh start/stop/restart/logs/status` (supports passing profile arguments via `DEV_ARGS`) |
| `dev-app` | Runs `go run ./cmd/server` locally (with version ldflags) |
| `dev-frontend` | Runs `npm run dev` locally |
| `build-lite` / `run-lite` / `package-lite` / `package-mac-app` | Lite mode build/run/package (see section 2.3) |
| `download_spatial` | `go run cmd/download/duckdb/duckdb.go` downloads the DuckDB spatial extension (used by the data analysis tool) |

### Model Vendor Catalog {#_3-5-model-vendor-catalog}

| Target | Function |
| --- | --- |
| `model-catalog-generate` | `python3 scripts/model-catalog/generate.py`; regenerates the vendor catalog's metadata and protocol overrides |
| `model-catalog-check` | Verifies that the generated output is up to date and runs `go test ./internal/models/...` (CI checks the same) |
| `model-catalog-diff` | Produces a report of model metadata differences against the models.dev output, for manual review only, without writing files back; use `VENDOR=deepseek` to look at a single vendor |

For how vendors are integrated, see [Extension Points](03-extension-points.md).

## Testing System {#_4-testing-system}

### Go Unit Tests (Main Module) {#_4-1-go-unit-tests-main-module}

```bash
make test          # go test -v ./...
# Or run by package:
go test ./internal/infrastructure/chunker/...
go test -run TestXxx ./internal/application/service/...
```

The main module's tests make extensive use of in-memory doubles like `go-sqlmock` and `miniredis` (see `go.mod`); most can run without a real database. Some packages depend on CGO (DuckDB/sqlite-vec).

### docreader Tests (Python) {#_4-2-docreader-tests-python}

Tests live in `docreader/tests/`, written using the standard library `unittest` (each file calls `unittest.main()`), covering parsing routing, concurrency, EPUB/Excel/MHTML/PDF parsing, SSRF protection, and more:

```bash
cd docreader
uv sync
uv run python -m unittest discover -s tests -v      # run all
uv run python -m unittest tests.test_parser_routing  # run a single one
```

### CLI Tests and Acceptance Tests {#_4-3-cli-tests-and-acceptance-tests}

`cli/` is an independent Go module with its own `cli/Makefile`:

```bash
cd cli
make test            # go test ./...
make test-coverage   # with coverage
make lint            # go vet
```

Cross-cutting contract/integration tests are centralized in `cli/acceptance/` (see `cli/acceptance/doc.go`):

- `cli/acceptance/contract/` — golden tests for the shape of envelope JSON output + consistency checks for the error.code registry;
- `cli/acceptance/e2e/` — black-box tests against a real WeKnora server (testscript-style), requiring environment variables pointing to the test server; on the CI side this is carried by `.github/workflows/cli-e2e.yml`, triggered **on demand** (manually via `workflow_dispatch`, or by tagging a PR with the `acceptance-e2e` label), using the secrets `WEKNORA_E2E_HOST` / `WEKNORA_E2E_TOKEN`.

### The tests/ Directory and Frontend Tests {#_4-4-the-tests-directory-and-frontend-tests}

- `tests/miniprogram/miniprogram.test.js` — integration test for the mini-program client (a Node test script), currently the only content in `tests/`;
- Frontend: `cd frontend && npm run type-check` (vue-tsc) and `npm test` (`tsx --test`, the Node test runner).

## Coding Standards and Submission Process {#_5-coding-standards-and-submission-process}

### Go Coding Standards {#_5-1-go-coding-standards}

Root `.golangci.yml` (golangci-lint v2 configuration format):

```yaml
version: 2
linters-settings:
  lll:
    line-length: 120
    tab-width: 4
linters:
  enable:
    - lll           # controls line width (120 columns)
    - govet
    - revive
formatters:
  enable:
    - gofmt
    - gofumpt
```

Before submitting, it's recommended to run:

```bash
make fmt && make lint && make test
```

Note that the formatting standard is **gofumpt** (stricter than gofmt), with a line-width limit of 120.

You can also install the repository's bundled Git hooks (`./scripts/install-git-hooks.sh`, which points `core.hooksPath` at `scripts/git-hooks`): pre-commit checks whitespace, runs gofmt automatically, and runs golangci-lint when it's installed; pre-push runs gofmt plus `go vet` / `go test` / `go build` the same way CI does. Set `SKIP_HOOKS=1` to skip them temporarily, or `HOOK_SKIP_TEST=1` to skip only the pre-push tests.

When adding or upgrading third-party dependencies, or changing data files distributed with releases, update `THIRD_PARTY_NOTICES.md` and `licenses/` accordingly, and self-check with `scripts/check-license-bundle.sh` (`app.yml` runs the same check).

### CI and Submission Process {#_5-2-ci-and-submission-process}

The actual configuration under `.github/`:

| File | Trigger path | Function |
| --- | --- | --- |
| `workflows/app.yml` | Root module Go code, `go.mod`, `config/`, `migrations/`, `scripts/`, `docker/Dockerfile.app`, license files, model catalog data | Main module checks: gofmt format validation (only for commits within the PR), `go vet`, `go test`, `go build ./cmd/server`; also validates the third-party license bundle, the model vendor catalog's generated output, and the Git hooks tests |
| `workflows/go-lint.yml` / `go-lint-cache.yml` | PR / main | golangci-lint only reports issues newly introduced by the PR relative to the merge base; `go-lint-cache.yml` warms the cache on main |
| `workflows/frontend.yml` | `frontend/`, `scripts/build_frontend_dist.sh` | Node 24: `npm test` + `npm run type-check` + `npm run build`; also builds the `frontend/Dockerfile` multi-stage image (not pushed) and verifies the embed page and MCP proxy routes inside the image (`scripts/test_embed_nginx.py`) |
| `workflows/docreader.yml` | `docreader/`, `testdata/`, `packages/`, related Dockerfiles | uv installs dependencies → `compileall` → `unittest discover docreader/tests`; then spins up the docreader gRPC service and runs `go test ./docreader/client ./docreader/proto` |
| `workflows/mcp-server.yml` | `mcp-server/` | Python 3.10-3.13 matrix testing; after merging into main, automatically publishes to PyPI via Trusted Publishing based on the version number in `pyproject.toml` (skips the upload if that version already exists, without relying on a tag push) |
| `workflows/cli.yml` | `cli/` | ubuntu/macos/windows three-platform matrix, Go 1.26, `go build` + `go test -race -coverprofile` + `go vet` + skill wire word-list check |
| `workflows/cli-e2e.yml` | manual / label | CLI end-to-end acceptance testing (label `acceptance-e2e` or manually triggered, see 4.3) |
| `workflows/docker-image.yml` | — | Docker image build and publish |
| `workflows/release-lite.yml` | — | Lite version release |
| `workflows/anydoc.yml` | `third_party/anydoc-go/`, `internal/infrastructure/docparser/` | Build and test of the in-process Office parsing engine |
| `workflows/dsh-plugin.yml` | `packages/dsh-weknora/` | DeepSeek Harness plugin tests and release |
| `pull_request_template.md` | — | PR template |
| `ISSUE_TEMPLATE/` | — | Issue templates |
| `dependabot.yml` | — | Dependency upgrade bot |

The four path-triggered checks (app / frontend / docreader / mcp-server) cover the main modules, but running them locally first is still more time-efficient. For the frontend, you can directly use `scripts/verify_frontend_pr.sh`, which runs `npm test` → `npm run type-check` → `npm run build` in the same order as CI.

Submission process: fork / branch → local `fmt + lint + test` → PR (filled out per the template) → CI triggered on the relevant paths.

## Debugging Tips {#_6-debugging-tips}

### Log Levels {#_6-1-log-levels}

Logging is implemented in `internal/logger/logger.go` (logrus). The level is controlled by the `LOG_LEVEL` environment variable, taking values `debug` / `info` / `warn` (`warning`) / `error` / `fatal`; if unset or invalid, it defaults to **debug** (`getLogLevelFromEnv()`). `LOG_PATH` controls the output path; both take effect immediately after `.env` is loaded in `main()`. docreader likewise reads `LOG_LEVEL` (passed through in compose).

Every request carries an `X-Request-ID` that runs through both the app and docreader logs (docreader's `init_logging_request_id`); when troubleshooting, grab the request ID first.

### GIN_MODE and Swagger {#_6-2-gin-mode-and-swagger}

- When `GIN_MODE=release`, Swagger UI is disabled (`internal/router/router.go`), and it also affects the security behavior of the embed channel; don't set it, or set it to `debug`, during development.
- After `make docs` generates the Swagger docs, start the service and visit `http://localhost:8080/swagger/index.html`.

### Database and Migration Debugging {#_6-3-database-and-migration-debugging}

- `AUTO_MIGRATE=false` disables automatic migration at startup; `AUTO_RECOVER_DIRTY` (enabled by default, set to `false` to disable) controls automatic recovery from a dirty state (`internal/container/container.go`). Migration failures only warn without blocking startup — watch for `Database migration failed` in the startup logs.
- `make migrate-version` quickly confirms the schema version.

### LLM Chain Observability (Langfuse) {#_6-4-llm-chain-observability-langfuse}

`dev.sh start` spins up a self-hosted Langfuse by default (`http://localhost:3000`). The locally `go run` app needs the following exported:

```bash
export LANGFUSE_HOST=http://localhost:3000
export LANGFUSE_PUBLIC_KEY=pk-lf-xxx
export LANGFUSE_SECRET_KEY=sk-lf-xxx
```

This lets you view the model call trace for each session in the Langfuse UI (document processing spans are also persisted to the `knowledge_processing_spans` table, visualized in the frontend).

### pprof {#_6-5-pprof}

The current code does **not** have a built-in `net/http/pprof` endpoint (no pprof references under `internal/` or `cmd/`). For performance profiling, you can temporarily `import _ "net/http/pprof"` in `cmd/server/main.go` and start a separate `http.ListenAndServe("localhost:6060", nil)`, or use `go test -bench . -cpuprofile` to profile a specific package.

### Chunking Strategy Diagnostics {#_6-6-chunking-strategy-diagnostics}

The chunker provides `SplitWithDiagnostics()` (`internal/infrastructure/chunker/strategy.go`), which returns the strategy chain selection, the rejection reasons for each tier, and the document profile; combined with `LOG_LEVEL=debug` (the `chunker: tier %s rejected` log), this can be used to troubleshoot chunking results.

### Cloud Image Maintenance Scripts {#cloud-image-scripts}

`scripts/cloud-image/` is used to prepare distribution images on a dedicated, disposable Linux build machine. Regular deployments use [Installation & Deployment](../01-getting-started/02-installation.md) and don't need to run these scripts.

- `prepare.sh` downloads the runtime files for `WEKNORA_REF`, pulls the images, and installs the systemd services. That ref is also used to set the image version, so make sure the corresponding image tag exists. The systemd units shipped with the repository are fixed to `/opt/WeKnora`, so changing only the script's directory variables isn't enough to move the install location.
- `cleanup.sh` wipes the image build machine's data, keys, SSH authorizations, logs, and Docker cache, then shuts it down; its effects aren't limited to the WeKnora directory, so only use it on a dedicated build machine you've inspected.
- `firstboot.sh` generates keys on a new instance, writes `.env`, and starts Compose; when done it disables the first-boot service, but doesn't delete the script itself.

Before building an image, check that the scripts match the selected version. Before distribution, you must verify startup, service health, and key independence on a fresh instance; this description doesn't mean the current scripts have passed deployment or listing review on any specific cloud platform.

For first-boot troubleshooting, check `journalctl -u weknora-firstboot`, `/var/log/weknora-firstboot.log`, and the Compose status under `/opt/WeKnora`. `.firstboot.done` is written after the keys are generated and before Compose starts, so the marker's presence doesn't mean the services are healthy. If startup fails, keep the generated `.env` and marker, fix the cause, and start Compose again; deleting the marker to regenerate keys can leave the configuration inconsistent with the already-initialized database.
