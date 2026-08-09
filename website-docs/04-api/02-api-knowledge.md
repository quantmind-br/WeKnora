# API Reference: Knowledge Bases and Knowledge

Route registration: `RegisterKnowledgeBaseRoutes`, `RegisterKnowledgeRoutes` in `internal/router/routes_knowledge.go`. Handlers: `internal/handler/knowledgebase.go`, `internal/handler/knowledge.go`.

Permissions cheat sheet: read routes require Viewer+ and read permission on the KB (owned / organization-shared / visible via shared Agent); write routes require "KB creator OR Admin+" and write permission. API key: read requires `retrieve`, content writes require `ingest`, KB lifecycle requires `manage_kbs` (all overridable by full-access), and are subject to the KB allowlist.

The chunking, tags, and chunk preview endpoints (`/chunks`, `/knowledge-bases/:id/tags`, `/chunker/preview`) are covered in [Chunking and Tags](./02-api-chunks.md).

## Knowledge Bases (/api/v1/knowledge-bases)

### POST /api/v1/knowledge-bases

Purpose: create a knowledge base. Permission: Contributor+; API key `manage_kbs`/full. Handler: `internal/handler/knowledgebase.go`

Request body (`types.KnowledgeBase`):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Name |
| `description` | string | No | Description |
| `type` | string | No | `document` (default) / `faq` / `wiki` |
| `embedding_model_id` | string | No | Embedding model ID |
| `chunking_config` | object | No | Chunking configuration (chunk_size/overlap/separators/strategy…) |
| `image_processing_config` | object | No | Image processing (multimodal) configuration |
| `storage_provider_config` | object | No | Storage configuration |
| `vector_store_id` | string | No | Vector store binding (invalid values return code 2200/2201) |
| `faq_config` / `wiki_config` / `extract_config` / `indexing_strategy` | object | No | Type-specific configuration |

Response: 201 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"产品文档","type":"document"}'
```

### GET /api/v1/knowledge-bases

Purpose: list knowledge bases. Permission: Viewer+; API key `retrieve`/full.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `agent_id` | string | No | Filter to KBs visible to a given shared Agent |
| `agent_source_tenant_id` | uint64 | No | When an Agent of the same name is shared by multiple spaces, specify the source space; the value is validated against the sharing relationship and an invalid value returns 400 directly |
| `creator` | string | No | `mine` / `others` |

Response: 200 `{"success":true,"data":[KnowledgeBase],"total","page","page_size"}`

```bash
curl $BASE/api/v1/knowledge-bases -H "X-API-Key: $API_KEY"
```

### GET /api/v1/knowledge-bases/:id

Purpose: knowledge base detail (a shared KB includes `my_permission`). Permission: Viewer+, KB read. Query parameter: `agent_id` (optional).

Response: 200 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id

Purpose: update a knowledge base. Permission: creator OR Admin+, KB write; API key `manage_kbs`/full.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required"`) | Name |
| `description` | string | No | Description |
| `config` | object | No | Partial configuration update (chunking/image/wiki/indexing strategy) |

