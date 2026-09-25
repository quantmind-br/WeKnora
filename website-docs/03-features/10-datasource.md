# Data Source Import (Data Source)

Data sources continuously sync content from platforms such as Feishu, Notion, Confluence, and Yuque into a knowledge base. Once a connection is set up, new and modified content can be fetched on schedule; content deleted at the source is handled according to the sync configuration.

Data sources are configured inside a knowledge base. Open the target knowledge base's edit settings, go to the "Data Source" tab, create a new connection and fill in the credentials, then choose the sync scope and interval. The first sync fetches the complete content; later syncs update incrementally according to the connector's capabilities.

<Screenshot
  src="/screenshots/datasource-sync.png"
  caption="Data Sources: connection list and sync status"
  hint="Shows configured data sources (type, target knowledge base, last sync time, status) and the sync log entry point." />

Connectors read external content, the scheduler triggers syncs, and the service layer compares changes and ingests knowledge.

## Setting Up a Connection and Syncing

1. As a space administrator, open the "Data Source" settings of the target knowledge base.
2. Choose a connector, fill in the credentials, and test the connection to confirm that the resources you need can be listed.
3. Choose the sync scope and interval, save, and start the first sync.
4. Check the sync status and logs to confirm that created, updated, skipped, and failed items match expectations.

Updating credentials requires submitting the complete configuration, which the system validates online. Pausing a data source stops subsequent scheduled syncs; resuming registers the schedule again.

## Choosing a Connector

| Connector | Type identifier | Synced objects |
| --- | --- | --- |
| Feishu / Lark Wiki | `feishu` / `lark` | Wiki spaces and nodes |
| Feishu Drive / Lark Drive | `feishu_drive` / `lark_drive` | Specified drive folders; see [Feishu Drive Integration](24-feishu-drive.md) |
| Notion | `notion` | Pages and databases |
| Confluence | `confluence` | Pages in Server / Data Center or Cloud spaces |
| Yuque | `yuque` | Knowledge base documents |
| DingTalk Docs | `dingtalk` | Knowledge bases, folders, and online documents |
| Tencent IMA | `ima` | Files and notes in knowledge bases |
| GitLab | `gitlab` | Directories under a specified branch/tag of a repository |
| RSS / Atom | `rss` | Feed articles |

For the formats, authentication, and deletion detection supported by each connector, see the reference section.

## Checking Changes and Failures

The first sync fetches the content in the selected scope; later syncs update according to the connector's cursor and modification information. Deletion detection is constrained by the connector and the sync configuration; entries that naturally roll off an RSS feed are not treated as deleted source documents. When a sync fails, first check the logs for credential, resource visibility, or parsing errors, then test the connection and retry.

## Connector and API Reference

### Connector Implementation Details

#### Connector Capability Comparison

| | Feishu / Lark | Notion | Yuque | RSS / Atom |
| --- | --- | --- | --- | --- |
| Source directory | `internal/datasource/connector/feishu/wiki/` (shares `feishu/core/`) | `connector/notion/` | `connector/yuque/` | `connector/rss/` |
| Type identifier | `feishu` / `lark` | `notion` | `yuque` | `rss` |
| Auth method | Self-built enterprise app `app_id` + `app_secret` (tenant_access_token) | Internal Integration Token (`api_key`) | Personal/team Token (`api_token`, `X-Auth-Token` header) | No auth, or custom request headers (`auth_headers`) |
| Credential fields | `app_id`, `app_secret`, `base_url` (optional override) | `api_key` (`base_url` goes under Settings) | `api_token`, `base_url` (optional, for private deployments) | `auth_headers` (optional, a credential); `feed_urls` belongs under Settings |
| Resource model | Wiki space → node tree (lazy-loaded, composite ID `spaceID:nodeToken`) | Full page/database tree (returned in one call with parent relationships) | Flat list of knowledge bases (book/repo) | One resource per feed URL (flat) |
| Content format | Export API → `.docx`/`.xlsx` file; drive files downloaded as-is | Block → Markdown; databases converted to Markdown tables; attachments downloaded | Raw `body` Markdown (`.md`) | Readability full-text extraction → HTML→Markdown |
| Incremental mechanism | Compared by content `obj_edit_time` (cursor: `SpaceNodeTimes`) | Compared by page/record `last_edited_time` (cursor: `PageEditTimes`) | Compared by document `content_updated_at` (cursor: `BookDocTimes`) | Dual-layer comparison: feed signal fingerprint + content SHA-256 fingerprint |
| Deletion detection | Supported (present in cursor, absent from current tree → `IsDeleted`; deletion detection is skipped when part of the listing fails) | Supported (distinguishes "deleted at source" from "deselected by user" — the latter is not reported as a deletion) | Supported | Not supported (feeds naturally roll off old entries) |
| Streaming resumable sync | Yes (`StreamingConnector`, checkpoint every 50 nodes or 30 seconds) | No | No | No |
| Rate-limit handling | 429 reads `Retry-After` + exponential backoff (2s/4s/8s, up to 3 retries); 5xx retried | — | 300ms delay between each `GetDocDetail` call (personal token ~100 req/5min) | — |
| Partial failure | A single failed document generates a placeholder item with error metadata, sync continues | A single failed page is logged and skipped | A single failed document generates a placeholder item | A single failed feed → `PartialFetchError`; only fails overall if all feeds fail |

