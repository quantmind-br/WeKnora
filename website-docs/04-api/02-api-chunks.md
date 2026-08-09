# API Reference: Chunks & Tags

A chunk is the smallest unit of retrieval, and tags are used to categorize documents. Both sets of endpoints are nested under the knowledge base, sharing the same permission rules as [Knowledge Bases & Knowledge](./02-api-knowledge.md): reads require Viewer+ with read permission on the parent KB (API key `retrieve`); writes require "KB creator OR Admin+" with write permission (API key `ingest`), and both are constrained by the API key's KB allowlist.

Route registration: `RegisterChunkRoutes`, `RegisterKnowledgeTagRoutes`, and `RegisterChunkerDebugRoutes` in `internal/router/routes_knowledge.go`.

For common conventions (Base URL, authentication, error codes, pagination), see the [API Overview](./01-api-overview.md).

## Chunks (/api/v1/chunks)

Handler: `internal/handler/chunk.go`. Read: Viewer+ with parent KB read (API key `retrieve`/full); Write: KB creator OR Admin+ with parent KB write (API key `ingest`/full).

### GET /api/v1/chunks/:knowledge_id

Purpose: list the chunks for a piece of knowledge.

| Query Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `page` | int | No | Default 1 |
| `page_size` | int | No | Default 10, max 100 |
| `chunk_type` | string | No | Repeatable, filters by chunk type |

Response: 200 `{"success":true,"data":[Chunk],"total","page","page_size"}`

```bash
curl "$BASE/api/v1/chunks/k-1?page=1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/chunks/by-id/:id

Purpose: fetch a single chunk by chunk ID (no knowledge_id required).

Response: 200 `{"success":true,"data":{Chunk}}`

```bash
curl $BASE/api/v1/chunks/by-id/c-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/chunks/:knowledge_id/:id

Purpose: edit chunk content or enable/disable status (versioned optimistic updates since migration `000078`).

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `content` | string | No | New content; must not be empty after trimming leading/trailing whitespace, max length 200000 bytes |
| `is_enabled` | bool | No | Enable/disable this chunk |
| `expected_revision` | int | No | Expected `content_revision`, used for optimistic concurrency control |

Constraints and side effects:

- Only `text`-type chunks can be edited; other types return 400;
- If `expected_revision` doesn't match the current `content_revision`, returns **409** (`Chunk was modified by another user; refresh and retry`);
- Introducing image URLs not present in the source content during editing is not allowed; deleting a Markdown image will also disable the corresponding OCR/caption sub-chunk;
- On successful edit, `content_revision` is incremented by 1, the old version is written to the `chunk_revisions` table, and `index_status` transitions through `processing` → `ready`; if rebuilding the retrieval index fails, the row is still saved but `index_status = failed`, and resubmitting the same content will trigger a retry;
- Editing a sub-chunk writes changes back to the parent chunk's content by offset (the parent chunk's `source_content` remains immutable);
- Changes to content or enabled status enqueue a document summary refresh.

Response: 200 `{"success":true,"data":{Chunk},"summary_status":"pending","description":"..."}`

```bash
curl -X PUT $BASE/api/v1/chunks/k-1/c-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"content":"Corrected content","expected_revision":2}'
```

### GET /api/v1/chunks/:knowledge_id/:id/revisions

Purpose: list the historical versions of a chunk (`chunk_revisions` table, sorted by revision descending).

Response: 200 `{"success":true,"data":[{ChunkRevision}]}`, each entry contains `revision`, `content`, `is_enabled`, `editor_id`, `edit_source`, `edited_at`.

```bash
curl $BASE/api/v1/chunks/k-1/c-1/revisions -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/chunks/:knowledge_id/:id/revert

Purpose: roll back to a historical version. A revert is itself a new edit: `content_revision` continues to increment, and the current content is saved as a new historical version.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `revision` | int | Yes | Target historical revision number (non-negative) |
| `expected_revision` | int | No | Optimistic lock, same semantics as above; returns 409 on conflict |

Response: 200 `{"success":true,"data":{Chunk},"summary_status":"...","description":"..."}`

```bash
curl -X POST $BASE/api/v1/chunks/k-1/c-1/revert -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"revision":1}'
```

### DELETE /api/v1/chunks/:knowledge_id/:id

Purpose: delete a single chunk.

Response: 200 `{"success":true,"message":"Chunk deleted"}`

```bash
curl -X DELETE $BASE/api/v1/chunks/k-1/c-1 -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/chunks/:knowledge_id

Purpose: delete all chunks under a piece of knowledge.

