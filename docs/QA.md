# Frequently Asked Questions

## 1. How do I view the logs?
```bash
docker compose logs -f app docreader postgres
```

## 2. How do I start and stop the services?
```bash
# Start services
./scripts/start_all.sh

# Stop services
./scripts/start_all.sh --stop

# Clear the database
./scripts/start_all.sh --stop && make clean-db
```

## 3. The service starts but I can't upload documents?

This is usually caused by the Embedding model and chat model not being configured correctly. Follow these steps to troubleshoot:

1. Check whether the model information in your `.env` configuration is complete. If you're using ollama to access a local model, make sure the local ollama service is running properly, and that the following environment variables in `.env` are set correctly:
```bash
# LLM Model
INIT_LLM_MODEL_NAME=your_llm_model
# Embedding Model
INIT_EMBEDDING_MODEL_NAME=your_embedding_model
# Embedding model vector dimension
INIT_EMBEDDING_MODEL_DIMENSION=your_embedding_model_dimension
# Embedding model ID, usually a string
INIT_EMBEDDING_MODEL_ID=your_embedding_model_id
```

If you're accessing the model via a remote API, you also need to provide the corresponding `BASE_URL` and `API_KEY`:
```bash
# LLM model access URL
INIT_LLM_MODEL_BASE_URL=your_llm_model_base_url
# LLM model API key, set this if authentication is required
INIT_LLM_MODEL_API_KEY=your_llm_model_api_key
# Embedding model access URL
INIT_EMBEDDING_MODEL_BASE_URL=your_embedding_model_base_url
# Embedding model API key, set this if authentication is required
INIT_EMBEDDING_MODEL_API_KEY=your_embedding_model_api_key
```

When you need the reranking feature, you'll need to configure a Rerank model as well, as follows:
```bash
# Name of the Rerank model to use
INIT_RERANK_MODEL_NAME=your_rerank_model_name
# Rerank model access URL
INIT_RERANK_MODEL_BASE_URL=your_rerank_model_base_url
# Rerank model API key, set this if authentication is required
INIT_RERANK_MODEL_API_KEY=your_rerank_model_api_key
```

2. Check the main service logs to see whether there is any `ERROR` output.

## 4. No images, or invalid image links are shown?

When using multimodal features, if you run into issues where images fail to display or show invalid links, troubleshoot as follows:

### 1. Confirm the multimodal feature is configured correctly

In the knowledge base settings, enable **Advanced Settings - Multimodal**, and configure the corresponding multimodal model in the interface.

### 2. Confirm the MinIO service is running

If the multimodal feature is configured to use MinIO storage, make sure the MinIO image has been started correctly:

```bash
# Start the MinIO service
docker-compose --profile minio up -d

# Or start the full set of services (including MinIO, Neo4j, Qdrant)
docker-compose --profile full up -d
```

### 3. Check MinIO bucket permissions

Make sure the corresponding MinIO bucket has the correct read/write permissions:

1. Access the MinIO console: `http://localhost:9001` (default port)
2. Log in using the `MINIO_ACCESS_KEY_ID` and `MINIO_SECRET_ACCESS_KEY` configured in `.env`
3. Go to the corresponding bucket, and check and set the access policy to **public read** or **public read/write**

**Important**:
- Bucket names should not contain special characters (including Chinese characters); it's recommended to use lowercase letters, digits, and hyphens
- If you can't modify the permissions of an existing bucket, you can enter a bucket name that doesn't exist yet in the configuration — the project will automatically create the corresponding bucket and set up the correct permissions

### 4. Configure MINIO_PUBLIC_ENDPOINT

In the `docker-compose.yml` file, the `MINIO_PUBLIC_ENDPOINT` variable defaults to `http://localhost:9000`.

**Important**: If you need to access images from another device or container, `localhost` may not work properly, and you'll need to replace it with the actual IP address of the machine:


## 5. Platform compatibility notes

**Important**: The `OCR_BACKEND=paddle` mode may not work properly on some platforms. If PaddleOCR fails to start, choose one of the following solutions:

### Option 1: Disable OCR recognition

Remove the `OCR_BACKEND` configuration from the `docreader` service in the `docker-compose.yml` file, then restart the docreader service.

