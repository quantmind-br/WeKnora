### How to Integrate a New Vector Database

This document provides a complete guide for adding new vector database support to the WeKnora project. By implementing standardized interfaces and following a structured process, developers can efficiently integrate a custom vector database.

### Integration Process

#### 1. Implement the base retrieval engine interface

First, you need to implement the `RetrieveEngine` interface in the `interfaces` package, which defines the core capabilities of the retrieval engine:

```go
type RetrieveEngine interface {
    // Returns the type identifier of the retrieval engine
    EngineType() types.RetrieverEngineType

    // Executes the retrieval operation and returns matching results
    Retrieve(ctx context.Context, params types.RetrieveParams) ([]*types.RetrieveResult, error)

    // Returns the list of retrieval types supported by this engine
    Support() []types.RetrieverType
}
```

#### 2. Implement the storage layer interface

Implement the `RetrieveEngineRepository` interface, which extends the base retrieval engine capabilities and adds index management functionality:

```go
type RetrieveEngineRepository interface {
    // Saves a single index entry
    Save(ctx context.Context, indexInfo *types.IndexInfo, params map[string]any) error
    
    // Batch-saves multiple index entries
    BatchSave(ctx context.Context, indexInfoList []*types.IndexInfo, params map[string]any) error
    
    // Estimates the storage space required for the index
    EstimateStorageSize(ctx context.Context, indexInfoList []*types.IndexInfo, params map[string]any) int64
    
    // Deletes indices by a list of chunk IDs
    DeleteByChunkIDList(ctx context.Context, indexIDList []string, dimension int) error
    
    // Copies index data, avoiding recomputation of embedding vectors
    CopyIndices(
        ctx context.Context,
        sourceKnowledgeBaseID string,
        sourceToTargetKBIDMap map[string]string,
        sourceToTargetChunkIDMap map[string]string,
        targetKnowledgeBaseID string,
        dimension int,
    ) error
    
    // Deletes indices by a list of knowledge IDs
    DeleteByKnowledgeIDList(ctx context.Context, knowledgeIDList []string, dimension int) error
    
    // Inherits the RetrieveEngine interface
    RetrieveEngine
}
```

#### 3. Implement the service layer interface

Create a service implementing the `RetrieveEngineService` interface, responsible for handling the business logic of index creation and management:

```go
type RetrieveEngineService interface {
    // Creates a single index
    Index(ctx context.Context,
        embedder embedding.Embedder,
        indexInfo *types.IndexInfo,
        retrieverTypes []types.RetrieverType,
    ) error

    // Batch-creates indices
    BatchIndex(ctx context.Context,
        embedder embedding.Embedder,
        indexInfoList []*types.IndexInfo,
        retrieverTypes []types.RetrieverType,
    ) error

    // Estimates the index storage space
    EstimateStorageSize(ctx context.Context,
        embedder embedding.Embedder,
        indexInfoList []*types.IndexInfo,
        retrieverTypes []types.RetrieverType,
    ) int64
    
    // Copies index data
    CopyIndices(
        ctx context.Context,
        sourceKnowledgeBaseID string,
        sourceToTargetKBIDMap map[string]string,
        sourceToTargetChunkIDMap map[string]string,
        targetKnowledgeBaseID string,
        dimension int,
    ) error

    // Deletes indices
    DeleteByChunkIDList(ctx context.Context, indexIDList []string, dimension int) error
    DeleteByKnowledgeIDList(ctx context.Context, knowledgeIDList []string, dimension int) error

    // Inherits the RetrieveEngine interface
    RetrieveEngine
}
```

#### 4. Add environment variable configuration

Add the necessary connection parameters for the new database in the environment configuration:

```
# Add the new database driver name to RETRIEVE_DRIVER (multiple drivers separated by commas)
RETRIEVE_DRIVER=postgres,elasticsearch_v8,your_database

# Connection parameters for the new database
YOUR_DATABASE_ADDR=your_database_host:port
YOUR_DATABASE_USERNAME=username
YOUR_DATABASE_PASSWORD=password
# Other necessary connection parameters...
```

