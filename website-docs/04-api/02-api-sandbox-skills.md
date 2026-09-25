# API Reference: Sandboxes, Skills & Personal Variables

Manage sandbox configurations, the skill catalog, installation tasks, session interactive terminals and personal environment variables. All paths are prefixed with `/api/v1`; in the examples, `$BASE` is the service address and `$TOKEN` is the current user's Bearer token. For UI steps, see [Skill Catalog & Sandbox](../03-features/22-skills-sandbox.md).

## Permissions

| Resource | Read | Write / Check |
| --- | --- | --- |
| Sandbox configuration list/details | Viewer+; API Key must be full-access | Admin+; API Key must be full-access |
| Sandbox instance inventory, installed skills and files | Admin+ | Admin+; API Key must be full-access |
| Skill catalog list | Viewer+, JWT | Adding/removing catalog entries, installing and browsing package files require Admin+; API Keys for the write group must be full-access |
| `/skills` available skills | Viewer+, JWT | Read-only, query parameter sandbox_config_id; shared agents also pass agent_id and agent_source_tenant_id |
| Session interactive terminal | Session owner, login JWT only | Obtain a ticket, then establish the WebSocket |
| `/me/env-vars*` | The signed-in current caller | Only modifies your own; the server derives the identity; no extra admin gate, uses Bearer JWT |

Cross-space IDs do not grant access. The personal variable endpoints cannot replace the space skill management endpoints.

## Sandbox configuration

| Method | Path | Request / Response |
| --- | --- | --- |
| GET | `/sandbox-configs` | 200 `{success,data:[ConfigResponse],workspace_scripts_disabled}` |
| POST | `/sandbox-configs` | `{name,description?,config}`; 201 `{success,data:ConfigResponse}` |
| GET | `/sandbox-configs/:id` | 200 `{success,data:ConfigResponse}` |
| PUT | `/sandbox-configs/:id` | `{name,description?,config}`, name is required; 200 same as details |
| DELETE | `/sandbox-configs/:id` | Optional `force=true`; 200 `{success:true}` |
| GET | `/sandbox-configs/:id/sandboxes` | 200 `{success,data:SandboxInventory}`, including usage and associated agents |
| PUT | `/sandbox-configs/workspace-policy` | `{"scripts_disabled":true}`; returns success/workspace_scripts_disabled |
| POST | `/sandbox-configs/templates/query` | `{config,config_id?,ensure_standard?,replace_standard?,ensure_desktop?,replace_desktop?}`; queries remote templates and can create/replace the standard or desktop template; replace requires config_id |

ConfigResponse is `{id,name,description,sandbox_type,config,created_at,updated_at}`, with credentials masked. Edits are applied on top of the saved configuration; masked placeholder values keep the old credentials, and `skill_image` is maintained by the installation service and cannot be replaced by the client.

### config fields

| Field | Type | Description |
| --- | --- | --- |
| `sandbox_type` | string | docker/cube/e2b; local has been removed; the Lite desktop host cannot be saved as a configuration |
| `default_timeout_sec` | int | Execution timeout; 0 uses the built-in default |
| `terminal_idle_disconnect_sec` | int | How long a terminal/desktop can be idle before disconnecting; 0 is the default of 900 seconds, effective range 60 seconds–24 hours |
| `desktop_enabled` | bool | Declares that the selected template is a Cube/E2B desktop template; cannot be switched once skills are installed |
| `allow_private_endpoints` | bool | Allows private-network cluster addresses; does not allow link-local/cloud metadata |
| `env_vars` | map[string]string | Environment for this sandbox configuration; values are stored encrypted |
| `skill_rollout` | string | next_turn (default)/new_session |
| `network` | object | Cube/E2B network policy, see below |
| `cube` / `e2b` / `docker` | object | Connection configuration matching sandbox_type |
| `volume_mount` | object | Optional volume configuration; whether it takes effect depends on backend capability, so the presence of the field alone does not mean it is supported |
| `skill_image` | object | Snapshot information maintained by the installation service, read-only |

volume_mount fields include enabled, mount_path, provider, volume_id and volume_name; volume_owner_fingerprint is managed by the server. The volume configuration and backend capability together determine whether it is mounted; keep the ownership information returned by the server when editing.

Backend configuration:

