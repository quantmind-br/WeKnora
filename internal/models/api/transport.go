package api

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	secutils "github.com/Tencent/WeKnora/internal/utils"
)

// LLM call timeout configuration. Only serves as a fallback for when the upper layer hasn't set a deadline, to avoid hung requests
// permanently blocking the worker. If the upper-layer ctx has already set a deadline (whether shorter or longer than the default),
// it will be respected as-is, without stacking the default timeout on top. Can be overridden via environment variables:
// - WEKNORA_LLM_CHAT_TIMEOUT_SECONDS    fallback timeout for non-streaming calls (default 600s)
// - WEKNORA_LLM_STREAM_TIMEOUT_SECONDS  fallback timeout for streaming calls (default 1800s)
var (
	DefaultChatTimeout   = envDurationSeconds("WEKNORA_LLM_CHAT_TIMEOUT_SECONDS", 300*time.Second)
	DefaultStreamTimeout = envDurationSeconds("WEKNORA_LLM_STREAM_TIMEOUT_SECONDS", 600*time.Second)
)

// envDurationSeconds reads an environment variable expressed in seconds, falling back to fallback if parsing fails or the value is non-positive.
func envDurationSeconds(key string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return time.Duration(n) * time.Second
}

// WithLLMTimeout only attaches a fallback timeout when the upper-layer ctx has no deadline;
// if the upper layer has already explicitly set a deadline (whether shorter or longer), it is returned as-is,
// leaving the caller with final say over its own timeout strategy.
func WithLLMTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

// HTTPClient is a shared HTTP client for raw HTTP LLM calls with connection-level timeouts.
// Per-request timeout is enforced via context deadline (see DefaultChatTimeout / DefaultStreamTimeout)
// rather than http.Client.Timeout, so streaming calls are not prematurely terminated.
// Uses SSRFSafeDialContext to prevent DNS rebinding attacks at the connection layer.
var httpTransport = &http.Transport{
	Proxy:               http.ProxyFromEnvironment,
	DialContext:         secutils.SSRFSafeDialContext,
	TLSHandshakeTimeout: 10 * time.Second,
	IdleConnTimeout:     90 * time.Second,
	MaxIdleConnsPerHost: 5,
}

// HTTPClient is the shared SSRF-safe client every protocol uses.
var HTTPClient = secutils.NewSSRFSafeHTTPClientWithTransport(
	secutils.SSRFSafeHTTPClientConfig{Timeout: 0, MaxRedirects: 10},
	httpTransport,
)
