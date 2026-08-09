# API Reference: System and Platform Administration

This group covers deployment-level endpoints: reading system information, plus the platform control panel exclusive to system administrators (global settings, runtime queues, platform API keys, cross-tenant audit, password resets). See [Platform Administration and System Administrators](../03-features/20-platform-admin.md) for feature details.

Route registration: `RegisterSystemAdminRoutes` and `RegisterSystemRoutes` in `internal/router/routes_auth_tenant.go`. Handlers: `internal/handler/system.go`, `internal/handler/audit_log.go`.

The entire `/system/admin/*` group is guarded by `SystemAdmin()`; platform API keys are scoped by capability (`system_settings_read/manage`, `system_runtime_read/manage`, `system_tenants_read/manage`, `system_audit_read`).

## System Information (/api/v1/system)

Handler: `internal/handler/system.go`. API key: `manage_vector_stores`/full. Responses in this group use the `{"code":0,"msg":"success","data":...}` wrapper.

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
