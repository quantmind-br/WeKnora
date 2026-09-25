# API Reference: Agent, MCP & Skills

Manage agents, MCP services and their credentials, skills, and resource favorites. An agent's tool scope and call-approval configuration are maintained through this group of endpoints.

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

Purpose: update an Agent. Permission: Creator OR Admin+. Request body: `name/description/avatar/config` (all optional). When the Agent is shared to an organization, knowledge bases newly added to its knowledge base scope must be ones the caller is allowed to share (the knowledge base creator or Admin+; changing it to `all` requires Admin+); otherwise 403.

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
| `description` | string | No | Legacy description, kept for compatibility; the management UI edits `usage_instructions` instead |
| `usage_instructions` | string | No | The service's purpose, applicable scenarios, and key constraints; the model uses it to decide when to use the service. The management UI requires it in step two |
| `enabled` | bool | No | Enabled |
| `transport_type` | string | Yes | `sse` / `http-streamable`; `stdio` is rejected for security reasons |
| `url` | *string | No | Service URL (SSE/HTTP) |
| `headers` | map[string]string | No | HTTP headers |
| `auth_config` | object | No | `auth_type` (`api_key/bearer/oauth`), `api_key_header`, `custom_headers`, `scopes`, `auth_server_metadata_url` (secrets go through the credentials sub-resource) |
| `advanced_config` | object | No | `{timeout,retry_count,retry_delay}`, defaults to 30 seconds / 3 times / 1 second; a `timeout` greater than 60 seconds also extends the Agent's wait window for a single call to this service's tools |
| `stdio_config` / `env_vars` | object | No | Kept only for compatibility with old data; stdio is disabled and these have no effect |

Response: 200 `{"success":true,"data":{MCPServiceResponse}}` (includes `credentials:{api_key:{configured},token:{configured}}`; services whose tool catalog has been synced also carry `catalog:{tool_count,stale,synced_at}`)

```bash
curl -X POST $BASE/api/v1/mcp-services -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"github","transport_type":"sse","url":"https://mcp.example.com/sse"}'
```

### GET /api/v1/mcp-services

Purpose: list MCP services. Permission: Viewer+. Response: 200 `{"success":true,"data":[MCPServiceResponse]}`

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `agent_id` | string | No | When passed together with `agent_source_tenant_id`, lists the MCP services that shared agent can @-mention |
| `agent_source_tenant_id` | int | No | Source space ID of the shared agent, same as the parameter of the same name in chat requests |

When both parameters are passed, the shared-agent path is used: only services that the agent specifies in `selected` mode in its source space and that are enabled are returned (`all`/`none` mode returns an empty list), and each item contains only the ID, name, description, usage instructions, transport type, enabled state, and tool catalog summary, without connection details such as the URL, request headers, or auth configuration; returns 403 when the caller is not allowed to use the agent. When only one or neither is passed, the services of the caller's own space are listed.

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

When `usage_instructions` is submitted, it must be a string that is non-empty after trimming leading and trailing whitespace, with at most 16000 characters. When only the connection or enabled state is changed, the field can be omitted and the existing value is kept.

Response: 200 `{"success":true,"data":{MCPServiceResponse}}`

```bash
curl -X PUT $BASE/api/v1/mcp-services/mcp-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":false}'
```

### POST /api/v1/mcp-services/:id/usage-instructions/generate

Purpose: generate concise usage instructions from the synced, non-stale MCP tool catalog. Permission: Admin+; API key needs `manage_mcp_services` or full.

Request: `{"language":"en-US"}`. Supports `zh-CN`, `en-US`, `ja-JP`, `ko-KR`, `ru-RU`; defaults to Simplified Chinese.

Uses the space's default chat model if available, otherwise the first available chat model. The input includes the service name, the server-side description, and the names and descriptions of enabled tools; OAuth catalogs follow the current user's authorization scope. It does not connect to the MCP service, call tools, or save the generated result automatically.

Response: 200 `{"success":true,"data":{"usage_instructions":"Query remote logs by module and time range; read the matching log when a query ID is already known."}}`. The target is a short 2–3 sentence description of at most 500 characters; users can edit it and save it via PUT. Returns 400 when the catalog is not synced or is stale, no tools are enabled, or no chat model is available.

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

### GET /api/v1/mcp-services/:id/metadata

Purpose: read the persisted tool catalog without connecting upstream. Permission: Viewer+; OAuth catalogs are isolated per currently effective authorization principal.

Response: 200 `{"success":true,"data":null}` means not synced; once synced, `data` is the catalog snapshot, containing server info, instructions, tools, and the sync time. After the connection configuration changes, the snapshot is marked `stale:true` and cannot be used to load runtime tools.

```bash
curl $BASE/api/v1/mcp-services/mcp-1/metadata -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/mcp-services/:id/metadata/refresh

Purpose: explicitly connect upstream, fetch the full tool catalog, and update it atomically. Catalogs with static authentication require Admin+; OAuth users can sync their own catalog (Viewer+). An API Key needs the MCP management capability.

The response is the updated catalog snapshot. On failure the previous snapshot is kept; returns 409 if the connection changed during the refresh, 400 if the catalog is invalid/too large or the upstream sync fails, and 503 if metadata storage is unavailable. It does not overwrite manually written usage instructions or per-tool enable/approval policies.

```bash
curl -X POST $BASE/api/v1/mcp-services/mcp-1/metadata/refresh -H "Authorization: Bearer $TOKEN"
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

