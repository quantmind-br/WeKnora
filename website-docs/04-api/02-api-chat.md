# API Reference: Sessions, Messages & Chat

Create and manage sessions, read messages and temporary attachments, and receive knowledge Q&A or agent answers over SSE.

Sessions are a "user-private" resource; ownership is enforced internally by the handlers. Route-level access requires Viewer+. API key: sessions/chat require the `chat` capability (or full-access); message search requires `message_history`; knowledge retrieval requires `retrieve`.

## Sessions (/api/v1/sessions)

### POST /api/v1/sessions

Purpose: create a session. Handler: `internal/handler/session/handler.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `title` | string | No | Title |
| `description` | string | No | Description |
| `project_dir` | string | No | Desktop edition only: binds the session to a local project directory the user has approved (absolute path, must exactly match the approved list); otherwise 400 is returned |

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

When stopping, appended messages that have not yet been delivered to the agent (see [Appending messages to a running answer](#steer) below) are discarded as well, and no new round starts automatically.

### POST /api/v1/sessions/:session_id/fork

Purpose: fork a new session from a historical message; the source session stays unchanged. Only the session owner can fork. Handler: `internal/handler/session/fork.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `message_id` | string | Yes | Fork point. User message: copies the history before it, and the client usually prefills that question into the input box; assistant message: copies the history up to and including that answer |
| `title` | string | No | Title of the new session; defaults to "source title (branch)" |

Copied messages keep their original timeline and generated files; the new session records `parent_session_id` and `forked_from_message_id`. When the source session is bound to a sandbox, the system snapshots the sandbox, and the new session starts from the workspace state of the round corresponding to the fork point the first time it uses the sandbox. When the workspace cannot be carried over, the fork still succeeds but returns `degraded: true` and a reason:

| `reason` | Meaning |
| --- | --- |
| `NO_CHECKPOINT` | The rounds before the fork point have no workspace checkpoint |
| `SANDBOX_REPLACED` | The checkpoint belongs to an old sandbox the session has since replaced |
| `SANDBOX_GONE` | The source session currently has no sandbox to snapshot |
| `SNAPSHOT_UNSUPPORTED` | The sandbox backend does not support snapshots, or the snapshot failed |

Response: 200 `{"success":true,"data":{"session_id":"new session ID","degraded":false}}`. Returns 409 (`code: FORK_SOURCE_BUSY`) when the source session is generating or the fork point is an unfinished answer; 404 when the session or message does not exist; 400 when the fork point is not a user or assistant message.

```bash
curl -X POST $BASE/api/v1/sessions/s-1/fork -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-3"}'
```

### POST /api/v1/sessions/:session_id/rewind

Purpose: roll the current session back in place to a given message. Only the session owner can roll back. Handler: `internal/handler/session/rewind.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `message_id` | string | Yes | Rollback point. User message: deletes it and every message after it; assistant message: keeps that answer and deletes the messages after it |

Generated files, follow-up suggestions, and chat history indexes of the deleted messages are cleaned up as well. When the session is bound to a sandbox, the system also resets `/workspace` to the checkpoint of the last round in the kept history; if the workspace reset fails, the whole operation fails and no messages are deleted.

Response: 200 `{"success":true,"data":{"deleted_messages":N,"workspace_reset":true,"reason":""}}`. When `workspace_reset` is false, `reason` explains why only the conversation was rolled back: `NO_SANDBOX` (the session is not bound to a sandbox) or `NO_CHECKPOINT` (the kept history has no checkpoint).

| Status code | `code` | Meaning |
| --- | --- | --- |
| 409 | `REWIND_SOURCE_BUSY` | The session is generating, or the rollback point is an unfinished answer |
| 409 | `REWIND_NO_CHECKPOINT` | The sandbox still exists, but no usable checkpoint can be found in the kept history; rejected to avoid files being ahead of the conversation |
| 409 | `REWIND_SANDBOX_REPLACED` | The checkpoint belongs to an old sandbox that has been replaced |
| 404 | — | The session or message does not exist |
| 500 | — | The workspace reset failed; you can retry |

```bash
curl -X POST $BASE/api/v1/sessions/s-1/rewind -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-3"}'
```

### Appending Messages to a Running Answer (steer) {#steer}

While a smart-reasoning answer is being generated, calling `agent-chat` again on the same session returns 409. In that case, use the following endpoints to queue a new message into the current round. Only the session owner can call them; quick Q&A has no execution loop to inject into, so appending is not supported. Handler: `internal/handler/session/steer.go`

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/api/v1/sessions/:session_id/steer` | Append a message |
| GET | `/api/v1/sessions/:id/steer` | List the queued messages of the current round that have not been delivered yet, used to restore the queue after a page refresh; returns `assistant_message_id` and `items[]` (`steer_id`, `content`, `delivery`, `mentioned_items`); `items` is empty when no round is running |
| DELETE | `/api/v1/sessions/:id/steer/:steer_id` | Withdraw a queued message |
| POST | `/api/v1/sessions/:session_id/steer/:steer_id/inject` | Change an `after` message to `inject` |

