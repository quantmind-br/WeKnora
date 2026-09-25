# API Reference: Tenants (Spaces) & Members

Manages workspaces, members, invitations, API keys, and audit logs. Operations apply to the currently active space, and cross-space access is checked against each endpoint's permissions.

All `/tenants/:id/*` routes mount `PathTenantMatch()` (`internal/middleware/access.go`) at the group level: the `:id` in the URL must match the currently active space (except for cross-tenant super admins), preventing unauthorized operations on someone else's space.

For the tenant `memory_config` fields and the personal memory endpoints, see the [Long-term Memory API](02-api-memory.md). When a space administrator updates the configuration, submit the complete object to keep.

## Space Lifecycle

### POST /api/v1/tenants

Purpose: create a space (self-service creation of a new workspace; the caller automatically becomes Owner). Permissions: any logged-in user (may have no space); API key requires a platform key with `system_tenants_manage`. Handler: `internal/handler/tenant.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required,min=1,max=128"`) | Space name |
| `description` | string | No (`binding:"max=512"`) | Description |

Cross-tenant super admins can submit a full `types.Tenant` (including `storage_quota`, `status`, etc.).

Response: 201 `{"success":true,"data":{Tenant}}` (may include `api_key` if configuration allows). Returns 403 (code 2005) if self-service creation is disabled, or 429 if the quota is exceeded.

```bash
curl -X POST $BASE/api/v1/tenants -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"My Space"}'
```

### GET /api/v1/tenants

Purpose: list spaces I can access. Permissions: logged-in; API key requires `manage_tenant_settings` or full access. Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"data":{"items":[TenantResponse]}}`

```bash
curl $BASE/api/v1/tenants -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/all

Purpose: list all spaces (cross-tenant super admins). Permissions: `CrossTenant()` (`CanAccessAllTenants` and the cluster has `EnableCrossTenantAccess` enabled); platform key requires `system_tenants_read|manage`. Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"data":{"items":[TenantResponse]}}`

```bash
curl $BASE/api/v1/tenants/all -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/search

Purpose: search spaces by keyword (cross-tenant super admins). Permissions: same as above. Handler: `internal/handler/tenant.go`

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `keyword` | string | No | Keyword |
| `tenant_id` | string | No | Exact space ID |
| `page` / `page_size` | int | No | Pagination (default 1/20, max 100) |

Response: 200 `{"success":true,"data":{"items":[...],"total","page","page_size"}}`

```bash
curl "$BASE/api/v1/tenants/search?keyword=demo&page=1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/:id

Purpose: space details. Permissions: Viewer+; platform key requires `system_tenants_read|manage`. Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"data":{TenantResponse}}`

```bash
curl $BASE/api/v1/tenants/1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/tenants/:id

Purpose: update space configuration. Permissions: Owner; platform key requires `system_tenants_manage`. Handler: `internal/handler/tenant.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | *string | No (`binding:"omitempty,min=1,max=128"`) | New name |
| `description` | *string | No (`binding:"omitempty,max=512"`) | New description |

Response: 200 `{"success":true,"data":{TenantResponse}}`

```bash
curl -X PUT $BASE/api/v1/tenants/1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"New name"}'
```

### DELETE /api/v1/tenants/:id

Purpose: delete a space. Permissions: Owner; platform key requires `system_tenants_manage`. The space record and all its memberships are soft-deleted, and members lose access immediately; data in the space such as knowledge bases and models is not physically purged right away, and queued Wiki tasks of a deleted space no longer call models. Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"message":"Workspace deleted successfully"}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1 -H "Authorization: Bearer $TOKEN"
```

## Space KV Configuration

`:key` is a configuration key, not a space ID (the space is taken from the auth context). Valid values: `web-search-config`, `prompt-templates`, `parser-engine-config`, `storage-engine-config`, `chat-history-config`, `retrieval-config`, `memory-config`.

### GET /api/v1/tenants/kv/:key

Purpose: read a space-level KV configuration. Permissions: Viewer+; API key requires `manage_tenant_settings` or full access. Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"data":{...corresponding config object...}}`

```bash
curl $BASE/api/v1/tenants/kv/retrieval-config -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/tenants/kv/:key

Purpose: update a space-level KV configuration. Permissions: Admin+; API key requires `manage_tenant_settings` or full access. Request body: the JSON config object corresponding to `:key`. Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"message":"Configuration updated"}`

