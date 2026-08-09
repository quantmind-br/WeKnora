package utils

import (
	"net/http"
	"strings"
)

// reservedHeaderKeys lists critical request headers that are not allowed to be overridden by custom user headers.
// These headers are controlled by each provider's signing, authentication, or SSE flow; overriding them can directly cause the call to fail.
var reservedHeaderKeys = map[string]struct{}{
	"authorization":     {},
	"api-key":           {},
	"x-api-key":         {},
	"x-goog-api-key":    {},
	"content-type":      {},
	"content-length":    {},
	"accept-encoding":   {},
	"host":              {},
	"connection":        {},
	"transfer-encoding": {},
}

// IsReservedHeader checks whether a header key is a reserved header; reserved headers cannot be overridden by custom headers.
func IsReservedHeader(key string) bool {
	_, ok := reservedHeaderKeys[strings.ToLower(strings.TrimSpace(key))]
	return ok
}

// ApplyCustomHeaders writes user-defined custom headers into an http.Request.
// Reserved headers (Authorization, api-key, Content-Type, etc.) are skipped to avoid breaking authentication/signing.
// Other headers directly override entries with the same name, allowing users to replace default values (e.g. Accept).
func ApplyCustomHeaders(req *http.Request, headers map[string]string) {
	if req == nil || len(headers) == 0 {
		return
	}
	for k, v := range headers {
		name := strings.TrimSpace(k)
		if name == "" {
			continue
		}
		if IsReservedHeader(name) {
			continue
		}
		req.Header.Set(name, v)
	}
}

// CustomHeadersRoundTripper is an http.RoundTripper wrapper that
// injects user-defined custom headers before each HTTP request is sent.
// Used for scenarios where the underlying *http.Request isn't directly accessible (e.g. the go-openai SDK).
type CustomHeadersRoundTripper struct {
	Headers map[string]string
	Base    http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface.
func (t *CustomHeadersRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if len(t.Headers) > 0 {
		// Clone the request to avoid mutating the object passed in by the caller.
		cloned := req.Clone(req.Context())
		ApplyCustomHeaders(cloned, t.Headers)
		return base.RoundTrip(cloned)
	}
	return base.RoundTrip(req)
}

// WrapHTTPClientWithHeaders returns a new *http.Client that injects custom headers on top of the original client.
// If headers is empty, return the original client directly to avoid unnecessary overhead.
func WrapHTTPClientWithHeaders(client *http.Client, headers map[string]string) *http.Client {
	if len(headers) == 0 {
		return client
	}
	if client == nil {
		client = &http.Client{}
	}
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	wrapped := *client
	wrapped.Transport = &CustomHeadersRoundTripper{Headers: headers, Base: base}
	return &wrapped
}
