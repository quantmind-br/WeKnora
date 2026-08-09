# FAQ Management API

[Back to Table of Contents](./README.md)

The FAQ interface is divided into two groups:

- `/knowledge-bases/:id/faq/*`: CRUD, batch operations, search, and import/export for FAQ entries within a knowledge base.
- `/faq/import/progress/:task_id`: **Not part of the knowledge base group** — used to query the progress of asynchronous import/dry-run tasks; only the task ID is needed to call it.

| Method | Path                                                              | Description                              |
| ------ | ----------------------------------------------------------------- | ----------------------------------------- |
| GET    | `/knowledge-bases/:id/faq/entries`                                | Get the list of FAQ entries               |
| GET    | `/knowledge-bases/:id/faq/entries/export`                         | Export FAQ entries (CSV)                  |
| GET    | `/knowledge-bases/:id/faq/entries/:entry_id`                      | Get a single FAQ entry (by seq_id)        |
| POST   | `/knowledge-bases/:id/faq/entries`                                | Batch upsert FAQ entries (async)          |
| POST   | `/knowledge-bases/:id/faq/entry`                                  | Synchronously create a single FAQ entry   |
| PUT    | `/knowledge-bases/:id/faq/entries/:entry_id`                      | Update a single FAQ entry                 |
| POST   | `/knowledge-bases/:id/faq/entries/:entry_id/similar-questions`    | Append similar questions to a FAQ entry   |
| PUT    | `/knowledge-bases/:id/faq/entries/fields`                         | Batch update fields (enabled/recommended/tags) |
| PUT    | `/knowledge-bases/:id/faq/entries/tags`                           | Batch update tags                         |
| DELETE | `/knowledge-bases/:id/faq/entries`                                | Batch delete FAQ entries                  |
| POST   | `/knowledge-bases/:id/faq/search`                                 | FAQ hybrid search                         |
| PUT    | `/knowledge-bases/:id/faq/import/last-result/display`             | Update the display state of the last import result card |
| GET    | `/faq/import/progress/:task_id`                                   | Query FAQ import task progress (public)   |

> **Path parameter note**: `:entry_id` is always the FAQ entry's `seq_id` (integer), not the string-form ID. Likewise, the `by_id` / `by_tag` / `exclude_ids` / `ids` fields in batch interfaces are all `seq_id` lists (integers).

## GET `/knowledge-bases/:id/faq/entries` - Get the list of FAQ entries

Supports pagination, filtering by tag, keyword search, and sorting.

**Query parameters**:

| Parameter    | Type   | Required | Description                                                                                     |
| ------------ | ------ | -------- | ------------------------------------------------------------------------------------------------ |
| page         | int    | No       | Page number, default 1                                                                           |
| page_size    | int    | No       | Items per page, default 20                                                                       |
| tag_id       | int    | No       | Filter by tag `seq_id`                                                                           |
| keyword      | string | No       | Keyword search                                                                                   |
| search_field | string | No       | Search field: `standard_question` / `similar_questions` / `answers`; if left blank, all fields are searched |
| sort_order   | string | No       | Sort order; `asc` sorts by update time ascending, defaults to descending by update time           |

**Request**:

```curl
# Search all fields
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries?page=1&page_size=10&keyword=password' \
--header 'X-API-Key: sk-xxxxx'

# Search only the standard question
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries?keyword=password&search_field=standard_question' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "total": 100,
        "page": 1,
        "page_size": 10,
        "data": [
            {
                "id": 1,
                "chunk_id": "chunk-00000001",
                "knowledge_id": "knowledge-00000001",
                "knowledge_base_id": "kb-00000001",
                "tag_id": 12,
                "tag_name": "Account",
                "is_enabled": true,
                "is_recommended": false,
                "standard_question": "How do I reset my password?",
                "similar_questions": ["What if I forgot my password", "Password recovery"],
                "negative_questions": ["How do I change my username"],
                "answers": ["You can reset your password by clicking the 'Forgot password' link on the login page."],
                "answer_strategy": "all",
                "index_mode": "hybrid",
                "chunk_type": "faq",
                "created_at": "2025-08-12T10:00:00+08:00",
                "updated_at": "2025-08-12T10:00:00+08:00"
            }
        ]
    },
    "success": true
}
```

## GET `/knowledge-bases/:id/faq/entries/export` - Export FAQ entries

