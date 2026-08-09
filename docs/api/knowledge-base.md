Traduzo doc agora. Saída abaixo, inglês natural, markdown intacto.

---

# Knowledge Base Management API

[Back to index](./README.md)

**Field Notes (Knowledge Base Object)**

- Knowledge base `type` is `document` or `faq`, defaulting to `document`.
- JSON object storage-related field: **`storage_config`** is the serialized field name (corresponds to the database column `cos_config`, kept for backward compatibility with old data). If legacy clients still send or receive `cos_config`, the server will parse it compatibly; new integrations should use **`storage_config`**.
- **`storage_provider_config`** is the new-style storage provider selection (e.g. `{"provider": "local"}`), used together with space-level storage engine credentials; it can be `null` when not configured.
- Nested config objects: `chunking_config`, `image_processing_config`, `vlm_config`, `asr_config`, `extract_config`, `faq_config`, `question_generation_config`, `auto_tag_config`. Of these, `extract_config`, `faq_config`, `question_generation_config`, and `auto_tag_config` may be `null`.
- **`vector_store_id`** is the vector store ID bound to the knowledge base (see [vector-store.md](./vector-store.md)). When unspecified (or `null`/`""`), the space-level default environment-variable-based store is used; once created, it cannot be changed. The detail endpoint response includes four read-only metadata fields for frontend display: `vector_store_name` / `vector_store_source` / `vector_store_engine_type` / `vector_store_status`.

| Method | Path                                       | Description                                       |
| ------ | ------------------------------------------- | -------------------------------------------------- |
| POST   | `/knowledge-bases`                          | Create a knowledge base                            |
| GET    | `/knowledge-bases`                          | List knowledge bases                                |
| GET    | `/knowledge-bases/:id`                      | Get knowledge base details                          |
| PUT    | `/knowledge-bases/:id`                      | Update a knowledge base                             |
| DELETE | `/knowledge-bases/:id`                      | Delete a knowledge base                             |
| PUT    | `/knowledge-bases/:id/pin`                  | Pin/unpin a knowledge base                          |
| POST   | `/knowledge-bases/:id/hybrid-search`        | Hybrid search (vector + keyword, recommended)       |
| GET    | `/knowledge-bases/:id/hybrid-search`        | Hybrid search (legacy client compatibility, requires JSON body) |
| POST   | `/knowledge-bases/copy`                     | Copy a knowledge base (async task)                  |
| GET    | `/knowledge-bases/copy/progress/:task_id`   | Get copy progress                                    |
| POST   | `/knowledge-bases/:id/duplicate`            | Create a knowledge base duplicate (settings only)    |
| GET    | `/knowledge-bases/:id/move-targets`         | Get the list of eligible migration target knowledge bases |

## POST `/knowledge-bases` - Create a Knowledge Base

**Parameters (Request Body)**:

| Field                       | Type    | Required | Description                                                             |
| ---------------------------- | ------- | -------- | ------------------------------------------------------------------------ |
| name                          | string  | Yes      | Knowledge base name                                                      |
| description                   | string  | No       | Knowledge base description                                               |
| type                          | string  | No       | Knowledge base type: `document` (default) or `faq`                       |
| is_temporary                  | boolean | No       | Whether this is a temporary knowledge base (default `false`; temporary bases are usually not shown in the UI list) |
| chunking_config               | object  | No       | Chunking configuration (see example below)                               |
| image_processing_config       | object  | No       | Image processing configuration                                           |
| embedding_model_id            | string  | No       | Embedding model ID                                                       |
| summary_model_id              | string  | No       | Summary model ID                                                         |
| vlm_config                    | object  | No       | VLM (vision model) configuration                                         |
| asr_config                    | object  | No       | ASR (speech recognition) configuration                                   |
| storage_provider_config       | object  | No       | Storage provider selection, e.g. `{"provider": "local"}`                 |
| storage_config                | object  | No       | Legacy COS storage credentials (compatibility field; leave empty for new integrations) |
| extract_config                | object  | No       | Knowledge graph extraction configuration; when `enabled=true`, `text`/`tags`/`nodes`/`relations` must be provided |
| faq_config                    | object  | No       | FAQ configuration (only needed for FAQ-type knowledge bases)             |
| question_generation_config    | object  | No       | Question generation configuration                                        |
| auto_tag_config               | object  | No       | Document auto-tagging configuration, disabled by default; only applies to `document`-type knowledge bases |
| vector_store_id               | string  | No       | Bound vector store ID. Omitting it or passing an empty string is equivalent to `null` (uses the environment-variable default store). When specified, it must be a vector store UUID owned by the caller's space; it cannot be changed after creation. An invalid UUID / cross-space ID / an ID not registered with the engine returns `400` |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-xxxxx' \
--data '{
    "name": "weknora",
    "description": "weknora description",
    "type": "document",
    "is_temporary": false,
    "chunking_config": {
        "chunk_size": 1000,
        "chunk_overlap": 200,
        "separators": [
            "."
        ],
        "enable_multimodal": true,
        "parser_engine_rules": [
            {
                "file_types": [".pdf", ".docx"],
                "engine": "builtin"
            }
        ],
        "enable_parent_child": false,
        "parent_chunk_size": 4096,
        "child_chunk_size": 384
    },
    "image_processing_config": {
        "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
    },
    "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
    "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
    "vlm_config": {
        "enabled": true,
        "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
    },
    "asr_config": {
        "enabled": false,
        "model_id": "",
        "language": ""
    },
    "storage_provider_config": {
        "provider": "local"
    },
    "storage_config": {
        "secret_id": "",
        "secret_key": "",
        "region": "",
        "bucket_name": "",
        "app_id": "",
        "path_prefix": ""
    },
    "extract_config": null,
    "faq_config": null,
    "question_generation_config": {
        "enabled": false,
        "question_count": 3
    },
    "auto_tag_config": {
        "enabled": true,
        "model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "max_tags": 3,
        "skip_if_tagged": true
    },
    "vector_store_id": "550e8400-e29b-41d4-a716-446655440000"
}'
```

**Response**:

```json
{
    "data": {
        "id": "b5829e4a-3845-4624-a7fb-ea3b35e843b0",
        "name": "weknora",
        "description": "weknora description",
        "type": "document",
        "is_temporary": false,
        "tenant_id": 1,
        "chunking_config": {
            "chunk_size": 1000,
            "chunk_overlap": 200,
            "separators": [
                "."
            ],
            "enable_multimodal": true,
            "parser_engine_rules": [
                {
                    "file_types": [".pdf", ".docx"],
                    "engine": "builtin"
                }
            ],
            "enable_parent_child": false,
            "parent_chunk_size": 4096,
            "child_chunk_size": 384
        },
        "image_processing_config": {
            "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
        },
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
        "vlm_config": {
            "enabled": true,
            "model_id": "f2083ad7-63e3-486d-a610-e6c56e58d72e"
        },
        "asr_config": {
            "enabled": false,
            "model_id": "",
            "language": ""
        },
        "storage_provider_config": {
            "provider": "local"
        },
        "storage_config": {
            "secret_id": "",
            "secret_key": "",
            "region": "",
            "bucket_name": "",
            "app_id": "",
            "path_prefix": ""
        },
        "extract_config": null,
        "faq_config": null,
        "question_generation_config": {
            "enabled": false,
            "question_count": 3
        },
        "auto_tag_config": {
            "enabled": true,
            "model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
            "max_tags": 3,
            "skip_if_tagged": true
        },
        "is_pinned": false,
        "pinned_at": null,
        "knowledge_count": 0,
        "chunk_count": 0,
        "processing_count": 0,
        "vector_store_id": "550e8400-e29b-41d4-a716-446655440000",
        "vector_store_name": "elasticsearch-hot",
        "vector_store_source": "user",
        "vector_store_engine_type": "elasticsearch",
        "vector_store_status": "available",
        "created_at": "2025-08-12T11:30:09.206238645+08:00",
        "updated_at": "2025-08-12T11:30:09.206238854+08:00",
        "deleted_at": null
    },
    "success": true
}
```

### Auto-Tagging Configuration

`auto_tag_config` asynchronously invokes a chat model after document parsing completes, selecting matching tags from the knowledge base's existing tags and incrementally associating them with the document. This process does not create new tags, nor does it delete or overwrite manually added tags.

| Field       | Type    | Default | Description |
| ---------- | ------- | ------ | ---- |
| `enabled`  | boolean | `false` | Whether auto-tagging is enabled |
| `model_id` | string  | `""` | The chat model ID to use; when empty, the knowledge base's `summary_model_id` is used |
| `max_tags` | integer | `3` | Maximum number of tags auto-associated per document, ranging from `1` to `10` |
| `skip_if_tagged` | boolean | `true` | Whether to skip auto-tagging when a document already has tags. When enabled, the model is not invoked, avoiding dilution of manual classification; when set to `false`, tags are appended to the existing ones |

Auto-tagging only applies to documents newly parsed or re-parsed after this configuration is enabled. A failed model call does not block document parsing from completing; the async task retries according to the task queue policy.

Candidate tags are sorted by knowledge base and the top 500 are used for classification; if the tag count exceeds this, a warning is logged and this prefix is used — the task is not skipped. The model returns results by candidate index; the server validates the index range and maps it back to tag IDs, discarding any out-of-range or duplicate indices.

**`vector_store_*` Response Field Descriptions**:

| Field                       | Type   | Description                                                                                                       |
| -------------------------- | ------ | ------------------------------------------------------------------------------------------------------------------ |
| `vector_store_id`          | string | The bound vector store ID (`null` and omitted from the response if unspecified at creation)                       |
| `vector_store_name`        | string | Display name of the bound store. Returns `"System default"` when unbound; hidden in cross-space shared KB views    |
| `vector_store_source`      | string | `"user"` (store created in DB) / `"env"` (environment-variable virtual store) / `"shared"` (cross-space shared KB) / `"unavailable"` (bound store can no longer be resolved) |
| `vector_store_engine_type` | string | Engine type (`elasticsearch` / `qdrant` / `milvus`, etc.). Empty for `shared` / `unavailable`                     |
| `vector_store_status`      | string | `"available"` / `"unavailable"`. `unavailable` means the bound store has been deleted or is not in the in-memory registry; the UI can prompt the user to rebind based on this |

**Error Codes (New in Phase 2)**:

| HTTP | code | Description                                                              |
| ---- | ---- | ------------------------------------------------------------------------- |
| 400  | 2200 | Invalid `vector_store_id`: malformed, nonexistent, or belongs to another space (returned uniformly to avoid enumeration leaks) |
| 400  | 2201 | The specified vector store is currently unavailable: it exists in the database but is not registered with the engine registry; check `connection_config` |

## GET `/knowledge-bases` - List Knowledge Bases

Returns all knowledge bases owned by the current space. When `agent_id` is passed, after validating the caller's access to that shared agent, returns the scope of knowledge bases visible to that agent's configuration (used for `@` mentions).

**Query Parameters**:

| Field     | Type   | Required | Description                                                       |
| -------- | ------ | ---- | ---------------------------------------------------------- |
| agent_id | string | No   | Shared agent ID; when passed, filters visible knowledge bases according to the agent's configuration (`all` / `selected` / `none`) |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**: `data` is an array; each element has the same field structure as the `POST /knowledge-bases` response, additionally carrying the aggregate and status fields `knowledge_count` / `chunk_count` / `processing_count` / `share_count` / `is_pinned` / `pinned_at`.

> **Note (Phase 2)**: The list endpoint does not include the four resolved metadata fields `vector_store_name` / `vector_store_source` / `vector_store_engine_type` / `vector_store_status` (to avoid N+1 queries); only `vector_store_id` comes directly from the database. To display the store name, call the detail endpoint or `/vector-stores/:id` separately.

## GET `/knowledge-bases/:id` - Get Knowledge Base Details

Retrieves knowledge base details by ID. When accessed via a shared agent, `agent_id` can be passed for permission validation; in that case, the returned object includes a `my_permission` field indicating the current user's role on this knowledge base (e.g. `viewer`).

**Path Parameters**:

| Field | Type   | Description       |
| ---- | ------ | --------- |
| id   | string | Knowledge base ID |

**Query Parameters**:

| Field     | Type   | Required | Description                                       |
| -------- | ------ | ---- | ------------------------------------------ |
| agent_id | string | No   | Shared agent ID (used to validate whether this agent has access) |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**: Same field structure as the `POST /knowledge-bases` response (including the Phase 2 `vector_store_*` metadata fields), plus the `is_pinned` / `pinned_at` / `knowledge_count` / `chunk_count` / `processing_count` status fields. When accessed via a shared agent, `my_permission` is also included; additionally `vector_store_name` / `vector_store_engine_type` are hidden (`vector_store_source` returns `"shared"`), to avoid leaking store display names across spaces.

## PUT `/knowledge-bases/:id` - Update a Knowledge Base

Can only be called by the knowledge base owner (admin) or a user with `editor` permission. Note: **`vector_store_id` cannot be modified after creation** — the update endpoint does not accept this field.

**Path Parameters**:

| Field | Type   | Description       |
| ---- | ------ | --------- |
| id   | string | Knowledge base ID |

**Parameters (Request Body)**:

| Field        | Type   | Required | Description                                                        |
| ----------- | ------ | ---- | ------------------------------------------------------------- |
| name        | string | Yes  | Knowledge base name                                                    |
| description | string | No   | Knowledge base description                                                    |
| config      | object | No   | Update configuration; includes `chunking_config` / `image_processing_config` / `faq_config` / `wiki_config` / `indexing_strategy` |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/b5829e4a-3845-4624-a7fb-ea3b35e843b0' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-xxxxx' \
--data '{
    "name": "weknora new",
    "description": "weknora description new",
    "config": {
        "chunking_config": {
            "chunk_size": 1000,
            "chunk_overlap": 200,
            "separators": [
                "\n\n",
                "\n",
                "。",
                "！",
                "？",
                ";",
                "；"
            ],
            "enable_multimodal": true,
            "parser_engine_rules": [
                {
                    "file_types": [".md", ".txt"],
                    "engine": "builtin"
                }
            ],
            "enable_parent_child": true,
            "parent_chunk_size": 4096,
            "child_chunk_size": 384
        },
        "image_processing_config": {
            "model_id": ""
        }
    }
}'
```

