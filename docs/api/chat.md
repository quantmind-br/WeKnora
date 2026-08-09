# Chat Feature API

[Back to index](./README.md)

| Method | Path                          | Description                     |
| ---- | ----------------------------- | ------------------------ |
| POST | `/knowledge-chat/:session_id` | Knowledge base-based Q&A         |
| POST | `/agent-chat/:session_id`     | Agent-based intelligent Q&A    |
| POST | `/knowledge-search`           | Knowledge base search     |
| GET  | `/sessions/:session_id/messages/:message_id/suggestions` | Retrieve already-generated post-answer suggestions |
| POST | `/sessions/:session_id/messages/:message_id/suggestions` | Force generation or fetch a new batch of suggestions |
| POST | `/sessions/:session_id/suggestion-events` | Report impression, click, and dismiss events |

## POST `/knowledge-chat/:session_id` - Knowledge base-based Q&A

RAG-based Q&A over the knowledge base, supporting SSE streaming responses.

**Query parameters**:

| Parameter | Values | Description |
|------|------|------|
| `resource_urls` | `handle` (default) / `public` | `public` makes the answer and the images in citations return directly loadable http(s) links, saving you from having to call the `/files` proxy one by one. See [Files and Image References](./README.md#file-and-image-references-resource-and-direct-links) for details |

This also applies to `/agent-chat/:session_id`, `/knowledge-search`, and `/sessions/continue-stream/:session_id` below.

**Request parameters**:

| Parameter | Type | Required | Description |
|------|------|------|------|
| `query` | string | Yes | Query text |
| `knowledge_base_ids` | string[] | No | List of knowledge base IDs |
| `knowledge_ids` | string[] | No | List of knowledge file IDs, to retrieve from specific files |
| `agent_id` | string | No | Custom Agent ID, specifying which agent to use |
| `summary_model_id` | string | No | Override the default summary model ID |
| `mentioned_items` | object[] | No | List of @-mentioned knowledge bases and files |
| `disable_title` | bool | No | Whether to disable automatic title generation (default false) |
| `images` | object[] | No | Attached images (base64 format), requires the Agent to have image upload enabled |
| `channel` | string | No | Source channel identifier: `web`, `api`, `im`, `browser_extension` |
| `suggestion_attribution` | object | No | When the user starts this turn from a suggested question, pass `{suggestion_set_id, question_id}`; the server validates attribution |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-chat/ceb9babb-1e30-41d7-817d-fd584954304b' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "Shape of a comet tail",
    "knowledge_base_ids": ["kb-00000001"],
    "agent_id": "builtin-quick-answer"
}'
```

**Response format**:
Server-Sent Events (Content-Type: text/event-stream)

**Response**:

```
event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"references","content":"","done":false,"knowledge_references":[{"id":"c8347bef-...","content":"Comet xxx.","knowledge_id":"a6790b93-...","chunk_index":0,"knowledge_title":"Comet.txt","score":4.04,"match_type":3,"chunk_type":"text","knowledge_filename":"Comet.txt"}]}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"The shape of a comet tail mainly depends on...","done":false,"knowledge_references":null}

