# API Reference: System and Platform Administration

Provides deployment-level system information and platform administration endpoints, including global settings, task queues, platform API keys, cross-space audit, and user password resets. See [Platform Administration and System Administrators](../03-features/20-platform-admin.md) for feature details.

The entire `/system/admin/*` group is guarded by `SystemAdmin()`; platform API keys are scoped by capability (`system_settings_read/manage`, `system_runtime_read/manage`, `system_tenants_read/manage`, `system_audit_read`).

## System Information (/api/v1/system)

Handler: `internal/handler/system.go`. API key: `manage_vector_stores`/full. Responses in this group use the `{"code":0,"msg":"success","data":...}` wrapper.

### GET /api/v1/system/capabilities

Viewer+；API Key 可读。返回 `{code:0,data:{edition,capabilities}}`，每个 capability 给出 supported/reason。前端据部署版本、实际注册路由和 Docker 开关控制菜单入口；隐藏菜单不代替后端权限校验。

```bash
curl "$BASE/api/v1/system/capabilities" -H "Authorization: Bearer $TOKEN"
```

`capabilities` 中的 `settings.sandbox.host` 表示当前部署能否使用本机操作系统沙箱，目前仅 macOS 原生桌面应用可能为 supported。

### POST /api/v1/system/host-project-dir

用途：在运行 WeKnora 的本机弹出系统文件夹选择框，供新会话绑定本机项目目录（v0.8.2 起，仅原生桌面应用）。权限：Viewer+，仅 JWT，API Key 一律拒绝。无请求体。

响应：200 `{"code":0,"msg":"success","data":{"dir":"/Users/me/project"}}`，用户取消选择时 `dir` 为空字符串；非桌面部署返回 404。

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

仅系统管理员；此接口不开放给 platform API Key。请求字段：username（2–50 字符）、email（合法邮箱）、password（可选或 null 自动生成）。显式空字符串仍要经过密码策略校验，不视为自动生成。

| HTTP 状态 | 响应与含义 |
| --- | --- |
| 201 | `{user:UserInfo,generated_password?}`，新建；仅自动生成时返回密码 |
| 200 | `{user:UserInfo}`，已有身份，不修改账号或密码 |
| 400 | 参数或密码策略不满足 |
| 409 | 邮箱与用户名对应不同身份 |

这是原始响应对象，没有 success/data 包装，也没有 idempotent 字段。用 HTTP 状态区分新增与已有账号。空间分配遵循 auth.default_tenant_mode。

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

## 实现参考

路由注册：`internal/router/routes_auth_tenant.go` 的 `RegisterSystemAdminRoutes` 与 `RegisterSystemRoutes`。Handler：`internal/handler/system.go`、`internal/handler/audit_log.go`。


## 模型目录（/api/v1/system/admin/model-catalog）

仅系统管理员用户会话可访问，API Key 不开放此组接口。

| 方法与路径 | 作用 |
| --- | --- |
| `GET /system/admin/model-catalog` | 返回当前 `version`、`baseline`、管理员 `overlay`、最近 20 个历史版本，以及 `builtin` / `deployment` / `effective` 目录 |
| `POST /system/admin/model-catalog/preview` | 校验覆盖文档并返回候选目录（仅 `effective` 与规范化后的 `overlay`，`history` / `builtin` / `deployment` 为 `null`），不持久化、不发布 |
| `PUT /system/admin/model-catalog` | 校验、保存新版本并发布；审计仅记录版本元信息 |

预览和发布使用相同请求体：

```json
{
  "version": 0,
  "baseline": "GET 返回的部署基线标识",
  "overlay": {
    "providers": {
      "openai": {
        "models": [{"id": "gpt-5", "context_window": 128000}]
      }
    }
  }
}
```

响应为未包装的目录状态对象。每个厂商条目额外带 `model_thinking_levels`（按对话模型 id 列出开启思考后可选的等级，已合并厂商映射与协议能力）和 `vendor_thinking_levels`（未单独配置等级的模型所用的厂商默认等级）。非法文档返回 400；版本过期或请求实例的部署基线不一致返回 409。发布先持久化再切换本实例，其他实例约 5 秒内同步。回滚使用历史 `overlay` 配合当前 `version` / `baseline` 再次发布。完整规则和限制见[模型管理](../03-features/06-models.md#系统管理员维护模型目录)。
