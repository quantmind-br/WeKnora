# API Reference: Agent, MCP & Skills

Route registration: `RegisterCustomAgentRoutes`, `RegisterMCPServiceRoutes`, `RegisterSkillRoutes`, `RegisterUserFavoriteRoutes` in `internal/router/router.go`. Handlers: `internal/handler/custom_agent.go`, `internal/handler/mcp_service.go`, `internal/handler/mcp_credentials.go`, `internal/handler/mcp_oauth.go`, `internal/handler/skill_handler.go`, `internal/handler/user_resource_favorite.go`.

## Agent (/api/v1/agents)

Read: Viewer+ (API key `read_agents`/`manage_agents`/`chat`/full); Write: Creator OR Admin+ (API key `manage_agents`/full); built-in Agents (`is_builtin=true`) always require Admin+.

### GET /api/v1/agents/placeholders

Purpose: prompt placeholder definitions (must be registered before `/:id`). Permission: Viewer+.

Response: 200 `{"success":true,"data":{"all":{...},"system_prompt":{...},"agent_system_prompt":{...},"context_template":{...},"rewrite_system_prompt":{...},"rewrite_prompt":{...},"fallback_prompt":{...}}}`

```bash
curl $BASE/api/v1/agents/placeholders -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/agents/type-presets

Purpose: presets for smart-reasoning Agent types (rag-qa / wiki-qa / hybrid / custom, etc.). Permission: Viewer+.

Response: 200 `{"success":true,"data":[{type,system_prompt,allowed_tools,kb_compatibility}]}`

```bash
curl $BASE/api/v1/agents/type-presets -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/agents

Purpose: create a custom Agent. Permission: Contributor+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required"`) | Name |
| `description` | string | No | Description |
| `avatar` | string | No | Avatar/emoji |
| `config` | object | No | Agent configuration (`types.CustomAgentConfig`, see below) |

Main `config` fields: `agent_mode` (`quick-answer`/`smart-reasoning`), `agent_type` (`rag-qa/wiki-qa/hybrid-rag-wiki/data-analysis/custom`), `system_prompt`, `model_id`, `temperature` (0-2, returns code 2103 if invalid), `max_iterations` (1-20, returns code 2102 if invalid), `allowed_tools` (at least one required for smart reasoning, code 2101), `mcp_selection_mode`/`mcp_services`, `skills_selection_mode`, `kb_selection_mode`/`knowledge_bases`, `web_search_enabled`, `question_suggestions`, etc. (full definition in `internal/types/custom_agent.go`).

Response: 201 `{"success":true,"data":{id,name,description,avatar,is_builtin,created_by,config,creator_name,...}}`

```bash
curl -X POST $BASE/api/v1/agents -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"Support Assistant","config":{"agent_mode":"quick-answer","kb_selection_mode":"selected","knowledge_bases":["kb-1"]}}'
```

### GET /api/v1/agents

Purpose: list Agents (including built-in ones). Permission: Viewer+. Query parameters: `creator` (`mine`/`others`, optional).

Response: 200 `{"success":true,"data":[Agent],"disabled_own_agent_ids":[...]}`

```bash
curl $BASE/api/v1/agents -H "X-API-Key: $API_KEY"
```

### GET /api/v1/agents/:id

Purpose: Agent details. Permission: Viewer+.

Response: 200 `{"success":true,"data":{Agent}}`

