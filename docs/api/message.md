Vou traduzir o documento diretamente.

---

# Message Management API

[Back to index](./README.md)

| Method | Path                            | Description                                |
| ------ | ------------------------------- | ------------------------------------------- |
| GET    | `/messages/:session_id/load`    | Get the list of recent session messages     |
| DELETE | `/messages/:session_id/:id`     | Delete a message                            |
| POST   | `/messages/search`              | Search chat history                         |
| GET    | `/messages/chat-history-stats`  | Get chat history knowledge base statistics  |

## GET `/messages/:session_id/load` - Get the list of recent session messages

**Query parameters**:

- `before_time`: The `created_at` field of the earliest message from the previous fetch; leave empty to fetch the most recent messages
- `limit`: Number of items per page (default 20)
- `resource_urls`: `handle` (default) or `public`. `public` replaces `resource://` image references in historical messages with loadable http(s) links — see [File and Image References](./README.md#file-and-image-references-resource-and-direct-links) for details

**Request**:

```curl
curl --location --request GET 'http://localhost:8080/api/v1/messages/ceb9babb-1e30-41d7-817d-fd584954304b/load?limit=3&before_time=2030-08-12T14%3A35%3A42.123456789Z' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "彗尾的形状"
}'
```

**Response**:

```json
{
    "data": [
        {
            "id": "b8b90eeb-7dd5-4cf9-81c6-5ebcbd759451",
            "session_id": "ceb9babb-1e30-41d7-817d-fd584954304b",
            "request_id": "hCA8SDjxcAvv",
            "content": "<think>\n好的",
            "role": "assistant",
            "knowledge_references": [
                {
                    "id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "content": "彗星xxx",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 0,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 2760,
                    "seq": 0,
                    "score": 4.038836479187012,
                    "match_type": 4,
                    "sub_chunk_id": [
                        "688821f0-40bf-428e-8cb6-541531ebeb76",
                        "c1e9903e-2b4d-4281-be15-0149288d45c2",
                        "7d955251-3f79-4fd5-a6aa-02f81e044091"
                    ],
                    "metadata": {},
                    "chunk_type": "text",
                    "parent_chunk_id": "",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                },
                {
                    "id": "fa3aadee-cadb-4a84-9941-c839edc3e626",
                    "content": "# 文档名称\n彗星.txt\n\n# 摘要\n彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会释放气体形成彗发和彗尾。其轨道周期差异大，来源包括柯伊伯带和奥尔特云。彗星与小行星的区别逐渐模糊，部分彗星已失去挥发物质，类似小行星。目前已知彗星数量众多，且存在系外彗星。彗星在古代被视为凶兆，现代研究揭示其复杂结构与起源。",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 6,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 0,
                    "seq": 6,
                    "score": 0.6131043121858466,
                    "match_type": 0,
                    "sub_chunk_id": null,
                    "metadata": {},
                    "chunk_type": "summary",
                    "parent_chunk_id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                }
            ],
            "agent_steps": [],
            "is_completed": true,
            "is_fallback": false,
            "agent_duration_ms": 2500,
            "channel": "web",
            "created_at": "2025-08-12T10:24:38.370548+08:00",
            "updated_at": "2025-08-12T10:25:40.416382+08:00",
            "deleted_at": null
        },
        {
            "id": "7fa136ae-a045-424e-baac-52113d92ae94",
            "session_id": "ceb9babb-1e30-41d7-817d-fd584954304b",
            "request_id": "3475c004-0ada-4306-9d30-d7f5efce50d2",
            "content": "彗尾的形状",
            "role": "user",
            "knowledge_references": [],
            "agent_steps": [],
            "is_completed": true,
            "mentioned_items": [],
            "images": [],
            "channel": "web",
            "created_at": "2025-08-12T14:30:39.732246+08:00",
            "updated_at": "2025-08-12T14:30:39.733277+08:00",
            "deleted_at": null
        },
        {
            "id": "9bcafbcf-a758-40af-a9a3-c4d8e0f49439",
            "session_id": "ceb9babb-1e30-41d7-817d-fd584954304b",
            "request_id": "3475c004-0ada-4306-9d30-d7f5efce50d2",
            "content": "<think>\n好的",
            "role": "assistant",
            "knowledge_references": [
                {
                    "id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "content": "彗星xxx",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 0,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 2760,
                    "seq": 0,
                    "score": 4.038836479187012,
                    "match_type": 3,
                    "sub_chunk_id": [
                        "688821f0-40bf-428e-8cb6-541531ebeb76",
                        "c1e9903e-2b4d-4281-be15-0149288d45c2",
                        "7d955251-3f79-4fd5-a6aa-02f81e044091"
                    ],
                    "metadata": {},
                    "chunk_type": "text",
                    "parent_chunk_id": "",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                },
                {
                    "id": "fa3aadee-cadb-4a84-9941-c839edc3e626",
                    "content": "# 文档名称\n彗星.txt\n\n# 摘要\n彗星是由冰和尘埃构成的太阳系小天体，接近太阳时会释放气体形成彗发和彗尾。其轨道周期差异大，来源包括柯伊伯带和奥尔特云。彗星与小行星的区别逐渐模糊，部分彗星已失去挥发物质，类似小行星。目前已知彗星数量众多，且存在系外彗星。彗星在古代被视为凶兆，现代研究揭示其复杂结构与起源。",
                    "knowledge_id": "a6790b93-4700-4676-bd48-0d4804e1456b",
                    "chunk_index": 6,
                    "knowledge_title": "彗星.txt",
                    "start_at": 0,
                    "end_at": 0,
                    "seq": 6,
                    "score": 0.6131043121858466,
                    "match_type": 3,
                    "sub_chunk_id": null,
                    "metadata": {},
                    "chunk_type": "summary",
                    "parent_chunk_id": "c8347bef-127f-4a22-b962-edf5a75386ec",
                    "image_info": "",
                    "knowledge_filename": "彗星.txt",
                    "knowledge_source": ""
                }
            ],
            "agent_steps": [],
            "is_completed": true,
            "is_fallback": false,
            "agent_duration_ms": 2500,
            "channel": "web",
            "created_at": "2025-08-12T14:30:39.735108+08:00",
            "updated_at": "2025-08-12T14:31:17.829926+08:00",
            "deleted_at": null
        }
    ],
    "success": true
}
```

## DELETE `/messages/:session_id/:id` - Delete a message

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/messages/ceb9babb-1e30-41d7-817d-fd584954304b/9bcafbcf-a758-40af-a9a3-c4d8e0f49439' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "message": "Message deleted successfully",
    "success": true
}
```

## POST `/messages/search` - Search chat history

Search historical chat messages, supporting hybrid search, keyword search, and vector search modes.

**Request parameters**:
- `query`: Search keyword (required)
- `mode`: Search mode, one of `hybrid`, `keyword`, `vector` (optional, default `hybrid`)
- `limit`: Number of results to return (optional, default 20)
- `session_ids`: List of session IDs to restrict the search to (optional)

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/messages/search' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "彗星的结构",
    "mode": "hybrid",
    "limit": 20,
    "session_ids": []
}'
```

