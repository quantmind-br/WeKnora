# Knowledge Management API

[Back to Table of Contents](./README.md)

Knowledge is a searchable piece of content under a knowledge base (from a file, a URL, or manually entered Markdown). This document covers the endpoints for creating, querying, updating, deleting, and migrating knowledge, as well as file preview/download.

| Method | Path                                       | Description                                       |
| ------ | ------------------------------------------ | ------------------------------------------------- |
| POST   | `/knowledge-bases/:id/knowledge/file`      | Upload a file to create knowledge (multipart)      |
| POST   | `/knowledge-bases/:id/knowledge/url`       | Create knowledge from a URL (web scraping or file download) |
| POST   | `/knowledge-bases/:id/knowledge/manual`    | Create manual Markdown knowledge                   |
| GET    | `/knowledge-bases/:id/knowledge`           | List knowledge under a knowledge base (pagination/filtering supported) |
| GET    | `/knowledge-bases/:id/knowledge/folders`   | Get the knowledge base's folder directory tree     |
| PUT    | `/knowledge-bases/:id/knowledge/folders`   | Rename or move a folder (including subdirectories) |
| DELETE | `/knowledge-bases/:id/knowledge`           | Clear all knowledge under a knowledge base (async task) |
| GET    | `/knowledge/batch`                         | Batch-fetch knowledge by ID list                   |
| GET    | `/knowledge/:id`                           | Get knowledge details                              |
| PUT    | `/knowledge/:id`                           | Update knowledge (title/description/tags, etc.)    |
| DELETE | `/knowledge/:id`                           | Delete a single piece of knowledge                 |
| PUT    | `/knowledge/manual/:id`                    | Update manual Markdown knowledge                   |
| POST   | `/knowledge/:id/reparse`                   | Re-parse knowledge (async)                         |
| POST   | `/knowledge/:id/cancel-parse`              | Cancel an in-progress parsing task                 |
| GET    | `/knowledge/:id/download`                  | Download the original file (attachment)            |
| GET    | `/knowledge/:id/preview`                   | Inline file preview (Content-Type set by extension) |
| PUT    | `/knowledge/image/:id/:chunk_id`           | Update chunk image information                     |
| PUT    | `/knowledge/tags`                          | Batch-update knowledge tags                        |
| GET    | `/knowledge/search`                        | Search/filter knowledge across knowledge bases      |
| POST   | `/knowledge/batch-reparse`                 | Batch re-parse knowledge within the same knowledge base (async task) |
| POST   | `/knowledge/batch-delete`                  | Batch delete knowledge within the same knowledge base (async task) |
| POST   | `/knowledge/folder`                        | Batch move knowledge into a target folder (reclassification only) |
| POST   | `/knowledge/move`                          | Migrate knowledge to another knowledge base (async task) |
| GET    | `/knowledge/move/progress/:task_id`        | Query the progress of a knowledge migration task    |

> **General notes**:
> - The `:id` in a path under the knowledge base route is the **knowledge base ID**; the `:id` in `/knowledge/:id` is the **knowledge ID**.
> - All write operations (create, update, delete, migrate, re-parse, cancel-parse) require the current user to have `editor` or `admin` permission within the organization the knowledge base belongs to; clearing a knowledge base's contents can only be performed by the KB **owner** (admin with a matching space).
> - Key status fields: `parse_status` takes values `pending` / `processing` / `finalizing` / `completed` / `failed` / `cancelled`; `enable_status` takes values `enabled` / `disabled`.
> - `processing` refers to the DocReader / chunking / vectorization stages; `finalizing` means the main parsing has completed but index optimization tasks such as summarization / question generation / graph extraction are still running; only once all subtasks reach a terminal state does it move to `completed`.
> - `cancelled` means parsing was actively cancelled by the user, and can be re-triggered via `reparse`. All three states `pending` / `processing` / `finalizing` can be terminated via `cancel-parse`.

## POST `/knowledge-bases/:id/knowledge/file` - Upload a file to create knowledge

Creates a knowledge entry by uploading a file via `multipart/form-data`. File size is limited by the `MAX_FILE_SIZE_MB` environment variable.

**Path parameters**:

| Field | Type   | Description      |
| ---- | ------ | --------- |
| id   | string | Knowledge base ID |

