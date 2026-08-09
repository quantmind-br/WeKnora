---
title: Integrating a Vector Database
tags: [Integration Extensions, Vector Database, Retrieval Engine, Extensions]
aliases: [Vector Database, VecDB, Vector Retrieval, RetrieveEngine]
source: Using Other Vector Databases.md
---

# Integrating a New Vector Database

This document provides a complete guide for adding support for a new vector database to the WeKnora project. By implementing standardized interfaces and following a structured process, developers can efficiently integrate a custom vector database.

> For a similar extension development pattern, see [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md)

## Integration Process

### 1. Implement the Base Retrieval Engine Interface

```go
type RetrieveEngine interface {
    EngineType() types.RetrieverEngineType
    Retrieve(ctx context.Context, params types.RetrieveParams) ([]*types.RetrieveResult, error)
    Support() []types.RetrieverType
}
```

### 2. Implement the Storage Layer Interface

Implement the `RetrieveEngineRepository` interface, extending the base retrieval engine capabilities and adding index management functionality:

- `Save` / `BatchSave` — save the index
- `EstimateStorageSize` — estimate storage size
- `DeleteByChunkIDList` / `DeleteByKnowledgeIDList` — delete the index
- `CopyIndices` — copy indices

### 3. Implement the Service Layer Interface

Create a `RetrieveEngineService` implementation responsible for the business logic of index creation and management.

### 4. Add Environment Variable Configuration

```
RETRIEVE_DRIVER=postgres,elasticsearch_v8,your_database
YOUR_DATABASE_ADDR=host:port
YOUR_DATABASE_USERNAME=username
YOUR_DATABASE_PASSWORD=password
```

### 5. Register the Retrieval Engine

Add the initialization and registration logic in `initRetrieveEngineRegistry` in `internal/container/container.go`.

### 6. Define Retrieval Engine Type Constants

Add the new engine type constant in `internal/types/retriever.go`.

## Reference Implementations

It is recommended to reference the existing implementations:

- PostgreSQL: `internal/application/repository/retriever/postgres/`
- ElasticsearchV7: `internal/application/repository/retriever/elasticsearch/v7/`
- ElasticsearchV8: `internal/application/repository/retriever/elasticsearch/v8/`
- Apache Doris 4.1: `internal/application/repository/retriever/doris/`
- Tencent VectorDB: `internal/application/repository/retriever/tencentvectordb/`

## Apache Doris 4.1 Integration Notes

Doris is an MPP-style analytical SQL database. Its integration approach differs from NoSQL vector databases in several respects:

- **Protocol**: the MySQL protocol (FE 9030) handles the main data path; the HTTP API (FE 8030) handles Stream Load partial updates.
- **Table structure**: one `<DORIS_TABLE_PREFIX>_<dim>` table per dimension, with `UNIQUE KEY(id)` + `enable_unique_key_merge_on_write=true`.
- **Indexes**: an INVERTED index covers the filter fields, with the `content` field using the `chinese` parser; the ANN index uses HNSW + cosine_distance (built asynchronously — after table creation, it automatically polls every 30s waiting for `FINISHED`).
- **Score semantics**: uses `1 - cosine_distance_approximate(...)`, with threshold comparison via `>=` and `ORDER BY DESC`, matching the direction of Qdrant cosine similarity.
- **Keyword retrieval**: based on `MATCH_ANY` against the Doris inverted index, requiring no jieba tokenization on the Go side.
- **Bulk field updates**: `BatchUpdateChunkEnabledStatus` and `BatchUpdateChunkTagID` are implemented via the Stream Load partial update protocol, supporting automatic 1MiB batching.

Environment variables:

```
RETRIEVE_DRIVER=doris
DORIS_ADDR=doris-fe:9030
DORIS_HTTP_PORT=8030
DORIS_DATABASE=weknora
DORIS_USERNAME=root
DORIS_PASSWORD=
DORIS_TABLE_PREFIX=weknora_embeddings
```

How to start: `docker compose --profile doris up -d`, then run `CREATE DATABASE weknora;` on the FE.

## Tencent VectorDB

WeKnora includes a built-in Tencent VectorDB adapter, with the driver name `tencent_vectordb`. This adapter supports vector retrieval, keyword retrieval based on BM25 sparse vectors, and index management, and can participate in WeKnora's upper-layer hybrid retrieval.

```env
RETRIEVE_DRIVER=tencent_vectordb
TENCENT_VECTORDB_ADDR=http://your-instance.tencentvectordb.com
TENCENT_VECTORDB_USERNAME=root
TENCENT_VECTORDB_API_KEY=your_tencent_vectordb_api_key
TENCENT_VECTORDB_DATABASE=weknora
TENCENT_VECTORDB_COLLECTION=weknora_embeddings
TENCENT_VECTORDB_REPLICA_NUMBER=1
```

`TENCENT_VECTORDB_COLLECTION` is the collection name prefix. WeKnora creates actual collections based on vector dimension, for example `weknora_embeddings_768`.
`TENCENT_VECTORDB_REPLICA_NUMBER` is the number of replicas used when creating a collection, defaulting to `1`; it can be set to `0` for single-node QA environments, and adjusted for production environments according to the scale of the Tencent VectorDB cluster.

Keyword retrieval depends on the Tencent VectorDB sparse vector index. Newly created collections automatically create a `sparse_vector` index; for vector collections created in older versions that lack this index, the collection must be rebuilt and the knowledge base data re-imported before keyword retrieval can be enabled.

## Related Topics

- [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md) — a similar extension development pattern (interface implementation + registration + DI)
- [Knowledge Graph](../Core-Features/Knowledge-Graph.md) — the knowledge graph feature depends on Neo4j rather than a vector database
- [FAQ](../Operations-Troubleshooting/FAQ.md) — Embedding model configuration and vector dimension related topics

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md) — a similar extension development pattern
- [Knowledge Graph](../Core-Features/Knowledge-Graph.md) — another knowledge retrieval approach (graph-based rather than vector-based)
- [Roadmap](../Project-Overview/Version-Roadmap.md) — retrieval capability extension directions in the roadmap
