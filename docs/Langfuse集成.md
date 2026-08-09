# Langfuse Integration

WeKnora includes a lightweight built-in integration with [Langfuse](https://langfuse.com) to track token consumption, trace LLM call chains, and generate a trace for every conversation that can be viewed in the Langfuse console. This integration addresses issue [#497](https://github.com/Tencent/WeKnora/issues/497) (token usage tracking) and discussion [#620](https://github.com/Tencent/WeKnora/discussions/620) (Langfuse integration).

## 1. Features

- Automatically reports prompts, responses, and token usage for all 5 model call types: **chat / embedding / rerank / VLM (Vision-Language Model) / ASR (Automatic Speech Recognition)**.
- Creates an end-to-end **trace** for every conversation, retrieval, and **file upload plus subsequent asynchronous processing**. The HTTP request is the root, and asynq tasks are attached to the same trace as SPANs — document parsing → chunk embedding → multimodal OCR/Captioning → summary/question generation are all visible within the same tree.
- Supports **streaming responses**: records Time-To-First-Token latency, with the complete response written once streaming ends.
- **Cross-process trace propagation**: the HTTP layer injects `trace_id` / `parent_observation_id` into the asynq payload, and the worker automatically resumes it at the asynq middleware layer; scheduled tasks (e.g., data source sync) fall back to standalone traces, still aggregated by task type (`asynq.<type>`).
- **Fully optional**: when the `LANGFUSE_*` environment variables aren't configured, Langfuse-related code paths are no-ops with zero performance overhead.
- **Asynchronous batch reporting**: doesn't block business requests; events are silently dropped when the queue is full, so observability data never affects user conversations.
- **Out-of-the-box deployment support**: Docker Compose (`docker-compose.yml` has the environment variables built in), Helm Chart (via `extraEnv`), and the Lite version (local single-machine) are all supported.

## 2. Quick Start

### 2.1 Obtain Langfuse Credentials

1. Log in at [cloud.langfuse.com](https://cloud.langfuse.com) or your self-hosted Langfuse instance.
2. Go to `Project Settings → API Keys` and generate a `Public Key` / `Secret Key` pair.

### 2.2 Configure by Deployment Method

#### (A) Docker Compose Deployment (Recommended)

`docker-compose.yml` already wires all `LANGFUSE_*` environment variables through to the `app` service. Two options are provided below.

##### A-1) Connect to Langfuse Cloud (Simplest)

Just add 3 lines to **`.env`**:

```bash
LANGFUSE_PUBLIC_KEY=pk-lf-xxxxxxxx
LANGFUSE_SECRET_KEY=sk-lf-xxxxxxxx
LANGFUSE_HOST=https://cloud.langfuse.com    # Use https://us.cloud.langfuse.com for the US region
```

Then restart the service:

```bash
docker compose up -d app
docker compose logs -f app | grep Langfuse
```

Seeing the following line confirms it's enabled:

```
[Langfuse] enabled host=https://cloud.langfuse.com flush_at=15 flush_interval=3s sample_rate=1.00
```

##### A-2) Self-Hosted Langfuse Stack (Offline / Internal Network / Data Compliance)

`docker-compose.yml` includes an optional `langfuse` profile that spins up Langfuse v3 with a single command.

**Designed to reuse WeKnora's existing containers as much as possible, avoiding wasted resources**:

| Component | Source | Notes |
| --- | --- | --- |
| PostgreSQL | Reuses `WeKnora-postgres` | A one-off `langfuse-db-init` container creates a separate `langfuse` database within the same pg instance. Isolated at the database level, no interference. |
| Redis | Reuses `WeKnora-redis` | Uses a dedicated Redis DB number (default DB 1; WeKnora uses DB 0). `REDIS_CONNECTION_STRING` specifies the DB suffix. |
| ClickHouse | New `langfuse-clickhouse` | Dedicated to Langfuse (OLAP event storage); WeKnora doesn't use it, so it must be separate. |
| MinIO | New `langfuse-minio` | Deliberately kept separate from WeKnora's `minio` (which is an optional profile that may not be active; Langfuse needs its own S3 bucket). |
| Web / Worker | New `langfuse-web` + `langfuse-worker` | The Langfuse application itself. |

In the end, `--profile langfuse` only adds **4 persistent containers + 1 one-off init container**, dropping memory overhead from the original ~1.5–2.5 GB down to roughly **1.0–1.5 GB**.

```bash
# 1. Start the self-hosted stack (first-time ClickHouse migration takes about 1–2 minutes)
docker compose --profile langfuse up -d

# 2. Open http://localhost:3000 in your browser to register an admin account
#    Then generate a Public/Secret Key under Project Settings → API Keys

# 3. Put the keys back into .env and change HOST to the internal container address
cat >> .env <<'EOF'
LANGFUSE_HOST=http://langfuse-web:3000
LANGFUSE_PUBLIC_KEY=pk-lf-xxxxxxxx
LANGFUSE_SECRET_KEY=sk-lf-xxxxxxxx
EOF

# 4. Reload the app's configuration
docker compose up -d app
```

> ⚠️ **Production deployment security notice**: the default passwords / `SALT` / `ENCRYPTION_KEY` in `.env.example` are development placeholders. For production, regenerate them with the following commands:
>
> ```bash
> echo "LANGFUSE_SALT=$(openssl rand -base64 32)"
> echo "LANGFUSE_ENCRYPTION_KEY=$(openssl rand -hex 32)"
> echo "LANGFUSE_NEXTAUTH_SECRET=$(openssl rand -base64 32)"
> ```
>
> Also replace `LANGFUSE_DB_PASSWORD` / `LANGFUSE_CLICKHOUSE_PASSWORD` / `LANGFUSE_REDIS_PASSWORD` / `LANGFUSE_MINIO_PASSWORD` with strong passwords across the board. See the "Langfuse self-hosted stack configuration" section of `.env.example` for the complete list of variables.

##### General Tuning

Optional tuning variables (`LANGFUSE_FLUSH_AT`, `LANGFUSE_SAMPLE_RATE`, etc.) are already pre-wired through in `docker-compose.yml` — just append the corresponding line to `.env` to take effect. See the Langfuse section of `.env.example`, or Section 3 of this document, for the complete list.

##### Resource Overhead Estimate (A-2 Self-Hosted Setup)

| Component | Type | Typical RSS | Notes |
| --- | --- | --- | --- |
| langfuse-db-init | One-off | – | Exits immediately after creating the `langfuse` database |
| langfuse-web | Persistent | 300–500 MB | Next.js |
| langfuse-worker | Persistent | 200–400 MB | Node.js, queue consumer |
| langfuse-clickhouse | Persistent | 500 MB–1 GB | Slightly higher during first migration; roughly 500 MB at steady state |
| langfuse-minio | Persistent | 100–200 MB | |
| (Reused) WeKnora-postgres | – | +~50 MB | One additional `langfuse` database |
| (Reused) WeKnora-redis | – | +30–80 MB | Shares DB 1 of the same instance |
| **New total** | | **≈ 1.0–1.5 GB** | 3 GB+ available memory recommended |

> Compared to a "fully isolated, separate pg/redis for everything" approach, this saves roughly **400–500 MB** of memory. The trade-off is that capacity planning for WeKnora's pg/redis needs to leave a bit of headroom for Langfuse; Langfuse's write volume isn't large (just metadata + task queue, with the bulk of events going to ClickHouse), so the actual impact is minimal.

For a single-machine deployment, if you only want to use the Langfuse Cloud approach (A-1), these containers are **entirely unnecessary**; CPU/memory usage of the existing services is unaffected.

##### Production Considerations

- **WeKnora-redis eviction policy**: Langfuse recommends `maxmemory-policy noeviction` (to prevent Redis from dropping queued tasks under memory pressure). If WeKnora's redis isn't configured with this policy, consider adding `--maxmemory-policy noeviction` to the redis command in `docker-compose.yml`.
- **Backups**: `pg_dump -d langfuse` can back up Langfuse's metadata independently; event data lives in the ClickHouse volume (`langfuse_clickhouse_data`).
- **For full isolation** (cross-machine deployment, strict ops separation): you can point `langfuse-web` / `langfuse-worker`'s `DATABASE_URL` and `REDIS_CONNECTION_STRING` directly at any external pg/redis (e.g., RDS + ElastiCache); the `langfuse-db-init` container can be skipped, and you can manually run `CREATE DATABASE langfuse` on the target pg instance instead.

#### (B) WeKnora Lite (Single Machine)

Add the following to `.env.lite` (or the environment variables exported by the startup script):

```bash
LANGFUSE_PUBLIC_KEY=pk-lf-xxxxxxxx
LANGFUSE_SECRET_KEY=sk-lf-xxxxxxxx
LANGFUSE_HOST=https://cloud.langfuse.com
```

After starting `weknora-lite` (or the macOS `.app`), the effect is the same as above.

#### (C) Helm Chart Deployment

Add to `app.extraEnv` in `values.yaml`:

```yaml
app:
  extraEnv:
    - name: LANGFUSE_PUBLIC_KEY
      valueFrom:
        secretKeyRef:
          name: langfuse-credentials
          key: public_key
    - name: LANGFUSE_SECRET_KEY
      valueFrom:
        secretKeyRef:
          name: langfuse-credentials
          key: secret_key
    - name: LANGFUSE_HOST
      value: https://cloud.langfuse.com
```

It's recommended to put the Secret Key into a Kubernetes Secret — never write it into values.yaml.

#### (D) Binary / Source Run

```bash
export LANGFUSE_PUBLIC_KEY="pk-lf-xxxx"
export LANGFUSE_SECRET_KEY="sk-lf-xxxx"
export LANGFUSE_HOST="https://cloud.langfuse.com"
./weknora-server
```

#### (E) Local Development (`docker-compose.dev.yml` + `go run`)

`docker-compose.dev.yml` only starts infrastructure containers (postgres/redis/docreader, etc.); the `app` runs locally via `go run ./cmd/server`. There are two ways to integrate Langfuse here:

**E-1) Connect Directly to Langfuse Cloud (most common for dev)**

No need to change any compose files — just export variables in your local shell:

```bash
export LANGFUSE_PUBLIC_KEY="pk-lf-xxxx"
export LANGFUSE_SECRET_KEY="sk-lf-xxxx"
export LANGFUSE_HOST="https://cloud.langfuse.com"
go run ./cmd/server
```

**E-2) Debug Against a Local Self-Hosted Stack**

