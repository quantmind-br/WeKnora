# API Reference: FAQ & Wiki

Manage FAQ entries and Wiki pages in a knowledge base, with support for import, retrieval, editing, and version restore.

Both groups are KB content sub-resources: reads require Viewer+ and KB read (API key `retrieve`/full); writes require "KB creator OR Admin+" and KB write (API key `ingest`/full), and are constrained by the KB whitelist.

## FAQ (/api/v1/knowledge-bases/:id/faq)

### GET /api/v1/knowledge-bases/:id/faq/entries

Purpose: List FAQ entries.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `page` / `page_size` | int | No | Pagination |
| `tag_id` | int | No | Legacy single-tag seq_id |
| `tag_ids` | string | No | Comma-separated tag UUIDs |
| `keyword` | string | No | Keyword |
| `search_field` | string | No | `standard_question`/`similar_questions`/`answers` (defaults to all fields) |
| `sort_order` | string | No | `asc` (defaults to descending by update time) |
| `is_enabled` | bool | No | Filter by enabled status: `true` returns enabled entries only, `false` disabled entries only; omit to return all; any other value returns 400 |

Response: 200 `{"success":true,"data":{paginated FAQEntry list}}`

```bash
curl "$BASE/api/v1/knowledge-bases/kb-1/faq/entries?page=1&is_enabled=false" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/faq/entries/export

Purpose: Export FAQs. Query parameter: `format` (`csv` default / `json`).

Response: 200 file download (`text/csv` or `application/json`).

```bash
curl -OJ "$BASE/api/v1/knowledge-bases/kb-1/faq/entries/export?format=csv" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/faq/entries/:entry_id

Purpose: FAQ entry details (`entry_id` is an integer seq_id).

Response: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/faq/entries/12 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledge-bases/:id/faq/entries

Purpose: Bulk upsert / import (async task). Handler method `UpsertEntries`.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `entries` | []FAQEntryPayload | Yes (`binding:"required"`) | Batch entries |
| `mode` | string | Yes (`binding:"oneof=append replace"`) | Append or replace |
| `knowledge_id` | string | No | FAQ knowledge entity ID |
| `task_id` | string | No | Custom task ID; only letters, digits, `_`, and `-` are allowed, max 128 characters, otherwise returns 400 |
| `dry_run` | bool | No | Validate only, no persistence |

Response: 200 `{"success":true,"data":{"task_id"}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/entries -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"mode":"append","entries":[{"standard_question":"How can I get a refund?","answers":["Contact support"]}]}'
```

### POST /api/v1/knowledge-bases/:id/faq/entry

Purpose: Create a single FAQ. Request body (`types.FAQEntryPayload`):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `standard_question` | string | Yes (`binding:"required"`) | Standard question |
| `similar_questions` | []string | No | Similar questions |
| `negative_questions` | []string | No | Negative example questions |
| `answers` | []string | No | Answer list |
| `answer_strategy` | string | No | `all` / `random` |
| `tag_id` | int64 | No | Tag seq_id |
| `tag_name` | string | No | Tag name |
| `is_enabled` / `is_recommended` | *bool | No | Enabled/recommended |

Response: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/entry -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"standard_question":"How can I get a refund?","answers":["Refundable within 7 days"]}'
```

### PUT /api/v1/knowledge-bases/:id/faq/entries/:entry_id

Purpose: Update a single FAQ entry (request body same as create).

Response: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/entries/12 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"standard_question":"How can I get a refund?","answers":["Refundable within 30 days"]}'
```

### POST /api/v1/knowledge-bases/:id/faq/entries/:entry_id/similar-questions

Purpose: Append similar questions. Request body: `{"similar_questions":["..."]}` (`binding:"required,min=1"`).

Response: 200 `{"success":true,"data":{FAQEntry}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/entries/12/similar-questions \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"similar_questions":["How do I request a refund?"]}'
```

### PUT /api/v1/knowledge-bases/:id/faq/entries/fields

Purpose: Bulk-update entry fields (`is_enabled`/`is_recommended`/`tag_id`).

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `by_id` | map[int64]object | No | Update by entry seq_id |
| `by_tag` | map[int64]object | No | Bulk update by tag |
| `exclude_ids` | []int64 | No | Entries to exclude when using `by_tag` |

Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/entries/fields -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"by_id":{"12":{"is_enabled":false}}}'
```

### PUT /api/v1/knowledge-bases/:id/faq/entries/tags

Purpose: Bulk-change entry tags. Request body: `{"updates":{"<entry_id>":<tag_id|null>}}` (`binding:"required,min=1"`; null removes the tag).

Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/entries/tags -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"updates":{"12":3}}'
```

### DELETE /api/v1/knowledge-bases/:id/faq/entries

