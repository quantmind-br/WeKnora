# Authentication Management API

[Back to table of contents](./README.md)

For the full OIDC call flow, see [../OIDC认证调用流程.md](../OIDC认证调用流程.md). This document serves as the endpoint reference.

## Description

WeKnora's `/auth/*` endpoints themselves **do not require an X-API-Key**, but some endpoints require the JWT returned by `/auth/login` or `/auth/oidc/callback` to be included in the `Authorization: Bearer <token>` header:

| Endpoint | Authentication method |
| --- | --- |
| `/auth/register` `/auth/login` | None |
| `/auth/oidc/config` `/auth/oidc/url` `/auth/oidc/callback` | None |
| `/auth/refresh` | refresh_token (carried in the request body) |
| `/auth/validate` `/auth/me` `/auth/logout` `/auth/change-password` | Bearer JWT |

The registration endpoint can be disabled via the environment variable `DISABLE_REGISTRATION=true`.

## Endpoint Overview

| Method | Path                       | Description                                       |
| ---- | -------------------------- | ------------------------------------------ |
| POST | `/auth/register`           | User registration                                   |
| POST | `/auth/login`              | User login                                   |
| GET  | `/auth/oidc/config`        | Get OIDC configuration metadata                       |
| GET  | `/auth/oidc/url`           | Get OIDC authorization link                         |
| GET  | `/auth/oidc/callback`      | OIDC authorization callback (triggered by IdP redirect)         |
| POST | `/auth/refresh`            | Exchange refresh_token for a new access_token       |
| GET  | `/auth/validate`           | Validate JWT validity                            |
| POST | `/auth/logout`             | Log out                                   |
| GET  | `/auth/me`                 | Get current user info                           |
| POST | `/auth/change-password`    | Change password                                   |

---

## POST `/auth/register` - User Registration

**Parameters (request body)**:

| Field    | Type   | Required | Validation                       | Description      |
| -------- | ------ | ---- | -------------------------- | --------- |
| username | string | Yes   | Length 2-50                   | Username    |
| email    | string | Yes   | Valid email format                   | Email      |
| password | string | Yes   | Minimum 6 characters                   | Password      |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/register' \
--header 'Content-Type: application/json' \
--data '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "secret123"
}'
```

**Response** (201 Created):

```json
{
    "success": true,
    "message": "Registration successful",
    "user": {
        "id": "usr-...",
        "username": "alice",
        "email": "alice@example.com",
        "tenant_id": 1,
        "is_active": true,
        "created_at": "2026-05-11T10:00:00+08:00",
        "updated_at": "2026-05-11T10:00:00+08:00"
    },
    "tenant": {
        "id": 1,
        "name": "alice's workspace",
        "api_key": "sk-..."
    }
}
```

**Errors**: Registration disabled → 403; parameter validation failed → 400.

---

## POST `/auth/login` - User Login

**Parameters (request body)**:

| Field    | Type   | Required | Description          |
| -------- | ------ | ---- | ------------- |
| email    | string | Yes   | Registered email      |
| password | string | Yes   | Password          |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/login' \
--header 'Content-Type: application/json' \
--data '{
    "email": "alice@example.com",
    "password": "secret123"
}'
```

**Response**:

```json
{
    "success": true,
    "message": "Login successful",
    "user": { "id": "usr-...", "username": "alice", "email": "alice@example.com" },
    "tenant": { "id": 1, "name": "alice's workspace", "api_key": "sk-..." },
    "token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi..."
}
```

**Errors**: Incorrect email or password → 401; account disabled → 403.

---

## GET `/auth/oidc/config` - Get OIDC Configuration Metadata

Returns whether OIDC is enabled and the Provider's display name; the frontend login page uses this to decide whether to show the OIDC login button.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/oidc/config'
```

**Response**:

```json
{
    "success": true,
    "enabled": true,
    "provider_display_name": "WeKnora SSO"
}
```

---

## GET `/auth/oidc/url` - Get OIDC Authorization Link

Returns the OIDC IdP authorization page URL the frontend should redirect to, along with the state code.

**Query parameters**:

| Field      | Type   | Required | Description                                                    |
| ---------- | ------ | ---- | ------------------------------------------------------- |
| redirect   | string | No   | The path the frontend expects to land on after a successful login (e.g. `/dashboard`), passed through in the state | 

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/oidc/url?redirect=%2Fdashboard'
```