```bash
curl $BASE/api/v1/agents/agent-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/agents/:id

Purpose: update an Agent. Permission: Creator OR Admin+. Request body: `name/description/avatar/config` (all optional).

Response: 200 `{"success":true,"data":{Agent}}`

```bash
curl -X PUT $BASE/api/v1/agents/agent-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"description":"Updated description"}'
```

### DELETE /api/v1/agents/:id

Purpose: delete an Agent. Permission: Creator OR Admin+.

Response: 200 `{"success":true,"message":"Agent deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/agents/agent-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/agents/:id/copy

Purpose: duplicate an Agent (the copy belongs to the caller). Permission: Contributor+. No request body.

Response: 201 `{"success":true,"data":{new Agent}}`

```bash
curl -X POST $BASE/api/v1/agents/agent-1/copy -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/agents/:id/suggested-questions

Purpose: Agent starter suggested questions (registered outside the group to avoid conflicting with `/agents/:id/shares`). Permission: Viewer+; API key `read_agents`/`manage_agents`/`chat`/full.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `knowledge_base_ids` | string | No | Comma-separated KB IDs |
| `knowledge_ids` | string | No | Comma-separated knowledge IDs |
| `tag_scopes` | string | No | JSON array of tag scopes |
| `limit` | int | No | Cap of 30 |

Response: 200 `{"success":true,"data":{"questions":[{question,source,knowledge_base_id}]}}`

```bash
curl "$BASE/api/v1/agents/agent-1/suggested-questions?limit=6" -H "X-API-Key: $API_KEY"
```

## MCP Services (/api/v1/mcp-services)

Space-level external tool service integrations. Read: Viewer+; Write/test/approval policy: Admin+. API key: `manage_mcp_services`/full. Handler: `internal/handler/mcp_service.go`

### POST /api/v1/mcp-services

Purpose: create an MCP service. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Name |
| `description` | string | No | Description |
| `enabled` | bool | No | Enabled |
| `transport_type` | string | Yes | `sse` / `http-streamable` / `stdio` |
| `url` | *string | No | Service URL (SSE/HTTP) |
| `headers` | map[string]string | No | HTTP headers |
| `auth_config` | object | No | `auth_type` (`api_key/bearer/oauth`), `api_key_header`, `custom_headers`, `scopes`, `auth_server_metadata_url` (secrets go through the credentials sub-resource) |
| `advanced_config` | object | No | Timeout/retries |
| `stdio_config` | object | No | stdio command and arguments |
| `env_vars` | map[string]string | No | Environment variables |

Response: 200 `{"success":true,"data":{MCPServiceResponse}}` (includes `credentials:{api_key:{configured},token:{configured}}`)

```bash
curl -X POST $BASE/api/v1/mcp-services -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"github","transport_type":"sse","url":"https://mcp.example.com/sse"}'
```

### GET /api/v1/mcp-services

Purpose: list MCP services. Permission: Viewer+. Response: 200 `{"success":true,"data":[MCPServiceResponse]}`

```bash
curl $BASE/api/v1/mcp-services -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id

Purpose: details. Permission: Viewer+. Response: 200 `{"success":true,"data":{MCPServiceResponse}}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/mcp-services/:id

Purpose: partial update (map semantics; `auth_config` may not carry api_key/token). Permission: Admin+. Fields same as creation (all optional).

Response: 200 `{"success":true,"data":{MCPServiceResponse}}`

```bash
curl -X PUT $BASE/api/v1/mcp-services/mcp-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":false}'
```

### DELETE /api/v1/mcp-services/:id

Purpose: delete. Permission: Admin+. Response: 200 `{"success":true,"message":"MCP service deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/mcp-services/mcp-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/mcp-services/:id/test

Purpose: connection test (probes the external service). Permission: Admin+. Response: 200 `{"success":true,"data":{"success","message","oauth_required","tools":[...],"resources":[...]}}`

```bash
curl -X POST $BASE/api/v1/mcp-services/mcp-1/test -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id/tools

Purpose: list tools. Permission: Viewer+. Response: 200 `{"success":true,"data":[{name,description,inputSchema,require_approval}]}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/tools -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id/resources

Purpose: list resources. Permission: Viewer+. Response: 200 `{"success":true,"data":[{uri,name,description,mimeType}]}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/resources -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/mcp-services/:id/credentials

Purpose: set secrets (`api_key`/`token`, pointer fields, omit to keep). Permission: Admin+. Handler: `internal/handler/mcp_credentials.go`

Response: 200 `{"success":true,"data":{"fields":{"api_key":{"configured"},"token":{"configured"}}}}`

```bash
curl -X PUT $BASE/api/v1/mcp-services/mcp-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"token":"ghp_..."}'
```

### DELETE /api/v1/mcp-services/:id/credentials/:field

Purpose: delete a credential field (`api_key` or `token`). Permission: Admin+. Response: 204.

```bash
curl -X DELETE $BASE/api/v1/mcp-services/mcp-1/credentials/token -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/mcp-services/:id/tool-approvals

Purpose: list tool manual-approval policies. Permission: Viewer+. Response: 200 `{"success":true,"data":[{service_id,tool_name,require_approval,...}]}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/tool-approvals -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/mcp-services/:id/tool-approvals/:tool_name

Purpose: set whether a given tool requires manual approval. Permission: Admin+. Request body: `{"require_approval":true}` (required).

Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/mcp-services/mcp-1/tool-approvals/create_issue \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"require_approval":true}'
```