**Response**: Same field structure as the `POST /knowledge-bases` response (returns the complete updated knowledge base object, including the Phase 2 `vector_store_*` metadata fields. `vector_store_id` remains the same as at creation and cannot be changed via this endpoint).

## DELETE `/knowledge-bases/:id` - Delete a Knowledge Base

Can only be called by the knowledge base owner (an admin matching the owning space); deletion cascades to clean up all knowledge and chunks under the knowledge base.

**Path Parameters**:

| Field | Type   | Description       |
| ---- | ------ | --------- |
| id   | string | Knowledge base ID |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/b5829e4a-3845-4624-a7fb-ea3b35e843b0' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "message": "Knowledge base deleted successfully",
    "success": true
}
```

## PUT `/knowledge-bases/:id/pin` - Pin/Unpin a Knowledge Base

Toggles the pin status of a knowledge base. No request body is needed; each call automatically flips the current `is_pinned`. When pinned, the `pinned_at` timestamp is written accordingly.

**Path Parameters**:

| Field | Type   | Description       |
| ---- | ------ | --------- |
| id   | string | Knowledge base ID |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/pin' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**: Same field structure as the `POST /knowledge-bases` response (including the Phase 2 `vector_store_*` metadata fields); after this endpoint runs, `is_pinned` flips and `pinned_at` is updated accordingly.

## POST `/knowledge-bases/:id/hybrid-search` - Hybrid Search

Performs vector recall + keyword recall hybrid search within a specified knowledge base. Request parameters are passed via the JSON request body (`SearchParams`).

> **Compatibility note**: The `GET` method is also available (requires a JSON request body) for legacy client compatibility; new integrations should use `POST`.

**Path Parameters**:

| Field | Type   | Description       |
| ---- | ------ | --------- |
| id   | string | Knowledge base ID |

**Parameters (Request Body)**:

| Field                     | Type     | Required | Description                                                       |
| ------------------------ | -------- | ---- | ---------------------------------------------------------------- |
| query_text               | string   | Yes  | Query text                                                         |
| vector_threshold         | number   | No   | Vector similarity threshold (0-1)                                            |
| keyword_threshold        | number   | No   | Keyword match threshold                                                   |
| match_count              | integer  | No   | Maximum number of results to return                                                 |
| disable_keywords_match   | boolean  | No   | Disable keyword recall                                                   |
| disable_vector_match     | boolean  | No   | Disable vector recall                                                     |
| knowledge_ids            | string[] | No   | Restrict recall to the specified knowledge IDs                                     |
| tag_ids                  | string[] | No   | Tag filtering (commonly used for priority filtering in FAQ type)                             |
| only_recommended         | boolean  | No   | Only return content flagged as recommended                                           |
| knowledge_base_ids       | string[] | No   | Cross-knowledge-base recall (requires sharing the same embedding model); takes precedence over the path `:id` |
| skip_context_enrichment  | boolean  | No   | Skip context enrichment from parent/child or adjacent chunks (used in the chat flow)               |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/hybrid-search' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query_text": "如何使用知识库",
    "vector_threshold": 0.5,
    "match_count": 10
}'
```

