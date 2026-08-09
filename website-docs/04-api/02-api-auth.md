Vou traduzir o documento diretamente, mantendo toda a estrutura markdown.

--- DOCUMENT START ---
# API Reference: Authentication & Users

Route registration: `RegisterAuthRoutes` and `RegisterMyInvitationRoutes` in `internal/router/router.go`. Handlers: `internal/handler/auth.go`, `internal/handler/auth_register_by_invite.go`, `internal/handler/tenant_invitation.go`.

Unless otherwise noted, endpoints in this group only require the caller to be "logged in" after the authentication middleware (no minimum role). No-auth endpoints are marked in their own entries.

## Authentication (/api/v1/auth)

### POST /api/v1/auth/register

Purpose: register a new user (self-service registration mode). No auth required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `username` | string | Yes (`binding:"required"`) | Username |
| `email` | string | Yes (`binding:"required"`) | Email |
| `password` | string | Yes (`binding:"required"`) | Password |
| `tenant_provisioning` | string | No | Tenant provisioning strategy |

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
| `password` | string | Yes (`binding:"required,min=6"`) | Password (≥6 characters) |

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

Purpose: one-click initialization (automatically creates an account and tenant for local/Lite scenarios). No auth required, no request body. Handler: `internal/handler/auth.go`

Response: 200, same as Login.

```bash
curl -X POST $BASE/api/v1/auth/auto-setup
```

### GET /api/v1/auth/config

Purpose: query authentication configuration such as registration mode. No auth required. Handler: `internal/handler/auth.go`

Response: 200 `{"success":true,"registration_mode":"self_serve|invite_only"}`

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

Response: 200 `{"success":true,"data":{"user":{UserInfo},"tenant":{TenantResponse},"memberships":[...],"tenant_required":bool,"capabilities":{"can_create_tenant":bool}}}`

```bash
curl $BASE/api/v1/auth/me -H "X-API-Key: $API_KEY"
```

### PUT /api/v1/auth/me/preferences

Purpose: update personal preferences (most recently active tenant). Login required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `last_active_tenant_id` | *uint64 | No | Most recently active tenant ID; null clears it |

Response: 200 `{"success":true,"data":{UserPreferences}}`

```bash
curl -X PUT $BASE/api/v1/auth/me/preferences -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"last_active_tenant_id":2}'
```

### POST /api/v1/auth/change-password

Purpose: change password. Login required. Handler: `internal/handler/auth.go`

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `old_password` | string | Yes (`binding:"required"`) | Old password |
| `new_password` | string | Yes (`binding:"required,min=6"`) | New password (≥6 characters) |

Response: 200 `{"success":true,"message":"Password changed successfully"}`

```bash
curl -X POST $BASE/api/v1/auth/change-password -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"old_password":"old","new_password":"newpass1"}'
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

--- DOCUMENT END ---

Tradução completa entregue, com toda a estrutura markdown, tabelas e blocos de código preservados intactos.
