# Sandbox Deployment and Troubleshooting

For the skill usage flow, see [Skill Catalog & Sandbox](../03-features/22-skills-sandbox.md); for field definitions, see the [Sandbox & Skills API](../04-api/02-api-sandbox-skills.md). This page adds the operational requirements for deployment, templates, desktops and backend integration.

## Backends and templates

Named space configurations can choose Docker, CubeSandbox or E2B; this page covers deploying these three backends. The Lite desktop client additionally has a `host` backend that relies on operating-system isolation. It is not a server-side named configuration; for its build requirements see [Lite local sandbox](#lite-local-sandbox) below. The old `local` host-process backend has been removed. Docker uses the Engine API directly; E2B uses the control-plane REST API and the envd data plane; Cube keeps a dedicated adapter for templates and network policy.

The standard image is defined by `docker/Dockerfile.sandbox` and includes Python 3.12, Node.js 20, Bash, jq and `/workspace`. Command and file operations use root inside the sandbox by default; the `user` account with UID 1000 is kept for explicit per-account execution. Cross-session isolation is provided by the container or remote sandbox; the working directory convention must not be interpreted as a file permission restriction on root.

| Build target | Image tag suffix | Purpose |
| --- | --- | --- |
| `sandbox` | none | Docker session image; base image for the E2B CLI template |
| `cube` | `-cube` | Cube CLI template, includes envd |
| `desktop` | `-desktop` | Base image for the E2B graphical desktop template |
| `desktop-cube` | `-desktop-cube` | Cube graphical desktop template, includes envd |

The image name is `wechatopenai/weknora-sandbox:<version><suffix>`. Template IDs belong to a specific cluster or account; do not copy IDs from other deployments directly. Production templates should correspond to a verified application version; for build targets and release architectures, the repository Dockerfile and release workflow are authoritative.

## Docker deployment

The Docker backend is disabled by default. A system admin enables it under "System settings → Network security"; before the setting is persisted, `WEKNORA_SANDBOX_DOCKER_ENABLED=true` can be used as a fallback. When disabled, existing configurations can still be viewed and deleted, but no new session containers are created.

One configuration connects to one daemon, and one session uses one long-running container. There is no cross-host scheduling; if different app replicas connect to their own independent local daemons, you cannot assume they share containers and skill snapshots.

| Setting | Default or requirement |
| --- | --- |
| daemon address | When empty, `DOCKER_HOST` or the current Docker context is detected; remote uses `tcp://host:2376` |
| TLS certificate directory | Required for remote TCP; a directory readable by the app containing `ca.pem`, `cert.pem` and `key.pem` |
| CPU / memory / PIDs | Default 2 cores / 2 GiB / 512 processes |
| Idle TTL | Default 1800 seconds |
| Network mode | Only `bridge` or `none`; `host`, `container:` and custom network names are not accepted |
| OCI runtime | Any runtime installed on the daemon can be selected, such as `runsc` |

When the app runs in a container, mount the actual Docker socket or connect to a remote daemon instead. The socket grants control over the host's Docker and should only be given to a trusted app. The entrypoint script configures appuser's groups from the socket GID before dropping privileges; do not open up the socket with `chmod 666`. If the socket is `root:root` and only writable by its owner, first configure suitable non-root group permissions on the host.

Prefer the standard image. Custom images must meet the adapter's requirements for root, Bash, GNU `find -printf`, coreutils `timeout` and a writable workspace; session file checkpoints and rollback also depend on Git. Cancelling the HTTP request alone does not terminate the container process; the adapter enforces command timeouts through `timeout` inside the container.

Idle sweeping is triggered by Create/Connect and runs in the background, judging by the TTL recorded when the container was created; it is not a timer built into the daemon. There is currently no hard lifetime limit, and a script that can execute commands can keep refreshing the activity marker, so deployers need to monitor long-lived containers separately.

### Skill snapshots and disk

Skill installation uses `docker commit` to produce local `weknora-skill/` images, and later sessions start from the installation snapshot. This saves the file system, not memory state; tenant-shared volumes and Docker graphical desktops are not currently supported.

Incremental snapshots inherit the old image layers, so deleting an old tag does not necessarily free disk space; uninstalling a skill adds a deletion marker layer, and the original files may remain in the parent layers. The current flow does not automatically flatten images or distribute them across daemons. Monitor layer count and disk usage, and confirm session and snapshot references before cleaning up; when you need a new base image, create a new configuration and reinstall the skills.

## CubeSandbox / E2B integration

1. First prepare a working control plane, data-plane gateway and sandbox domain. Fill in the API, Proxy, domain and credentials in the space settings; private-network/loopback endpoints require explicitly allowing private addresses, and cloud metadata addresses are still not allowed.
2. Click "Connect and continue" to verify the control plane and load the template catalog. A successful connection only proves the control plane is available.
3. If the standard template is missing, create it explicitly; if you need a desktop, create the desktop template separately. Wait until the status is `READY` before selecting it. Cube should use the `-cube` variant with envd; regular Docker images do not provide `:49983/health`.
4. Run "Full verification", which actually creates a sandbox, runs probes and destroys it, confirming that the data plane and script environment work too.
5. After saving, select the configuration in an agent and verify attachment reading, command state persistence and artifact download.

`proxy_url` is for a self-hosted data-plane gateway: it keeps the sandbox Host routing when connecting to the gateway, which suits environments without wildcard DNS. When the control plane is reachable but execution fails, focus on the Proxy, sandbox domain, inbound credentials and reachability from the app to the gateway.

Cube guest DNS is part of the template configuration. After changing DNS/images, rebuild the template for the changes to reach new environments. A configuration that already has skills installed cannot change or rebuild its base image, nor switch between CLI/desktop; create a new configuration and install the skills instead, to avoid a mismatch between snapshots and the base image.

Multiple replicas must configure a shared Redis to store session→sandbox bindings and desktop state. Only single-instance development can use in-memory storage. Existing sandboxes do not automatically apply backend configuration changes; for the `next_turn` / `new_session` skill update strategies, see [Skill Catalog & Sandbox](../03-features/22-skills-sandbox.md).

| Symptom | What to check |
| --- | --- |
| Connection verification fails | Whether the Dashboard address was entered as the API by mistake; whether credentials, TLS and the private-network switch are correct |
| Connection succeeds but execution fails | Proxy, sandbox domain, gateway routing and inbound token |
| Template build fails | Check the build error returned by the cluster; whether the image can be pulled, the architecture matches and the Cube image includes envd |
| Sandbox has no internet access | Template network policy, guest DNS, cluster egress proxy; allowing a private control plane does not mean scripts can reach the internet |
| Dependency installation fails | Whether package sources are allowed when egress is denied by default; skill installation and sessions use the same network policy |
| State lost after reconnecting | Whether Redis is shared, whether the TTL expired, whether a skill update triggered a rebuild |

## Interactive terminal and graphical desktop {#terminal-and-desktop}

Both the terminal and the desktop in the conversation sidebar connect to the session sandbox over WebSocket; only Cube/E2B support them, and the Docker backend does not. Browsers cannot send authentication headers in a WebSocket handshake, so both first exchange a signed-in POST for a short-lived ticket valid for two minutes, then put the ticket in the handshake query:

| Capability | Get ticket | WebSocket |
| --- | --- | --- |
| Terminal | `POST /api/v1/sessions/:session_id/sandbox/terminal-ticket` | `GET /api/v1/sessions/:id/sandbox/terminal?ticket=...` |
| Desktop | `POST /api/v1/sessions/:session_id/sandbox/desktop-ticket` | `GET /api/v1/sessions/:id/sandbox/desktop?ticket=...` |

The ingress proxy must forward the WebSocket Upgrade for these two paths, relax the read timeout, and avoid recording the ticket query in access logs. The standard frontend Nginx already configures a log format without the query for `^/api/v1/sessions/[^/]+/sandbox/(terminal|desktop)$`; custom Ingresses must handle this themselves. Once a terminal is connected, the server re-checks the login state, space membership and session ownership about once a minute; logging out or being removed from the space disconnects the terminal. For endpoint parameters, see the [Terminal API](../04-api/02-api-sandbox-skills.md#session-terminal) and the [Desktop API](../04-api/02-api-chat.md#sandbox-desktop).

The terminal and desktop disconnect after being idle for the configured `terminal_idle_disconnect_sec` (default 900 seconds, minimum 60 seconds, maximum 24 hours), after which the sandbox pauses according to the provider TTL. The terminal counts keyboard input and PTY output as activity; the desktop counts keyboard and mouse input.

### Graphical desktop

Only Cube/E2B desktop templates support the conversation sidebar desktop. The backend starts the desktop process when it is first opened; the configuration must select a desktop template and set `desktop_enabled`. Once skills are installed, the base image cannot be switched in place.

```text
Browser noVNC → WeKnora ticket relay → provider gateway → websockify :6080 → local x11vnc :5900
```

The browser never holds the sandbox API Key, inbound token or websockify password. Desktop tickets can be consumed only once; for the full definition, see the [Session API](../04-api/02-api-chat.md#sandbox-desktop). The `exposedPorts` of the Cube desktop template only exposes envd's 49983; **do not add 6080 to the host NAT**; the desktop must go through the gateway and the WeKnora relay.

Each session allows only one desktop relay at a time, and multi-replica slots are coordinated by Redis. After a sandbox rebuild, `SANDBOX_REBUILT` signals the disconnect; do not treat the new desktop as the old instance with its original temporary files. Idle detection uses RFB keyboard and mouse activity; screenshot requests do not count as user activity.

## Lite local sandbox

The `host` backend is only compiled into the Lite desktop program with the `desktop` build tag (`build:tags` in `cmd/desktop/wails.json`); the server and the single-binary Lite do not include it. It runs commands in sessions with no sandbox configuration selected; for usage, see [Skill Catalog & Sandbox](../03-features/22-skills-sandbox.md#lite-host).

| Platform | Status |
| --- | --- |
| macOS | Uses the system `sandbox-exec` (Seatbelt) to run each command; not enabled if detected as unavailable at startup |
| Windows | Not yet implemented; reported as unavailable and never falls back to unisolated execution |
| Linux | Not supported |

Every command is a new local process with no session-level instance, so no session sandbox binding is written and no workspace checkpoints are made. Project directories the user approves through the system folder picker are stored in `project_dirs` in `desktop-prefs.json`; a session can only bind a directory in that list itself, not its subdirectories or manually entered paths. `approval_mode` currently only supports `auto` (free read/write within the workspace, network access forbidden), and writing any other value is rejected. For the preferences file location, see [Desktop Client](../05-clients/05-desktop.md#_5-preferences-storage-cmd-desktop-prefs-go).

## Development verification

Unit tests do not need a real daemon:

```bash
go test ./internal/sandbox -run 'TestDocker' -count=1
```

Real Docker verification requires building the standard image first and creates test containers:

```bash
docker build -f docker/Dockerfile.sandbox --target sandbox -t wechatopenai/weknora-sandbox:dev .
DOCKER_INTEGRATION_IMAGE=wechatopenai/weknora-sandbox:dev \
go test -tags=docker_integration ./internal/sandbox \
  -run '^TestDocker.*Integration' -count=1 -v -timeout=15m
```

Conformance tests for remote integrations live in `internal/sandbox/cube_integration_test.go` and `e2b_compatible_integration_test.go`; configure the test cluster credentials as described at the top of each file before running them. Verification should cover session state persistence, shell reuse, attachment staging, artifact collection and timeouts, not just Health. After the tests finish, confirm the test instances have been cleaned up.
