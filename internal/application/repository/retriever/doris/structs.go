package doris

import (
	"database/sql"
	"net/http"
	"sync"
)

// dorisRepository is the retrieval engine repository implementation for Apache Doris 4.1.
//
// Communication channels:
// - Main read/write path: MySQL protocol (database/sql + go-sql-driver/mysql), FE default port 9030.
// - Stream Load: HTTP (FE default port 8030), legacy mode used for partial update.
//
// Table schema is sharded by dimension: <tableBaseName>_<dim>.
// Compatibility mode is determined by DORIS_COMPAT_MODE:
//   - legacy：UNIQUE KEY(id) + cosine_distance ANN + Stream Load partial update
//   - inner_product_duplicate：DUPLICATE KEY(id) + normalized inner product + delete/insert rewrite
// this setting cannot be swapped directly after the embedding tables are created; these tables must be rebuilt before switching modes.
//
// Like Qdrant/Milvus/Weaviate, initializedTables caches dimensions that have "already been ensured to exist,"
// avoiding a SHOW TABLES call on every write.
type dorisRepository struct {
	db *sql.DB

	httpClient *http.Client
	// fe HTTP base, e.g. "http://doris-fe:8030". The Stream Load path
	// is built by streamLoadURL(table): <feHTTPBase>/api/<database>/<table>/_stream_load.
	feHTTPBase string

	username string
	password string
	database string

	tableBaseName  string
	bucketsNum     int // 0 -> default 10
	replicationNum int // 0 -> default 1
	compatModeRequested dorisCompatMode
	compatModeResolved  dorisCompatMode
	compatResolveOnce   sync.Once
	compatResolveErr    error

	// Set of dimensions already ensured via ensureTable: dim -> true.
	initializedTables sync.Map
}

// DorisVectorEmbedding is the domain model for a row stored in a Doris table.
//
// Field order matches the INSERT column order in schema.go;
// createInsert and columns must be updated together when making changes.
type DorisVectorEmbedding struct {
	ID              string
	Content         string
	SourceID        string
	SourceType      int
	ChunkID         string
	KnowledgeID     string
	KnowledgeBaseID string
	TagID           string
	IsEnabled       bool
	Embedding       []float32
}

// DorisVectorEmbeddingWithScore is the domain model for a search result,
// Score is computed per the current compat mode for vector search, and uniformly set to 1.0 for keyword search.
type DorisVectorEmbeddingWithScore struct {
	DorisVectorEmbedding
	Score float64
}