#### 5. Register the retrieval engine

Add the initialization and registration logic for the new database in the `initRetrieveEngineRegistry` function in `internal/container/container.go`:

```go
func initRetrieveEngineRegistry(db *gorm.DB, cfg *config.Config) (interfaces.RetrieveEngineRegistry, error) {
    registry := retriever.NewRetrieveEngineRegistry()
    retrieveDriver := strings.Split(os.Getenv("RETRIEVE_DRIVER"), ",")
    log := logger.GetLogger(context.Background())

    // Existing PostgreSQL and Elasticsearch initialization code...
    
    // Add initialization code for the new vector database
    if slices.Contains(retrieveDriver, "your_database") {
        // Initialize the database client
        client, err := your_database.NewClient(your_database.Config{
            Addresses: []string{os.Getenv("YOUR_DATABASE_ADDR")},
            Username:  os.Getenv("YOUR_DATABASE_USERNAME"),
            Password:  os.Getenv("YOUR_DATABASE_PASSWORD"),
            // Other connection parameters...
        })
        
        if err != nil {
            log.Errorf("Create your_database client failed: %v", err)
        } else {
            // Create the retrieval engine repository
            yourDatabaseRepo := your_database.NewYourDatabaseRepository(client, cfg)
            
            // Register the retrieval engine
            if err := registry.Register(
                retriever.NewKVHybridRetrieveEngine(
                    yourDatabaseRepo, types.YourDatabaseRetrieverEngineType,
                ),
            ); err != nil {
                log.Errorf("Register your_database retrieve engine failed: %v", err)
            } else {
                log.Infof("Register your_database retrieve engine success")
            }
        }
    }

    return registry, nil
}
```

#### 6. Define the retrieval engine type constant

Add the new retrieval engine type constant in `internal/types/retriever.go`:

```go
// RetrieverEngineType defines the retrieval engine type
const (
    ElasticsearchRetrieverEngineType RetrieverEngineType = "elasticsearch"
    PostgresRetrieverEngineType      RetrieverEngineType = "postgres"
    YourDatabaseRetrieverEngineType  RetrieverEngineType = "your_database" // Add the new database type
)
```

## Reference implementation examples

It is recommended to reference the existing PostgreSQL and Elasticsearch implementations as development templates. These implementations are located in the following directories:

- PostgreSQL: `internal/application/repository/retriever/postgres/`
- ElasticsearchV7: `internal/application/repository/retriever/elasticsearch/v7/`
- ElasticsearchV8: `internal/application/repository/retriever/elasticsearch/v8/`
- Apache Doris 4.1: `internal/application/repository/retriever/doris/`
- Tencent VectorDB: `internal/application/repository/retriever/tencentvectordb/`

By following the steps above and referencing the existing implementations, you can successfully integrate a new vector database into the WeKnora system, extending its vector retrieval capabilities.

## Apache Doris 4.1 Integration Notes

Doris is an MPP-style analytical database; its integration strategy has a few special points compared to NoSQL vector databases (Qdrant/Milvus/Weaviate):

### Protocol layer

| Channel | Port | Purpose |
| ---- | ---- | ---- |
| MySQL protocol | FE 9030 | Main-path CRUD, ANN retrieval, full-text search |
| HTTP API | FE 8030 / BE 8040 | Stream Load partial update |

WeKnora calls the MySQL protocol via `database/sql + go-sql-driver/mysql`;
it calls Stream Load via `net/http`. Both channels share the same username/password.

### Table structure and dimension sharding

Each embedding dimension corresponds to a physical table `<DORIS_TABLE_PREFIX>_<dim>` (e.g. `weknora_embeddings_768`).
Key table properties:

```sql
ENGINE=OLAP
UNIQUE KEY(id)
DISTRIBUTED BY HASH(id) BUCKETS 10
PROPERTIES(
    "replication_num"="1",
    "enable_unique_key_merge_on_write"="true"
);
```

`enable_unique_key_merge_on_write=true` is a prerequisite for Stream Load partial update.

### Indexes

