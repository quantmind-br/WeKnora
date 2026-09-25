package cohererank

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Tencent/WeKnora/internal/models/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zhipuResponse is the documented response of Zhipu's rerank endpoint, whose
// result objects carry document / index / relevance_score and whose envelope
// adds id, request_id, created and usage:
// https://docs.bigmodel.cn/api-reference/模型-api/文本重排序
const zhipuResponse = `{
  "id": "8305283656248111323",
  "request_id": "8305283656248111323",
  "created": 1769040000,
  "results": [
    {"index": 1, "relevance_score": 0.9819, "document": "Shanghai climate"},
    {"index": 0, "relevance_score": 0.0021, "document": "Beijing cuisine"}
  ],
  "usage": {"prompt_tokens": 24, "total_tokens": 24}
}`

// jinaResponse spells the echoed document as an object instead of a string,
// which is the other shape vendors use for the same field:
// https://jina.ai/reranker/
const jinaResponse = `{
  "model": "jina-reranker-v3.5",
  "results": [
    {"index": 0, "relevance_score": 0.95, "document": {"text": "Shanghai climate"}},
    {"index": 1, "relevance_score": 0.11, "document": {"text": "Beijing cuisine"}}
  ],
  "usage": {"total_tokens": 32}
}`

func newClient(t *testing.T, url string, settings api.RerankSettings) *Client {
	t.Helper()
	return New(Config{
		Endpoint: api.Endpoint{BaseURL: url, Model: "rerank", Auth: api.BearerAuth("k")},
		Settings: settings,
	})
}

func serve(t *testing.T, payload string) (*httptest.Server, *string, *map[string]any) {
	t.Helper()
	t.Setenv("SSRF_WHITELIST", "127.0.0.1")
	path := new(string)
	body := new(map[string]any)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(payload))
	}))
	return server, path, body
}

func TestRequestBodyCarriesOnlyWhatTheVendorDeclares(t *testing.T) {
	c := newClient(t, "https://example.invalid/v1", api.RerankSettings{})
	body, err := c.BuildRequestBody("Shanghai weather", []string{"Shanghai climate", "Beijing cuisine"})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"model":     "rerank",
		"query":     "Shanghai weather",
		"documents": []any{"Shanghai climate", "Beijing cuisine"},
	}, body, "optional fields must be absent unless the vendor asked for them")
}

func TestRequestBodyAddsTheOptionalFieldsAVendorDeclares(t *testing.T) {
	c := newClient(t, "https://example.invalid/v1",
		api.RerankSettings{SendTopN: true, SendReturnDocs: true})
	body, err := c.BuildRequestBody("q", []string{"a", "b", "c"})
	require.NoError(t, err)

	assert.Equal(t, float64(3), body["top_n"], "top_n asks for every document back")
	assert.Equal(t, true, body["return_documents"])
}

func TestDecodesAStringDocument(t *testing.T) {
	server, _, _ := serve(t, zhipuResponse)
	defer server.Close()

	c := newClient(t, server.URL, api.RerankSettings{})
	got, err := c.Rerank(context.Background(), "Shanghai weather", []string{"Beijing cuisine", "Shanghai climate"})
	require.NoError(t, err)

	assert.Equal(t, []api.RerankResult{
		{Index: 1, Score: 0.9819, Text: "Shanghai climate"},
		{Index: 0, Score: 0.0021, Text: "Beijing cuisine"},
	}, got)
}

func TestDecodesAnObjectDocument(t *testing.T) {
	server, _, _ := serve(t, jinaResponse)
	defer server.Close()

	c := newClient(t, server.URL, api.RerankSettings{})
	got, err := c.Rerank(context.Background(), "Shanghai weather", []string{"Shanghai climate", "Beijing cuisine"})
	require.NoError(t, err)

	assert.Equal(t, []api.RerankResult{
		{Index: 0, Score: 0.95, Text: "Shanghai climate"},
		{Index: 1, Score: 0.11, Text: "Beijing cuisine"},
	}, got)
}

// Some gateways serving this shape spell the score `score`. The pre-catalog
// client accepted both and so does this one.
func TestDecodesTheScoreAlias(t *testing.T) {
	server, _, _ := serve(t, `{"results":[{"index":0,"score":0.42}]}`)
	defer server.Close()

	c := newClient(t, server.URL, api.RerankSettings{})
	got, err := c.Rerank(context.Background(), "q", []string{"d"})
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.InDelta(t, 0.42, got[0].Score, 1e-9)
}

func TestAppendsTheRerankPathToABareBaseURL(t *testing.T) {
	server, path, _ := serve(t, `{"results":[]}`)
	defer server.Close()

	c := newClient(t, server.URL+"/v1", api.RerankSettings{})
	_, err := c.Rerank(context.Background(), "q", []string{"d"})
	require.NoError(t, err)
	assert.Equal(t, "/v1/rerank", *path)
}

// Zhipu's default base URL already ends in /rerank, and operators paste full
// endpoints too. Appending the path again would 404.
func TestDoesNotDoubleTheRerankPath(t *testing.T) {
	server, path, _ := serve(t, `{"results":[]}`)
	defer server.Close()

	c := newClient(t, server.URL+"/api/paas/v4/rerank", api.RerankSettings{})
	_, err := c.Rerank(context.Background(), "q", []string{"d"})
	require.NoError(t, err)
	assert.Equal(t, "/api/paas/v4/rerank", *path)
}

// WeKnora Cloud serves the same shape on its own path.
func TestHonoursAVendorDeclaredPath(t *testing.T) {
	server, path, _ := serve(t, `{"results":[]}`)
	defer server.Close()

	c := newClient(t, server.URL, api.RerankSettings{Path: "/api/v1/rerank"})
	_, err := c.Rerank(context.Background(), "q", []string{"d"})
	require.NoError(t, err)
	assert.Equal(t, "/api/v1/rerank", *path)
}

func TestRejectsAnOutOfRangeIndex(t *testing.T) {
	server, _, _ := serve(t, `{"results":[{"index":9,"relevance_score":0.5}]}`)
	defer server.Close()

	c := newClient(t, server.URL, api.RerankSettings{})
	_, err := c.Rerank(context.Background(), "q", []string{"d"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestSurfacesTheVendorErrorBody(t *testing.T) {
	t.Setenv("SSRF_WHITELIST", "127.0.0.1")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"message":"documents exceeds the limit"}}`))
	}))
	defer server.Close()

	c := newClient(t, server.URL, api.RerankSettings{})
	_, err := c.Rerank(context.Background(), "q", []string{"d"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "documents exceeds the limit")
}

// truncate_prompt_tokens is a vLLM extension that appears in no managed
// vendor's documented schema. It is sent only where the vendor declared it
// and the operator opted in, and never by default — sending it unasked is how
// an undocumented field ends up on every request to every gateway.
func TestTruncatePromptTokensIsAbsentByDefault(t *testing.T) {
	c := newClient(t, "https://example.invalid/v1", api.RerankSettings{})
	body, err := c.BuildRequestBody("q", []string{"d"})
	require.NoError(t, err)
	assert.NotContains(t, body, "truncate_prompt_tokens")
}

func TestTruncatePromptTokensIsSentWhenTheRowOptedIn(t *testing.T) {
	c := newClient(t, "https://example.invalid/v1", api.RerankSettings{
		AcceptsTruncatePromptTokens: true, TruncatePromptTokens: 512,
	})
	body, err := c.BuildRequestBody("q", []string{"d"})
	require.NoError(t, err)
	assert.Equal(t, float64(512), body["truncate_prompt_tokens"])
}
