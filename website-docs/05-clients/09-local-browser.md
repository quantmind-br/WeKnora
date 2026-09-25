# Local Browser Integration and Deployment

WeKnora connects the browser on the user's computer to smart reasoning conversations through the BrowserSkill daemon and the [BrowserSkill](https://github.com/Tencent/BrowserSkill) browser extension. The extension supports Chrome and Microsoft Edge (based on Chromium 125 or later); other Chromium browsers are not guaranteed to be compatible. The browser runs on the user's computer and the daemon runs alongside the WeKnora app, so no skill sandbox is needed. Web clipping and sidebar knowledge base Q&A use a separate [knowledge assistant extension](06-chrome-extension.md).

## Installing the extension

The extension must be **version 0.3.1 or later**. Install it in any of these ways:

- Chrome: [Chrome Web Store](https://chromewebstore.google.com/detail/hhcmgoofomhgciiibhipgmgkgnoenaoi)
- Edge: [Edge Add-ons](https://microsoftedge.microsoft.com/addons/detail/browserskill/emacgiaaaiojkkpkddmmdfhmokgmnikg)
- Matching ZIP: when the store is unreachable, download it from "Install manually (alternative)" on the "Browser connection" page. After unzipping, enable "Developer mode" in `chrome://extensions` (`edge://extensions` in Edge) and choose "Load unpacked".

When the connected extension version is below 0.3.1, the "Browser connection" page prompts you to upgrade.

## Pairing and usage

1. Open "Toolbox → Browser connection" in the sidebar. Old "Settings → Browser connection" links redirect here automatically.
2. Copy the one-time pairing link generated on the page, paste it into "Connection settings → Remote connection" in the extension and save.
3. Turn on "Local browser" in the smart reasoning input box, then submit the task. Successful pairing does not mean it is enabled automatically for every turn.
4. The first call creates a separate task window; the conversation preview can locate the page and pause, resume or end the current browser task.

<Screenshot
  src="/screenshots/browser-connection.png"
  caption="Toolbox → Browser connection: pairing link, connection status and extension install entry points"
  hint="Paired and online state: the connection status on the tab, device information, the pairing link area, Chrome/Edge store entries and &quot;Install manually (alternative)&quot;, and the &quot;Show connection status in the sidebar&quot; switch; a green status dot is visible next to Toolbox in the sidebar." />

The connection status is shown on the "Browser connection" tab (connected, offline or not paired). When paired, the small browser icon to the right of "Toolbox" in the sidebar also shows the status with a dot: green for connected, orange for offline. If you don't need it, turn off "Show connection status in the sidebar" on the "Browser connection" page; this setting is stored only in the current browser. The same page also lets you set "Browser search instructions" to specify a preferred search engine and search URL.

Authorization is stored per space and user; multiple conversations in the same space share the device authorization but each has its own task. Currently one device authorization is kept per space + user, and activating a new device replaces the old one.

Pairing links are valid for five minutes. Device tokens are valid for 90 days and rotate automatically after 30 days; generating a new link does not immediately disconnect the old device, and the authorization is replaced only after successful activation. Revoking the device, disabling the user or removing the space member blocks further use.

### Human involvement and task recovery

A task can operate the tabs it created itself; new tabs opened by a click or keypress inside the task window that pass origin verification can also be operated, and are kept when the task ends. Standalone pop-up windows and the user's existing tabs are borrowed through `tab_borrow`, operated after authorization is confirmed according to the browser settings, and returned with `tab_return` when done. When the task ends, pages the task explicitly created are closed and borrowed pages are returned.

Manually dragging a page into the task window does not authorize it; if an unauthorized page is already in the task window, the user must first move it to a regular window and then borrow it. The borrow confirmation prompt needs an HTTP(S) page in a regular window to host it; extension settings pages, new tab pages and standalone pop-ups cannot host it. If the original window disappears after borrowing, returning may create a regular fallback window; returned pages are not closed when the task ends.

Login, CAPTCHA or authorization steps are initiated by `request_help`. Follow the preview prompt to enter the browser, and after completing the step confirm it in the browser help prompt; the agent then observes the page and continues. Simply saying "please log in" in the chat does not create a human takeover prompt. Help waits at most five minutes; a timeout or cancellation keeps the current state and pauses.

Pausing cancels the current browser call but does not revoke pairing; after clicking resume, if the agent's turn has already ended, you also need to send a continue instruction. Tasks after an error or service restart do not automatically replay actions such as clicks or submissions; check the page first after recovery.

Normally successful tasks end automatically; the current state can be kept when the user asks to keep the page, when human help is needed, or on abnormal interruption. When the preview shows "Last preview retained", it means polling has stopped, not that browser control has been released. Ending the browser task does not stop the whole conversation.

## Single-instance deployment

The matching app image includes the daemon, the extension ZIP and the license, with preset file paths; the matching frontend Nginx handles WebSocket. When updating, update the app, the frontend and the users' extension together.

For native deployments, build from the repository root:

```bash
./scripts/build_browserskill.sh
```

This requires Git, Node.js, Python 3, Rust/Cargo and a C compiler; Linux additionally needs CMake. The pinned source version is managed by `scripts/browserskill-release.json`; the upstream source is built directly without extra patches, and the output goes to `artifacts/browserskill/`. Use the daemon for the matching operating system and architecture, and configure it according to the actual installation path:

```dotenv
BROWSERSKILL_BINARY=/opt/weknora/browserskill/bsk
BROWSERSKILL_EXTENSION_PATH=/opt/weknora/browserskill/browser-skill-weknora-0.3.1.zip
BROWSERSKILL_MAX_CONNECTIONS=32
```

By default the pairing WSS address is generated from the origin of the user's current page, keeping the port. Only set an override for deployments behind a separate gateway domain or a special path:

```dotenv
BROWSERSKILL_PUBLIC_URL=wss://weknora.example.com/api/v1/local-browser/extension
```

Localhost WS works locally; remote deployments require a WSS certificate trusted by the browser. Intranets can use a trusted enterprise CA. Explicitly setting `BROWSERSKILL_BINARY=` disables the capability; after changing Docker environment variables, recreate the containers with `docker compose up -d app frontend`.

The matching extension and the daemon must be upgraded together, and the extension must be 0.3.1 or later (Chrome Web Store, Edge Add-ons or the matching ZIP all work). Overwriting the original unpacked directory and reloading keeps the extension ID; if reinstalling changes the ID, you need to pair again. Keep `BrowserSkill-LICENSE` when redistributing.

## Multiple replicas and ingress proxy

All app instances share the database; device authorizations, connection leases and interruption markers are persisted in the database. Each app starts a shared daemon on demand, and other replicas hand tasks to the node holding the extension connection through signed internal RPC.

```dotenv
# Different for each replica; must reach that node directly, not a load balancer address
BROWSERSKILL_INTERNAL_URL=http://10.0.0.12:8080
# Same for all replicas; inject a random value of at least 32 characters from a Secret
BROWSERSKILL_CLUSTER_SECRET=<shared-random-secret>
```

Kubernetes can use Pod IPs to form the direct addresses. Do not replace the shared database with a separate SQLite file per replica. Internal HTTP is only suitable for trusted, isolated networks; use HTTPS with trusted certificates when transport confidentiality is required.

The ingress proxy must:

- Forward the WebSocket Upgrade for `/api/v1/local-browser/extension`, as well as the `/authorize` POST.
- Block access to `/api/v1/local-browser/internal` from the public ingress and only allow app nodes to reach each other. The matching frontend already rejects this path; handle it too when exposing the app directly or using a custom Ingress.
- Avoid logging Authorization, Sec-WebSocket-Protocol and the authorization request body.

Connection leases last 45 seconds and are renewed every ten seconds; recovery after a node failure depends on old lease expiry and the extension's reconnect backoff. After the service recovers, check the task status; operations with unknown results are not replayed automatically.

## Troubleshooting and acceptance

| Symptom | What to check |
| --- | --- |
| No matching extension download or capability unavailable | daemon and extension file paths, executable permission, target architecture and app image version |
| Extension version too low | Upgrade to 0.3.1 or later from the Chrome Web Store, Edge Add-ons or the matching ZIP; overwrite the original unpacked directory and reload |
| Authorization succeeds but WSS fails | Public address/port, trusted certificate, outer proxy WebSocket Upgrade |
| Device connected but tasks don't run | The per-turn switch, whether the task is paused, whether it is waiting for tab borrowing or human help |
| No human help prompt | Whether the execution record called `request_help`; whether the extension disabled human help |
| Fails after switching replicas | Whether the database is shared, whether the node direct address is correct, the shared secret and network policy |
| Preview is stuck | Whether the page is hidden, whether the task kept its last preview, the specific CDP timeout in the extension; the preview is low-frame-rate screenshots, not video |

The default of 32 is a per-node active device protection limit, not a throughput guarantee. Before going live, verify pairing, reconnection, revocation, human help and cross-replica routing on the target cluster, and observe daemon CPU, RSS, screenshot traffic, RPC latency and reconnect time. The real extension test entry point is `internal/browserskill/extension_test.go`, which uses a separate test browser profile directory.