- Inverted index (INVERTED): `chunk_id / knowledge_id / knowledge_base_id / source_id / tag_id / is_enabled` are used for filtering; `content` combined with `parser=chinese` supports Chinese full-text search.
- ANN index (HNSW + cosine_distance): built on the `embedding ARRAY<FLOAT>` column.

Note: Doris's ANN index is built **asynchronously** after table creation; while the index is not yet ready, queries fall back to brute-force search (correct results, but slower). In `ensureTable`, WeKnora polls `SHOW INDEX FROM <table>` waiting for `idx_emb` to reach the `FINISHED`/`NORMAL` state, with a timeout cap of 30s.

### Score semantics

Vector retrieval uses:

```sql
1 - cosine_distance_approximate(embedding, <vec>) AS score
```

This flips distance into similarity, matching the direction of Qdrant's cosine similarity: higher values mean more similar.
Threshold comparison uses `HAVING score >= ?`, and ordering uses `ORDER BY score DESC LIMIT ?`.

### Keyword search

Relies on Doris's built-in `MATCH_ANY` and `chinese` parser, so no jieba tokenization is needed on the Go side.
For multiple tables across dimensions, each table is queried individually and results are merged to take the top K, consistent with the current Milvus/Weaviate behavior.

### Batch field updates

`BatchUpdateChunkEnabledStatus / BatchUpdateChunkTagID` are implemented via Stream Load partial update:

- HTTP PUT `http://<fe_http>/api/<db>/<table>/_stream_load`
- Headers: `partial_columns: true`, `columns: id,is_enabled`, `merge_type: APPEND`, `format: json`, `strip_outer_array: true`
- Body: `[{"id": "...", "is_enabled": true}, ...]`

Batches ≤ 1MiB are split automatically, and the request body is constructed via a `bytes.Reader` + `req.GetBody` closure,
ensuring the body can be resent for the FE → BE 307 redirect.

### Environment variables

```bash
RETRIEVE_DRIVER=doris

DORIS_ADDR=doris-fe:9030       # FE MySQL protocol address
DORIS_HTTP_PORT=8030           # FE HTTP port (Stream Load)
DORIS_DATABASE=weknora         # Target database
DORIS_USERNAME=root
DORIS_PASSWORD=
DORIS_TABLE_PREFIX=weknora_embeddings
```

### Running Doris locally

```bash
docker compose --profile doris up -d
docker exec -it WeKnora-doris-fe mysql -h 127.0.0.1 -P 9030 -uroot \
    -e "CREATE DATABASE IF NOT EXISTS weknora;"
```

Then start the WeKnora backend; tables will be created automatically per dimension as data is written to the knowledge base.

## Tencent VectorDB

WeKnora includes a built-in Tencent VectorDB adapter, with the driver name `tencent_vectordb`. This adapter supports vector retrieval, BM25 sparse-vector-based keyword search, and index management, and can participate in WeKnora's upper-layer hybrid retrieval.

### Environment variable example

```env
RETRIEVE_DRIVER=tencent_vectordb
TENCENT_VECTORDB_ADDR=http://your-instance.tencentvectordb.com
TENCENT_VECTORDB_USERNAME=root
TENCENT_VECTORDB_API_KEY=your_tencent_vectordb_api_key
TENCENT_VECTORDB_DATABASE=weknora
TENCENT_VECTORDB_COLLECTION=weknora_embeddings
TENCENT_VECTORDB_REPLICA_NUMBER=1
```

`TENCENT_VECTORDB_COLLECTION` is the collection name prefix. WeKnora creates the actual collection based on the vector dimension, e.g. `weknora_embeddings_768`, to isolate data across different embedding model dimensions.
`TENCENT_VECTORDB_REPLICA_NUMBER` is the number of replicas used when creating the collection, defaulting to `1`; it can be set to `0` for single-node QA environments, and adjusted for production based on the scale of the Tencent VectorDB cluster.

Keyword search relies on the Tencent VectorDB sparse vector index. Newly created collections automatically get a `sparse_vector` index; for existing vector collections created in older versions without this index, you need to recreate the collection and reimport the knowledge base data before keyword search can be enabled.
