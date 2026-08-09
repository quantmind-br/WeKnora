---
title: Data Source Import Development
tags: [Integration & Extensions, Data Source, Feishu, Sync, Connector]
aliases: [Data Source Import, DataSource, Data Sync]
source: Data-Source-Import-Development.md
---

# Data Source Import Development

WeKnora's data source import module supports automatically importing and syncing content from external platforms (Feishu, WeCom, Notion, Confluence, etc.) into the knowledge base. Users can configure data source connections, select the resources to sync, and complete incremental/full content sync automatically via manual trigger or scheduled jobs.

Data sources are bound to a knowledge base, and one knowledge base can connect to multiple data sources. Credentials are stored encrypted using AES-256-GCM.

> Data source import and [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) both involve Feishu integration, so consider sharing Feishu app credentials

## Currently Supported Connectors

| Connector | Auth Method | Incremental Sync | Deletion Sync |
|--------|---------|:-:|:-:|
| Feishu | OAuth2 (Tenant Access Token) | ✅ | ✅ |

## Quick Start: Feishu Knowledge Base

1. Create a Feishu app and obtain the App ID / App Secret
2. Enable permissions: `wiki:wiki:readonly`, `drive:drive:readonly`, `drive:export:readonly`, `docx:document:readonly`
3. Add a data source on the knowledge base settings page, select Feishu, and fill in the credentials
4. Test the connection, select the knowledge base space, and configure the sync strategy
5. Trigger an initial sync to verify

> Note: Feishu's international version (Lark) is also supported, automatically adapting to the `https://open.larksuite.com` API endpoint

## Feishu docx Parsing Modes and Environment Variables

The parsing path for Feishu's new-generation cloud documents (docx) is controlled by the environment variable `FEISHU_DOCX_PARSE_MODE` (applies to the app service, and takes effect for both the Feishu Knowledge Base and Feishu Drive connectors simultaneously):

| Mode | Value | Parsing Path | Image-Document Association | Speed | Attachments Within docx |
|---|---|---|---|---|---|
| export (default) | empty / `export` | Async export -> .docx binary -> docreader parsing | ✅ Images inlined into the parent document, associated via `parent_chunk_id` | Slow | Lost |
| blocks | `blocks` | blocks API -> Markdown | ❌ Images become standalone knowledge entries, disconnected from the document | Fast | Preserved |

**Pros and Cons Comparison:**

- **export**: Exports .docx for docreader to parse, with images inlined into the parent document (consistent with a regular docx upload), establishing a parent-child association within the same knowledge entry via `parent_chunk_id`, so image content can be associated in all three scenarios; the trade-off is slower sync (async export + docx parsing), loss of attachments within the docx, and image OCR/captioning depends on multimodal configuration.
- **blocks**: Image blocks render as empty `![Image]()` placeholders, with images downloaded separately as standalone knowledge entries — retrieval / Wiki / agents cannot associate the image content back to the document; however, sync is fast and file block attachments within the docx are preserved.

Configuration: export is the default, so no setting is needed; to use blocks mode, set `FEISHU_DOCX_PARSE_MODE=blocks` in the app service environment variables in `.env` or `docker-compose.yml`, then restart the app service for it to take effect. See [Feishu Drive Data Source Integration Guide](Feishu-Drive-DataSource-Integration.md#6-docx-parsing-modes-and-environment-variables) for details.

## Architecture Design

```
External Platform API → Connector → ConnectorRegistry → DataSourceService → WeKnora Core (Knowledge Ingestion Pipeline)
```

Core design patterns:

| Pattern | Purpose |
|------|------|
| Adapter Pattern | Unifies differences across platforms |
| Registry Pattern | Dynamically looks up connectors by type |
| Strategy Pattern | SyncMode (incremental/full), ConflictStrategy (overwrite/skip) |
| Cursor-based Pagination | Incremental sync tracks changes based on SyncCursor |

## Key Concepts

- **DataSource** — the binding between an external platform connection and a knowledge base
- **Connector** — the adaptation layer that interacts with an external platform
- **SyncCursor** — a state marker for incremental sync
- **FetchedItem** — a fetched document item

## API Endpoints

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/datasource/types` | Get available connector types |
| POST | `/api/v1/datasource/validate-credentials` | Validate credentials |
| POST | `/api/v1/datasource` | Create a data source |
| GET | `/api/v1/datasource?kb_id=xxx` | List data sources |
| POST | `/api/v1/datasource/:id/sync` | Manually trigger sync |
| POST | `/api/v1/datasource/:id/pause` | Pause |
| POST | `/api/v1/datasource/:id/resume` | Resume |

## Extending with New Connectors

1. Implement the `Connector` interface (Validate, ListResources, FetchAll, FetchIncremental)
2. Register the connector with `ConnectorRegistry`
3. Add frontend configuration

> The extension development pattern is similar to [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md) and [Integrating a Vector Database](../Integration-Extension/Integrating-a-Vector-Database.md)

## Related Topics

- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — Feishu integration within IM channels
- [Shared Space Guide](../Security-Authentication/Shared-Spaces-Guide.md) — Knowledge base sharing mechanism
- [FAQ](../Operations-Troubleshooting/FAQ.md) — Troubleshooting data source sync issues

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — Also involves Feishu integration, can share app credentials
- [Shared Space Guide](../Security-Authentication/Shared-Spaces-Guide.md) — Knowledge base sharing complements data source import
- [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md) — Similar extension development pattern
