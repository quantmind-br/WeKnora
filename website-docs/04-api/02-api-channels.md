# API Reference: IM, Embed, and File Services

Manages IM and web embed channels, and provides channel callback, visitor session, and file access endpoints. The admin side, IM platform callbacks, and Embed visitors each use their own authentication method.

## IM Callback (no global authentication)

### GET|POST /api/v1/im/callback/:channel_id

Purpose: Event callback and URL verification for IM platforms (WeChat/Feishu/Slack/Telegram/DingTalk/QQBot/Yunzhijia, etc.). Registered before the authentication middleware, using each platform's own signature verification; returns 403 on signature verification failure, 404 if the channel does not exist. Messages are ACKed immediately upon receipt and processed asynchronously. Handler: `internal/handler/im.go`

Response: 200 `{"success":true}` or the ACK format required by the platform.

```bash
curl -X POST $BASE/api/v1/im/callback/ch-1 -H 'Content-Type: application/json' -d '{"event":"..."}'
```

## IM Channel Management (authentication required)

API key: `manage_channels`/full. IM channels carry external bot credentials: listing requires Viewer+, changes/toggling/QR login require Admin+.

For Feishu/Lark, credentials.api_base_url affects both the HTTP API and the WebSocket bootstrap; Yunzhijia supports session_mode=thread. See [IM Integration](../03-features/12-im-integration.md) for configuration examples and network requirements. The memory preference of IM/Embed comes from config.memory_enabled of the bound Agent; the channel endpoints currently have no separate memory_enabled parameter.

### POST /api/v1/agents/:id/im-channels

Purpose: Create an IM channel for an Agent. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `platform` | string | Yes | `wecom/feishu/lark/slack/telegram/dingtalk/mattermost/wechat/qqbot/yunzhijia` |
| `name` | string | No | Display name |
| `mode` | string | No | `websocket` (default; mattermost/yunzhijia default to `webhook`)/`webhook`/`longpoll` (wechat forces longpoll) |
| `output_mode` | string | No | `stream` (default)/`full` (wechat forces full) |
| `locale` | string | No | Reply language: `zh-CN`/`en-US`/`ja-JP`/`ko-KR`/`ru-RU`; empty (default) uses the deployment default language (`WEKNORA_LANGUAGE`, `zh-CN` when unset); other values return 400 |
| `session_mode` | string | No | `user` (default)/`thread` |
| `knowledge_base_id` | string | No | KB into which attachments are additionally ingested; must belong to this space, otherwise 400 |
| `credentials` | object | No | Platform credentials |
| `enabled` | bool | No | Default true |

Response: 200 `{"data":{IMChannel}}`; returns 409 if a bot already exists on the same channel.

```bash
curl -X POST $BASE/api/v1/agents/agent-1/im-channels -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"platform":"feishu","name":"Feishu Support","locale":"en-US"}'
```

### GET /api/v1/agents/:id/im-channels

Purpose: List IM channels for an Agent (summary). Permission: Viewer+.

Response: 200 `{"data":[IMChannel]}`

```bash
curl $BASE/api/v1/agents/agent-1/im-channels -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/im-channels

Purpose: Overview of all IM channels across the space (excluding credentials). Permission: Viewer+.

Response: 200 `{"data":[IMChannel]}`

```bash
curl $BASE/api/v1/im-channels -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/im-channels/:id

Purpose: Update a channel (partial update: `name/mode/output_mode/locale/session_mode/knowledge_base_id/credentials/enabled/agent_id` are all optional; passing an empty string for `knowledge_base_id` removes the association, and passing an empty string for `locale` restores the default language). Permission: Admin+.

Response: 200 `{"data":{IMChannel}}`

```bash
curl -X PUT $BASE/api/v1/im-channels/ch-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":false}'
```

### DELETE /api/v1/im-channels/:id

Purpose: Delete a channel. Permission: Admin+.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/im-channels/ch-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/im-channels/:id/toggle

Purpose: Enable/disable toggle. Permission: Admin+. No request body.

Response: 200 `{"data":{IMChannel}}`

```bash
curl -X POST $BASE/api/v1/im-channels/ch-1/toggle -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/wechat/qrcode

Purpose: Generate a WeChat login QR code (bind a personal WeChat account to the space). Permission: Admin+. No request body. Handler: `internal/handler/wechat_qrcode.go`

Response: 200 `{"data":{"qrcode_url","qrcode"}}`

```bash
curl -X POST $BASE/api/v1/wechat/qrcode -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/wechat/qrcode/status

Purpose: Poll QR code scan status; returns credentials once confirmed. Permission: Admin+. Request body: `{"qrcode":"<identifier>"}` (required).

Response: 200 `{"data":{"status":"pending|scanned|confirmed|expired","credentials":{bot_token,ilink_bot_id,ilink_user_id,baseurl}}}`

