# Space Management API

[Back to table of contents](./README.md)

Contains two groups of endpoints:
- Space CRUD (`/tenants`, `/tenants/:id`): the currently authenticated user manages spaces they belong to; cross-space access requires admin privileges.
- Cross-space endpoints (`/tenants/all`, `/tenants/search`): **requires the server to have `EnableCrossTenantAccess` enabled and the current user to hold the `CanAccessAllTenants` permission**, otherwise returns 403.
- Space KV configuration (`/tenants/kv/:key`): general-purpose configuration items at the current space level, where **`tenant_id` is obtained from the authentication context and is not passed in the URL**.

| Method | Path                       | Description                                              |
| ------ | -------------------------- | ---------------------------------------------------------|
| GET    | `/tenants/all`             | Get a list of all spaces (requires cross-space permission) |
| GET    | `/tenants/search`          | Paginated space search (requires cross-space permission) |
| POST   | `/tenants`                 | Create a new space                                       |
| GET    | `/tenants/:id`             | Get info for a specific space                            |
| PUT    | `/tenants/:id`             | Update space info                                        |
| DELETE | `/tenants/:id`             | Delete a space                                            |
| GET    | `/tenants/:id/api-keys`    | List space API Keys (Owner)                               |
| POST   | `/tenants/:id/api-keys`    | Create an API Key with a role (Owner)                     |
| DELETE | `/tenants/:id/api-keys/:key_id` | Revoke a specific API Key (Owner)                     |
| GET    | `/tenants/:id/api-principal-config` | Get the API Key principal configuration (Owner)   |
| PUT    | `/tenants/:id/api-principal-config` | Update the API Key principal configuration (Owner) |
| GET    | `/tenants`                 | Get the list of spaces visible to the current user       |
| GET    | `/tenants/kv/:key`         | Get the KV configuration of the current space (space determined by authentication context) |
| PUT    | `/tenants/kv/:key`         | Update the KV configuration of the current space (space determined by authentication context) |

## GET `/tenants/all` - Get list of all spaces

Gets the list of all spaces in the system; requires cross-space permission.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants/all' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-An7_t_izCKFIJ4iht9Xjcjnj_MC48ILvwezEDki9ScfIa7KA'
```

**Response**:

```json
{
    "data": {
        "items": [
            {
                "id": 10001,
                "name": "weknora-1",
                "description": "weknora workspaces 1",
                "status": "active",
                "business": "wechat",
                "created_at": "2025-08-11T20:37:28.39698+08:00",
                "updated_at": "2025-08-11T20:37:28.405693+08:00"
            },
            {
                "id": 10002,
                "name": "weknora-2",
                "description": "weknora workspaces 2",
                "status": "active",
                "business": "wechat",
                "created_at": "2025-08-11T20:52:58.05679+08:00",
                "updated_at": "2025-08-11T20:52:58.060495+08:00"
            }
        ]
    },
    "success": true
}
```

## GET `/tenants/search` - Search spaces

Searches spaces by keyword; requires cross-space permission.

**Query parameters**:
- `keyword`: search keyword (optional)
- `tenant_id`: filter by space ID (optional)
- `page`: page number (default 1)
- `page_size`: items per page (default 20)

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants/search?keyword=weknora&page=1&page_size=10' \
--header 'Content-Type: application/json' \
--header 'X-API-Key: sk-An7_t_izCKFIJ4iht9Xjcjnj_MC48ILvwezEDki9ScfIa7KA'
```

**Response**:

```json
{
    "data": {
        "items": [
            {
                "id": 10002,
                "name": "weknora",
                "description": "weknora workspaces",
                "status": "active",
                "business": "wechat",
                "created_at": "2025-08-11T20:52:58.05679+08:00",
                "updated_at": "2025-08-11T20:52:58.060495+08:00"
            }
        ],
        "total": 1,
        "page": 1,
        "page_size": 10
    },
    "success": true
}
```

## POST `/tenants` - Create a new space

Creates a new space. **By default, no** API Key is automatically issued; after creation, create a key via `POST /tenants/:id/api-keys`. When upgrading from an older version, the existing `tenants.api_key` is migrated to the `tenant_api_keys` table and remains usable until revoked.