Response: 200 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"产品文档 v2"}'
```

### DELETE /api/v1/knowledge-bases/:id

Purpose: delete a knowledge base (locked to the owning space + Admin; a shared editor cannot delete). Permission: creator OR Admin+, KB write; API key `manage_kbs`/full.

Response: 200 `{"success":true,"message":"Knowledge base deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id/pin

Purpose: pin/unpin (stored per user). Permission: Viewer+, KB read. No request body.

Response: 200 `{"success":true,"data":{KnowledgeBase(is_pinned toggled)}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/pin -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledge-bases/:id/hybrid-search (GET also supported)

Purpose: hybrid search within a KB (vector + keyword). Permission: Viewer+, KB read; API key `retrieve`/full. GET with a JSON body is supported only for backward compatibility (#1727); POST is recommended.

Request body (`types.SearchParams`):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query_text` | string | Conditionally required | Query text (unless `query_embedding` is provided) |
| `query_embedding` | []float32 | No | Precomputed vector |
| `vector_threshold` / `keyword_threshold` | float64 | No | Match thresholds |
| `match_count` | int | No | Maximum number of results |
| `disable_keywords_match` / `disable_vector_match` | bool | No | Disable a given retrieval path |
| `knowledge_ids` | []string | No | Restrict to specific knowledge entries |
| `tag_ids` | []string | No | Tag filter (OR) |
| `only_recommended` | bool | No | FAQ: only recommended entries |
| `skip_context_enrichment` | bool | No | Skip parent-chunk/context enrichment |

Response: 200 `{"success":true,"data":[SearchResult]}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/hybrid-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query_text":"退款流程","match_count":5}'
```

### POST /api/v1/knowledge-bases/copy

Purpose: copy content across KBs (asynchronous task). Permission: Contributor+; API key `manage_kbs`/full (source/target KB allowlist validated in the handler).

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `source_id` | string | Yes (`binding:"required"`) | Source KB |
| `target_id` | string | No | Target KB (auto-created if empty) |
| `task_id` | string | No | Custom task ID |

Response: 200 `{"success":true,"data":{"task_id","source_id","target_id","message"}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/copy -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"source_id":"kb-1"}'
```

### POST /api/v1/knowledge-bases/:id/duplicate

Purpose: create a KB duplicate (copies settings only, not content/index/sharing). Permission: Contributor+, source KB read; API key `manage_kbs`/full. No request body.

Response: 201 `{"success":true,"data":{"source_id","target_id","message","knowledge_base":{...}}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/duplicate -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/copy/progress/:task_id

Purpose: query copy progress (tasks are isolated per space). Permission: Viewer+; API key `retrieve`/`manage_kbs`/full.

Response: 200 `{"success":true,"data":{status,progress,message,...}}`

```bash
curl $BASE/api/v1/knowledge-bases/copy/progress/task-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/move-targets

Purpose: list KBs that can serve as a move target (same type/same embedding). Permission: Viewer+, KB read.

Response: 200 `{"success":true,"data":[KnowledgeBase]}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/move-targets -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/files

Purpose: KB-scoped file proxy (renders images inside shared KB content; the context tenant is rewritten to the KB owner). Permission: Viewer+, KB read; a KB-restricted key is rejected, a full-space `retrieve`/full key is allowed. Registered in `serveKBScopedFiles` (`internal/router/router.go`).

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_path` | string | Yes | `provider://...` storage path (`..` forbidden) |

Response: 200 file stream (`Content-Type` inferred from extension; `Cache-Control: private`).

```bash
curl "$BASE/api/v1/knowledge-bases/kb-1/files?file_path=local://1/exports/chart.png" \
  -H "Authorization: Bearer $TOKEN" -o chart.png
```

## Knowledge (KB content, /api/v1/knowledge-bases/:id/knowledge and /api/v1/knowledge)

### POST /api/v1/knowledge-bases/:id/knowledge/file

Purpose: upload a file to create a knowledge entry. Permission: KB creator OR Admin+, KB write; API key `ingest`/full. Handler: `internal/handler/knowledge.go`

multipart/form-data fields:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `file` | file | Yes | Uploaded file |
| `fileName` | string | No | Override display name |
| `metadata` | JSON string | No | Custom metadata |
| `enable_multimodel` | bool | No | Multimodal processing toggle |
| `tag_ids` | string | No | Comma-separated tag IDs |
| `channel` | string | No | Ingestion channel |
| `process_config` | JSON string | No | Parsing configuration override (KnowledgeProcessOverrides) |

Response: 200 `{"success":true,"data":{Knowledge}}`; a duplicate file returns 409 with `data` set to the existing Knowledge.

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/file \
  -H "X-API-Key: $API_KEY" -F 'file=@./manual.pdf' -F 'enable_multimodel=true'
```

### POST /api/v1/knowledge-bases/:id/knowledge/url

Purpose: create a knowledge entry by fetching a URL. Permission/API key: same as above.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `url` | string | Yes (`binding:"required"`) | URL to fetch |
| `file_name` / `file_type` / `title` | string | No | Override info |
| `enable_multimodel` | *bool | No | Multimodal toggle |
| `tag_ids` | []string | No | Tags |
| `channel` | string | No | Channel |
| `process_config` | object | No | Parsing override |

Response: 201 `{"success":true,"data":{Knowledge}}`; a duplicate URL returns 409.

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/url -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"url":"https://example.com/doc"}'
```

### POST /api/v1/knowledge-bases/:id/knowledge/manual

Purpose: create a manual (Markdown) knowledge entry. Permission/API key: same as above.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `title` | string | No | Title |
| `content` | string | No | Markdown content |
| `status` | string | No | `draft` / `publish` |
| `tag_ids` | []string | No | Tags |
| `channel` | string | No | Channel |
| `process_config` | object | No | Parsing override |

Response: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/manual -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"FAQ 汇总","content":"# 内容","status":"publish"}'
```

### GET /api/v1/knowledge-bases/:id/knowledge

Purpose: list knowledge entries under a KB. Permission: Viewer+, KB read; API key `retrieve`/full.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `page` / `page_size` | int | No | Pagination (default 1/20) |
| `tag_ids` | string | No | Comma-separated tags (OR) |
| `keyword` | string | No | Keyword |
| `file_type` | string | No | File type filter |
| `parse_status` | string | No | `pending/processing/completed/failed` |
| `source` | string | No | Channel or `manual`/`url` |
| `start_time` / `end_time` | string | No | RFC3339, filtered on `updated_at` |

Response: 200 `{"success":true,"data":[Knowledge],"total","page","page_size"}`

```bash
curl "$BASE/api/v1/knowledge-bases/kb-1/knowledge?page=1&parse_status=completed" -H "X-API-Key: $API_KEY"
```

### GET /api/v1/knowledge-bases/:id/knowledge/folders

Purpose: get the knowledge base's folder directory tree. When uploading an entire directory, the folder structure is preserved (the `knowledges.folder_path` column exists since migration `000079`; earlier data that had the path embedded in `file_name` has been backfilled). Permission: Viewer+ + KBAccessRead.

Response: 200 `{"success":true,"data":[{FolderNode}]}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/knowledge/folders -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id/knowledge/folders