Exports all FAQ entries under a knowledge base as CSV (UTF-8 with BOM, Excel-compatible).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/export' \
--header 'X-API-Key: sk-xxxxx' \
--output faq_export.csv
```

**Response**: `Content-Type: text/csv; charset=utf-8`, with the filename `faq_export.csv` attached.

## GET `/knowledge-bases/:id/faq/entries/:entry_id` - Get a single FAQ entry

Retrieves the details of a single FAQ entry by `seq_id` (integer).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/1' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "id": 1,
        "chunk_id": "chunk-00000001",
        "knowledge_id": "knowledge-00000001",
        "knowledge_base_id": "kb-00000001",
        "tag_id": 12,
        "tag_name": "Account",
        "is_enabled": true,
        "is_recommended": false,
        "standard_question": "How do I reset my password?",
        "similar_questions": ["What if I forgot my password", "Password recovery"],
        "negative_questions": [],
        "answers": ["You can reset your password by clicking the 'Forgot password' link on the login page."],
        "answer_strategy": "all",
        "index_mode": "hybrid",
        "chunk_type": "faq",
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

## POST `/knowledge-bases/:id/faq/entries` - Batch upsert FAQ entries (async)

**Asynchronously** batch-imports or updates FAQ entries. The interface returns a `task_id` immediately; the caller must query progress and results via `GET /faq/import/progress/:task_id`.

Supports `dry_run=true`: runs asynchronous validation only (format / in-batch duplicates / duplicates against existing entries / content safety) without actually writing data.

**Request body (`types.FAQBatchUpsertPayload`)**:

| Field        | Type                       | Required | Description                                                                       |
| ------------ | -------------------------- | -------- | ----------------------------------------------------------------------------------- |
| entries      | `[]FAQEntryPayload`        | Yes      | Array of FAQ entries                                                              |
| mode         | string                     | Yes      | `append` or `replace` (replace clears existing entries)                            |
| knowledge_id | string                     | No       | Associated FAQ Knowledge ID (if omitted, the knowledge base's default FAQ knowledge is used) |
| task_id      | string                     | No       | Task ID; a UUID is auto-generated if omitted                                       |
| dry_run      | boolean                    | No       | Validate only, without importing                                                   |

`FAQEntryPayload` fields:

| Field               | Type      | Required | Description                                                             |
| ------------------- | --------- | -------- | ------------------------------------------------------------------------ |
| id                  | int64     | No       | Specifies the `seq_id` (for data migration scenarios; must be less than the auto-increment starting value of 100000000) |
| standard_question   | string    | Yes      | Standard question                                                       |
| similar_questions   | string[]  | No       | List of similar questions                                               |
| negative_questions  | string[]  | No       | List of negative example questions                                      |
| answers             | string[]  | No       | List of answers                                                         |
| answer_strategy     | string    | No       | Answer return strategy: `all` or `random`                                |
| tag_id              | int64     | No       | Tag `seq_id`                                                            |
| tag_name            | string    | No       | Tag name (used to match a tag by name)                                  |
| is_enabled          | boolean   | No       | Whether it's enabled                                                     |
| is_recommended      | boolean   | No       | Whether it's recommended                                                 |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "mode": "append",
    "entries": [
        {
            "standard_question": "How do I contact support?",
            "similar_questions": ["Support phone", "Online support"],
            "answers": ["You can reach support by calling 400-xxx-xxxx."],
            "tag_id": 1
        },
        {
            "standard_question": "What is the refund policy?",
            "answers": ["We offer a 7-day no-questions-asked refund."]
        }
    ]
}'
```

**Response**:

```json
{
    "data": { "task_id": "task-00000001" },
    "success": true
}
```

> Use `GET /faq/import/progress/:task_id` to query the task's final status.

## POST `/knowledge-bases/:id/faq/entry` - Synchronously create a single FAQ entry

**Synchronously** creates a single FAQ entry, immediately validating the standard question/similar questions against existing entries for duplicates.