**Form fields**:

| Field                | Type    | Required | Description                                                                 |
| ------------------- | ------- | ---- | -------------------------------------------------------------------- |
| `file`              | file    | Yes   | The file to upload                                                         |
| `fileName`          | string  | No   | Custom file name, used to preserve the relative path (e.g. `docs/intro.md`) during "folder upload" |
| `metadata`          | string  | No   | JSON string, deserialized into `map[string]string`                     |
| `enable_multimodel` | string  | No   | `"true"` / `"false"`, whether to enable multimodal image-text parsing                        |
| `process_config`    | string  | No   | JSON string, per-batch parsing configuration overrides (`KnowledgeProcessOverrides`); written to `knowledge.metadata.process_overrides`. If omitted, behavior matches the current production defaults |
| `tag_id`            | string  | No   | Tag ID; pass `__untagged__` or an empty string to indicate "untagged"                      |
| `channel`           | string  | No   | Source channel identifier (written to the `channel` field, defaults to `web`)                      |

Optional fields in `process_config` include: `parser_engine_rules`, `chunking_config`, `enable_multimodel`, `vlm_config`, `asr_config`, `question_generation_config`, `graph_enabled`, `extract_config`. If both `enable_multimodel` and `process_config.enable_multimodel` are passed, `process_config` takes precedence.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/file' \
--header 'X-API-Key: sk-xxxxx' \
--form 'file=@"/Users/xxxx/tests/彗星.txt"' \
--form 'enable_multimodel="true"' \
--form 'tag_id="tag-00000001"' \
--form 'metadata="{\"source\":\"manual_upload\"}"'
```

> Note: when using `-F`/`--form`, curl automatically sets `Content-Type: multipart/form-data; boundary=...` — don't manually add `--header 'Content-Type: application/json'`, or the request body will be parsed incorrectly.

**Response** (creation successful, `parse_status=processing` indicates the parsing task has been queued):

```json
{
    "data": {
        "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "file",
        "title": "彗星.txt",
        "description": "",
        "source": "",
        "channel": "web",
        "tag_id": "tag-00000001",
        "summary_status": "none",
        "parse_status": "processing",
        "enable_status": "disabled",
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "file_name": "彗星.txt",
        "file_type": "txt",
        "file_size": 7710,
        "file_hash": "d69476ddbba45223a5e97e786539952c",
        "file_path": "data/files/1/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/1754970756171067621.txt",
        "storage_size": 0,
        "metadata": null,
        "created_at": "2025-08-12T11:52:36.168632288+08:00",
        "updated_at": "2025-08-12T11:52:36.173612121+08:00",
        "processed_at": null,
        "error_message": "",
        "deleted_at": null
    },
    "success": true
}
```

If the file is a duplicate, a reference to the existing knowledge entry is returned with HTTP 409; if it exceeds the size limit, HTTP 400 is returned with `File size cannot exceed N MB`.

## POST `/knowledge-bases/:id/knowledge/url` - Create knowledge from a URL

Can create either **web page knowledge** or **remote file knowledge**. The backend determines which automatically based on the following rules:

- If either `file_name` / `file_type` is explicitly provided, or the URL path contains a known file extension, it is handled in "file download mode" (the remote file is fetched and saved);
- Otherwise it is handled in "web scraping mode".

The URL undergoes SSRF safety validation and is prohibited from pointing to internal/loopback addresses.

**Request body**:

| Field                | Type    | Required | Description                                              |
| ------------------- | ------- | ---- | -------------------------------------------------- |
| `url`               | string  | Yes   | Target URL                                          |
| `file_name`         | string  | No   | Explicitly specify the file name, forcing file download mode                |
| `file_type`         | string  | No   | Explicitly specify the file type (e.g. `pdf`, `docx`)              |
| `enable_multimodel` | boolean | No   | Whether to enable multimodal parsing                                |
| `title`             | string  | No   | Custom title                                        |
| `tag_id`            | string  | No   | Tag ID                                           |
| `channel`           | string  | No   | Source channel identifier                                      |

**Request (web page mode)**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/url' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "url": "https://github.com/Tencent/WeKnora",
    "enable_multimodel": true
}'
```