> **Legacy behavior compatibility (optional)**: To restore the old behavior of "issuing a default API Key automatically on space creation," set the system setting `tenant.auto_create_api_key` to `true` (or set the environment variable `WEKNORA_TENANT_AUTO_CREATE_API_KEY=true`). Once enabled, creating a space automatically generates an API Key with `full_access` permission, and its plaintext token is returned in the response body's `data.api_key` field (returned only in this creation response — please store it securely). Defaults to `false`.

**Parameters (request body)**:

| Field             | Type   | Required | Description                                             |
| ----------------- | ------ | -------- | -------------------------------------------------------- |
| name              | string | Yes      | Space name                                                |
| description       | string | No       | Space description                                          |
| business          | string | No       | Business identifier (e.g. `wechat`)                       |
| retriever_engines | object | No       | Retrieval engine combination config (`engines` array: each item contains `retriever_type` and `retriever_engine_type`) |
| storage_quota     | int    | No       | Storage quota (bytes)                                      |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants' \
--header 'Content-Type: application/json' \
--data '{
    "name": "weknora",
    "description": "weknora workspaces",
    "business": "wechat",
    "retriever_engines": {
        "engines": [
            {
                "retriever_type": "keywords",
                "retriever_engine_type": "postgres"
            },
            {
                "retriever_type": "vector",
                "retriever_engine_type": "postgres"
            }
        ]
    }
}'
```

**Response** (default, without API Key):

```json
{
    "data": {
        "id": 10000,
        "name": "weknora",
        "description": "weknora workspaces",
        "status": "active",
        "retriever_engines": {
            "engines": [
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "keywords"
                },
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "vector"
                }
            ]
        },
        "business": "wechat",
        "storage_quota": 10737418240,
        "storage_used": 0,
        "created_at": "2025-08-11T20:37:28.396980093+08:00",
        "updated_at": "2025-08-11T20:37:28.396980301+08:00",
        "deleted_at": null
    },
    "success": true
}
```

When `tenant.auto_create_api_key` (or `WEKNORA_TENANT_AUTO_CREATE_API_KEY=true`) is enabled, the response `data` additionally includes an `api_key` field (the plaintext token of the `full_access` key):

```json
{
    "data": {
        "id": 10000,
        "name": "weknora",
        "description": "weknora workspaces",
        "api_key": "sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG",
        "status": "active",
        "business": "wechat",
        "storage_quota": 10737418240,
        "storage_used": 0,
        "created_at": "2025-08-11T20:37:28.396980093+08:00",
        "updated_at": "2025-08-11T20:37:28.396980301+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## GET `/tenants/:id` - Get info for a specific space

Gets details for the space with the specified ID. You can only access spaces you belong to; accessing other spaces requires cross-space permission, otherwise returns 403.

**Path parameters**:

| Field | Type | Description |
| ----- | ---- | ------------ |
| id    | int  | Space ID     |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants/10000' \
--header 'X-API-Key: sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "id": 10000,
        "name": "weknora",
        "description": "weknora workspaces",
        "api_key": "sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG",
        "status": "active",
        "retriever_engines": {
            "engines": [
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "keywords"
                },
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "vector"
                }
            ]
        },
        "business": "wechat",
        "storage_quota": 10737418240,
        "storage_used": 0,
        "created_at": "2025-08-11T20:37:28.39698+08:00",
        "updated_at": "2025-08-11T20:37:28.405693+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## PUT `/tenants/:id` - Update space info

Updates basic info for the specified space. Access rules are the same as `GET /tenants/:id`.

**Path parameters**:

| Field | Type | Description |
| ----- | ---- | ------------ |
| id    | int  | Space ID     |

