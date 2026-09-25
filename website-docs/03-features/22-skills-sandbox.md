# Skill Catalog and Sandbox

A skill is made up of a `SKILL.md` description, scripts, templates and reference material; the sandbox provides the environment in which the scripts run. Once a skill is added to the space catalog and installed into a sandbox, select that sandbox and skill in an agent to use it in Smart Reasoning conversations.

## Configure and use a skill {#from-configuration-to-first-run}

1. A space Admin/Owner creates a configuration under **Settings → Sandbox Config**, chooses Docker, CubeSandbox or E2B, and fills in the connection details.
2. Follow the wizard to connect the cluster and pick a template; run **Full verification** when you need to check the whole chain. Full verification actually creates a sandbox, runs probes and cleans up.
3. In the sidebar, open **Toolbox → Skill Management**, add a ZIP or a source link, then choose the sandboxes to install into (old **Settings → Skill Management** links redirect here automatically). A successful catalog entry only means the package has been saved; the skill can run once its installation status is ready.
4. Open the **Skills** section of the agent editor, choose a sandbox, and set the skill scope to all, selected skills or disabled. Skill execution is used in Smart Reasoning mode.
5. Ask a question in a conversation, or use `@skill` to prompt the agent to prefer that skill. Upload attachments when files are needed; generated deliverables can be previewed and downloaded in the **Files** tab of the **Sandbox** panel on the right side of the conversation.

`@skill` does not revoke the agent's access to the other skills it is authorized for. When no sandbox is selected (the Lite desktop app on macOS uses the [local sandbox](#lite-host) instead), the space has scripts turned off, or a required backend capability is unavailable, the execution tools are not registered; prompts cannot get around these conditions.

<Screenshot
  src="/screenshots/skill-catalog.png"
  caption="Space skill catalog: skills and their installation status in each sandbox" />

## Manage installations and updates {#catalog-installation-and-updates}

The space catalog stores one copy of each skill package; every sandbox configuration has its own installation records and an image snapshot containing its skills. The same skill can be installed into several sandboxes, with installation progress and failure reasons recorded separately.

| Action | Result |
| --- | --- |
| Add to catalog | Saves the package, name, version and description without installing |
| Install | Builds the runtime environment in the selected sandbox, installs dependencies and checks that the skill loads, then publishes a new snapshot on success |
| Retry | Reinstalls using the package already saved for that sandbox, optionally with installation notes; skipped when the same package is already ready |
| Upgrade | After the catalog registers a new version, installations of the old version are marked upgradable; upgrading installs the catalog's new version into the selected sandbox |
| Stop installation | Aborts an installation in progress; the status becomes failed, after which you can retry or uninstall |
| Disable | Keeps the installation record but stops agents from selecting the skill |
| Uninstall from sandbox | Updates that sandbox's skill image and keeps the package in the catalog so other sandboxes can still install it |
| Delete catalog entry | Requires that no sandbox installation references it anymore; does not implicitly uninstall from all sandboxes |

The installation page shows a percentage, stages and logs, and the detailed run can be viewed in the installation record. Closing the progress drawer or disconnecting the progress stream does not stop the installation. When there is no live progress, refresh the skill status; an expired detailed event log does not mean the skill package was lost.

Installation is performed by a built-in installer agent: it reads `SKILL.md` to install dependencies and checks the runtime prerequisites the skill needs, such as external command-line tools. When a required command is missing or a prerequisite is unresolved (for example, a service that has to be deployed separately), the installation is judged failed with a reason; a skill is not marked ready just because it has no dependency list. While an installation is running, admins can add installation notes in **Install log**, such as CLIs that need installing, installation docs or environment constraints; after the installation ends, they can also reinstall with notes.

When an upgrade or retry fails, the sandbox keeps serving the last ready version and agents can still use it; the new version replaces it only after it installs successfully.

`skill_rollout` controls image updates: the default `next_turn` rebuilds the sandbox on the next turn of existing sessions; `new_session` only makes sessions that create a sandbox afterwards use the new image. Rebuilding a sandbox loses the old instance's temporary runtime state, so files that need to be delivered should be written to `/workspace/output` and collected by the system.

### Supported sources

| Input | Description |
| --- | --- |
| ZIP file | Export the skill directory and upload it |
| `@owner/slug`, `@owner/slug@1.2.0` | ClawHub with a specific author/version |
| `slug`, `slug@1.2.0` | ClawHub slug |
| ClawHub, SkillHub or self-hosted SkillHub page | Resolved through the corresponding source |
| GitHub/GitLab repository or directory URL | Fetches the corresponding skill package |
| `https://skills.sh/owner/repo/slug`, ClawHub skills.sh pages | Resolved by the install resolver to a specific version and directory in the repository |
| Direct ZIP or SKILL.md URL | Downloads the package or entry file |

Sources must be readable anonymously; downloads do not carry the user's private repository credentials. For private skills, export a ZIP first. `owner/slug` is ambiguous; use `@owner/slug` or the full URL instead.