**Note**: After setting it to `no_ocr`, document parsing will no longer use the OCR feature, which may affect text recognition for images and scanned documents.

### Option 2: Use an external OCR model (recommended)

If you need OCR functionality, you can use an external Vision Language Model (VLM) instead of PaddleOCR. Configure the following in the `docreader` service in the `docker-compose.yml` file:

```yaml
environment:
  - OCR_BACKEND=vlm
  - OCR_API_BASE_URL=${OCR_API_BASE_URL:-}
  - OCR_API_KEY=${OCR_API_KEY:-}
  - OCR_MODEL=${OCR_MODEL:-}
```

Then restart the docreader service.

**Advantage**: Using an external OCR model gets better recognition results and isn't limited by platform.

## 6. How do I use the data analysis feature?

Before using the data analysis feature, make sure the agent has the relevant tools configured:

1. **Intelligent reasoning**: You need to check the following two tools in the tool configuration:
   - View data metadata
   - Data analysis

2. **Quick Q&A agent**: No need to manually select tools — you can perform simple data queries directly.

### Notes and usage guidelines

1. **Supported file formats**
   - Currently only **CSV** (`.csv`) and **Excel** (`.xlsx`, `.xls`) file formats are supported.
   - For complex Excel files, if reading fails, it's recommended to convert them to standard CSV format before re-uploading.

2. **Query restrictions**
   - Only **read-only queries** are supported, including statements like `SELECT`, `SHOW`, `DESCRIBE`, `EXPLAIN`, `PRAGMA`, etc.
   - Any operations that modify data are prohibited, such as `INSERT`, `UPDATE`, `DELETE`, `CREATE`, `DROP`, etc.

## 7. Configuration I just saved on the page disappears again after a few seconds?

This kind of issue is usually not the system actually clearing the configuration — rather, browser proxies, caching, or plugin interference cause the frontend to read an abnormal response, and the page then gets overwritten by stale state.

We recommend troubleshooting in the following order:

1. First disable your browser proxy, packet-capturing tools, and any plugins that auto-rewrite requests, then reopen the page.
2. Confirm your browser isn't routing `localhost` or the domain you're accessing through a proxy; if you have a PAC configured, add `localhost`, `127.0.0.1`, and the actual deployment domain to the direct-connection list.
3. Force-refresh the page, or log in again in an incognito window and save the configuration once more.
4. Open the `Network` panel in your browser's developer tools, and confirm that the request related to saving the configuration returns the latest content, and hasn't been rewritten by a proxy, served from cache, or redirected to another environment.
5. If this is a debug-mode deployment, try restarting the `app` service and verify again:

```bash
docker compose restart app
```

If things return to normal briefly after restarting but the same issue reappears on a later visit, you should still prioritize checking for browser proxy, caching, and cross-environment issues rather than jumping to the conclusion that the backend configuration was lost.

## 8. SSRF validation whitelist (`SSRF_WHITELIST`)

Optional configuration. Set `SSRF_WHITELIST` in `.env` to add specified targets to a whitelist during URL validation and similar steps, allowing them to bypass the usual SSRF restrictions. The value is a comma-separated list of rules, where each entry can be:

- **Exact domain**: e.g. `api.internal`
- **Wildcard domain**: e.g. `*.example.com`
- **IPv4**: e.g. `203.0.113.5`
- **IPv6**: e.g. `2001:db8::1` (do not include brackets)
- **CIDR**: e.g. `10.0.0.0/8`, `2001:db8::/32`

Whitelisted addresses will bypass the usual SSRF rules during URL validation and similar checks — **configure this with caution in production**, and only add targets that are genuinely needed and trusted.

Example (consistent with `.env.example`; uncomment and modify as needed):

```bash
# SSRF_WHITELIST=internal.service,*.corp.example,172.16.0.0/12,2001:db8::1,fd00::/8
```


## 9. How do I enable and view Langfuse observability tracing?

WeKnora supports full-chain tracing of the Agent's ReAct loop, LLM token consumption, tool calls, and asynchronous task pipelines via Langfuse.

