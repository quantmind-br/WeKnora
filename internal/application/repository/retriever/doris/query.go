package doris

import (
	"math"
	"strconv"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

// Field name constants. Doris is a SQL store, and field names are reused across SELECT/WHERE/INSERT,
// use constants to prevent typos consistently.
const (
	fieldID              = "id"
	fieldContent         = "content"
	fieldSourceID        = "source_id"
	fieldSourceType      = "source_type"
	fieldChunkID         = "chunk_id"
	fieldKnowledgeID     = "knowledge_id"
	fieldKnowledgeBaseID = "knowledge_base_id"
	fieldTagID           = "tag_id"
	fieldIsEnabled       = "is_enabled"
	fieldEmbedding       = "embedding"
)

// columns is the standard column order used for INSERT / SELECT.
var columns = []string{
	fieldID, fieldContent, fieldSourceID, fieldSourceType,
	fieldChunkID, fieldKnowledgeID, fieldKnowledgeBaseID, fieldTagID,
	fieldIsEnabled, fieldEmbedding,
}

// columnsForRetrieve is the column order for SELECT during Retrieve,
// excluding embedding (the vector itself doesn't need to be returned in query results, saving bandwidth).
var columnsForRetrieve = []string{
	fieldID, fieldContent, fieldSourceID, fieldSourceType,
	fieldChunkID, fieldKnowledgeID, fieldKnowledgeBaseID, fieldTagID,
	fieldIsEnabled,
}

// columnsForCopy is the column order used for the paginated SELECT in CopyIndices,
// it includes embedding in addition to columnsForRetrieve, since the copy's purpose is to move the vector itself.
var columnsForCopy = []string{
	fieldID, fieldContent, fieldSourceID, fieldSourceType,
	fieldChunkID, fieldKnowledgeID, fieldKnowledgeBaseID, fieldTagID,
	fieldIsEnabled, fieldEmbedding,
}

// whereCond represents a WHERE sub-condition: clause is a parameterized SQL fragment (with ? placeholders),
// args are the corresponding parameter values in order. All user-input fields (IDs) must be passed via args,
// never concatenated directly into the clause string.
type whereCond struct {
	clause string
	args   []any
}

// whereBuilder translates the filter conditions in RetrieveParams into a SQL WHERE clause.
//
// Each add* method corresponds to an IN / NOT IN / = operator; build() finally joins them with AND.
type whereBuilder struct {
	conds []whereCond
}

// addEqual appends a field = ? condition.
func (w *whereBuilder) addEqual(field string, value any) {
	w.conds = append(w.conds, whereCond{
		clause: field + " = ?",
		args:   []any{value},
	})
}

// addIn appends a field IN (?, ?, ...) condition. Appends nothing if values is empty.
func (w *whereBuilder) addIn(field string, values []string) {
	if len(values) == 0 {
		return
	}
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		args[i] = v
	}
	w.conds = append(w.conds, whereCond{
		clause: field + " IN (" + strings.Join(placeholders, ", ") + ")",
		args:   args,
	})
}

// addNotIn appends a field NOT IN (?, ?, ...) condition.
func (w *whereBuilder) addNotIn(field string, values []string) {
	if len(values) == 0 {
		return
	}
	placeholders := make([]string, len(values))
	args := make([]any, len(values))
	for i, v := range values {
		placeholders[i] = "?"
		args[i] = v
	}
	w.conds = append(w.conds, whereCond{
		clause: field + " NOT IN (" + strings.Join(placeholders, ", ") + ")",
		args:   args,
	})
}

// build returns the WHERE clause (without the "WHERE " prefix) and the parameter array.
// Returns ("1 = 1", nil) when there are no conditions, so callers can concatenate without special-casing.
func (w *whereBuilder) build() (string, []any) {
	if len(w.conds) == 0 {
		return "1 = 1", nil
	}
	parts := make([]string, len(w.conds))
	var args []any
	for i, c := range w.conds {
		parts[i] = c.clause
		args = append(args, c.args...)
	}
	return strings.Join(parts, " AND "), args
}