**Response**:

```json
{
    "data": [
        {
            "id": "chunk-00000001",
            "content": "知识库是用于存储和检索知识的系统...",
            "knowledge_id": "knowledge-00000001",
            "chunk_index": 0,
            "knowledge_title": "知识库使用指南",
            "start_at": 0,
            "end_at": 500,
            "seq": 1,
            "score": 0.95,
            "chunk_type": "text",
            "image_info": "",
            "metadata": {},
            "knowledge_filename": "guide.pdf",
            "knowledge_source": "file"
        }
    ],
    "success": true
}
```

## POST `/knowledge-bases/copy` - Copy a Knowledge Base

Asynchronously copies an entire knowledge base (configuration + all knowledge content). The request is enqueued to an Asynq background task (queue `default`, up to 3 retries) and immediately returns a `task_id` for progress polling.

**Constraint**: The source knowledge base `source_id` must belong to the caller's space; if `target_id` is specified, the target knowledge base must also belong to the caller's space, otherwise `403 Forbidden` is returned.

**Phase 2 Synchronous Pre-checks (when `target_id` is non-empty)**:

| Check           | Response on failure                                                                    |
| -------------- | ------------------------------------------------------------------------------------- |
| Embedding model consistency | `400` `source and target knowledge bases use different embedding models; clone into a target with the same embedding model` |
| Vector store consistency | `400` `source and target knowledge bases are bound to different vector stores; cross-store cloning is not yet supported`     |

