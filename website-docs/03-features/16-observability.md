# Observability and Auditing

Once you're running in production, you'll care about three kinds of questions: why a given response was slow, or why it was wrong; who changed what and when; and whether background tasks are piling up. WeKnora provides tracing, audit logs, and a queue dashboard to answer them, respectively.

| What you want to know | Where to look |
| --- | --- |
| What a given Q&A retrieved, how many model calls it made, and how many tokens it used | Once Langfuse is integrated, view the full call chain in Langfuse |
| Who changed a knowledge base / members / system settings | The knowledge base settings "Activity" tab, and "Settings → Audit Log" |
| Whether background parsing, summarization, or wiki tasks are piling up or failing | "Settings → Runtime Queue" |
| Whether the service is alive | `GET /health` |
| How a single request threads through logs across services | Search logs by the `X-Request-ID` response header |

<Screenshot
  src="/screenshots/queue-dashboard.png"
  caption="Runtime task queue: backlog, failures, and retries per queue"
  hint="Shows the queue dashboard, including queue names, pending/in-progress/failed counts, and dead-letter task action entry points." />

<Screenshot
  src="/screenshots/observability-langfuse.png"
  caption="Langfuse trace: the full call chain of a single Q&A"
  hint="Shows the expanded view of a trace in Langfuse, including retrieval, reranking, and generation spans along with token usage." />

Below we go through logging, tracing, auditing, rate limiting, and health checks in turn.

## 1. Observability Data Flow Overview

```mermaid
flowchart TB
    subgraph HTTP["HTTP request path (Gin)"]
        RID["middleware.RequestID<br/>(X-Request-ID generation/passthrough)"]
        RLOG["middleware.Logger<br/>(request/response body collection with redaction)"]
        LFMW["langfuse.GinMiddleware<br/>(opens a Trace for whitelisted paths)"]
        RBAC["middleware RBAC<br/>(LogDenied on rejection)"]
        H["business Handler"]
        RID --> RLOG --> LFMW --> RBAC --> H
    end

    subgraph ASYNC["Async task path (asynq worker)"]
        INJ["InjectTracing<br/>(writes traceparent into payload)"]
        AMW["langfuse.AsynqMiddleware<br/>(resumes trace + SPAN)"]
        WH["task Handler"]
        INJ --> AMW --> WH
    end
    H -->|"Enqueue(payload embeds TracingContext)"| INJ

    subgraph SINKS["Data sinks"]
        STDOUT["stdout + LOG_PATH file<br/>(lumberjack rotation: 50MB x 3, 28 days, gzip)"]
        LLMDBG["llm_debug/ files split by request_id<br/>(LLM_DEBUG_LOG, cleaned up after 7 days)"]
        LFB["Langfuse / LiteFuse backend<br/>POST /api/public/otel/v1/traces<br/>(OTLP HTTP + Basic Auth)"]
        ADB["audit_logs table (append-only)"]
        DLDB["task_dead_letters table"]
    end

    RLOG --> STDOUT
    H --> STDOUT
    WH --> STDOUT
    H -.->|"LLMDebugLog"| LLMDBG
    WH -.->|"LLMDebugLog"| LLMDBG
    LFMW -->|"batch export via BatchSpanProcessor"| LFB
    AMW --> LFB
    GEN["model langfuse_wrapper<br/>(chat / embedding / rerank / vlm / asr)"] --> LFB
    H --> GEN
    WH --> GEN
    RBAC -->|"rbac.access_denied (1-minute dedup)"| ADB
    H -->|"AuditLogService.Log"| ADB
    WH -->|"retries exhausted"| DLDB

    subgraph READERS["Query surfaces"]
        API1["GET /tenants/:id/audit-log"]
        API2["GET /knowledge-bases/:id/activity"]
        API3["GET /system/admin/audit-log"]
        RET["AuditLogRetentionRunner<br/>(daily sweep, defaults to 90-day retention)"]
    end
    ADB --> API1
    ADB --> API2
    ADB --> API3
    RET -->|"DeleteOlderThan"| ADB
```

## 2. Logging System (`internal/logger`)

### 2.1 Format and Levels

