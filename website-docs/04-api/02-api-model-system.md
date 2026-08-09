# API Reference: Models and Initialization

Route registration: `RegisterModelRoutes`, `RegisterInitializationRoutes`, `RegisterEvaluationRoutes`, `RegisterWeKnoraCloudRoutes` in `internal/router/router.go`. Handlers: `internal/handler/model.go`, `internal/handler/model_credentials.go`, `internal/handler/initialization.go`, `internal/handler/evaluation.go`, `internal/handler/weknoracloud.go`.

For system information and system administration (`/system`, `/system/admin`) endpoints, see [System and Platform Management](./02-api-system.md).

## Models (/api/v1/models)

API key: `manage_models` or full-access.

### GET /api/v1/models/providers

Purpose: list of model providers. Permission: Viewer+. Query parameters: `model_type` (optional: `chat/embedding/rerank/vllm/asr`). Handler: `internal/handler/model.go`

Response: 200 `{"success":true,"data":[{value,label,description,defaultUrls,modelTypes}]}`

```bash
curl "$BASE/api/v1/models/providers?model_type=chat" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/models

Purpose: create a model. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required"`) | Model name |
| `display_name` | string | No | Display name |
| `type` | string | Yes (`binding:"required"`) | Model type |
| `source` | string | Yes (`binding:"required"`) | Source (local/remote…) |
| `description` | string | No | Description |
| `parameters` | object | Yes (`binding:"required"`) | Connection parameters (base_url, etc.; keys are managed through the credentials sub-resource) |

Response: 201 `{"success":true,"data":{ModelResponse}}` (`id,name,type,source,parameters,is_default,is_builtin,status,credentials,...`)

```bash
curl -X POST $BASE/api/v1/models -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"gpt-4o-mini","type":"chat","source":"remote","parameters":{"base_url":"https://api.openai.com/v1"}}'
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

Purpose: debug a saved model (triggers a real upstream call, which incurs costs). Permission: Admin+. form-data fields: `input` (≤64KB), `options` (JSON-encoded debug options), `documents` (JSON array, ≤100 items), `file` (optional).

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

Response: 200 `{"success":true,"data":{"hasFiles",llm,embedding,rerank,multimodal,documentSplitting,nodeExtract,questionGeneration}}`

```bash
curl $BASE/api/v1/initialization/config/kb-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/initialization/initialize/:kbId

Purpose: initialize the KB's model and parsing configuration (first-time setup wizard). Permission: KB creator OR Admin+, KB write.

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
| `dimension` | int | No | Embedding dimension |
| `customHeaders` / `extraConfig` | map | No | Extensions |
| `modelId` | string | No | Fetch credentials from an existing stored model |

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
