# API Reference: Knowledge Bases and Knowledge

Create knowledge bases, import and manage documents, check processing progress, and copy or move content.

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
| `summary_model_id` | string | No | Summary model, also the default model for auto-tagging and AI descriptions |
| `auto_tag_config` / `profile_config` | object | No | Auto-tagging and AI knowledge base description (`document` type only; see below) |

Response: 201 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Product documentation","type":"document"}'
```

### 自动标签与 AI 描述配置

创建知识库时 `auto_tag_config`、`profile_config` 位于顶层；更新时放在 `config.auto_tag_config`、`config.profile_config`。两者仅 document 知识库支持，默认 enabled=false。

`auto_tag_config`（自动标签）：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `enabled` | bool | false | 解析后异步从已有标签中选择 |
| `model_id` | string | 空 | 为空时使用知识库 `summary_model_id` |
| `max_tags` | int | 3 | 每篇最多关联数量，上限 10 |
| `skip_if_tagged` | bool | true | 已有标签则跳过；false 允许补充标签 |

开启后对新解析/重新解析的文档生效，不自动扫描全部旧文档。无候选标签或无可用模型时不阻断入库。

`profile_config`（AI 知识库描述）：

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `enabled` | bool | false | 开启后，文档新增、删除、移动或摘要更新会自动刷新 `generated_profile` |
| `model_id` | string | 空 | 为空时使用知识库 `summary_model_id` |
| `custom_instructions` | string | 空 | 追加到生成提示词的补充要求 |

`generated_profile` 为只读字段，由系统写入，不覆盖手写 `description`；也可通过下文的 `profile/generate` 立即生成。更新示例：

```bash
curl -X PUT "$BASE/api/v1/knowledge-bases/kb-1" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"产品文档","config":{"auto_tag_config":{"enabled":true,"max_tags":3,"skip_if_tagged":true}}}'
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
| `config` | object | No | Partial configuration update: `chunking_config`, `image_processing_config`, `faq_config`, `wiki_config`, `auto_tag_config`, `profile_config`, `indexing_strategy` |

Response: 200 `{"success":true,"data":{KnowledgeBase}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Product documentation v2"}'
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

Purpose: low-level recall within a KB (vector + keyword). No rerank by default, so recall scores are returned; rerank can optionally be enabled. Suited to scenarios that need control over raw recall, such as evaluating recall or passing precomputed vectors; for general retrieval use [`knowledge-search`](./02-api-chat.md), and see [Choosing a retrieval API](./01-api-overview.md#retrieval-api) for how to pick. Permission: Viewer+, KB read; API key `retrieve`/full. GET with a JSON body is supported only for backward compatibility (#1727); POST is recommended.

Query parameter: `resource_urls=handle|public` (`public` replaces `resource://` in the result `content` / `image_info` with loadable direct links; see the [API Overview](./01-api-overview.md) for details).

Request body (`types.SearchParams`):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query_text` | string | Conditionally required | Query text (unless `query_embedding` is provided; required when rerank is enabled) |
| `query_embedding` | []float32 | No | Precomputed vector |
| `vector_threshold` / `keyword_threshold` | float64 | No | Match thresholds |
| `match_count` | int | No | Maximum number of results (default 50) |
| `disable_keywords_match` / `disable_vector_match` | bool | No | Disable a given retrieval path |
| `knowledge_base_ids` | []string | No | Search multiple knowledge bases at once; the `:id` in the path must be among them, and these knowledge bases must share the same embedding model, otherwise 400 is returned |
| `knowledge_ids` | []string | No | Restrict to specific knowledge entries |
| `tag_ids` | []string | No | Tag filter (OR) |
| `only_recommended` | bool | No | FAQ: only recommended entries |
| `skip_context_enrichment` | bool | No | Skip parent-chunk/context enrichment |
| `rerank` | object | No | Passing it enables rerank (`{}` uses the model configured for the space); see the [rerank object](./01-api-overview.md#retrieval-api) for fields |

Response: 200 `{"success":true,"data":[SearchResult]}`; with `rerank`, there's an additional `meta.rerank` (see [meta.rerank diagnostics](./01-api-overview.md#retrieval-api)).

```bash
curl -X POST "$BASE/api/v1/knowledge-bases/kb-1/hybrid-search?resource_urls=public" -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query_text":"refund process","match_count":5}'