| Backend | Fields |
| --- | --- |
| cube | api_url, proxy_url, sandbox_domain, template_id; api_key as required by the cluster; http_timeout_sec, cube_sandbox_ttl_seconds, dns_servers |
| e2b | api_key and template_id are required; api_url, sandbox_domain and proxy_url for self-hosting; http_timeout_sec, e2b_sandbox_ttl_seconds |
| docker | image is required; host (empty for the local socket), tls_cert_path (required for TCP), cpu_limit, memory_limit_mb, pids_limit, network_mode (bridge/none), runtime, idle_ttl_seconds, http_timeout_sec |

```bash
curl -X POST "$BASE/api/v1/sandbox-configs" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"name":"E2B workspace","config":{"sandbox_type":"e2b","e2b":{"api_key":"<e2b-key>","template_id":"<template-id>"}}}'

curl "$BASE/api/v1/sandbox-configs/cfg-1/sandboxes" \
  -H "Authorization: Bearer $TOKEN"
```

Changing the backend identity or deleting a configuration checks remote instances. A 409 error.code can be `sandboxes_still_live`, `sandbox_inventory_unverifiable`, `skill_snapshot_release_failed` and so on; 423 means the configuration is being modified by another request. `force=true` only allows deletion when the inventory cannot be verified; it does not skip confirmed active/paused instances. The usage data in the response helps locate the related sessions and agents.

### Network policy

`network`:

| Field | Description |
| --- | --- |
| `deny_egress_by_default` | false allows egress by default; true denies by default |
| `allow_out` | IPv4/CIDR/domain or single-label wildcard domain; domains are used in default-deny mode |
| `deny_out` | IPv4/CIDR deny list |
| `cube_rules` | Cube name/scheme/sni/host/methods/path/deny/audit/inject rules; order matters |
| `e2b_host_rules` | E2B host/headers rules; the host must also be listed in allow_out |
| `allow_public_inbound` | Accepted for legacy input and cleared on save; inbound always requires credentials |

`cube_rules[].inject` is `[{header,secret,format}]` and `e2b_host_rules[].headers` is header→secret; secret fields are stored encrypted and masked. Docker uses `docker.network_mode` and does not support these L7 rules.

```json
{
  "deny_egress_by_default": true,
  "allow_out": ["pypi.org", "files.pythonhosted.org"],
  "deny_out": []
}
```

### POST /system/sandbox-check

Request `{config,config_id?,deep?}`; config_id can be used to restore saved masked credentials. `deep=false` performs a connection check; true also runs a temporary script, and remote backends actually create and destroy a sandbox, which may incur backend usage.

A probe that executes returns `{success:true,data:{ok,provider,checks,capabilities}}`; a failed probe can also be HTTP 200, so check data.ok. Parameter errors return 400 with code/msg. Each check contains name/ok/message/reason/latency_ms, and `ok=null` means skipped. For example, when egress is denied by default the internet probe is skipped by policy, which is not the same as passing internet access verification.

```bash
curl -X POST "$BASE/api/v1/system/sandbox-check" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"config_id":"cfg-1","deep":true}'
```

## Skill catalog

| Method | Path | Request / Response |
| --- | --- | --- |
| GET | `/skills/catalog` | 200 `{success,data:[CatalogItem]}`; CatalogItem contains `installations:[{sandbox_config_id,status,enabled,version,bundle_sha256,served?,...}]` |
| POST | `/skills/catalog` | multipart `file` or JSON `{"source":"@owner/slug"}`; 201 `{success,data:{id,name,version,description}}` |
| POST | `/skills/catalog/:id/install` | `{"sandbox_config_ids":["cfg-1","cfg-2"]}`; 202 `{success,data:{installs,errors?}}` |
| GET | `/skills/catalog/:id/files` | 200 `{success,data:[FileEntry]}` |
| GET | `/skills/catalog/:id/files/content?path=SKILL.md` | 200 `{success,data:FileContent}` |
| DELETE | `/skills/catalog/:id` | Deletes when there are no installation references; 200 `{success:true}` |

Installing to multiple sandboxes may be partially accepted: check `data.installs` and `data.errors`; HTTP 202 does not mean every installation completed. Deleting a catalog entry does not implicitly uninstall it from each sandbox.