**Parameters (request body)**: Same fields as `POST /tenants`; fields not passed retain their original values.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/tenants/10000' \
--header 'X-API-Key: sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG' \
--header 'Content-Type: application/json' \
--data '{
    "name": "weknora new",
    "description": "weknora workspaces new",
    "status": "active",
    "retriever_engines": {
        "engines": [
            {
                "retriever_engine_type": "postgres",
                "retriever_type": "keywords"
            },
            {
                "retriever_engine_type": "postgres",
                "retriever_type": "vector"
            }
        ]
    },
    "business": "wechat",
    "storage_quota": 10737418240
}'
```

**Response**:

```json
{
    "data": {
        "id": 10000,
        "name": "weknora new",
        "description": "weknora workspaces new",
        "api_key": "sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG",
        "status": "active",
        "retriever_engines": {
            "engines": [
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "keywords"
                },
                {
                    "retriever_engine_type": "postgres",
                    "retriever_type": "vector"
                }
            ]
        },
        "business": "wechat",
        "storage_quota": 10737418240,
        "storage_used": 0,
        "created_at": "2025-08-11T20:37:28.39698+08:00",
        "updated_at": "2025-08-11T20:49:02.13421034+08:00",
        "deleted_at": null
    },
    "success": true
}
```

## DELETE `/tenants/:id` - Delete a space

Deletes the specified space. Access rules are the same as `GET /tenants/:id`.

**Path parameters**:

| Field | Type | Description |
| ----- | ---- | ------------ |
| id    | int  | Space ID     |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/tenants/10000' \
--header 'X-API-Key: sk-aaLRAgvCRJcmtiL2vLMeB1FB5UV0Q-qB7DlTE1pJ9KA93XZG' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "message": "Workspace deleted successfully",
    "success": true
}
```

## API Key management (`tenant_api_keys`)

Since the scoped API Key redesign, keys are stored as independent records and support:

- **role**: `viewer` (read-only + semantic search POST), `contributor` (knowledge base write access), `admin` (space-level management, excluding the `/api-keys` management surface)
- **knowledge_base_ids**: optional; restricts the key to specified knowledge bases
- **Revocation**: `DELETE /tenants/:id/api-keys/:key_id`
- **Expiration**: optionally set `expires_at_unix` at creation

Space keys are permanently bound to the space they were created in. Route-level capability authorization and KB access guards continue to be enforced after `X-API-Key` authentication.

### Platform API Key

System administrators can create keys not bound to a single space under "System Management → Platform API Key". By default, a platform key may target any existing space, but each operation still requires the corresponding capability; platform keys do not support `full_access`.

- Management endpoints: `GET/POST /system/admin/api-keys`, `DELETE /system/admin/api-keys/:key_id`, callable only from human SystemAdmin sessions — platform keys cannot create or revoke other platform keys.
- When calling regular space APIs, `X-Tenant-ID: <space ID>` must also be passed; after the server resolves the target space, it continues to reuse the existing space Context, route capability, and knowledge base scope checks.
- No `X-Tenant-ID` is needed when calling explicitly exposed `/system/admin/*` control-plane endpoints; a `system_*` capability is required instead.
- The platform key's plaintext is returned only once, in the creation response's `data.token` field; listings return only masked values.

```bash
curl 'http://localhost:8080/api/v1/knowledge-bases' \
  -H 'X-API-Key: <platform-api-key>' \
  -H 'X-Tenant-ID: 10000'
```

Platform capabilities:

| capability | permission |
| --- | --- |
| `system_tenants_read` | List, search, and view all spaces |
| `system_tenants_manage` | Create, update, delete spaces, and apply global space configuration |
| `system_settings_read` | Read system settings |
| `system_settings_manage` | Update and reset system settings |
| `system_runtime_read` | View runtime queues and tasks |
| `system_runtime_manage` | Retry, run immediately, cancel, and delete runtime tasks |
| `system_audit_read` | Read platform audit logs |

Platform keys can also carry existing space capabilities, such as `retrieve`, `ingest`, `manage_kbs`; these capabilities apply to the space specified by `X-Tenant-ID` in the request.

## API Key Principal: isolation boundaries and security notes

`api-principal-config` controls how `X-API-Key` requests are mapped to an end-user **Principal**. Please understand the following boundaries before choosing a mode.

### Principal isolation scope (current implementation)

The Principal is **only** used to isolate the following capabilities per end user:

- **Conversation Sessions** (creation, listing, and reading are separated by external user; **`tenant`-only** mode still shares a space-level Session)
- **MCP OAuth** access tokens (different external users under the same space are each authorized separately; tokens are not shared)
- In-conversation flows tied to the end user, such as MCP OAuth prompts and MCP tool approvals

The Principal **does not** narrow the API Key's HTTP route permissions: route access is controlled by the key's `role`; the space-level RBAC role matches the `role`. Fine-grained access to resources such as knowledge bases and agents is further governed by the KB guard.

### Modes and security assumptions

| mode | applicable scenario | security assumption |
| ---- | -------- | -------- |
| `tenant` | No per-user MCP needs | The whole space shares a single MCP OAuth identity |
| `direct_header` | Trusted server-to-server only | The user ID comes from a request header supplied by the caller, **which can be forged by any caller holding the API Key** (impersonating another external user and sharing/hijacking their MCP OAuth authorization). **Forbidden** for use with end users or untrusted clients; if it must be used, enable `require_direct_header` and ensure the API Key is stored only on a trusted backend |
| `signed_token` | End-user-facing integrations (**recommended**) | The business backend issues a short-lived HS256 JWT for the external user using `hmac_secret`; an invalid or missing token returns 401 and **does not fall back** to a space-level Principal |

In `direct_header` mode, if the user ID header is not provided: with `require_direct_header=false`, it falls back to a space-level Principal; with `require_direct_header=true`, it returns 401.

## GET `/tenants/:id/api-principal-config` - Get the API Key principal configuration

Returns the configuration for how space-level API Key requests are mapped to an end-user Principal. **Requires Owner permission**.

**Response fields**:

| Field | Type | Description |
| ---- | ---- | ---- |
| mode | string | `tenant` / `direct_header` / `signed_token` |
| direct_header_name | string | Header name used for passing the user ID directly, default `X-External-User-ID` |
| signed_token_header_name | string | Header name for signed token mode, default `X-External-User-Token` |
| require_direct_header | bool | Whether the user ID header is mandatory in `direct_header` mode |
| has_hmac_secret | bool | Whether an HMAC secret is configured (plaintext is not returned) |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants/10000/api-principal-config' \
--header 'Authorization: Bearer <token>'
```

**Response**:

```json
{
  "success": true,
  "data": {
    "mode": "signed_token",
    "direct_header_name": "X-External-User-ID",
    "signed_token_header_name": "X-External-User-Token",
    "require_direct_header": false,
    "has_hmac_secret": true
  }
}
```

## PUT `/tenants/:id/api-principal-config` - Update the API Key principal configuration

Updates the Principal mapping method for API Key requests. **Requires Owner permission**.

**Request body**:

| Field | Type | Description |
| ---- | ---- | ---- |
| mode | string | Required, `tenant` / `direct_header` / `signed_token` |
| direct_header_name | string | Optional |
| signed_token_header_name | string | Optional |
| require_direct_header | bool | Optional; whether a missing header returns 401 in `direct_header` mode |
| hmac_secret | string | Optional; the HMAC key for `signed_token` mode — if omitted, the existing value is retained |

`hmac_secret` must be provided the first time `signed_token` mode is enabled.

External user JWT requirements: HS256 signature, `aud=weknora`, must include `sub` and `tenant_id`, and validity period no longer than 24 hours.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/tenants/10000/api-principal-config' \
--header 'Authorization: Bearer <token>' \
--header 'Content-Type: application/json' \
--data '{
  "mode": "direct_header",
  "direct_header_name": "X-External-User-ID",
  "require_direct_header": true
}'
```

## GET `/tenants` - Get list of spaces