Purpose: rename or move a folder, updating the path of all its subdirectories together. If the target path already exists, the two folders are merged; moving a folder into its own subdirectory is not allowed. Permission: KB owner or Admin+ + KBAccessWrite.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `from` | string | Yes | Original path |
| `to` | string | Yes | New path |

Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/knowledge/folders -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"from":"设计文档/旧版","to":"归档/设计文档"}'
```

### DELETE /api/v1/knowledge-bases/:id/knowledge

Purpose: clear all content in a KB (destructive). Permission: Admin+, KB write; API key full-access only.

Response: 200 `{"success":true,"message":"Knowledge base contents clear task submitted","data":{"deleted_count":N}}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1/knowledge -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/batch

Purpose: batch-fetch knowledge entries by ID (across KBs; the handler validates access itself). Permission: Viewer+; API key `retrieve`/full.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `ids` | []string | Yes | Knowledge IDs (repeated parameter or comma-separated) |
| `kb_id` | string | No | Restrict to a KB |
| `agent_id` | string | No | Shared Agent scope |
| `agent_source_tenant_id` | uint64 | No | Source-space selector for a shared Agent, validated against the sharing relationship |

Response: 200 `{"success":true,"data":[Knowledge]}`

```bash
curl "$BASE/api/v1/knowledge/batch?ids=k-1&ids=k-2" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id

Purpose: knowledge entry detail. Permission: Viewer+, parent KB read.

Response: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl $BASE/api/v1/knowledge/k-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id/stages and GET /api/v1/knowledge/:id/spans

Purpose: parsing stages/trace (both paths share the same handler, `GetKnowledgeSpans`). Permission: Viewer+, parent KB read. Query parameter: `attempt` (int, 0 = latest).

Response: 200 `{"success":true,"data":{"knowledge_id","attempt","latest_attempt","parse_status","current_stage","trace":{...},"last_error":{...}}}`

```bash
curl $BASE/api/v1/knowledge/k-1/spans -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/knowledge/:id