**Response**:

```json
{
    "success": true,
    "provider_display_name": "WeKnora SSO",
    "authorization_url": "https://idp.example.com/oauth/authorize?client_id=...&state=...",
    "state": "abcdef..."
}
```

---

## GET `/auth/oidc/callback` - OIDC Authorization Callback

The IdP redirects to this endpoint after the user authorizes. Client code generally does not need to call this directly — its purpose is to pass the login result back to the frontend homepage via the browser URL hash.

**Query parameters**:

| Field              | Type   | Required | Description                          |
| ----------------- | ------ | ---- | ----------------------------- |
| code              | string | Yes   | Authorization code issued by the IdP |
| state             | string | Yes   | Must match the value returned by `/auth/oidc/url` |
| error             | string | No   | Error identifier returned by the IdP            |
| error_description | string | No   | Error details returned by the IdP            |

**Response**: Always returns `302 Found`, redirecting to `/`, with the result encoded into the URL hash:

- Success: `/#oidc_result=<base64url(JSON payload)>`, where the payload contains `success` / `user` / `tenant` / `token` / `refresh_token` / `is_new_user`, consistent with the login response.
- Failure: `/#oidc_error=<reason>[&oidc_error_description=<message>]`, common reasons include `invalid_state`, `missing_code`, `login_failed`, `payload_encode_failed`.

---

## POST `/auth/refresh` - Refresh Token

**Parameters (request body)**:

| Field         | Type   | Required | Description              |
| ------------- | ------ | ---- | ----------------- |
| refreshToken  | string | Yes   | The refresh_token issued at login |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/refresh' \
--header 'Content-Type: application/json' \
--data '{
    "refreshToken": "eyJhbGciOi..."
}'
```

**Response**:

```json
{
    "success": true,
    "message": "Token refreshed successfully",
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi..."
}
```

**Errors**: refresh_token invalid or expired → 401.

---

## GET `/auth/validate` - Validate JWT

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/validate' \
--header 'Authorization: Bearer eyJhbGciOi...'
```

**Response**:

```json
{
    "success": true,
    "valid": true,
    "user_id": "usr-...",
    "tenant_id": 1
}
```

An invalid token returns 401.

---

## POST `/auth/logout` - Log Out

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/auth/logout' \
--header 'Authorization: Bearer eyJhbGciOi...'
```

**Response**: `{ "success": true, "message": "Logged out successfully" }`

---

## GET `/auth/me` - Get Current User Info

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/me' \
--header 'Authorization: Bearer eyJhbGciOi...'
```

**Response**:

```json
{
    "success": true,
    "user": {
        "id": "usr-...",
        "username": "alice",
        "email": "alice@example.com",
        "avatar": "",
        "tenant_id": 1,
        "is_active": true,
        "can_access_all_tenants": false,
        "created_at": "2026-05-11T10:00:00+08:00",
        "updated_at": "2026-05-11T10:00:00+08:00"
    }
}
```

---

## POST `/auth/change-password` - Change Password

Changes the current user's login password. The new password must be **8–32 characters** and **contain both letters and digits**; it cannot be the same as the current password. On success, **all sessions are revoked**, and the user must log in again with the new password.

**Parameters (request body)**:

| Field         | Type   | Required | Validation    | Description      |
| ------------- | ------ | ---- | ------- | --------- |
| old_password  | string | Yes   |          | Current password  |
| new_password  | string | Yes   | 8–32 characters, must contain letters and digits, and must differ from the old password | New password    |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/auth/change-password' \
--header 'Authorization: Bearer eyJhbGciOi...' \
--header 'Content-Type: application/json' \
--data '{
    "old_password": "secret123",
    "new_password": "newsecret456"
}'
```

**Response**: `{ "success": true, "message": "Password changed successfully" }`

**Errors** (400):

| `error.details`       | Meaning                         |
| --------------------- | ---------------------------- |
| `invalid_old_password` | Current password is incorrect               |
| `password_policy`      | New password does not meet length/complexity requirements  |
| `same_password`        | New password is the same as the current password         |
