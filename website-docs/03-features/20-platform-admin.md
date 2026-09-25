# Platform Management and System Administrators

System administrators manage the global settings, task queues, platform API keys, and cross-space audit of the entire WeKnora deployment. A space Owner manages a single workspace; the two identities are granted separately. For space roles, see [Tenants, Users, and Auth](01-tenant-auth.md).

The management scope of the two identities is as follows:

| | Space Owner | System Administrator |
| --- | --- | --- |
| Scope | A single workspace | The entire deployment |
| How obtained | Obtained when creating a space, or transferred by the previous Owner | Granted by an existing system administrator; the first one is bootstrapped via an environment variable |
| What it manages | Space members, models, knowledge bases, integrations, space audit | Global system settings, task queues, platform API keys, cross-space audit, resetting user passwords |
| Does it stack automatically | — | **No**: being an Owner in some space doesn't imply being a system administrator, and vice versa |

Cross-space data access is controlled separately by `CanAccessAllTenants`. System administrator status does not automatically grant access to knowledge base content in other spaces.

<Screenshot
  src="/screenshots/settings-system-admin.png"
  caption="Platform console: system settings, task queue, platform API keys, and system audit log"
  hint="Open 'Settings' as a system administrator, showing the four sections at the bottom of the sidebar that are visible only to system administrators; the system settings page can be shown as the body." />

## Setting up the first system administrator {#_1-where-the-first-system-administrator-comes-from}

A new deployment needs its first system administrator set up first:

1. First, register an account through the normal flow;
2. Set `WEKNORA_BOOTSTRAP_SYSTEM_ADMIN_EMAIL=<that account's email>` on the app service, and restart;
3. On startup, only if the deployment has no system administrator yet, the user matching that email is made a system administrator.

The bootstrap flow only promotes an existing account. If the email isn't registered yet, a warning is logged and the check runs again on the next restart; a misconfiguration does not block service startup. Once the deployment has a system administrator, this variable no longer grants anything.

After that, system administrators can be added or revoked in the UI. You can't revoke yourself, and you can't revoke the last system administrator. The corresponding endpoints are `POST /system/admin/promote` and `/revoke`; revoking a user who's already not an admin again returns 200, and the audit record marks `changed=false` to show that no privilege change happened.

## Using the platform console {#_2-what-the-console-can-do}

System administrators can see the following management sections in the "Settings" sidebar:

| Section | Purpose | Endpoint |
| --- | --- | --- |
| System Settings | Global runtime toggles (registration mode, space policy, concurrency, SSRF whitelist, etc.), applied according to each setting's own effect rule — see the settings table below | `GET/PUT/DELETE /system/admin/settings[/:key]` |
| Model Catalog | View catalog models and their sources, edit or add models, or bulk-edit them as JSON (changes take effect on save), and restore from version history; see [Model Catalog Management](06-models.md#model-catalog-maintenance-by-system-administrators) for details | `/system/admin/model-catalog*` |
| Task Queue | View real-time backlog for each asynq queue, retry/archive/delete individual tasks, bulk-clear archived tasks; returns `available=false` in Lite mode | `/system/admin/runtime/queues*` |
| Platform API Key | Platform-scoped keys for control-plane automation, with capabilities including `system_tenants_read/manage`, `system_settings_read/manage`, `system_runtime_read/manage`, `system_audit_read` | `/system/admin/api-keys` |
| System Audit Log | Platform-level events with `tenant_id = 0` (settings changes, admin promotion/revocation, queue operations, etc.). The space-level audit endpoint filters by tenant and cannot see these rows | `GET /system/admin/audit-log` |

System administrators can also perform the following operations:

- **Reset a user's password** (`POST /system/admin/users/reset-password`): replaces the target user's local password and revokes all their sessions. **You cannot reset your own** — self-service password changes still require the old password;
- **Bulk-apply the default storage quota** (`POST /system/admin/tenants/apply-default-storage-quota`): writes the current default quota to all existing spaces. This operation updates the quota data of existing spaces.

### Creating users

Since v0.8.2, system administrators can click "Create user" in "Settings → System Settings → Accounts & access" to provision a local account: fill in a username (2–50 characters) and an email, then either keep "Auto-generate a random password" on, or turn it off and enter a password that satisfies the password policy. This suits deployments where public registration is disabled and administrators provision accounts centrally. Space assignment follows `auth.default_tenant_mode`: `create_personal` creates a personal space, while `tenantless` leaves the user waiting to join a space.

<Screenshot
  src="/screenshots/system-admin-create-user.png"
  caption="System administrator creating a user: account details and the one-time password display"
  hint="Open the 'Create user' dialog under 'Settings → System Settings → Accounts & access', showing the username, email, and 'Auto-generate a random password' toggle; optionally add the result page shown after creation, with the one-time password and the 'Copy account details' button." />

An auto-generated password is shown only in the result of that creation, so copy the full account details right away; the dialog can't be closed until you confirm, and the plaintext can't be retrieved again once it's closed. Every creation is written to the system audit log (`system.user_created`). A duplicate identity returns the existing user without changing its password; creation is rejected when the email and the username point to different users. For the endpoint, see the [System API](../04-api/02-api-system.md).

Creating a user and bootstrapping the first administrator are separate operations: bootstrap still only promotes an existing user and never creates accounts.

## Runtime settings reference {#_3-runtime-editable-system-settings}

Supported runtime settings can be changed in the console. Setting values stored in the database take precedence over environment variables, and most take effect immediately; for settings that affect new resources or require rebuilding the runtime, follow the notes in the table.

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

The following toggles also follow the database-first rule:

| Key | Default | Purpose |
| --- | --- | --- |
| `auth.complex_password_enabled` | false | New registrations/new passwords must contain uppercase and lowercase letters, digits, and special characters |
| `tenant.auto_accept_invitation` | false | Inviting an already-registered user by email writes the membership directly |
| `sandbox.docker_enabled` | false | Enables Docker sandbox configuration and instance creation |

`tenant.auto_create_api_key` is off by default. Integrations that depend on the old registration response can enable this compatibility switch so that new spaces automatically create a full-access key and return it in plaintext.

::: tip Resetting runtime settings
Once a setting has been saved in the console, changing the corresponding environment variable does not override the database value. When troubleshooting a configuration that isn't taking effect, check the runtime settings first; resetting a setting (`DELETE /system/admin/settings/:key`) deletes the database override and falls back to the environment variable or built-in default again.
:::

## Related documentation {#related}

- In-space role tiers and API keys: [Tenants, Users, and Auth](01-tenant-auth.md)
- Queue topology and worker pools: [Async Task System](../02-architecture/05-async-tasks.md)
- Audit logs and tracing: [Observability and Auditing](16-observability.md)
- Endpoint reference: [System and Platform Administration API](../04-api/02-api-system.md)

## Implementation reference

- `cmd/server/bootstrap.go`: first system administrator bootstrap.
- `frontend/src/config/settingsAccess.ts`: platform settings sections.
- `internal/application/service/system_setting.go`: runtime settings registry.
