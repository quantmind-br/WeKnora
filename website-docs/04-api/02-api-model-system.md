# API Reference: Models and Initialization

Manage models, test connections, initialize Knowledge Bases, and start evaluation tasks. The WeKnoraCloud endpoints are used to connect to the related cloud services.

For system information and system administration (`/system`, `/system/admin`) endpoints, see [System and Platform Management](./02-api-system.md).

## Models (/api/v1/models)

API key: `manage_models` or full-access.

### GET /api/v1/models/providers

Purpose: provider catalog (the frontend uses it to dynamically render the provider dropdown, icons, extra fields, and model selection). Permission: Viewer+. Query parameters: `model_type` (optional: `chat/embedding/rerank/vllm/asr`; backend values such as `KnowledgeQA` are also accepted; unknown values return 400). When specified, only providers that support that type, and only models of that type, are returned. Handler: `internal/handler/model_catalog.go`

Response: 200 `{"success":true,"data":[ModelProviderDTO]}`, where each item contains:

| Field | Description |
| --- | --- |
| `value` / `label` / `labels` / `description` / `descriptions` / `website` | Provider id, brand name, and per-language names and descriptions |
| `icon` | `data:image/svg+xml;base64,...`, usable directly in `<img src>` |
| `api` / `auth` / `requiresAuth` | Default protocol (`openai-completions`, etc.), authentication method, and whether a key is required |
| `defaultUrls` / `modelTypes` | Default URLs per model type and the supported types. `defaultUrls` is returned only to Admin+ (or a full-access / `manage_tenant_settings` API key); it is empty for other callers |
| `extraFields` | Definitions of provider-specific extra configuration fields (`key,label,labels,type,required,default,placeholder,options,model_types,secret`); values are stored in `parameters.extra_config` |
| `credentialLabels` | Names and hints of the credential input for some model types (for example, the API Key field for Volcengine and LKEAP rerank is actually an Access Key ID / SecretId) |
| `models` | Built-in model catalog (`id,name,type,api,reasoning,input,context_window,max_output_tokens,dimension,thinking_levels,cost,source`); `source` is the link to the provider documentation the model parameters are based on |
| `thinking` | Provider-level thinking encoding summary (`format`, `levels`) |
| `order` | Sort value in the list |

```bash
curl "$BASE/api/v1/models/providers?model_type=chat" -H "Authorization: Bearer $TOKEN"
```

### GET|POST /api/v1/models/catalog/resolve

Purpose: resolve the effective access configuration (protocol, thinking levels, context) from the provider, model name, `base_url`, and `extra_config`, for live display in the model editor. Permission: Viewer+.

Parameters (query parameters for GET, JSON request body for POST, with the same fields): `provider` (provider ID), `model`, `base_url`, `model_type` (default `chat`), `api`, `thinking_control`, `remote_model_name`, plus any non-secret extra fields the provider declares (such as Azure's `api_version`). The POST request body can also carry a `spec` object (same as the model's `parameters.spec`) to preview a single-row catalog override. Secret fields are never accepted.

Response: 200 `{"success":true,"data":{provider,api,remote_model,cataloged,model,capabilities,base_url,url}}`, where `capabilities` is `{provider,api,cataloged,reasoning,thinking_levels,thinking_format,input,context_window,max_output_tokens,max_tokens_field}`. `base_url` and `url` (the actual request URL, returned only for providers that compute the address themselves, such as Azure) are returned only to Admin+ (or a full-access / `manage_tenant_settings` API key). Returns 400 when the configuration cannot be resolved. The same `capabilities` structure is also returned in `ModelResponse.capabilities` for remote chat/vision models.