Response: 200 `{"success":true,"message":"All chunks under knowledge deleted"}`

```bash
curl -X DELETE $BASE/api/v1/chunks/k-1 -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/chunks/by-id/:id/questions

Purpose: delete a specific generated question under a chunk. Request body: `{"question_id":"..."}` (`binding:"required"`).

Response: 200 `{"success":true,"message":"Generated question deleted"}`

```bash
curl -X DELETE $BASE/api/v1/chunks/by-id/c-1/questions -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"question_id":"q-1"}'
```

### PUT /api/v1/chunks/by-id/:id/questions

Purpose: add or edit a generated question for a chunk. Request body: `{"question":"...","question_id":"..."}`; `question` is required; leaving `question_id` empty indicates a new entry.

Response: 200 `{"success":true,"data":{GeneratedQuestion}}`

```bash
curl -X PUT $BASE/api/v1/chunks/by-id/c-1/questions -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"question_id":"q-1","question":"How is the vector store configured in WeKnora?"}'
```

### POST /api/v1/chunks/by-id/:id/questions/regenerate

Purpose: regenerate retrieval questions based on the chunk's current content. After content edits, the existing questions are not deleted but marked as "stale" (revision no longer matches the current text) and can be refreshed via this endpoint.

Response: 200 `{"success":true,"data":[{GeneratedQuestion}]}`

```bash
curl -X POST $BASE/api/v1/chunks/by-id/c-1/questions/regenerate -H "Authorization: Bearer $TOKEN"
```

## Tags (/api/v1/knowledge-bases/:id/tags)

Handler: `internal/handler/tag.go`. Read: Viewer+ + KB read (API key `retrieve`/full); Write: KB creator OR Admin+ + KB write (API key `ingest`/full).

### GET /api/v1/knowledge-bases/:id/tags

Purpose: list tags. Query parameters: `page`, `page_size`, `keyword` (all optional).

Response: 200 `{"success":true,"data":[KnowledgeTag]}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/tags -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledge-bases/:id/tags

Purpose: create a tag.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required"`) | Tag name |
| `color` | string | No | Color |
| `sort_order` | int | No | Sort order |

Response: 200 `{"success":true,"data":{KnowledgeTag}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/tags -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"After-sales"}'
```

### PUT /api/v1/knowledge-bases/:id/tags/:tag_id

Purpose: update a tag (`tag_id` supports either a UUID or an integer seq_id). Request body: `name`/`color`/`sort_order` (pointer fields, all optional).

Response: 200 `{"success":true,"data":{KnowledgeTag}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/tags/t-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"After-sales Support"}'
```

### DELETE /api/v1/knowledge-bases/:id/tags/:tag_id

Purpose: delete a tag. Query parameters: `force` (bool, force delete), `content_only` (bool, delete content only while keeping the tag). Request body (optional): `{"exclude_ids":[int64]}`.

Response: 200 `{"success":true}`

```bash
curl -X DELETE "$BASE/api/v1/knowledge-bases/kb-1/tags/t-1?force=true" -H "Authorization: Bearer $TOKEN"
```

## Chunker Debugging

### POST /api/v1/chunker/preview

Purpose: stateless chunking preview (used by the KB editor's debug panel). Permission: Viewer+; API key `retrieve`/`ingest`/full. Handler: `internal/handler/chunker_debug.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `text` | string | Yes (validated non-empty in code, ≤64k characters) | Sample text |
| `chunking_config.chunk_size` | int | No | Chunk size in characters |
| `chunking_config.chunk_overlap` | int | No | Overlap |
| `chunking_config.separators` | []string | No | Separators |
| `chunking_config.strategy` | string | No | `auto/heading/heuristic/recursive/legacy` |
| `chunking_config.token_limit` | int | No | Token limit |
| `chunking_config.languages` | []string | No | Language hints |
| `chunking_config.enable_parent_child` | bool | No | Trial-split by parent/child chunking; returns the child chunks (matching retrieval granularity) |
| `chunking_config.parent_chunk_size` / `child_chunk_size` | int | No | Parent/child chunk size, defaulting to 4096 / 384 |

Response: 200 `{"success":true,"data":{"selected_tier","tier_chain","rejected","profile","chunks":[...],"stats":{count,avg_chars,min_chars,max_chars,stddev_chars,truncated_to}}}`; 413 if the text is too long; 504 if chunking times out (5s).

```bash
curl -X POST $BASE/api/v1/chunker/preview -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"text":"# Title\nBody...","chunking_config":{"chunk_size":512}}'
```

---

Tradução completa concluída, estrutura markdown preservada integralmente.
