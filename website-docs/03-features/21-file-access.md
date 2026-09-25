# External Access to Images and Files

Images and files are accessed differently in different channels. The web console and the embed widget can carry credentials to access the proxy; IM platforms need time-limited links that can be loaded publicly; API clients can choose internal references or external links depending on their use.

When integrating, confirm the reference form, the caller's permissions, and address reachability together. If an image fails to display, work through the troubleshooting table at the end of this page.

## File references and access methods {#_1-the-four-forms}

Images and attachments in the knowledge base are stored in file storage. The body text uses a stable internal reference, which is converted into an access address per channel at response or render time:

| Form | Example address | Access condition | Validity period |
| --- | --- | --- | --- |
| **Internal reference** | `resource://<handle>` | A stable handle resolved on the server side; cannot be accessed directly as a URL | — |
| **Authenticated proxy** | `/files`, `/api/v1/knowledge-bases/:id/files`, `/api/v1/embed/:channel_id/files`, message-level `/api/v1/sessions/:id/messages/:message_id/files` | Clients with the corresponding credentials (login session / KB access / Embed token) | Depends on the credential |
| **Capability short link** | `/r/<token>` | Anyone who gets the link (**anonymously readable**) | Grant issued by WeKnora, 2 hours |
| **Storage pre-signed link** | http(s) link given directly by the storage backend | Anyone who gets the link (**anonymously readable**) | Determined by storage; MinIO defaults to 24 hours |

Capability short links and storage pre-signed links let whoever holds them read the file anonymously within the validity period. When sharing or logging these links, treat them according to the access scope of the file itself. Revoking an organization share or leaving an organization does not invalidate links that have already been issued; they remain usable until they expire (2 hours for short links, up to 24 hours for storage pre-signed links).

::: tip Default access method
By default the system returns internal references, which clients with credentials read through the authenticated proxy. External links require a publicly reachable storage endpoint, or WeKnora issuing access links once `APP_EXTERNAL_URL` is configured. The default MinIO address `minio:9000` is only reachable inside the container network.
:::

## Accessing files per channel {#_2-how-each-channel-obtains-it}

```mermaid
flowchart TD
    R["resource:// reference in body text"] --> Q{"Which channel"}
    Q -->|"Web console"| W["Frontend rewrites to /files proxy<br/>with Bearer + X-Tenant-ID"]
    Q -->|"Embed widget"| E["/api/v1/embed/:channel_id/files<br/>with Embed token"]
    Q -->|"IM bot"| I{"Storage backend publicly reachable?"}
    I -->|"Yes"| IP["Fall back to storage pre-signed URL"]
    I -->|"No"| IE{"APP_EXTERNAL_URL configured?"}
    IE -->|"Yes"| IR["Rewrite to APP_EXTERNAL_URL/r/token<br/>needs nginx proxy for /r/"]
    IE -->|"No"| IF["Keep resource:// as-is<br/>IM cannot load the image, warning logged"]
    Q -->|"REST API"| A{"resource_urls=public?"}
    A -->|"No (default)"| AH["Return resource://<br/>client calls /files proxy again"]
    A -->|"Yes"| AP["Return pre-signed or /r/token external link"]
```

### Web console

The web frontend converts `resource://` and `provider://` references into authenticated proxy addresses. Regular resources go through `/files`, with requests carrying a Bearer token and `X-Tenant-ID`; knowledge bases shared across spaces go through `/api/v1/knowledge-bases/:id/files`, which reads files from the source space based on knowledge base access rights. No additional configuration is needed.

Images in answers from a shared Agent or an organization-shared knowledge base go through `/api/v1/sessions/:id/messages/:message_id/files?file_path=...` first. The server verifies that the caller can read the message and that the requested resource is actually referenced by the message, and re-checks the authorization for the knowledge base or shared Agent that owns the resource. Once a share is revoked, historical messages no longer grant access to the resource. Regular knowledge base browsing can still use the KB-level proxy; each uses its own context.

### IM bots {#im-bots-the-most-problem-prone-one}

IM platforms can't carry WeKnora's credentials, so they need a publicly accessible HTTP(S) URL. Before a message is sent, the system generates an external link based on the storage and deployment configuration:

1. **The storage backend itself is publicly reachable** — object storage uses a public endpoint, or `MINIO_ENDPOINT` is set to a public host. In this case it falls back to a storage pre-signed URL, requiring no additional configuration;
2. **`APP_EXTERNAL_URL` is configured** — the reference is rewritten to `<APP_EXTERNAL_URL>/r/<token>`, and the request is proxied back to the app via nginx's `location ^~ /r/`. The official frontend image already includes this location block; self-built reverse proxies must add it, otherwise requests fall into the SPA fallback and return a blank page.

The default MinIO internal deployment and `local` storage need `APP_EXTERNAL_URL` configured. When the external link conditions aren't met, the system keeps the original reference and logs a warning, and the IM platform cannot display the image directly. When an IM channel is enabled but `APP_EXTERNAL_URL` is empty, a warning is also logged at startup.

### Embed widget

