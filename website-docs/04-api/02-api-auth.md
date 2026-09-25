# API Reference: Authentication & Users

Provides endpoints for registration, login, token refresh, user profile, and invitation handling. Authentication requirements vary by endpoint; public endpoints are marked in their own entries.

Unless otherwise noted, endpoints in this group only require the caller to be "logged in" after the authentication middleware (no minimum role). No-auth endpoints are marked in their own entries.

## Authentication (/api/v1/auth)

### POST /api/v1/auth/register

Purpose: register a new user (self-service registration mode). No auth required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `username` | string | Yes | Username, 2–50 characters |
| `email` | string | Yes | Email |
| `password` | string | Yes | Password (8–32 characters, letters + digits; complex mode additionally requires uppercase, lowercase, and special characters) |

Whether a personal space is created automatically is decided by the server's `auth.default_tenant_mode`; the request body cannot specify it. Returns 403 in `invite_only` mode.

Response: 201 `{"success":true,"message":"...","user":{User}}`

```bash
curl -X POST $BASE/api/v1/auth/register -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"a@ex.com","password":"secret123"}'
```

### POST /api/v1/auth/register-by-invite

Purpose: register and join a tenant via an invitation/share-link token. No auth required, IP rate limit of 30 requests/minute. Handler: `internal/handler/auth_register_by_invite.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `token` | string | Yes (`binding:"required"`) | Invitation token |
| `email` | string | Yes (`binding:"required,email"`) | Email |
| `username` | string | Yes (`binding:"required"`) | Username |
| `password` | string | Yes (`binding:"required,min=6"`) | Password (8–32 characters, letters + digits; complex mode additionally requires uppercase, lowercase, and special characters) |

Response: 201, same as Login (`user/active_tenant/memberships/token/refresh_token`).

```bash
curl -X POST $BASE/api/v1/auth/register-by-invite -H 'Content-Type: application/json' \
  -d '{"token":"<invite_token>","email":"a@ex.com","username":"alice","password":"secret123"}'
```

### POST /api/v1/auth/invitations/lookup

Purpose: anonymously look up the tenant information associated with an invitation token (preview before registering). No auth required, IP rate limited. Handler: `internal/handler/auth_register_by_invite.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `token` | string | Yes (`binding:"required"`) | Invitation token |

Response: 200 `{"success":true,"data":{"tenant_id","tenant_name","role","expires_at"}}`

```bash
curl -X POST $BASE/api/v1/auth/invitations/lookup -H 'Content-Type: application/json' -d '{"token":"<invite_token>"}'
```

### POST /api/v1/auth/login

Purpose: log in with email and password. No auth required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `email` | string | Yes (`binding:"required"`) | Email |
| `password` | string | Yes (`binding:"required"`) | Password |

Response: 200 `{"success":true,"user":{...},"active_tenant":{...},"memberships":[...],"token":"...","refresh_token":"..."}`

```bash
curl -X POST $BASE/api/v1/auth/login -H 'Content-Type: application/json' -d '{"email":"a@ex.com","password":"secret123"}'
```

### POST /api/v1/auth/auto-setup

Purpose: automatic account creation, space creation, and login for the native desktop Lite app. The request must carry the per-process random `X-WeKnora-Desktop-Token` provided by the desktop native bridge; anonymous HTTP requests are not accepted. Regular browsers use the register/login endpoints. Handler: `internal/handler/auth.go`

Response: 200, same as Login; a missing or wrong desktop credential returns 401.

### GET /api/v1/auth/config

Purpose: query authentication configuration such as registration mode. No auth required. Handler: `internal/handler/auth.go`

Response: 200 `{"success":true,"registration_mode":"self_serve|invite_only","complex_password_enabled":false}`

```bash
curl $BASE/api/v1/auth/config
```

### POST /api/v1/auth/switch-tenant

Purpose: switch the current active tenant and reissue a token. Login required (callable even without a tenant). Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `tenant_id` | uint64 | Yes (`binding:"required"`) | Target tenant ID |
| `refresh_token` | string | No | Used to reissue a new token |

Response: 200, same as Login.

换签成功会把目标空间写入账号级「最近活跃租户」偏好，下次登录（密码/OIDC/换设备）与 refresh 都回到该空间。refresh JWT 不含 `tenant_id`，因此偏好写入失败则整次换签失败、不会发出新 token。API 客户端无需再补发 `PUT /auth/me/preferences`。Web UI 切空间不走本接口。一次换签会改变该用户所有设备的下次落点。

```bash
curl -X POST $BASE/api/v1/auth/switch-tenant -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"tenant_id":2}'
```

### GET /api/v1/auth/oidc/config

Purpose: query whether OIDC is enabled and its display name. No auth required. Handler: `internal/handler/auth.go`

Response: 200 `{"success":true,"enabled":bool,"provider_display_name":"..."}`

```bash
curl $BASE/api/v1/auth/oidc/config
```

### GET /api/v1/auth/oidc/start

免登录，直接返回 302 和指向 IdP 的 Location，可供企业门户链接使用。无需先请求 JSON 授权地址；回调由请求 origin 构造为 `/api/v1/auth/oidc/callback`。回调后的登录结果与原 OIDC 链路一致。

```bash
curl -i "$BASE/api/v1/auth/oidc/start"
```

### GET /api/v1/auth/oidc/url

Purpose: get the OIDC authorization redirect URL. No auth required. Handler: `internal/handler/auth.go`

| Query Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `redirect_uri` | string | Yes | Callback address |

Response: 200 `{"success":true,"authorization_url":"...","nonce":"..."}`

```bash
curl "$BASE/api/v1/auth/oidc/url?redirect_uri=https://app.example.com/callback"
```

### GET /api/v1/auth/oidc/callback

