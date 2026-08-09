Segue a tradução completa do documento para inglês, com toda a estrutura markdown preservada.

---

# Model Management API

[Back to directory](./README.md)

The model management endpoints are used to maintain the LLM / Embedding / Rerank / VLLM / ASR model configurations available in the current space.

| Method | Path                | Description                |
| ------ | ------------------- | --------------------------- |
| GET    | `/models/providers` | Get the list of model providers |
| POST   | `/models`           | Create a model              |
| GET    | `/models`           | Get the list of models      |
| GET    | `/models/:id`       | Get model details           |
| PUT    | `/models/:id`       | Update a model               |
| DELETE | `/models/:id`       | Delete a model               |

## Provider Support

WeKnora supports multiple mainstream AI model providers. When creating a model, you can specify the provider type via the `parameters.provider` field for better compatibility.

### List of supported providers

| Provider ID    | Name                          | Supported model types           |
| -------------- | ----------------------------- | -------------------------------- |
| `generic`      | Custom (OpenAI-compatible interface) | Chat, Embedding, Rerank, VLLM |
| `openai`       | OpenAI                        | Chat, Embedding, Rerank, VLLM    |
| `aliyun`       | Alibaba Cloud DashScope        | Chat, Embedding, Rerank, VLLM    |
| `zhipu`        | Zhipu BigModel                 | Chat, Embedding, Rerank, VLLM    |
| `volcengine`   | Volcengine                     | Chat, Embedding, Rerank, VLLM    |
| `hunyuan`      | Tencent Hunyuan                | Chat, Embedding                  |
| `deepseek`     | DeepSeek                       | Chat                              |
| `minimax`      | MiniMax                        | Chat                              |
| `mimo`         | Xiaomi MiMo                    | Chat                              |
| `siliconflow`  | SiliconFlow                    | Chat, Embedding, Rerank, VLLM    |
| `jina`         | Jina                            | Embedding, Rerank                |
| `openrouter`   | OpenRouter                     | Chat, VLLM                        |
| `requesty`     | Requesty                       | Chat, Embedding, VLLM             |
| `gemini`       | Google Gemini                  | Chat                              |
| `modelscope`   | ModelScope                     | Chat, Embedding, VLLM             |
| `moonshot`     | Moonshot AI                    | Chat, VLLM                        |
| `qianfan`      | Baidu Cloud Qianfan            | Chat, Embedding, Rerank, VLLM     |
| `qiniu`        | Qiniu Cloud                    | Chat                              |
| `longcat`      | LongCat AI                     | Chat                              |
| `gpustack`     | GPUStack                       | Chat, Embedding, Rerank, VLLM     |

> The actually available providers are determined by the response of `GET /models/providers`.

## GET `/models/providers` - Get the list of model providers

Get the list of supported providers and their configuration information based on the model type (system-level metadata, unrelated to any specific space).

**Query parameters**:

| Field      | Type   | Required | Description                                                |
| ---------- | ------ | -------- | ------------------------------------------------------------ |
| model_type | string | No       | Model type, possible values: `chat` / `embedding` / `rerank` / `vllm` / `asr`; if omitted, all types are returned |

**Request**:

```curl
# Get all providers
curl --location 'http://localhost:8080/api/v1/models/providers' \
--header 'X-API-Key: your_api_key'

# Get providers that support the Embedding type
curl --location 'http://localhost:8080/api/v1/models/providers?model_type=embedding' \
--header 'X-API-Key: your_api_key'
```

**Response**:

```json
{
    "success": true,
    "data": [
        {
            "value": "aliyun",
            "label": "Alibaba Cloud DashScope",
            "description": "qwen-plus, tongyi-embedding-vision-plus, qwen3-rerank, etc.",
            "defaultUrls": {
                "chat": "https://dashscope.aliyuncs.com/compatible-mode/v1",
                "embedding": "https://dashscope.aliyuncs.com/compatible-mode/v1",
                "rerank": "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank"
            },
            "modelTypes": ["chat", "embedding", "rerank", "vllm"]
        },
        {
            "value": "zhipu",
            "label": "Zhipu BigModel",
            "description": "glm-4.7, embedding-3, rerank, etc.",
            "defaultUrls": {
                "chat": "https://open.bigmodel.cn/api/paas/v4",
                "embedding": "https://open.bigmodel.cn/api/paas/v4/embeddings",
                "rerank": "https://open.bigmodel.cn/api/paas/v4/rerank"
            },
            "modelTypes": ["chat", "embedding", "rerank", "vllm"]
        }
    ]
}
```

## POST `/models` - Create a model

Creates a new model configuration for the current space.

**Parameters (request body)**:

| Field       | Type   | Required | Description                                                            |
| ----------- | ------ | -------- | ------------------------------------------------------------------------ |
| name        | string | Yes      | Model name (for remote models, this is the model id from the corresponding provider; for local models, this is the Ollama tag) |
| type        | string | Yes      | Model type, possible values: `KnowledgeQA` / `Embedding` / `Rerank` / `VLLM` / `ASR` |
| source      | string | Yes      | Model source, possible values: `local` / `remote`                        |
| description | string | No       | Model description                                                        |
| parameters  | object | Yes      | Model parameters, see [Parameters](#parameters-model-parameters) below   |

> When `parameters.base_url` is not empty, the backend performs SSRF validation; if validation fails, a 400 response is returned.

### Create a chat model (KnowledgeQA)

**Local Ollama model**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "qwen3:8b",
    "type": "KnowledgeQA",
    "source": "local",
    "description": "LLM Model for Knowledge QA",
    "parameters": {
        "base_url": "",
        "api_key": ""
    }
}'
```

**Remote API model (specific provider)**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "qwen-plus",
    "type": "KnowledgeQA",
    "source": "remote",
    "description": "Alibaba Cloud Qwen LLM",
    "parameters": {
        "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
        "api_key": "sk-your-dashscope-api-key",
        "provider": "aliyun"
    }
}'
```

### Create an embedding model (Embedding)

**Local Ollama model**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "nomic-embed-text:latest",
    "type": "Embedding",
    "source": "local",
    "description": "Embedding Model",
    "parameters": {
        "base_url": "",
        "api_key": "",
        "embedding_parameters": {
            "dimension": 768,
            "truncate_prompt_tokens": 0
        }
    }
}'
```

**Remote API model (Alibaba Cloud DashScope)**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "text-embedding-v3",
    "type": "Embedding",
    "source": "remote",
    "description": "Alibaba Cloud Tongyi Qianwen Embedding model",
    "parameters": {
        "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
        "api_key": "sk-your-dashscope-api-key",
        "provider": "aliyun",
        "embedding_parameters": {
            "dimension": 1024,
            "truncate_prompt_tokens": 0
        }
    }
}'
```

**Remote API model (Jina AI)**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "jina-embeddings-v3",
    "type": "Embedding",
    "source": "remote",
    "description": "Jina AI Embedding model",
    "parameters": {
        "base_url": "https://api.jina.ai/v1",
        "api_key": "jina_your_api_key",
        "provider": "jina",
        "embedding_parameters": {
            "dimension": 1024,
            "truncate_prompt_tokens": 0
        }
    }
}'
```

### Create a rerank model (Rerank)

**Remote API model (Alibaba Cloud DashScope)**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "gte-rerank",
    "type": "Rerank",
    "source": "remote",
    "description": "Alibaba Cloud GTE Rerank model",
    "parameters": {
        "base_url": "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank",
        "api_key": "sk-your-dashscope-api-key",
        "provider": "aliyun"
    }
}'
```

**Remote API model (Jina AI)**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "jina-reranker-v2-base-multilingual",
    "type": "Rerank",
    "source": "remote",
    "description": "Jina AI Rerank model",
    "parameters": {
        "base_url": "https://api.jina.ai/v1",
        "api_key": "jina_your_api_key",
        "provider": "jina"
    }
}'
```

**Remote API model (Volcengine VikingDB)**:

Volcengine Rerank uses AK/SK signing rather than an Ark API Key. `api_key` stores the Access Key ID, and
`app_secret` stores the Secret Access Key; both are encrypted and stored as model credentials.

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "doubao-seed-rerank",
    "type": "Rerank",
    "source": "remote",
    "description": "Volcengine hosted Rerank model",
    "parameters": {
        "base_url": "https://api-knowledgebase.mlp.cn-beijing.volces.com",
        "api_key": "your-volcengine-access-key-id",
        "app_secret": "your-volcengine-secret-access-key",
        "provider": "volcengine"
    }
}'
```

### Create a vision model (VLLM)

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "qwen-vl-plus",
    "type": "VLLM",
    "source": "remote",
    "description": "Alibaba Cloud Tongyi Qianwen vision model",
    "parameters": {
        "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
        "api_key": "sk-your-dashscope-api-key",
        "provider": "aliyun"
    }
}'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "09c5a1d6-ee8b-4657-9a17-d3dcbd5c70cb",
        "tenant_id": 1,
        "name": "text-embedding-v3",
        "type": "Embedding",
        "source": "remote",
        "description": "Alibaba Cloud Tongyi Qianwen Embedding model",
        "parameters": {
            "base_url": "https://dashscope.aliyuncs.com/compatible-mode/v1",
            "api_key": "sk-***",
            "provider": "aliyun",
            "embedding_parameters": {
                "dimension": 1024,
                "truncate_prompt_tokens": 0
            }
        },
        "is_default": false,
        "status": "active",
        "created_at": "2025-08-12T10:39:01.454591766+08:00",
        "updated_at": "2025-08-12T10:39:01.454591766+08:00",
        "deleted_at": null
    }
}
```

## GET `/models` - Get the list of models

Returns all models in the current space. For built-in models (`is_builtin = true`), the `base_url` and `api_key` fields are cleared to hide sensitive information.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/models' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key'
```