The skill package limit is separate from regular documents: `MAX_SKILL_BUNDLE_SIZE_MB` defaults to 256 MiB, is at least `MAX_FILE_SIZE_MB` when unset, and is capped at 512 MiB. GitHub downloads count the whole repository archive, not just the skill subdirectory. After changing it, restart app and frontend so the application and Nginx limits match.

## Choose a sandbox backend

| Backend | What to fill in | How it runs |
| --- | --- | --- |
| Docker | Image; optionally the daemon address, TLS certificate directory, CPU/memory/PID limits, network mode, runtime and idle TTL | One long-running container per session |
| CubeSandbox | Control plane address, data plane proxy, sandbox domain, template; API key per cluster configuration | Session-level remote sandbox |
| E2B | API key, template; when self-hosted, also the API address, sandbox domain and data plane proxy | E2B Cloud or an E2B-compatible control plane |
| host | Nothing to fill in. Lite desktop app only, currently macOS only | When the agent has no sandbox configuration selected, runs inside the local project directory, with out-of-bounds access blocked by the operating system |

The `local` host-process backend has been removed: it ran directly on the host with no isolation at all. Lite's `host` is not one of the named space configurations above, nor a replacement for `local`. For the full fields of the current configuration, see the [Sandbox & Skills API](../04-api/02-api-sandbox-skills.md).

The Docker backend is off by default. A system admin enables it under **System Settings → Network security**, or `WEKNORA_SANDBOX_DOCKER_ENABLED=true` serves as the fallback when nothing is stored in the database. A local connection also requires mounting the actual Docker socket into app, which grants app control over the host's Docker. A remote TCP daemon needs a TLS certificate directory containing `ca.pem`, `cert.pem` and `key.pem`. The Docker network accepts only `bridge` or `none`, and an installed OCI runtime such as `runsc` can be selected.

For self-hosted E2B/Cube, `proxy_url` points to the data plane gateway: WeKnora connects to the gateway but keeps the sandbox Host, for clusters without wildcard DNS. `allow_private_endpoints` allows connecting to private/loopback cluster addresses, but still does not allow link-local/cloud metadata addresses; it is a separate setting from whether scripts inside the sandbox can reach the network.

Scripts run as the `root` account inside the sandbox by default; the template's `user` account must be selected explicitly. Execution isolation is provided by the container or remote sandbox; `/workspace` only sets the working directory convention and does not restrict file access for root commands.

### Network policy

`config.network` for Cube/E2B applies to conversation sandboxes, skill installation and full verification alike:

- Outbound traffic is allowed by default; `deny_egress_by_default=true` switches to deny by default, after which `allow_out` allows IPs, CIDRs or domains.
- `deny_out` accepts IPv4/CIDR; domain allow rules must be combined with deny by default.
- Cube's `cube_rules` can set rules by Host/SNI, method and path, reorder them, and configure auditing and HTTPS header injection; E2B's `e2b_host_rules` configure request header injection for allowed domains.
- Injected credentials are stored encrypted and redacted in responses. Inbound access always requires credentials; the legacy `allow_public_inbound` field does not open anonymous inbound access.
- Docker controls egress with `docker.network_mode` and cannot reuse the fine-grained Cube/E2B rules. Once egress is denied by default, the origins needed to install dependencies must also be allowed explicitly.

Existing instances do not pick up a policy change automatically; verify it after creating or rebuilding a sandbox. Before changing a backend's identity or deleting a configuration, the system checks for running/paused instances and associated agents; if any are in use, the operation is refused and the settings page shows what is using it.

### Lite local sandbox (host) {#lite-host}

Since v0.8.2, the [Lite desktop app](../05-clients/05-desktop.md) on macOS isolates the agent's commands and file tools with the system Seatbelt. When the agent has a Docker, E2B or Cube configuration selected, it still uses the remote sandbox; the local sandbox is used only when no sandbox configuration is selected. Windows and Linux do not offer this capability yet, so agents without a selected configuration cannot run commands there.

- **Working directory**: On the new conversation page, click **Select project** and pick the project folder in the system folder picker; the conversation title bar shows the project name. Without a selection it shows **Temporary workspace** and automatically creates a session directory under `~/Documents/WeKnoraLite/<date>/session-*`. You cannot select the home directory itself, a directory that contains the home directory, or a directory under `~/Library`.
- **Paths**: The workspace is a real local path; there is no `/workspace`, `/workspace/input` or `/workspace/output`. The agent edits files directly in the project, and these files do not appear in the conversation's artifact list; deleting the conversation does not delete the local directory.
- **Restrictions**: Commands cannot access the network by default and can only write to the project directory; the project's `.git` is read-only. Home directory contents other than common development toolchains (such as `.nvm`, `.pyenv`, `.cargo`) are unreadable, and credential locations such as `.ssh`, `.aws` and the keychain are always unreadable.
- **Rewind**: Rewinding a conversation does not restore local project files; manage them with Git yourself if needed.

