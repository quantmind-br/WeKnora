# API Reference: System and Platform Administration

Provides deployment-level system information and platform administration endpoints, including global settings, task queues, platform API keys, cross-space audit, and user password resets. See [Platform Administration and System Administrators](../03-features/20-platform-admin.md) for feature details.

The entire `/system/admin/*` group is guarded by `SystemAdmin()`; platform API keys are scoped by capability (`system_settings_read/manage`, `system_runtime_read/manage`, `system_tenants_read/manage`, `system_audit_read`).

## System Information (/api/v1/system)

Handler: `internal/handler/system.go`. API key: `manage_vector_stores`/full. Responses in this group use the `{"code":0,"msg":"success","data":...}` wrapper.

### GET /api/v1/system/capabilities

Viewer+; readable with an API Key. Returns `{code:0,data:{edition,capabilities}}`, where each capability reports supported/reason. The frontend uses the deployment edition, the actually registered routes, and the Docker switch to control menu entries; hiding a menu does not replace backend permission checks.

```bash
curl "$BASE/api/v1/system/capabilities" -H "Authorization: Bearer $TOKEN"
```

`settings.sandbox.host` in `capabilities` indicates whether the current deployment can use the host operating system sandbox; currently only the native macOS desktop app can report supported.

### POST /api/v1/system/host-project-dir

Purpose: open the system folder picker on the machine running WeKnora so a new session can bind a local project directory (since v0.8.2, native desktop app only). Permission: Viewer+, JWT only; API Keys are always rejected. No request body.

Response: 200 `{"code":0,"msg":"success","data":{"dir":"/Users/me/project"}}`; `dir` is an empty string when the user cancels the selection; non-desktop deployments return 404.

### GET /api/v1/system/info

Purpose: system version and engine information. Permission: Viewer+.

Response: 200 `{"code":0,"msg":"success","data":{version,edition,commit_id,build_time,go_version,keyword_index_engine,vector_store_engine,graph_database_engine,minio_enabled,db_version,started_at,uptime_seconds}}`

```bash
curl $BASE/api/v1/system/info -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/parser-engines

Purpose: list of parser engines and DocReader connection status. Permission: Viewer+.

Response: 200 `{"code":0,"msg":"success","data":[...],"docreader_addr","docreader_transport","connected"}`

```bash
curl $BASE/api/v1/system/parser-engines -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/parser-engines/check

Purpose: probe a parser engine using the given configuration (`types.ParserEngineConfig` request body). Permission: Admin+.

Response: 200, same as above.

```bash
curl -X POST $BASE/api/v1/system/parser-engines/check -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{}'
```

### POST /api/v1/system/docreader/reconnect

Purpose: reconnect to DocReader. Permission: Admin+. Request body: `{"addr":"host:port"}` (`binding:"required"`).

Response: 200 `{"code":0,"msg":"Connected successfully",...,"connected":true}`

```bash
curl -X POST $BASE/api/v1/system/docreader/reconnect -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"addr":"docreader:50051"}'
```

### GET /api/v1/system/storage-engine-status

Purpose: object storage engine availability. Permission: Viewer+.

Response: 200 `{"code":0,"msg":"success","data":{"engines":[{name,allowed,available,description}],"allowed_providers":[...],"minio_env_available":bool}}`

