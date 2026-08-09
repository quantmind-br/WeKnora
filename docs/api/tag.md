# Tag Management API

[Back to Table of Contents](./README.md)

| Method | Path                                  | Description               |
| ------ | -------------------------------------- | -------------------------- |
| GET    | `/knowledge-bases/:id/tags`           | Get the knowledge base tag list |
| POST   | `/knowledge-bases/:id/tags`           | Create a tag               |
| PUT    | `/knowledge-bases/:id/tags/:tag_id`   | Update a tag                |
| DELETE | `/knowledge-bases/:id/tags/:tag_id`   | Delete a tag                |

## GET `/knowledge-bases/:id/tags` - Get the knowledge base tag list

**Query Parameters**:
- `page`: Page number (default 1)
- `page_size`: Items per page (default 20)
- `keyword`: Keyword search on tag name (optional)

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags?page=1&page_size=10' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "total": 2,
        "page": 1,
        "page_size": 10,
        "data": [
            {
                "id": "tag-00000001",
                "tenant_id": 1,
                "knowledge_base_id": "kb-00000001",
                "name": "技术文档",
                "color": "#1890ff",
                "sort_order": 1,
                "created_at": "2025-08-12T10:00:00+08:00",
                "updated_at": "2025-08-12T10:00:00+08:00",
                "knowledge_count": 5,
                "chunk_count": 120
            },
            {
                "id": "tag-00000002",
                "tenant_id": 1,
                "knowledge_base_id": "kb-00000001",
                "name": "常见问题",
                "color": "#52c41a",
                "sort_order": 2,
                "created_at": "2025-08-12T10:00:00+08:00",
                "updated_at": "2025-08-12T10:00:00+08:00",
                "knowledge_count": 3,
                "chunk_count": 45
            }
        ]
    },
    "success": true
}
```

## POST `/knowledge-bases/:id/tags` - Create a tag

**Path Parameters**:

| Field | Type   | Description        |
| ----- | ------ | ------------------- |
| id    | string | Knowledge base ID    |

**Parameter Description (Request Body)**:

| Field       | Type   | Required | Description                          |
| ----------- | ------ | -------- | ------------------------------------- |
| name        | string | Yes      | Tag name (unique within the knowledge base) |
| color       | string | No       | Tag color (CSS color string)          |
| sort_order  | int    | No       | Sort value (lower values appear first) |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "产品手册",
    "color": "#faad14",
    "sort_order": 3
}'
```

**Response**:

```json
{
    "data": {
        "id": "tag-00000003",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "name": "产品手册",
        "color": "#faad14",
        "sort_order": 3,
        "created_at": "2025-08-12T11:00:00+08:00",
        "updated_at": "2025-08-12T11:00:00+08:00"
    },
    "success": true
}
```

## PUT `/knowledge-bases/:id/tags/:tag_id` - Update a tag

**Path Parameters**:

| Field  | Type   | Description        |
| ------ | ------ | ------------------- |
| id     | string | Knowledge base ID    |
| tag_id | string | Tag ID               |

**Parameter Description (Request Body)**: Same as the create endpoint; all fields are optional. Fields that are not provided retain their original value.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags/tag-00000003' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "产品手册更新",
    "color": "#ff4d4f"
}'
```

**Response**:

```json
{
    "data": {
        "id": "tag-00000003",
        "tenant_id": 1,
        "knowledge_base_id": "kb-00000001",
        "name": "产品手册更新",
        "color": "#ff4d4f",
        "sort_order": 3,
        "created_at": "2025-08-12T11:00:00+08:00",
        "updated_at": "2025-08-12T11:30:00+08:00"
    },
    "success": true
}
```

## DELETE `/knowledge-bases/:id/tags/:tag_id` - Delete a tag

**Path Parameters**:

| Field  | Type   | Description     |
| ------ | ------ | ----------------- |
| id     | string | Knowledge base ID |
| tag_id | string | Tag ID             |

**Query Parameters**:

| Field | Type    | Default | Description                                                  |
| ----- | ------- | ------- | -------------------------------------------------------------- |
| force | boolean | false   | When set to `true`, forces deletion (even if the tag is referenced) |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/tags/tag-00000003?force=true' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "success": true
}
```