**Request (remote file mode)**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/url' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "url": "https://example.com/papers/whitepaper.pdf",
    "file_name": "whitepaper.pdf",
    "file_type": "pdf"
}'
```

**Response** (HTTP 201):

```json
{
    "data": {
        "id": "9c8af585-ae15-44ce-8f73-45ad18394651",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "url",
        "title": "",
        "description": "",
        "source": "https://github.com/Tencent/WeKnora",
        "channel": "web",
        "tag_id": "",
        "summary_status": "none",
        "parse_status": "processing",
        "enable_status": "disabled",
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "file_name": "",
        "file_type": "",
        "file_size": 0,
        "file_hash": "",
        "file_path": "",
        "storage_size": 0,
        "metadata": null,
        "created_at": "2025-08-12T11:55:05.709266776+08:00",
        "updated_at": "2025-08-12T11:55:05.712918234+08:00",
        "processed_at": null,
        "error_message": "",
        "deleted_at": null
    },
    "success": true
}
```

## POST `/knowledge-bases/:id/knowledge/manual` - Create manual Markdown knowledge

Suitable for scenarios where Markdown content is written directly (no source file).

**Request body**:

| Field      | Type   | Required | Description                                                |
| --------- | ------ | ---- | --------------------------------------------------- |
| `title`   | string | Yes   | Title                                                |
| `content` | string | Yes   | Markdown body                                                       |
| `status`  | string | No   | Business status such as draft/published (draft does not trigger parsing)             |
| `tag_id`  | string | No   | Tag ID                                             |
| `channel` | string | No   | Source channel identifier                                        |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge/manual' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "title": "产品使用指南",
    "content": "# 产品使用指南\n\n## 快速入门\n\n这是一份产品使用指南...",
    "status": "published",
    "tag_id": "tag-00000001"
}'
```

**Response**:

```json
{
    "data": {
        "id": "5a3b2c1d-0e9f-4a8b-7c6d-5e4f3a2b1c0d",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "manual",
        "title": "产品使用指南",
        "description": "",
        "source": "",
        "channel": "web",
        "tag_id": "tag-00000001",
        "summary_status": "none",
        "parse_status": "processing",
        "enable_status": "disabled",
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "file_name": "",
        "file_type": "md",
        "file_size": 0,
        "file_hash": "",
        "file_path": "",
        "storage_size": 0,
        "metadata": null,
        "created_at": "2025-08-12T12:00:00.000000+08:00",
        "updated_at": "2025-08-12T12:00:00.000000+08:00",
        "processed_at": null,
        "error_message": "",
        "deleted_at": null
    },
    "success": true
}
```

## GET `/knowledge-bases/:id/knowledge` - List knowledge under a knowledge base

Supports pagination and filtering by tag/keyword/file type.

**Path parameters**:

| Field | Type   | Description      |
| ---- | ------ | --------- |
| id   | string | Knowledge base ID |

**Query parameters**:

| Field           | Type    | Default | Description                                                                                                |
| -------------- | ------- | ---- | ----------------------------------------------------------------------------------------------------------------- |
| `page`         | integer | 1    | Page number (starting from 1)                                                                                   |
| `page_size`    | integer | 20   | Items per page                                                                                            |
| `tag_id`       | string  | -    | Filter by tag ID                                                                                      |
| `keyword`      | string  | -    | Filter by title/content keyword                                                                               |
| `file_type`    | string  | -    | Filter by a single file extension (e.g. `pdf`); the special values `manual` / `url` match the `type` column                              |
| `parse_status` | string  | -    | Filter by parse status: `pending` / `processing` / `completed` / `failed`                                    |
| `source`       | string  | -    | Filter by source/channel: `web` / `api` / `browser_extension` / `feishu` / `notion` / `yuque` / `wechat`, etc.; the special values `manual` / `url` match the `type` column |
| `start_time`   | string  | -    | Update time start, accepts RFC3339 (`2024-05-01T00:00:00+08:00`) or `YYYY-MM-DD HH:MM:SS` / `YYYY-MM-DD`     |
| `end_time`     | string  | -    | Update time end, same format as `start_time`                                                                   |
| `folder_path`  | string  | -    | Filter by folder path; **folder-based filtering is enabled only when this parameter is passed**. An empty string means the knowledge base root directory (excluding documents in subfolders); if omitted, all documents across all folders are listed (flat view) |
| `folder_recursive` | bool | false | When `true`, also returns documents in subdirectories of `folder_path`; only takes effect when `folder_path` is passed |