```bash
curl -X PUT $BASE/api/v1/tenants/kv/web-search-config -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"enabled":true}'
```

## API Keys & API Principals

### GET /api/v1/tenants/:id/api-keys

Purpose: list space API keys (masked). Permissions: Owner, JWT only (API key denied by default). Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"data":[{id,scope_type,name,api_key(masked),full_access,knowledge_base_ids,capabilities,last_used_at,expires_at,created_at}]}`

```bash
curl $BASE/api/v1/tenants/1/api-keys -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/api-keys

Purpose: create a space API key (plaintext returned only once). Permissions: Owner, JWT only. Handler: `internal/handler/tenant.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Key name |
| `full_access` | bool | No | Full-access key for the space (default false) |
| `knowledge_base_ids` | []string | No | KB allowlist (scoped key) |
| `capabilities` | []string | No | List of capabilities (see overview) |
| `expires_at_unix` | *int64 | No | Expiration timestamp |

Response: 201 `{"success":true,"data":{...,"api_key":"<plaintext>","token":"<plaintext>"}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/api-keys -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"name":"ingest-bot","capabilities":["ingest","retrieve"],"knowledge_base_ids":["kb-1"]}'
```

### PUT /api/v1/tenants/:id/api-keys/:key_id

Owner，仅 JWT。更新已有 Key 的 name、full_access、knowledge_base_ids、capabilities、expires_at_unix，授权字段按整份配置提交；不是只改一个字段的 PATCH。expires_at_unix 省略或 null 会清除已有到期时间。更改权限后使用同一 token，新授权在后续认证时生效，不重新返回明文。

返回 200 `{success,data:APIKeyResponse}`，Key 脱敏；非法能力/知识库范围返回 400，不存在返回 404。

```bash
curl -X PUT "$BASE/api/v1/tenants/1/api-keys/5" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"search-bot","full_access":false,"knowledge_base_ids":["kb-1"],"capabilities":["retrieve"]}'
```

### DELETE /api/v1/tenants/:id/api-keys/:key_id

Purpose: delete an API key. Permissions: Owner, JWT only. Path parameter: `key_id`.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1/api-keys/5 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/tenants/:id/api-principal-config

Purpose: read the API external user principal configuration. Permissions: Owner, JWT only. Handler: `internal/handler/tenant.go`

Response: 200 `{"success":true,"data":{"mode":"tenant|direct|signed_token","direct_header_name","signed_token_header_name","require_direct_header","has_hmac_secret"}}`

```bash
curl $BASE/api/v1/tenants/1/api-principal-config -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/tenants/:id/api-principal-config

Purpose: update the API external user principal configuration. Permissions: Owner, JWT only.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `mode` | string | Yes | `tenant` / `direct` / `signed_token` |
| `require_direct_header` | bool | No | Whether the direct mode enforces the header |
| `hmac_secret` | *string | No | Secret for signed_token mode (pass `***` to keep the original value) |

Response: 200, same as GET.

```bash
curl -X PUT $BASE/api/v1/tenants/1/api-principal-config -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"mode":"signed_token","hmac_secret":"topsecret"}'
```

### POST /api/v1/tenants/:id/api-principal-test-token

Purpose: issue an external user JWT for testing purposes. Permissions: Owner, JWT only.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `external_user_id` | string | Yes | External user ID (≤128 characters) |
| `expires_in_seconds` | int | No | 1-3600, default 900 |

Response: 200 `{"success":true,"data":{"token","header_name","expires_in_seconds","expires_at_unix","external_user_id"}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/api-principal-test-token -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"external_user_id":"u-123"}'
```

## Member Management (/tenants/:id/members)

Handler: `internal/handler/tenant_member.go`. API key requires `manage_members` or full access.

### GET /api/v1/tenants/:id/members

Purpose: member list. Permissions: Viewer+.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `q` | string | No | Filter by email/username |
| `page` / `page_size` | int | No | Pagination |

Response: 200 `{"success":true,"data":{"members":[{user_id,email,username,avatar,role,status,invited_by,joined_at}],"total","page","page_size"}}`