The embed widget uses the channel file proxy `/api/v1/embed/:channel_id/files`. The Embed token determines the channel's space, and the server checks that the resource path belongs to it. Embed channels always return internal references; neither `RESOURCE_URL_MODE=public` nor `?resource_urls=public` changes this behavior, so that reads always go through channel authentication.

### REST API and SDK

The API returns `resource://` by default, and clients fetch the file through the authenticated proxy. When you need external links for direct rendering, you can set:

- Single request: `?resource_urls=public`
- Whole deployment: `RESOURCE_URL_MODE=public`

The per-request parameter takes precedence over the environment variable, so even after setting the deployment default to `public`, you can still use `?resource_urls=handle` to fall back individually. For the interfaces that support this parameter, its scope of coverage, and its security boundaries, see the "File Reference Forms" section of the [API Overview](../04-api/01-api-overview.md).

An API Key scoped to specific knowledge bases gets a 403 when using `public`; such keys also cannot access the general `/files` proxy. When the external link conditions aren't met, the response keeps `resource://`, and clients with the corresponding permissions can use the authenticated proxy instead.

## Troubleshooting access problems {#_3-troubleshooting-by-symptom}

| Symptom | Possible cause | What to do |
| --- | --- | --- |
| Image doesn't display in IM | `APP_EXTERNAL_URL` not configured and storage isn't publicly reachable | Configure `APP_EXTERNAL_URL`, confirm nginx proxies `/r/`; check the app logs for a `rewriteStorageURLs no-op` WARN |
| The IM image link opens but returns a blank page | nginx is missing `location ^~ /r/`, request falls into the SPA fallback | Add that location block (already included in the official frontend image), see [Web Frontend](../05-clients/01-frontend.md) |
| `APP_EXTERNAL_URL` is set to an internal address or `localhost` | The IM platform is on the public side and can't reach it | Change it to an address reachable by the IM platform; for local development use ngrok / cloudflared / frp |
| The image URL returned by the API is `resource://` | This is the internal reference by default | Add `?resource_urls=public`, or call the `/files` proxy |
| Added `resource_urls=public` but still returns `resource://` | Deployment lacks external link capability (e.g., `local` storage without `APP_EXTERNAL_URL`) | Satisfy the external link conditions, or use the `/files` proxy instead |
| Added `resource_urls=public` and got a 403 | Using a knowledge-base-scoped API Key | Switch to `handle` mode, or use an authorized full-access Key instead |
| Image doesn't display in the embed widget, but works fine on the web | The widget goes through the channel proxy, different from the main site's credentials | Confirm the widget page carries a valid Embed token; `resource_urls=public` has no effect on embed channels |
| Images in a shared answer return 403/404 | Missing message context, resource not bound, or share revoked | Use the message-level proxy and check the current share permissions; don't construct `/files` addresses for the owning tenant |
| Links already sent stop working after an upgrade or key change | After the signing key (`SYSTEM_SIGNING_KEY`, or the `SYSTEM_AES_KEY` fallback) changes, old signatures can no longer be verified | Fetch the links again; in multi-replica deployments confirm that all instances use the same key |
| The external link stops working after a while | External links are time-limited (grant 2 hours / MinIO pre-signed 24 hours) | Don't cache the external link itself; fetch it again when needed. The same file will reuse the same link within its validity period |
| Image 404 on the web frontend, logs show tenant mismatch | The image in a cross-tenant shared library exists under the owning tenant | This scenario should go through `/api/v1/knowledge-bases/:id/files`; confirm the frontend is getting the KB-scoped proxy address |

## Configuration reference {#_4-related-configuration}

| Configuration | Purpose |
| --- | --- |
| `APP_EXTERNAL_URL` | The externally reachable address for IM channel image external links; the prerequisite for rewriting `resource://` into `<APP_EXTERNAL_URL>/r/<token>` |
| `RESOURCE_URL_MODE` | The default form of file references in API responses (`handle` / `public`) |
| `MINIO_ENDPOINT` and other storage endpoints | When set to a public address, external links can be provided by storage pre-signing, without depending on `APP_EXTERNAL_URL` |
| `SYSTEM_SIGNING_KEY` (falls back to `SYSTEM_AES_KEY` when unset) | Signing key. Pre-signed links from `/api/v1/files/presigned` depend on it; it is also used to reuse `/r/<token>` grants within their validity period, keep direct link URLs stable, and reduce write pressure on read endpoints. When it is unset, shorter than 16 characters, or left at the example value, pre-signed links cannot be issued; after it changes, links already issued stop working |

## Related documentation {#_5-related-sections}

- [IM Integration](12-im-integration.md): rewrite logic and startup warnings
- [Web Embed](13-embed-channel.md): channel authentication and anonymous sessions
- [API Overview](../04-api/01-api-overview.md): the full semantics of `resource_urls`
- [Configuration Reference](../01-getting-started/04-configuration.md): the environment variables above
- [Web Frontend](../05-clients/01-frontend.md): nginx's `/files` and `/r/` proxies

## Implementation reference

- `frontend/src/utils/protectedFileAccess.ts`: web file references and proxy paths.
- Before IM messages are sent, `rewriteStorageURLs()` converts references into external links.
