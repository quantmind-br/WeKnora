package doris

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
)

// Default bucket count / replica count. Doris falls back to cluster defaults when PROPERTIES doesn't specify them,
// so this gives a conservative value that's friendlier to single-node/small clusters.
const (
	defaultBucketsNum     = 10
	defaultReplicationNum = 1

	// Maximum wait time for ANN index readiness polling. An unready index doesn't block the write path,
	// it only blocks ensureTable itself (first-time table creation), so 30s is acceptable.
	annReadyTimeout = 30 * time.Second
	annReadyPoll    = 1 * time.Second
)

// getTableName returns the physical table name for a given dimension: <base>_<dim>.
//
// Consistent with the collection naming convention used by Qdrant/Milvus/Weaviate,
// so data from different embedding models (different dimensions) never conflicts.
func (r *dorisRepository) getTableName(dimension int) string {
	return fmt.Sprintf("%s_%d", r.tableBaseName, dimension)
}

// ensureTable guarantees that the table for the target dimension already exists;
// if it doesn't, it's created with CREATE TABLE IF NOT EXISTS, then polls for ANN index readiness after creation.
//
// This method is called before every Save / BatchSave, and the result is cached in initializedTables,
// so within the same process the same dimension only actually triggers SHOW TABLES + DDL once.
func (r *dorisRepository) ensureTable(ctx context.Context, dimension int) error {
	if _, ok := r.initializedTables.Load(dimension); ok {
		return nil
	}
	compatMode, err := r.resolveCompatMode(ctx)
	if err != nil {
		return err
	}

	log := logger.GetLogger(ctx)
	tableName := r.getTableName(dimension)

	exists, err := r.tableExists(ctx, tableName)
	if err != nil {
		log.Errorf("[Doris] Failed to check table existence: %v", err)
		return fmt.Errorf("check table existence: %w", err)
	}

	if !exists {
		log.Infof("[Doris] Creating table %s with dimension %d in compat mode %s", tableName, dimension, compatMode)
		if err := r.createTable(ctx, tableName, dimension, compatMode); err != nil {
			log.Errorf("[Doris] Failed to create table: %v", err)
			return fmt.Errorf("create table: %w", err)
		}

		// The ANN index is built asynchronously on the Doris side. Polling for readiness happens here in a background goroutine,
		// so the write path isn't blocked — while the index isn't ready, search falls back to brute-force (correct results, slower),
		// which is more acceptable than blocking the first batch of writes for 30s.
		go func(tn string) {
			// Uses a separate context (with timeout) so that cancellation of the request-level ctx doesn't also kill the background polling.
			bgCtx, cancel := context.WithTimeout(context.Background(), annReadyTimeout)
			defer cancel()
			if err := r.waitANNReady(bgCtx, tn); err != nil {
				logger.GetLogger(bgCtx).Warnf(
					"[Doris] ANN index for %s not ready within %s: %v "+
						"(queries may fall back to brute force temporarily)",
					tn, annReadyTimeout, err)
				return
			}
			logger.GetLogger(bgCtx).Infof("[Doris] ANN index for %s ready", tn)
		}(tableName)
	}

	r.initializedTables.Store(dimension, true)
	return nil
}