Returns the space corresponding to the current authentication context (a single entry for regular users; admins still only get their own space).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "items": [
            {
                "id": 10002,
                "name": "weknora",
                "description": "weknora workspaces",
                "api_key": "sk-An7_t_izCKFIJ4iht9Xjcjnj_MC48ILvwezEDki9ScfIa7KA",
                "status": "active",
                "retriever_engines": {
                    "engines": [
                        {
                            "retriever_engine_type": "postgres",
                            "retriever_type": "keywords"
                        },
                        {
                            "retriever_engine_type": "postgres",
                            "retriever_type": "vector"
                        }
                    ]
                },
                "business": "wechat",
                "storage_quota": 10737418240,
                "storage_used": 0,
                "created_at": "2025-08-11T20:52:58.05679+08:00",
                "updated_at": "2025-08-11T20:52:58.060495+08:00",
                "deleted_at": null
            }
        ]
    },
    "success": true
}
```

## GET `/tenants/kv/:key` - Get space KV configuration

Gets a KV configuration item for the current space. **The space ID is obtained from the authentication context** (i.e., determined by the space associated with the `X-API-Key` / Bearer Token); it is neither required nor accepted in the URL.

**Path parameters**:

| Field | Type   | Description                                           |
| ----- | ------ | ------------------------------------------------------ |
| key   | string | Configuration key name (see the list of supported keys below; unsupported keys return 400) |

**Supported key values**:

| key                    | Description                          |
| ---------------------- | ----------------------------- |
| `agent-config`         | Agent configuration (max iterations, temperature, system prompt, available tools, etc.) |
| `web-search-config`    | Web search configuration                 |
| `conversation-config`  | Standard-mode session/conversation configuration        |
| `prompt-templates`     | System prompt templates (read-only, localized by user language) |
| `parser-engine-config` | Parser engine configuration (e.g. MinerU)    |
| `storage-engine-config`| Storage engine configuration (Local/MinIO/COS) |
| `chat-history-config`  | Chat history indexing configuration             |
| `retrieval-config`     | Global retrieval configuration                 |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/tenants/kv/agent-config' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response (using `agent-config` as an example)**:

```json
{
    "data": {
        "max_iterations": 10,
        "allowed_tools": ["knowledge_search", "web_search"],
        "temperature": 0.3,
        "system_prompt": "...",
        "use_custom_system_prompt": false,
        "available_tools": [
            { "name": "knowledge_search", "label": "Knowledge base search", "description": "..." }
        ],
        "available_placeholders": [
            { "name": "web_search_status", "label": "Web search status", "description": "..." }
        ]
    },
    "success": true
}
```

On failure (unsupported key):

```json
{ "success": false, "error": "unsupported key" }
```

## PUT `/tenants/kv/:key` - Update space KV configuration

Updates a KV configuration item for the current space. **The space ID is obtained from the authentication context**; the request body structure varies by `key`. `prompt-templates` is read-only and does not support PUT.

**Path parameters**:

| Field | Type   | Description                          |
| ----- | ------ | ----------------------------- |
| key   | string | Configuration key name (see the supported list in the GET endpoint, except `prompt-templates`) |

**Request (using `agent-config` as an example)**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/tenants/kv/agent-config' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "max_iterations": 20,
    "temperature": 0.3,
    "system_prompt": ""
}'
```

**Response**:

```json
{
    "data": {
        "max_iterations": 20,
        "allowed_tools": ["knowledge_search", "web_search"],
        "temperature": 0.3,
        "system_prompt": "",
        "use_custom_system_prompt": false
    },
    "message": "Agent configuration updated successfully",
    "success": true
}
```

**Constraints**:

- `agent-config`: `max_iterations` range `(0, 30]`; `temperature` range `[0, 2]`.
- `web-search-config`: `max_results` range `[1, 50]`.
- `conversation-config`: includes several threshold validations (e.g. `keyword_threshold` / `vector_threshold` ∈ `[0, 1]`, `rerank_threshold` ∈ `[-10, 10]`, `temperature` ∈ `[0, 2]`, `max_completion_tokens` ∈ `[1, 100000]`, etc.).
- `retrieval-config`: `embedding_top_k` / `rerank_top_k` ∈ `[0, 200]`; threshold ranges as above.
- `storage-engine-config`: `default_provider` must be within the list allowed by `STORAGE_ALLOW_LIST`.
- `chat-history-config`: when enabled with `embedding_model_id` set but not yet associated with a knowledge base, a hidden knowledge base is automatically created and its ID is written into the configuration.