The dev compose file also supports a symmetric `langfuse` profile (reusing the same dev postgres + redis):

```bash
# Bring up infrastructure + the Langfuse stack
docker compose -f docker-compose.dev.yml up -d postgres redis docreader
docker compose -f docker-compose.dev.yml --profile langfuse up -d

# Open http://localhost:3000 in your browser to register and generate a key

# Connect the local app (note: localhost, not langfuse-web, since go run runs on the host)
export LANGFUSE_HOST=http://localhost:3000
export LANGFUSE_PUBLIC_KEY=pk-lf-xxxxxxxx
export LANGFUSE_SECRET_KEY=sk-lf-xxxxxxxx
go run ./cmd/server
```

All dev-related containers use a `-dev` suffix and a dedicated network `WeKnora-network-dev`, so they **do not conflict** with the production compose setup.

### 2.3 Verification

Make a knowledge Q&A request (`POST /api/v1/knowledge-chat/:session_id`) or a knowledge search request (`POST /api/v1/knowledge-search`). After waiting 3 seconds (or until the batch size reaches `flush_at`), the corresponding trace will appear on the **Traces** page of the Langfuse console:

- Top-level node: the HTTP request (with `userId` / `sessionId`).
- Child nodes are the specific model calls in sequence — rerank, chat, VLM, etc. — click any one to view its prompt, response, and usage (prompt/completion/total tokens).
- Streaming conversations additionally show Time-To-First-Token.