// tableExists checks whether the table exists via information_schema.
//
// It doesn't use SHOW TABLES directly because SHOW TABLES LIKE is case-sensitive on Doris 4.1,
// and information_schema has better MySQL compatibility.
func (r *dorisRepository) tableExists(ctx context.Context, tableName string) (bool, error) {
	const q = `SELECT COUNT(1) FROM information_schema.tables
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`
	var n int
	if err := r.db.QueryRowContext(ctx, q, r.database, tableName).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

// createTable issues the CREATE TABLE DDL. Doris DDL is synchronous (except for ANN index building),
// so a successful return means the table is already writable.
func (r *dorisRepository) createTable(ctx context.Context, tableName string, dimension int, compatMode dorisCompatMode) error {
	buckets := r.bucketsNum
	if buckets <= 0 {
		buckets = defaultBucketsNum
	}
	replication := r.replicationNum
	if replication <= 0 {
		replication = defaultReplicationNum
	}

	ddl := buildCreateTableDDL(tableName, dimension, buckets, replication, compatMode)
	_, err := r.db.ExecContext(ctx, ddl)
	if err != nil && compatMode == dorisCompatModeLegacy {
		return fmt.Errorf(
			"legacy Doris table creation failed: %w. If your Doris build rejects ANN indexes on UNIQUE KEY tables, set %s=%s before creating embedding tables. %s is not interchangeable after %s_* tables are created",
			err,
			envDorisCompatMode,
			dorisCompatModeInnerProductDuplicate,
			envDorisCompatMode,
			r.tableBaseName,
		)
	}
	return err
}

// buildCreateTableDDL generates the CREATE TABLE DDL based on the dimension.
//
// Key points:
// - DUPLICATE KEY(id): complies with the current Doris/SelectDB table model requirement for ANN indexes.
// WeKnora uses delete + insert on the Go side to preserve replace-by-id write semantics.
// - The INVERTED index covers all filter fields + a full-text index on content with Chinese tokenization.
// - The ANN index uses HNSW + inner_product; Doris normalizes vectors before write/query,
// so overall it still keeps the same cosine similarity semantics as other vector stores.
//
// Note: the dimension / buckets / replication numeric fields in the DDL are formatted and concatenated on the Go side,
// and there's no SQL injection risk (all sources are controlled IndexConfig ints).
func buildCreateTableDDL(tableName string, dimension, buckets, replication int, compatMode dorisCompatMode) string {
	metricType := "inner_product"
	keyMode := "DUPLICATE KEY(id)"
	properties := fmt.Sprintf("\t\"replication_num\"=\"%d\"", replication)
	if compatMode == dorisCompatModeLegacy {
		metricType = "cosine_distance"
		keyMode = "UNIQUE KEY(id)"
		properties = fmt.Sprintf("\t\"replication_num\"=\"%d\",\n\t\"enable_unique_key_merge_on_write\"=\"true\"", replication)
	}

	const tpl = `CREATE TABLE IF NOT EXISTS ` + "`%s`" + ` (
    id                VARCHAR(64)  NOT NULL,
    chunk_id          VARCHAR(64),
    knowledge_id      VARCHAR(64),
    knowledge_base_id VARCHAR(64),
    source_id         VARCHAR(255),
    source_type       INT,
    tag_id            VARCHAR(64),
    is_enabled        BOOLEAN,
    content           TEXT,
    embedding         ARRAY<FLOAT> NOT NULL,
    INDEX idx_chunk    (chunk_id)          USING INVERTED,
    INDEX idx_kb       (knowledge_base_id) USING INVERTED,
    INDEX idx_kid      (knowledge_id)      USING INVERTED,
    INDEX idx_src      (source_id)         USING INVERTED,
    INDEX idx_tag      (tag_id)            USING INVERTED,
    INDEX idx_enabled  (is_enabled)        USING INVERTED,
    INDEX idx_content  (content)           USING INVERTED PROPERTIES("parser"="chinese","support_phrase"="true"),
    INDEX idx_emb      (embedding)         USING ANN PROPERTIES(
        "index_type"="hnsw",
		"metric_type"="%s",
        "dim"="%d",
        "max_degree"="32",
        "ef_construction"="200"
    )
) ENGINE=OLAP
%s
DISTRIBUTED BY HASH(id) BUCKETS %d
PROPERTIES(
	%s
);`
	return fmt.Sprintf(tpl, tableName, metricType, dimension, keyMode, buckets, properties)
}

// waitANNReady polls SHOW INDEX, waiting for the ANN index to reach FINISHED state.
//
// Doris builds the ANN index asynchronously after table creation; queries fall back to brute-force during that time (correct results, slower).
// This is only a "best-effort" wait: if it's not ready by the deadline, it just logs a warning without blocking writes.
func (r *dorisRepository) waitANNReady(ctx context.Context, tableName string) error {
	deadline := time.Now().Add(annReadyTimeout)
	for {
		ready, err := r.annIndexReady(ctx, tableName)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("ann index not ready within %s", annReadyTimeout)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(annReadyPoll):
		}
	}
}

// annIndexReady checks whether the ANN index's State is FINISHED.
//
// SHOW INDEX FROM <table> returns multiple columns on Doris; column order varies slightly across minor versions,
// so matching is done by column name (using information_schema.statistics + a custom view isn't feasible,
// Just use SHOW INDEX and scan the results.
//
// Compatibility strategy: if no idx_emb row is found in the SHOW INDEX result (very old versions), treat it as ready,
// avoid deadlocking startup due to output differences across Doris versions.
func (r *dorisRepository) annIndexReady(ctx context.Context, tableName string) (bool, error) {
	rows, err := r.db.QueryContext(ctx,
		fmt.Sprintf("SHOW INDEX FROM `%s`", tableName))
	if err != nil {
		return false, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return false, err
	}
	keyNameIdx, stateIdx := -1, -1
	for i, c := range cols {
		switch strings.ToLower(c) {
		case "key_name":
			keyNameIdx = i
		case "state", "index_state":
			stateIdx = i
		}
	}

	for rows.Next() {
		// Use sql.RawBytes to receive values, for compatibility with different column types.
		raw := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range raw {
			ptrs[i] = &raw[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return false, err
		}

		var keyName, state string
		if keyNameIdx >= 0 {
			keyName = bytesToString(raw[keyNameIdx])
		}
		if stateIdx >= 0 {
			state = bytesToString(raw[stateIdx])
		}

		if keyName != "idx_emb" {
			continue
		}
		if stateIdx < 0 {
			// Older versions don't expose the state column, so optimistically assume it's ready.
			return true, nil
		}
		if !strings.EqualFold(state, "FINISHED") &&
			!strings.EqualFold(state, "NORMAL") {
			return false, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	// Two cases lead here:
	// 1. the idx_emb row was found, and state is already FINISHED/NORMAL (or an older version with stateIdx<0);
	// 2. the idx_emb row was not found (very old Doris doesn't expose this index name);
	// both are treated as ready and don't block. The not-ready branch already returns false early inside the loop.
	return true, nil
}

// listEmbeddingTables returns all tables in the current database named <base>_%,
// used for keyword search / cross-dimension BatchUpdate.
func (r *dorisRepository) listEmbeddingTables(ctx context.Context) ([]string, error) {
	const q = `SELECT TABLE_NAME FROM information_schema.tables
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME LIKE ?`
	rows, err := r.db.QueryContext(ctx, q, r.database, r.tableBaseName+"\\_%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		names = append(names, n)
	}
	return names, rows.Err()
}

// bytesToString safely converts the raw any returned by SHOW INDEX (usually []byte or string)
// into a string.
func bytesToString(v any) string {
	switch s := v.(type) {
	case []byte:
		return string(s)
	case string:
		return s
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", s)
	}
}