POST request body:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query` | string | Yes | Message content, at most 10000 characters |
| `delivery` | string | No | `after` (default): sent as the next round's question after the current round ends; `inject`: delivered to the agent at the next iteration boundary of the current round (including right before the final answer), and the agent adjusts accordingly and continues |
| `mentioned_items` | []object | No | @-mentioned items, same format as the chat request |
| `steer_id` | string | No | Client-generated UUID used to deduplicate retries; the same ID with different content returns 409 |
| `expected_assistant_message_id` | string | No | The ID of the running assistant message the client sees; returns 409 if the run has switched, and the client should retry |
| `channel` | string | No | Source channel |

`status` in the response:

| `status` | Meaning |
| --- | --- |
| `queued` | Queued; returns `steer_id`, `delivery`, and `assistant_message_id` |
| `new_run` | No round is currently running; the client should send it through a normal `agent-chat` call instead |
| `already_injected` | The message has already been delivered to the agent (can happen on retry, withdrawal, or change to inject) and cannot be withdrawn |
| `deleted` / `gone` | DELETE only: withdrawn (`removed` indicates whether a message was actually deleted) / no round is currently running |

At most 10 undelivered messages can be queued per round at the same time; beyond that, 400 is returned. After the current round ends normally, the server starts the next round directly with the first undelivered message as its question, and the remaining messages are carried into that round with their original delivery mode; this round has no corresponding `agent-chat` connection, so use the `assistant_message_id` returned by `GET /steer` to call continue-stream to receive it. When the user stops generation, all undelivered messages are discarded. Delivered messages are written into the session history, and the client is notified through the `user_message_injected` event. If querying the run state fails, 503 is returned and you can retry.

```bash
curl -X POST $BASE/api/v1/sessions/s-1/steer -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"query":"Only look at data from 2024","delivery":"inject"}'
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

