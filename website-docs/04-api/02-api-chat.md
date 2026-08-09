# API Reference: Sessions, Messages & Chat

Route registration: `RegisterSessionRoutes`, `RegisterChatRoutes`, `RegisterMessageRoutes` in `internal/router/router.go`. Handlers: `internal/handler/session/` (handler.go, qa.go, stream.go, title.go, temporary_document.go), `internal/handler/message.go`, `internal/handler/message_suggestion.go`.

Sessions are a "user-private" resource; ownership is enforced internally by the handlers. Route-level access requires Viewer+. API key: sessions/chat require the `chat` capability (or full-access); message search requires `message_history`; knowledge retrieval requires `retrieve`.

## Sessions (/api/v1/sessions)

### POST /api/v1/sessions

Purpose: create a session. Handler: `internal/handler/session/handler.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `title` | string | No | Title |
| `description` | string | No | Description |

Response: 201 `{"success":true,"data":{Session}}` (`id,title,description,tenant_id,user_id,is_pinned,last_request_state,created_at,...`)

```bash
curl -X POST $BASE/api/v1/sessions -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"title":"New chat"}'
```

### GET /api/v1/sessions

Purpose: list sessions. Handler: `internal/handler/session/handler.go`

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `page` / `page_size` | int | No | Pagination |
| `keyword` | string | No | Fuzzy title search |
| `source` | string | No | Source filter (web/embed/api/feishu/wechat/slack/...) |
| `agent_id` | string | No | Filter by Agent (IM sessions) |

Response: 200 `{"success":true,"data":[SessionListItem],"total","page","page_size"}`

```bash
curl "$BASE/api/v1/sessions?page=1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/:id

Purpose: session details.

Response: 200 `{"success":true,"data":{Session}}`