Purpose: OIDC authorization callback (entered via browser redirect). No auth required. Handler: `internal/handler/auth.go`

Query parameters: `code`, `state`, `error`, `error_description` (all returned by the OIDC provider).

Response: 302 redirect to the frontend; on success carries `#oidc_result=<base64url>`, on failure carries `#oidc_error=...`.

```bash
curl -i "$BASE/api/v1/auth/oidc/callback?code=xxx&state=yyy"
```

### POST /api/v1/auth/refresh

Purpose: use a refresh token to reissue a new token. No auth required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `refreshToken` | string | Yes (`binding:"required"`) | Refresh token |

Response: 200 `{"success":true,"access_token":"...","refresh_token":"..."}`

```bash
curl -X POST $BASE/api/v1/auth/refresh -H 'Content-Type: application/json' -d '{"refreshToken":"<rt>"}'
```

### GET /api/v1/auth/validate

Purpose: verify whether the current token is valid. Login required (callable without a tenant). Handler: `internal/handler/auth.go`

Response: 200 `{"success":true,"message":"Token is valid","user":{UserInfo}}`

```bash
curl $BASE/api/v1/auth/validate -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/auth/logout

Purpose: log out (invalidate the current token). Login required. No request body. Handler: `internal/handler/auth.go`

Response: 200 `{"success":true,"message":"Logout successful"}`

```bash
curl -X POST $BASE/api/v1/auth/logout -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/auth/me

Purpose: query the current caller's identity (user/tenant/membership/capabilities). Login required; an API key also works (policy `apiKeyAny()`, any valid key). Handler: `internal/handler/auth.go`

Response: 200 `{"success":true,"data":{"user":{UserInfo},"tenant":{TenantResponse},"memberships":[...],"tenant_required":bool,"capabilities":{"can_create_tenant":bool,"auto_accept_invitation":bool}}}`

```bash
curl $BASE/api/v1/auth/me -H "X-API-Key: $API_KEY"
```

### PUT /api/v1/auth/me/preferences

Purpose: update personal preferences (most recently active tenant). Login required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `last_active_tenant_id` | *uint64 | No | A positive integer sets/replaces it; `0` clears it (the next login returns to home); omit it to leave it unchanged. The server writes the same field when `POST /auth/switch-tenant` succeeds. |

Response: 200 `{"success":true,"data":{UserPreferences}}`

```bash
curl -X PUT $BASE/api/v1/auth/me/preferences -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"last_active_tenant_id":2}'
```

密码策略由 `GET /auth/config` 提供。复杂模式允许的特殊字符集为 `!@#$%^&*()_+-=[]{}|;:,.<>?`；不能只根据结构体 binding 的长度标签推导完整校验规则。

### POST /api/v1/auth/change-password

Purpose: change password. Login required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `old_password` | string | Yes (`binding:"required"`) | Old password |
| `new_password` | string | Yes (`binding:"required"`) | New password (8–32 characters, letters + digits; complex mode additionally requires uppercase, lowercase, and special characters) |

Response: 200 `{"success":true,"message":"Password changed successfully"}`

```bash
curl -X POST $BASE/api/v1/auth/change-password -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"old_password":"old","new_password":"NewPass123!"}'
```

修改密码必须提供正确的旧密码，新密码不能与旧密码相同。成功后撤销全部会话，需要重新登录。默认策略 8–32 位、字母和数字；复杂模式要求大小写字母、数字和特殊字符。错误详情可为 `invalid_old_password`、`password_policy`、`same_password`，均返回 400。

### POST /api/v1/me/invitations/accept-by-token

需登录，仅操作当前用户，无空间的新用户也可调用。请求 `{"token":"<invite-token>"}`；有效共享邀请使当前用户加入对应空间。无效、过期、撤销的链接返回 410；空 token 返回 400。成功响应为 `{success:true,data:{membership:{tenant_id,role,status,joined_at},tenant_name}}`，前端据此切换空间。

```bash
curl -X POST "$BASE/api/v1/me/invitations/accept-by-token" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"token":"<invite-token>"}'
```

## My Invitations (/api/v1/me/invitations)

The service layer guarantees that "only the invitee can accept/decline"; no minimum role required (usable even by new users with no tenant). Handler: `internal/handler/tenant_invitation.go`

### GET /api/v1/me/invitations

Purpose: list invitations sent to me.

| Query Parameter | Type | Required | Description |
| --- | --- | --- | --- |
| `include_terminal` | bool | No | When `true`, includes invitations that have already reached a terminal state |

Response: 200 `{"success":true,"data":{"invitations":[TenantInvitationResponse],"total":N}}`

```bash
curl $BASE/api/v1/me/invitations -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/me/invitations/pending-count

Purpose: count of pending invitations (lightweight polling).

Response: 200 `{"success":true,"data":{"pending_count":N}}`

```bash
curl $BASE/api/v1/me/invitations/pending-count -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/me/invitations/:inv_id/accept

Purpose: accept an invitation, creating the membership record. Path parameter: `inv_id` invitation ID. No request body.

Response: 200 `{"success":true,"data":{"membership":{"tenant_id","role","status","joined_at"}}}`

```bash
curl -X POST $BASE/api/v1/me/invitations/12/accept -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/me/invitations/:inv_id/decline

Purpose: decline an invitation. Path parameter: `inv_id`. No request body.

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/me/invitations/12/decline -H "Authorization: Bearer $TOKEN"
```

## 实现参考

路由注册：`internal/router/routes_auth_tenant.go` 的 `RegisterAuthRoutes` 与 `RegisterMyInvitationRoutes`。Handler：`internal/handler/auth.go`、`internal/handler/auth_register_by_invite.go`、`internal/handler/tenant_invitation.go`。
