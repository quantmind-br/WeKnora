# API Reference: Infrastructure & Data Sources

Registers and manages vector stores, file storage, web search services, and data sources, and provides connection testing and sync operations.

Common conventions: read access requires Viewer+, write/connection-test access requires Admin+ (since credential probing reaches external systems). API key capabilities: vector stores `manage_vector_stores`, storage backends `manage_storage_backends`, web search `manage_web_search`, data sources `manage_datasources` (all support full-access).

## Vector Stores (/api/v1/vector-stores)

### GET /api/v1/vector-stores/types

Purpose: available engine types and their config schema. Permission: Viewer+.

Response: 200 `{"success":true,"data":[type definitions]}`

```bash
curl $BASE/api/v1/vector-stores/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/vector-stores/test

Purpose: test a connection using a raw configuration (not persisted). Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `engine_type` | string | Yes (`binding:"required"`) | Engine type |
| `connection_config` | object | Yes (`binding:"required"`) | Connection configuration |

Response: 200 `{"success":true|false,"version":"...","error":"..."}`

```bash
curl -X POST $BASE/api/v1/vector-stores/test -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"engine_type":"qdrant","connection_config":{"addr":"qdrant:6334"}}'
```

### POST /api/v1/vector-stores

Purpose: create a vector store configuration. Permission: Admin+. Fields: `name` (required), `engine_type` (required), `connection_config` (required), `index_config` (optional).

Response: 201 `{"success":true,"data":{VectorStoreResponse}}` (`id,tenant_id,name,engine_type,connection_config,index_config,...`)

```bash
curl -X POST $BASE/api/v1/vector-stores -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"qdrant-main","engine_type":"qdrant","connection_config":{"addr":"qdrant:6334"}}'
```

### GET /api/v1/vector-stores

Purpose: list vector stores (env-injected `__env_*` stores appear first). Permission: Viewer+.

Response: 200 `{"success":true,"data":[VectorStoreResponse]}`

```bash
curl $BASE/api/v1/vector-stores -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/vector-stores/:id

Purpose: vector store details (supports `__env_*` IDs). Permission: Viewer+.

Response: 200 `{"success":true,"data":{VectorStoreResponse}}`