## MCP OAuth

Handler: `internal/handler/mcp_oauth.go`

### GET /api/v1/mcp-oauth/callback

Purpose: third-party OAuth authorization callback (no auth required, authenticated via the single-use `state` parameter; registered outside the `/mcp-services` group). Query parameters: `code`, `state`, `error`.

Response: 302 redirect to the frontend (`#mcp_oauth_result=success` on success, `#mcp_oauth_error=<code>` on failure).

```bash
curl -i "$BASE/api/v1/mcp-oauth/callback?code=xxx&state=yyy"
```

### POST /api/v1/mcp-services/:id/oauth/authorize-url

Purpose: generate a user-level authorization URL. Permission: Viewer+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `redirect_uri` | string | Yes | Backend callback URL (absolute address) |
| `frontend_redirect` | string | No | Frontend redirect after callback (defaults to `/`) |

Response: 200 `{"success":true,"data":{"authorization_url","authorization_attempt"}}`

```bash
curl -X POST $BASE/api/v1/mcp-services/mcp-1/oauth/authorize-url -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"redirect_uri":"'$BASE'/api/v1/mcp-oauth/callback"}'
```

### GET /api/v1/mcp-services/:id/oauth/status

Purpose: query the caller's own authorization status. Permission: Viewer+. Query parameters: `authorization_attempt` (optional).

Response: 200 `{"success":true,"data":{"authorized","state":"authorized|pending","refresh_available","expires_at"}}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/oauth/status -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/mcp-services/:id/oauth/token

Purpose: revoke the caller's own OAuth token. Permission: Viewer+. Response: 204.

```bash
curl -X DELETE $BASE/api/v1/mcp-services/mcp-1/oauth/token -H "Authorization: Bearer $TOKEN"
```

## Agent Runtime Interaction (/api/v1/agent)

In-conversation manual approvals and OAuth resumption; permissions are all Viewer+ (only the session initiator has the context), API key access is denied by default.

### POST /api/v1/agent/tool-approvals/:pending_id

Purpose: decide on a pending tool call approval. Handler: `ResolveToolApproval` in `internal/handler/mcp_service.go`.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `decision` | string | Yes (`binding:"required"`) | `approve` / `reject` |
| `modified_args` | JSON | No | Modified tool arguments |
| `reason` | string | No | Reason |

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/agent/tool-approvals/p-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"decision":"approve"}'
```

### POST /api/v1/agent/mcp-oauth-resolutions/:pending_id

Purpose: resume an Agent run paused due to MCP OAuth. Handler: `internal/handler/mcp_oauth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `service_id` | string | Yes (`binding:"required"`) | MCP service ID |
| `decision` | string | No | `authorize` (default) / `cancel` |

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/agent/mcp-oauth-resolutions/p-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"service_id":"mcp-1"}'
```

### POST /api/v1/agent/mcp-oauth-resolutions/:pending_id/cancel

Purpose: cancel a paused OAuth flow. No request body.

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/agent/mcp-oauth-resolutions/p-1/cancel -H "Authorization: Bearer $TOKEN"
```

## Skills (/api/v1/skills)

### GET /api/v1/skills

Purpose: list preloaded skills (read-only). Permission: Viewer+, JWT only. Handler: `internal/handler/skill_handler.go`

Response: 200 `{"success":true,"data":[{name,description}],"skills_available":bool}`

```bash
curl $BASE/api/v1/skills -H "Authorization: Bearer $TOKEN"
```

## User Favorites (/api/v1/user/favorites)

Stored per user (not per resource creator); permissions are all Viewer+, JWT only (API key access denied by default). Handler: `internal/handler/user_resource_favorite.go`

### GET /api/v1/user/favorites

Purpose: list favorites. Query parameters: `type` (required, `kb` or `agent`).

Response: 200 `{"success":true,"data":[{type,id,created_at}]}`

```bash
curl "$BASE/api/v1/user/favorites?type=kb" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/user/favorites

Purpose: add a favorite. Request body: `{"type":"kb|agent","id":"<resource ID>"}` (both required).

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/user/favorites -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"type":"kb","id":"kb-1"}'
```

### DELETE /api/v1/user/favorites/:type/:id

Purpose: remove a favorite. Path parameters: `type`, `id`.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/user/favorites/kb/kb-1 -H "Authorization: Bearer $TOKEN"
```