// buildBaseFilter translates the filter conditions in RetrieveParams into a whereBuilder.
// By default appends is_enabled = TRUE, consistent with Qdrant/Milvus/Weaviate:
// disabled chunks are excluded from retrieval.
func buildBaseFilter(params types.RetrieveParams) *whereBuilder {
	w := &whereBuilder{}
	w.addEqual(fieldIsEnabled, true)

	if len(params.KnowledgeBaseIDs) > 0 {
		w.addIn(fieldKnowledgeBaseID, params.KnowledgeBaseIDs)
	}
	if len(params.KnowledgeIDs) > 0 {
		w.addIn(fieldKnowledgeID, params.KnowledgeIDs)
	}
	if len(params.TagIDs) > 0 {
		w.addIn(fieldTagID, params.TagIDs)
	}
	if len(params.ExcludeKnowledgeIDs) > 0 {
		w.addNotIn(fieldKnowledgeID, params.ExcludeKnowledgeIDs)
	}
	if len(params.ExcludeChunkIDs) > 0 {
		w.addNotIn(fieldChunkID, params.ExcludeChunkIDs)
	}
	return w
}

// parseEmbeddingLiteral parses the literal string returned by Doris ARRAY<FLOAT> over the MySQL protocol
// (in the form "[1,2,3]") into []float32.
//
// The CopyIndices path needs to read the vector itself from the source row and write it back to the target row; parsing here favors fault tolerance:
// also accepts values without [], and returns nil for an empty array.
func parseEmbeddingLiteral(raw []byte) ([]float32, error) {
	s := strings.TrimSpace(string(raw))
	if s == "" {
		return nil, nil
	}
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	out := make([]float32, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		f, err := strconv.ParseFloat(p, 32)
		if err != nil {
			return nil, err
		}
		out = append(out, float32(f))
	}
	return out, nil
}

// validateEmbedding checks that all vector elements are finite values.
//
// strconv.FormatFloat outputs "NaN"/"+Inf"/"-Inf" for NaN/±Inf,
// these literals cause a SQL syntax error when concatenated by Doris (or produce undefined results in some versions).
// Under normal conditions the upstream (embedding model) won't output non-finite values, but GPU OOM, upstream bugs,
// or test stubs could trigger this; failing fast here is safer than silently writing corrupt data.
func validateEmbedding(vec []float32) error {
	for i, v := range vec {
		if f := float64(v); math.IsNaN(f) || math.IsInf(f, 0) {
			return errInvalidEmbedding{index: i, value: v}
		}
	}
	return nil
}

// normalizeEmbedding returns a unit-length copy of vec so Doris inner-product
// ANN search can preserve cosine-style similarity semantics.
func normalizeEmbedding(vec []float32) []float32 {
	if len(vec) == 0 {
		return nil
	}
	var sumSquares float64
	for _, value := range vec {
		f := float64(value)
		sumSquares += f * f
	}
	if sumSquares == 0 {
		return append([]float32(nil), vec...)
	}
	norm := float32(math.Sqrt(sumSquares))
	normalized := make([]float32, len(vec))
	for i, value := range vec {
		normalized[i] = value / norm
	}
	return normalized
}

// errInvalidEmbedding describes which index contains a non-finite value; a struct is used instead of fmt.Errorf
// so the caller can get the index from the logs for troubleshooting.
type errInvalidEmbedding struct {
	index int
	value float32
}

func (e errInvalidEmbedding) Error() string {
	return "doris: embedding[" + strconv.Itoa(e.index) +
		"] is not finite: " + strconv.FormatFloat(float64(e.value), 'g', -1, 32)
}

// embeddingLiteral converts []float32 into a Doris ARRAY<FLOAT> literal string:
// "[1.23,4.56,...]"。
//
// why not use a placeholder: go-sql-driver/mysql doesn't support parameter binding for the ARRAY type,
// and Doris only accepts the literal form on its end. Here strconv.FormatFloat ('g' + bitSize=32) is used
// Instead of fmt.Sprintf("%f", v), for two reasons:
// 1. fmt uses thousands separators in some locales, which breaks SQL syntax;
// 2. 'g' is shorter than 'f' and doesn't lose precision.
//
// Injection risk: []float32 elements are finite-precision floats output by an embedding model, so after serialization they can only
// contain [0-9eE+-.\s] characters and can't escape the literal context.
func embeddingLiteral(vec []float32) string {
	if len(vec) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.Grow(len(vec) * 12)
	sb.WriteByte('[')
	for i, v := range vec {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(strconv.FormatFloat(float64(v), 'g', -1, 32))
	}
	sb.WriteByte(']')
	return sb.String()
}