event: message
data: {"id":"3475c004-0ada-4306-9d30-d7f5efce50d2","response_type":"answer","content":"","done":true,"knowledge_references":null}
```

## POST `/agent-chat/:session_id` - Agent-based intelligent Q&A

Agent mode supports more intelligent Q&A, including tool calling, web search, multi-knowledge-base retrieval, and other capabilities.

**Request parameters**:

| Parameter | Type | Required | Description |
|------|------|------|------|
| `query` | string | Yes | Query text |
| `knowledge_base_ids` | string[] | No | List of knowledge base IDs; lets you dynamically specify which knowledge bases to use for this query |
| `knowledge_ids` | string[] | No | List of knowledge file IDs; lets you dynamically specify which specific files to use for this query |
| `agent_enabled` | bool | No | Whether to enable Agent mode (default false, Agent configuration takes priority) |
| `agent_id` | string | No | Custom Agent ID, specifying which agent to use (supports shared Agents) |
| `web_search_enabled` | bool | No | Whether to enable web search (default false) |
| `summary_model_id` | string | No | Override the default summary model ID |
| `mentioned_items` | object[] | No | List of @-mentioned knowledge bases and files |
| `disable_title` | bool | No | Whether to disable automatic title generation (default false) |
| `images` | object[] | No | Attached images (base64 format), requires the Agent to have image upload enabled |
| `channel` | string | No | Source channel identifier: `web`, `api`, `im`, `browser_extension` |
| `suggestion_attribution` | object | No | When the user starts this turn from a suggested question, pass `{suggestion_set_id, question_id}`; the server validates attribution |

## Post-Answer Suggested Questions

Once the main answer message is complete, the server asynchronously generates suggested questions without blocking the SSE `complete`/`done` events. The generated results are persisted and deduplicated by "space, assistant message, position, configuration snapshot, and language."

```http
POST /api/v1/sessions/{session_id}/messages/{message_id}/suggestions
Content-Type: application/json

{"regenerate": false}
```

Status values include `generating`, `ready`, `suppressed`, and `failed`. Each question has a stable `id` once `ready`; a click should report the event first, then carry `suggestion_attribution` on the next chat request.

```http
POST /api/v1/sessions/{session_id}/suggestion-events
Content-Type: application/json

{
  "suggestion_set_id": "...",
  "question_id": "...",
  "event_type": "click"
}
```

Web embedding provides an isomorphic interface: `/api/v1/embed/{channel_id}/sessions/{session_id}/...`, continuing to use the embed token and `X-Embed-Session`.

**mentioned_items structure**:

| Field | Type | Description |
|------|------|------|
| `id` | string | Knowledge base or file ID |
| `name` | string | Display name |
| `type` | string | Type: `kb` (knowledge base) or `file` (file) |
| `kb_type` | string | Knowledge base type: `document` or `faq` (only when `type=kb`) |

**images structure**:

| Field | Type | Description |
|------|------|------|
| `data` | string | Base64-encoded image data (`data:image/png;base64,...`) |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/agent-chat/ceb9babb-1e30-41d7-817d-fd584954304b' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "Check today's weather",
    "agent_enabled": true,
    "web_search_enabled": true,
    "knowledge_base_ids": ["kb-00000001"],
    "agent_id": "builtin-smart-reasoning",
    "mentioned_items": [
        {
            "id": "kb-00000001",
            "name": "Weather Knowledge Base",
            "type": "kb",
            "kb_type": "document"
        }
    ]
}'
```

**Response format**:
Server-Sent Events (Content-Type: text/event-stream)

**Response type reference**:

| response_type | Description |
|---------------|------|
| `agent_query` | Agent begins processing the query |
| `thinking` | Agent's thinking process |
| `tool_call` | Tool call information |
| `tool_result` | Tool call result |
| `references` | Knowledge base retrieval references |
| `answer` | Final answer content |
| `reflection` | Agent reflection content |
| `session_title` | Automatically generated session title |
| `error` | Error information |

**Response example**:

```
event: message
data: {"id":"req-001","response_type":"thinking","content":"The user wants the weather, so I need to use the web search tool...","done":false}

event: message
data: {"id":"req-001","response_type":"tool_call","content":"","done":false,"data":{"tool_name":"web_search","arguments":{"query":"today's weather"}}}

event: message
data: {"id":"req-001","response_type":"tool_result","content":"Search results: sunny today, 25°C...","done":false}

event: message
data: {"id":"req-001","response_type":"answer","content":"According to the results, today is sunny with a temperature around 25°C.","done":false}

event: message
data: {"id":"req-001","response_type":"answer","content":"","done":true}
```