## 3. Environment Variable Reference

| Variable | Default | Description |
| --- | --- | --- |
| `LANGFUSE_ENABLED` | Automatic | Explicit toggle. When unset, it's automatically enabled as long as both `PUBLIC_KEY` and `SECRET_KEY` are present. Supports `true/false/1/0/yes/no`. |
| `LANGFUSE_HOST` | `https://cloud.langfuse.com` | Address of the Langfuse instance. Use `https://us.cloud.langfuse.com` for the US region, or `https://langfuse.your-domain.com` for a self-hosted instance. |
| `LANGFUSE_PUBLIC_KEY` | — | The project's Public Key (`pk-lf-...`). |
| `LANGFUSE_SECRET_KEY` | — | The project's Secret Key (`sk-lf-...`); inject it via a secrets management tool — don't commit it to the repo. |
| `LANGFUSE_RELEASE` | — | Optional. The version number reported to Langfuse, e.g., a CI build number. |
| `LANGFUSE_ENVIRONMENT` | — | Optional. Environment label (`production` / `staging` / `dev`) for filtering in the UI. |
| `LANGFUSE_FLUSH_AT` | `15` | Batch size: reports immediately once the buffer accumulates this many events. |
| `LANGFUSE_FLUSH_INTERVAL` | `3s` | Periodic flush interval. Supports Go duration notation like `500ms`, `5s`, `1m`; plain numbers are treated as seconds. |
| `LANGFUSE_QUEUE_SIZE` | `2048` | In-memory queue capacity. New events are silently dropped when the queue is full (to avoid slowing down business logic). |
| `LANGFUSE_REQUEST_TIMEOUT` | `10s` | Timeout for a single HTTP ingest request. |
| `LANGFUSE_SAMPLE_RATE` | `1.0` | Sampling rate (0..1). `0` is treated as `1.0`. Can be lowered in high-traffic environments. |
| `LANGFUSE_DEBUG` | `false` | When enabled, prints detailed reasons for reporting failures in WeKnora's logs — turn on temporarily while troubleshooting. |