> **Folder filtering semantics**: whether `folder_path` appears in the query determines the list mode — an empty string alone cannot distinguish "root directory" from "no folder filtering". Integrators who need to browse a specific folder should explicitly pass `folder_path` (pass `folder_path=` for the root directory); to get a flat list of the entire knowledge base, omit this parameter.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge?page=1&page_size=1&tag_id=tag-00000001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "id": "9c8af585-ae15-44ce-8f73-45ad18394651",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "type": "url",
            "title": "",
            "description": "",
            "source": "https://github.com/Tencent/WeKnora",
            "channel": "web",
            "tag_id": "tag-00000001",
            "summary_status": "none",
            "parse_status": "pending",
            "enable_status": "disabled",
            "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
            "file_name": "",
            "folder_path": "",
            "file_type": "",
            "file_size": 0,
            "file_hash": "",
            "file_path": "",
            "storage_size": 0,
            "metadata": null,
            "created_at": "2025-08-12T11:55:05.709266+08:00",
            "updated_at": "2025-08-12T11:55:05.709266+08:00",
            "processed_at": null,
            "error_message": "",
            "deleted_at": null
        }
    ],
    "page": 1,
    "page_size": 1,
    "total": 2,
    "success": true
}
```

## GET `/knowledge-bases/:id/knowledge/folders` - Get the folder directory tree

Returns a directory tree aggregated from `folder_path`, including the direct document count and the total count (including subdirectories) for each folder. Read-only, with the same permissions as listing knowledge (Viewer+ with read access to the KB).

**Response**:

```json
{
    "success": true,
    "data": {
        "root_document_count": 2,
        "total_document_count": 10,
        "folders": [
            {
                "path": "docs",
                "name": "docs",
                "document_count": 0,
                "total_count": 4,
                "children": [
                    {
                        "path": "docs/spec",
                        "name": "spec",
                        "document_count": 3,
                        "total_count": 4,
                        "children": []
                    }
                ]
            }
        ]
    }
}
```

## PUT `/knowledge-bases/:id/knowledge/folders` - Rename or move a folder

Changes a folder and all of its subdirectories to a new path. If the target path already exists, the two folders are merged; a folder cannot be moved into its own subdirectory. Requires the KB **creator** or Admin+, with write access to the KB.

**Request body**:

| Field   | Type   | Required | Description                         |
| ------ | ------ | ---- | ----------------------------- |
| `from` | string | Yes   | Source folder path (cannot be empty)     |
| `to`   | string | Yes   | Target folder path (cannot be empty)   |

**Response**:

```json
{
    "success": true,
    "data": {
        "moved_count": 3,
        "folder_path": "handbook"
    }
}
```

A `moved_count` of 0 means the source folder does not exist, or the operation was already a no-op.

## POST `/knowledge/folder` - Batch move knowledge into a folder

Bulk-modifies the `folder_path` of knowledge entries — this only adjusts classification and does **not** trigger re-parsing, chunking, or vectorization. If the target path doesn't exist, it is created automatically; an empty path means moving back to the knowledge base root. Requires editor/admin, and the caller must be the KB creator or Admin+.

**Request body**:

| Field            | Type     | Required | Description                                      |
| --------------- | -------- | ---- | ----------------------------------------- |
| `kb_id`         | string   | Yes   | Knowledge base ID                                 |
| `knowledge_ids` | string[] | Yes   | List of knowledge IDs (up to 200)               |
| `folder_path`   | string   | No   | Target folder path; omitted or an empty string means the root directory |

**Response**:

```json
{
    "success": true,
    "data": {
        "moved_count": 2,
        "folder_path": "archive/2026"
    }
}
```

## DELETE `/knowledge-bases/:id/knowledge` - Clear all knowledge under a knowledge base

Asynchronously submits a "clear task" that deletes all knowledge entries under the knowledge base; the knowledge base itself is retained. **Only the KB owner (admin with a matching space) can perform this operation**.

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/knowledge' \
--header 'X-API-Key: sk-xxxxx'
```