```bash
curl -X POST "$BASE/api/v1/skills/catalog" \
  -H "Authorization: Bearer $TOKEN" -F 'file=@skill.zip'
curl -X POST "$BASE/api/v1/skills/catalog/catalog-1/install" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"sandbox_config_ids":["cfg-1"]}'
```

For source syntax, anonymous download requirements and ZIP limits, see [Skill sources](../03-features/22-skills-sandbox.md#supported-sources).

## Skills in a sandbox {#skills-in-sandbox}

The following paths are prefixed with `/sandbox-configs/:id/skills`; every operation, including reads, requires Admin+.

| Method | Suffix | Behavior |
| --- | --- | --- |
| GET | empty | `{success,data:[SkillResponse]}` |
| POST | empty | Install from a ZIP file or source JSON; 202 `{success,data:{skill_id}}` |
| GET | `/:skillId` | `{success,data:SkillResponse}` |
| POST | `/:skillId/reinstall` | Reinstall from the existing package, with optional `{"instructions":"..."}` (≤10000 characters) as installation instructions; 202 `{success,data:{skill_id}}` |
| POST | `/:skillId/stop` | Stop installation; 200 returns the skill status; can be called repeatedly once failed, and other non-installing states are rejected |
| GET | `/:skillId/guidance` | Instructions for the current installation run: `{success,data:{accepting,messages:[{id,content,status}]}}` |
| POST | `/:skillId/guidance` | Append instructions while installation is in progress `{expected_message_id,steer_id,content}`: `expected_message_id` comes from the SkillResponse `install_message_id`, `steer_id` is a client-generated UUID (retries do not append twice), and `content` is 1–10000 characters; 202 `{success:true}`, returns 409 if the installation has moved to another round |
| PATCH | `/:skillId` | `enabled?`, `envs?`; omitted fields are not modified |
| DELETE | `/:skillId` | Uninstall from this sandbox, keeping the catalog package |
| GET | `/:skillId/files` | File list |
| GET | `/:skillId/files/content?path=SKILL.md` | Read a file by relative path; traversal is rejected |
| GET | `/:skillId/install-events` | SSE installation progress |
| GET | `/:skillId/transcript` | SSE installation process events |

SkillResponse includes id/name/version/description/enabled/status/error/bundle_sha256/installed_snapshot_id/install_session_id/install_message_id/created_at/updated_at, plus `envs:[{name,description,required,is_set}]`. While an upgrade or reinstall is in progress, or after it fails, `served:{version}` indicates the previous ready version the sandbox is still serving; the skills available to agents follow that version. File responses can be UTF-8 text, base64 for small images, or binary metadata; do not assume every file has text content.

PATCH envs only updates declared variables; undeclared names are ignored, and an empty string clears the value while keeping the declaration.

```bash
curl -X PATCH "$BASE/api/v1/sandbox-configs/cfg-1/skills/skill-1" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"enabled":true,"envs":{"API_TOKEN":"<workspace-token>"}}'

curl -N "$BASE/api/v1/sandbox-configs/cfg-1/skills/skill-1/install-events" \
  -H "Authorization: Bearer $TOKEN"
```

The progress payload is `{percent,stage,log?,status?,done}`. Disconnecting does not cancel the installation; done may only mean progress can no longer be followed, and the final result is the skill status. Without Redis, the stream tells the client to poll the status instead. A 204 from transcript means installation has just started and event positioning is not ready yet; 404 can mean there are no logs or the logs have expired, and does not by itself indicate installation failure.

`GET /skills?sandbox_config_id=cfg-1` returns skill metadata usable by agents, `{success,data:[{name,description}],skills_available}`, which differs from the full set of installation records admins see.

When both `agent_id` and `agent_source_tenant_id` (the source space ID of a shared agent) are passed, the skills that shared agent can @-mention are listed: the sandbox configuration is taken from the agent itself and `sandbox_config_id` in the query is ignored; when the agent's skill scope is `selected` only the specified skills are returned, and when disabled an empty list is returned with `skills_available:false`. Returns 403 when the caller is not allowed to use the agent.

## Session interactive terminal {#session-terminal}

The terminal in the conversation sidebar connects to this session's remote sandbox over WebSocket; only Cube/E2B are supported. The routes are registered by `internal/router/routes_chat.go` and are not yet in Swagger; for the desktop endpoints see the [Session API](02-api-chat.md#sandbox-desktop), and for proxy requirements see [Sandbox Deployment](../06-development/04-sandbox-deployment.md#terminal-and-desktop).

### POST /api/v1/sessions/:session_id/sandbox/terminal-ticket

Issues a handshake ticket valid for two minutes. Requires the session owner's login Bearer access token; API Keys cannot call it.

```bash
curl -X POST "$BASE/api/v1/sessions/$SESSION_ID/sandbox/terminal-ticket" \
  -H "Authorization: Bearer $TOKEN"
```

Response: 200 `{"success":true,"data":{"ticket":"<ticket>","expires_in":120}}`.

### GET /api/v1/sessions/:id/sandbox/terminal

WebSocket handshake; it does not go through the regular authentication middleware:

| Query parameter | Required | Description |
| --- | --- | --- |
| `ticket` | Yes | The ticket from the previous step, bound to the user, space, session and the access token at issue time |
| `provision` | No | `1` allows creating or waking a sandbox (may incur charges); when omitted, only a running sandbox is connected |
| `agent_id` / `agent_source_tenant_id` | No | Used with `provision=1` to specify which agent's (including shared agents') sandbox configuration to create from |
| `cols` / `rows` | No | Initial terminal size |
| `pty_id` | No | Reattach to the shell process returned by a previous `ready` frame |