```bash
curl -X POST $BASE/api/v1/wechat/qrcode/status -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"qrcode":"qr-1"}'
```

## Embed Channel Management (authentication required)

API key: `manage_channels`/full. Handler: `internal/handler/embed_channel.go`

### POST /api/v1/agents/:id/embed-channels

Purpose: Create a web embed channel for an Agent. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | No | Name |
| `enabled` | bool | No | Default true |
| `allowed_origins` | []string | Yes | At least one origin (exact URL / `*.domain`; `*` forbidden in production) |
| `welcome_message` | string | No | Welcome message |
| `rate_limit_per_minute` | int | No | Per IP/minute, default 30 |
| `rate_limit_per_day` | int | No | Per channel/day, default 10000 |
| `primary_color` / `page_title` / `widget_position` | string | No | Appearance (position: `bottom-right` default, plus the other three corners) |
| `header_title_mode` | string | No | `channel` (default)/`session` |
| `show_suggested_questions` | bool | No | Default true |
| `allow_web_search` / `allow_file_upload` | bool | No | Default false |
| `default_locale` | string | No | `zh-CN/en-US/ko-KR/ja-JP/ru-RU`/empty (follows browser) |
| `webhook_url` / `webhook_secret` | string | No | Visitor event webhook |
| `agent_id` | string | No | Bound Agent |

Response: 201 `{"success":true,"data":{embedChannelResponse including publish_token}}`

```bash
curl -X POST $BASE/api/v1/agents/agent-1/embed-channels -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"Website Support","allowed_origins":["https://example.com"]}'
```

### GET /api/v1/agents/:id/embed-channels

Purpose: List embed channels for an Agent. Permission: Viewer+.

Response: 200 `{"success":true,"data":[embedChannelResponse]}`

```bash
curl $BASE/api/v1/agents/agent-1/embed-channels -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/embed-channels

Purpose: List embed channels across the whole space (excluding publish token). Permission: Viewer+.

Response: 200 `{"success":true,"data":[embedChannelResponse]}`

```bash
curl $BASE/api/v1/embed-channels -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/embed-channels/:channel_id

Purpose: Channel details (including publish token, for copying deployment code). Permission: Viewer+.

Response: 200 `{"success":true,"data":{embedChannelResponse}}`

```bash
curl $BASE/api/v1/embed-channels/ec-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/embed-channels/:channel_id

Purpose: Update a channel (same fields as creation, all optional). Permission: Admin+.

Response: 200 `{"success":true,"data":{embedChannelResponse}}`

```bash
curl -X PUT $BASE/api/v1/embed-channels/ec-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":false}'
```

### DELETE /api/v1/embed-channels/:channel_id

Purpose: Delete a channel. Permission: Admin+.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/embed-channels/ec-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/embed-channels/:channel_id/rotate-token

Purpose: Rotate the publish token (old token becomes invalid). Permission: Admin+. No request body.

Response: 200 `{"success":true,"data":{embedChannelResponse including new publish_token}}`

```bash
curl -X POST $BASE/api/v1/embed-channels/ec-1/rotate-token -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/embed-channels/:channel_id/preview-session

Purpose: Issue a short-lived session token for admin-side preview (no publish token needed). Permission: Viewer+.

Response: 200 `{"success":true,"data":{"session_token","expires_in"}}`; returns 403 if the channel is disabled.