**Response** (queued):

```json
{
    "success": true,
    "message": "Knowledge base contents clear task submitted",
    "data": { "deleted_count": 42 }
}
```

When the knowledge base is already empty:

```json
{
    "success": true,
    "message": "Knowledge base is already empty",
    "data": { "deleted_count": 0 }
}
```

## GET `/knowledge/batch` - Batch-fetch knowledge

Fetches multiple knowledge details at once by ID list, commonly used to restore a selected list after refreshing a page.

**Query parameters**:

| Field        | Type     | Required | Description                                                                  |
| ----------- | -------- | ---- | ----------------------------------------------------------------------------- |
| `ids`       | string[] | Yes   | Knowledge IDs; pass multiple by repeating `ids=...`                                        |
| `kb_id`     | string   | No   | Restricts the scope to a knowledge base; used in shared knowledge base scenarios to validate permissions by KB and resolve the effective space       |
| `agent_id`  | string   | No   | Shared Agent ID; fetches based on the Agent's space, commonly used to backfill files after a refresh in shared scenarios   |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/batch?ids=9c8af585-ae15-44ce-8f73-45ad18394651&ids=4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "id": "9c8af585-ae15-44ce-8f73-45ad18394651",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "type": "url",
            "title": "",
            "source": "https://github.com/Tencent/WeKnora",
            "parse_status": "pending",
            "enable_status": "disabled",
            "created_at": "2025-08-12T11:55:05.709266+08:00",
            "updated_at": "2025-08-12T11:55:05.709266+08:00"
        },
        {
            "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "type": "file",
            "title": "彗星.txt",
            "file_name": "彗星.txt",
            "file_type": "txt",
            "file_size": 7710,
            "parse_status": "completed",
            "enable_status": "enabled",
            "created_at": "2025-08-12T11:52:36.168632+08:00",
            "updated_at": "2025-08-12T11:52:53.376871+08:00"
        }
    ],
    "success": true
}
```

> The fields in the response above omit empty values and some large fields; the actual response is the complete `Knowledge` object.

## GET `/knowledge/:id` - Get knowledge details

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "file",
        "title": "彗星.txt",
        "description": "彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会形成彗发和彗尾。",
        "source": "",
        "channel": "web",
        "tag_id": "tag-00000001",
        "summary_status": "completed",
        "parse_status": "completed",
        "enable_status": "enabled",
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "file_name": "彗星.txt",
        "file_type": "txt",
        "file_size": 7710,
        "file_hash": "d69476ddbba45223a5e97e786539952c",
        "file_path": "data/files/1/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/1754970756171067621.txt",
        "storage_size": 33689,
        "metadata": null,
        "created_at": "2025-08-12T11:52:36.168632+08:00",
        "updated_at": "2025-08-12T11:52:53.376871+08:00",
        "processed_at": "2025-08-12T11:52:53.376573+08:00",
        "error_message": "",
        "deleted_at": null
    },
    "success": true
}
```

## PUT `/knowledge/:id` - Update knowledge

Updates the metadata of a knowledge entry (title/description/tags, etc.). The request body is a `Knowledge` struct, but only fields whitelisted by the service side are actually updated.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "title": "彗星 - 天文百科",
    "description": "彗星条目，已校对",
    "tag_id": "tag-00000001",
    "enable_status": "enabled"
}'
```

**Response**:

```json
{
    "success": true,
    "message": "Knowledge chunk updated successfully"
}
```

## DELETE `/knowledge/:id` - Delete a single piece of knowledge

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge/9c8af585-ae15-44ce-8f73-45ad18394651' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "message": "Deleted successfully"
}
```

## PUT `/knowledge/manual/:id` - Update manual Markdown knowledge

