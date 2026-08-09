# Knowledge Search API

[Back to index](./README.md)

| Method | Path                | Description     |
| ------ | ------------------- | ---------------- |
| POST   | `/knowledge-search` | Knowledge search |

## POST `/knowledge-search` - Knowledge Search

Search for relevant content in the knowledge base (without using an LLM summary), returning retrieval results directly.

**Parameters (request body)**:

| Field               | Type     | Required | Description                                                       |
| ------------------- | -------- | -------- | ------------------------------------------------------------------ |
| query               | string   | Yes      | Search query text                                                  |
| knowledge_base_id   | string   | No       | Single knowledge base ID (for backward compatibility); mutually exclusive with `knowledge_base_ids` |
| knowledge_base_ids  | string[] | No       | List of multiple knowledge base IDs, for cross-knowledge-base search |
| knowledge_ids       | string[] | No       | Further restrict to specific knowledge (files); if omitted, searches across the entire knowledge base |

> At least one of `knowledge_base_id` or `knowledge_base_ids` must be specified.

**Query parameters**:

- `resource_urls`: `handle` (default) or `public`. `public` replaces the `resource://` references in the retrieval results' `content` / `image_info` with loadable http(s) links. See [Files and Image References](./README.md#文件与图片引用resource-与直链) for details

**Request**:

```curl
# Search a single knowledge base
curl --location 'http://localhost:8080/api/v1/knowledge-search' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "如何使用知识库",
    "knowledge_base_id": "kb-00000001"
}'

# Search multiple knowledge bases
curl --location 'http://localhost:8080/api/v1/knowledge-search' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "如何使用知识库",
    "knowledge_base_ids": ["kb-00000001", "kb-00000002"]
}'

# Search a specific file
curl --location 'http://localhost:8080/api/v1/knowledge-search' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "如何使用知识库",
    "knowledge_ids": ["4c4e7c1a-09cf-485b-a7b5-24b8cdc5acf5"]
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

**Response field descriptions (data[])**:

| Field               | Type    | Description                                    |
| ------------------- | ------- | ----------------------------------------------- |
| id                  | string  | Chunk ID                                        |
| content             | string  | The matched chunk text                          |
| knowledge_id        | string  | ID of the knowledge item this chunk belongs to  |
| chunk_index         | int     | The chunk's sequence number within the knowledge item |
| knowledge_title     | string  | Title of the source knowledge item              |
| start_at / end_at   | int     | Character offset of the chunk in the source document |
| seq                 | int     | Match ranking number                            |
| score               | number  | Similarity (final score after rerank normalization) |
| chunk_type          | string  | Chunk type (`text` / `image` / ...)             |
| image_info          | string  | Additional info for image chunks (JSON string)  |
| metadata            | object  | Custom metadata                                 |
| knowledge_filename  | string  | Source file name                                |
| knowledge_source    | string  | Source type (`file` / `url` / `manual`)         |