- The underlying implementation is a **private** logrus instance (`appLogger`, which avoids external dependencies mutating the global logrus instance and causing log loss), with a custom `CustomFormatter`.
- Default single-line format: `LEVEL[timestamp] [request_id fields...] caller | message`, where caller is `file:line[function name]` (`addCaller`).
- A template can be supplied via the `LOG_FORMAT` environment variable, with placeholders: `%d`=timestamp, `%level`=level, `%thread`=goroutine ID (only fetched when the template actually references it, to avoid running `runtime.Stack` on every log line), `%logger`=caller, `%traceId`=request_id, `%msg`=message+structured fields. A single-pass `strings.NewReplacer` avoids double-substitution issues.
- Level is controlled by `LOG_LEVEL` (`debug`/`info`/`warn`/`error`/`fatal`; **defaults to debug** when unset or invalid).
- Color: ANSI color is enabled when stdout is a terminal; disabled for non-terminal output (Docker log collection); when writing to a file, `ansiStripWriter` strips ANSI sequences to keep the output plain text.
- Structured field API: `logger.WithField(ctx, k, v)` / `WithFields` store an entry with fields attached into the context (`types.LoggerContextKey`), and subsequent calls to `logger.Infof(ctx, ...)` automatically carry them; `WarnWithFields` is dedicated to audit-related events (cross-tenant probing, invariant violations), making it easier for log aggregators to index by tenant/resource.
- `CloneContext` copies the key context values (tenant/user/request_id/role/language, etc.) when spawning a background goroutine, while also preserving both the Langfuse `*Trace` handle and the **active OTel span**, preventing child spans from becoming orphaned traces.

### 2.2 Output and Rotation

`ConfigureFromEnv()` (runs at init time, and can be re-invoked after `main` loads `.env`): always writes to stdout; when `LOG_PATH` is non-empty (or automatically, when running as a macOS `.app` bundle, falls back to `~/Library/Logs/<App>/<App>.log`), it additionally writes to disk via lumberjack:

```go
// internal/logger/logger.go openLogFile()
return &lumberjack.Logger{
    Filename:   logPath,
    MaxSize:    50, // megabytes
    MaxBackups: 3,
    MaxAge:     28, // days
    Compress:   true,
}, nil
```

### 2.3 LLM Debug Logging (`internal/logger/llm_logger.go`)

When `LLM_DEBUG_LOG=true|1|<directory>` is enabled, every model call (Chat / Chat Stream / Embedding / Rerank / VLM) writes its **complete** input messages, tool calls, output, and errors to the `llm_debug/` directory, with **all calls sharing the same request_id appended to the same file** (`<request_id>.log`), making it easy to reconstruct all model interactions within a single session. Files older than 7 days in the directory are cleaned up in the background at startup (`cleanupOldDebugFiles`).

### 2.4 Request Logging Middleware (`internal/middleware/logger.go`)

