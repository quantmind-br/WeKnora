# Initialization Configuration API

[Back to Index](./README.md)

| Method | Path                                                 | Description                             |
| ------ | ----------------------------------------------------- | ---------------------------------------- |
| GET    | `/initialization/config/:kb_id`                       | Get knowledge base initialization config |
| POST   | `/initialization/initialize/:kb_id`                   | Initialize knowledge base model config   |
| PUT    | `/initialization/config/:kb_id`                       | Update knowledge base model config       |
| GET    | `/initialization/ollama/status`                       | Check Ollama status                      |
| GET    | `/initialization/ollama/models`                       | Get local Ollama model list              |
| POST   | `/initialization/ollama/models/check`                 | Check whether an Ollama model is available |
| POST   | `/initialization/ollama/models/download`              | Download an Ollama model                 |
| GET    | `/initialization/ollama/download/progress/:task_id`   | Get download progress                    |
| GET    | `/initialization/ollama/download/tasks`               | Get all download tasks                   |
| POST   | `/initialization/remote/check`                        | Check remote model API                   |
| POST   | `/initialization/embedding/test`                      | Test embedding model                     |
| POST   | `/initialization/rerank/check`                        | Check rerank model                       |
| POST   | `/initialization/multimodal/test`                     | Test multimodal model                    |
| POST   | `/initialization/extract/text-relation`                | Extract text relations                   |

## GET `/initialization/config/:kb_id` - Get knowledge base initialization config

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/config/kb-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "chat_model_id": "model-00000001",
        "embedding_model_id": "model-00000002",
        "rerank_model_id": "model-00000003",
        "multimodal_id": "model-00000004"
    },
    "success": true
}
```

## POST `/initialization/initialize/:kb_id` - Initialize knowledge base model config

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/initialize/kb-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "chat_model_id": "model-00000001",
    "embedding_model_id": "model-00000002",
    "rerank_model_id": "model-00000003",
    "multimodal_id": "model-00000004"
}'
```

**Response**:

```json
{
    "success": true
}
```

## PUT `/initialization/config/:kb_id` - Update knowledge base model config

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/initialization/config/kb-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "chat_model_id": "model-00000010",
    "embedding_model_id": "model-00000002"
}'
```

**Response**:

```json
{
    "success": true
}
```

## GET `/initialization/ollama/status` - Check Ollama status

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/ollama/status' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "available": true
    },
    "success": true
}
```

## GET `/initialization/ollama/models` - Get local Ollama model list

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/ollama/models' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "name": "llama3:8b",
            "size": 4661211648,
            "modified_at": "2025-08-10T15:30:00+08:00"
        },
        {
            "name": "nomic-embed-text:latest",
            "size": 274302976,
            "modified_at": "2025-08-11T09:00:00+08:00"
        }
    ],
    "success": true
}
```

## POST `/initialization/ollama/models/check` - Check whether an Ollama model is available

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/ollama/models/check' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "models": ["llama3:8b", "nomic-embed-text:latest", "mistral:7b"]
}'
```

**Response**:

```json
{
    "data": {
        "llama3:8b": true,
        "nomic-embed-text:latest": true,
        "mistral:7b": false
    },
    "success": true
}
```

## POST `/initialization/ollama/models/download` - Download an Ollama model

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/ollama/models/download' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "model": "mistral:7b"
}'
```

**Response**:

```json
{
    "data": {
        "id": "task-00000001",
        "modelName": "mistral:7b",
        "status": "downloading",
        "progress": 0,
        "message": "Download started",
        "startTime": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

## GET `/initialization/ollama/download/progress/:task_id` - Get download progress

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/ollama/download/progress/task-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "id": "task-00000001",
        "modelName": "mistral:7b",
        "status": "downloading",
        "progress": 45.6,
        "message": "Downloading 2.1GB / 4.6GB",
        "startTime": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

## GET `/initialization/ollama/download/tasks` - Get all download tasks

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/ollama/download/tasks' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "id": "task-00000001",
            "modelName": "mistral:7b",
            "status": "completed",
            "progress": 100,
            "message": "Download complete",
            "startTime": "2025-08-12T10:00:00+08:00",
            "endTime": "2025-08-12T10:15:00+08:00"
        },
        {
            "id": "task-00000002",
            "modelName": "llama3:70b",
            "status": "downloading",
            "progress": 30.2,
            "message": "Downloading 12.5GB / 41.4GB",
            "startTime": "2025-08-12T10:20:00+08:00"
        }
    ],
    "success": true
}
```

## POST `/initialization/remote/check` - Check remote model API

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/remote/check' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "api_url": "https://api.openai.com/v1",
    "api_key": "sk-xxxxx",
    "model": "gpt-4o"
}'
```

**Response**:

```json
{
    "data": {
        "success": true,
        "message": "Model available"
    },
    "success": true
}
```

## POST `/initialization/embedding/test` - Test embedding model

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/embedding/test' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "api_url": "https://api.openai.com/v1",
    "api_key": "sk-xxxxx",
    "model": "text-embedding-3-small"
}'
```

**Response**:

```json
{
    "data": {
        "success": true,
        "message": "Embedding model test passed"
    },
    "success": true
}
```

## POST `/initialization/rerank/check` - Check rerank model

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/rerank/check' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "api_url": "https://api.cohere.ai/v1",
    "api_key": "sk-xxxxx",
    "model": "rerank-english-v3.0"
}'
```

**Response**:

```json
{
    "data": {
        "success": true,
        "message": "Rerank model available"
    },
    "success": true
}
```

## POST `/initialization/multimodal/test` - Test multimodal model

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/multimodal/test' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "api_url": "https://api.openai.com/v1",
    "api_key": "sk-xxxxx",
    "model": "gpt-4o"
}'
```

**Response**:

```json
{
    "data": {
        "success": true,
        "message": "Multimodal model test passed"
    },
    "success": true
}
```

## POST `/initialization/extract/text-relation` - Extract text relations

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/initialization/extract/text-relation' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "text": "WeKnora is a knowledge management platform that supports parsing and retrieval of multiple document formats.",
    "model_id": "model-00000001"
}'
```

**Response**:

```json
{
    "data": {
        "entities": [
            {"name": "WeKnora", "type": "Product"},
            {"name": "Knowledge Management Platform", "type": "Concept"}
        ],
        "relations": [
            {
                "source": "WeKnora",
                "target": "Knowledge Management Platform",
                "relation": "is_a"
            }
        ]
    },
    "success": true
}
```