```bash
curl $BASE/api/v1/tenants/1/members -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/members

Purpose: add a member directly. Permissions: Owner.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `email` | string | Yes (`binding:"required,email"`) | Member's email (must already be registered) |
| `role` | string | Yes (`binding:"required"`) | `owner/admin/contributor/viewer` |

Response: 201 `{"success":true,"data":{member object}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/members -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"b@ex.com","role":"contributor"}'
```

### PUT /api/v1/tenants/:id/members/:user_id

Purpose: change a member's role. Permissions: Owner. Request body: `{"role":"admin"}` (`binding:"required"`).

Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/tenants/1/members/u-123 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"role":"admin"}'
```

### DELETE /api/v1/tenants/:id/members/:user_id

Purpose: remove a member. Permissions: Owner.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1/members/u-123 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/leave

Purpose: leave a space (any member may leave on their own; the service layer rejects a leave that would leave the space without an Owner). Permissions: Viewer+, JWT only.

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/tenants/1/leave -H "Authorization: Bearer $TOKEN"
```

## Space Invitations (/tenants/:id/invitations and invite-links)

Handlers: `internal/handler/tenant_invitation.go`, `internal/handler/tenant_invite_link.go`. API key requires `manage_members` or full access.

### GET /api/v1/tenants/:id/invitations

Purpose: list space invitations. Permissions: Viewer+.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `include_terminal` | bool | No | Include invitations that have already reached a terminal state |
| `page` / `page_size` | int | No | Pagination |

Response: 200 `{"success":true,"data":{"invitations":[{id,tenant_id,invitee_email,inviter_email,role,status,message,expires_at,is_share_link,accepted_count,...}],"total","page","page_size"}}`

```bash
curl $BASE/api/v1/tenants/1/invitations -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/invitations

Purpose: invite a member (only stored once the invitee confirms it under `/me/invitations`). Permissions: Owner.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `email` | string | Yes (`binding:"required,email"`) | Invitee's email |
| `role` | string | Yes (`binding:"required"`) | Role to grant |
| `message` | string | No | Optional message |

Response: 201 `{"success":true,"data":{TenantInvitationResponse}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/invitations -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"c@ex.com","role":"viewer"}'
```

### DELETE /api/v1/tenants/:id/invitations/:inv_id

Purpose: revoke an invitation. Permissions: Owner.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/tenants/1/invitations/12 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/tenants/:id/invite-links

Purpose: create a share link (a reusable sign-up invitation link). Permissions: Owner. Handler: `internal/handler/tenant_invite_link.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `role` | string | Yes (`binding:"required"`) | Role granted by the link |
| `message` | string | No | Optional message |

Response: 201 `{"success":true,"data":{id,token,invite_url,role,status,expires_at,is_share_link:true,accepted_count}}`

```bash
curl -X POST $BASE/api/v1/tenants/1/invite-links -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"role":"viewer"}'
```

## Audit Log

Handler: `internal/handler/audit_log.go`. Cursor-based pagination.

### GET /api/v1/tenants/:id/audit-log

Purpose: space audit log (includes records of denied operations). Permissions: Admin+, JWT only.

| Query parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `after_id` | int | No | Cursor (from the previous response's `next_cursor`) |
| `limit` | int | No | 1-100, default 50 |
| `action` | string | No | Filter by action (e.g. `rbac.member_added`) |
| `outcome` | string | No | `success` / `denied` |
| `actor` | string | No | Filter by the actor's user_id |

Response: 200 `{"success":true,"data":[AuditLog],"next_cursor":N}`

```bash
curl "$BASE/api/v1/tenants/1/audit-log?limit=50" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/knowledge-bases/:id/activity

Purpose: activity stream for a single KB (read-only audit). Permissions: KB creator OR Admin+, and must have read permission on the KB; JWT only. Query parameters same as above (`after_id/limit/action/outcome/actor`). Registered in `RegisterKnowledgeBaseActivityRoutes`.

Response: 200 `{"success":true,"data":[AuditLog],"next_cursor":N}`. `details` is the action payload; if the entry was triggered by an API Key, it includes `api_key_id` and `api_key_name` (a snapshot of the name, never the plaintext key).

```bash
curl $BASE/api/v1/knowledge-bases/kb-1/activity -H "Authorization: Bearer $TOKEN"
```

## 实现参考

路由注册：`internal/router/routes_auth_tenant.go` 的 `RegisterTenantRoutes`。Handler：`internal/handler/tenant.go`、`internal/handler/tenant_member.go`、`internal/handler/tenant_invitation.go`、`internal/handler/tenant_invite_link.go`、`internal/handler/audit_log.go`。