**Setup steps**:
1. Prepare a usable Langfuse instance (cloud or self-hosted).
2. Configure the following environment variables in the `.env` file:
```bash
LANGFUSE_PUBLIC_KEY=pk-lf-...
LANGFUSE_SECRET_KEY=sk-lf-...
LANGFUSE_HOST=https://cloud.langfuse.com # or your self-hosted address
```
3. After restarting the services, the system will automatically trace all supported model calls and Agent run trajectories. You can view the detailed execution waterfall diagram and token statistics for each conversation and background task directly in the Langfuse Traces panel.

## 10. What is Wiki mode? How do I use it?

Wiki mode allows the Agent to automatically generate and maintain a structured, interlinked Markdown wiki knowledge base based on the source documents, enabling systematic consolidation and graph-based organization of complex knowledge.

**How to use it**:
1. Go to the settings of the specific **knowledge base** -> **Indexing Strategy**.
2. Enable the **Wiki** indexing feature (this can be combined with enabling **Knowledge Graph** at the same time).
3. When you upload documents to that knowledge base, the system automatically triggers an asynchronous task that uses the LLM to extract entities and core concepts from the documents, and automatically generates structured wiki pages along with knowledge graph links between pages.
4. In the "Wiki" tab of that knowledge base, you can use the dedicated wiki browser to view and manage pages, and view the relationships between different content via the visualized knowledge graph.

## 11. After upgrading to 0.6.0, an operation I used to be able to do now says "Insufficient permissions"?

0.6.0 introduces in-space RBAC (role matrix + resource ownership) — all write endpoints are now authorized based on role + `creator_id`. Common scenarios:

- **You can see it but can't click it**: You're most likely a `Viewer` on this resource, or a `Contributor` who isn't the creator. The UI already hides/disables write actions in this case. Check the role badge under **User menu → Current workspace**.
- **KBs / Agents in a shared space**: KBs shared with you by others are treated as `Viewer` access by default; to get write access, you need to be granted `Admin+` in the source space.
- **API Key calls**: `X-API-Key` synthesizes a virtual user that's fixed as `Admin` of the space it belongs to (only deleting the space requires `Owner`) — scripts generally don't need any migration.
- **Cross-space super admin**: Requires `User.CanAccessAllTenants=true` and `enable_cross_tenant_access=true`, and switching spaces via `X-Tenant-ID`.

If you need to temporarily roll back to an "audit only, no enforcement" grayscale window, you can set `tenant.enable_rbac=false` in the configuration (or the environment variable `WEKNORA_TENANT_ENABLE_RBAC=false`). For the complete role matrix and ownership chain, see [`docs/RBAC-Guide.md`](./RBAC-Guide.md).

## 12. Why doesn't the system automatically return to my last workspace after login?

After upgrading to 0.6.0, the system remembers your "last active workspace" and automatically restores it after login. If it's still not being restored, this is usually because:

1. Your browser cleared LocalStorage, or you switched browsers;
2. The workspace you last visited has removed you (via `/leave` or being removed by an admin) — the system will fall back to the default space;
3. The JWT carries a `tenant_id` that's no longer valid — simply log out and log back in.

## 13. How do I correctly assign permissions for multi-person collaboration?

Following the role matrix in [`docs/RBAC-Guide.md`](./RBAC-Guide.md):

