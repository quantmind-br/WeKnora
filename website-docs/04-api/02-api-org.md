# API Reference: Organizations & Sharing

Route registration: `RegisterOrganizationRoutes` in `internal/router/router.go`. Handler: `internal/handler/organization.go`.

An Organization uses "spaces" (tenants) as its member unit. The API key policy for the organization route group is `manage_spaces` or full-access; KB/Agent share management is available only to full-access keys.

## Organization Management (/api/v1/organizations)

### POST /api/v1/organizations

Purpose: Create an organization. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Organization name |
| `description` | string | No | Description |
| `avatar` | string | No | Avatar URL |
| `searchable` | bool | No | Whether the organization can be discovered via search |
| `require_approval` | bool | No | Whether joining requires approval |
| `member_limit` | int | No | Maximum number of member spaces |
| `invite_code_validity_days` | int | No | Invite code validity period (days) |

Response: 201 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X POST $BASE/api/v1/organizations -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"R&D Organization"}'
```

### GET /api/v1/organizations

Purpose: List the organizations I belong to. Permission: Viewer+.

Response: 200 `{"success":true,"data":{"organizations":[...],"total":N,"resource_counts":{"knowledge_bases":{"by_organization":{}},"agents":{"by_organization":{}}}}}`

```bash
curl $BASE/api/v1/organizations -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/preview/:code

Purpose: Preview an organization by invite code (without joining). Permission: Viewer+. Path parameter: `code` invite code.

Response: 200 `{"success":true,"data":{id,name,description,avatar,member_count,share_count,agent_share_count,is_already_member,require_approval,created_at}}`

```bash
curl $BASE/api/v1/organizations/preview/ABC123 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/join

Purpose: Join an organization using an invite code. Permission: Admin+. Request body: `{"invite_code":"..."}` (required).

Response: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X POST $BASE/api/v1/organizations/join -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"invite_code":"ABC123"}'
```

### POST /api/v1/organizations/join-request

Purpose: Submit a join request (for organizations requiring approval). Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `invite_code` | string | Yes | Invite code |
| `message` | string | No | Request note |
| `role` | string | No | Desired role |

Response: 200 `{"success":true,"data":{JoinRequest}}`

```bash
curl -X POST $BASE/api/v1/organizations/join-request -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"invite_code":"ABC123","message":"Request to join"}'
```

### GET /api/v1/organizations/search

Purpose: Search for discoverable (searchable) organizations. Permission: Viewer+.

| Query Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `q` | string | No | Keyword |
| `limit` | int | No | Default 20, maximum 100 |

Response: 200 `{"success":true,"data":[SearchableOrganization],"total":N}`

```bash
curl "$BASE/api/v1/organizations/search?q=R&D" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/join-by-id

Purpose: Join a discoverable organization by organization ID (no invite code needed). Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `organization_id` | string | Yes | Target organization ID |
| `message` | string | No | Note |
| `role` | string | No | Desired role |

Response: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X POST $BASE/api/v1/organizations/join-by-id -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"organization_id":"org-1"}'
```

### GET /api/v1/organizations/:id

Purpose: Organization details. Permission: Viewer+.

Response: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl $BASE/api/v1/organizations/org-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/organizations/:id

