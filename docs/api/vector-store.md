# Vector Store API

[Back to index](./README.md)

The Vector Store API manages a space's vector database connection configurations, supporting engines such as Elasticsearch, PostgreSQL, Qdrant, Milvus, Weaviate, Tencent VectorDB, and SQLite. The endpoints manage both configurations created by the user in the DB (`source: "user"`) and virtual stores configured via the `RETRIEVE_DRIVER` environment variable (`source: "env"`, read-only).

| Method | Path                       | Description                                        |
| ------ | --------------------------- | --------------------------------------------------- |
| GET    | `/vector-stores/types`      | Get supported engine types and field metadata        |
| POST   | `/vector-stores/test`       | Test a connection using raw credentials (not persisted) |
| POST   | `/vector-stores`            | Create a vector store                                |
| GET    | `/vector-stores`            | Get the list of vector stores                        |
| GET    | `/vector-stores/:id`        | Get vector store details                             |
| PUT    | `/vector-stores/:id`        | Update a vector store (only the name can be changed) |
| DELETE | `/vector-stores/:id`        | Delete a vector store (soft delete)                   |
| POST   | `/vector-stores/:id/test`   | Test connectivity of a saved or environment-variable store |

## GET `/vector-stores/types` - Get supported engine types

Returns the definitions of all supported engine types, along with their connection configuration fields and index configuration fields, which can be used to dynamically generate frontend forms. This is system-level metadata and doesn't require authorization awareness, but it still requires `X-API-Key`.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/vector-stores/types' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": [
        {
            "type": "elasticsearch",
            "display_name": "Elasticsearch (Keywords + Vector)",
            "connection_fields": [
                { "name": "addr", "type": "string", "required": true, "description": "Elasticsearch URL (e.g., http://localhost:9200)" },
                { "name": "username", "type": "string", "required": false },
                { "name": "password", "type": "string", "required": false, "sensitive": true }
            ],
            "index_fields": [
                { "name": "index_name", "type": "string", "required": false, "default": "xwrag_default" },
                { "name": "number_of_shards", "type": "number", "required": false },
                { "name": "number_of_replicas", "type": "number", "required": false }
            ]
        },
        {
            "type": "postgres",
            "display_name": "PostgreSQL (Keywords + Vector)",
            "connection_fields": [
                { "name": "use_default_connection", "type": "boolean", "required": false, "default": true, "description": "Use the application's default database connection" },
                { "name": "addr", "type": "string", "required": false, "description": "PostgreSQL connection string (required if use_default_connection is false)" },
                { "name": "username", "type": "string", "required": false },
                { "name": "password", "type": "string", "required": false, "sensitive": true }
            ]
        }
    ]
}
```

## POST `/vector-stores/test` - Test a connection using raw credentials

Runs a connectivity test using credentials from the frontend form that haven't been saved yet, without writing anything to the database. On success, it returns the automatically detected server version (e.g., the ES version number); some engines (such as Milvus and SQLite) can't detect a version, in which case `version` is returned as an empty string.

**Parameters (request body)**:

| Field              | Type   | Required | Description                                                     |
| ------------------- | ------ | -------- | ----------------------------------------------------------------- |
| engine_type         | string | Yes      | Engine type, taken from the `type` field in `/vector-stores/types` |
| connection_config   | object | Yes      | Connection configuration fields for that engine (corresponding to `connection_fields`) |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/vector-stores/test' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "engine_type": "elasticsearch",
    "connection_config": {
        "addr": "http://es:9200",
        "username": "elastic",
        "password": "changeme"
    }
}'
```

**Response (success)**:

```json
{
    "success": true,
    "version": "7.10.1"
}
```

**Response (failure)**:

```json
{
    "success": false,
    "error": "failed to connect to elasticsearch: connection refused or authentication failed"
}
```

> Note: when the test fails, the HTTP status code is still `200`; the error message is returned via the `success: false` + `error` fields.

## POST `/vector-stores` - Create a vector store

Creates a new vector store configuration for the current space. The same endpoint + index combination cannot be duplicated within a space (it will also conflict with environment-variable-configured stores).

**Parameters (request body)**:

| Field              | Type   | Required | Description                                                       |
| ------------------- | ------ | -------- | -------------------------------------------------------------------- |
| name                | string | Yes      | Display name of the store (friendly name within the space)            |
| engine_type         | string | Yes      | Engine type, taken from `/vector-stores/types`                        |
| connection_config   | object | Yes      | Connection configuration (corresponding to the selected engine's `connection_fields`) |
| index_config        | object | No       | Index configuration (corresponding to the selected engine's `index_fields`) |

> Tencent VectorDB uses `engine_type: "tencent_vectordb"`. In `connection_config`, `addr`, `username`, and `api_key` are required, while `database` is optional; `index_config.collection_name` is the collection name prefix — the actual collection will have a suffix appended based on the vector dimension (e.g., `weknora_embeddings_768`); `index_config.replica_number` is the number of replicas used when creating the collection. This adapter supports both vector retrieval and keyword retrieval based on BM25 sparse vectors; collections created in older versions without a `sparse_vector` index need to be rebuilt and have data re-imported before keyword retrieval can be enabled.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/vector-stores' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "elasticsearch-hot",
    "engine_type": "elasticsearch",
    "connection_config": {
        "addr": "http://es-hot:9200",
        "username": "elastic",
        "password": "changeme"
    },
    "index_config": {
        "index_name": "my_index"
    }
}'
```

**Tencent VectorDB request example**:

```curl
curl --location 'http://localhost:8080/api/v1/vector-stores' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "tencent-vectordb",
    "engine_type": "tencent_vectordb",
    "connection_config": {
        "addr": "http://your-instance.tencentvectordb.com",
        "username": "root",
        "api_key": "your_api_key",
        "database": "weknora"
    },
    "index_config": {
        "collection_name": "weknora_embeddings",
        "replica_number": 1
    }
}'
```

**Response** (201):

```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "elasticsearch-hot",
        "engine_type": "elasticsearch",
        "connection_config": {
            "addr": "http://es-hot:9200",
            "username": "elastic",
            "password": "***"
        },
        "index_config": {
            "index_name": "my_index"
        },
        "source": "user",
        "readonly": false,
        "created_at": "2026-04-07T10:00:00Z",
        "updated_at": "2026-04-07T10:00:00Z"
    }
}
```

> Sensitive fields in the response (`password`, `api_key`, etc.) are masked as `"***"`. The `connection_config.version` field is only populated automatically after a connection test succeeds; it is empty on creation.

## GET `/vector-stores` - Get the list of vector stores

Returns all vector stores for the current space, including virtual stores configured via the `RETRIEVE_DRIVER` environment variable (`source: "env"`, `readonly: true`) and stores created by the user in the DB (`source: "user"`, `readonly: false`). Environment-variable stores are listed first.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/vector-stores' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": [
        {
            "id": "__env_postgres__",
            "name": "postgres (env)",
            "engine_type": "postgres",
            "connection_config": {
                "use_default_connection": true
            },
            "source": "env",
            "readonly": true
        },
        {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "name": "elasticsearch-hot",
            "engine_type": "elasticsearch",
            "connection_config": {
                "addr": "http://es-hot:9200",
                "username": "elastic",
                "password": "***"
            },
            "source": "user",
            "readonly": false
        }
    ]
}
```

## GET `/vector-stores/:id` - Get vector store details

Gets a single vector store by ID. Supports both DB store UUIDs and environment-variable store IDs in the `__env_*` format (e.g., `__env_postgres__`).

**Path parameters**:

| Field | Type   | Required | Description                                            |
| ----- | ------ | -------- | -------------------------------------------------------- |
| id    | string | Yes      | Vector store ID (DB UUID or `__env_{driver}__`)          |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/vector-stores/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "elasticsearch-hot",
        "engine_type": "elasticsearch",
        "connection_config": {
            "addr": "http://es-hot:9200",
            "username": "elastic",
            "password": "***",
            "version": "7.10.1"
        },
        "index_config": {
            "index_name": "my_index"
        },
        "source": "user",
        "readonly": false,
        "created_at": "2026-04-07T10:00:00Z",
        "updated_at": "2026-04-07T10:00:00Z"
    }
}
```

## PUT `/vector-stores/:id` - Update a vector store

Only supports updating `name`. `engine_type`, `connection_config`, and `index_config` cannot be changed after creation; environment-variable stores cannot be modified (returns `400`).

**Path parameters**:

| Field | Type   | Required | Description       |
| ----- | ------ | -------- | -------------------- |
| id    | string | Yes      | Vector store ID    |

**Parameters (request body)**:

| Field | Type   | Required | Description              |
| ----- | ------ | -------- | -------------------------- |
| name  | string | Yes      | New display name for the store |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/vector-stores/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "elasticsearch-hot-renamed"
}'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "elasticsearch-hot-renamed",
        "engine_type": "elasticsearch",
        "connection_config": {
            "addr": "http://es-hot:9200",
            "username": "elastic",
            "password": "***"
        },
        "index_config": {
            "index_name": "my_index"
        },
        "source": "user",
        "readonly": false,
        "created_at": "2026-04-07T10:00:00Z",
        "updated_at": "2026-04-07T10:05:00Z"
    }
}
```