```bash
curl $BASE/api/v1/vector-stores/vs-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/vector-stores/:id

Purpose: update (rename only; env stores cannot be modified). Permission: Admin+. Request body: `{"name":"..."}` (`binding:"required"`).

Response: 200 `{"success":true,"data":{VectorStoreResponse}}`

```bash
curl -X PUT $BASE/api/v1/vector-stores/vs-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"qdrant-prod"}'
```

### DELETE /api/v1/vector-stores/:id

Purpose: delete (env stores cannot be deleted). Permission: Admin+.

Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/vector-stores/vs-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/vector-stores/:id/test

Purpose: test a saved/env vector store. Permission: Admin+.

Response: 200 `{"success":true|false,"version","error"}`

```bash
curl -X POST $BASE/api/v1/vector-stores/vs-1/test -H "Authorization: Bearer $TOKEN"
```

## Storage Backends (/api/v1/storage-backends)

Request body (shared by Create/Update/TestRaw as `storageBackendRequest`):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required"`) | Name |
| `provider` | string | Yes (`binding:"required"`) | Provider: `local`/`minio`/`cos`/`tos`/`s3`/`oss`/`ks3`/`obs`, restricted by `STORAGE_ALLOW_LIST` |
| `config` | object | No | Provider configuration; see [Storage Backends](../03-features/19-storage-backends.md#connection-parameters) for the fields (credentials masked in responses) |
| `status` | string | No | `active` (default)/`disabled` |

### GET /api/v1/storage-backends/types

Purpose: list of provider names allowed by `STORAGE_ALLOW_LIST` (all providers when unset). Permission: Viewer+. Response: 200 `{"success":true,"data":["local","minio",...]}`

```bash
curl $BASE/api/v1/storage-backends/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/storage-backends/test

Purpose: connection test using a raw configuration. Permission: Admin+. Response: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/storage-backends/test -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"t","provider":"minio","config":{"endpoint":"minio:9000"}}'
```

### POST /api/v1/storage-backends

Purpose: create a storage backend. Permission: Admin+. Response: 201 `{"success":true,"data":{StorageBackend}}`

```bash
curl -X POST $BASE/api/v1/storage-backends -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"minio-main","provider":"minio","config":{"endpoint":"minio:9000"}}'
```

### GET /api/v1/storage-backends

Purpose: list (includes `default_storage_backend_id`). Permission: Viewer+. Response: 200 `{"success":true,"data":[...],"default_storage_backend_id":"..."}`

```bash
curl $BASE/api/v1/storage-backends -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/storage-backends/:id

Purpose: details (credentials masked). Permission: Viewer+. Response: 200 `{"success":true,"data":{StorageBackend}}`

```bash
curl $BASE/api/v1/storage-backends/sb-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/storage-backends/:id

Purpose: update. Permission: Admin+. Response: 200 `{"success":true,"data":{StorageBackend}}`

```bash
curl -X PUT $BASE/api/v1/storage-backends/sb-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"minio-prod","provider":"minio"}'
```

### DELETE /api/v1/storage-backends/:id

Purpose: delete. Permission: Admin+. Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/storage-backends/sb-1 -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/storage-backends/:id/test

Purpose: test a saved backend. Permission: Admin+. Response: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/storage-backends/sb-1/test -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/storage-backends/:id/default

Purpose: set as the default backend. Permission: Admin+. Response: 200 `{"success":true}`

```bash
curl -X PUT $BASE/api/v1/storage-backends/sb-1/default -H "Authorization: Bearer $TOKEN"
```

## Web Search (/api/v1/web-search and /api/v1/web-search-providers)

当前注册 14 个搜索提供商，包括 Metaso、Exa、Bocha、Brave、Serply。各自的 api_key 与 extra_config 参数见[联网搜索](../03-features/11-web-search.md)。

### GET /api/v1/web-search/providers

Purpose: catalog of built-in search providers (read-only). Permission: Viewer+, JWT only (no API key policy declared). Handler: `internal/handler/web_search.go`

Response: 200 `{"success":true,"data":[...]}`

```bash
curl $BASE/api/v1/web-search/providers -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/web-search-providers/types

Purpose: provider types and their parameter schema. Permission: Viewer+. Handler: `internal/handler/web_search_provider.go`

Response: 200 `{"success":true,"data":[...]}`

```bash
curl $BASE/api/v1/web-search-providers/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/web-search-providers/test

Purpose: test raw credentials (not persisted). Permission: Admin+. Request body: `provider` (`binding:"required"`), `parameters` (optional).

Response: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/web-search-providers/test -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"provider":"tavily","parameters":{"api_key":"tvly-..."}}'
```

### POST /api/v1/web-search-providers

Purpose: create a provider configuration. Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes (`binding:"required"`) | Name |
| `provider` | string | Yes (`binding:"required"`) | Type (bing/tavily/google…) |
| `description` | string | No | Description |
| `parameters` | object | No | Parameters (it's recommended to set api_key via the credentials sub-resource) |
| `is_default` | bool | No | Default provider |

Response: 201 `{"success":true,"data":{WebSearchProviderResponse}}`

```bash
curl -X POST $BASE/api/v1/web-search-providers -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"tavily-main","provider":"tavily"}'
```

### GET /api/v1/web-search-providers

Purpose: list providers. Permission: Viewer+. Response: 200 `{"success":true,"data":[...]}`

```bash
curl $BASE/api/v1/web-search-providers -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/web-search-providers/:id

Purpose: details. Permission: Viewer+. Response: 200 `{"success":true,"data":{...}}`

```bash
curl $BASE/api/v1/web-search-providers/wsp-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/web-search-providers/:id

Purpose: update (empty fields keep their original value; APIKey is preserved). Permission: Admin+. Request body: `name/description/parameters/is_default` (all optional).

Response: 200 `{"success":true,"data":{...}}`

```bash
curl -X PUT $BASE/api/v1/web-search-providers/wsp-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"is_default":true}'
```

### DELETE /api/v1/web-search-providers/:id

Purpose: delete. Permission: Admin+. Response: 200 `{"success":true}`

```bash
curl -X DELETE $BASE/api/v1/web-search-providers/wsp-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/web-search-providers/:id/credentials

