# Data Source Import Development Documentation

WeKnora's data source import module supports automatically importing and syncing content from external platforms (Feishu, WeCom, Notion, Confluence, etc.) into the knowledge base. Users can configure data source connections, select the resources to sync, and complete incremental/full content synchronization automatically via manual triggers or scheduled scheduling.

Data sources are bound to a knowledge base, and a single knowledge base can connect to multiple data sources. All configuration is managed through the frontend knowledge base settings page, and credentials are stored encrypted in the database using AES-256-GCM.

## Table of Contents

- [Quick Start Guide](#quick-start-guide)
  - [Feishu Knowledge Base Integration](#feishu-knowledge-base-integration)
- [Frontend Management](#frontend-management)
- [Architecture Overview](#architecture-overview)
- [Data Model](#data-model)
- [API Endpoints](#api-endpoints)
- [Core Concepts](#core-concepts)
- [Sync Processing Flow](#sync-processing-flow)
- [Interface Definitions](#interface-definitions)
- [Connector Details](#connector-details)
  - [Feishu](#feishu-feishu)
- [Scheduled Scheduling](#scheduled-scheduling)
- [Key Parameters and Thresholds](#key-parameters-and-thresholds)
- [Error Handling](#error-handling)
- [Extending with New Connectors](#extending-with-new-connectors)

---

## Quick Start Guide

### Prerequisites

- WeKnora is deployed and running
- At least one knowledge base has been created
- You have admin permissions on the external platform (e.g., Feishu, WeCom) to create apps and authorize them

### Feishu Knowledge Base Integration

Sync documents from a Feishu knowledge base (Wiki) to WeKnora. Automatic import is supported for docx, doc, sheet, bitable, and file type documents.

#### Step 1: Create a Feishu App

1. Log in to the [Feishu Open Platform](https://open.feishu.cn/) → **Developer Console** → **Create a Custom App**
2. On the **Credentials & Basic Info** page, obtain:
   - **App ID**
   - **App Secret**

#### Step 2: Enable Permissions

Search for and enable the following permissions in **Permission Management**:

| Permission | Description |
|------|------|
| `wiki:wiki:readonly` | Read the list of knowledge base spaces and the node tree |
| `drive:drive:readonly` | Read basic cloud document information |
| `drive:export:readonly` | Export document content (docx/xlsx) |
| `docx:document:readonly` | Read new-version document content |

> **Drill-down permissions for embedded docx content (optional)**
>
> If you need to sync attachments and images embedded within docx documents, you'll also need to enable the following additional permissions (if missing, the system automatically falls back to exporting the docx — sync won't fail, but embedded tables/attachments won't be retrieved):
>
> - `docx:document:readonly` — read document blocks (already included above; listed here just for reference)
> - `drive:drive:readonly` / `docs:document.media:download` — download attachments and images
> - `sheets:spreadsheet:readonly` — read embedded spreadsheets
> - `bitable:app:readonly` — read embedded multi-dimensional tables

#### Step 3: Publish the App

Create a version in **Version Management & Release** and submit it for review. The API can only be called normally once the review is approved.

#### Step 4: Add the Data Source in WeKnora

1. Go to the knowledge base settings page → **Data Sources** tab
2. Click **Add Data Source**
3. Select connector type: **Feishu**
4. Fill in the credentials:
   - **App ID**: enter the App ID obtained from Feishu
   - **App Secret**: enter the App Secret obtained from Feishu
5. Click **Test Connection** to verify the credentials are valid
6. Select the knowledge base spaces (Wiki Spaces) to sync
7. Configure the sync strategy:
   - **Sync Mode**: incremental sync (recommended) or full sync
   - **Sync Frequency**: choose a preset Cron expression (e.g., every 30 minutes, every hour, daily at 2 AM)
   - **Conflict Strategy**: overwrite (recommended) or skip
   - **Sync Deletions**: whether to sync deletions from the source

8. Click Save

#### Step 5: Verify

After saving, you can click **Sync Now** to trigger the first sync. Check the sync progress and results in the sync log:

- **Success**: shows the number of documents created/updated/skipped/failed
- **Failure**: shows the error message; common causes are insufficient permissions or an expired token

---

### Lark Knowledge Base Integration

Lark is the international version of Feishu. The Wiki / docx / drive APIs are identical to Feishu's, **sharing the same connector code**. The integration steps are the same as [Feishu Knowledge Base Integration](#feishu-knowledge-base-integration), with only three differences:

| | Feishu | Lark |
|---|---|---|
| Open Platform | <https://open.feishu.cn/> | <https://open.larksuite.com/> |
| Connector Type | "Feishu" | "Lark" |
| API Endpoint | `https://open.feishu.cn` | `https://open.larksuite.com` (used automatically once the Lark type is selected) |

#### Lark Permissions

The permission identifiers are the same as Feishu's; enable them in **Permission Management** on the Lark Open Platform:

| Permission | Description | Verified against Lark's official docs |
|------|------|------|
| `wiki:wiki:readonly` | Read the list of knowledge base spaces and the node tree | ✅ Verified |
| `drive:export:readonly` | Export document content (docx/xlsx) | ✅ Verified |
| `drive:drive:readonly` | Read basic cloud document information, download files | ✅ Verified |
| `docx:document:readonly` | Read new-version document content | ⚠️ Could not be verified (Lark's documentation page rendering is limited); carried over from the Feishu-side identifier |

> **Do not reuse the permission JSON from the [IM Integration Documentation](./IM-Integration-Development.md#feishu-integration)**: that list mixes in permissions such as
> `aily:file:*` and `corehr:file:download`, which don't exist on Lark (importing the whole set will fail),
> and it's **missing** the `drive:*` / `docx:*` permissions the data source actually needs, as shown in the table above.
>
> If the same Lark app serves both as an IM bot and for knowledge base syncing, use the merged list from
> [IM Integration Documentation — Lark Permission Configuration](./IM-Integration-Development.md#lark-integration).

#### Apps Are Not Interchangeable

An app created at open.feishu.cn cannot read a Lark knowledge base, and vice versa — credentials are only valid on the cloud where they were created.
If you previously connected to Lark using the "Feishu" connector with `base_url=https://open.larksuite.com`, that configuration
still works (`base_url` remains available as an explicit override), but for new data sources, select the "Lark" type directly.

---

## Frontend Management

Data sources are managed on the **Data Sources** tab of the knowledge base settings page.

### Creation Wizard

Adding a data source uses a multi-step wizard flow:

1. **Select Type**: choose a platform from the list of available connectors
2. **Configure Credentials**: enter the platform credentials, with instant "Test Connection" verification supported
3. **Select Resources**: browse and check the resources to sync (e.g., Feishu knowledge base spaces)
4. **Sync Strategy**: configure Cron scheduling, sync mode, conflict strategy, etc.

### Data Source List

Each data source is displayed as a card, showing:

- **Connector Type Badge**: Feishu (blue), etc.
- **Data Source Name**: user-defined
- **Sync Status**: Active / Paused / Error
- **Last Sync Time**: time of the last successful sync
- **Last Sync Result**: summary of the number of documents synced
- **Action Buttons**: Sync Now, Pause/Resume, Edit, Delete

### Sync Logs

Each data source has a history of sync logs, including:

- Sync status: Running / Success / Partial Success / Failed / Canceled
- Start time / End time
- Document counts: total, created, updated, deleted, skipped, failed
- Error message (if any)

---

## Architecture Overview

```
┌───────────────────────────────────────────────────────────────────────────┐
│                          Data Source Import Architecture                                     │
│                                                                           │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│   │  Feishu   │  │  Notion  │  │Confluence│  │  GitHub  │  │ Others... │   │
│   │  (Wiki)  │  │          │  │          │  │          │  │          │   │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│        │ API         │ API         │ API         │ API         │          │
│   ─────┼─────────────┼────────────┼─────────────┼─────────────┼──────    │
│        ▼             ▼            ▼             ▼             ▼          │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│   │  Feishu  │  │  Notion  │  │Confluence│  │  GitHub  │  │   ...    │  │
│   │Connector │  │Connector │  │Connector │  │Connector │  │Connector │  │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  │
│        │             │             │              │             │   Connec-  │
│   ─────┴─────────────┴─────────────┴──────────────┴─────────────┘   tor  │
│                      │                                                    │
│                      ▼                                                    │
│   ┌──────────────────────────────────────────────┐                        │
│   │         ConnectorRegistry                    │   Connector Registry          │
│   │   · Find connector instances by type                        │                       │
│   │   · Connector metadata (name, auth method, capabilities)          │                       │
│   └──────────────────┬───────────────────────────┘                        │
│                      │                                                    │
│   ───────────────────┼────────────────────────────────────────────────    │
│                      ▼                                                    │
│   ┌──────────────────────────────────────────────┐                        │
│   │       DataSourceService (business logic layer)          │                        │
│   │                                              │                        │
│   │  ┌────────────────────────────────────────┐  │                        │
│   │  │ · Data source CRUD                          │  │                        │
│   │  │ · Connection validation (ValidateConnection)        │  │                        │
│   │  │ · Resource discovery (ListAvailableResources)    │  │                        │
│   │  │ · Manual sync (ManualSync)                │  │                        │
│   │  │ · Pause / resume                            │  │                        │
│   │  │ · Sync execution (ProcessSync)               │  │                        │
│   │  │ · Content ingestion (ingestItem)                │  │                        │
│   │  └────────────────────────────────────────┘  │                        │
│   └──────────────────┬───────────────────────────┘                        │
│                      │                                                    │
│   ───────────────────┼────────────────────────────────────────────────    │
│                      ▼                                                    │
│   ┌──────────────────────────────────┐  ┌────────────────────────────┐    │
│   │      Scheduler (Cron scheduler)     │  │    Task Queue (asynq)     │    │
│   │  · robfig/cron (with seconds)           │  │  · Async sync tasks            │    │
│   │  · DB loads active data sources             │  │  · Redis / Lite mode      │    │
│   │  · Task dedup (TaskID + Running)   │  │  · TypeDataSourceSync     │    │
│   └──────────────────────────────────┘  └────────────────────────────┘    │
│                      │                                                    │
│   ───────────────────┼────────────────────────────────────────────────    │
│                      ▼                                                    │
│   ┌──────────────────────────────────────────────────────┐                │
│   │            WeKnora Core (knowledge ingestion pipeline)                │                │
│   │  KnowledgeService · KnowledgeBaseService             │                │
│   │  document parsing → chunking → embedding → indexing                      │                │
│   └──────────────────────────────────────────────────────┘                │
└───────────────────────────────────────────────────────────────────────────┘
```

**Design Patterns:**

| Pattern | Purpose |
|------|------|
| Adapter Pattern | Unifies differences across platforms; each platform implements the `Connector` interface |
| Registry Pattern | Dynamically looks up connector instances by type via `ConnectorRegistry` |
| Strategy Pattern | Selectable strategies for `SyncMode` (incremental/full) and `ConflictStrategy` (overwrite/skip) |
| Producer-Consumer | Manual/scheduled trigger → asynq task queue → Worker executes sync asynchronously |
| Cursor-based Pagination | Incremental sync tracks changes based on `SyncCursor` state |

---

## Data Model

### data_sources Table

Data source configurations are stored in the `data_sources` table, bound to a knowledge base:

```sql
CREATE TABLE data_sources (
    id                      VARCHAR(36) PRIMARY KEY,
    tenant_id               BIGINT NOT NULL,
    knowledge_base_id       VARCHAR(36) NOT NULL,
    name                    VARCHAR(255) NOT NULL DEFAULT '',
    type                    VARCHAR(50) NOT NULL,          -- Connector type
    config                  JSONB NOT NULL DEFAULT '{}',   -- encrypted credentials and config
    sync_schedule           VARCHAR(100) DEFAULT '',       -- Cron expression (6 fields, with seconds)
    sync_mode               VARCHAR(20) DEFAULT 'incremental',  -- 'incremental' | 'full'
    status                  VARCHAR(20) DEFAULT 'active',  -- 'active' | 'paused' | 'error' | 'deleted'
    conflict_strategy       VARCHAR(20) DEFAULT 'overwrite',    -- 'overwrite' | 'skip'
    sync_deletions          BOOLEAN DEFAULT true,
    last_sync_at            TIMESTAMPTZ,
    last_sync_cursor        JSONB,
    last_sync_result        JSONB,
    error_message           TEXT DEFAULT '',
    sync_log_retention_days INTEGER DEFAULT 30,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at              TIMESTAMPTZ
);
```

**Structure of the `config` field (after decryption):**

```json
{
  "type": "feishu",
  "credentials": {
    "app_id": "cli_xxx",
    "app_secret": "xxx"
  },
  "resource_ids": ["space_id_1", "space_id_2"],
  "settings": {}
}
```

**Credential fields per connector:**

| Connector Type | Credential Fields | Auth Method |
|-----------|---------|---------|
| Feishu (feishu) | `app_id`, `app_secret` | OAuth2 (Tenant Access Token) |

### sync_logs Table

The sync log records the execution of each sync operation:

```sql
CREATE TABLE sync_logs (
    id              VARCHAR(36) PRIMARY KEY,
    data_source_id  VARCHAR(36) NOT NULL REFERENCES data_sources(id) ON DELETE CASCADE,
    tenant_id       BIGINT NOT NULL,
    status          VARCHAR(20) NOT NULL,    -- 'running' | 'success' | 'partial' | 'failed' | 'canceled'
    started_at      TIMESTAMPTZ NOT NULL,
    finished_at     TIMESTAMPTZ,
    items_total     INTEGER DEFAULT 0,
    items_created   INTEGER DEFAULT 0,
    items_updated   INTEGER DEFAULT 0,
    items_deleted   INTEGER DEFAULT 0,
    items_skipped   INTEGER DEFAULT 0,
    items_failed    INTEGER DEFAULT 0,
    error_message   TEXT DEFAULT '',
    result          JSONB,                   -- detailed sync result
    created_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

---

## API Endpoints

All endpoints are under the `/api/v1/datasource` path and require authentication.

### Connector Metadata

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/datasource/types` | Get the list of available connector types |

**Example response:**

```json
[
  {
    "type": "feishu",
    "name": "Feishu",
    "description": "Import documents from Feishu/Lark Wiki spaces",
    "icon": "feishu",
    "priority": 1,
    "auth_type": "oauth2",
    "capabilities": ["incremental", "deletion_sync"]
  }
]
```

### Credential Validation (not persisted)

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/datasource/validate-credentials` | Validate credentials (test connection before creating) |

**Request body:**

```json
{
  "type": "feishu",
  "credentials": {
    "app_id": "cli_xxx",
    "app_secret": "xxx"
  }
}
```

### Data Source Management

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/datasource` | Create a data source |
| GET | `/api/v1/datasource?kb_id=xxx` | List all data sources for a knowledge base |
| GET | `/api/v1/datasource/:id` | Get data source details |
| PUT | `/api/v1/datasource/:id` | Update data source configuration |
| DELETE | `/api/v1/datasource/:id` | Delete a data source (soft delete) |

### Operations

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/datasource/:id/validate` | Test the connection for an existing data source (updates status) |
| GET | `/api/v1/datasource/:id/resources` | Browse available resources on the external system |
| POST | `/api/v1/datasource/:id/sync` | Manually trigger a sync |
| POST | `/api/v1/datasource/:id/pause` | Pause a data source (stops scheduled sync) |
| POST | `/api/v1/datasource/:id/resume` | Resume a data source |

### Sync Logs

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/datasource/:id/logs?limit=10&offset=0` | Get the list of sync logs |
| GET | `/api/v1/datasource/logs/:log_id` | Get details of a single sync log |

---

## Core Concepts

### DataSource

Each data source represents a binding between an external platform connection and a WeKnora knowledge base. A single knowledge base can add multiple data sources, aggregating content from multiple platforms.

Core attributes of a data source:

- **Type**: the bound connector type, e.g. `feishu`, `notion`, etc.
- **Config**: encrypted-at-rest credentials, the list of selected resource IDs, and additional settings
- **SyncMode**: `incremental` (only syncs changes) or `full` (pulls everything every time)
- **SyncSchedule**: a 6-field Cron expression (including seconds); if empty, only manual triggering is supported
- **ConflictStrategy**: how to handle cases where both the source and local copy have changes
- **SyncDeletions**: whether to delete local knowledge when the source document is deleted

### Connector

A connector is the adapter layer that interacts with an external platform; each platform implements one connector. All connectors are registered and managed via `ConnectorRegistry`.

Connectors provide four core capabilities:

| Capability | Description |
|------|------|
| `Validate` | Verifies whether the credentials and connection are usable |
| `ListResources` | Lists the resources available for selection in the external system (e.g. knowledge base spaces, folders) |
| `FetchAll` | Fully fetches all documents within specified resources |
| `FetchIncremental` | Incrementally fetches changed documents based on a cursor |

### SyncCursor

A state marker for incremental sync, recording the position of the last sync. The cursor content varies by connector type and is managed by each connector itself. For example, the Feishu connector's cursor includes the last edit time of each node, used to determine which documents have changed.

```go
type SyncCursor struct {
    LastSyncTime    *time.Time             // last sync time
    ConnectorCursor map[string]interface{} // Connector custom state
    LastSchemaHash  string                 // schema change detection
}
```

### FetchedItem

After a connector fetches a document from an external platform, it is wrapped uniformly as a `FetchedItem`:

```go
type FetchedItem struct {
    ExternalID       string            // unique document ID on the external platform
    Title            string            // document title
    Content          []byte            // document content (binary)
    ContentType      string            // MIME type
    FileName         string            // file name (with extension)
    URL              string            // original URL
    UpdatedAt        *time.Time        // last update time
    Metadata         map[string]string // metadata (source, author, etc.)
    IsDeleted        bool              // marked as deleted (incremental sync)
    SourceResourceID string            // owning resource ID
}
```

### Resource

A resource node in the external system that can be selected for syncing, supporting a hierarchical structure:

```go
type Resource struct {
    ExternalID  string            // external ID
    Name        string            // name
    Type        string            // type (e.g. wiki_space)
    Description string            // description
    URL         string            // external link
    ModifiedAt  *time.Time        // last modified time
    ParentID    string            // parent node ID (supports tree structures)
    Metadata    map[string]string // additional info
}
```

---

## Sync Processing Flow

### Complete Sync Flow

```
Manual trigger / Cron scheduled trigger
        │
        ▼
┌─ Create SyncLog (status: running) ──────────────────┐
│  Enqueue async task (asynq: TypeDataSourceSync)           │
│  · Manual sync → queue: default                        │
│  · Scheduled sync → queue: low, TaskID dedup               │
└──────────────────────────┬────────────────────────┘
                           │
                           ▼
┌─ ProcessSync (async worker) ───────────────────┐
│  1. Parse the task body (DataSourceSyncPayload)             │
│  2. Load the data source config                                  │
│  3. Decrypt credentials → get the Connector                      │
│  4. Set the workspace context                                  │
│  5. Create/get the auto label (Connector name · data source name)            │
│  6. Determine the sync mode:                                   │
│     ├─ ForceFull or SyncModeFull → FetchAll        │
│     └─ SyncModeIncremental → FetchIncremental      │
│  7. Iterate over the FetchedItem list:                         │
│     └─ ingestItem → KnowledgeService               │
│  8. Update SyncLog (count, status, duration)                 │
│  9. Update DataSource (cursor, last sync time, result)       │
└──────────────────────────────────────────────────┘
```

### ingestItem — Single Document Ingestion

```
FetchedItem
    │
    ├─ IsDeleted = true?
    │     └─ Look up existing knowledge by external_id → soft delete
    │
    ├─ Has file content (Content)?
    │     └─ CreateKnowledgeFromFile
    │        · Look up existing knowledge by external_id
    │        · Exists → delete old version → rebuild
    │        · Does not exist → create new
    │        · metadata: external_id, source_resource_id, datasource_id
    │        · channel: Connector type (e.g. "feishu")
    │        · Auto-associate the label
    │
    └─ Only URL (no Content)?
          └─ CreateKnowledgeFromURL
             · Same dedup logic as above
```

### Data Source Lifecycle

```
Create data source (frontend wizard)
        │
        ▼
┌─ DataSourceService.Create ────────────────────┐
│  1. Verify the knowledge base exists                              │
│  2. Get the corresponding Connector                         │
│  3. Validate config and credentials (connector.Validate)         │
│  4. Encrypt credentials and persist                               │
│  5. If SyncSchedule exists → register Cron job         │
└───────────────────────────────────────────────┘

Pause data source:
  status → paused, remove Cron job

Resume data source:
  status → active, re-register Cron job

Delete data source:
  deleted_at = NOW(), remove Cron job, cascade-delete sync_logs

Connector errors:
  ValidateConnection fails → status → error, record error_message
  ValidateConnection succeeds and was error → status → active, clear error_message
```

---

## Interface Definitions

### Connector — Connector Interface (must be implemented)

```go
type Connector interface {
    Type() string
    Validate(ctx context.Context, config *types.DataSourceConfig) error
    ListResources(ctx context.Context, config *types.DataSourceConfig) ([]types.Resource, error)
    FetchAll(ctx context.Context, config *types.DataSourceConfig, resourceIDs []string) ([]types.FetchedItem, error)
    FetchIncremental(ctx context.Context, config *types.DataSourceConfig, cursor *types.SyncCursor) ([]types.FetchedItem, *types.SyncCursor, error)
}
```

| Method | Responsibility |
|------|------|
| `Type()` | Returns the connector type identifier, used for registration and routing |
| `Validate()` | Verifies credentials and connection usability, e.g. by attempting to obtain an access token |
| `ListResources()` | Lists the top-level resources available for selection in the external system (e.g. knowledge base spaces, workspaces) |
| `FetchAll()` | Fully fetches all documents under the specified resources, returning a list of `FetchedItem` |
| `FetchIncremental()` | Incrementally fetches changes based on the last cursor, returning the change list and a new cursor |

### DataSourceService — Business Service Interface

```go
type DataSourceService interface {
    CreateDataSource(ctx context.Context, ds *types.DataSource) (*types.DataSource, error)
    GetDataSource(ctx context.Context, id string) (*types.DataSource, error)
    ListDataSources(ctx context.Context, knowledgeBaseID string) ([]*types.DataSource, error)
    UpdateDataSource(ctx context.Context, ds *types.DataSource) (*types.DataSource, error)
    DeleteDataSource(ctx context.Context, id string) error
    ValidateConnection(ctx context.Context, dataSourceID string) error
    ValidateCredentials(ctx context.Context, connectorType string, credentials map[string]interface{}) error
    ListAvailableResources(ctx context.Context, dataSourceID string) ([]types.Resource, error)
    ManualSync(ctx context.Context, dataSourceID string) (*types.SyncLog, error)
    PauseDataSource(ctx context.Context, dataSourceID string) error
    ResumeDataSource(ctx context.Context, dataSourceID string) error
    GetSyncLogs(ctx context.Context, dataSourceID string, limit, offset int) ([]*types.SyncLog, error)
    GetSyncLog(ctx context.Context, syncLogID string) (*types.SyncLog, error)
    ProcessSync(ctx context.Context, task *asynq.Task) error
}
```

### ConnectorMetadata — Connector Metadata

```go
type ConnectorMetadata struct {
    Type         string   `json:"type"`
    Name         string   `json:"name"`
    Description  string   `json:"description"`
    Icon         string   `json:"icon"`
    Priority     int      `json:"priority"`
    AuthType     string   `json:"auth_type"`
    Capabilities []string `json:"capabilities"`
}
```

| Field | Description |
|------|------|
| `Type` | Connector type identifier (e.g. `feishu`) |
| `Name` | Display name (e.g. `Feishu`) |
| `Priority` | Sort priority; lower values come first |
| `AuthType` | Auth method: `oauth2` / `api_key` / `token` / `password` / `none` |
| `Capabilities` | Supported capabilities: `incremental` (incremental sync), `webhook`, `deletion_sync` (deletion sync) |

---

## Connector Details

### Feishu

The Feishu connector supports syncing documents from a Feishu knowledge base (Wiki) to WeKnora.

#### Authentication Mechanism

```
App ID + App Secret
        │
        ▼
  POST /open-apis/auth/v3/tenant_access_token/internal
        │
        ▼
  Get Tenant Access Token (valid for 2 hours)
        │
        ▼
  Cache the token, refresh 5 minutes before expiry
```

- **Auth Type**: Custom enterprise app, Tenant Access Token
- **Token Validity**: 2 hours, built-in caching, automatically refreshed 5 minutes before expiry
- **API Endpoint**: defaults to `https://open.feishu.cn`; the international Lark version uses `https://open.larksuite.com`

#### Resource Discovery

`ListResources` calls the Feishu Wiki Space API to list all knowledge base spaces:

```
GET /open-apis/wiki/v2/spaces (paginated, page_size=50)
```

Each knowledge base space is mapped to a `Resource`:

| Field | Value |
|------|------|
| `Type` | `wiki_space` |
| `ExternalID` | Space ID (space_id) |
| `URL` | `https://feishu.cn/wiki/{space_id}` |
| `Metadata.visibility` | Space visibility |

#### Full Sync (FetchAll)

```
For each selected Wiki Space:
  1. Recursively list all nodes in the space (ListAllWikiNodesRecursive)
     └─ GET /open-apis/wiki/v2/spaces/{space_id}/nodes (empty parent → top level)
     └─ Recurse into nodes with has_child=true
  2. For each node:
     └─ fetchNodeContent → determine document type → export/download
```

#### Incremental Sync (FetchIncremental)

Incremental sync is based on comparing node edit times, rather than Feishu event subscriptions:

```
1. Load the last cursor (feishuCursor)
   └─ SpaceNodeTimes[space_id][node_token] = last recorded edit time

2. Recursively list all nodes for each Space (full-tree traversal)

3. Compare changes:
   ├─ New node (not in cursor) → fetch content
   ├─ obj_edit_time changed (preferred) or node_edit_time changed → fetch content
   ├─ Edit time unchanged → skip
   └─ In the cursor but absent this pass → mark as IsDeleted

4. Return the new cursor (with the latest edit time of each node this round)
```

> **Note**: The API call volume for incremental sync scales with the size of the knowledge base space's node tree, since the full node tree must be traversed to detect changes. For large spaces, it's recommended to lower the sync frequency accordingly.

#### Supported Document Types

| `obj_type` | Supported | Retrieval Method | Export Format |
|------------|------|---------|---------|
| `docx` | Yes | Export .docx (default) / blocks API converted to Markdown (blocks mode) | `.docx` / Markdown |
| `doc` | Yes | Export task (Export API) | `.docx` |
| `sheet` | Yes | Export task (Export API) | `.xlsx` |
| `bitable` | Yes | Export task (Export API) | `.xlsx` |
| `file` | Yes | Direct download (Drive Download API) | Original format |
| `mindnote` | No | — | — |
| `slides` | No | — | — |

#### Content Retrieval Flow

**docx (new-version cloud documents):** The parsing path is controlled by the `FEISHU_DOCX_PARSE_MODE` environment variable (applies to the app service; takes effect for both the Feishu knowledge base and Feishu Drive connectors).

- **export mode (default, empty or `export`)**: uses the asynchronous export API
  ```
  1. CreateExportTask -> create the export task
  2. Poll GetExportTaskStatus (every 2 seconds, up to ~60 seconds)
  3. DownloadExportFile -> download the .docx binary
  4. Hand the .docx to docreader for parsing (images inline into the parent document, same as a regular docx upload)
  ```
- **blocks mode (`FEISHU_DOCX_PARSE_MODE=blocks`)**: uses the blocks API
  ```
  1. GET /open-apis/docx/v1/documents/{obj_token}/blocks (paginated, 500 per page)
  2. blocksToMarkdown -> convert to Markdown body
     ├─ text/heading/list/table/code block -> Markdown
     ├─ image block -> ![image]() empty placeholder (image downloaded separately as an independent knowledge item)
     └─ file block -> attachment, kept as an independent knowledge item
  3. blocks API fails or renders empty -> fall back to export
  ```

**doc / sheet / bitable:** uses the asynchronous export API (same flow as export mode), exported to .docx / .xlsx and then parsed by docreader.

**file:**

```
DownloadDriveFile -> GET /drive/v1/files/{token}/download
```

##### blocks vs export: Pros and Cons

| | export (default) | blocks |
|---|---|---|
| Parsing path | Export .docx -> docreader | blocks API -> Markdown |
| Image-document association | ✅ Images inline within the parent document, linked via `parent_chunk_id` | ❌ Images become separate knowledge entries, disconnected from the document |
| Retrieval / Wiki / Agent image association | Yes | No |
| Sync speed | Slow (async export + docx parsing) | Fast |
| Attachments within docx (file block) | Lost (not included in .docx export) | Preserved |
| Required permissions | Not required (but recommended to keep for switching) | Requires `docx:document:readonly` |

- **export**: images are inlined into the parent document (same as a regular docx upload), linked to the same knowledge entry as a parent-child relationship via `parent_chunk_id`, allowing image content association in all three scenarios; the tradeoff is slower sync, loss of attachments within the docx, and image OCR/captioning depending on multimodal configuration.
- **blocks**: fast and preserves attachments, but image blocks render as empty placeholders, images are stored as independent knowledge entries, and there's only a weak metadata-level association with the parent document — retrieval, Wiki, and Agent features can't link image content back to the document.

> See section E9 of `.env.example` for environment variable configuration.

#### Source Files

| File | Responsibility |
|------|------|
| `internal/datasource/connector/feishu/core/types.go` | Feishu API type definitions, config structure (Config), Region constants |
| `internal/datasource/connector/feishu/core/client.go` | API client: token management, Wiki/Drive API calls, export/download |
| `internal/datasource/connector/feishu/core/blocks.go` | docx blocks API types (DocxBlock, etc.), listDocumentBlocks |
| `internal/datasource/connector/feishu/core/markdown.go` | blocksToMarkdown: converts block arrays to Markdown |
| `internal/datasource/connector/feishu/core/shared.go` | Shared logic: FetchDocxWithBlocks, ParseFeishuConfig, exportDocxFallback |
| `internal/datasource/connector/feishu/core/engine.go` | Generic sync engine: NodeOps interface, FetchStreamEngine / FetchAllEngine |
| `internal/datasource/connector/feishu/core/region.go` | Region (Feishu / Lark cloud distinction), URL construction |
| `internal/datasource/connector/feishu/wiki/connector.go` | Feishu knowledge base Connector implementation (wikiOps) |
| `internal/datasource/connector/feishu/drive/connector.go` | Feishu Drive Connector implementation (driveOps) |
| `internal/datasource/connector/feishu/{core,wiki,drive}/*_test.go` | Unit tests: using HTTP mocks to simulate the Feishu Open Platform |


## Scheduled Scheduling

### Cron Scheduler

Data source sync scheduling is based on `robfig/cron/v3`, using a **6-field Cron expression (including seconds)**:

```
sec min hour day month weekday
```

**Common Presets:**

| Expression | Description |
|--------|------|
| `0 */30 * * * *` | Every 30 minutes |
| `0 0 * * * *` | Every hour, on the hour |
| `0 0 */6 * * *` | Every 6 hours |
| `0 0 0 * * *` | Daily at midnight |
| `0 0 2 * * *` | Daily at 2 AM |

### Scheduling Lifecycle

```
Service startup
    │
    ▼
Scheduler.Start(ctx)
    │
    ├─ 1. Load all active data sources from DB (FindActive)
    │     Condition: status = active, sync_schedule != '', deleted_at IS NULL
    │
    ├─ 2. Register a Cron Entry for each data source
    │
    └─ 3. Start the Cron Runner

When Cron fires:
    ├─ Check whether a sync is running (HasRunningSync)
    │     Yes → skip this trigger
    │
    ├─ Create SyncLog (status: running)
    │
    └─ Enqueue an asynq task
          Queue: low
          TaskID: dssync:<dsID>:<UTC minute> (dedup)
```

### Task Deduplication (Two-Layer Protection)

1. **DB Layer**: `HasRunningSync` checks whether the data source already has a sync task in progress, preventing overlap when a sync takes longer than the scheduling interval
2. **Queue Layer**: deduplicated based on `TaskID` (format `dssync:<dataSourceID>:<UTC minute>`) — if multiple instances trigger within the same minute, only the first is successfully enqueued, and the rest are marked `canceled`

### Dynamic Management

| Action | Behavior |
|------|------|
| Create a data source (with schedule) | Registers a Cron Entry |
| Update a data source's schedule | Removes the old Entry → registers a new Entry |
| Pause a data source | Removes the Cron Entry |
| Resume a data source | Re-registers the Cron Entry |
| Delete a data source | Removes the Cron Entry |

---

## Key Parameters and Thresholds

| Parameter | Value | Description |
|------|------|------|
| Token cache safety margin | 5 min | Token refreshed ahead of expiry (applies to both Feishu and WeCom) |
| Feishu Wiki Space page size | 50 | Number of items returned per page by ListResources |
| Feishu export task polling interval | 2s | Interval for checking Feishu Export API status |
| Feishu export task max wait | ~60s | Export task timeout |
| WeCom document list page size | 50 | Number of items returned per page by wedoc/doc_list |
| WeCom WeDrive file list page size | 1000 | Max items fetched per call by wedrive/file_list |
| WeCom access_token validity | 7200s | 2 hours; cache refreshed 5 minutes ahead |
| Sync log retention | 30 days | Default value, configurable |
| Sync log default pagination | 10 | Default limit for GetSyncLogs |
| Cron expression format | 6 fields, including seconds | robfig/cron WithSeconds() |
| Scheduled sync queue | `low` | asynq low-priority queue |
| Manual sync queue | `default` | asynq default queue |

---

## Error Handling

| Scenario | Handling Strategy |
|------|---------|
| Credential validation failed | Returns an error message; data source status is marked `error`, `error_message` is recorded |
| Credential validation succeeded (previously in error state) | Status restored to `active`, `error_message` cleared |
| Connector not found | Returns `ErrConnectorNotFound` |
| Knowledge base does not exist | Validated during create/update, returns `ErrKnowledgeBaseNotFound` |
| Data source does not exist | Returns `ErrDataSourceNotFound` |
| Sync triggered while data source is inactive | Manual sync only allowed for `active` or `error` status |
| Sync task enqueue failed | SyncLog marked `failed`, error recorded |
| Scheduled sync task deduplicated | SyncLog marked `canceled`, error message `deduplicated: another instance enqueued first` |
| Single document fetch failed | Error recorded in FetchedItem.Metadata; does not interrupt the overall sync |
| Single document ingestion failed | Counted in `items_failed`; does not interrupt the overall sync |
| Duplicate document (DuplicateKnowledgeError) | Counted in `items_skipped` |
| Unsupported document type | Silently skipped (e.g. Feishu's mindnote, slides) |

---

## Extending with New Connectors

Integrating a new external platform only takes 3 steps:

### 1. Implement the Connector Interface

Create the connector under `internal/datasource/connector/<type>/`:

```go
package myplatform

type Connector struct {
    // Connector configuration
}

func NewConnector() *Connector {
    return &Connector{}
}

func (c *Connector) Type() string {
    return types.ConnectorTypeMyPlatform
}

func (c *Connector) Validate(ctx context.Context, config *types.DataSourceConfig) error {
    // Validate credentials: try to get a token or call the API
}

func (c *Connector) ListResources(ctx context.Context, config *types.DataSourceConfig) ([]types.Resource, error) {
    // List selectable resources (workspace, spaces, folders, etc.)
}

func (c *Connector) FetchAll(ctx context.Context, config *types.DataSourceConfig, resourceIDs []string) ([]types.FetchedItem, error) {
    // Fetch all documents
}

func (c *Connector) FetchIncremental(ctx context.Context, config *types.DataSourceConfig, cursor *types.SyncCursor) ([]types.FetchedItem, *types.SyncCursor, error) {
    // Incrementally fetch changes (can reuse FetchAll logic + edit-time comparison)
}
```

Recommended file structure:

```
internal/datasource/connector/myplatform/
├── types.go        # platform API type definitions
├── client.go       # API client (token management, HTTP calls)
└── connector.go    # Connector interface implementation
```

### 2. Register the Connector

Register it in `initConnectorRegistry` inside `internal/container/container.go`:

```go
func initConnectorRegistry() *datasource.ConnectorRegistry {
    registry := datasource.NewConnectorRegistry()
    registry.Register(feishuConnector.NewConnector())
    registry.Register(myplatform.NewConnector()) // add
    return registry
}
```

Add metadata in `ConnectorMetadataRegistry` inside `internal/datasource/connector.go`:

```go
types.ConnectorTypeMyPlatform: {
    Type:         types.ConnectorTypeMyPlatform,
    Name:         "My Platform",
    Description:  "Import documents from My Platform",
    Icon:         "myplatform",
    Priority:     10,
    AuthType:     "api_key",
    Capabilities: []string{"incremental"},
},
```

Add the type constant in `internal/types/datasource.go`:

```go
const ConnectorTypeMyPlatform = "myplatform"
```

### 3. Add Frontend Configuration

In `frontend/src/views/knowledge/settings/DataSourceEditorDialog.vue`:

- Add the connector type option
- Add credential form fields for the platform
- Mark `available: true`

Add the platform name translation in the i18n files.

> The Service layer (`DataSourceService`) requires no changes — data source management, sync scheduling, content ingestion, and logging are all handled uniformly by the Service. You can refer to the Feishu connector (`internal/datasource/connector/feishu/`) as an implementation reference.