# Fix the recall parameters, then rerank with a specific model
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/hybrid-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query_text":"refund process","vector_threshold":0.3,"match_count":5,"rerank":{"model_id":"rr-1","threshold":0.2}}'
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

### POST /api/v1/knowledge-bases/:id/profile/generate

用途：立即重新生成知识库的 AI 描述（`generated_profile`），同步执行一次文档画像聚合和一次小模型调用，不修改手写 `description`。权限：与更新知识库相同（创建者/Admin 且 KB write）；API key `manage_kbs`/full。无请求体。仅 document 类型；未配置模型返回 400。

响应：200 `{"success":true,"data":{"gist","topics":[...],"typical_questions":[...],"stats":{"document_count",...},"status":"ready","model_id","generated_at"}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/profile/generate -H "Authorization: Bearer $TOKEN"
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

Purpose: KB-scoped file proxy (renders images inside shared KB content; the context tenant is rewritten to the KB owner). Permission: Viewer+, KB read; a KB-restricted key is rejected, a full-space `retrieve`/full key is allowed. Registered in `serveKBScopedFiles` (`internal/router/files.go`).

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
| `process_config` | JSON string | No | Parsing configuration override (KnowledgeProcessOverrides); see the table below |

Common `process_config` fields (all optional; the knowledge base configuration is used when omitted):

| Field | Type | Default | Description |
| --- | --- | --- | --- |
| `summary_enabled` | bool | true | Whether to generate summaries for the documents in this import; when off, parsing, indexing, and other processing still run as usual |
| `parser_engine_rules` | []object | Knowledge base configuration | Parser engine per file type |
| `parser_engine_overrides` | map[string]string | Empty | Engine parameters, e.g. `pdf_force_scanned` |
| `chunking_config` | object | Knowledge base configuration | Chunking parameters |
| `enable_multimodel` / `vlm_config` / `asr_config` | - | Knowledge base configuration | Multimodal and speech recognition |
| `question_generation_config` | object | Knowledge base configuration | Question generation |
| `graph_enabled` / `extract_config` | - | Knowledge base configuration | Graph extraction |

Response: 200 `{"success":true,"data":{Knowledge}}`; a duplicate file returns 409 with `data` set to the existing Knowledge. Files with the same name that are being deleted or failed to parse don't count as duplicates.

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
  -H 'Content-Type: application/json' -d '{"title":"FAQ Summary","content":"# Content","status":"publish"}'
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
| `folder_path` | string | No | Filter by folder; an empty string means the knowledge base root, and omitting it disables folder filtering |
| `folder_recursive` | bool | No | Used with `folder_path`; when `true`, documents in subfolders are included |
| `sort_by` | string | No | Sort field: `updated_at`, `created_at`, or `file_name`; default `created_at` |
| `sort_order` | string | No | Sort direction: `asc` or `desc`; default `desc` |

Without sort parameters, results are sorted by `created_at desc`; values outside the ranges above return 400. With `updated_at`, re-parsing, editing, or status changes affect the order; with `file_name`, results are sorted case-insensitively by the displayed file name, and an empty file name falls back to the title and then the source. Ties on the sort value are ordered by knowledge ID, so pagination stays stable.

Response: 200 `{"success":true,"data":[Knowledge],"total","page","page_size"}`

```bash
curl "$BASE/api/v1/knowledge-bases/kb-1/knowledge?page=1&parse_status=completed" -H "X-API-Key: $API_KEY"
```

### POST /api/v1/knowledge-bases/:id/knowledge/batch-download

用途：把同一知识库中的多个文档原始文件打包为 ZIP 下载。权限与单文件下载相同：Contributor+ 且 KB write（组织共享 Viewer 不可下载）；API key `retrieve`/full。

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `ids` | []string | 是 | 知识 ID 列表，1～200 个 |

行为：

- 原始文件合计不超过 512 MiB，超出返回 400；
- 没有原始文件的条目（如网页导入）会被跳过；所选条目都没有原始文件时返回 400；
- ZIP 内保留知识库文件夹结构，重名文件自动加序号；
- 任一 ID 不存在或不属于该知识库返回 404，读取失败返回 500，不会生成缺文件的压缩包；
- 同一实例同时最多处理 4 个批量下载，超出返回 429。

响应：200 `application/zip` 文件流，文件名形如 `knowledge-files-20260923-150405.zip`。

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/knowledge/batch-download \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"ids":["k-1","k-2"]}' -o knowledge-files.zip
```