Purpose: Update an organization (the service layer verifies the caller's space is the organization owner). Permission: Admin+. Request body fields are the same as creation (all optional).

Response: 200 `{"success":true,"data":{OrganizationResponse}}`

```bash
curl -X PUT $BASE/api/v1/organizations/org-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"description":"Updated description"}'
```

### DELETE /api/v1/organizations/:id

Purpose: Delete an organization. Permission: Admin+ (the service layer requires organization owner).

Response: 200 `{"success":true,"message":"Organization deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/organizations/org-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/:id/leave

Purpose: Have the current space leave the organization. Permission: Admin+. No request body.

Response: 200 `{"success":true,"message":"Left organization successfully"}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/leave -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/:id/request-upgrade

Purpose: Request an upgrade of the current space's role within the organization. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `requested_role` | string | Yes | Desired organization role (`viewer/editor/admin`) |
| `message` | string | No | Note |

Response: 200 `{"success":true,"data":{JoinRequest}}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/request-upgrade -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"requested_role":"editor"}'
```

### POST /api/v1/organizations/:id/invite-code

Purpose: Generate an organization invite code. Permission: Admin+ (the service layer requires organization admin). No request body.

Response: 200 `{"success":true,"data":{"invite_code":"..."}}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/invite-code -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/search-tenants

Purpose: Search for spaces that can be invited (returns candidates grouped by space). Permission: Admin+.

| Query Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `q` | string | Yes | Space name keyword |
| `limit` | int | No | Default 10, maximum 50 |

Response: 200 `{"success":true,"data":[{"tenant_id","tenant_name"}]}`

```bash
curl "$BASE/api/v1/organizations/org-1/search-tenants?q=demo" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/search-users

Purpose: Deprecated alias, behaves the same as `search-tenants` (returns space-grouped results). Permission: Admin+. Parameters same as above.

```bash
curl "$BASE/api/v1/organizations/org-1/search-users?q=demo" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/organizations/:id/invite

Purpose: Directly invite a space to join the organization. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `tenant_id` | uint64 | One of two | Target space ID (recommended) |
| `user_id` | string | One of two | Compatibility path: user ID (resolved to their space) |
| `representative_user_id` | string | No | Representative user for this space |
| `role` | string | Yes | Role within the organization |

Response: 200 `{"success":true,"message":"Member added successfully"}`

```bash
curl -X POST $BASE/api/v1/organizations/org-1/invite -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"tenant_id":2,"role":"viewer"}'
```

### GET /api/v1/organizations/:id/members

Purpose: List of organization members (spaces). Permission: Viewer+.

Response: 200 `{"success":true,"data":{"members":[{id,user_id,representative_user_id,role,tenant_id,tenant_name,username,email,avatar,joined_at}],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/members -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/organizations/:id/members/:tenant_id

Purpose: Modify a member space's organization role. Permission: Admin+. Path parameter `tenant_id` is the member space ID. Request body: `{"role":"editor"}` (required, `viewer/editor/admin`).

Response: 200 `{"success":true,"message":"Member role updated successfully"}`

```bash
curl -X PUT $BASE/api/v1/organizations/org-1/members/2 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"role":"editor"}'
```

### DELETE /api/v1/organizations/:id/members/:tenant_id

Purpose: Remove a member space (including self-removal). Permission: Admin+.

Response: 200 `{"success":true,"message":"Member removed successfully"}`

```bash
curl -X DELETE $BASE/api/v1/organizations/org-1/members/2 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/join-requests

Purpose: Join request queue. Permission: Admin+.

Response: 200 `{"success":true,"data":{"requests":[{id,user_id,username,email,message,request_type,prev_role,requested_role,status,created_at,reviewed_at}],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/join-requests -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/organizations/:id/join-requests/:request_id/review

Purpose: Approve or reject a join/upgrade request. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `approved` | bool | Yes | Approve/reject |
| `message` | string | No | Review note |
| `role` | string | No | Role granted upon approval |

Response: 200 `{"success":true,"message":"Review completed"}`

```bash
curl -X PUT $BASE/api/v1/organizations/org-1/join-requests/req-1/review \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' -d '{"approved":true}'
```

### GET /api/v1/organizations/:id/shares

Purpose: View the list of KBs shared with this organization. Permission: Viewer+.

Response: 200 `{"success":true,"data":{"shares":[KnowledgeBaseShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/shares -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/agent-shares

Purpose: View the list of Agents shared with this organization. Permission: Viewer+.

Response: 200 `{"success":true,"data":{"shares":[AgentShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/organizations/org-1/agent-shares -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/shared-knowledge-bases

Purpose: Organization space view: all KBs shared within the organization (including my own). Permission: Viewer+.

Response: 200 `{"success":true,"data":[...includes is_mine, source_from_agent flags...],"total":N}`

```bash
curl $BASE/api/v1/organizations/org-1/shared-knowledge-bases -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/organizations/:id/shared-agents

Purpose: Organization space view: all Agents shared within the organization. Permission: Viewer+.

Response: 200 `{"success":true,"data":[SharedAgentInfo],"total":N}`

```bash
curl $BASE/api/v1/organizations/org-1/shared-agents -H "Authorization: Bearer $TOKEN"
```

## KB Sharing (/api/v1/knowledge-bases/:id/shares)

API key: full-access only. Handler: `internal/handler/organization.go`

### POST /api/v1/knowledge-bases/:id/shares

Purpose: Share a KB with an organization. Permission: KB creator OR Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `organization_id` | string | Yes | Target organization |
| `permission` | string | Yes | Share permission (organization role semantics, e.g. `viewer/editor`) |

Response: 201 `{"success":true,"data":{KBShare}}`

```bash
curl -X POST $BASE/api/v1/knowledge-bases/kb-1/shares -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"organization_id":"org-1","permission":"viewer"}'
```

### GET /api/v1/knowledge-bases/:id/shares

Purpose: View the share list for this KB. Permission: Viewer+.

Response: 200 `{"success":true,"data":{"shares":[KnowledgeBaseShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/shares -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/knowledge-bases/:id/shares/:share_id

Purpose: Modify share permissions. Permission: KB creator OR Admin+. Request body: `{"permission":"editor"}` (required).

Response: 200 `{"success":true,"message":"Share permission updated successfully"}`

```bash
curl -X PUT $BASE/api/v1/knowledge-bases/kb-1/shares/s-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"permission":"editor"}'
```

### DELETE /api/v1/knowledge-bases/:id/shares/:share_id

Purpose: Cancel a share. Permission: KB creator OR Admin+.

Response: 200 `{"success":true,"message":"Share removed successfully"}`

```bash
curl -X DELETE $BASE/api/v1/knowledge-bases/kb-1/shares/s-1 -H "Authorization: Bearer $TOKEN"
```

## Agent Sharing (/api/v1/agents/:id/shares)

API key: full-access only. Handler: `internal/handler/organization.go`

### POST /api/v1/agents/:id/shares

Purpose: Share an Agent with an organization. Permission: Agent creator OR Admin+. Request body is the same as KB sharing (`organization_id` + `permission`, required).

Response: 201 `{"success":true,"data":{AgentShare}}`

```bash
curl -X POST $BASE/api/v1/agents/agent-1/shares -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"organization_id":"org-1","permission":"viewer"}'
```

### GET /api/v1/agents/:id/shares

Purpose: View the share list for this Agent. Permission: Agent creator OR Admin+.

Response: 200 `{"success":true,"data":{"shares":[AgentShareResponse],"total":N}}`

```bash
curl $BASE/api/v1/agents/agent-1/shares -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/agents/:id/shares/:share_id

Purpose: Cancel an Agent share. Permission: Agent creator OR Admin+.

Response: 200 `{"success":true,"message":"Share removed successfully"}`

```bash
curl -X DELETE $BASE/api/v1/agents/agent-1/shares/s-1 -H "Authorization: Bearer $TOKEN"
```

## Aggregated Shared Resource Views

### GET /api/v1/shared-knowledge-bases

Purpose: List KBs shared with me via organizations (with owner-side vector store metadata stripped out). Permission: Viewer+; API key requires `manage_spaces` or full-access.

Response: 200 `{"success":true,"data":[...],"total":N}`

```bash
curl $BASE/api/v1/shared-knowledge-bases -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/shared-agents

Purpose: List Agents shared with me via organizations. Permission: Viewer+; API key same as above.

Response: 200 `{"success":true,"data":[SharedAgentInfo],"total":N}`. `SharedAgentInfo` includes `source_tenant_id` (source space), `org_name`, `shared_by_username`, `permission`, and `web_search_ready` — a single boolean flag indicating only "whether web search is available in the source space." It does not expose the source space's provider configuration (which would leak configuration), nor does it compare against the receiving space's provider ID (which would cause false unavailability reports).

When calling other endpoints using a shared Agent, if an Agent with the same name is shared by multiple spaces, you can specify `agent_source_tenant_id` to indicate the source space; this value is validated against the sharing relationship one by one, and an error is raised directly if it is invalid or unauthorized — it will not silently fall back to a different source.

```bash
curl $BASE/api/v1/shared-agents -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/shared-agents/disabled

Purpose: Set "this space disables a given shared Agent" (affects the conversation dropdown for the entire space). Permission: Admin+; API key same as above.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `agent_id` | string | Yes (`binding:"required"`) | Shared Agent ID |
| `disabled` | bool | No | Whether to disable (default false) |

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/shared-agents/disabled -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"agent_id":"agent-1","disabled":true}'
```