```bash
curl -X POST $BASE/api/v1/embed-channels/ec-1/preview-session -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/embed-channels/:channel_id/stats

Purpose: Channel usage statistics. Permission: Viewer+.

Response: 200 `{"success":true,"data":{"session_count":N}}`

```bash
curl $BASE/api/v1/embed-channels/ec-1/stats -H "Authorization: Bearer $TOKEN"
```

## Embed Page Frame Policy

### GET /api/v1/embed-frame-policy

Purpose: Lets an Nginx `auth_request` subrequest fetch the channel's `Content-Security-Policy: frame-ancestors` before serving the `/embed/:channel_id` page. No token required; it returns only the policy header, not the channel configuration; the response carries `Cache-Control: no-store`.

| Request header | Required | Description |
| --- | --- | --- |
| `X-Embed-Page-URI` | Yes | Relative URI of the embed page, such as `/embed/<channel_id>` |

Response: 204 with the CSP header; 403 when the channel doesn't exist, is disabled, the path is invalid, or the allowlist is empty.

```bash
curl -i $BASE/api/v1/embed-frame-policy -H "X-Embed-Page-URI: /embed/ec-1"
```

## Embed Public Routes (/api/v1/embed/:channel_id, EmbedAuth)

Authentication: `Authorization: Embed <publish_token|session_token>`; session-level operations additionally require `X-Embed-Session: <sig>`. See the overview for rate limiting and Origin validation. Handler: `internal/handler/embed_channel.go`; middleware: `internal/middleware/embed_auth.go`

Below, `$ET` denotes the Embed token header: `-H "Authorization: Embed $EMBED_TOKEN"`.

### POST /api/v1/embed/:channel_id/exchange

Purpose: Exchange a publish token for a short-lived session token (a session token cannot be exchanged again). No request body.

Response: 200 `{"success":true,"data":{"session_token","expires_in"}}`

```bash
curl -X POST $BASE/api/v1/embed/ec-1/exchange -H "Authorization: Embed $PUBLISH_TOKEN"
```

### GET /api/v1/embed/:channel_id/config

Purpose: Public configuration for the channel (no secrets).

Response: 200 `{"success":true,"data":{channel_id,name,display_title,knowledge_base_ids,agent_id,agent_name,welcome_message,primary_color,widget_position,allow_web_search,allow_file_upload,default_locale,...}}`

```bash
curl $BASE/api/v1/embed/ec-1/config -H "Authorization: Embed $EMBED_TOKEN"
```

### GET /api/v1/embed/:channel_id/suggested-questions

Purpose: Initial suggested questions. Query parameter: `limit` (≤12).

Response: 200 `{"success":true,"data":{"questions":[...]}}`

```bash
curl "$BASE/api/v1/embed/ec-1/suggested-questions?limit=6" -H "Authorization: Embed $EMBED_TOKEN"
```

### GET /api/v1/embed/:channel_id/chunks/:chunk_id

Purpose: View a referenced chunk (content redacted; returns 403 on unauthorized access).

Response: 200 `{"success":true,"data":{chunk}}`

```bash
curl $BASE/api/v1/embed/ec-1/chunks/c-1 -H "Authorization: Embed $EMBED_TOKEN"
```

### POST /api/v1/embed/:channel_id/sessions

Purpose: Create a visitor session, returning a session ID and signed handle. No request body.

Response: 201 `{"success":true,"data":{"id":"<session_id>","sig":"<signature>"}}`

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions -H "Authorization: Embed $EMBED_TOKEN"
```

### POST /api/v1/embed/:channel_id/knowledge-chat/:session_id

Purpose: Visitor knowledge Q&A (SSE; the payload is rewritten per channel constraints then delegated to KnowledgeQA). Requires `X-Embed-Session`. Request body same as `/knowledge-chat` (`query` required).

```bash
curl -N -X POST $BASE/api/v1/embed/ec-1/knowledge-chat/s-1 \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"query":"What are your business hours?"}'
```

### POST /api/v1/embed/:channel_id/agent-chat/:session_id

Purpose: Visitor Agent Q&A (SSE). Requires `X-Embed-Session`. Same request body as above.

```bash
curl -N -X POST $BASE/api/v1/embed/ec-1/agent-chat/s-1 \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"query":"Help me place an order"}'
```

### GET /api/v1/embed/:channel_id/messages/:session_id/load

Purpose: Load visitor session messages (delegates to `LoadMessages`, query parameters `limit/before_time`). Requires `X-Embed-Session`.

Response: 200 `{"success":true,"data":[Message]}`

```bash
curl "$BASE/api/v1/embed/ec-1/messages/s-1/load?limit=20" \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG"
```

### POST /api/v1/embed/:channel_id/sessions/:session_id/stop

Purpose: Stop generation (delegates to StopSession; request body `{"message_id":"..."}`). Requires `X-Embed-Session`.

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/stop \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"message_id":"m-1"}'
```

### GET|POST /api/v1/embed/:channel_id/sessions/:session_id/messages/:message_id/suggestions

Purpose: Read / trigger generation of message suggestions (returns `suppressed` if the channel has disabled suggestions). Requires `X-Embed-Session`.

Response: 200 `{"success":true,"data":{"status","questions":[...]}}`

```bash
curl $BASE/api/v1/embed/ec-1/sessions/s-1/messages/m-1/suggestions \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG"
```

### POST /api/v1/embed/:channel_id/sessions/:session_id/suggestion-events

Purpose: Report a suggestion interaction event (delegates to RecordEvent, same fields as the authenticated version). Requires `X-Embed-Session`. Response: 204.

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/suggestion-events \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"suggestion_set_id":"ss-1","event_type":"impression"}'
```

### POST /api/v1/embed/:channel_id/sessions/:session_id/events

Purpose: Forward a visitor event to the channel's webhook. Requires `X-Embed-Session`.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `type` | string | Yes | `message_sent` / `message_received` |
| `query` / `content` | string | No | User question / bot response |