## 4. Observability Data Explained

| Langfuse Concept | WeKnora Equivalent | Notes |
| --- | --- | --- |
| Trace | One HTTP request (including all asynq tasks it triggers) | For online requests such as `knowledge-chat`, `agent-chat`, `knowledge-search`, `generate_title`, `evaluation`, model connectivity tests, etc.; as well as ingestion requests like file upload/URL ingestion/manual/reparse/move/copy, FAQ import, knowledge editing, wiki auto-fix, manually triggered data sources, etc. — the HTTP layer opens a trace for all of these and injects `trace_id` / `parent_observation_id` into the asynq payload. |
| Span (type=SPAN) | The execution window of each asynq task / each Agent execution and each of its rounds / each tool call | Registered by `internal/tracing/langfuse/AsynqMiddleware` in `mux.Use`; automatically creates an `asynq.<task_type>` SPAN for each handler, recording `task_id` / `queue` / `retry` / `payload_bytes`. Scheduled tasks (with no upstream trace) fall back to a standalone `asynq.<task_type>` trace. **Agent-related**: `AgentEngine.Execute` opens a top-level `agent.execute` SPAN, under which each ReAct loop round opens an `agent.round.N` SPAN, and each tool call opens an `agent.tool.<tool_name>` SPAN (arguments, output, duration, success/failure, and errors are all recorded). |
| Generation (type=GENERATION) | Each chat / embedding / rerank / VLM / ASR call | `parentObservationId` is automatically set when nested under a span, so the Langfuse UI presents a trace → asynq-span → generation tree; in Agent mode it's the full tree trace → agent.execute → agent.round.N → (chat.completion.stream + agent.tool.X → rerank/embedding...). |
| Input Tokens | `TokenUsage.PromptTokens` | From the usage field returned by the model. |
| Output Tokens | `TokenUsage.CompletionTokens` | From the usage field returned by the model. |
| Total Tokens | `TokenUsage.TotalTokens` | Returned by most providers; auto-summed when not returned. |
| Cache Read Tokens | `TokenUsage.CacheReadTokens` | Normalized from OpenAI-compatible `cached_tokens`, DeepSeek's native hit tokens, or Anthropic cache read tokens. |
| Cache Write Tokens | `TokenUsage.CacheWriteTokens` | Anthropic cache creation tokens or the provider's equivalent field; not double-counted with Input Tokens. |
| Cache Miss Tokens | `TokenUsage.CacheMissTokens` | Input tokens not read from cache, under the already-reported cache accounting. |
| Generation Metadata | `call_purpose` / `prompt_prefix_fingerprint` | Groups caching metrics by call purpose; only an irreversible short hash of the prefix is reported, never the raw prompt. |
| `userId` | `X-User-ID` / tenant ID | Falls back to `tenant:<id>` when not logged in, making it easy to aggregate consumption by tenant; written into the payload at enqueue time, so the worker retains attribution even with no upstream trace. |
| `sessionId` | `:session_id` in the URL (or `RequestID` as a fallback) | Lets you aggregate an entire conversation, or a single async batch, in Langfuse's Sessions view. |
| Time-To-First-Token | Time of the first valid chunk in a streaming call | Reported via `generation-update.completionStartTime`. |