- Read-only users → `Viewer`
- Regular members (upload documents, maintain "their own" KBs / Agents) → `Contributor`
- Operations staff (manage shared models, vector stores, parsers, and other infrastructure) → `Admin`
- Space owner (has permission to delete the space; at least one per space, can have multiple, and the last one can't be demoted or removed) → `Owner`

If you want to enable "invite-only" mode (disallowing self-service sign-up to this space), you can turn on the invitation system in the space settings and issue invite codes or links via the "Invite" entry point.

## 14. Document parsing is stuck at "Processing" / the parsing trace timeline won't open?

Starting in 0.6.1, every document parse records a Langfuse-style span tree (the `knowledge_processing_spans` table), which you can open via the "Trace" entry point on the knowledge base card menu or card itself, to view a step-by-step sidebar timeline. Common scenarios:

- **A document stays at "Processing" for a long time**: First open the timeline to see which stage isn't progressing (parsing / splitting / vectorization / post-processing). 0.6.1 has fixed most "stuck" scenarios and added watchdog polling; if you confirm a particular parse has hung, you can click "Abort parsing" in the timeline panel, and the document will enter the finalizing post-processing state before ending.
- **The timeline keeps showing "Updating" but has no data**: This is usually because the polling request is silently failing (network / reverse proxy truncating the SSE stream). 0.6.1 now explicitly surfaces polling failures — refresh the page or check whether Nginx is buffering the response.
- **No timeline data after upgrading**: Confirm that database migrations `000055_knowledge_processing_spans` and `000056_knowledge_pending_subtasks` have run (these run automatically when the service starts).

## 15. How do I enable OpenSearch as the vector store?

0.6.1 adds an OpenSearch vector store driver (k-NN). In **Settings → Vector Store**, add a new OpenSearch engine and fill in the connection address and credentials — a KB can then be bound to this vector store. Notes:

- The connection address goes through SSRF policy validation; internal / loopback addresses must comply with the allow rules. You can use "Test Connection" to validate first.
- For integration test and index mapping details, see [`docs/dev/opensearch-integration-test.md`](./dev/opensearch-integration-test.md).

## 16. How are built-in models managed declaratively via YAML?

Starting in 0.6.1, the platform's built-in models are driven declaratively by `config/builtin_models.yaml`, which supports `${ENV}` variable interpolation, and stays in sync between the database and YAML via the `managed_by` field and drift reconciliation. Common issues:

- **Changed the YAML but it's not taking effect**: Built-in models undergo lifecycle reconciliation (drift sweep) at service startup; confirm you've restarted the service and that the entries pass schema validation (ID length, required fields).
- **Environment variables not injected under Docker**: `builtin_models` relies on variables being injected via the `env_file` array form — confirm your compose file mounts `.env` in array form.
- Reference example: `config/builtin_models.yaml.example`.

## 17. How do System Admin and platform settings work?

0.6.1 introduces the System Admin role and a unified Platform Settings panel (including a platform audit log), distinct from in-space RBAC: the System Admin manages "platform-level" configuration, not resources within a single space. To enable this for the first time, you need to promote the first admin via the System Admin bootstrap process; revoking admin privileges has safety protections (to avoid an accidental revocation leaving no one able to manage the platform). The related migration is `000053_system_admin_and_settings`.

## 18. How do I customize the parsing configuration (process_config) at upload time?

Starting in 0.6.2, file / URL / folder uploads can carry a `process_config` (`KnowledgeProcessOverrides`), which overrides the knowledge base's default parsing engine, chunking, multimodal (VLM / ASR), question generation, graph extraction, and other settings **for this batch only**, without changing the KB's global configuration. The web UI pops up a confirmation dialog for adjustments before uploading; the API and `weknora doc upload` accept a JSON payload with the same field names.

- **Relationship with KB default configuration**: Fields that aren't passed fall back to the KB default values; `graph_enabled` only takes effect when `extract_config.enabled` is true.
- **Reparsing**: `POST /knowledge/:id/reparse` can pass `process_config` in the body to rerun parsing with new settings — the overrides get written to `knowledge.metadata.process_overrides`.
- **Image / audio validation**: If a batch contains images, the KB must have a VLM configured; if it contains audio, an ASR must be configured, otherwise the upload will be rejected.
- See [`docs/api/knowledge.md`](./api/knowledge.md) for details.

## 19. After upgrading to 0.6.2, the `weknora` CLI login or MCP tools throw errors?

0.6.2 ships with **CLI v0.9** (breaking changes); common migration steps:

- **`auth login` no longer creates a profile**: First run `weknora profile add <name> --host <url> --use`, then `weknora auth login`; switch profiles with the global `--profile <name>`.
- **`auth logout` / `auth refresh` no longer take `--name`**: They now act on the current active profile.
- **The MCP tool `agent_invoke` has been renamed to `session_ask`**: External MCP clients need to refresh their tool schema.
- **`agent create --kb` has become `--attach-kb`**; the `--kb` flag on `doc delete --all` and `search chunks` / `search docs` is now required, and supports either a name or an ID.
- Added `weknora session stop <session-id>` to abort an in-progress Agent run; the repo now ships with the built-in `weknora-rag-search` / `weknora-shared` Skills.
- See [`cli/CHANGELOG.md`](../cli/CHANGELOG.md) for details.

## 20. pgvector retrieval is slow, or what do I need to do right after upgrading?

0.6.2 adds migration `000059_embeddings_hnsw_1024`, which creates an HNSW index on PostgreSQL pgvector for **1024-dimension** embeddings (such as bge-m3). This migration runs automatically at service startup; if you use a different dimension, this index may not apply, and you'll need to tune it separately for your own embedding dimension. During the first large-scale ingestion after upgrading, index building may consume extra I/O — this is expected behavior.

## 21. How do I embed a WeKnora Agent on a website (Embed Widget)?

Starting in 0.6.3, **embed channels** are supported: create an embed channel in the **Integration Center** or the Agent editor, bind it to a custom Agent, and get a channel ID and publishable token (`em_…`) — then embed `weknora-widget.js` into an external web page to provide visitor Q&A.

- **Domain whitelist**: You must fill in the Origins allowed to load the Widget in the channel configuration, otherwise the exchange will return a 403.
- **Secure mode (recommended)**: In production, don't put the `em_…` token directly in the page HTML; instead, have your backend provide a `token-endpoint` that uses the publishable token to call `POST /api/v1/embed/:id/exchange` to obtain a short-lived token `ems_…` (valid for about 30 minutes). See [`docs/embed-secure-mode.md`](./embed-secure-mode.md) and [`docs/embed-subdomain.md`](./embed-subdomain.md) for details.
- **Rate limiting**: Channels can be configured with per-minute / per-day request limits; exceeding the limit returns 429.
- **Subdomain deployment**: If the embed page and API are on different subdomains, see `docs/embed-subdomain.md` for CORS and Nginx configuration.

## 22. How do I set multiple tags on a document?

0.6.3 upgrades document tags from single-select to **multi-tag** (migration `000063_knowledge_multi_tags`). In the knowledge base list, you can assign multiple tags to a document, and the sidebar supports filtering by tag; the **Tag Management** drawer lets you batch-manage tags. In the API, pass a `tag_ids` array when uploading / updating knowledge items (replacing the old single `tag_id`).

## 23. How do I batch reparse documents?

Select multiple documents in the knowledge base document list using checkboxes, then use **Reparse** in the batch action bar; alternatively, call `POST /knowledge/batch-reparse`, with the body optionally containing `ids` and `process_config`. The task is enqueued asynchronously, and the UI refreshes the status once queued. For a single document, you can still use `POST /knowledge/:id/reparse`.

## 24. How do I configure an RSS data source?

0.6.3 adds an **RSS / Atom** connector. In the knowledge base's **Settings → Data Sources**, select RSS, and fill in the Feed URL and sync strategy to pull content into the knowledge base, either as a full sync or incrementally. If some entries fail, the sync log will show partial failure details; editing a data source's configuration and saving it will **not** automatically trigger a sync — you need to click sync manually.

## 25. How do I configure OAuth2 for a remote MCP service?

0.6.3 supports **OAuth2 authorization** for MCP services (migration `000062_mcp_oauth`). In **Settings → MCP**, add an HTTP-type service and select OAuth2, then complete the authorization callback via the wizard; custom HTTP headers and JSON **code import** for quickly pasting configuration are also supported. Authorization tokens are stored encrypted, and you'll need to re-authorize in the UI once they expire.

## 26. How can I override the Embedding dimension?

When editing an Embedding model in **Settings → Models**, you can fill in a **dimensions** override value (e.g. 1024, 1536). 0.6.3 fixed an issue where some providers' requests weren't carrying `dimensions` (#1654). If the vector store's index dimension doesn't match the model, retrieval may behave abnormally — make sure the vector store bound to the KB stays consistent with the model's dimension.

## 27. The Agent shows "Model not ready" and I can't chat?

0.6.3 introduces **model readiness validation** in the Agent selector: if the bound LLM / Embedding / Rerank / VLM is missing or misconfigured, the conversation is blocked and remediation guidance is shown. You can open the **Debug Drawer** on a model card to test connectivity first; confirm that the models referenced by both the KB and the Agent exist and are usable.

## 28. How do I create and scope-restrict an API Key?

0.7.0 introduces the **scoped API Key and Principal model** (migrations `000064_principal_model`, `000065_tenant_api_keys`). An API Key is no longer equivalent to a specific human user — it's now an independent Principal, carrying explicit role and capability authorizations:

- In **Settings → API Integrations** (visible to Owners), you can create a Key and check capabilities (such as `manage_kbs` for full KB lifecycle management, `manage_storage_backends`, etc.), and optionally restrict it to specific knowledge bases.
- The Key's `last_used_at` is updated with throttling, to avoid high-frequency database writes.
- Route-level guards reject out-of-scope access; management endpoints deny API Key Principals by default — for integrations, use a Key with the appropriate capabilities rather than a full-access Key.
- MCP OAuth and embed sessions are isolated by Principal, so different integrations don't cross-contaminate.

### How do I use a single API Key to automate management across multiple spaces?

A SystemAdmin can create a `scope_type=platform` Key under **System Admin → Platform API Keys**. A platform Key isn't bound to a single space: when calling regular space APIs, it must carry `X-Tenant-ID`, and remains subject to the usual capability and knowledge-base-scope guards; calling open system control-plane endpoints requires the corresponding `system_*` capability. Platform Keys don't support `full_access`, and can't create, rotate, or revoke other platform Keys.

## 29. How can a space bind multiple object storage instances?

0.7.0 supports **multi-instance storage backends** (migration `000068_storage_backends`). A space can register multiple storage instances (`local` / `minio` / `cos` / `tos` / `s3` / `oss` / `ks3` / `obs`), with different knowledge bases bound to different instances, and the space also has a default instance:

- In **Settings → Storage Backends**, you can create/test/set an instance as default (requires Admin+; API Keys need the `manage_storage_backends` capability).
- New knowledge bases without an explicit binding use the space's default instance; `access_key_id` / `secret_access_key` are masked in responses, and submitting the masked placeholder on update won't overwrite the real credentials in the database.
- If you're told the storage engine is unavailable when creating a knowledge base, confirm the target provider is within the `STORAGE_ALLOW_LIST` allowed range. See [`docs/api/storage-backend.md`](./api/storage-backend.md) for details.

## 30. What do I do about backlogged parsing/ingestion tasks, or troubleshooting failed tasks?

0.7.0 adds a System Admin **Runtime Task Queue panel** and **worker pool governance**. Document processing has moved from a single aggregated pool to separate pools per stage (core / post-processing / enrichment / maintenance) plus an elastic shared pool, with Wiki governed independently:

- In **System Settings → Runtime Queue**, you can view queue depth, per-model concurrency statistics, and failed task details, and manually retry tasks.
- You can adjust each pool's concurrency via `WEKNORA_ASYNQ_*_CONCURRENCY` environment variables and `asynq.*_concurrency` system settings (requires a service restart); `model.max_concurrency` is used to constrain a single model's background concurrency.
- See [`docs/worker-pool-governance.md`](./worker-pool-governance.md) for details. Note: worker concurrency is only a scheduling budget — it's still constrained by model quotas, DocReader capacity, vector store, and database connection limits.

## 31. How do I temporarily upload an image/document in a conversation for a one-off Q&A?

0.7.0 supports **session-level temporary attachments** (migration `000070_temporary_documents`). You can upload an image or document in the conversation input area; the system parses it asynchronously and uses it only for Q&A in the current session, without writing it to the knowledge base. Images and attachments share a combined count limit; attachment content is retained across multiple turns of conversation.

## 32. How do I integrate with QQBot / Lark (international Feishu)?

0.7.0 adds **QQBot** platform integration, and supports the international version of Feishu, **Lark** (region-aware routing). In **Settings → IM Integrations**, add the corresponding channel and fill in the credentials; Lark replies are sent via the reply-message interface, and replies land in the original message thread.

## 33. How do I enable TLS for Redis?

0.7.0 supports **TLS connections** for Redis (#1930). After enabling TLS via environment variables, the startup log will print the TLS configuration status for confirmation. If the connection fails, check the certificate/CA configuration and whether the Redis server requires TLS.

## 34. After upgrading to 0.7.0, `weknora` CLI commands are missing or behave differently?

0.7.0 ships with **CLI v0.10** (Agent-first, breaking changes): added `model` / `message` / `config` / `skills` command groups, `doc reparse` / `doc update`, `kb config` / `kb config set`; `session continue` has been renamed to `session resume`, and `session tool-approval` has been added; agent-first chat and `session ask` output modes are now provided, along with strengthened SSE reliability and typed errors. See [`cli/CHANGELOG.md`](../cli/CHANGELOG.md) for details.

## 35. How do I integrate with Yunzhijia?

0.7.1 adds **Yunzhijia IM integration**. In **Settings → IM Integrations**, add a Yunzhijia channel and fill in the application credentials; the integration is based on a persistent WebSocket connection to receive messages, supports ingesting image messages (with SSRF-safe downloading), and replies in **Markdown** format by default. If images fail to download, check outbound network access and whether the credentials have download permission.

## 36. How do I use Volcano Engine Rerank / Zhipu AI web search?

0.7.1 adds two new providers:

- **Volcano Engine Rerank**: In **Settings → Models**, add a Rerank model and select Volcano Engine. When a single request's document count exceeds the API limit, the client will automatically batch the requests and merge the results. vLLM Rerank no longer sends `truncate_prompt_tokens` by default, to improve compatibility.
- **Zhipu AI web search**: In **Settings → Web Search**, select Zhipu AI as the search provider and fill in the credentials, for use in Agent web search.

## 37. After upgrading to 0.7.1, conversation memory settings are gone? Do I still need Neo4j?

0.7.1 **removes the Neo4j-based conversation memory (episodic memory)** feature — the related API fields, settings, and embedding toggle have all been removed, and conversations no longer rely on Neo4j for memory recall. **Note: Knowledge Graph (GraphRAG / graph retrieval) still uses Neo4j**, so if you have graph retrieval enabled, Neo4j is still a required component — there's no need to remove it from your deployment. If you previously deployed Neo4j only for the memory feature and don't use the knowledge graph, you can remove it as needed.

## 38. Where can I find the official documentation? How do I run the docs site locally or deploy it standalone?

0.7.2 adds complete official product documentation, located in the repository's [`website-docs/`](../website-docs/README.md) directory, organized into six sections — "Getting Started → Architecture → Features → API → Clients → Development" — covering about 360 API endpoints, about 150 environment variables, and 9 major extension points.

This directory is also a VitePress site, with two ways to use it:

```bash
# Local preview
cd website-docs && npm install && npm run dev

# Standalone container deployment (Nginx inside the container listens on 8081)
docker build -t weknora-docs website-docs
docker run -d -p 8081:8081 weknora-docs
```

The site's version number is automatically read from the repository root's `VERSION` file at build time, so you don't need to manually update the docs after a version upgrade. If a screenshot shows as a dashed placeholder box, it means the corresponding image is missing under `website-docs/public/screenshots/` — just add the image, no Markdown changes needed.

`website-docs/sample-data/` also provides 4 sample Markdown documents and 1 FAQ import JSON, which you can use directly to run through a "create KB → upload → Q&A" flow; `examples/mcp-demo/` is a locally runnable MCP service example.

## 39. After uploading a folder, the document title becomes one long path?

This was the behavior prior to 0.7.2: folder uploads stuffed the relative directory into `file_name`, causing the full path to display in the list, with no way to filter by folder.

0.7.2 splits the path out into a separate `folder_path` field (migration `000079_knowledge_folder_path`), and **automatically backfills historical data**, so after upgrading, existing knowledge bases will also display the correct folder tree without needing to re-upload. A folder tree appears on the left side of the document list, letting you browse and rename folders like a file manager; you can also use the inline folder picker to re-file a document into a different folder. Documents uploaded individually (not via folder) are all attached at the root of the tree.

## 40. Chunk content is inaccurate — can I edit it manually? Does editing rebuild the index?

Yes. 0.7.2 supports editing retrieval chunks directly in the interface (migration `000078_chunk_editing_and_custom_metadata`):

- Every edit saves the pre-edit version to `chunk_revisions`, and you can view a version-by-version diff in "Chunk Edit History" and roll back with one click.
- **The chunk's index is automatically rebuilt after an edit is saved** (the `index_status` field tracks rebuild status), so no manual reparse is needed.
- A chunk's generated questions can be individually added, removed, edited, and regenerated, and are preserved after content edits.
- Note: reparsing the entire document rebuilds chunks based on the new parsing result, and any manual edits made previously will not be preserved — proceed with caution.

## 41. A Wiki page got overwritten by the Agent — can I recover an old version?

Yes. 0.7.2 introduces version history for wiki pages (migration `000075_wiki_page_revisions`): a snapshot is saved every time a page is about to be overwritten. Open the "Version History" drawer in the top-right corner of the wiki browser to view the full history and line-level diffs, and roll back to any version with one click. Each version records its source (`pipeline` for the pipeline, `agent` for the repair tool, `user` for manual edits, or `revert` for a rollback), making it easy to tell who made a change.

Additionally, 0.7.2 removes the duplicate operation log in the wiki browser (migration `000077_remove_wiki_log`) — wiki change records are now consolidated into the **knowledge base activity feed**.

## 42. Third-party apps get an image link of `resource://...` that won't display — what do I do?

By default, the API returns an internal handle `resource://<handle>`, and the client needs to call the authenticated `/files` proxy again to fetch the image. 0.7.2 adds a direct-link mode, letting the API return a loadable http(s) link directly:

- **Single request**: Add `?resource_urls=public` to the URL.
- **Entire deployment**: Set the environment variable `RESOURCE_URL_MODE=public`.

Notes:

- Direct links depend on `APP_EXTERNAL_URL` (or the storage backend itself being publicly reachable) to be generated; if it can't be generated, the reference stays as `resource://`, and the client can still fall back to `/files`.
- `public` issues a **time-limited, anonymously readable** link for each referenced file (2 hours on the WeKnora side, 24 hours on MinIO) — evaluate whether this meets your security requirements.
- Anonymous embed channels and API Keys scoped to specific knowledge bases **always return a handle**, unaffected by this variable.
- It's recommended to also configure `SYSTEM_AES_KEY`, so grant rows can be reused, direct-link URLs stay stable, and write pressure on read endpoints is reduced.

See [API Documentation · File and Image References](./api/README.md) for details.

## 43. Using AWS S3 but don't want to put an AK/SK in the configuration?

0.7.2 supports the **AWS SDK default credential chain** (#2008): leave **both** `S3_ACCESS_KEY` and `S3_SECRET_KEY` empty, and the SDK will try EC2/ECS/EKS instance roles, IRSA / Web Identity, environment variables, and shared configuration files, in that order. Note that both must be either filled in or left empty together — filling in only one will cause a configuration error. `S3_ENDPOINT` can also be left empty, in which case the standard AWS endpoint corresponding to `S3_REGION` is used.

## 44. The MCP Server fails to start with `uvx`, or which package should I install?

Please install the **official package `tencent-weknora-mcp`** (published by the Tencent/WeKnora repository's CI via Trusted Publishing). The previous community package `weknora-mcp` was not officially maintained — please migrate your install command.

0.7.2 ships with MCP Server 1.1.x, which has migrated to the mcp 2.x advanced `MCPServer` API, fixing a startup crash when `uvx` pulls SDK 2.x (`AttributeError: 'Server' object has no attribute 'list_tools'`), and restoring routing compatibility for HTTP (`stateless_http`) and SSE (`/sse/messages/`) transports. The total tool count is 29, with new additions `create_knowledge_from_text` (create a knowledge entry directly from Markdown text) and `list_shared_knowledge_bases` (shared knowledge bases are now also included in name-based resolution).

Behavior change notice: When a tool execution fails, MCPServer 2.x returns `CallToolResult(isError=True)`, instead of returning a successful response with an `"Error executing …"` text prefix as the old low-level API did. Clients that only parse `content[0].text` will generally be unaffected, while integrations that rely on the `isError` flag will now behave more in line with the MCP specification.

## P.S.
If the above methods don't resolve your issue, please describe your problem in an issue and provide the necessary log information to help us troubleshoot.