Purpose: Bulk-delete entries. Request body: `{"ids":[int64]}` (`binding:"required,min=1"`).

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1/faq/entries -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"ids":[12,13]}'
```

### POST /api/v1/knowledge-bases/:id/faq/search

Purpose: FAQ retrieval (read-only semantics; a scoped key with `retrieve` can also call this).

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query_text` | string | Yes (`binding:"required"`) | Query |
| `vector_threshold` | float64 | No | Vector threshold, defaults to 0.7 |
| `match_count` | int | No | Defaults to 10, capped at 50 |
| `first_priority_tag_ids` / `second_priority_tag_ids` | []int64 | No | Tag priority filters |
| `only_recommended` | bool | No | Recommended entries only |

Response: 200 `{"success":true,"data":[FAQEntry(includes match_type/score)]}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/faq/search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query_text":"refund"}'
```

### PUT /api/v1/knowledge-bases/:id/faq/import/last-result/display

Purpose: Set the display state of the most recent import result panel. Request body: `{"display_status":"open|close"}` (`binding:"required,oneof=open close"`).

Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/faq/import/last-result/display \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"display_status":"close"}'
```

### GET /api/v1/faq/import/progress/:task_id

Purpose: Query FAQ import/dry-run progress (tasks are isolated per space/tenant). Permissions: Viewer+; API key `retrieve`/`ingest`/full.

Response: 200 `{"success":true,"data":{status,progress,failed_entries,...}}`

```bash
curl $BASE/api/v1/faq/import/progress/task-1 -H "X-API-Key: $API_KEY"
```

## Wiki (/api/v1/knowledgebase/:kb_id/wiki)

Note that this group's prefix is `/knowledgebase/:kb_id/wiki` (singular, no hyphen). Handler: `internal/handler/wiki_page.go`. Responses in this group are mostly **raw objects** (not wrapped in `success`).

### GET /api/v1/knowledgebase/:kb_id/wiki/pages

Purpose: List Wiki pages.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `page_type` | string | No | Comma-separated types |
| `status` | string | No | Page status |
| `query` | string | No | Full-text search |
| `category_path` | string | No | `/`-separated path filter |
| `folder_id` | string | No | Exact folder filter (empty string = root) |
| `category_depth` | int | No | Folder depth |
| `page` / `page_size` | int | No | Pagination (default 1/20) |
| `sort_by` / `sort_order` | string | No | Sort field: `title`, `created_at`, `updated_at`, `page_type`, `wiki_path`, `sort_order`, `depth`; any other value sorts by `updated_at`; defaults to `updated_at` desc |

Response: 200 `WikiPageListResponse`

```bash
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/pages?page=1" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/pages

Purpose: Create a page. Request body (`types.WikiPage`): `slug`, `title`, `content`, `folder_id`, `page_type`, etc. (all optional; slug is auto-generated when omitted).

Response: 201 `WikiPage`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/pages -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"Architecture Overview","content":"# Overview"}'
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/move-page

Purpose: Move a page into a folder. Request body: `{"slug":"<page slug>","folder_id":"<folder ID|empty=root>"}` (slug required).

Response: 200 `WikiPage`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/move-page -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"slug":"overview","folder_id":"f-1"}'
```

### GET /api/v1/knowledgebase/:kb_id/wiki/pages/*slug

Purpose: Get a page (`*slug` is a wildcard path).

Response: 200 `WikiPage`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/pages/overview -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/pages/*slug

Purpose: Update a page (request body same as create). The old version is first snapshotted in full into `wiki_page_revisions`, `version` is incremented, and `last_edit_source` is recorded as `user` (recorded as `agent` when written by an Agent tool).

Response: 200 `WikiPage`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/pages/overview -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"content":"# Updated Overview"}'
```

### GET /api/v1/knowledgebase/:kb_id/wiki/revisions/*slug

Purpose: Page revision history (migration `000075`). Permissions: Viewer+ + KBAccessRead.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `version` | int | No | When provided, returns the **full content of that version** (for diffing); an invalid value or < 1 returns 400, not found returns 404 (versions written before the upgrade and snapshots already cleaned up by the retention policy have no full content; this is expected) |
| `limit` | int | No | Defaults to 50, capped at 200; only applies in list mode |
| `offset` | int | No | Pagination offset |

Without `version`, returns a history list (version number descending, **content excluded**) plus the page's current version number; each entry includes `edit_source` (`pipeline` / `agent` / `user` / `revert`), `editor_id`, and `edited_at`.

Revision retention has two tiers: a soft cap of 50 versions only trims `pipeline` and empty-source snapshots, while a hard cap of 200 versions applies to all sources — so manual edits are never purged by the pipeline.