Response: 200 `{"success":true}`; returns 400 for unsupported types.

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/events \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"type":"message_sent","query":"Hello"}'
```

### MCP OAuth and Tool Approval (visitor side)

All routes below require `X-Embed-Session` and delegate to the corresponding authenticated handler (`internal/handler/mcp_oauth.go`, `internal/handler/mcp_service.go`):

| Method + Path | Purpose |
| --- | --- |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/mcp-oauth-resolutions/:pending_id` | Resume a run paused for OAuth (body: `service_id` required, `decision` optional) |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/mcp-oauth-resolutions/:pending_id/cancel` | Cancel the OAuth flow |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/mcp-services/:id/oauth/authorize-url` | Generate an authorization URL (body: `redirect_uri` required) |
| `GET /api/v1/embed/:channel_id/sessions/:session_id/mcp-services/:id/oauth/status` | Query authorization status |
| `POST /api/v1/embed/:channel_id/sessions/:session_id/tool-approvals/:pending_id` | Tool approval (body: `decision` required) |

```bash
curl -X POST $BASE/api/v1/embed/ec-1/sessions/s-1/tool-approvals/p-1 \
  -H "Authorization: Embed $EMBED_TOKEN" -H "X-Embed-Session: $SIG" \
  -H 'Content-Type: application/json' -d '{"decision":"approve"}'
```

### GET /api/v1/embed/:channel_id/files

Purpose: Visitor-side image proxy (images embedded in bot replies; EmbedAuth injects the channel's tenant, and the handler enforces the same-tenant path). Query parameter: `file_path` (required).

Response: 200 file stream.

```bash
curl "$BASE/api/v1/embed/ec-1/files?file_path=local://1/exports/chart.png" \
  -H "Authorization: Embed $EMBED_TOKEN" -o chart.png
```

## File Services

Implemented in `internal/router/files.go` (not in the handler package).

### GET /files

Purpose: Unified authenticated file proxy (local/MinIO/COS/TOS, etc.). Permission: any authenticated space member; API keys require non-KB-restricted access (full-access or space-wide retrieve, `middleware.AllowFileServeAPIKey()`); the path is enforced to be within the same tenant (`ValidateStoragePathTenant`).

| Query Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_path` | string | Yes | `provider://...` path (`..` forbidden; cross-tenant access returns 403) |

Response: 200 file stream (`X-Content-Type-Options: nosniff`; non-whitelisted types are forced to `Content-Disposition: attachment`).

```bash
curl "$BASE/files?file_path=local://1/docs/a.png" -H "Authorization: Bearer $TOKEN" -o a.png
```

### GET|HEAD /api/v1/files/presigned

Purpose: HMAC-signed URL file access (for images embedded in IM platforms; no authentication required, uses signature verification + expiration check, with `SYSTEM_AES_KEY` participating in the signature).

| Query Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `file_path` | string | Yes | Storage path |
| `tenant_id` | uint64 | Yes | Tenant ID |
| `expires` | string | Yes | Unix expiration timestamp |
| `sig` | string | Yes | HMAC signature |

Response: 200 file stream (HEAD returns headers only); 403 if the signature is invalid or expired.

```bash
curl "$BASE/api/v1/files/presigned?file_path=local://1/x.png&tenant_id=1&expires=1790000000&sig=abc" -o x.png
```

### GET /api/v1/files/presigned-preview

Purpose: Diagnostic endpoint: returns the presigned HTTP URL that would be generated for a given path. Permission: Admin+, explicitly denies API key principals (`DenyAPIKeyPrincipal`). Query parameter: `file_path` (required).

`file_path` must belong to the current space, just like `/files`: a `resource://` handle requires the resource to belong to this space, and a storage path requires its tenant segment to be this space's ID; otherwise 403 is returned and no URL is issued.

Response: 200 `{"file_path","provider","url","rewritten":bool,"hint"}`

```bash
curl "$BASE/api/v1/files/presigned-preview?file_path=local://1/x.png" -H "Authorization: Bearer $TOKEN"
```

### GET|HEAD /r/:token

Purpose: Short-lived resource-authorization URL (for clients such as IM platforms that cannot carry authentication headers). No authentication required; the token itself is the capability credential; returns 404 if invalid or expired.

Response: 200 file stream (`Cache-Control: private, max-age=300`).

```bash
curl $BASE/r/abc123 -o file.png
```

## Implementation Reference

Route registration: `RegisterIMRoutes`, `RegisterIMChannelRoutes`, `RegisterEmbedChannelRoutes`, and `RegisterEmbedPublicRoutes` in `internal/router/routes_agent.go`; `serveFilesWithResources`, `servePresignedFiles`, `servePresignedPreview`, and `serveResourceGrants` in `internal/router/files.go`. Handlers: `internal/handler/im.go`, `internal/handler/wechat_qrcode.go`, `internal/handler/embed_channel.go`.