```bash
curl $BASE/api/v1/sessions/s-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/sessions/:id

Purpose: update a session (title/description/pin). Request body: `title`, `description`, `is_pinned` (all optional).

Response: 200 `{"success":true,"data":{Session}}`

```bash
curl -X PUT $BASE/api/v1/sessions/s-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"title":"Rename"}'
```

### DELETE /api/v1/sessions/:id

Purpose: delete a session.

Response: 200 `{"success":true,"message":"Session deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/sessions/s-1 -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/sessions/batch

Purpose: batch-delete sessions. Request body: `{"ids":["s-1"],"delete_all":false}` (either `ids` or `delete_all:true`).

Response: 200 `{"success":true,"message":"Sessions deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/sessions/batch -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"ids":["s-1","s-2"]}'
```

### DELETE /api/v1/sessions/:id/messages

Purpose: clear a session's messages.

Response: 200 `{"success":true,"message":"Session messages cleared successfully"}`

```bash
curl -X DELETE $BASE/api/v1/sessions/s-1/messages -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/sessions/:session_id/generate_title

Purpose: generate a session title based on context messages. Handler: `internal/handler/session/title.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `messages` | []Message | Yes (`binding:"required"`) | Messages to use as context |

Response: 200 `{"success":true,"data":"Generated title"}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/generate_title -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"messages":[{"role":"user","content":"Introduce the product"}]}'
```

### POST /api/v1/sessions/:session_id/stop

Purpose: stop an in-progress generation. Handler: `internal/handler/session/stream.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `message_id` | string | Yes (`binding:"required"`) | Assistant message ID |

Response: 200 `{"success":true,"message":"Generation stopped"}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/stop -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-1"}'
```

### POST /api/v1/sessions/:session_id/pin and DELETE /api/v1/sessions/:id/pin

Purpose: pin / unpin a session. No request body. Handler: `internal/handler/session/handler.go`

Response: 200 `{"success":true,"is_pinned":true|false}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/pin -H "Authorization: Bearer $TOKEN"
curl -X DELETE $BASE/api/v1/sessions/s-1/pin -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/continue-stream/:session_id

Purpose: resume an active stream after a disconnect (replays historical events + polls for new deltas every 100ms). Handler: `internal/handler/session/stream.go`

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `message_id` | string | Yes | Assistant message ID to resume |

Response: 200 SSE (`text/event-stream`; see the overview's "Streaming API protocol" for the event format).

```bash
curl -N "$BASE/api/v1/sessions/continue-stream/s-1?message_id=m-1" -H "Authorization: Bearer $TOKEN"
```

## Session Attachments (Temporary Documents)

Handler: `internal/handler/session/temporary_document.go`

### POST /api/v1/sessions/:session_id/attachments

Purpose: upload a session-level temporary document (parsed asynchronously). Multipart fields: `file` (required), `agent_id` (optional, determines the parsing engine/ASR model), `parser_engine` (optional).

Response: 202 `{"success":true,"data":{TemporaryDocument}}` (`id,session_id,file_name,file_type,file_size,status(uploaded/processing/ready/failed),resource_ref,...`)

```bash
curl -X POST $BASE/api/v1/sessions/s-1/attachments -H "Authorization: Bearer $TOKEN" -F 'file=@notes.pdf'
```

### GET /api/v1/sessions/:id/attachments

Purpose: list attachments.

Response: 200 `{"success":true,"data":[TemporaryDocument]}`

```bash
curl $BASE/api/v1/sessions/s-1/attachments -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/:id/attachments/:attachment_id

Purpose: attachment details (including parsing status).

Response: 200 `{"success":true,"data":{TemporaryDocument}}`

```bash
curl $BASE/api/v1/sessions/s-1/attachments/a-1 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/sessions/:id/attachments/:attachment_id/preview

Purpose: preview the original attachment file.

Response: 200 file stream (`Content-Disposition: inline|attachment`, `Cache-Control: private`).

```bash
curl $BASE/api/v1/sessions/s-1/attachments/a-1/preview -H "Authorization: Bearer $TOKEN" -o preview.pdf
```

### DELETE /api/v1/sessions/:id/attachments/:attachment_id

Purpose: delete an attachment.

Response: 204 No Content

```bash
curl -X DELETE $BASE/api/v1/sessions/s-1/attachments/a-1 -H "Authorization: Bearer $TOKEN"
```

## Answer Suggestions

Handler: `internal/handler/message_suggestion.go`

### GET /api/v1/sessions/:id/messages/:message_id/suggestions

Purpose: read the follow-up suggestions for a given assistant message.

Response: 200 `{"success":true,"data":{MessageSuggestionSet}}` (`status(generating/ready/suppressed/failed),questions:[{id,text,category,source,knowledge_base_ids}],allow_regenerate,...`)

```bash
curl $BASE/api/v1/sessions/s-1/messages/m-1/suggestions -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/sessions/:session_id/messages/:message_id/suggestions

Purpose: ensure suggestions are generated (idempotent trigger). Request body: `{"regenerate":true}` (optional, forces regeneration).

Response: 200 (ready) or 202 (generating) `{"success":true,"data":{MessageSuggestionSet|null}}`

```bash
curl -X POST $BASE/api/v1/sessions/s-1/messages/m-1/suggestions -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{}'
```

### POST /api/v1/sessions/:session_id/suggestion-events

Purpose: report suggestion interaction events (analytics tracking).

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `suggestion_set_id` | string | Yes (`binding:"required"`) | Suggestion set ID |
| `question_id` | string | No | Required for click/regenerate |
| `event_type` | string | Yes (`binding:"required"`) | `impression/click/dismiss/regenerate` |

Response: 204 No Content

```bash
curl -X POST $BASE/api/v1/sessions/s-1/suggestion-events -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"suggestion_set_id":"ss-1","event_type":"impression"}'
```

## Chat & Retrieval

Handler: `internal/handler/session/qa.go`. API key: chat requires `chat`/full; `knowledge-search` requires `retrieve`/full.

### POST /api/v1/knowledge-chat/:session_id

Purpose: knowledge base Q&A (SSE streaming).

Request body (shared by KnowledgeQA/AgentQA):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query` | string | Yes (`binding:"required"`) | User's question |
| `knowledge_base_ids` | []string | No | KBs to retrieve from |
| `knowledge_ids` | []string | No | Restrict to specific knowledge files |
| `agent_enabled` | bool | No | Whether to enable Agent mode |
| `agent_id` | string | No | Custom Agent ID |
| `web_search_enabled` | bool | No | Web search |
| `summary_model_id` | string | No | Summarization model |
| `mcp_service_ids` | []string | No | @-mentioned MCP services |
| `skill_names` | []string | No | @-mentioned skills |
| `tag_ids` | []string | No | Tag filter |
| `mentioned_items` | []object | No | @-mentioned items (type/kb_id/kb_name/service_id/skill_name) |
| `disable_title` | bool | No | Disable auto-titling |
| `images` | []object | No | Images (`data` base64 / `url` / `caption`) |
| `attachment_uploads` | []object | No | Inline attachments (`data` base64, `file_name`, `file_size`) |
| `attachment_ids` | []string | No | IDs of already-uploaded session attachments |
| `channel` | string | No | Source channel |
| `suggestion_attribution` | object | No | Attribution info for a clicked suggestion |

Response: 200 SSE stream, `event: message` + `data: StreamResponse` (see overview), ending with a `complete` event.

```bash
curl -N -X POST $BASE/api/v1/knowledge-chat/s-1 -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"What is the refund policy?","knowledge_base_ids":["kb-1"]}'
```

### POST /api/v1/agent-chat/:session_id

Purpose: Agent Q&A (SSE streaming, including `thinking/tool_call/tool_result/tool_approval_required/mcp_oauth_required` and other events). Request body is the same as above.

```bash
curl -N -X POST $BASE/api/v1/agent-chat/s-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"query":"Analyze last quarter data","agent_id":"agent-1"}'
```

### POST /api/v1/knowledge-search

Purpose: sessionless knowledge retrieval (non-streaming). Handler: `SearchKnowledge` in `internal/handler/session/qa.go`.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query` | string | Yes (`binding:"required"`) | Query |
| `knowledge_base_id` | string | No | Single KB (legacy compatibility) |
| `knowledge_base_ids` | []string | No | Multiple KBs |
| `knowledge_ids` | []string | No | Restrict to specific files |
| `tag_ids` | []string | No | Tag filter |
| `mentioned_items` | []object | No | Tag mentions scoped to a KB |

Response: 200 `{"success":true,"data":[SearchResult]}` (`id,content,knowledge_id,knowledge_title,score,chunk_type,knowledge_base_id,...`)

```bash
curl -X POST $BASE/api/v1/knowledge-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query":"Deployment requirements","knowledge_base_ids":["kb-1"]}'
```

## Messages (/api/v1/messages)

Handler: `internal/handler/message.go`

### POST /api/v1/messages/search

Purpose: chat history search. Permission: Viewer+; API key `message_history`/full.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query` | string | Yes (`binding:"required"`) | Query |
| `mode` | string | No | `keyword/vector/hybrid` (default hybrid) |
| `limit` | int | No | Default 20 |
| `session_ids` | []string | No | Restrict to specific sessions |

Response: 200 `{"success":true,"data":{"total":N,"results":[{session_id,message_id,role,content,created_at,score}]}}`

```bash
curl -X POST $BASE/api/v1/messages/search -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"query":"Quote"}'
```

### GET /api/v1/messages/chat-history-stats

Purpose: chat history index statistics. Permission: Viewer+; API key `message_history`/full.

Response: 200 `{"success":true,"data":{indexed_message_count,knowledge_base_size,last_indexed_at,...}}`

```bash
curl $BASE/api/v1/messages/chat-history-stats -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/messages/:session_id/load

Purpose: load a session's messages (forward pagination via time cursor). Permission: Viewer+; API key `chat`/full.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `limit` | int | No | Default 20 |
| `before_time` | string | No | RFC3339/RFC3339Nano timestamp |

Response: 200 `{"success":true,"data":[Message]}` (`id,session_id,role,content,is_completed,images,attachments,agent_steps,...`)

```bash
curl "$BASE/api/v1/messages/s-1/load?limit=20" -H "X-API-Key: $API_KEY"
```

### DELETE /api/v1/messages/:session_id/:id

Purpose: delete a single message. Permission: Viewer+ (handler validates session ownership); API key `chat`/full.

Response: 200 `{"success":true,"message":"Message deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/messages/s-1/m-1 -H "Authorization: Bearer $TOKEN"
```

---