Purpose: set the API key (`{"api_key":"..."}`; when omitted, returns the current status). Permission: Admin+. Handler: `internal/handler/web_search_provider_credentials.go`

Response: 200 `{"success":true,"data":{"fields":{"api_key":{"configured":bool}}}}`

```bash
curl -X PUT $BASE/api/v1/web-search-providers/wsp-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"api_key":"tvly-..."}'
```

### DELETE /api/v1/web-search-providers/:id/credentials/:field

Purpose: delete a credential field (`field` can only be `api_key`). Permission: Admin+. Response: 204.

```bash
curl -X DELETE $BASE/api/v1/web-search-providers/wsp-1/credentials/api_key -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/web-search-providers/:id/test

Purpose: test a saved provider. Permission: Admin+. Response: 200 `{"success":bool,"error"}`

```bash
curl -X POST $BASE/api/v1/web-search-providers/wsp-1/test -H "Authorization: Bearer $TOKEN"
```

## Data Sources (/api/v1/datasource)

External content connectors (Feishu/Notion/Yuque, etc.); sync jobs write into a KB. Handler: `internal/handler/datasource.go`. Most responses in this group are raw objects/arrays (no `success` wrapper).

当前已注册类型为 feishu、lark、feishu_drive、lark_drive、notion、confluence、yuque、dingtalk、ima、rss、gitlab。各连接器的 credentials、资源选择与同步限制见[数据源导入](../03-features/10-datasource.md)。sync_deletions 开启后会真实删除该数据源归属下的已删除知识；source_created_at/source_updated_at 保存在知识 metadata 中。

### GET /api/v1/datasource/types

Purpose: catalog of available connectors. Permission: Viewer+.

Response: 200 `[{type,name,description,icon,priority,auth_type,capabilities}]`

```bash
curl $BASE/api/v1/datasource/types -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/validate-credentials

Purpose: validate raw credentials (the "test connection" button; not persisted). Permission: Admin+.

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `type` | string | Yes (`binding:"required"`) | Connector type |
| `credentials` | map | Yes (`binding:"required"`) | Credentials |

Response: 200 `{"status":"connected"}`; on failure 400 `{"error":"..."}`

```bash
curl -X POST $BASE/api/v1/datasource/validate-credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"type":"notion","credentials":{"api_key":"ntn_xxx"}}'
```

### POST /api/v1/datasource

Purpose: create a data source. Permission: Admin+. Request body (`types.DataSource`):

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `knowledge_base_id` | string | Yes | Target KB (must belong to this space) |
| `name` | string | Yes | Name |
| `type` | string | Yes | Connector type |
| `config` | object | Yes | Credentials (stored encrypted) + resource selection + settings |
| `sync_schedule` | string | No | Cron expression |
| `sync_mode` | string | No | `incremental` (default) / `full` |
| `conflict_strategy` | string | No | `overwrite` (default) / `skip` |
| `sync_deletions` | bool | No | Defaults to true |
| `sync_log_retention_days` | int | No | Defaults to 30 |

Response: 201 `DataSourceResponse` (credentials stripped, see `internal/handler/dto/datasource.go`).

```bash
curl -X POST $BASE/api/v1/datasource -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{"knowledge_base_id":"kb-1","name":"notion sync","type":"notion","config":{}}'
```

### GET /api/v1/datasource

Purpose: list data sources. Permission: Viewer+. Query parameters: `kb_id` (required).

Response: 200 `[DataSourceResponse]`

```bash
curl "$BASE/api/v1/datasource?kb_id=kb-1" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/:id

Purpose: details. Permission: Viewer+. Response: 200 `DataSourceResponse`; 404 `{"error":"data source not found"}`

```bash
curl $BASE/api/v1/datasource/ds-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/datasource/:id