### Covered asynq Task Types

The table below lists the asynq tasks that currently show up automatically as SPANs in Langfuse; each task's payload embeds `types.TracingContext`, and at enqueue time `langfuse.InjectTracing(ctx, &payload)` copies `trace_id` / `parent_observation_id` from the current HTTP trace.

| Task Type Constant | Handler | Typical Trigger Source |
| --- | --- | --- |
| `document:process` | `knowledgeService.ProcessDocument` | The four ingestion paths (file / URL / text / file_url); reparse; internal re-dispatch during knowledge base cloning |
| `manual:process` | `knowledgeService.ProcessManualKnowledge` | Manual knowledge creation / update |
| `image:multimodal` | `ImageMultimodalService.Handle` | Images discovered during document parsing |
| `knowledge:post_process` | `KnowledgePostProcessService.Handle` | Unified scheduling of summary/question generation after document parsing completes |
| `summary:generation` / `question:generation` | `KnowledgePostProcessService` subtasks | Dispatched by `knowledge:post_process` |
| `chunk:extract` | `ChunkExtractor.Handle` | Graph extraction (when NEO4J is enabled) |
| `datatable:summary` | `DataTableSummaryService.Handle` | Table file parsing |
| `faq:import` | FAQ bulk import handler | FAQ import / bulk creation |
| `knowledge:move` / `knowledge:list_delete` / `index:delete` / `kb:clone` / `kb:delete` | Knowledge move / bulk delete / index cleanup / knowledge base clone / knowledge base delete | Corresponding HTTP routes |
| `wiki:ingest` | `wikiIngestService.ProcessWikiIngest` | Wiki auto-fix / link rebuilding |
| `datasource:sync` | `dataSourceSyncService.Handle` | Manually triggered data source sync + scheduled sync (scheduled runs produce a standalone trace) |

### Usage Handling Strategy per Model

| Model Type | Reported Name | Token Metering Method | Notes |
| --- | --- | --- | --- |
| Chat | `chat.completion` / `chat.completion.stream` | Uses the model's returned `prompt_tokens` / `completion_tokens` / `total_tokens` directly | Streaming requests also record TTFT. |
| Embedding | `embedding.embed` / `embedding.batch_embed` | Estimates input tokens as `rune_count/4 + 1` when the model doesn't return usage | Batch calls report the batch size and a preview of the first 5 texts, avoiding stuffing the entire batch's content into the trace. |
| Rerank | `rerank` | Estimates input tokens from the rune count of `query + all documents` | Only the first 10 `(index, score)` entries are reported for output. |
| VLM | `vlm.predict` | Prompt/result input/output are each estimated as `rune/4` | Raw image bytes are never uploaded; only the image count and total byte size are recorded. |
| ASR | `asr.transcribe` | Metered in **seconds** (`SECONDS`), using the `end` value of the last segment in the transcription result as the audio duration | Makes it convenient for Langfuse to bill Whisper-type APIs "per minute." |

> Tip: Langfuse's `Settings → Models` page lets you configure unit pricing (per 1K tokens, per minute, etc.) for custom models (local Ollama, Alibaba Cloud Bailian, etc.), and Langfuse will automatically calculate cost accordingly.