```bash
curl $BASE/api/v1/system/storage-engine-status -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/storage-engine-check

Purpose: validate storage configuration (probed after SSRF protection). Permission: Admin+. Request body: `provider` (required, `minio/cos/tos/s3/oss/ks3/obs`) + the corresponding `minio|cos|tos|s3|oss|ks3|obs` configuration object.

Response: 200 `{"code":0,"data":{"ok","message","bucket_created"}}`

```bash
curl -X POST $BASE/api/v1/system/storage-engine-check -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"provider":"minio","minio":{"endpoint":"minio:9000"}}'
```

## System Administration (/api/v1/system/admin, SystemAdmin only)

Group-level `SystemAdmin()` guard (always enforced, regardless of EnableRBAC); platform API keys require the corresponding `system_*` capability. Most read endpoints in this group return raw rows/arrays (no wrapper). Handlers: `internal/handler/system.go`, `internal/handler/audit_log.go`.

### POST /api/v1/system/admin/promote

Purpose: grant SystemAdmin. Request body: `user_id` (UUID, preferred) or `email` (one of the two).

Response: 200 `UserInfo` (raw object).

```bash
curl -X POST $BASE/api/v1/system/admin/promote -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"admin@ex.com"}'
```

### POST /api/v1/system/admin/revoke

Purpose: revoke SystemAdmin. Request body: `{"user_id":"..."}` (`binding:"required"`).

Response: 200 `UserInfo`

```bash
curl -X POST $BASE/api/v1/system/admin/revoke -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"user_id":"u-1"}'
```

### GET /api/v1/system/admin/list

Purpose: list of SystemAdmins. Query parameters: `offset` (default 0), `limit` (default 50, max 200).

Response: 200 `{"total":N,"admins":[UserInfo]}`

```bash
curl $BASE/api/v1/system/admin/list -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/users/create

System administrators only; this endpoint is not available to platform API Keys. Request fields: username (2–50 characters), email (a valid email address), password (optional, or null to auto-generate). An explicit empty string still goes through password policy validation and is not treated as auto-generation.

| HTTP status | Response and meaning |
| --- | --- |
| 201 | `{user:UserInfo,generated_password?}`, created; the password is returned only when auto-generated |
| 200 | `{user:UserInfo}`, existing identity; the account and password are not modified |
| 400 | Invalid parameters or password policy not met |
| 409 | The email and username belong to different identities |

This is the raw response object, with no success/data wrapper and no idempotent field. Use the HTTP status to tell a new account from an existing one. Space assignment follows auth.default_tenant_mode.

```bash
curl -i -X POST "$BASE/api/v1/system/admin/users/create" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"username":"alice","email":"alice@example.com"}'
```

### POST /api/v1/system/admin/users/reset-password

Purpose: reset a user's password. Request body: `email` (`binding:"required,email"`), `new_password` (`binding:"required"`).

Response: 200 `{"message":"Password reset successfully"}`

```bash
curl -X POST $BASE/api/v1/system/admin/users/reset-password -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"email":"a@ex.com","new_password":"newpass1"}'
```

### GET /api/v1/system/admin/api-keys

Purpose: list platform API keys (masked).

Response: 200 `{"success":true,"data":[{id,name,api_key,capabilities,expires_at_unix,...}]}`

```bash
curl $BASE/api/v1/system/admin/api-keys -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/api-keys

Purpose: create a platform API key (plaintext is returned only once). Request body: `name` (non-empty), `capabilities` (list of `system_*`, required), `expires_at_unix` (optional, must be a future time).

Response: 201 `{"success":true,"data":{...,"api_key":"<plaintext>","token":"<plaintext>"}}`

```bash
curl -X POST $BASE/api/v1/system/admin/api-keys -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"ops","capabilities":["system_tenants_read"]}'
```

### DELETE /api/v1/system/admin/api-keys/:key_id

Purpose: delete a platform API key.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/system/admin/api-keys/3 -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/settings and GET /api/v1/system/admin/settings/:key

Purpose: list / read a single platform runtime setting (platform key requires `system_settings_read|manage`).

Response: 200 `[SystemSetting]` / `SystemSetting` (raw, no wrapper; fields: `key,value,value_type,description,last_modified_by,last_modified_at`).

```bash
curl $BASE/api/v1/system/admin/settings -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/system/admin/settings/:key

Purpose: update a setting (platform key requires `system_settings_manage`). Request body: `{"value":<any JSON, validated against the registry type>}` (required).

Response: 200 `SystemSetting`

```bash
curl -X PUT $BASE/api/v1/system/admin/settings/default_storage_quota -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"value":10737418240}'
```

### DELETE /api/v1/system/admin/settings/:key

Purpose: restore a setting to its default value.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/system/admin/settings/default_storage_quota -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/runtime/queues

Purpose: asynq queue depth and concurrency status (Lite mode returns `available:false`; platform key requires `system_runtime_read|manage`).