Purpose: update (`id/tenant_id/knowledge_base_id` are locked to their original values). Permission: Admin+. Request body same as create.

Response: 200 `DataSourceResponse`

```bash
curl -X PUT $BASE/api/v1/datasource/ds-1 -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"name":"notion sync v2","type":"notion","knowledge_base_id":"kb-1","config":{}}'
```

### DELETE /api/v1/datasource/:id

Purpose: delete. Permission: Admin+. Response: 204.

```bash
curl -X DELETE $BASE/api/v1/datasource/ds-1 -H "Authorization: Bearer $TOKEN"
```

### PUT /api/v1/datasource/:id/credentials

Purpose: replace credentials wholesale (data source credentials are an atomic map under a single logical field, `credentials`). Permission: Admin+. Request body: `{"credentials":{...}}` (a non-empty map is required). Handler: `internal/handler/datasource_credentials.go`

Response: 200 `{"success":true,"data":{"fields":{"credentials":{"configured":bool}}}}`

```bash
curl -X PUT $BASE/api/v1/datasource/ds-1/credentials -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"credentials":{"api_key":"ntn_xxx"}}'
```

### DELETE /api/v1/datasource/:id/credentials/:field

Purpose: clear credentials (`field` must be `credentials`). Permission: Admin+. Response: 204.

```bash
curl -X DELETE $BASE/api/v1/datasource/ds-1/credentials/credentials -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/:id/validate

Purpose: validate a saved data source's connection. Permission: Admin+. Response: 200 `{"status":"connected"}`

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/validate -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/:id/resources

Purpose: browse the external resource tree (lazy-loaded). Permission: Admin+. Query parameters: `parent_id` (optional; empty = top level).

Response: 200 `[{external_id,name,type,description,url,modified_at,parent_id,has_children,metadata}]`

```bash
curl "$BASE/api/v1/datasource/ds-1/resources?parent_id=" -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/:id/resource-ancestors

Purpose: resolve a resource's ancestor chain (for selector expansion). Permission: Admin+. Request body: `{"resource_ids":["..."]}` (required).

Response: 200 `{"ancestors":[...]}`

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/resource-ancestors -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d '{"resource_ids":["page-1"]}'
```

### POST /api/v1/datasource/:id/sync

Purpose: manually trigger a sync. Permission: Admin+. Response: 200 `SyncLog` (`id,status,started_at,items_total,items_created,items_updated,items_deleted,items_failed,...`)

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/sync -H "Authorization: Bearer $TOKEN"
```

### POST /api/v1/datasource/:id/pause and POST /api/v1/datasource/:id/resume

Purpose: pause / resume scheduled sync. Permission: Admin+.

Response: 200 `{"status":"paused"}` / `{"status":"active"}`

```bash
curl -X POST $BASE/api/v1/datasource/ds-1/pause -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/:id/logs

Purpose: list sync logs. Permission: Viewer+. Query parameters: `limit` (default 10, max 100), `offset` (default 0).

Response: 200 `[SyncLog]`

```bash
curl "$BASE/api/v1/datasource/ds-1/logs?limit=10" -H "Authorization: Bearer $TOKEN"
```

### GET /api/v1/datasource/logs/:log_id

Purpose: a single sync log entry. Permission: Viewer+. Response: 200 `SyncLog`; 404 `{"error":"sync log not found"}`

```bash
curl $BASE/api/v1/datasource/logs/log-1 -H "Authorization: Bearer $TOKEN"
```

## 实现参考

路由注册：`internal/router/routes_infra.go` 的 `RegisterVectorStoreRoutes`、`RegisterStorageBackendRoutes`、`RegisterWebSearchRoutes`、`RegisterWebSearchProviderRoutes`、`RegisterDataSourceRoutes`。Handler：`internal/handler/vectorstore.go`、`internal/handler/storagebackend.go`、`internal/handler/web_search.go`、`internal/handler/web_search_provider.go`、`internal/handler/web_search_provider_credentials.go`、`internal/handler/datasource.go`、`internal/handler/datasource_credentials.go`。