Purpose: list tool enable/disable and manual-approval policies. Permission: Viewer+. Response: 200 `{"success":true,"data":[{service_id,tool_name,require_approval,enabled,...}]}`

```bash
curl $BASE/api/v1/mcp-services/mcp-1/tool-approvals -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/mcp-services/:id/tool-approvals/:tool_name

Purpose: update a tool's enabled (enable/disable) and require_approval (manual approval) settings. Permission: Admin+. At least one of the two must be provided; omitted fields keep their current values. Tools without a record are enabled and do not require approval by default.

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

## MCP Server Endpoints (/api/v1/mcp-endpoints) {#mcp-server-endpoints}

Manages the MCP endpoints the current space publishes; external MCP clients connect to `/mcp/:endpoint_id`. Read: Viewer+; write: Admin+. API key: `manage_channels`/full. Handler: `internal/handler/mcp_endpoint.go`. For purposes and tool descriptions, see [MCP Integration](../03-features/08-mcp.md#serving-external-clients).

| Method | Path | Description |
| --- | --- | --- |
| GET | `/mcp-endpoints` | Endpoint list (without tokens) |
| GET | `/mcp-endpoints/tools` | Tool catalog: `{groups,tools:[{name,group,destructive}],default_tools}` |
| POST | `/mcp-endpoints` | Create; 201, the response includes a one-time `token` |
| GET | `/mcp-endpoints/:endpoint_id` | Details |
| PUT | `/mcp-endpoints/:endpoint_id` | Partial update; omitted fields keep their current values |
| DELETE | `/mcp-endpoints/:endpoint_id` | Delete; clients using this endpoint stop working immediately |
| POST | `/mcp-endpoints/:endpoint_id/rotate-token` | Rotate the token; the response includes the new `token` and the old token stops working immediately |

Request fields (the same for create and update, all optional):

| Field | Type | Description |
| --- | --- | --- |
| `name` | string | Name; required on create |
| `description` | string | Description |
| `enabled` | bool | Defaults to true; once disabled, connections return 403 |
| `knowledge_base_ids` | string[] | Accessible knowledge bases; an empty array means all knowledge bases in the space |
| `tools` | string[] | Exposed tools, at least one; if omitted on create, `default_tools` (all read-only tools) is used |
| `default_agent_id` | string | Agent used by `ask`; empty means the built-in quick Q&A; internal built-in Agents cannot be selected |
| `rate_limit_per_minute` | int | Maximum tool calls per minute; 0 or omitted means 60, maximum 6000 |

The response `data` is `{id,tenant_id,name,description,enabled,token_hint,knowledge_base_ids,tools,default_agent_id,rate_limit_per_minute,path,last_used_at,created_at,updated_at}`; create and rotate additionally carry `token`. When calling with a restricted API Key, the capabilities required by the endpoint's knowledge bases and tools must not exceed that Key's scope; otherwise 403.

```bash
curl -X POST $BASE/api/v1/mcp-endpoints -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"Product Docs Assistant","knowledge_base_ids":["kb-1"],"tools":["search_knowledge","read_document","ask"]}'
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

## Skills, Sandbox, and Personal Variables

`GET /api/v1/skills?sandbox_config_id=...` returns the names/descriptions of skills available under the given configuration, plus skills_available; when `agent_id` + `agent_source_tenant_id` are passed, the result follows the shared agent's source space and its sandbox configuration; see the [Sandbox and Skills API](02-api-sandbox-skills.md#skills-in-sandbox). For the full endpoints covering catalog listing, installation, templates, progress, files, and personal variables, see the [Sandbox and Skills API](02-api-sandbox-skills.md).

The agent config adds `sandbox_config_id`; together with skills_selection_mode and selected_skills, it determines the available skills. Shell/file tools are registered according to backend capabilities; the old read_skill / execute_skill_script are no longer registered.

## Long-Term Memory

When `memory_enabled` in the agent config is nil, the space setting is inherited; false disables memory reads and writes for this agent. For personal management, topic/document preferences, export, and immediate consolidation, see the [Long-term Memory API](02-api-memory.md); for usage steps, see [Cross-session Long-term Memory](../03-features/23-memory.md).

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

## Implementation Reference

Route registration: called from `internal/router/router.go`; `RegisterCustomAgentRoutes`, `RegisterSkillRoutes`, and `RegisterUserFavoriteRoutes` are defined in `routes_agent.go`, `RegisterMCPServiceRoutes` (including MCP OAuth and the `/agent` runtime interaction) in `routes_infra.go`, and `RegisterMCPEndpointRoutes` and the public `/mcp/:endpoint_id` in `routes_mcp_endpoint.go`. Handlers: `internal/handler/custom_agent.go`, `internal/handler/mcp_service.go`, `internal/handler/mcp_credentials.go`, `internal/handler/mcp_oauth.go`, `internal/handler/mcp_endpoint.go`, `internal/handler/skill_handler.go`, `internal/handler/user_resource_favorite.go`.
