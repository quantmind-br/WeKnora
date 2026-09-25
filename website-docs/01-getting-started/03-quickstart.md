# Quick Start

By signing up, creating a knowledge base, configuring models, uploading documents, and asking a question, you can complete your first knowledge base Q&A and view source citations in the answer. The steps below use the web UI; the matching API example is at the end of this document.

Before you begin, complete [Installation & Deployment](02-installation.md) and have a usable chat model and embedding model ready.

## Before you start {#_1-before-you-start}

- Start the service: after starting it following [Installation & Deployment](./02-installation.md), the frontend is at `http://localhost` and the backend at `http://localhost:8080`;
- Prepare model connection details: a local Ollama (default address inside the container is `http://host.docker.internal:11434`), or any OpenAI-compatible service with a `base_url` + `api_key`. You need at least one chat model and one embedding model;
- Check the backend health: `curl http://localhost:8080/health` returns `{"status":"ok"}`.

## Sign up and log in {#_2-sign-up-and-log-in}

On your first visit you'll land on the login page. When the deployment allows public registration (`self_serve`), the page shows a registration tab; the system has no default account. With the default configuration, registering creates a personal workspace and makes the new user the Owner of that workspace.

<Screenshot
  src="/screenshots/quickstart-register.png"
  caption="The registration page on first visit"
  hint="Show the registration form (username / email / password) along with the login link." />

Registration requirements and deployment differences:

- Usernames are 2–50 characters; passwords must be 8–32 characters and contain at least letters and digits. When the complex password policy is enabled, they must also contain both uppercase and lowercase letters and special characters; the UI and the API use the same policy;
- Team deployments can disable public registration and add members afterward via invite links. You can set `DISABLE_REGISTRATION=true` (forces the registration mode to `invite_only` at startup), or have a system admin change `auth.registration_mode` to `invite_only` under "Settings → System" (takes effect immediately, no restart needed);
- If the deployment has the default workspace policy set to `tenantless` (`auth.default_tenant_mode`), a workspace is **not** automatically created after registration — instead you're guided to `/onboarding/workspace`, where you must create your own workspace or accept an invitation to join one before continuing;
- The desktop app requires no registration — a local account is created and logged in automatically on startup; the Lite single binary still requires registration and login when accessed in a browser.

::: tip Workspace and Platform Permissions
A workspace Owner manages the members, models, and knowledge bases of that workspace. Global system settings, the platform task queue, and cross-workspace auditing require the system admin identity; the two kinds of permission are granted independently.

To set up the first system admin, register an account first, then set `WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL=<that account's email>` on the app service and restart it. This flow only takes effect when the deployment has no system admin yet. For the full steps and limitations, see [Platform Management and System Administrators](../03-features/20-platform-admin.md).
:::

## Create a knowledge base and configure a model {#_3-create-a-knowledge-base-and-configure-a-model}

After you create a knowledge base on the "Knowledge Bases" page, the initialization wizard guides you through configuring the models that base uses. Each knowledge base selects its models separately.

1. On the "Knowledge Bases" page, click New, enter a name, and choose a type: `document` (regular document base) or `faq` (Q&A pair base);
2. In the initialization wizard that pops up, pick your models:
   - **Chat model (LLM)**: generates answers;
   - **Embedding model**: converts documents into vectors; changing it requires rebuilding the index;
   - Rerank, VLM for image understanding, ASR for speech transcription, knowledge graph extraction, and question generation can be configured according to your material types and needs;
3. Use the "Test" button in the wizard to confirm the model connection works, then save.

<Screenshot
  src="/screenshots/quickstart-init-wizard.png"
  caption="Initialization wizard: choosing a chat model and an embedding model for the knowledge base"
  hint="Show the model source (Ollama / remote API), model name, Base URL input field, and a message confirming the connectivity test passed." />

::: tip Connecting to Ollama from a container
Inside the backend container, `localhost` points to the container itself. To connect to Ollama on the host machine, use `http://host.docker.internal:11434`.
:::

## Upload documents {#_4-upload-documents}

After entering the knowledge base, drag in files or paste a web page URL. In the upload confirmation dialog, you can set tags and parsing options for this batch of files.

Supported formats include PDF, Word, Excel, PPT, Markdown, HTML, EPUB, images, audio, and more — see [Document Parsing Service](../03-features/03-document-parsing.md) for the full list.

<Screenshot
  src="/screenshots/quickstart-upload.png"
  caption="Upload confirmation dialog: selecting files, adding tags, adjusting parsing options"
  hint="Show the list of files pending upload, tag selection, and parsing engine options." />

After uploading, documents are parsed asynchronously, moving through the states `pending → processing → finalizing → completed`. Scanned PDFs and large files take longer; the list page refreshes progress in real time.

<Screenshot
  src="/screenshots/quickstart-document-list.png"
  caption="Document list: three documents finished parsing"
  hint="Show columns for document name, type, parsing status as “Completed”, chunk count, etc." />