After connecting, binary frames carry terminal input and output in both directions; text frames are JSON control messages: the server sends `ready` (with `pty_id`, `backend`), `exited` (with `exit_code`) and `error`, and the client can send `{"type":"resize","cols":..,"rows":..}`. Error codes include `SANDBOX_NOT_BOUND`, `SANDBOX_PAUSED`, `TERMINAL_UNSUPPORTED`, `IDLE_DISCONNECTED`, `AUTH_REVOKED` and `INTERNAL`. Each session can have at most 5 terminals at once; beyond that, 429 is returned.

## Lite local project directory

Only available in the Lite desktop edition; other deployments return 404. It opens the system folder picker on the computer running Lite, and the folder the user selects is added to the approved list; when creating a session afterwards, pass the returned `dir` as `project_dir` to let that session use the folder in the [local sandbox](../03-features/22-skills-sandbox.md#lite-host). For the `project_dir` field, see the [Session API](02-api-chat.md).

| Method | Path | Request / Response |
| --- | --- | --- |
| POST | `/system/host-project-dir` | No request body; 200 `{"code":0,"msg":"success","data":{"dir":"/Users/me/project"}}`, and `dir` is empty when the selection is cancelled. Only accepts a login JWT, Viewer+; API Keys are rejected |

## Personal environment variables {#personal-env-vars}

The caller's identity is determined by the current authentication context; user_id is not accepted. The personal endpoints return variable names, sources and update times, never plaintext values.

| Method | Path | Request |
| --- | --- | --- |
| GET | `/me/env-vars` | Returns configuration variables and skill variables grouped by sandbox_config_id |
| PUT | `/me/env-vars/skill` | `{skill_id,name,value}`; the name must be one the skill has declared |
| DELETE | `/me/env-vars/skill` | `{skill_id,name}` |
| PUT | `/me/env-vars/sandbox` | `{sandbox_config_id,name,value}` |
| DELETE | `/me/env-vars/sandbox` | `{sandbox_config_id,name}` |

Reads return `{success,data:[{sandbox_config_id,sandbox_config_name,description,vars,skills}]}`, where source indicates user/workspace/unset and similar states. A successful set/delete returns success; deleting an unset item returns 404. Personal sandbox variables reject the WEKNORA_ prefix and reserved names such as PATH; skill-declared variables can use the WEKNORA_* credential names they need.

```bash
curl -X PUT "$BASE/api/v1/me/env-vars/skill" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"skill_id":"skill-1","name":"API_TOKEN","value":"<my-token>"}'

curl -X DELETE "$BASE/api/v1/me/env-vars/skill" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"skill_id":"skill-1","name":"API_TOKEN"}'
```

The override order is personal skill value > personal sandbox value > space skill value. Writing a personal value does not change the admin configuration. Implementation reference: `internal/router/routes_infra.go`, `routes_agent.go`, `routes_auth_tenant.go`, and `internal/handler/sandbox_config.go`, `sandbox_skill.go`, `skill_catalog.go`, `me_env_var.go`, `host_project_picker.go`.