## DELETE `/vector-stores/:id` - Delete a vector store

Performs a soft delete on a store in the DB. Environment-variable stores cannot be deleted (returns `400`).

**Phase 2 — Binding protection**:

The delete request runs within a transaction, and counts the number of active knowledge bases in the current space that are still bound to that store, using the `(tenant_id, vector_store_id)` composite index. **As long as any bound knowledge base exists (soft-deleted KBs are not counted), the deletion is rejected**, and the caller must first unbind or delete those knowledge bases before proceeding. On PostgreSQL, a `SELECT … FOR UPDATE` row lock is placed on the `vector_stores` row during the transaction, preventing concurrent knowledge base creation requests from silently landing on a store that is being deleted (on SQLite, the same semantics are achieved via WAL + single-writer serialization).

**Path parameters**:

| Field | Type   | Required | Description       |
| ----- | ------ | -------- | -------------------- |
| id    | string | Yes      | Vector store ID    |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/vector-stores/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx'
```

**Response (success)**:

```json
{
    "success": true
}
```

**Response (binding protection triggered)**:

```json
{
    "success": false,
    "error": {
        "code": 1000,
        "message": "vector store still has 3 knowledge base(s) bound to it; unbind or delete them before removing the store"
    }
}
```

HTTP `400`. The error message includes the specific number of knowledge bases (to help operations pinpoint the issue), but does not include any KB IDs/names, in order to avoid cross-space information leakage. When the deletion is rejected, the store row in the DB remains unchanged, and the in-process engine registry is not cleared either.

## POST `/vector-stores/:id/test` - Test the connection of a saved or environment-variable store

Runs a connection test against a saved DB store or an environment-variable virtual store. On success, returns the detected server version; for DB stores, the detected version is automatically written back to `connection_config.version`, while environment-variable stores are not updated.

**Path parameters**:

| Field | Type   | Required | Description                                            |
| ----- | ------ | -------- | -------------------------------------------------------- |
| id    | string | Yes      | Vector store ID (DB UUID or `__env_{driver}__`)          |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/vector-stores/550e8400-e29b-41d4-a716-446655440000/test' \
--header 'X-API-Key: sk-xxxxx'
```

**Response (success)**:

```json
{
    "success": true,
    "version": "7.10.1"
}
```

**Response (failure)**:

```json
{
    "success": false,
    "error": "failed to connect to elasticsearch: connection refused or authentication failed"
}
```

> Consistent with `/vector-stores/test`: when the test fails, the HTTP status code is still `200`, and the error is returned via `success: false` + `error`.

## Environment-variable stores

Vector stores configured via the `RETRIEVE_DRIVER` environment variable appear as virtual entries in the list and detail views. Characteristics of these entries:

- **ID format**: `__env_{driver}__` (e.g., `__env_postgres__`, `__env_elasticsearch_v8__`)
- **source**: `"env"`
- **readonly**: `true`
- **Cannot be modified/deleted**: `PUT` and `DELETE` return `400`
- **Connectivity can be tested**: `POST /vector-stores/:id/test` works normally
- **When bound to a knowledge base**: a knowledge base created without specifying `vector_store_id` uses the environment-variable store by default; such a knowledge base is displayed in responses as `vector_store_name="System default"` + `vector_store_source="env"`.

For Tencent VectorDB, the default collection replica count can be overridden via the `TENCENT_VECTORDB_REPLICA_NUMBER` environment variable. The default value is `1`; it can be set to `0` for single-node QA environments, and adjusted for production based on the scale of the Tencent VectorDB cluster.

## Error codes

| HTTP Status Code | code | Meaning                                                              |
| ------------------ | ---- | ----------------------------------------------------------------------- |
| 400                 | 1000 | Invalid request parameters, validation failure, attempt to modify an environment-variable store, or knowledge bases still bound at deletion time |
| 400                 | 2200 | The `vector_store_id` referenced during knowledge base creation is invalid (doesn't exist or belongs to another space) |
| 400                 | 2201 | The store referenced during knowledge base creation is currently unavailable (exists in the DB but not registered with the engine) |
| 401                 | 1001 | Not authenticated (missing space context or API Key)                     |
| 404                 | 1003 | Vector store does not exist                                              |
| 409                 | 1005 | The same endpoint + index combination already exists                     |
| 500                 | 1007 | Internal server error                                                     |

> `2200` / `2201` are returned by knowledge base creation paths such as `POST /knowledge-bases` (see [knowledge-base.md](./knowledge-base.md) for details); they are listed here only for complete coverage of all error codes related to vector stores.