## Ask a question {#_5-ask-a-question}

Go to the chat page and select the knowledge base, then ask your question. The default "Quick Q&A" Agent retrieves relevant chunks and generates an answer; click a citation to view the source text.

<Screenshot
  src="/screenshots/quickstart-chat.png"
  caption="Knowledge Q&A: an answer with clickable citation sources"
  hint="Show one round of Q&A, with citation markers in the answer body and the expanded citation source panel." />

If the answer displays correctly and its citations can be opened, this round of document ingestion and Q&A is complete.

## Going further {#_6-going-further}

- [Configure Agents](../03-features/07-agent.md): use Smart Reasoning for multi-step questions, and enable web search and MCP tools as needed.
- [Adjust chunking](../03-features/04-chunking.md) and [retrieval parameters](../03-features/05-retrieval-engines.md): tune the configuration based on document structure and retrieval results.
- [Connect data sources](../03-features/10-datasource.md): continuously sync content from Feishu, Notion, Yuque, or RSS.
- [Connect IM](../03-features/12-im-integration.md) or [embed in a web page](../03-features/13-embed-channel.md): let users ask questions through channels they already use.

## Completing your first Q&A via the API {#_7-walking-through-the-same-flow-via-the-api}

The example below calls the API in order: register, log in, create a knowledge base, initialize models, upload, and ask. All paths use the `/api/v1` prefix.

```bash
BASE=http://localhost:8080/api/v1

# 1) Register (for the first deployment; username 2–50 chars; password 8–32 chars with letters and digits, stricter if the complex policy is enabled)
curl -s -X POST $BASE/auth/register -H "Content-Type: application/json" \
  -d '{"username":"admin","email":"admin@example.com","password":"pass123456"}'

# 2) Log in and get the JWT
TOKEN=$(curl -s -X POST $BASE/auth/login -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"pass123456"}' | jq -r '.token')
AUTH="Authorization: Bearer $TOKEN"

# 3) Create a knowledge base
KB_ID=$(curl -s -X POST $BASE/knowledge-bases -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"My knowledge base","description":"demo","type":"document"}' | jq -r '.data.id')

# 4) Initialize the knowledge base (using local Ollama as an example; change source/baseUrl/apiKey for a remote model)
curl -s -X POST $BASE/initialization/initialize/$KB_ID -H "$AUTH" -H "Content-Type: application/json" -d '{
  "llm":       {"source":"local","modelName":"qwen3:8b"},
  "embedding": {"source":"local","modelName":"bge-m3","dimension":1024},
  "rerank":    {"enabled":false},
  "multimodal":{"enabled":false},
  "documentSplitting":{"chunkSize":512,"chunkOverlap":50,"separators":["\n\n","\n","。"]},
  "nodeExtract":{"enabled":false},
  "questionGeneration":{"enabled":false}}'

# 5) Upload a document (multipart, field name "file")
curl -s -X POST $BASE/knowledge-bases/$KB_ID/knowledge/file -H "$AUTH" \
  -F "file=@./demo.pdf"
# Poll parsing status: GET /knowledge-bases/$KB_ID/knowledge until parse_status=completed

# 6) Create a session
SESSION_ID=$(curl -s -X POST $BASE/sessions -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"title":"First conversation"}' | jq -r '.data.id')

# 7) Knowledge Q&A (SSE streaming output)
curl -N -X POST $BASE/knowledge-chat/$SESSION_ID -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"query":"What does this document cover?","knowledge_base_ids":["'$KB_ID'"]}'

# 7b) Agent chat (also SSE; agent_id can be the built-in builtin-smart-reasoning)
curl -N -X POST $BASE/agent-chat/$SESSION_ID -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"query":"Summarize the document's key points and list the evidence","agent_enabled":true,"agent_id":"builtin-smart-reasoning","knowledge_base_ids":["'$KB_ID'"]}'

# 8) Retrieval only, no generation (structured JSON result)
curl -s -X POST $BASE/knowledge-search -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"query":"keyword","knowledge_base_ids":["'$KB_ID'"]}'
```

The chat request body also supports fields like `knowledge_ids` (restrict to a single document), `web_search_enabled`, `summary_model_id`, `mcp_service_ids`, `skill_names`, and `images` / `attachment_uploads` (multimodal attachments) — see [API Reference: Sessions & Chat](../04-api/02-api-chat.md) for the full details.

### Three authentication methods

| Method | Request header | Use case |
| --- | --- | --- |
| JWT | `Authorization: Bearer <token>` | Browser / interactive calls, issued by the login endpoint |
| API Key | `X-API-Key: <key>` | Server-side integration; created in "Workspace Settings" or via `POST /api/v1/tenants/:id/api-keys`, supports fine-grained capabilities (`retrieve`/`chat`/`ingest`/`manage_kbs`, etc.) |
| Specify workspace | `X-Tenant-ID: <id>` | For multi-workspace users to switch their current active workspace |