- `RequestID()`: reads or generates `X-Request-ID`, writes it back into the response header, and places the request_id together with a field-tagged logger into both the gin context and the `http.Request` context — enabling full-chain log correlation by request_id (including passthrough tagged via the asynq worker's session labels).
- `Logger()`: records method, path (with query parameters scrubbed of OAuth-sensitive params like `token`/`code`/`state` via `sanitizeQuery`), status_code, latency, client_ip, size, and up to 10KB of request/response body. Request/response bodies are redacted via `sensitiveFieldRegex` (values of fields like password/token/api_key/secret/private_key are replaced with `"***"`, compatible with both snake_case and camelCase); SSE response bodies are recorded as `[SSE streaming response, skipped]`; `/assets/` and wiki stats polling paths are skipped entirely.
- Trusted proxies: `r.SetTrustedProxies(...)` (`WEKNORA_TRUSTED_PROXIES`) prevents forged `X-Forwarded-For` headers from bypassing `ClientIP`-based rate limiting.

## 3. Langfuse Tracing (`internal/tracing/langfuse`)

WeKnora's distributed tracing is not a generic OTel integration, but rather a **Langfuse v3+ / LiteFuse client built on the OpenTelemetry Go SDK**: spans carry Langfuse semantic-convention attributes (`langfuse.observation.*`, mirroring the langfuse-python v4 `_client/attributes.py`), and are exported via OTLP/HTTP to `POST <host>/api/public/otel/v1/traces`. It's entirely opt-in: when not enabled, all entry points are zero-cost no-ops.

### 3.1 Configuration (Environment Variables, `config.go`)

| Environment Variable | Default | Description |
| --- | --- | --- |
| `LANGFUSE_ENABLED` | Auto-enabled when public/secret keys are present | Master switch (consistent with the Python SDK's convention) |
| `LANGFUSE_HOST` | `https://cloud.langfuse.com` | Langfuse/LiteFuse base URL (can be self-hosted) |
| `LANGFUSE_PUBLIC_KEY` / `LANGFUSE_SECRET_KEY` | — | Basic Auth project credentials |
| `LANGFUSE_RELEASE` / `LANGFUSE_ENVIRONMENT` | — | Attached to every trace for filtering in the UI |
| `LANGFUSE_FLUSH_AT` | 15 | Batch export batch size (`BatchSpanProcessor`'s `MaxExportBatchSize`) |
| `LANGFUSE_FLUSH_INTERVAL` | 3s | Maximum interval for batch export (`BatchTimeout`) |
| `LANGFUSE_QUEUE_SIZE` | 2048 | In-memory buffer cap (prevents unbounded growth when the endpoint is unreachable) |
| `LANGFUSE_REQUEST_TIMEOUT` | 10s | HTTP timeout for a single ingestion request |
| `LANGFUSE_SAMPLE_RATE` | 1.0 | Sampling rate for `ParentBased(TraceIDRatioBased)`, 0..1 |
| `LANGFUSE_DEBUG` | false | Verbose logging for batch-send errors |

### 3.2 Exporter (`exporter.go`)

An OTLP/HTTP exporter with `Authorization: Basic base64(public:secret)`; the `x-langfuse-ingestion-version: 4` header is required for the OTel direct-write path of Langfuse v3/LiteFuse (its absence returns a 400), and `x-langfuse-sdk-name/version` are compatibility markers. `Manager` (`manager.go`) holds its own independent `TracerProvider` (with a `service.name=weknora` resource), and deliberately **does not** call `otel.SetTextMapPropagator` or any other global OTel mutation, so as not to affect other OTel instrumentation elsewhere in the process; the W3C `TraceContext` propagator is a package-private value.

### 3.3 Observation Model and Instrumentation Points

There are three handle types (`tracer.go`): `Trace` (the root, one per request), `Span` (a non-LLM logical unit of work), and `Generation` (a single model call, including `TokenUsage` token statistics and streaming time-to-first-token via `MarkCompletionStart`). Parent-child relationships are established automatically via the OTel span context; when there's no active trace, an auto-trace is opened automatically to prevent orphaned spans.

Main instrumentation points:

| Point | Source | Output |
| --- | --- | --- |
| HTTP entry point | `middleware.go` `GinMiddleware` | Opens a root Trace for `shouldTrace`-whitelisted paths (knowledge-chat / agent-chat / knowledge-search / various ingestion POST/PUT endpoints / FAQ import / wiki auto-fix / evaluation / initialization detection, etc.), named `METHOD /path`, with metadata containing http.method/path/query/request_id, and output containing status and response.size; extracts the upstream W3C `traceparent` header to inherit the caller's trace id |
| asynq worker | `asynq.go` `AsynqMiddleware` | Restores the traceparent from the payload to resume the HTTP trace, or otherwise opens a new `asynq.<task_type>` trace; wraps it in a SPAN with metadata containing task_id/queue/retry/max_retry/payload_bytes; the payload preview is limited to the first 1KB |
| Enqueue-side injection | `asynq.go` `InjectTracing` + `internal/types/tracing.go` `TracingContext` | Embeds the traceparent and user/session labels as `lf_*` JSON fields into the task payload, carrying them across process boundaries |
| Model calls | `internal/models/{chat,embedding,rerank,vlm,asr}/langfuse_wrapper.go` | One Generation per call (model name, input, parameters, output, token usage, errors) |
| Retrieval/rerank summaries | `retrieval_obs.go` | `SummarizeRetrieveOutput` / `SummarizeSearchResults` and similar functions compress retrieval results into a top-25 preview (rank/chunk_id/score/160-character preview), avoiding dumping full text into the trace |
| Agent execution | `internal/agent/engine.go`, `act.go` | SPANs such as agent.execute, kept in the same tree as the HTTP root trace via `logger.CloneContext` |

Reported content (span attributes, `events.go`): `langfuse.observation.type/input/output/metadata/model.name/model.parameters/usage_details/completion_start_time`, `langfuse.trace.name/input/output/metadata/tags`, `user.id` (explicit user or `tenant:<id>`), `session.id`, `langfuse.environment/release`.

```mermaid
flowchart LR
    A["GinMiddleware<br/>Trace: POST /api/v1/agent-chat"] --> B["Span: agent.execute"]
    B --> C["Generation: chat (LLM planning/response)"]
    B --> D["Generation: embedding (retrieval)"]
    B --> E["Generation: rerank"]
    A --> F["InjectTracing -> asynq payload"]
    F --> G["AsynqMiddleware<br/>Span: asynq.document:process"]
    G --> H["Generation: embedding / vlm / chat"]
```

## 4. Audit Logs

### 4.1 Data Model (`internal/types/audit_log.go`)

The `audit_logs` table is **append-only** (no UpdatedAt, no soft delete), with a monotonic id serving as both primary key and cursor:

| Field | Type | Description |
| --- | --- | --- |
| `id` | uint64 auto-increment | Primary key + pagination cursor (`WHERE id < after_id ORDER BY id DESC`) |
| `tenant_id` | uint64 | Space; `0` = system-scope event |
| `actor_user_id` / `actor_role` | varchar | The actor and their role at the time (empty for system-triggered events) |
| `action` | varchar(64) | Dot-separated name `<area>.<event>` (see 4.2) |
| `scope_type` / `scope_id` | varchar | Resource scope (e.g. `knowledge_base` + kbID, which drives the KB activity page) |
| `target_type` / `target_id` / `target_user_id` | varchar | The specific target resource / user |
| `request_path` / `request_method` | varchar | Route template (not the raw URL, to prevent cursor bloat; the raw URL is stored in Details.raw_path) |
| `outcome` | varchar(16) | `success` / `accepted` (accepted asynchronously, not yet in a terminal state) / `denied` / `failed` / `partial` / `canceled` |
| `details` | jsonb | Action-specific payload; secret values are **never** stored (e.g. vector_store only records the names of changed fields) |
| `created_at` | timestamp | Basis for retention-policy sweeps |

### 4.2 List of Audit Actions

| Group | Actions |
| --- | --- |
| RBAC / members | `rbac.member_added`, `rbac.member_removed`, `rbac.member_role_changed`, `rbac.member_left`, `rbac.access_denied`, `rbac.invitation_sent`, `rbac.invitation_accepted`, `rbac.invitation_declined`, `rbac.invitation_revoked`, `rbac.invitation_expired` |
| Vector store | `vector_store.created`, `vector_store.updated`, `vector_store.deleted` |
| OpenSearch-derived resources | `opensearch.index_created`, `opensearch.index_deleted`, `opensearch.reindex_executed` |
| System administration (tenant_id=0) | `system.setting_changed`, `system.admin_promoted`, `system.admin_revoked`, `system.user_password_reset`, `system.api_key_created`, `system.api_key_revoked` |
| Runtime queue operations (tenant_id=0) | `system.queue_task_retried`, `system.queue_task_deleted`, `system.queue_task_run_now`, `system.queue_task_cancelled`, `system.queue_archived_purged` |
| Knowledge base | `kb.created`, `kb.updated`, `kb.deleted`, `kb.duplicated`, `kb.clone_started`, `kb.clone_completed`, `kb.clone_failed`, `kb.share_added`, `kb.share_permission_changed`, `kb.share_removed` |
| Knowledge | `knowledge.created`, `knowledge.updated`, `knowledge.deleted`, `knowledge.batch_deleted`, `knowledge.reparse_started`, `knowledge.parse_canceled`, `knowledge.move_started`, `knowledge.move_completed`, `knowledge.move_failed` |
| Tags / data sources | `tag.created`, `tag.updated`, `tag.deleted`, `datasource.created`, `datasource.updated`, `datasource.deleted`, `datasource.sync_started`, `datasource.sync_completed`, `datasource.sync_failed`, `datasource.paused`, `datasource.resumed` |
| Wiki / FAQ | `wiki.content_changed`, `faq.import_started`, `faq.import_completed`, `faq.import_failed` |

### 4.3 Write Path (Service + Middleware)

- `auditLogService.Log` (`internal/application/service/audit_log.go`) is the canonical write entry point: defaults `outcome=success`, populates `CreatedAt`; **write failures are only logged as ERROR and never propagated upward** — an audit failure must never interrupt a business operation.
- `LogDenied` records RBAC middleware rejections: it dedups using a **1-minute sliding window** keyed by `(tenant_id, actor, action=rbac.access_denied, route template)` (`denyDedupWindow`, `repo.CountSinceForDedup`), preventing a probing client from flooding the table (100 RPS hitting the same endpoint only produces 1 row per minute); the route template rather than the raw URL is used as the dedup key, to prevent traversing UUIDs to bypass the window. The stderr-side `[rbac] role insufficient` log is unaffected by dedup and is written on every rejection.
- `middleware/audit_provider.go`'s `AuditServiceProvider` injects the service into the gin context (key `weknora.audit_service`); the RBAC middleware retrieves it via `AuditServiceFromContext`, which is nil-safe (Lite mode can run without audit configured).

### 4.4 Query API (`internal/handler/audit_log.go`)

| Route | Permission | Description |
| --- | --- | --- |
| `GET /api/v1/tenants/:id/audit-log` | PathTenantMatch + Admin | Space-level audit stream; only returns rows with `scope_type=''` (`UnscopedOnly`) |
| `GET /api/v1/knowledge-bases/:id/activity` | KB creator or space Admin, and must be in the owning space (organization-shared consumers cannot read it) | KB activity projection filtered by `scope_type=knowledge_base` + `scope_id=kbID` |
| `GET /api/v1/system/admin/audit-log` | SystemAdmin (+ platform API Key with `system.audit_read`) | Platform-level events with `tenant_id=0` (settings / promote / queue operations, etc.) |

Common query parameters: `after_id` (cursor, returns rows with a smaller id), `limit` (1–100, default 50, hard cap `auditLogListLimitMax=100`), and exact filters on `action` / `outcome` / `actor`. The response includes `next_cursor` (the smallest id on the page; 0 means the end has been reached).

### 4.5 Retention Policy (`internal/application/service/audit_log_retention.go`)

- Configuration: `audit.retention_days` (YAML) / `WEKNORA_AUDIT_RETENTION_DAYS` (env override); defaults to **90 days** when the `audit:` section is omitted; an explicit 0 disables sweeping (for compliance scenarios with out-of-band archival), and a negative value triggers a config validation error.
- `AuditLogRetentionRunner`: a bare `time.Ticker`-based background goroutine (no cron / asynq dependency), with a 10-minute startup delay (to avoid migration and startup traffic), after which it runs `Purge` → `DeleteOlderThan(now - retention_days)` **every 24h** (a single indexed DELETE with a 30s timeout). Deletion counts are logged at INFO, failures at WARN (retried on the next cycle). It's wired up by `internal/container/container.go` and registered as a `ResourceCleaner` for graceful shutdown (`Stop` is idempotent, and returns immediately if never Started).

## 5. Rate Limiting (`internal/ratelimit` and Middleware)

### 5.1 General-Purpose Sliding-Window Limiter (`internal/ratelimit/limiter.go`)

- Redis-first: a Lua script atomically performs "evict expired ZSET members → `ZCARD` count → `ZADD` + `PEXPIRE` if under the limit," sharing budget across multiple instances; members are keyed as `<instanceID>:<ms>` to guarantee uniqueness.
- When Redis is unavailable (error, or Lite mode without Redis), it **automatically falls back** to an in-process `localLimiter` (a `sync.Map` plus a per-key timestamp array), with `StartCleanup` periodically evicting empty keys.
- `max` is passed in on each `Allow` call, so the same limiter can apply different budgets to different keys (e.g. separate quotas per embed channel).
- Consumers: the Web embed public interface (two limiters — per-minute and per-24h — keyed by channel+ClientIP, in `internal/middleware/embed_auth.go`) and the IM service (`internal/im/service.go`).

### 5.2 Public Authentication Endpoint IP Rate Limiting (`internal/middleware/auth_public_ratelimit.go`)

`PublicAuthRateLimit()` protects unauthenticated invitation-link endpoints (`/auth/invitations/lookup`, `/auth/register-by-invite`): an in-process sliding window allowing **30 requests/minute per IP** (a shared bucket across both endpoints), returning 429 (`ErrTooManyRequests`) when exceeded. This is a purely local implementation (suited to low-traffic endpoints); the code comments note that a Redis-backed version from `internal/ratelimit` should be used instead if horizontal scaling is needed.

## 6. Health Checks

`internal/router/router.go` registers an unauthenticated health probe (the public path whitelist in `internal/middleware/auth.go` includes `/health`):

```go
// internal/router/router.go
r.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
```

This is a pure liveness probe (it does not check DB/Redis dependencies), suitable as a target for container / LB health checks. Both `langfuse.shouldTrace` and request-log sampling exclude it as well, to avoid probe noise. Process uptime is exposed to the ops dashboard via `MarkServerStarted`/`ServerUptime` in `internal/runtime/server.go`.

## 7. Model Usage-by-Reference Statistics (`internal/application/repository/model_usage.go`)

This file provides **usage-by-reference** queries — answering "which resources are currently using a given model" — used to protect against deleting a model that's still in use, rather than token-usage billing:

- `scopeKnowledgeBasesByModelID`: matches against any of the model-binding fields in `knowledge_bases` — `embedding_model_id`, `summary_model_id`, `image_processing_config.model_id`, `vlm_config.model_id`, `asr_config.model_id`, `wiki_config.synthesis_model_id` (using the `->>` JSON operator on Postgres and `json_extract` on SQLite, equivalent across both dialects).
- `scopeCustomAgentsByModelID`: matches against `model_id`, `rerank_model_id`, `vlm_model_id`, `asr_model_id`, `query_understand_model_id`, and `question_suggestions.follow_ups.model_id` within `custom_agents.config`.
- Consumers: `CountByModelID` in the `knowledgebase.go` / `custom_agent.go` repositories, called by the deletion guard in `internal/application/service/model.go` (a model deletion is blocked if its KB or Agent reference count is > 0).

Token-level model usage, on the other hand, is reported via the `usage_details` field of a Langfuse Generation (`TokenUsage`: input/output/total/cache_*), and can be aggregated by model / user (`tenant:<id>`) / session in the Langfuse UI.

## 8. Operations Quick Reference

| What you want to know… | Where to look |
| --- | --- |
| What happened across the full chain for a given request | grep the application logs by the `X-Request-ID` response header; with `LLM_DEBUG_LOG` enabled, check `llm_debug/<request_id>.log` |
| The LLM call tree and token consumption for a chat/parse operation | Langfuse UI (trace names `POST /api/v1/agent-chat` or `asynq.document:process`) |
| Who changed what and when | Space audit log at `/tenants/:id/audit-log`; KB activity at `/knowledge-bases/:id/activity`; platform audit log at `/system/admin/audit-log` |
| Why a given document keeps failing | The `task_dead_letters` table (scope=knowledge/knowledge_base) + the `last_error` field of archived tasks in the runtime dashboard |
| Whether the service is alive | `GET /health` (200 `{"status":"ok"}`) |
| Whether configuration loaded as expected | The `[startup-env]` banner in the startup logs (`internal/runtime/startup.go`; sensitive values only show their length) |

## Implementation Reference

To navigate the source code, use the table below (paths relative to the repository root):

| Capability | Source Path |
| --- | --- |
| Application logging | `internal/logger/logger.go` |
| LLM call debug logging | `internal/logger/llm_logger.go` |
| Request logging / RequestID middleware | `internal/middleware/logger.go` |
| Langfuse tracing (OTel SDK) | `internal/tracing/langfuse/` (`config.go`, `manager.go`, `exporter.go`, `tracer.go`, `middleware.go`, `asynq.go`, `events.go`, `retrieval_obs.go`, `context.go`) |
| Cross-process trace carrier | `internal/types/tracing.go` |
| Audit log handler / service / repo | `internal/handler/audit_log.go`, `internal/application/service/audit_log.go`, `internal/application/repository/audit_log.go` |
| Audit retention policy | `internal/application/service/audit_log_retention.go`, `internal/config/config.go` (`applyAuditDefaults`) |
| Audit actions / model | `internal/types/audit_log.go` |
| Rate limiting | `internal/ratelimit/limiter.go`, `internal/middleware/auth_public_ratelimit.go` |
| Health check | `internal/router/router.go` (`GET /health`) |
| Model usage-by-reference statistics | `internal/application/repository/model_usage.go` |