If the pre-check fails, the task is not enqueued and no `task_id` is generated; the caller receives a `400` immediately. These two checks are also performed again in the async worker (defense in depth), but rejecting immediately at handshake time avoids the user having to poll `progress` just to see the error. When `target_id` is empty (creating a new target base), the target base automatically copies the source base's `vector_store_id` and `embedding_model_id`, so the pre-check is not triggered.

**Parameters (Request Body)**:

| Field       | Type   | Required | Description                                                        |
| ---------- | ------ | ---- | ------------------------------------------------------------- |
| source_id  | string | Yes  | Source knowledge base ID (must belong to the current space)                               |
| target_id  | string | No   | Target knowledge base ID (if reusing an existing knowledge base; must also belong to the current space)     |
| task_id    | string | No   | Custom task ID; if not provided, the server generates one (based on space, source ID, and timestamp)  |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/copy' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "source_id": "kb-00000001"
}'
```

**Response**:

```json
{
    "data": {
        "task_id": "kb_clone_1_kb-00000001_1736582400",
        "source_id": "kb-00000001",
        "target_id": "",
        "message": "Knowledge base copy task started"
    },
    "success": true
}
```

## GET `/knowledge-bases/copy/progress/:task_id` - Get Copy Progress

Queries the current status and progress of a copy task (data written to Redis by the worker).

**Path Parameters**:

| Field    | Type   | Description                                              |
| ------- | ------ | -------------------------------------------------- |
| task_id | string | The task ID returned by `POST /knowledge-bases/copy`     |

**Response Fields (`data`)**:

| Field       | Type    | Description                                                         |
| ---------- | ------- | ------------------------------------------------------------ |
| task_id    | string  | Task ID                                                       |
| source_id  | string  | Source knowledge base ID                                                   |
| target_id  | string  | Target knowledge base ID (populated once the task starts)                             |
| status     | string  | `pending` / `processing` / `completed` / `failed`             |
| progress   | integer | Progress percentage 0–100                                              |
| total      | integer | Total number of knowledge items planned to be copied                                            |
| processed  | integer | Number of knowledge items processed so far                                                 |
| message    | string  | Description of the current status                                                  |
| error      | string  | Error message on failure                                             |
| created_at | integer | Task creation time (Unix seconds)                                       |
| updated_at | integer | Last updated time (Unix seconds)                                       |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/copy/progress/kb_clone_1_kb-00000001_1736582400' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "task_id": "kb_clone_1_kb-00000001_1736582400",
        "source_id": "kb-00000001",
        "target_id": "kb-00000002",
        "status": "completed",
        "progress": 100,
        "total": 10,
        "processed": 10,
        "message": "Task completed successfully",
        "error": "",
        "created_at": 1736582400,
        "updated_at": 1736582460
    },
    "success": true
}
```

## POST `/knowledge-bases/:id/duplicate` - Create a Knowledge Base Duplicate

Synchronously creates a new knowledge base duplicate containing **settings only**. Copies setting fields such as chunking, models, indexing strategy, and Wiki/FAQ configuration, but **does not** copy knowledge entries, chunk content, FAQ entries, wiki pages, vector/keyword indices, data source bindings, sharing relationships, or pin status.

Differences from `POST /knowledge-bases/copy`:

| Capability | `/duplicate` | `/copy` |
| ---- | ------------ | ------- |
| Execution mode | Synchronous, returns the new KB immediately | Async task, requires progress polling |
| Content copied | Settings only | Settings + all knowledge content |
| New KB ID | Auto-generated UUID by the server | Can specify an existing target base or create a new one |

**Permissions**: Requires `Contributor+`, and at least `Viewer` read permission on the source knowledge base (route-layer `KBAccessRead`). The source knowledge base must belong to the caller's space, otherwise `403 Forbidden` is returned.

**Naming rule**: The new KB name appends a localized suffix (based on `Accept-Language` or `WEKNORA_LANGUAGE`) to the source name, e.g. Chinese `原名 副本`, English `Original Name Copy`; if a name collision occurs, it increments to `原名 副本 2`, `Original Name Copy 2`, etc.

**Path Parameters**:

| Field | Type   | Description        |
| ---- | ------ | ----------- |
| id   | string | Source knowledge base ID |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/duplicate' \
--header 'Authorization: Bearer <token>' \
--header 'Accept-Language: zh-CN' \
--request POST
```

**Response** (HTTP 201):

```json
{
    "success": true,
    "data": {
        "source_id": "kb-00000001",
        "target_id": "kb-00000002",
        "message": "Knowledge base duplicate created",
        "knowledge_base": {
            "id": "kb-00000002",
            "name": "产品文档 副本",
            "type": "document",
            "description": "…",
            "embedding_model_id": "embed-1",
            "chunking_config": {},
            "knowledge_count": 0,
            "chunk_count": 0
        }
    }
}
```

**Response Fields (`data`)**:

| Field            | Type   | Description                         |
| --------------- | ------ | ---------------------------- |
| source_id       | string | Source knowledge base ID                  |
| target_id       | string | Newly created knowledge base ID            |
| message         | string | Description of the operation result                 |
| knowledge_base  | object | The complete knowledge base object of the new duplicate       |

**Common Errors**:

| Scenario                 | HTTP | Description |
| -------------------- | ---- | ---- |
| Source knowledge base does not exist       | 404  | `Source knowledge base not found` |
| Source base belongs to another space     | 403  | `No permission to duplicate this knowledge base` |
| Invalid vector store binding     | 400  | The vector store bound to the source base is unavailable |

## GET `/knowledge-bases/:id/move-targets` - Get the List of Eligible Migration Target Knowledge Bases

Returns the list of target knowledge bases that the current knowledge base's content **can be migrated to**. Filtering rules:

- Same `type` as the source knowledge base
- Same `embedding_model_id` as the source knowledge base
- Non-temporary knowledge bases (`is_temporary = false`)
- Excludes the source knowledge base itself
- Only knowledge bases in the same space

**Path Parameters**:

| Field | Type   | Description          |
| ---- | ------ | ------------- |
| id   | string | Source knowledge base ID   |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/move-targets' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "id": "kb-00000002",
            "name": "技术文档知识库",
            "description": "技术文档相关知识",
            "type": "document",
            "is_temporary": false,
            "tenant_id": 1,
            "chunking_config": {
                "chunk_size": 1000,
                "chunk_overlap": 200,
                "separators": ["\n\n", "\n"],
                "enable_multimodal": true,
                "parser_engine_rules": [],
                "enable_parent_child": false,
                "parent_chunk_size": 4096,
                "child_chunk_size": 384
            },
            "image_processing_config": {
                "model_id": ""
            },
            "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
            "summary_model_id": "8aea788c-bb30-4898-809e-e40c14ffb48c",
            "vlm_config": {
                "enabled": false,
                "model_id": ""
            },
            "asr_config": {
                "enabled": false,
                "model_id": "",
                "language": ""
            },
            "storage_provider_config": {
                "provider": "local"
            },
            "storage_config": {
                "secret_id": "",
                "secret_key": "",
                "region": "",
                "bucket_name": "",
                "app_id": "",
                "path_prefix": ""
            },
            "extract_config": null,
            "faq_config": null,
            "question_generation_config": null,
            "is_pinned": false,
            "pinned_at": null,
            "knowledge_count": 8,
            "chunk_count": 210,
            "processing_count": 0,
            "created_at": "2025-08-12T11:30:09.206238+08:00",
            "updated_at": "2025-08-12T11:30:09.206238+08:00",
            "deleted_at": null
        }
    ],
    "success": true
}
```

---

Doc completo, estrutura md intacta, exemplos JSON/curl não tocados (só chaves texto traduzidas).