For server-side integration, an API Key is recommended over a JWT:

```bash
# Create an API Key as the Owner (TENANT_ID comes from the login response)
curl -s -X POST $BASE/tenants/$TENANT_ID/api-keys -H "$AUTH" -H "Content-Type: application/json" \
  -d '{"name":"ci-bot","full_access":true}'
# All subsequent requests then use:
curl -s $BASE/knowledge-bases -H "X-API-Key: <key returned at creation time>"
```

### Endpoints corresponding to the initialization wizard

Every step of the on-screen wizard has its own dedicated endpoint, which you can reuse directly if you're building your own admin console:

| Step | Endpoint | Description |
| --- | --- | --- |
| Read current config | `GET /api/v1/initialization/config/:kbId` | Returns the llm / embedding / rerank / multimodal / documentSplitting / nodeExtract / questionGeneration sections plus `hasFiles` (restricts embedding changes once files already exist) |
| Detect Ollama | `GET /api/v1/initialization/ollama/status`, `GET /api/v1/initialization/ollama/models` | Checks Ollama availability and already-installed models |
| Download an Ollama model | `POST /api/v1/initialization/ollama/models/download` → `GET /api/v1/initialization/ollama/download/progress/:taskId` | Asynchronous download with progress polling |
| Test a remote model | `POST /api/v1/initialization/remote/check`, `/initialization/embedding/test`, `/initialization/rerank/check`, `/initialization/asr/check`, `/initialization/multimodal/test` | Connectivity verification before saving |
| Trial knowledge graph extraction | `POST /api/v1/initialization/extract/text-relation` (paired with `fabri-text` / `fabri-tag` to generate an example) | Preview entity/relation extraction results |
| Save config | `POST /api/v1/initialization/initialize/:kbId` (first time) / `PUT /api/v1/initialization/config/:kbId` (update) | Persists the config: creates/updates Model records and writes the KnowledgeBase configuration |

`source` can be `local` (Ollama) or a remote vendor identifier (`openai`, `deepseek`, `aliyun`, `zhipu`, `siliconflow`, etc.). Valid range for `chunkSize` is 100–10000.

### What happens across the whole flow

```mermaid
sequenceDiagram
    autonumber
    participant U as "User (browser)"
    participant FE as "frontend (Nginx)"
    participant APP as "app backend (:8080)"
    participant DR as "docreader (gRPC)"
    participant DB as "ParadeDB / vector index"
    participant LLM as "LLM (Ollama / remote API)"
    U->>FE: Register / log in
    FE->>APP: POST /api/v1/auth/register → login
    APP-->>FE: JWT + auto-created tenant
    U->>APP: POST /api/v1/knowledge-bases (create knowledge base)
    U->>APP: POST /api/v1/initialization/initialize/:kbId (configure models)
    APP->>LLM: Connectivity test (remote/check, embedding/test)
    U->>APP: POST /api/v1/knowledge-bases/:id/knowledge/file (upload)
    APP->>DR: gRPC document parsing (OCR / layout / images)
    DR-->>APP: Structured text + images
    APP->>DB: Chunking → Embedding → vector/keyword index (async via Asynq)
    U->>APP: POST /api/v1/sessions (create session)
    U->>APP: POST /api/v1/knowledge-chat/:session_id (ask a question)
    APP->>DB: Hybrid retrieval (vector+BM25) → RRF → Rerank
    APP->>LLM: Assemble context and generate an answer
    APP-->>U: SSE streaming answer + citation sources
```

## Stuck? Check here {#_8-stuck-check-here}

| Symptom | What to check |
| --- | --- |
| Stuck on `processing` after upload | `docker logs WeKnora-docreader`; large files are bound by `MAX_FILE_SIZE_MB` (default 50) and `WEKNORA_DOCUMENT_PROCESS_TIMEOUT` (default 2h) |
| Ollama detection fails during initialization | The default address inside the container is `http://host.docker.internal:11434` (`OLLAMA_BASE_URL`); on Linux make sure `extra_hosts: host.docker.internal:host-gateway` is in effect |
| No citations in answers / empty retrieval | Confirm document parsing is `completed`; lower `vector_threshold`; check that the embedding model matches the one used when the base was created |
| Registration tab missing | Check `registration_mode` from `GET /auth/config`. Its value may come from a database setting under "Settings → System", not just `DISABLE_REGISTRATION`; invite links and OIDC first-time login are two other paths that aren't affected by it |
| API Key request returns 403 | The key's capabilities don't include what's needed, or `knowledge_base_ids` whitelist doesn't include the target knowledge base |

Next steps: for detailed configuration options see [Configuration Reference](./04-configuration.md); to understand how the system works overall see [Architecture Overview](../02-architecture/01-overview.md).