Response: 200 `{"available",upstream_concurrency,parse_concurrency,wiki_concurrency,pools,queues,model_limiter_available,models,timestamp}`

```bash
curl $BASE/api/v1/system/admin/runtime/queues -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/runtime/queues/:queue/tasks

Purpose: list queue tasks. Query parameters: `state` (`pending/active/scheduled/retry/archived/completed`), `cursor`, `page_size` (default 20, max 100).

Response: 200 `{"available","tasks":[RuntimeTaskInfo],"page_size","has_more","next_cursor"}`

```bash
curl "$BASE/api/v1/system/admin/runtime/queues/default/tasks?state=pending" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/runtime/queues/:queue/tasks/:task_id/actions/:action

Purpose: perform a task action (`action` ∈ `cancel/run_now/delete`; platform key requires `system_runtime_manage`).

Response: 200 `{"success":true}`

```bash
curl -X POST $BASE/api/v1/system/admin/runtime/queues/default/tasks/t-1/actions/cancel \
  -H "Authorization: Bearer $TOKEN"
```

### DELETE /api/v1/system/admin/runtime/queues/:queue/archived

Purpose: clear archived tasks.

Response: 200 `{"success":true,"deleted":N}`

```bash
curl -X DELETE $BASE/api/v1/system/admin/runtime/queues/default/archived -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/system/admin/tenants/apply-default-storage-quota

Purpose: bulk-apply the current default storage quota to all tenants (platform key requires `system_tenants_manage`). No request body.

Response: 200 `{"affected":N,"quota_bytes":N,"quota_gb":N}`

```bash
curl -X POST $BASE/api/v1/system/admin/tenants/apply-default-storage-quota -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/system/admin/audit-log

Purpose: platform-level audit log (rows with tenant_id=0; platform key requires `system_audit_read`). Query parameters same as tenant audit (`after_id/limit/action/outcome/actor`). Handler: `internal/handler/audit_log.go`

Response: 200 `{"success":true,"data":[AuditLog],"next_cursor":N}`

```bash
curl $BASE/api/v1/system/admin/audit-log -H "Authorization: Bearer $TOKEN"
```

## Implementation Reference

Route registration: `RegisterSystemAdminRoutes` and `RegisterSystemRoutes` in `internal/router/routes_auth_tenant.go`. Handlers: `internal/handler/system.go`, `internal/handler/audit_log.go`.


## Model Catalog (/api/v1/system/admin/model-catalog)

Accessible only from system administrator user sessions; this group of endpoints is not available to API Keys.

| Method and path | Purpose |
| --- | --- |
| `GET /system/admin/model-catalog` | Returns the current `version`, `baseline`, the admin `overlay`, the 20 most recent history versions, and the `builtin` / `deployment` / `effective` catalogs |
| `POST /system/admin/model-catalog/preview` | Validates the overlay document and returns the candidate catalog (only `effective` and the normalized `overlay`; `history` / `builtin` / `deployment` are `null`), without persisting or publishing |
| `PUT /system/admin/model-catalog` | Validates, saves a new version, and publishes it; the audit log records only version metadata |

Preview and publish use the same request body:

```json
{
  "version": 0,
  "baseline": "deployment baseline identifier returned by GET",
  "overlay": {
    "providers": {
      "openai": {
        "models": [{"id": "gpt-5", "context_window": 128000}]
      }
    }
  }
}
```

The response is the unwrapped catalog state object. Each provider entry additionally carries `model_thinking_levels` (the levels selectable once thinking is enabled, listed per chat model id, with provider mappings and protocol capabilities already merged) and `vendor_thinking_levels` (the provider's default levels, used by models without their own level configuration). An invalid document returns 400; a stale version or a deployment baseline that does not match the requesting instance returns 409. Publishing persists first and then switches the current instance; other instances sync within about 5 seconds. To roll back, publish a historical `overlay` again together with the current `version` / `baseline`. See [Model Management](../03-features/06-models.md#model-catalog-maintenance-by-system-administrators) for the full rules and limits.