**Request body**: Same as `POST /knowledge-bases/:id/knowledge/manual`, with all fields optional.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge/manual/5a3b2c1d-0e9f-4a8b-7c6d-5e4f3a2b1c0d' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "title": "产品使用指南 V2",
    "content": "# 产品使用指南 V2\n\n## 更新内容\n\n..."
}'
```

**Response**:

```json
{
    "data": {
        "id": "5a3b2c1d-0e9f-4a8b-7c6d-5e4f3a2b1c0d",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "manual",
        "title": "产品使用指南 V2",
        "parse_status": "processing",
        "enable_status": "enabled",
        "created_at": "2025-08-12T12:00:00.000000+08:00",
        "updated_at": "2025-08-12T12:30:00.000000+08:00"
    },
    "success": true
}
```

## POST `/knowledge/:id/reparse` - Re-parse knowledge

Asynchronously re-parses: deletes existing chunks/vectors and re-parses according to the latest configuration. Commonly used when the parsing configuration changes, or to retry after a previous parse failure.

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/reparse' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "message": "Knowledge reparse task submitted",
    "data": {
        "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "file",
        "title": "彗星.txt",
        "parse_status": "pending",
        "enable_status": "enabled",
        "created_at": "2025-08-12T11:52:36.168632+08:00",
        "updated_at": "2025-08-12T13:00:00.000000+08:00"
    }
}
```

After calling, `parse_status` first changes to `pending`, and is then transitioned by a background worker to `processing` → `completed`/`failed`.

## POST `/knowledge/:id/cancel-parse` - Cancel parsing

Aborts an in-progress parsing task, commonly used to proactively give up on parsing the current document when resources are tight.

**Behavior**:

- Sets `parse_status` to `cancelled`, writes "Parsing cancelled by user" into `error_message`, and resets `pending_subtasks_count` to zero.
- Chunks/indexes already written to the database are retained, and parsing can be re-triggered on the same record via the `reparse` endpoint.
- A background asynchronous process makes a best-effort attempt to remove downstream tasks corresponding to this knowledge entry from the queue (multimodal, question generation, summarization, graph extraction, post-process, etc.), and sends a stop signal to any worker currently executing; the worker exits at its next checkpoint.
- **Cancellable states**: `pending` / `processing` / `finalizing`. `finalizing` means the main parsing has completed but index optimization tasks such as summarization / question generation / graph extraction are still running; cancelling in this state promptly stops further LLM consumption (graph extraction is invoked per chunk and is the most expensive).
- Knowledge that has already completed (`completed`) or failed (`failed`) cannot be cancelled; knowledge that is being deleted (`deleting`) cannot be cancelled.
- The endpoint is idempotent: calling it repeatedly on an already-`cancelled` record simply returns the current status.

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/cancel-parse' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "message": "Knowledge parse cancelled",
    "data": {
        "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "type": "file",
        "title": "彗星.txt",
        "parse_status": "cancelled",
        "error_message": "用户已取消解析",
        "enable_status": "disabled",
        "created_at": "2025-08-12T11:52:36.168632+08:00",
        "updated_at": "2025-08-12T13:05:00.000000+08:00"
    }
}
```

## GET `/knowledge/:id/download` - Download the original file

Downloads the original file corresponding to a knowledge entry as an `attachment`.

**Response headers**:

```
Content-Type: application/octet-stream
Content-Disposition: attachment; filename="彗星.txt"
```

**Request**:

```curl
curl --location -OJ 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/download' \
--header 'X-API-Key: sk-xxxxx'
```

The response body is the file's binary stream.

## GET `/knowledge/:id/preview` - Inline file preview

Returns the original file for **inline preview** in the browser:

- `Content-Type` is mapped based on the file extension (`.pdf` → `application/pdf`, `.png` → `image/png`, `.txt`/`.md`/`.json` etc. → the corresponding text MIME type with `charset=utf-8`; unknown extensions fall back to `application/octet-stream`).
- `Content-Disposition: inline; filename="<original file name>"`, causing the browser to render inline rather than download.
- `Cache-Control: private, max-age=3600`.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/preview' \
--header 'X-API-Key: sk-xxxxx' \
-D -
```

**Example response headers**:

```
HTTP/1.1 200 OK
Content-Type: text/plain; charset=utf-8
Content-Disposition: inline; filename="彗星.txt"
Cache-Control: private, max-age=3600
```

The response body is the file content (to be interpreted according to `Content-Type`).

## PUT `/knowledge/image/:id/:chunk_id` - Update chunk image information

Updates metadata such as the description/alt text for a specific image chunk under a given knowledge entry.

**Path parameters**:

| Field       | Type   | Description     |
| ---------- | ------ | -------- |
| `id`       | string | Knowledge ID  |
| `chunk_id` | string | Chunk ID  |

**Request body**:

| Field         | Type   | Required | Description                                 |
| ------------ | ------ | ---- | ------------------------------------ |
| `image_info` | string | Yes   | Image information (a business-side JSON string)       |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge/image/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "image_info": "{\"description\":\"产品架构图\",\"alt_text\":\"WeKnora 系统架构\"}"
}'
```

**Response**:

```json
{
    "success": true,
    "message": "Knowledge chunk image updated successfully"
}
```

## PUT `/knowledge/tags` - Batch-update knowledge tags

Batch sets/clears tags for multiple knowledge entries.

**Request body**:

| Field      | Type                       | Required | Description                                                                       |
| --------- | --------------------------- | ---- | -------------------------------------------------------------------------- |
| `updates` | object<string, string\|null> | Yes   | Mapping of knowledge ID → tag ID; a value of `null` clears the tag on that knowledge entry                |
| `kb_id`   | string                     | No   | Restricts the scope to a knowledge base; when specified, edit permission is validated against this KB (required in shared KB scenarios)             |

If `kb_id` is not passed, the service infers the owning knowledge base from the first knowledge ID in `updates` and authorizes against that.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge/tags' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "kb_id": "kb-00000001",
    "updates": {
        "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5": "tag-00000001",
        "9c8af585-ae15-44ce-8f73-45ad18394651": null
    }
}'
```

**Response**:

```json
{ "success": true }
```

## GET `/knowledge/search` - Search/filter knowledge across knowledge bases

Searches knowledge by keyword within the current space (including entries shared to the current space); can filter by file type; when `agent_id` is specified, the search is scoped to the knowledge base range configured for that shared Agent.

**Query parameters**:

| Field         | Type    | Default | Description                                                                  |
| ------------ | ------- | ---- | ----------------------------------------------------------------------- |
| `keyword`    | string  | -    | Keyword (optional)                                                       |
| `offset`     | integer | 0    | Offset                                                                |
| `limit`      | integer | 20   | Number of results to return                                                              |
| `file_types` | string  | -    | Comma-separated list of extensions, e.g. `txt,pdf,docx`                            |
| `agent_id`   | string  | -    | Shared Agent ID; scoped according to that Agent's KB selection mode (`all`/`selected`/`none`) |

**Request**:

```curl
curl --location --get 'http://localhost:8080/api/v1/knowledge/search' \
--header 'X-API-Key: sk-xxxxx' \
--data-urlencode 'keyword=彗星' \
--data-urlencode 'offset=0' \
--data-urlencode 'limit=10' \
--data-urlencode 'file_types=txt,pdf'
```

**Response**:

```json
{
    "success": true,
    "data": [
        {
            "id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
            "tenant_id": 1,
            "knowledge_base_id": "kb-00000001",
            "type": "file",
            "title": "彗星.txt",
            "description": "彗星是由冰和尘埃构成的太阳系小天体...",
            "file_name": "彗星.txt",
            "file_type": "txt",
            "file_size": 7710,
            "parse_status": "completed",
            "enable_status": "enabled",
            "created_at": "2025-08-12T11:52:36.168632+08:00",
            "updated_at": "2025-08-12T11:52:53.376871+08:00"
        }
    ],
    "has_more": false
}
```

> Note: unlike other list endpoints, `data` here is an **array** rather than a nested `{data, has_more}` object; `has_more` is a sibling of `data`.

When `agent_id=...` is passed and that Agent's KB selection mode is `none`, the response directly returns `data: []` and `has_more: false`.

## POST `/knowledge/batch-reparse` - Batch re-parse within the same knowledge base

Batch re-parses knowledge by ID list within a single knowledge base (async task). Up to 200 IDs per call; the service validates that all IDs exist and belong to the same `kb_id`.

**Request body**:

| Field             | Type     | Required | Description                                             |
| ---------------- | -------- | ---- | ------------------------------------------------ |
| `kb_id`          | string   | Yes   | Target knowledge base ID                                    |
| `ids`            | string[] | Yes   | List of knowledge IDs to re-parse (≤ 200)                |
| `process_config` | object   | No   | Processing configuration overrides shared by this batch; if omitted, each knowledge entry's original configuration is used |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/batch-reparse' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "kb_id": "kb-00000001",
    "ids": [
        "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "9c8af585-ae15-44ce-8f73-45ad18394651"
    ]
}'
```

**Response**:

```json
{
    "success": true,
    "message": "Batch reparse task submitted",
    "data": {
        "task_id": "a1b2c3d4",
        "reparse_count": 2
    }
}
```

An HTTP success indicates that the batch wrapper task has been queued. In the background, each knowledge entry in the list is attempted; if a single item's submission fails, that entry enters the `failed` state with the error retained, and the batch task is also marked as failed in the run queue. To avoid re-cleaning up entries that were already successfully submitted, the entire batch is not automatically retried after a partial failure — you can filter for the failed entries and re-submit a batch re-parse for those.

If any ID does not belong to `kb_id` or does not exist, HTTP 400 is returned and the entire batch is rejected.

## POST `/knowledge/batch-delete` - Batch delete within the same knowledge base

Batch deletes knowledge by ID list within a single knowledge base (async task). Up to 200 IDs per call; the service validates that all IDs belong to the same `kb_id`.

**Request body**:

| Field    | Type     | Required | Description                              |
| ------- | -------- | ---- | ---------------------------------- |
| `kb_id` | string   | Yes   | Target knowledge base ID                     |
| `ids`   | string[] | Yes   | List of knowledge IDs to delete (≤ 200)     |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/batch-delete' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "kb_id": "kb-00000001",
    "ids": [
        "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
        "9c8af585-ae15-44ce-8f73-45ad18394651"
    ]
}'
```

**Response**:

```json
{
    "success": true,
    "message": "Batch delete task submitted",
    "data": {
        "task_id": "kg_delete_1_kb-00000001_xxxx",
        "deleted_count": 2
    }
}
```

If any ID does not belong to `kb_id` or does not exist, HTTP 400 is returned and the entire batch is rejected.

## POST `/knowledge/move` - Migrate knowledge to another knowledge base

Migrates one or more knowledge entries **in the `completed` state** from the source KB to the target KB (async). Constraints:

- The source/target KB must both belong to the current space;
- The source and target must be of the same KB type and **use the same Embedding model**;
- Source KB ≠ target KB;
- Only knowledge with `parse_status=completed` can be migrated.

**Request body**:

| Field            | Type     | Required | Description                                                                            |
| --------------- | -------- | ---- | --------------------------------------------------------------------------------- |
| `knowledge_ids` | string[] | Yes   | List of knowledge IDs to migrate (at least 1)                                              |
| `source_kb_id`  | string   | Yes   | Source knowledge base ID                                                                     |
| `target_kb_id`  | string   | Yes   | Target knowledge base ID                                                                     |
| `mode`          | string   | Yes   | Migration mode: `reuse_vectors` (reuse vector data, zero cost) / `reparse` (re-parse in the target KB) |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/move' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "knowledge_ids": ["4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5"],
    "source_kb_id": "kb-00000001",
    "target_kb_id": "kb-00000002",
    "mode": "reuse_vectors"
}'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "task_id": "kg_move_1_kb-00000001_xxxx",
        "source_kb_id": "kb-00000001",
        "target_kb_id": "kb-00000002",
        "knowledge_count": 1,
        "message": "Knowledge move task started"
    }
}
```

After obtaining the `task_id`, poll progress via the next endpoint.

## GET `/knowledge/move/progress/:task_id` - Query migration progress

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge/move/progress/kg_move_1_kb-00000001_xxxx' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "task_id": "kg_move_1_kb-00000001_xxxx",
        "source_kb_id": "kb-00000001",
        "target_kb_id": "kb-00000002",
        "status": "completed",
        "progress": 100,
        "total": 1,
        "processed": 1,
        "failed": 0,
        "message": "迁移完成",
        "error": "",
        "created_at": 1731312000,
        "updated_at": 1731312045
    }
}
```

`status` takes values: `pending` / `processing` / `completed` / `failed`; `progress` is an integer percentage from 0 to 100; `created_at` / `updated_at` are Unix timestamps in seconds.
