# Platform Management and System Administrators

WeKnora's permissions are split into two layers: the four **in-space** role tiers (see [Tenants, Users, and Auth](01-tenant-auth.md)), and the **platform-level** system administrator. This article covers the latter — it doesn't manage a single knowledge base, but the entire deployment.

First, let's draw the line:

| | Space Owner | System Administrator |
| --- | --- | --- |
| Scope | A single workspace | The entire deployment |
| How obtained | Becomes Owner of their own space upon registration, or via transfer | Promoted by an existing system administrator; the first one is bootstrapped via an environment variable |
| What it manages | Space members, models, knowledge bases, integrations, space audit | Global system settings, task queues, platform API keys, cross-space audit, resetting user passwords |
| Does it stack automatically | — | **No**: being an Owner in some space doesn't imply being a system administrator, and vice versa |

There's also a third flag, `CanAccessAllTenants` (cross-space access), which controls whether you can read/write another space's data — and it's also separate from system administrator status: by default, a system administrator cannot see knowledge base content inside other people's spaces.

<Screenshot
  src="/screenshots/settings-system-admin.png"
  caption="Platform console: system settings, task queue, platform API keys, and system audit log"
  hint="Open 'Settings' as a system administrator, showing the four sections at the bottom of the sidebar that are visible only to system administrators; the system settings page can be shown as the body." />

## 1. Where the first system administrator comes from

A new deployment has **no system administrators at all**. The bootstrap flow lives in `cmd/server/bootstrap.go`:

1. First, register an account through the normal flow;
2. Set `WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL=<that account's email>` on the app service, and restart;
3. On startup, `bootstrapSystemAdmin()` checks — **only if the current deployment has zero system administrators** — and promotes the user matching that email to system administrator.

A few deliberate design choices:

- **It will not create a user**: if the email isn't registered yet, it just logs a WARN and retries on the next restart. Account creation involves password hashing, space allocation, and auditing, which isn't something to shortcut inside a startup hook;
- **Idempotent, and self-invalidating**: once a system administrator exists, this variable no longer grants anything — this prevents a privilege an admin just revoked in the UI from quietly being restored on the next restart. So it's safe to leave in the deployment manifest long-term;
- **Failure doesn't block startup**: the whole bootstrap process is best-effort, so a misconfigured variable still lets the service come up, and you can fix it afterward.

After that, adding/removing admins is done through the UI (corresponding to `POST /system/admin/promote` / `revoke`). Revocation has two safeguards: **you can't revoke yourself**, and **you can't revoke the last system administrator** — otherwise the platform would permanently lose system-level administrative capability. Revoking a user who's already not an admin returns 200 (idempotent), but the audit record shows `changed=false`, which makes it easy to distinguish a real revocation from a no-op afterward.

## 2. What the console can do

The UI entry point is at the bottom of the "Settings" sidebar, visible only to system administrators (the frontend allowlist is `SYSTEM_ADMIN_SETTINGS_SECTIONS` in `frontend/src/config/settingsAccess.ts`), with four sections:

| Section | Purpose | Endpoint |
| --- | --- | --- |
| System Settings | Global runtime toggles (registration mode, space policy, concurrency, SSRF whitelist, etc.), taking effect immediately after changes — see §9.3 | `GET/PUT/DELETE /system/admin/settings[/:key]` |
| Task Queue | View real-time backlog for each asynq queue, retry/archive/delete individual tasks, bulk-clear archived tasks; returns `available=false` in Lite mode | `/system/admin/runtime/queues*` |
| Platform API Key | Platform-scoped keys for control-plane automation, with capabilities including `system_tenants_read/manage`, `system_settings_read/manage`, `system_runtime_read/manage`, `system_audit_read` | `/system/admin/api-keys` |
| System Audit Log | Platform-level events with `tenant_id = 0` (settings changes, admin promotion/revocation, queue operations, etc.). The space-level audit endpoint filters by tenant and cannot see these rows | `GET /system/admin/audit-log` |

Two other capabilities aren't in the table above but also belong to system administrators:

- **Reset a user's password** (`POST /system/admin/users/reset-password`): replaces the target user's local password and revokes all their sessions. **You cannot reset your own** — self-service password changes still require the old password;
- **Bulk-apply the default storage quota** (`POST /system/admin/tenants/apply-default-storage-quota`): writes the current default quota to all existing spaces. This lives under `/tenants` rather than `/settings` because it modifies space data, not a setting entry.

## 3. Runtime-editable system settings

`internal/application/service/system_setting.go` maintains a registry of keys that can be changed in the console — **database values override environment variables**, and most changes take effect immediately without a restart:

| Key | Type | Default | When it takes effect |
| --- | --- | --- | --- |
| `auth.registration_mode` | `self_serve` / `invite_only` | `self_serve` | Immediately |
| `auth.default_tenant_mode` | `create_personal` / `tenantless` | `create_personal` | Only affects users who register afterward |
| `tenant.self_service_creation_enabled` | bool | `true` | Immediately |
| `tenant.max_owned_per_user` | int | `10` (0 = use the built-in default, negative = no limit) | Read each time a space is created |
| `tenant.default_storage_quota_gb` | int | `10` | **Only read when a new space is created**, not written back to existing spaces |
| `tenant.auto_create_api_key` | bool | `false` | Read each time a space is created |
| `ssrf.whitelist` | String list | Empty | Immediately (`SSRF_WHITELIST_EXTRA` is still maintained solely by the deployer, and isn't overridden here) |
| `asynq.core/postprocess/enrichment/maintenance/shared/wiki_concurrency` | int | See [Async Task System](../02-architecture/05-async-tasks.md) | Each worker pool is reassembled |
| `model.max_concurrency` | int | `32` | Immediately |

`tenant.auto_create_api_key` is a compatibility switch: the old behavior — "creating a space automatically issues a full-access key and returns it in plaintext in the response" — is a breaking change, so integrations that depend on it can turn on this switch to fall back to the old behavior; it's off by default.

::: warning Configuration sources aren't limited to environment variables
Once any of the keys above has been changed in the console, a row now exists in the database, and **changing the environment variable afterward no longer has any effect**. When troubleshooting "I changed the env var but it's not taking effect," check this first; resetting a setting (`DELETE /system/admin/settings/:key`) deletes the DB row and falls back to the environment variable or built-in default again.
:::

## Related

- In-space role tiers and API keys: [Tenants, Users, and Auth](01-tenant-auth.md)
- Queue topology and worker pools: [Async Task System](../02-architecture/05-async-tasks.md)
- Audit logs and tracing: [Observability and Auditing](16-observability.md)
- Endpoint reference: [API Reference: Models and System](../04-api/02-api-model-system.md)