### GET /api/v1/knowledge-bases/:id/knowledge/folders

Purpose: get the knowledge base's folder directory tree. When uploading an entire directory, the folder structure is preserved (the `knowledges.folder_path` column exists since migration `000079`; paths from historical `file_name` values have been backfilled into this field). Permission: Viewer+ + KBAccessRead.

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
  -H 'Content-Type: application/json' -d '{"from":"Design Documents/legacy","to":"Archive/Design Documents"}'
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

Response: 200 `{"success":true,"data":[Knowledge]}`. Knowledge in `pending`/`processing`/`finalizing` additionally carries `last_activity_at` (RFC3339), the later of the row's `updated_at` and the most recent write to any of that knowledge's spans. Knowledge with no progress for more than 20 minutes also carries `stall_state`: `queued` means there are still tasks waiting in the asynq queue or the Wiki persistent queue (a backlog), and `stalled` means no task is left to move it forward (suspected stuck). The judgment is the same as housekeeping's backlog judgment; the queue side is a single full-queue scan shared by all requests and cached for 60 seconds. When the probe fails, `stall_state` isn't returned, and the frontend shows it as ordinary parsing in progress.

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

Response: 200 `{"success":true,"data":{"knowledge_id","attempt","latest_attempt","parse_status","current_stage","last_activity_at","stall_state","trace":{...},"last_error":{...}}}`

`last_activity_at` is only returned while parsing is in progress, and is the later of the row's `updated_at` and the most recent write to any span of this attempt; `stall_state` has the same meaning as above. `current_stage` is the stage still running; when no stage is running (e.g. `finalizing`, where the post-processing stage has closed but child tasks such as summarization are still running), it's the stage that owns a still-running child span. For knowledge that housekeeping judged to be stuck, the span where it got stuck is marked failed with `TASK_STALLED`, and `last_error` points to it first.

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

Purpose: update knowledge metadata. Permission: same as above. Request body (subset of `types.Knowledge`): `title`, `description`, `tags`, `custom_metadata` (all optional). Omitting description keeps the existing summary, an explicit empty string clears the summary, and a non-empty value saves a manual summary; the UI lets you edit it on the document content page.

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
  -d '{"title":"New title","custom_metadata":{"department":"R&D Center","classification":"Internal","version":3}}'
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
  -H 'Content-Type: application/json' -d '{"content":"# Updated content","status":"publish"}'
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
  -H 'Content-Type: application/json' -d '{"caption":"Architecture diagram"}'
```

### GET /api/v1/knowledge/search

Purpose: cross-KB file search (used by the conversation @file picker). Permission: Viewer+; API key `retrieve`/full.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `keyword` / `query` | string | Conditionally required | Keyword (the two are equivalent); when empty, `recent=true` must be passed, otherwise 400 is returned |
| `file_types` | string | No | Comma-separated extension filter, e.g. `csv,xlsx` |
| `offset` / `limit` | int | No | Pagination; `limit` defaults to 20, range 1–100 |
| `recent` | bool | No | Returns recent files when the keyword is empty |
| `agent_id` | string | No | Shared Agent scope |
| `agent_source_tenant_id` | uint64 | No | Source-space selector for a shared Agent, validated against the sharing relationship |

Response: 200 `{"success":true,"data":[Knowledge],"has_more":bool,"total":N}`

```bash
curl "$BASE/api/v1/knowledge/search?keyword=report&limit=20" -H "Authorization: Bearer $TOKEN"
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
  -d '{"kb_id":"kb-1","knowledge_ids":["k-1","k-2"],"folder_path":"Design Documents"}'
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

## 实现参考

路由注册：`internal/router/routes_knowledge.go` 的 `RegisterKnowledgeBaseRoutes`、`RegisterKnowledgeRoutes`。Handler：`internal/handler/knowledgebase.go`、`internal/handler/knowledge.go`。