Response: 200 SSE (`text/event-stream`; see the overview's "Streaming API protocol" for the event format). When replaying stored events, consecutive unfinished `answer`/`thinking`/`reflection` increments under the same event ID are merged into one frame, so clients that accumulate content by event ID get the same text; the live-push part is not merged.

```bash
curl -N "$BASE/api/v1/sessions/continue-stream/s-1?message_id=m-1" -H "Authorization: Bearer $TOKEN"
```

## Sandbox Graphical Desktop {#sandbox-desktop}

These routes are registered by `internal/router/routes_chat.go` and are not yet in Swagger. Only Cube/E2B desktop templates are supported; for deployment and proxy requirements, see [Sandbox Deployment](../06-development/04-sandbox-deployment.md).

### POST /api/v1/sessions/:session_id/sandbox/desktop-ticket

Issues a one-time WebSocket ticket valid for two minutes. Requires a valid login Bearer access token of the session owner; an API Key alone is not enough. The JWT is only placed in the authentication header of this POST.

```bash
curl -X POST "$BASE/api/v1/sessions/$SESSION_ID/sandbox/desktop-ticket" \
  -H "Authorization: Bearer $TOKEN"
```

Response: 200 `{"success":true,"data":{"ticket":"<opaque-ticket>","expires_in":120}}`.

### GET /api/v1/sessions/:id/sandbox/desktop

The WebSocket handshake uses `?ticket=<opaque-ticket>` and does not go through the regular JWT middleware. The ticket is bound to the user, space, session, and original access token, and becomes invalid after one use; used, expired, and unknown tickets are all rejected. Proxy logs must not record the ticket query.

Only one relay is allowed per session at a time. States such as the sandbox not being bound, being paused, not supporting a desktop, or failing to start may first complete the WebSocket upgrade and then disconnect with a close reason such as `SANDBOX_NOT_BOUND`, `SANDBOX_PAUSED`, `DESKTOP_UNSUPPORTED`, or `DESKTOP_START_FAILED`; clients should read the close reason.

### POST /api/v1/sessions/:session_id/sandbox/desktop/activity

The session owner reports keyboard and mouse activity; the response is 200 `{"success":true}`. It is used for renewal only when the server-side RFB parser has degraded; it is ignored when the parser works normally, and should not be used to extend the sandbox's lifetime by polling.

## Session Attachments (Temporary Documents)

Handler: `internal/handler/session/temporary_document.go`

### POST /api/v1/sessions/:session_id/attachments

Purpose: upload a session-level temporary document (parsed asynchronously). Multipart fields: `file` (required), `agent_id` (optional, determines the parsing engine/ASR model), `parser_engine` (optional; ignored when using a shared agent, in which case the agent's parsing rules decide).

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
| `query` | string | Yes (`binding:"required"`) | User's question; returns 400 when empty, whitespace-only, containing control characters, or invalid UTF-8 |
| `knowledge_base_ids` | []string | No | KBs to retrieve from |
| `knowledge_ids` | []string | No | Restrict to specific knowledge files |
| `agent_enabled` | bool | No | Whether to enable Agent mode |
| `agent_id` | string | No | Custom Agent ID |
| `agent_source_tenant_id` | uint64 | No | Source space of a shared agent, used to disambiguate when shared agents with the same name come from multiple spaces |
| `reasoning_effort` | string | No | Thinking intensity for this round: `off`, `auto`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max` (`none`/`false` are treated as `off`, `true`/`on` as `auto`); when omitted, the agent configuration is used. Applies only to this round and does not modify the agent; if the model does not support the selected level, it is automatically adjusted to a nearby level. Invalid values return 400 |
| `web_search_enabled` | bool | No | Web search; takes effect only when the agent itself has web search enabled |
| `local_browser_enabled` | bool | No | Allows this round to use the connected local browser; only valid for `agent-chat` with a smart-reasoning agent, otherwise 400 is returned. See [Local Browser](../05-clients/09-local-browser.md) |
| `summary_model_id` | string | No | Summarization model; ignored when using a shared agent, which always uses the model configured on the agent |
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
| `question_origin` | object | No | Where the suggested question the user clicked came from: `knowledge_base_id`, optional `knowledge_id`. Used only in smart-reasoning mode, where the agent retrieves from that source first; ignored when the source is outside this round's retrieval scope, so the scope is never widened |

The selected `reasoning_effort` is written to the session's `last_request_state.reasoning_effort`, and the frontend restores it when the session is reopened.

Response: 200 SSE stream, `event: message` + `data: StreamResponse` (see overview), ending with a `complete` event. When the answer is truncated by the model's single-output limit, the `answer` event carries `data.truncated: true`, and the content generated before truncation is kept as usual.

```bash
curl -N -X POST $BASE/api/v1/knowledge-chat/s-1 -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"What is the refund policy?","knowledge_base_ids":["kb-1"]}'
```

### POST /api/v1/agent-chat/:session_id

Purpose: Agent Q&A (SSE streaming, including `thinking/tool_call/tool_result/tool_approval_required/mcp_oauth_required` and other events). Request body is the same as above.

- 工具执行失败以 `tool_result` 事件返回（`data.success: false`，`data.error` 为原因），智能体会继续处理；`error` 事件只表示整轮执行失败。
- 智能体中途接收追加消息时发出 `user_message_injected`；命令执行过程中以 `command_output` 更新工具卡片输出。
- 同一会话已有智能推理回答在生成时返回 409，此时应改用 [steer 接口](#steer)追加消息。

```bash
curl -N -X POST $BASE/api/v1/agent-chat/s-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"query":"Analyze last quarter data","agent_id":"agent-1"}'
```

### POST /api/v1/knowledge-search

Purpose: sessionless knowledge retrieval (non-streaming), the preferred endpoint for external systems to get retrieval results. It follows the same retrieval flow as Q&A in the product (recall → rerank → merge → truncate), with the same ranking as Q&A on the page. For choosing between this and `hybrid-search`, see [Choosing a retrieval API](./01-api-overview.md#retrieval-api). Handler: `SearchKnowledge` in `internal/handler/session/qa.go`.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `query` | string | Yes (`binding:"required"`) | Query |
| `knowledge_base_id` | string | No | Single KB (legacy compatibility) |
| `knowledge_base_ids` | []string | No | Multiple KBs |
| `knowledge_ids` | []string | No | Restrict to specific files |
| `tag_ids` | []string | No | Tag filter |
| `mentioned_items` | []object | No | Tag mentions scoped to a KB |
| `vector_threshold` / `keyword_threshold` | float | No | Recall thresholds; when omitted, the space's retrieval configuration is used (defaults 0.15 / 0.3) |
| `match_count` | int | No | Number of results; when omitted, the space's configured `rerank_top_k` is used (default 10). The recall depth is automatically raised to at least this value |
| `disable_keywords_match` / `disable_vector_match` | bool | No | Turn off one recall path; setting both to `true` returns 400 |
| `rerank` | object | No | Overrides the rerank settings; `{"enabled":false}` disables rerank. See [the rerank object](./01-api-overview.md#retrieval-api) for the fields. When `rerank.top_k` is also given, it takes precedence over `match_count` |

Omitted fields all fall back to the space's retrieval configuration (`GET /tenants/kv/retrieval-config`); when none of the new fields are passed, the behavior is the same as before.

Response: 200 `{"success":true,"data":[SearchResult],"meta":{"rerank":{...}}}`. `SearchResult` contains `id,content,knowledge_id,knowledge_title,score,chunk_type,knowledge_base_id,...`; reranked results carry `model_score` and `base_score` in `metadata`. When `data` is empty, check `meta.rerank.outcome` for the reason; see [meta.rerank diagnostics](./01-api-overview.md#retrieval-api).

```bash
curl -X POST $BASE/api/v1/knowledge-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' -d '{"query":"Deployment requirements","knowledge_base_ids":["kb-1"]}'