## 5. High-Traffic Deployment Recommendations

- **Raise `LANGFUSE_FLUSH_AT`** to 50–100 to reduce the frequency of ingest HTTP calls.
- **Sampling**: setting `LANGFUSE_SAMPLE_RATE=0.1` samples only 10% of conversations, typically giving a good balance between production cost and signal-to-noise ratio.
- **Increase `LANGFUSE_QUEUE_SIZE`** to 8192 to prevent short-lived spikes from triggering event drops.
- Deploying the Langfuse instance in the same data center as WeKnora (e.g., a self-hosted Langfuse reachable via an internal network address) can significantly reduce reporting latency.
- Turning on `LANGFUSE_DEBUG=true` for a few minutes is enough to confirm the pipeline is working; keep it off in normal production operation to avoid log noise.

## 6. Disabling

Remove or leave blank `LANGFUSE_PUBLIC_KEY` / `LANGFUSE_SECRET_KEY`, or explicitly set `LANGFUSE_ENABLED=false`, then restart the service. All Langfuse-related code paths will fall back to no-ops, without affecting other observability components (OpenTelemetry, LLM Debug Log).

## 7. Troubleshooting

| Symptom | Suggested Steps |
| --- | --- |
| No `[Langfuse] enabled` in the startup log | Check whether `LANGFUSE_PUBLIC_KEY` / `LANGFUSE_SECRET_KEY` are actually being read by the service process; inside the container, verify with `env \| grep LANGFUSE`. |
| No traces visible in the console | Turn on `LANGFUSE_DEBUG=true` and watch the logs for `[Langfuse] flush ... failed`. Common causes: wrong `LANGFUSE_HOST`, corporate firewall blocking HTTPS, or the Secret Key was rotated but not updated. |
| Some chunks are missing | Increase `LANGFUSE_QUEUE_SIZE`; confirm the Langfuse ingest API isn't returning 429/503. |
| Token count is 0 | The model doesn't include a usage field in its response (common with some local Ollama / self-hosted models). Enable usage statistics on the model side, or configure a tokenizer for that model in Langfuse. |

## 8. Code Locations

- `internal/tracing/langfuse/` — Langfuse client, asynchronous batch reporting, Gin middleware, **asynq middleware**, Span/Trace resume implementation.
  - `tracer.go` — exposes `Trace` / `Span` / `Generation` + `StartTrace` / `StartSpan` / `StartGeneration` / `ResumeTrace`.
  - `asynq.go` — `AsynqMiddleware()` wraps handlers uniformly on the mux; `InjectTracing(ctx, payload)` injects the trace/span ID into the payload on the enqueue side.
  - `middleware.go` — Gin middleware + `shouldTrace` allowlist (covering chat / ingestion / FAQ / wiki / data source, and other paths).
- `internal/types/tracing.go` — `TracingContext` POCO; all asynq payloads carry `lf_trace_id` / `lf_parent_obs_id` / `lf_user_id` / `lf_session_id` by embedding this struct.
- `internal/models/chat/langfuse_wrapper.go` — Chat call decorator (including streaming).
- `internal/models/embedding/langfuse_wrapper.go` — Embedding call decorator.
- `internal/models/rerank/langfuse_wrapper.go` — Rerank call decorator.
- `internal/models/vlm/langfuse_wrapper.go` — VLM (Vision-Language Model) call decorator.
- `internal/models/asr/langfuse_wrapper.go` — ASR (Automatic Speech Recognition) call decorator.
- `internal/agent/engine.go` — top-level `agent.execute` SPAN and per-round `agent.round.<N>` SPAN.
- `internal/agent/act.go` — `agent.tool.<tool_name>` tool call SPAN (including arguments, output, duration, and success/failure).
- `internal/router/router.go` — registers `langfuse.GinMiddleware()`.
- `internal/router/task.go` — `mux.Use(langfuse.AsynqMiddleware())` on the asynq mux, so all handlers are automatically traced.
- `internal/container/container.go` — initialization + resource cleanup.
- `docker-compose.yml` / `.env.example` / `.env.lite.example` — pre-wired `LANGFUSE_*` environment variables passthrough.