```bash
curl "$BASE/api/v1/models/catalog/resolve?provider=deepseek&model=deepseek-v4-pro" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/models

Purpose: create a model. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required"`) | Model name |
| `display_name` | string | No | Display name |
| `type` | string | Yes (`binding:"required"`) | Model type: `KnowledgeQA` / `Embedding` / `Rerank` / `VLLM` / `ASR` (stored as-is; frontend forms such as `chat` are not accepted) |
| `source` | string | Yes (`binding:"required"`) | Source (`local` / `remote`) |
| `description` | string | No | Description |
| `parameters` | object | Yes (`binding:"required"`) | Connection parameters (`base_url`, `provider`, `extra_config`, `spec`, `context_window`, etc.; see [Model Management](../03-features/06-models.md#model-configuration-fields) for the fields). `api_key` / `app_secret` can be passed directly at creation time and changed later through the credentials sub-resource |

`parameters` is validated against the model catalog (unknown protocols, wrong compat keys, and invalid thinking levels return 400), and `base_url` goes through SSRF validation.

Response: 201 `{"success":true,"data":{ModelResponse}}` (`id,name,type,source,parameters,is_default,is_builtin,status,credentials,capabilities,...`; the response contains no keys)

```bash
curl -X POST $BASE/api/v1/models -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"gpt-5.5","type":"KnowledgeQA","source":"remote","parameters":{"provider":"openai","base_url":"https://api.openai.com/v1","api_key":"sk-..."}}'
```

### GET /api/v1/models

Purpose: list models. Permission: Viewer+.

Response: 200 `{"success":true,"data":[ModelResponse]}`

```bash
curl $BASE/api/v1/models -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/models/:id

Purpose: model details. Permission: Viewer+.

Response: 200 `{"success":true,"data":{ModelResponse}}`

```bash
curl $BASE/api/v1/models/m-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/models/:id/debug

Purpose: debug a saved model (triggers a real upstream call, which incurs costs). Permission: Admin+. form-data fields: `input` (≤64KB), `options` (JSON-encoded debug options: `system_prompt`, `temperature` (0~2), `top_p`, `max_tokens` (1~8192), `thinking`, `reasoning_effort` (`off/auto/minimal/low/medium/high/xhigh/max`; when set, it overrides `thinking`)), `documents` (JSON array, ≤100 items), `file` (optional).

Response: 200 `{"success":true,"data":{"ok",elapsed_ms,request,raw_response,observations,error}}`

```bash
curl -X POST $BASE/api/v1/models/m-1/debug -H "Authorization: Bearer $TOKEN" -F 'input=hello'
```

### PUT /api/v1/models/:id

Purpose: update a model (built-in models are restricted to SystemAdmin at the service layer). Permission: Admin+ or SystemAdmin (`AdminOrSystemAdmin`). Request body: `name`, `display_name` (pointer), `description`, `parameters` (existing stored keys are preserved), `source`, `type` (all optional).

Response: 200 `{"success":true,"data":{ModelResponse}}`

```bash
curl -X PUT $BASE/api/v1/models/m-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"display_name":"GPT-4o mini"}'
```

### DELETE /api/v1/models/:id

Purpose: delete a model. Permission: Admin+.

Response: 200 `{"success":true,"message":"Model deleted"}`

When the model is still referenced by a knowledge base, agent, or long-term memory in the current space, the response is HTTP 400; the compatibility `message` is kept, along with `error.code=2300`, and `error.details` lists the specific objects and where they reference the model:

```json
{
  "success": false,
  "error": {
    "code": 2300,
    "message": "model is used by 2 knowledge base(s); reconfigure or remove those references before deleting",
    "details": {
      "knowledge_bases": [
        {"id": "kb-1", "name": "Product docs", "bindings": ["vlm_model"]},
        {"id": "kb-2", "name": "Engineering", "bindings": ["vlm_model"]}
      ],
      "agents": [],
      "long_term_memory": {"bindings": []},
      "knowledge_base_total": 2,
      "agent_total": 0
    }
  }
}
```

Knowledge base binding values: `embedding_model`, `summary_model`, `image_processing_model`, `vlm_model`, `asr_model`, `wiki_synthesis_model`, `auto_tag_model`; agent binding values: `chat_model`, `rerank_model`, `vlm_model`, `asr_model`, `query_understand_model`, `follow_up_model`; long-term memory binding values: `embedding_model`, `extract_model`. The details include each object's `id`, `name`, and merged `bindings`, plus `knowledge_base_total` / `agent_total`. Each list holds at most 50 entries; the deletion guard is based on the totals.

```bash
curl -X DELETE $BASE/api/v1/models/m-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/models/:id/credentials

Purpose: set model credentials (keys are not transmitted via the main PUT). Permission: Admin+ or SystemAdmin. Handler: `internal/handler/model_credentials.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `api_key` | *string | No | New API Key |
| `app_secret` | *string | No | New App Secret (when both are omitted, only status is returned) |

Response: 200 `{"success":true,"data":{"fields":{"api_key":{"configured":bool},"app_secret":{"configured":bool}}}}`

```bash
curl -X PUT $BASE/api/v1/models/m-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"api_key":"sk-..."}'
```

### DELETE /api/v1/models/:id/credentials/:field

Purpose: delete a specific credential field (`api_key` or `app_secret`). Permission: Admin+ or SystemAdmin.

Response: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/models/m-1/credentials/api_key -H "Authorization: Bearer $TOKEN"
```

## WeKnoraCloud

Handler: `internal/handler/weknoracloud.go`. API key: `manage_models`/full.

### POST /api/v1/weknoracloud/credentials

Purpose: save WeKnoraCloud SaaS credentials. Permission: Admin+. Request body: `{"app_id":"...","app_secret":"..."}` (both `binding:"required"`).

Response: 200 `{"success":true,"message":"Credentials saved successfully"}`

```bash
curl -X POST $BASE/api/v1/weknoracloud/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"app_id":"app","app_secret":"secret"}'
```

### GET /api/v1/models/weknoracloud/status

Purpose: probe WeKnoraCloud readiness status. Permission: Viewer+.

Response: 200 service status object.

```bash
curl $BASE/api/v1/models/weknoracloud/status -H "Authorization: Bearer $TOKEN"
```

## Initialization (/api/v1/initialization)

Handler: `internal/handler/initialization.go`. KB configuration endpoints: API key `manage_kbs` (write) / `retrieve` (read); model detection endpoints: `manage_models` (all can use full-access).

### GET /api/v1/initialization/config/:kbId

Purpose: read the KB's current model/parsing configuration. Permission: Viewer+, KB read.

The model `baseUrl` is returned only to Admin+ of the space that owns the KB (or a full-access / `manage_tenant_settings` API key). A space accessing the KB through organization sharing can only see whether credentials are configured (`credentials.*`), not the source space's model URLs or bucket information.

Response: 200 `{"success":true,"data":{"hasFiles",llm,embedding,rerank,multimodal,documentSplitting,nodeExtract,questionGeneration}}`

```bash
curl $BASE/api/v1/initialization/config/kb-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/initialization/initialize/:kbId

Purpose: initialize the KB's model and parsing configuration (first-time setup wizard). Permission: KB creator OR Admin+, KB write.

Only the space that owns the KB can call this endpoint; a space that has edit permission through organization sharing is rejected (403). When the KB already has bound models, this endpoint updates those models' configuration in place, which requires the same permission as `PUT /models/:id` (Admin+, or an API key with the `manage_models` capability); otherwise it returns 403.

Main fields (`InitializationRequest`):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `llm.source` / `llm.modelName` | string | Yes | LLM source and model name |
| `llm.baseUrl` / `llm.apiKey` | string | No | Connection parameters |
| `embedding.source` / `embedding.modelName` | string | Yes | Embedding model |
| `embedding.baseUrl` / `embedding.apiKey` / `embedding.dimension` | — | No | Connection and dimension |
| `rerank.enabled` + `rerank.modelName/baseUrl/apiKey` | — | No | Rerank configuration |
| `multimodal.enabled` + `multimodal.vlm.*` + `multimodal.storageType` + `multimodal.cos.*|minio.*` | — | No | Multimodal and image storage |
| `documentSplitting.chunkSize` / `separators` | int / []string | Yes | Chunking configuration |
| `documentSplitting.chunkOverlap` | int | No | Overlap |
| `nodeExtract.*` | — | No | Graph extraction (enabled/text/tags/nodes/relations) |
| `questionGeneration.*` | — | No | Question generation (enabled/questionCount) |

Response: 200 `{"success":true,"message":"Knowledge base configuration updated successfully","data":{"models":[Model],"knowledge_base":{KnowledgeBase}}}`

```bash
curl -X POST $BASE/api/v1/initialization/initialize/kb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"llm":{"source":"remote","modelName":"gpt-4o-mini"},"embedding":{"source":"remote","modelName":"text-embedding-3-small"},"documentSplitting":{"chunkSize":512,"separators":["\n\n"]}}'
```

### PUT /api/v1/initialization/config/:kbId

Purpose: update the KB's model/chunking configuration (`KBModelConfigRequest`: `llmModelId` required; `embeddingModelId`, `vlm_config`, `asr_config`, `documentSplitting.*`, `multimodal.enabled`, `storageProvider`, `storageBackendId`, `nodeExtract.*`, `questionGeneration.*` optional). Permission: KB creator OR Admin+, KB write.

When accessed through organization sharing, the effective share permission must be admin; editor can only edit content, not change settings (403). The storage binding (`storageBackendId` / `storageProvider`) can only be changed by the space that owns the KB; other spaces submitting a value different from the current one get 403.

Response: 200 `{"success":true,"message":"Configuration updated successfully"}`

```bash
curl -X PUT $BASE/api/v1/initialization/config/kb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"llmModelId":"m-1","embeddingModelId":"m-2"}'
```

### GET /api/v1/initialization/ollama/status

Purpose: probe Ollama availability. Permission: Viewer+.

Response: 200 `{"success":true,"data":{"available","version","baseUrl","error"}}`

```bash
curl $BASE/api/v1/initialization/ollama/status -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/initialization/ollama/models

Purpose: list local Ollama models. Permission: Viewer+.

Response: 200 `{"success":true,"data":{"models":[...]}}`

```bash
curl $BASE/api/v1/initialization/ollama/models -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/initialization/ollama/models/check

Purpose: batch-check whether models already exist. Permission: Admin+. Request body: `{"models":["llama3"]}` (`binding:"required"`).

Response: 200 `{"success":true,"data":{"models":{"llama3":true}}}`

```bash
curl -X POST $BASE/api/v1/initialization/ollama/models/check -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"models":["llama3"]}'
```

### POST /api/v1/initialization/ollama/models/download

Purpose: pull an Ollama model (asynchronous task). Permission: Admin+. Request body: `{"modelName":"llama3"}` (`binding:"required"`).

Response: 200 `{"success":true,"data":{"taskId","modelName","status","progress"}}`

```bash
curl -X POST $BASE/api/v1/initialization/ollama/models/download -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"modelName":"llama3"}'
```

### GET /api/v1/initialization/ollama/download/progress/:taskId

Purpose: download task progress. Permission: Viewer+.

Response: 200 `{"success":true,"data":{id,modelName,status,progress,message,startTime,endTime}}`

```bash
curl $BASE/api/v1/initialization/ollama/download/progress/task-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/initialization/ollama/download/tasks

Purpose: list all download tasks. Permission: Viewer+.

Response: 200 `{"success":true,"data":[DownloadTask]}`

```bash
curl $BASE/api/v1/initialization/ollama/download/tasks -H "Authorization: Bearer $TOKEN"
```

### Model connectivity checks (all POST, permission Admin+)

The request body uniformly follows `ModelTestRequest`:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `source` | string | No | Defaults to `remote` |
| `modelName` | string | Yes | Model name |
| `baseUrl` / `apiKey` / `appSecret` | string | No | Connection parameters |
| `provider` / `interfaceType` | string | No | Provider/interface type |
| `dimension` / `supportsDimensionOverride` | int / bool | No | Embedding dimension; whether to specify the dimension in the request |
| `customHeaders` / `extraConfig` | map | No | Extensions |
| `spec` | object | No | Single-row catalog override, same as the model's `parameters.spec` |
| `modelId` | string | No | ID of an existing stored model: keys, `extraConfig`, and `spec` missing from the request are filled in from that model |

| Endpoint | Purpose | Response data |
| --- | --- | --- |
| `POST /api/v1/initialization/remote/check` | LLM remote connectivity | `{available,message}` |
| `POST /api/v1/initialization/embedding/test` | Embedding test | `{available,message,dimension}` |
| `POST /api/v1/initialization/rerank/check` | Rerank test | `{available,message}` |
| `POST /api/v1/initialization/asr/check` | ASR test | `{available,message}` |

```bash
curl -X POST $BASE/api/v1/initialization/remote/check -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"modelName":"gpt-4o-mini","baseUrl":"https://api.openai.com/v1","apiKey":"sk-..."}'
```

### POST /api/v1/initialization/multimodal/test

Purpose: end-to-end multimodal (VLM + image storage) test. Permission: Admin+. multipart fields: `image` (required), `vlm_model`, `vlm_base_url` (required), `vlm_api_key`, `vlm_interface_type`, `storage_type` (`cos|minio`, required) and the corresponding `cos_*`/`minio_*` fields, `chunk_size`, `chunk_overlap`, `separators`.

Response: 200 `{"success":true,"data":{"success","caption","ocr","processing_time"}}`

```bash
curl -X POST $BASE/api/v1/initialization/multimodal/test -H "Authorization: Bearer $TOKEN" \
  -F 'image=@demo.png' -F 'vlm_model=qwen-vl' -F 'vlm_base_url=http://x' -F 'storage_type=minio'