**Request body**: Same `FAQEntryPayload` as above (`standard_question` required).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entry' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "standard_question": "How do I contact support?",
    "similar_questions": ["Support phone", "Online support"],
    "answers": ["You can reach support by calling 400-xxx-xxxx."],
    "tag_id": 1,
    "is_enabled": true
}'
```

**Response**:

```json
{
    "data": {
        "id": 1,
        "chunk_id": "chunk-00000001",
        "knowledge_id": "knowledge-00000001",
        "knowledge_base_id": "kb-00000001",
        "tag_id": 1,
        "tag_name": "Support",
        "is_enabled": true,
        "is_recommended": false,
        "standard_question": "How do I contact support?",
        "similar_questions": ["Support phone", "Online support"],
        "negative_questions": [],
        "answers": ["You can reach support by calling 400-xxx-xxxx."],
        "answer_strategy": "all",
        "index_mode": "hybrid",
        "chunk_type": "faq",
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

**Error response** (when the standard question or a similar question is a duplicate):

```json
{
    "success": false,
    "error": {
        "code": "BAD_REQUEST",
        "message": "The standard question duplicates an existing FAQ"
    }
}
```

## PUT `/knowledge-bases/:id/faq/entries/:entry_id` - Update a single FAQ entry

Updates a single FAQ entry by `seq_id`; the request body is the same as `FAQEntryPayload`.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/1' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "standard_question": "How do I reset my account password?",
    "similar_questions": ["What if I forgot my password", "Password recovery", "Reset password"],
    "answers": ["You can reset your password as follows: 1. Click \"Forgot password\" on the login page 2. Enter your registered email 3. Check for the reset email"],
    "is_enabled": true
}'
```

**Response**: Returns the updated FAQ entry, with the same structure as the create interface.

## POST `/knowledge-bases/:id/faq/entries/:entry_id/similar-questions` - Append similar questions

Appends similar questions to the specified FAQ entry (`seq_id`).

**Request body**:

| Field              | Type     | Required | Description                        |
| ----------------- | -------- | -------- | ------------------------------------ |
| similar_questions | string[] | Yes      | Array of similar questions to append |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/1/similar-questions' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "similar_questions": ["How to change password", "Password reset methods"]
}'
```

**Response**: Returns the complete FAQ entry after the append.

## PUT `/knowledge-bases/:id/faq/entries/fields` - Batch update fields

A **unified** batch field-update interface that supports updating `is_enabled` / `is_recommended` / `tag_id` simultaneously, and supports two scopes:

- **By entry ID** (`by_id`): the key is the entry's `seq_id`, the value is the fields to update for that entry.
- **By tag ID** (`by_tag`): the key is the tag's `seq_id`, applying the same field update to all entries under that tag; can be combined with `exclude_ids` to exclude specific entries.

At least one of `by_id` or `by_tag` must be provided; both can be used together.

**Request body (`types.FAQEntryFieldsBatchUpdate`)**:

| Field       | Type                              | Required | Description                                     |
| ----------- | ---------------------------------- | -------- | ------------------------------------------------- |
| by_id       | `map[int64]FAQEntryFieldsUpdate`  | No       | Update by entry `seq_id`                          |
| by_tag      | `map[int64]FAQEntryFieldsUpdate`  | No       | Update all entries under a tag by tag `seq_id`     |
| exclude_ids | `int64[]`                         | No       | Used with `by_tag` to exclude specific entry `seq_id`s |

`FAQEntryFieldsUpdate` fields (all optional; only the fields provided are updated):

| Field          | Type    | Description        |
| -------------- | ------- | -------------------- |
| is_enabled     | boolean | Whether it's enabled |
| is_recommended | boolean | Whether it's recommended |
| tag_id         | int64   | Tag `seq_id`        |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/fields' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "by_id": {
        "1": {"is_enabled": true, "is_recommended": false},
        "2": {"is_enabled": false}
    },
    "by_tag": {
        "100": {"is_recommended": true}
    },
    "exclude_ids": [3, 4]
}'
```

**Response**:

```json
{ "success": true }
```

## PUT `/knowledge-bases/:id/faq/entries/tags` - Batch update tags

Updates only the tag association. The key is the entry's `seq_id`; the value is the target tag's `seq_id`; a value of `null` clears the tag.

**Request body**:

| Field   | Type                  | Required | Description                                        |
| ------- | ---------------------- | -------- | ---------------------------------------------------- |
| updates | `map[int64]int64?`    | Yes      | Key: entry `seq_id`; Value: tag `seq_id` or `null`   |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries/tags' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "updates": {
        "1": 10,
        "2": 11,
        "3": null
    }
}'
```

**Response**:

```json
{ "success": true }
```

## DELETE `/knowledge-bases/:id/faq/entries` - Batch delete

**Request body**:

| Field | Type      | Required | Description                              |
| ---- | --------- | -------- | ------------------------------------------ |
| ids  | `int64[]` | Yes      | List of FAQ entry `seq_id`s to delete       |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/entries' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "ids": [1, 2, 3]
}'
```

**Response**:

```json
{ "success": true }
```

## POST `/knowledge-bases/:id/faq/search` - FAQ hybrid search

Vector + keyword hybrid retrieval, supporting two-tier priority tag recall.

**Request body (`types.FAQSearchRequest`)**:

| Field                   | Type      | Required | Description                                                                       |
| ----------------------- | --------- | -------- | ------------------------------------------------------------------------------------ |
| query_text              | string    | Yes      | Search text                                                                         |
| vector_threshold        | float     | No       | Vector similarity threshold (0–1)                                                    |
| match_count             | int       | No       | Number of results to return, default 10, max 200                                     |
| first_priority_tag_ids  | `int64[]` | No       | List of first-priority tag `seq_id`s (highest-priority recall scope)                 |
| second_priority_tag_ids | `int64[]` | No       | List of second-priority tag `seq_id`s                                                |
| only_recommended        | boolean   | No       | Whether to return only entries with `is_recommended=true`                            |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/search' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query_text": "How to reset password",
    "vector_threshold": 0.5,
    "match_count": 10,
    "first_priority_tag_ids": [12],
    "only_recommended": false
}'
```

**Response**:

```json
{
    "data": [
        {
            "id": 1,
            "chunk_id": "chunk-00000001",
            "knowledge_id": "knowledge-00000001",
            "knowledge_base_id": "kb-00000001",
            "tag_id": 12,
            "tag_name": "Account",
            "is_enabled": true,
            "is_recommended": false,
            "standard_question": "How do I reset my password?",
            "similar_questions": ["What if I forgot my password", "Password recovery"],
            "answers": ["You can reset your password by clicking the 'Forgot password' link on the login page."],
            "answer_strategy": "all",
            "chunk_type": "faq",
            "score": 0.95,
            "match_type": "vector",
            "matched_question": "What if I forgot my password",
            "created_at": "2025-08-12T10:00:00+08:00",
            "updated_at": "2025-08-12T10:00:00+08:00"
        }
    ],
    "success": true
}
```

## PUT `/knowledge-bases/:id/faq/import/last-result/display` - Update last import result display state

Controls the display/hide state of the frontend result card after the last import completes.

**Request body**:

| Field          | Type   | Required | Description             |
| -------------- | ------ | -------- | ------------------------ |
| display_status | string | Yes      | `open` or `close`        |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/faq/import/last-result/display' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "display_status": "close"
}'
```

**Response**:

```json
{ "success": true }
```

## GET `/faq/import/progress/:task_id` - Query FAQ import progress

> **Note**: This interface is **not** under the `/knowledge-bases/:id/faq` group — the path starts directly with `/faq/import/progress/:task_id`. The task ID is returned by `POST /knowledge-bases/:id/faq/entries`.

**Path parameters**:

| Parameter | Type   | Description        |
| ------- | ------ | -------------------- |
| task_id | string | The ID of the import task |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/faq/import/progress/task-00000001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response** (excerpt of key fields):

```json
{
    "data": {
        "task_id": "task-00000001",
        "kb_id": "kb-00000001",
        "knowledge_id": "knowledge-00000001",
        "status": "completed",
        "progress": 100,
        "total": 100,
        "processed": 100,
        "success_count": 95,
        "failed_count": 3,
        "partial_failed_count": 2,
        "skipped_count": 0,
        "failed_entries": [
            {
                "index": 5,
                "reason": "duplicates an existing FAQ",
                "standard_question": "Duplicated question"
            }
        ],
        "success_entries": [
            { "index": 0, "seq_id": 101, "standard_question": "How do I contact support?" }
        ],
        "message": "",
        "error": "",
        "created_at": 1736582400,
        "updated_at": 1736582460,
        "dry_run": false,
        "import_mode": "append",
        "imported_at": "2025-08-12T10:01:00+08:00",
        "display_status": "open",
        "processing_time": 60000
    },
    "success": true
}
```

Possible values for `status`: `pending` / `processing` / `completed` / `failed`.

When there are too many failed entries, `failed_entries` may not be returned directly, and instead a CSV download URL is provided via `failed_entries_url`.

Tasks in `dry_run=true` mode are also queried via this interface; the `seq_id` in `success_entries` will not actually be written.