# Vector recall only, 5 results, with rerank disabled
curl -X POST $BASE/api/v1/knowledge-search -H "X-API-Key: $API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"query":"Deployment requirements","knowledge_base_ids":["kb-1"],"disable_keywords_match":true,"match_count":5,"rerank":{"enabled":false}}'
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

## Session-Generated Files

The following endpoints require Viewer+; an API Key needs chat or full-access, and access is checked against session ownership. Sessions that do not exist or are inaccessible return 404.

| Method | Path | Response |
| --- | --- | --- |
| GET | `/api/v1/sessions/:id/artifacts` | 200 `{success:true,data:[Artifact]}`, aggregating the session's files |
| GET | `/api/v1/sessions/:id/messages/:message_id/artifacts` | Same as above, only this message's files |
| GET | `/api/v1/sessions/:id/messages/:message_id/artifacts/:index/download` | 200 file stream, Content-Disposition: attachment |
| DELETE | `/api/v1/sessions/:id/messages/:message_id/artifacts/:index` | 200 `{success:true,data:{file_name,deleted}}`, deletes the file |

Artifact fields: index, handle (optional resource:// reference), file_name, file_type, file_size, source_path, mod_time, created_at. The response does not return the underlying object storage URL. The download index starts at 0 and must be the index from the corresponding message's list; do not use the session-level aggregate index to build a message download URL. Invalid indexes return 400; out-of-range indexes or missing files return 404.

```bash
curl "$BASE/api/v1/sessions/session-1/messages/message-1/artifacts" \
  -H "Authorization: Bearer $TOKEN"
curl "$BASE/api/v1/sessions/session-1/messages/message-1/artifacts/0/download" \
  -H "Authorization: Bearer $TOKEN" -o result.pdf
```

### Deleting Generated Files

Deletion reclaims the bytes in object storage and is **irreversible**. Unlike download, deletion is only open to the session owner: read-only access obtained through a shared agent can download files but not delete them — sessions that don't belong to you are always treated as 404 (no distinction between "does not exist" and "no permission", consistent with the other session endpoints). A deleted file returns 404, and deleting it again also returns 404.

The bytes are only actually reclaimed when there are no other holders: a file that has been saved to a knowledge base, re-referenced by a later answer, or copied into a fork of its session is kept. A reclamation failure does not affect the deletion result (the endpoint still returns 200), and the file has disappeared from every list.

After deletion, the file no longer appears in the list endpoints or the artifact library, but its **position in the message is preserved**: `index` is the download address, and if later files shifted forward, existing download links would point to the wrong file. For the same reason, a file with the same name in the sandbox is not re-collected in the next round — its mtime has not changed just because the user deleted it.

| Parameter | Description |
| --- | --- |
| `all_versions` | Also delete all historical versions with the same `source_path` in this session. Boolean, default `false` |

```bash
curl -X DELETE "$BASE/api/v1/sessions/session-1/messages/message-1/artifacts/0" \
  -H "Authorization: Bearer $TOKEN"
```

### Cross-Session Artifact List

`GET /api/v1/artifacts` lists all generated files in the current user's web conversations, used by the "Artifacts" page in the homepage sidebar. The scope matches `source=web` in the session list: the user's own sessions and historical tenant-level web sessions without an owner; IM channel, web widget (embed), and API Key sessions are never included, even if an IM session has no owner in the database. Files with the same `source_path` in the same session are treated as multiple versions of one file; only the latest version is returned, and `version_count` gives the number of versions. Files in deleted sessions or messages are not returned. Permission requirements are the same as in the table above.

| Parameter | Description |
| --- | --- |
| `keyword` | Filter by file name, case-insensitive |
| `file_types` | Comma-separated extensions, such as `.pdf,.pptx` (the dot can be omitted) |
| `page` / `page_size` | Pagination; `page_size` maximum 100, default 20 |

Response `{success, data:[LibraryArtifact], total, page, page_size}`, in reverse order of generation time. LibraryArtifact fields: session_id, session_title, message_id, index, handle (optional), file_name, file_type, file_size, source_path, created_at, version_count. To download, use its session_id, message_id, and index with the download endpoint in the table above.

```bash
curl "$BASE/api/v1/artifacts?file_types=.pptx,.pdf&keyword=report&page=1&page_size=30" \
  -H "Authorization: Bearer $TOKEN"
```

`DELETE /api/v1/artifacts` deletes a file from the artifact library, located by the query parameters `session_id`, `message_id`, and `index`, with the same semantics as the in-session deletion above. The only difference is that `all_versions` defaults to `true` when omitted (when a value is passed explicitly, both endpoints parse it the same way): a row in the artifact library represents a file (`version_count` gives the number of versions) rather than one generation, and deleting only the latest version would leave the row in the list, showing the previous version. Pass `all_versions=false` to delete only the current version.

```bash
curl -X DELETE "$BASE/api/v1/artifacts?session_id=session-1&message_id=message-1&index=0" \
  -H "Authorization: Bearer $TOKEN"
```

### Images and File References in Answers

`GET /api/v1/sessions/:id/messages/:message_id/files?file_path=...` is a message-level authenticated proxy. Pass as file_path a resource handle or supported storage reference cited by that message; clients should URL-encode it. The backend verifies access to the message, the binding between the resource and the message, and current access to the knowledge base / shared Agent; arbitrary file paths cannot be accessed just by knowing the session ID. After authorization is revoked, references in old messages are rejected as well. This applies to shared Agents, answer images from organization-shared knowledge bases, and message artifacts; see [File Access](../03-features/21-file-access.md).

### Per-Turn Usage

Messages return the persisted usage, and the Agent completion event carries turn_usage, containing the aggregated results of this round's model calls for each purpose. Tool calls themselves do not all produce tokens; usage is based on what the provider returns or what the backend has collected. See [Observability](../03-features/16-observability.md) for the fields.

## Implementation Reference

Route registration: `RegisterSessionRoutes`, `RegisterChatRoutes`, `RegisterMessageRoutes` in `internal/router/routes_chat.go` (called by `internal/router/router.go`). Handlers: `internal/handler/session/` (handler.go, qa.go, stream.go, title.go, temporary_document.go, fork.go, rewind.go, steer.go, artifact_*.go), `internal/handler/message.go`, `internal/handler/message_suggestion.go`.
