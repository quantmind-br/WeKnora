# Chunk Management API

[Back to index](./README.md)

| Method | Path                              | Description                          |
| ------ | --------------------------------- | ------------------------------------- |
| GET    | `/chunks/:knowledge_id`           | Get the list of chunks for a knowledge item |
| PUT    | `/chunks/:knowledge_id/:id`       | Update a chunk                        |
| DELETE | `/chunks/:knowledge_id/:id`       | Delete a single chunk                 |
| DELETE | `/chunks/:knowledge_id`           | Delete all chunks under a knowledge item |
| GET    | `/chunks/by-id/:id`               | Get a chunk directly by chunk ID      |
| DELETE | `/chunks/by-id/:id/questions`     | Delete a generated question under a chunk |

## GET `/chunks/:knowledge_id` - Get the list of chunks for a knowledge item

**Path parameters**:

| Field         | Type   | Description     |
| ------------- | ------ | --------------- |
| knowledge_id  | string | Knowledge ID    |

**Query parameters**:

| Field      | Type | Default | Description        |
| ---------- | ---- | ------- | ------------------- |
| page       | int  | 1       | Page number          |
| page_size  | int  | 20      | Items per page        |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/chunks/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5?page=1&page_size=1' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "id": "df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7",
            "tenant_id": 1,
            "knowledge_id": "4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5",
            "knowledge_base_id": "kb-00000001",
            "tag_id": "",
            "content": "Comet xxxx",
            "chunk_index": 0,
            "is_enabled": true,
            "status": 2,
            "start_at": 0,
            "end_at": 964,
            "pre_chunk_id": "",
            "next_chunk_id": "",
            "chunk_type": "text",
            "parent_chunk_id": "",
            "relation_chunks": null,
            "indirect_relation_chunks": null,
            "metadata": null,
            "content_hash": "",
            "image_info": "",
            "created_at": "2025-08-12T11:52:36.168632+08:00",
            "updated_at": "2025-08-12T11:52:53.376871+08:00",
            "deleted_at": null
        }
    ],
    "page": 1,
    "page_size": 1,
    "success": true,
    "total": 5
}
```

## PUT `/chunks/:knowledge_id/:id` - Update a chunk

Updates the content and properties of the specified chunk. All fields are optional; any field not provided keeps its original value.

**Path parameters**:

| Field         | Type   | Description     |
| ------------- | ------ | --------------- |
| knowledge_id  | string | Knowledge ID    |
| id            | string | Chunk ID        |

**Parameters (request body)**:

| Field        | Type    | Required | Description                          |
| ------------ | ------- | -------- | -------------------------------------- |
| content      | string  | No       | Chunk content                          |
| chunk_index  | int     | No       | The chunk's sequence number within the knowledge item |
| is_enabled   | boolean | No       | Whether it is enabled                  |
| start_at     | int     | No       | Start position (character offset)      |
| end_at       | int     | No       | End position (character offset)        |
| image_info   | string  | No       | Metadata for an image chunk (JSON string) |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/chunks/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "content": "Updated chunk content",
    "is_enabled": true
}'
```

**Response**:

```json
{
    "data": {
        "id": "df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7",
        "content": "Updated chunk content",
        "is_enabled": true,
        "...": "Other fields are the same as the GET response"
    },
    "success": true
}
```

## DELETE `/chunks/:knowledge_id/:id` - Delete a single chunk

**Path parameters**: Same as PUT.

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/chunks/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5/df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "message": "Chunk deleted",
    "success": true
}
```

## DELETE `/chunks/:knowledge_id` - Delete all chunks under a knowledge item

**Path parameters**:

| Field         | Type   | Description     |
| ------------- | ------ | --------------- |
| knowledge_id  | string | Knowledge ID    |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/chunks/4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "message": "All chunks under knowledge deleted",
    "success": true
}
```

## GET `/chunks/by-id/:id` - Get a chunk directly by ID

Retrieves a chunk without needing to provide `knowledge_id`. Commonly used for displaying references across knowledge bases.

**Path parameters**:

| Field | Type   | Description  |
| ---- | ------ | ------------- |
| id   | string | Chunk ID      |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/chunks/by-id/df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**: Same as a single data entry in the `GET /chunks/:knowledge_id` list.

## DELETE `/chunks/by-id/:id/questions` - Delete a generated question under a chunk

Deletes a specific generated question associated with the specified chunk.

**Path parameters**:

| Field | Type   | Description  |
| ---- | ------ | ------------- |
| id   | string | Chunk ID      |

**Parameters (request body)**:

| Field        | Type   | Required | Description   |
| ----------- | ------ | ---- | ----------- |
| question_id | string | Yes  | Question ID |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/chunks/by-id/df10b37d-cd05-4b14-ba8a-e1bd0eb3bbd7/questions' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "question_id": "q-00000001"
}'
```

**Response**:

```json
{
    "message": "Question deleted successfully",
    "success": true
}
```