## Environment variables and credentials

Personal variables live under **Settings → Sandbox secrets**; space-level skill variables are configured on the skill card in **Toolbox → Skill Management**. Lists only show variable declarations, whether they are set, and their source; secret values are never echoed back.

| Level | Purpose |
| --- | --- |
| Space sandbox `config.env_vars` | Injected into sandboxes created by that configuration for use by its scripts |
| Space skill variables | Default values an admin fills in for variables a skill has declared |
| Personal sandbox variables | Used when you run commands with that sandbox configuration |
| Personal skill variables | Used when that skill runs for you |

For skill variable resolution, **personal skill value > personal sandbox value > space skill value**; only names that are unset fall back. The space sandbox environment is part of the runtime environment, so do not put secrets there that scripts should not read. Deleting a personal override makes the lower-level value apply again; disabling a skill does not delete personal credentials.

Personal skill variables can only use names the skill has declared, which may include `WEKNORA_*` credentials the skill needs. Personal sandbox variables do not accept reserved names such as `WEKNORA_*` and `PATH`. Variables the system detects from executed commands only fill in unset personal values and never override existing personal or space configuration. For fields and examples, see the [Personal Variables API](../04-api/02-api-sandbox-skills.md#personal-env-vars).

## Generate and download files {#files-and-delivery}

`read_file(path="skill://<name>/SKILL.md")` reads the instructions; scripts bundled with a skill run through `shell_exec(skill_name=..., command=...)`. `$WEKNORA_SKILL_DIR` in a command points to the skill's actual installation directory; `skill://` is a read address and cannot be used directly as a shell path.

Attachments are staged in `/workspace/input`, working scripts go in `/workspace`, and downloadable artifacts go in `/workspace/output`. Use `write_sandbox_file` to create or append, `edit_sandbox_file` for partial replacements, and `read_file` to read in pages. For file tool boundaries, output budgets and rebuild behavior, see the [Agent Engine](07-agent.md); for delivery entry points, see [Sessions and Conversation Experience](18-chat-experience.md).

<Screenshot
  src="/screenshots/skill-sandbox-chat.png"
  caption="After the sandbox generates a Word file, preview and download it in the conversation" />

While a command runs, the tool card in the conversation shows the last few lines of output in real time, so you can follow the progress of long commands; this intermediate output is not written into the model context.

### Sandbox panel

In conversations that use a remote sandbox, the **Sandbox** panel on the right has three tabs:

| Tab | Content | Availability |
| --- | --- | --- |
| Files | Downloadable files generated in this turn or all turns, searchable by file name | All remote sandboxes |
| Terminal | An interactive shell (xterm) connected to this session's sandbox; after a page refresh it can reattach to a terminal that is still running | Cube, E2B; not supported on Docker |
| Desktop | Operate the XFCE graphical desktop inside the sandbox from the browser; only one connection per session at a time | Cube and E2B desktop templates, with `desktop_enabled` turned on in the configuration |

Opening the panel does not create or wake a sandbox: while the sandbox is running, the terminal and desktop connect directly; when the sandbox is paused or not yet created, click **Start terminal**, **Connect desktop** or **Create and start**, and the woken or newly created sandbox is billed according to the space configuration. The terminal or desktop disconnects automatically after a long period of inactivity, and the sandbox is then paused according to the provider TTL; the disconnect time is controlled by the sandbox configuration's **Terminal / desktop idle disconnect (s)**, 900 seconds by default, ranging from 60 seconds to 24 hours. After a skill update triggers a sandbox rebuild, unsaved content in the previous terminal and desktop is lost.

<Screenshot
  src="/screenshots/sandbox-panel-terminal.png"
  caption="Sandbox panel: view files and use the terminal and desktop next to the conversation"
  hint="Shows a conversation using a Cube or E2B sandbox, with the Sandbox panel on the right switched to the Terminal tab; the terminal shows a colored user@host prompt and one command's output, and the Files / Terminal / Desktop tabs are visible at the top of the panel." />

## Deployment and troubleshooting

For Docker socket/TLS, template versions, remote gateways, multi-replica Redis, skill snapshot disk usage, terminal and desktop relays, and the build requirements of the Lite local sandbox, see [Sandbox Deployment and Troubleshooting](../06-development/04-sandbox-deployment.md).

## Implementation reference

- `internal/handler/sandbox_config.go`, `sandbox_skill.go`, `skill_catalog.go`, `me_env_var.go`
- `internal/application/service/tenant_skill_install.go`, `tenant_skill_runtime_verify.go`, `tenant_skill_steer.go`, `user_env_resolver.go`
- `internal/types/tenant.go`, `sandbox_network_policy.go`
- `internal/sandbox/remote_client.go`, `docker_remote_client.go`, `gateway_transport.go`, `terminal.go`
- `internal/localsandbox/` (Lite local sandbox: `core/policy_builder.go`, `seatbelt/`, `adapter/`)