```bash
# History list
curl $BASE/api/v1/knowledgebase/kb-1/wiki/revisions/entity/acme-corp -H "Authorization: Bearer $TOKEN"
# Fetch the full content of version 3
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/revisions/entity/acme-corp?version=3" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/revert

Purpose: Roll a page back to a historical version. Permissions: KB owner or Admin+ + KBAccessWrite.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `slug` | string | Yes | Target page |
| `version` | int | Yes | Target version number (≥ 1) |

Reverting **does not roll the version number back**: the target version's content is written as a new version, with `last_edit_source` recorded as `revert` — so a revert can itself be reverted. Reverting to the current version returns 400 (usually means the frontend's history list is stale).

Response: 200 `WikiPage`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/revert -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"slug":"entity/acme-corp","version":3}'
```

### DELETE /api/v1/knowledgebase/:kb_id/wiki/pages/*slug

Purpose: Delete a page.

Response: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/knowledgebase/kb-1/wiki/pages/overview -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/folders

Purpose: List folders. Query parameters: `parent_id` (empty = root), `page_types` (comma-separated).

Response: 200 `WikiFolderListResponse`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/folders -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/folders

Purpose: Create a folder.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Folder name |
| `parent_id` | string | No | Parent folder |

Response: 201 `WikiFolder`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/folders -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Design Documents"}'
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/folders/:folder_id

Purpose: Rename/move a folder. Request body: `name`, `parent_id`, `move_parent` (bool), all optional.

Response: 200 `WikiFolder`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/folders/f-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Architecture Design"}'
```

### DELETE /api/v1/knowledgebase/:kb_id/wiki/folders/:folder_id

Purpose: Delete a folder.

Response: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/knowledgebase/kb-1/wiki/folders/f-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/index

Purpose: Wiki index page (window grouped by type). Query parameters: `types` (comma-separated), `limit` (1-200, default 50), `cursor` (cursor).

Response: 200 `WikiIndexResponse`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/index -H "Authorization: Bearer $TOKEN"
```

::: warning Removed
`GET /api/v1/knowledgebase/:kb_id/wiki/log` (Wiki change log) was decommissioned along with migration `000077_remove_wiki_log`, and the `wiki_log_entries` table was dropped. Wiki changes are now projected uniformly into the knowledge base activity stream — use `GET /api/v1/knowledge-bases/:id/activity` instead.
:::

### GET /api/v1/knowledgebase/:kb_id/wiki/graph

Purpose: Page relationship graph.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `mode` | string | No | `overview` (default) / `ego` |
| `center` | string | No | Ego-mode center slug (required in ego mode) |
| `depth` | int | No | 1-3, default 1 |
| `types` | string | No | page_type filter |
| `limit` | int | No | Defaults to 500, capped at 2000 |

Response: 200 `WikiGraphData`

```bash
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/graph?mode=overview" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/stats

Purpose: Wiki statistics.

Response: 200 `WikiStats`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/stats -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/search

Purpose: Page search. Query parameters: `q` (required), `limit` (default 10).

Response: 200 `{"pages":[WikiPage]}`

```bash
curl "$BASE/api/v1/knowledgebase/kb-1/wiki/search?q=deploy" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/rebuild-links

Purpose: Rebuild page cross-links. Write permission. No request body.

Response: 200 `{"message":"Links rebuilt successfully"}`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/rebuild-links -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/lint

Purpose: Wiki consistency check report.

Response: 200 `WikiLintReport`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/lint -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/knowledgebase/:kb_id/wiki/auto-fix

Purpose: Auto-fix lint issues. Write permission. No request body.

Response: 200 `{"fixed":N,"message":"Auto-fixed N issues"}`

```bash
curl -X POST $BASE/api/v1/knowledgebase/kb-1/wiki/auto-fix -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledgebase/:kb_id/wiki/issues

Purpose: List issues. Query parameters: `slug` (filter by page), `status` (`pending/ignored/resolved`).

Response: 200 `[WikiPageIssue]`

```bash
curl $BASE/api/v1/knowledgebase/kb-1/wiki/issues -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledgebase/:kb_id/wiki/issues/:issue_id/status

Purpose: Update issue status. Write permission. Request body: `{"status":"pending|ignored|resolved"}` (`binding:"required"`). `issue_id` must belong to the knowledge base in the path, otherwise 404.

Response: 200 `{"message":"Issue status updated successfully"}`

```bash
curl -X PUT $BASE/api/v1/knowledgebase/kb-1/wiki/issues/i-1/status -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"status":"resolved"}'
```

## Implementation Reference

Route registration: `RegisterFAQRoutes` and `RegisterWikiPageRoutes` in `internal/router/routes_knowledge.go`. Handlers: `internal/handler/faq.go`, `internal/handler/wiki_page.go`.