**Response**:

```json
{
    "data": {
        "items": [
            {
                "request_id": "3475c004-0ada-4306-9d30-d7f5efce50d2",
                "session_id": "ceb9babb-1e30-41d7-817d-fd584954304b",
                "session_title": "彗星知识问答",
                "query_content": "彗尾的形状",
                "answer_content": "彗尾的形状主要取决于...",
                "score": 0.85,
                "match_type": "hybrid",
                "created_at": "2025-08-12T14:30:39.732246+08:00"
            }
        ],
        "total": 1
    },
    "success": true
}
```

## GET `/messages/chat-history-stats` - Get chat history knowledge base statistics

Get index statistics for the chat history knowledge base in the current workspace.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/messages/chat-history-stats' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "enabled": true,
        "embedding_model_id": "dff7bc94-7885-4dd1-bfd5-bd96e4df2fc3",
        "knowledge_base_id": "kb-chat-00000001",
        "knowledge_base_name": "聊天历史知识库",
        "indexed_message_count": 1024,
        "has_indexed_messages": true
    },
    "success": true
}
```

---

Note: example field values (such as Chinese message content and summaries in the JSON blocks) are kept as-is., pois são dados de amostra/identificadores, não prosa estrutural do documento — traduzi-los alteraria o valor literal do JSON de exemplo. Se preferir que eu traduza também esses valores de amostra, posso ajustar.