Purpose: delete a knowledge entry (asynchronous). Permission: KB creator OR Admin+, KB write; API key `ingest`/full.

Response: 200 `{"success":true,"message":"Delete task submitted","data":{"task_id"}}`

```bash
curl -X DELETE $BASE/api/v1/knowledge/k-1 -H "X-API-Key: $API_KEY"
```

### PUT /api/v1/knowledge/:id

Purpose: update knowledge metadata. Permission: same as above. Request body (subset of `types.Knowledge`): `title`, `description`, `tags`, `custom_metadata` (all optional).

`custom_metadata` is user-supplied descriptive metadata (stored separately from the internally used `metadata`, migration `000078`); validation rules are in `internal/application/service/knowledge.go`:

| Constraint | Value |
| --- | --- |
| Number of fields | ≤ 20 |
| Key length | 1-64 characters, cannot be blank |
| Value type | string / number / boolean / null |
| Value length | ≤ 1000 characters |

This is a full overwrite update (the passed-in object replaces the existing one). If the metadata changes and the document already has a summary, a summary refresh is automatically queued (`summary_status` becomes `pending`). Metadata text feeds into summary generation and the document-level model context (`Knowledge.CustomMetadataText()`).

Response: 200 `{"success":true,"message":"Knowledge updated successfully","data":{Knowledge}}`

```bash
curl -X PUT $BASE/api/v1/knowledge/k-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"title":"新标题","custom_metadata":{"部门":"研发中心","密级":"内部","版本":3}}'
```

### POST /api/v1/knowledge/:id/regenerate-summary

Purpose: regenerate a document's summary after its chunk content or custom metadata has been edited. Permission: KB owner or Admin+, and write permission on the parent KB.

Behavior falls into two cases: if the document had no prior summary (`summary_status` empty or `none`), generation is triggered synchronously; if a summary already exists, a refresh task is queued instead, `summary_status` becomes `pending`, and it is executed asynchronously by `knowledge_summary_refresh.go`.

Response: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge/k-1/regenerate-summary -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge/manual/:id

Purpose: update manual knowledge content (subset of `ManualKnowledgePayload`: `title/content/status/...`). Permission: same as above.

Response: 200 `{"success":true,"data":{Knowledge}}`

```bash
curl -X PUT $BASE/api/v1/knowledge/manual/k-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"content":"# 更新内容","status":"publish"}'
```

### POST /api/v1/knowledge/:id/reparse

Purpose: reparse a knowledge entry. Permission: same as above. Request body (optional): `{"process_config":{...}}`.

Response: 200 `{"success":true,"message":"Reparse task submitted","data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge/k-1/reparse -H "X-API-Key: $API_KEY"
```

### POST /api/v1/knowledge/:id/cancel-parse

Purpose: cancel parsing. Permission: same as above. No request body.

Response: 200 `{"success":true,"message":"Knowledge parse cancelled","data":{Knowledge}}`

```bash
curl -X POST $BASE/api/v1/knowledge/k-1/cancel-parse -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id/download

Purpose: download the original source file (stricter than preview: Contributor+ and KB write; an organization-shared Viewer cannot download the source file). API key `retrieve`/full.

Response: 200 binary stream (`application/octet-stream`).

```bash
curl -OJ $BASE/api/v1/knowledge/k-1/download -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/:id/preview

Purpose: preview the parsed file content. Permission: Viewer+, KB read.

Response: 200 preview stream (text/HTML).