**Response**: `data` is an array; each element has the same field structure as the `POST /models` response. For built-in models, the `base_url` and `api_key` fields are empty strings.

## GET `/models/:id` - Get model details

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| id    | string | Yes      | Model ID     |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/models/dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key'
```

**Response**: Same field structure as the `POST /models` response. A 404 indicates the model does not exist.

## PUT `/models/:id` - Update a model

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| id    | string | Yes      | Model ID     |

**Parameters (request body)**:

| Field       | Type   | Required | Description                                                            |
| ----------- | ------ | -------- | ------------------------------------------------------------------------ |
| name        | string | No       | Model name (leaving it as an empty string preserves the original value) |
| description | string | No       | Model description (always overwritten; passing an empty string clears it) |
| type        | string | No       | Model type, same values as the create endpoint                          |
| source      | string | No       | Model source, same values as the create endpoint                        |
| parameters  | object | No       | Model parameters; `parameter_size` is managed by the backend and does not need to be provided in the request; if `extra_config` is empty, the previous value is retained |

> `parameters.base_url` also undergoes SSRF validation; a 400 response is returned on failure.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/models/8fdc464d-8eaa-44d4-a85b-094b28af5330' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key' \
--data '{
    "name": "gte-rerank-v2",
    "type": "Rerank",
    "source": "remote",
    "description": "Alibaba Cloud GTE Rerank model V2",
    "parameters": {
        "base_url": "https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank",
        "api_key": "sk-your-new-api-key",
        "provider": "aliyun"
    }
}'
```

**Response**: Same field structure as the `POST /models` response, returning the full, updated model object.

## DELETE `/models/:id` - Delete a model

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| id    | string | Yes      | Model ID     |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/models/8fdc464d-8eaa-44d4-a85b-094b28af5330' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: your_api_key'
```

**Response**:

```json
{
    "success": true,
    "message": "Model deleted"
}
```

A 404 indicates the model does not exist.

## Parameter description

### ModelType (model type)

| Value       | Frontend alias | Description             | Use case                              |
| ----------- | --------------- | ------------------------ | -------------------------------------- |
| KnowledgeQA | `chat`          | Chat model                | Knowledge base Q&A, conversation generation |
| Embedding   | `embedding`     | Embedding model           | Text vectorization, knowledge base retrieval |
| Rerank      | `rerank`        | Rerank model               | Reranking retrieval results, relevance optimization |
| VLLM        | `vllm`          | Vision-language model      | Multimodal analysis, image-text understanding |
| ASR         | `asr`           | Speech recognition model   | Audio transcription                    |

> The `type` field in the create/update endpoint request body uses the backend enum values in the first column (e.g. `KnowledgeQA`); the `model_type` query parameter of `GET /models/providers?model_type=` uses the frontend alias in the second column (e.g. `chat`).

### ModelSource (model source)

| Value  | Description   | Configuration requirements            |
| ------ | -------------- | --------------------------------------- |
| local  | Local model    | Requires Ollama to be installed with the model pulled |
| remote | Remote API     | Requires `base_url` and `api_key` to be provided |

### Parameters (model parameters)

| Field                 | Type                   | Required | Description                                                       |
| --------------------- | ----------------------- | -------- | -------------------------------------------------------------------- |
| base_url              | string                  | No       | API service address; required for remote models, subject to SSRF validation |
| api_key               | string                  | No       | API key; required for remote models, encrypted with AES-256 when stored |
| provider              | string                  | No       | Provider identifier (see the supported list above), used to select a specific API adapter |
| interface_type        | string                  | No       | Interface style identifier (leave empty for OpenAI-compatible)      |
| embedding_parameters  | object                  | No       | Parameters specific to Embedding models, see below                  |
| parameter_size        | string                  | No       | Model parameter scale (e.g. `7B`/`13B`/`70B`), usually written by the backend |
| extra_config          | object<string,string>   | No       | Provider-specific extra configuration                                |
| custom_headers        | object<string,string>   | No       | Custom HTTP headers appended when calling the upstream API; reserved headers are ignored |
| supports_vision       | bool                    | No       | Whether the model supports image/multimodal input                    |

### EmbeddingParameters (embedding parameters)

| Field                   | Type | Required | Description                        |
| ------------------------ | ---- | -------- | ------------------------------------ |
| dimension                | int  | No       | Vector dimension (e.g. 768, 1024)    |
| truncate_prompt_tokens   | int  | No       | Number of tokens to truncate (0 means no truncation) |