```

### POST /api/v1/initialization/extract/text-relation

Purpose: text graph extraction test. Permission: Admin+. Request body: `text` (required, ≤5000 characters), `tags` (required, at least one), `model_id` (required).

Response: 200 `{"success":true,"data":{"nodes":[GraphNode],"relations":[GraphRelation]}}`

```bash
curl -X POST $BASE/api/v1/initialization/extract/text-relation -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"text":"Xiao Ming works at Tencent","tags":["person","company"],"model_id":"m-1"}'
```

### POST /api/v1/initialization/extract/fabri-tag

Purpose: generate sample tags. Permission: Admin+. No request body.

Response: 200 `{"success":true,"data":{"tags":[...]}}`

```bash
curl -X POST $BASE/api/v1/initialization/extract/fabri-tag -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/initialization/extract/fabri-text

Purpose: generate sample text based on tags. Permission: Admin+. Request body: `{"tags":[...],"model_id":"m-1"}` (model_id required).

Response: 200 `{"success":true,"data":{"text":"..."}}`

```bash
curl -X POST $BASE/api/v1/initialization/extract/fabri-text -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"model_id":"m-1","tags":["person"]}'
```

## Evaluation (/api/v1/evaluation)

Handler: `internal/handler/evaluation.go`. API key: `run_evaluations`/full.