#### Feishu / Lark (`connector/feishu/wiki/`)

For drive app permissions, folder authorization, resource selection, and the `FEISHU_DOCX_PARSE_MODE` trade-offs, see [Feishu Drive Integration](24-feishu-drive.md). The default export mode and the blocks mode handle images and attachments differently; when sync deletion is enabled, the knowledge entries corresponding to the current data source are deleted.

Feishu and Lark (the international version, open.larksuite.com) are the same product deployed on two isolated clouds — their Wiki/docx/drive APIs are identical, so they **share the same connector code**, with the `Region` struct in `feishu/core/region.go` selecting the cloud (`RegionFeishu` / `RegionLark`, corresponding to type `feishu` / `lark`, and API domains `open.feishu.cn` / `open.larksuite.com` respectively). The `base_url` credential field can explicitly override this (for compatibility with legacy data sources where a feishu connector was historically pointed at larksuite).

- **Authentication** (`core/client.go`): `POST /open-apis/auth/v3/tenant_access_token/internal` exchanges for a tenant_access_token, cached with a mutex and refreshed on expiry.
- **Resource listing** (`ListResources`): three levels of lazy loading — `parentID==""` lists Wiki spaces; `parentID==spaceID` lists the space's top-level nodes; `parentID=="spaceID:nodeToken"` lists that node's children. Earlier versions eagerly recursed the entire tree, which timed out on large Wikis (issue #1672); recursion now only happens during sync. `ResolveResourceAncestors` walks up via `GetWikiNode`'s `parent_node_token` one level at a time, O(depth), to reflect deeply nested selections.
- **Content fetching** (`fetchNodeContent`) dispatches by `obj_type`:
  - `docx`/`doc` → asynchronous export API (`POST /drive/v1/export_tasks`) exports `.docx`;
  - `sheet`/`bitable` → exports `.xlsx`;
  - `file` → downloads the raw drive file (PDF/Word/images, etc.);
  - `mindnote`/`slides` → **skipped** (no content-reading API), with a summary log emitted via `fetchTally` (`discovered/fetched/failed/skipped_unsupported by_type`) explaining "why only 3 of the 13 discovered documents were synced" (issue #2136).
- **Incremental logic**: cursor `FeishuCursor.SpaceNodeTimes` (`resourceID → nodeToken → editTime`). Change detection uses `obj_edit_time` (document content edit time), **not** `node_edit_time` (which only reflects title changes or moves). Nodes that fail to fetch **do not advance the cursor** (the old editTime is kept, so prev != current next time and a retry is forced), preventing a document from being permanently skipped due to a transient export failure.
- **FetchStream**: unifies the full/incremental paths (cursor==nil means full sync); the cursor is checkpointed every `FeishuStreamCheckpointInterval = 50` nodes processed, or whenever more than `FeishuStreamCheckpointMaxInterval = 30s` has passed since the last checkpoint — the latter guards against the scenario of "few documents, but each export is extremely slow (rate-limited)," which could otherwise go without a single checkpoint before the 2-hour timeout.
- **Error classification** (`feishuFailure`): classifies raw errors into stable i18n codes (`feishu_auth_or_permission` / `feishu_rate_limited` / `feishu_timeout` / `feishu_server_unavailable` / `feishu_api_error`(+code) / `sync_failed`) for localized display on the frontend; the raw status/body/log_id is kept only in server-side logs.

#### GitLab (`connector/gitlab/`)

Select GitLab in the data source, fill in credentials.base_url and access_token, then choose the project, branch or tag, and directories. The token must be able to read the repository of the selected project; the visibility of private projects is determined by the GitLab credentials.

`config.settings.projects` is a non-empty array; each item contains a string project_id, an optional ref, and paths. An empty ref uses the default branch; empty paths select the whole repository, and directories use relative paths with forward slashes. Validate the credentials and browse the resources first, then save the scheduled sync.

Streaming sync supports resuming from checkpoints; incremental sync updates files based on repository commit diffs, and deletions at the source are handled according to sync_deletions. Selected repository files still go through WeKnora's file type, size, and parser engine checks, so not every code or binary file can be ingested directly. Files with identical content under different paths (such as README templates in each subdirectory) are kept as separate knowledge entries; duplicate detection only applies within the same data source and the same file path.

```json
{"credentials":{"base_url":"https://gitlab.example.com","access_token":"<token>"},"settings":{"projects":[{"project_id":"123","ref":"main","paths":["docs"]}]}}
```

#### Tencent IMA (`connector/ima/`)

Fill in credentials.client_id and api_key; base_url is optional and defaults to `https://ima.qq.com`. The resource list is the IMA knowledge bases visible to the credentials (a flat list, directories are not expanded), and the selection is saved as config.resource_ids; during sync, all subfolders of the selected knowledge bases are traversed recursively. When authorization fails at the source or resources are not visible, first check the IMA credentials and knowledge base access permissions.

Downloadable files go into document parsing, and web-page items are fetched by URL; notes are read through the note OpenAPI. AI sessions and video parsing have no usable content-reading entry point and are skipped. Both full and incremental sync are supported: a stable identity is built from the knowledge base, parent directory, and title, and when a file with the same name is replaced, the changed media_id triggers an update; deletions are detected only after a complete listing succeeds.

```json
{"credentials":{"client_id":"<client-id>","api_key":"<api-key>"},"resource_ids":["<resource-id-from-tree>"]}
```

For Feishu/Lark, the update time of synced records is taken from the content edit time, so body changes are not missed by relying only on Wiki node operation times. GitLab and IMA both support deletion detection; entries that naturally roll off an RSS feed are not treated as deletions.

#### Notion (`connector/notion/`)

- **Authentication**: Internal Integration Token (credential field `api_key`), API version `NotionAPIVersion = "2026-03-11"`, default `https://api.notion.com` (overridable via `Settings.base_url`).
- **Resource listing**: the Search API fetches all visible pages and databases in one call, returning a full tree with `ParentID` set (so lazy-load requests with `parentID != ""` return empty directly; `ResolveResourceAncestors` similarly has nothing to do). `resolveParentID` handles the `data_source` object from the 2025-09-03+ API: its `parent` points to the containing database, but the real workspace location depends on `database_parent`.
- **Fetching**: `fetchPage` recursively processes pages — `GetBlockChildrenAll` fetches blocks → `BlocksToMarkdown` (`markdown.go`) converts to Markdown; `file_upload`-type file blocks first go through `ResolveBlock` to get a temporary download URL; **attachments** (PDFs etc. — excluding images, which are already inlined in Markdown as `![](url)`) are downloaded as separate items; `child_page` / `child_database` blocks are recursed into. Databases come in two forms: an entire database is rendered as one Markdown table (`buildDatabaseItem`, with a per-record block-content appendix), while a database record appearing on its own is rendered as "property list + block content" (`buildRecordItem`). Property extraction (`propertyToString`) generically follows the `type` chain, covering all 22 property types; property names are sorted alphabetically to ensure deterministic incremental comparisons.
- **Incremental logic**: on the first sync (empty cursor), it simply delegates to `FetchAll` and builds the cursor from the returned items' `UpdatedAt`; subsequent syncs use `discoverAllResources` with the Search API + BFS to enumerate all descendants under the selected roots, comparing `last_edited_time` page by page. Databases go through `fetchDatabaseIncremental`: if any record changed, the entire table is rebuilt.
- **Distinguishing deletion from deselection**: a page that has disappeared at the source is reported as `IsDeleted`; a page that is still visible but no longer reachable because the user deselected an ancestor goes into the excluded set and is **not** falsely reported as deleted. `computeExcludedSet` also ensures that "new pages the user has never seen" are not excluded — a selected parent node automatically picks up new child pages.

#### Confluence (`connector/confluence/`)

Supports Confluence Server / Data Center and Confluence Cloud, both using HTTP Basic authentication.

| Field | Required | Description |
| --- | --- | --- |
| `edition` | No | `server` (default, Server / Data Center) or `cloud` |
| `base_url` | Yes | Confluence address; `https://` is added when the scheme is missing; for Cloud, `/wiki` is appended automatically when you enter `https://<site>.atlassian.net` |
| `username` | Yes | Server/DC username; for Cloud, the Atlassian account email |
| `password` | Required for Server/DC | Server/DC account password |
| `api_token` | Required for Cloud | Created on the API tokens page of your Atlassian account |

`edition`, `base_url`, and `username` are not secrets, so the UI also writes them into Settings so they can be shown when editing; `password` / `api_token` are stored only as encrypted credentials.

<Screenshot
  src="/screenshots/datasource-confluence.png"
  caption="Confluence data source: edition selection and credential entry"
  hint="Show the data source edit dialog with Confluence selected, displaying the 'Confluence edition' dropdown (Server / Data Center, Cloud), the address, username, and Cloud API token fields, and the space selection list after testing the connection." />

- **Scope**: the resource list is the spaces visible to the credentials (a flat list), and at least one space must be selected; all published pages in the space are synced, while attachments and blog posts are not imported.
- **Content**: the rendered HTML of each page is converted to Markdown (one `.md` item per page); when the body is empty, only the title is kept. Item metadata carries `space_key`, `space_name`, and `page_id`, plus `creator` when there is an author.
- **Incremental**: compared by page version number; pages whose version has not changed are skipped. Streaming sync is implemented, saving the cursor after each page is ingested; the mid-run cursor of a full sync also keeps the previous page baseline, so retried tasks can resume and still reconcile deletions.
- **Deletion protection**: when a selected space is inaccessible, when the page list is empty although there was content last time, or when at least 20 pages disappeared in this run and they make up more than 80% of the previous page count, the whole sync reports an error and no deletions are performed. With sync deletion enabled, pages that disappear in other cases are treated as deleted.
- **Errors**: a single failed page generates a placeholder item with the error information and the sync continues; the error codes are `confluence_auth_or_permission`, `confluence_not_found`, `confluence_rate_limited`, `confluence_server_unavailable`, `confluence_api_error`, and `confluence_sync_failed`, and the raw response is only recorded in server-side logs.

```json
{"credentials":{"edition":"cloud","base_url":"https://team.atlassian.net","username":"me@example.com","api_token":"<token>"},"resource_ids":["<space-id>"]}
```

#### Yuque (`connector/yuque/`)

- **Authentication**: a personal Token (Yuque Settings → Token) or a team Token, credential field `api_token` (header `X-Auth-Token`) + optional `base_url` (for a private, self-hosted domain; `https://` is auto-prepended if the scheme is missing).
- **Resource listing**: `GET /api/v2/user` determines the token's identity — `type=="Group"` means a team token, which lists team repos directly; otherwise it lists personal repos plus repos of groups the user has joined (Yuque returns 404 if the user hasn't joined any group, which is treated as empty). Outputs a flat list of `book` resources, stably sorted by ExternalID.
- **Fetching** (`walk`, shared by full and incremental sync): `ListBookDocs` lists documents → filters out `type != "Doc"` (skipping Sheet/Thread/Board/Table) and `status != "1"` (skipping drafts) → sleeps 300ms between each `GetDocDetail` call to avoid rate limiting → when `format` is `markdown`/`lake`, the raw `body` Markdown is ingested (other formats such as html are defensively skipped and logged with `skip_reason`).
- **Incremental logic**: cursor `yuqueCursor.BookDocTimes` (`bookID → docID → content_updated_at`); unchanged items are skipped. Deletion detection: present in the cursor but absent from the current list → `IsDeleted`.

#### DingTalk Docs (`connector/dingtalk/`)

- **Authentication**: the Client ID and Client Secret of an internal enterprise app, plus the Union ID of an operator who has access to the target knowledge base; enable `Wiki.Workspace.Read`, `Wiki.Node.Read`, and `Storage.File.Read`, then publish the app.
- **Scope**: select knowledge bases, folders, or individual `ALIDOC/adoc` online documents, which are converted to Markdown through the public Wiki / Blocks APIs. DingTalk sheets and ordinary uploaded attachments are not imported for now, and there is no dependency on asynchronous export callbacks.
- **Sync**: reads incrementally by document `modifiedTimestamp` (milliseconds), falling back to `modifiedTime` when it is missing; overlapping selections are merged. A full sync also reconciles deletions against the previous cursor, so `sync_mode=full` does not miss deletions. Deletions are deferred when directory traversal is incomplete.
- **Body**: the public Blocks API only returns the first-level blocks under the document root; containers such as highlight blocks are rendered further if the response includes `children`, otherwise `nested_blocks_unavailable` is marked in the metadata, so an incomplete body is not treated as a complete success.
- **Validation**: testing the connection lists knowledge bases, probes the root node list, and tries to read Blocks when online documents exist under the root, so a missing `Wiki.Node.Read` / `Storage.File.Read` is discovered early.
- **Failure and recovery**: an invalid resource does not block other scopes; failed scopes and documents whose body failed keep their old versions so they can be retried. When any scope cannot be scanned completely, deletions are deferred and records pending verification are kept. An invalid individual selection requires checking permissions or selecting again.
- **Deletion switch**: local knowledge confirmed as deleted at the source is removed only when sync deletion is enabled; inaccessible resources are never treated as deleted outright.

#### RSS / Atom (`connector/rss/`)

- **Configuration**: `feed_urls` (newline/comma-separated, deduplicated) is stored under **Settings** (non-secret, directly editable in the UI); `auth_headers` (one `Name: Value` per line, attached only to feed requests, never sent to third-party article pages) is stored under **Credentials** and encrypted. `HasConfiguredCredentials` has a special case for RSS: only `auth_headers` counts as configured credentials.
- **Fetching**: `gofeed` parses RSS/Atom/JSON feeds; when an entry has a link, the original article page is fetched and run through the readability extractor — if successful, the full text is used; otherwise it falls back to the feed's own content (`content:encoded`/`description`); HTML is converted to Markdown via `html-to-markdown/v2`. The entry ID is taken as the first non-empty value among `GUID > Link > Title`.
- **Incremental logic**: dual-layer fingerprinting — first compare the feed-side signal fingerprint (`feedSignalFingerprint`; if unchanged, the original article page isn't even fetched); then compare the SHA-256 fingerprint of the fetched content. **Deletion sync is not supported** (feeds naturally roll off old entries).
- **Partial failure**: if a single feed's fetch/parse fails, the old cursor is retained for it (`copyFeedCursor`) and the remaining feeds continue processing; the overall result is reported as `datasource.PartialFetchError` (SyncLog records `partial`); the whole sync only fails if every feed fails.

### Data Source Lifecycle and REST API

Routes are registered in `RegisterDataSourceRoutes` in `internal/router/routes_infra.go` (list, details, and sync logs require Viewer+, all other operations require Admin+; an API Key needs `manage_datasources` or full-access):

| Method & Path | Permission | Description |
| --- | --- | --- |
| `GET /api/v1/datasource/types` | Viewer | List of available connector metadata (`ListAvailableConnectors`, sorted by Priority) |
| `POST /api/v1/datasource/validate-credentials` | Admin | Test connectivity with raw credentials (not persisted), for the "Test Connection" button in the creation wizard |
| `POST /api/v1/datasource` | Admin | Create a data source (verify KB belongs to tenant → verify connector type → online Validate → persist → register cron) |
| `GET /api/v1/datasource?kb_id=` | Viewer | List data sources by knowledge base (includes the most recent SyncLog) |
| `GET /api/v1/datasource/:id` | Viewer | Details |
| `PUT /api/v1/datasource/:id` | Admin | Update (credentials fields are ignored; online validation is only triggered if the config actually changed and credentials already exist; cron is updated accordingly) |
| `DELETE /api/v1/datasource/:id` | Admin | Soft delete + remove cron + cancel pending/running SyncLogs |
| `PUT /api/v1/datasource/:id/credentials` | Admin | Atomically replace credentials (see [Encrypted Credential Storage](#encrypted-credential-storage)) |
| `DELETE /api/v1/datasource/:id/credentials/:field` | Admin | Clear credentials (field only accepts `credentials`) |
| `POST /api/v1/datasource/:id/validate` | Admin | Run a connectivity test against an existing data source; sets `status=error` on failure, clears error status on success |
| `GET /api/v1/datasource/:id/resources?parent_id=` | Admin | List selectable resources from the external system (parent_id supports lazy expansion) |
| `POST /api/v1/datasource/:id/resource-ancestors` | Admin | Resolve the ancestor chain of selected resources (to reflect deeply nested selections when editing) |
| `POST /api/v1/datasource/:id/sync` | Admin | Manually trigger a sync (creates a SyncLog + enqueues an Asynq task) |
| `POST /api/v1/datasource/:id/pause` / `resume` | Admin | Pause/resume (also removes/re-registers cron) |
| `GET /api/v1/datasource/:id/logs`, `GET /api/v1/datasource/logs/:log_id` | Viewer | Sync history |

All `:id` paths first go through `getOwnedDataSource` → `getOwnedKnowledgeBase` for **tenant isolation** checks (the data source's owning KB must belong to the current tenant, and pass the KB authorization check for the API key).

Lifecycle state transitions:

```mermaid
flowchart LR
    A["Create<br/>POST /datasource"] --> B["Authorize<br/>PUT /:id/credentials<br/>(AES-256-GCM encrypted persistence + online Validate)"]
    B --> C["Select resources<br/>GET /:id/resources<br/>(ResourceIDs written to Config)"]
    C --> D["active<br/>(cron schedule / manual sync)"]
    D -- "sync failed" --> E["error"]
    E -- "validate passed / sync succeeded" --> D
    D -- "POST /:id/pause" --> F["paused"]
    F -- "POST /:id/resume" --> D
    F -- "manual sync still allowed" --> D
    D -- "DELETE /:id" --> G["soft delete<br/>(remove cron + cancel unfinished SyncLog)"]
```

## Sync and Storage Reference

### Sync Scheduling (internal/datasource/scheduler.go)

`Scheduler` is based on `robfig/cron` (`cron.WithSeconds()`, supporting 6-field, second-level expressions) and maintains a cron entry for every active data source with a `SyncSchedule` configured; on service startup, `Start()` loads all active data sources from the DB and registers them in bulk.

Because robfig/cron fires based on **absolute wall-clock time** (e.g., `0 0 * * * *` always fires on the hour), all instances in a multi-instance deployment fire simultaneously. Deduplication relies on two layers:

1. **DB-level overlap prevention**: `syncLogRepo.HasRunningSync` — if the previous sync is still running, the current run is skipped (prevents overlapping executions when sync duration exceeds the cron interval).
2. **Redis-level cross-instance dedup**: a deterministic `asynq.TaskID = "dssync:<dsID>:<yyyyMMddHHmm>"` (truncated to the minute). All instances produce the same TaskID within the same minute; Redis guarantees only one enqueue succeeds, and the rest get `asynq.ErrTaskIDConflict`, with the corresponding SyncLog marked `canceled` ("deduplicated: another instance enqueued first").

Enqueue parameters: queue `types.QueueSync`, `MaxRetry(5)`, `Timeout(2*time.Hour)`. The task type is `types.TypeDataSourceSync` (`"datasource:sync"`), consumed by `mux.HandleFunc(types.TypeDataSourceSync, params.DataSourceService.ProcessSync)` in `internal/router/task.go`.

### Sync Execution and Knowledge Ingestion (datasource_service.go)

`ProcessSync` is the Asynq task handler; the full flow is shown in the sequence diagram below. Key points:

- **Defensive cancellation**: if the data source or knowledge base has already been deleted, the SyncLog is set to `canceled` and nil is returned (no further retries).
- **Two fetch paths**: connectors implementing `StreamingConnector` go through `processSyncStreaming` (streaming); otherwise, `ForceFull || SyncMode==full` goes through `FetchAll`, or with a cursor from `ParseSyncCursor()`, through `FetchIncremental` (batch).
- **Cursor strategy for the streaming path** (`streamStartCursor`): a user-triggered full sync discards the cursor and fetches everything on its **first attempt**; Asynq **retries** (attempt > 0) as well as all incremental syncs resume from the last checkpoint. Connectors that implement `FullStreamingConnector` (currently Confluence) use `FetchFullStream` for full syncs instead: all items are fetched again while the old cursor is kept as the baseline for reconciling deletions.
- **Ingestion core, `applyFetchedItem` → `ingestItem`**:
  - When `IsDeleted=true` and `sync_deletions=true`, the corresponding knowledge is looked up by tenant, knowledge base, data source ID, and external_id, and actually deleted; with sync deletion turned off, existing knowledge is kept. Deletion also depends on whether the connector provides reliable deletion detection;
  - Items with `Content` bytes are wrapped into a `multipart.FileHeader` and go through `KnowledgeService.CreateKnowledgeFromFile` (the full document parsing pipeline); items with only a `URL` go through `CreateKnowledgeFromURL`, downloaded and parsed by WeKnora;
  - **Update = delete then recreate**: if an existing knowledge entry is found via the `external_id` metadata, it is first `DeleteKnowledge`'d and then rebuilt, counted as Updated;
  - Duplicate files (`DuplicateKnowledgeError`) are counted as Skipped, not as a failure;
  - Every item automatically gets metadata attached: `external_id`, `source_resource_id`, `datasource_id`, plus any metadata added by the connector. If the source provides timestamps, `source_created_at` / `source_updated_at` are also stored in UTC RFC3339 format; they represent the source document's times and are separate from WeKnora's created_at/updated_at.
- **Automatic tagging**: `resolveAutoTagIDs` finds-or-creates a tag in the target KB based on the data source's name, and attaches it to all synced items, making it easy to identify the source within the KB; a tagging failure does not block the sync.
- **Result status**: if all items fail → `failed` (`allFetchedItemsFailedError`); if some RSS feeds fail (`PartialFetchError`) or the streaming path has failed documents → `partial`; otherwise → `success`. Failure samples are kept as `SyncItemError`, capped at 100.
- If a fetch fails but the connector still returned a new cursor (e.g., RSS), it is still persisted, avoiding a forced full re-fetch after a transient failure.

```mermaid
sequenceDiagram
    autonumber
    participant U as "User / Cron Scheduler"
    participant H as "DataSourceHandler"
    participant S as "DataSourceService"
    participant Q as "Asynq (QueueSync)"
    participant C as "Connector (e.g. Feishu)"
    participant EXT as "External System API"
    participant K as "KnowledgeService"
    participant DB as "PostgreSQL"

    U->>H: POST /datasource/:id/sync (or cron-triggered)
    H->>S: ManualSync(dsID)
    S->>DB: create SyncLog (status=running)
    S->>Q: Enqueue(datasource:sync, MaxRetry=5, Timeout=2h)
    Q-->>S: ProcessSync(payload)
    S->>DB: load DataSource / SyncLog / verify KB exists
    S->>S: ParseConfig() decrypts credentials
    alt "StreamingConnector (Feishu/Lark, GitLab, Confluence)"
        S->>C: FetchStream(config, cursor, handler)
        loop "iterate Wiki nodes"
            C->>EXT: ListWikiNodesRecursive / ExportAndDownload
            EXT-->>C: document content (.docx/.xlsx/raw file)
            C->>S: handler.Emit(item)
            S->>K: CreateKnowledgeFromFile (delete-then-create = update)
            C->>S: handler.Checkpoint(cursor) every 50 nodes or 30s
            S->>DB: persist LastSyncCursor + SyncLog progress
        end
        C-->>S: final cursor
    else "batch connectors (Notion/Yuque/DingTalk/IMA/RSS)"
        S->>C: FetchAll or FetchIncremental(cursor)
        C->>EXT: list + fetch changed content
        EXT-->>C: document / Markdown
        C-->>S: []FetchedItem + nextCursor
        loop "each item"
            S->>K: applyFetchedItem → ingestItem
        end
    end
    S->>DB: update SyncLog (success/partial/failed) + DataSource (LastSyncAt/Cursor/Result)
    S->>DB: write audit log (recordKBActivity)
```

### Encrypted Credential Storage

Credentials are handled separately on write, on read, and in API responses:

**1. Encryption on write — `DataSourceConfig.ToJSON()`** (`internal/types/datasource.go`):

```go
// When SYSTEM_AES_KEY is configured, every string value in Credentials is
// AES-256-GCM encrypted before serialization. This is the only write path for
// credentials into the DB (GORM's JSON type is a byte passthrough), so encrypting
// here guarantees DataSource.Config is stored as ciphertext end to end.
if key := utils.GetAESKey(); key != nil && len(out.Credentials) > 0 {
    ...
    if enc, err := utils.EncryptAESGCM(s, key); err == nil { encCreds[k] = enc }
}
```

**2. Decryption on read — `DataSource.ParseConfig()`**: transparently handles three cases — an empty string is returned as-is; legacy plaintext without the `enc:v1:` prefix is returned as-is (no migration needed); ciphertext is decrypted with `SYSTEM_AES_KEY`. If decryption fails (key lost/rotated), the corresponding credential field is cleared, the UI shows "credentials not configured," and the user just re-enters them without losing the data source's other settings.

**3. A dedicated credentials sub-resource — `internal/handler/datasource_credentials.go`**: credentials are not handled via the regular `PUT /datasource/:id` — instead they go through a separate `/credentials` sub-resource and are replaced atomically as a whole, ensuring the connector always receives a complete set of credentials:

- `PUT /api/v1/datasource/:id/credentials` — replaces the entire credentials map; immediately after replacement, the connector's `Validate` is called for an online check (invalid credentials return an error right away)
- `DELETE /api/v1/datasource/:id/credentials/credentials` — clears everything
- The response **never** returns the ciphertext/plaintext — only `{"credentials": {"configured": true/false}}`; the list/detail endpoints also strip `Credentials` by construction when serialized via `dto.NewDataSourceResponse`

The regular update endpoint `UpdateDataSource` (`datasource_service.go`) **always preserves the credentials already stored in the database**, even if the request body includes credentials — those are ignored and a warning is logged. Additionally, `StripNonSecretCredentials` strips out any non-secret fields mistakenly placed under credentials (currently only RSS's `feed_urls`, which belongs under `Settings`).

### Security Restrictions (internal/datasource/httpclient.go and errors.go)

`httpclient.go` provides two SSRF protection entry points shared by all connectors:

```go
// ValidateConnectorBaseURL applies SSRF policy validation to the connector base_url (empty passes through; the caller applies defaults)
func ValidateConnectorBaseURL(rawURL string) error {
    ...
    if err := utils.ValidateURLForSSRF(url); err != nil { ... }
}

// NewConnectorHTTPClient returns an HTTP client with redirect- and dial-time SSRF protection
func NewConnectorHTTPClient(timeout time.Duration) *http.Client {
    cfg := utils.DefaultSSRFSafeHTTPClientConfig()
    cfg.Timeout = timeout
    return utils.NewSSRFSafeHTTPClient(cfg)
}
```

The underlying `internal/utils/security.go` rejects targets such as private-network addresses, loopback addresses, and link-local addresses, and re-validates on **every redirect and at actual dial time** (not just the initial URL), preventing a malicious feed or custom base_url from directing WeKnora at an internal service. Each connector's `parseXXXConfig` calls `ValidateConnectorBaseURL` on its base_url.

`errors.go` defines module-level sentinel errors (`ErrConnectorNotFound`, `ErrDataSourceInvalid`, `ErrInvalidCredentials`, `ErrSyncFailed`, etc.) and `PartialFetchError` (some resources succeeded, some failed; the caller should process the items obtained so far, persist the cursor, and present `Details` to the user as a partial status).

### Core Abstraction: the Connector Interface

All connectors must implement the `Connector` interface in `internal/datasource/connector.go`:

```go
type Connector interface {
    // Type returns the connector type identifier (e.g. "feishu", "notion")
    Type() string
    // Validate verifies the config and credentials by actually calling the external API
    Validate(ctx context.Context, config *types.DataSourceConfig) error
    // ListResources lists synchronizable resources (documents, spaces, folders, etc.).
    // parentID enables lazy loading of hierarchical resources: "" = top level; non-empty = direct children of that resource
    ListResources(ctx context.Context, config *types.DataSourceConfig, parentID string) ([]types.Resource, error)
    // ResolveResourceAncestors resolves the ancestor chain of selected resources for deep-selection echo in the lazy-loaded picker
    ResolveResourceAncestors(ctx context.Context, config *types.DataSourceConfig, resourceIDs []string) ([]string, error)
    // FetchAll performs a full sync of the given resources
    FetchAll(ctx context.Context, config *types.DataSourceConfig, resourceIDs []string) ([]types.FetchedItem, error)
    // FetchIncremental performs cursor-based incremental sync, returning changed items and the new cursor for the next sync
    FetchIncremental(ctx context.Context, config *types.DataSourceConfig, cursor *types.SyncCursor) ([]types.FetchedItem, *types.SyncCursor, error)
}
```

#### Optional extension: StreamingConnector (streaming, resumable sync)

`StreamingConnector` fetches, ingests, and checkpoints the cursor item by item, which suits large-scale syncs and avoids the memory cost of buffering all content at once:

```go
type StreamHandler interface {
    // Emit stores one fetched item; returning an error aborts the whole stream
    Emit(ctx context.Context, item types.FetchedItem) error
    // Checkpoint persists a cursor snapshot (must be a fully restorable snapshot, not an incremental one)
    Checkpoint(ctx context.Context, cursor *types.SyncCursor) error
}

type StreamingConnector interface {
    Connector
    FetchStream(ctx context.Context, config *types.DataSourceConfig,
        cursor *types.SyncCursor, h StreamHandler) (*types.SyncCursor, error)
}
```

After a task times out, processing can continue from the most recent checkpoint. The Asynq sync task timeout is 2 hours, and the streaming path advances item by item, avoiding buffering the bodies of all files. The Feishu/Lark (Wiki and Drive), GitLab, and Confluence connectors implement `StreamingConnector`; Confluence additionally implements `FullStreamingConnector`, so full syncs can also reconcile deletions. DingTalk currently uses the batch `FetchAll`/`FetchIncremental`/`FetchAllFromCursor`; for very large knowledge bases, incremental mode is recommended to avoid loading all Markdown at once.

#### ConnectorRegistry: registration and lookup

`ConnectorRegistry` is a simple `map[string]Connector` registry. Actual registration happens in `initConnectorRegistry()` in `internal/container/container.go`:

```go
registry.Register(wiki.NewConnector(core.RegionFeishu))             // feishu
registry.Register(wiki.NewConnector(core.RegionLark))               // lark (international edition, same implementation with a different Region)
registry.Register(drive.NewDriveConnector(core.RegionFeishuDrive))  // feishu_drive
registry.Register(drive.NewDriveConnector(core.RegionLarkDrive))    // lark_drive
registry.Register(notionConnector.NewConnector())                   // notion
registry.Register(confluenceConnector.NewConnector())               // confluence
registry.Register(yuqueConnector.NewConnector())                    // yuque
registry.Register(dingtalkConnector.NewConnector())                 // dingtalk
registry.Register(imaConnector.NewConnector())                      // ima
registry.Register(rssConnector.NewConnector())                      // rss
registry.Register(gitlabConnector.NewConnector())                   // gitlab
```

> Note: the `ConnectorMetadataRegistry` in `connector.go` still includes connectors that are not yet implemented (GitHub, Google Drive, OneDrive, Web Crawler, Slack, IMAP, etc.). The types currently registered and usable are: `feishu`, `lark`, `feishu_drive`, `lark_drive`, `notion`, `confluence`, `yuque`, `dingtalk`, `ima`, `rss`, `gitlab`. Unregistered types are rejected by `connectorRegistry.Get()` with `ErrConnectorNotFound` when creating a data source.

### Data Model (internal/types/datasource.go)

| Struct | Description |
| --- | --- |
| `DataSource` | Data source configuration entity (table `data_sources`). Key fields: `Type` (connector type), `Config` (JSONB, containing encrypted credentials), `SyncSchedule` (cron expression), `SyncMode` (`incremental`/`full`), `Status` (`active`/`paused`/`error`/`deleted`), `ConflictStrategy`, `SyncDeletions`, `LastSyncCursor` (incremental cursor, JSONB), `LastSyncAt`, `LastSyncResult`, `SyncLogRetentionDays` |
| `SyncLog` | Record of a single sync run (table `sync_logs`). Status: `running`/`success`/`partial`/`failed`/`canceled`; counts: `ItemsTotal/Created/Updated/Deleted/Skipped/Failed`; `Result` stores the `SyncResult` JSON |
| `DataSourceConfig` | Decrypted configuration struct: `Type` + `Credentials map[string]interface{}` + `ResourceIDs []string` (selected resources) + `Settings map[string]interface{}` (non-secret configuration) |
| `Resource` | A selectable resource from the external system: `ExternalID`, `Name`, `Type`, `URL`, `ParentID`, `HasChildren`, `ModifiedAt`, `Metadata` |
| `FetchedItem` | A single fetched document: `ExternalID`, `Title`, `Content []byte`, `ContentType`, `FileName`, `URL`, `UpdatedAt`, `Metadata`, `IsDeleted`, `SourceResourceID` |
| `SyncCursor` | Incremental cursor: `LastSyncTime` + `ConnectorCursor map[string]interface{}` (connector-specific structure) |
| `SyncResult` | Sync result summary + `Errors []SyncItemError` (failure samples, capped at 100, see `maxSyncResultErrors`) |
| `SyncItemError` | User-facing failure sample: a stable i18n `Code` + interpolation `Params` + fallback `Message`; the raw API status code/response body is kept only in server-side logs |
| `DataSourceSyncPayload` | Asynq task payload: `DataSourceID`, `TenantID`, `SyncLogID`, `ForceFull`, `Trigger` (`manual`/`schedule`) |

## References

- Connector development guide (maintained alongside the code): `internal/datasource/CONNECTOR_IMPLEMENTATION_GUIDE.md`
- Module documentation (maintained alongside the code): `internal/datasource/README.md`

## Implementation Reference

- Connector framework and implementations: `internal/datasource/` (`connector.go`, `scheduler.go`, `httpclient.go`, `errors.go`, implementations under `connector/`)
- HTTP interface layer: `internal/handler/datasource.go`, `internal/handler/datasource_credentials.go`
- Business service layer: `internal/application/service/datasource_service.go`
- Data model: `internal/types/datasource.go`