```bash
curl $BASE/api/v1/knowledge/k-1/preview -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge/image/:id/:chunk_id

Purpose: update image info for a given chunk (caption/OCR, etc.). Permission: KB creator OR Admin+, KB write. Path parameters: `id` knowledge ID, `chunk_id` chunk ID. Request body is the image info JSON.

Response: 200 `{"success":true,...}`

```bash
curl -X PUT $BASE/api/v1/knowledge/image/k-1/c-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"caption":"架构图"}'
```

### GET /api/v1/knowledge/search

Purpose: cross-KB file search (used by the conversation @file picker). Permission: Viewer+; API key `retrieve`/full.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `q` | string | No | Keyword (if empty and `recent=true`, returns recent files) |
| `file_type` / `file_types` | string | No | Type filter (the latter comma-separated) |
| `page` / `page_size` | int | No | Pagination |
| `recent` | bool | No | Recent-files mode |
| `agent_id` | string | No | Shared Agent scope |
| `agent_source_tenant_id` | uint64 | No | Source-space selector for a shared Agent, validated against the sharing relationship |

Response: 200 `{"success":true,"data":[Knowledge]}`

```bash
curl "$BASE/api/v1/knowledge/search?q=报告&recent=false" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge/move/progress/:task_id

Purpose: query move task progress. Permission: Viewer+; API key `retrieve`/full.

Response: 200 `{"success":true,"data":{MoveProgress}}`

```bash
curl $BASE/api/v1/knowledge/move/progress/task-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge/tags

Purpose: batch-update knowledge tags. Permission: Contributor+; API key `ingest`/full (KB allowlist validated in the handler).

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `updates` | map[string][]string | Yes (`binding:"required,min=1"`) | knowledge_id → tag_ids |
| `kb_id` | string | No | Restrict to a KB |

Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge/tags -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"updates":{"k-1":["t-1"]},"kb_id":"kb-1"}'
```

### POST /api/v1/knowledge/batch-reparse

Purpose: batch reparse. Permission: Contributor+; API key `ingest`/full.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kb_id` | string | Yes (`binding:"required"`) | KB ID |
| `ids` | []string | Yes (`binding:"required"`) | List of knowledge IDs |
| `process_config` | object | No | Parsing override |

Response: 200 `{"success":true,"message":"Batch reparse task submitted","data":{"task_id"}}`

```bash
curl -X POST $BASE/api/v1/knowledge/batch-reparse -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"kb_id":"kb-1","ids":["k-1","k-2"]}'
```

### POST /api/v1/knowledge/batch-delete

Purpose: batch delete (≤200 entries). Permission: Contributor+; API key `ingest`/full.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kb_id` | string | Yes (`binding:"required"`) | KB ID |
| `ids` | []string | Yes (`binding:"required"`) | List of knowledge IDs (≤200) |

Response: 200 `{"success":true,"message":"Batch delete task submitted","data":{"task_id","deleted_count"}}`

```bash
curl -X POST $BASE/api/v1/knowledge/batch-delete -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"kb_id":"kb-1","ids":["k-1"]}'
```

### POST /api/v1/knowledge/folder

Purpose: assign a set of documents to a given folder (only changes the folder assignment; does not change the knowledge base ownership or trigger reparsing). Permission: Contributor+ / API key `ingest`.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `kb_id` | string | Yes | Knowledge base ID |
| `knowledge_ids` | []string | Yes | Documents to move |
| `folder_path` | string | No | Target folder; an empty string moves the document back to the knowledge base root |

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/knowledge/folder -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"kb_id":"kb-1","knowledge_ids":["k-1","k-2"],"folder_path":"设计文档"}'
```

### POST /api/v1/knowledge/move

Purpose: move knowledge entries across KBs (asynchronous). Permission: Contributor+; API key `ingest`/full (both source and target KB must be on the allowlist).

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `knowledge_ids` | []string | Yes (`binding:"required,min=1"`) | Knowledge entries to move |
| `source_kb_id` | string | Yes (`binding:"required"`) | Source KB |
| `target_kb_id` | string | Yes (`binding:"required"`) | Target KB |
| `mode` | string | Yes (`binding:"required,oneof=reuse_vectors reparse"`) | Reuse vectors or reparse |

Response: 200 `{"success":true,"data":{"task_id","source_kb_id","target_kb_id","knowledge_count","message"}}`

```bash
curl -X POST $BASE/api/v1/knowledge/move -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"knowledge_ids":["k-1"],"source_kb_id":"kb-1","target_kb_id":"kb-2","mode":"reuse_vectors"}'
```