### POST /api/v1/evaluation

Purpose: launch an evaluation task (drives LLM calls, which incurs costs). Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `dataset_id` | string | No | Dataset ID |
| `knowledge_base_id` | string | No | Target KB |
| `chat_id` | string | No | Conversation model ID |
| `rerank_id` | string | No | Rerank model ID |

Response: 200 `{"success":true,"data":{evaluation task}}`

```bash
curl -X POST $BASE/api/v1/evaluation -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"knowledge_base_id":"kb-1","chat_id":"m-1"}'
```

### GET /api/v1/evaluation

Purpose: query evaluation results. Permission: Viewer+. Query parameters: `task_id` (required).

Response: 200 `{"success":true,"data":{evaluation result}}`

```bash
curl "$BASE/api/v1/evaluation?task_id=task-1" -H "Authorization: Bearer $TOKEN"
```

## Implementation Reference

Route registration: `internal/router/router.go` calls `RegisterModelRoutes`, `RegisterInitializationRoutes`, `RegisterEvaluationRoutes`, and `RegisterWeKnoraCloudRoutes` (defined in `internal/router/routes_infra.go`). Handlers: `internal/handler/model.go`, `internal/handler/model_catalog.go`, `internal/handler/model_credentials.go`, `internal/handler/initialization.go`, `internal/handler/evaluation.go`, `internal/handler/weknoracloud.go`.
