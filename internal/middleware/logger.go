package middleware

import (
	"bytes"
	"context"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
	secutils "github.com/Tencent/WeKnora/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	maxBodySize = 1024 * 10 // Log body content up to a maximum of 10KB.
)

// loggerResponseBodyWriter: custom ResponseWriter used to capture response content (for the logger middleware).
type loggerResponseBodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write: overrides the Write method to write to both the buffer and the original writer.
// Limit buffer size to avoid unbounded memory growth from streaming responses such as SSE.
func (r loggerResponseBodyWriter) Write(b []byte) (int, error) {
	if r.body.Len() < maxBodySize {
		remaining := maxBodySize - r.body.Len()
		if len(b) <= remaining {
			r.body.Write(b)
		} else {
			r.body.Write(b[:remaining])
		}
	}
	return r.ResponseWriter.Write(b)
}

// sensitiveFieldRegex matches sensitive fields in JSON (case-insensitive, compatible with snake_case / camelCase / PascalCase).
// $1 captures the original field name (including surrounding quotes), keeping the field name unchanged in the log while replacing only the value with "***".
var sensitiveFieldRegex = regexp.MustCompile(
	`(?i)("(?:new[_-]?password|old[_-]?password|password|passwd|ticket|token|access[_-]?token|` +
		`refresh[_-]?token|next[_-]?token|device[_-]?token|pending[_-]?token|pairing[_-]?link|` +
		`id[_-]?token|authorization|auth[_-]?token|api[_-]?key|` +
		`api[_-]?secret|secret[_-]?key|client[_-]?secret|private[_-]?key|secret|` +
		`authorization[_-]?url|authorization[_-]?attempt)")\s*:\s*"[^"]*"`,
)

// sanitizeBody: sanitize sensitive information.
func sanitizeBody(body string) string {
	return sensitiveFieldRegex.ReplaceAllString(body, `$1:"***"`)
}

var sensitiveQueryFields = map[string]struct{}{
	"access_token":          {},
	"authorization_attempt": {},
	"code":                  {},
	"id_token":              {},
	"refresh_token":         {},
	"state":                 {},
	// ticket is the sandbox terminal's WebSocket handshake credential. A
	// browser cannot set Authorization on an upgrade, so it travels in the
	// query string; anyone holding it for its 2-minute TTL can open a shell
	// in the session's sandbox, which is why it must never reach a log line.
	"ticket": {},
	"token":  {},
}

// sanitizeQuery prevents OAuth authorization codes, CSRF/attempt state, and
// handshake credentials from being copied into access logs. Parsing the query
// also covers repeated and percent-encoded parameters without relying on
// fragile string replacement.
func sanitizeQuery(raw string) string {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return "[invalid query omitted]"
	}
	for key := range values {
		if _, sensitive := sensitiveQueryFields[strings.ToLower(key)]; sensitive {
			values[key] = []string{"***"}
		}
	}
	return values.Encode()
}

// readRequestBody: read the request body (size-limited for logging, but read in full for resetting).
func readRequestBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}

	// Check Content-Type; only log JSON type.
	contentType := c.GetHeader("Content-Type")
	if !strings.Contains(contentType, "application/json") &&
		!strings.Contains(contentType, "application/x-www-form-urlencoded") &&
		!strings.Contains(contentType, "text/") {
		return "[Non-text type, skipped]"
	}

	// Read the full body content (no size limit), since it needs to be fully reset for use by the following handler.
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "[Failed to read request body]"
	}

	// Reset the request body with the full content, ensuring subsequent handlers can read the complete data.
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Body used for logging (size-limited).
	var logBodyBytes []byte
	if len(bodyBytes) > maxBodySize {
		logBodyBytes = bodyBytes[:maxBodySize]
	} else {
		logBodyBytes = bodyBytes
	}

	bodyStr := string(logBodyBytes)
	if len(bodyBytes) > maxBodySize {
		bodyStr += "... [content too long, truncated]"
	}

	return sanitizeBody(bodyStr)
}

// RequestID middleware adds a unique request ID to the context
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get request ID from header or generate a new one
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		safeRequestID := secutils.SanitizeForLog(requestID)
		// Set request ID in header
		c.Header("X-Request-ID", requestID)

		// Set request ID in context
		c.Set(types.RequestIDContextKey.String(), requestID)

		// Set logger in context
		requestLogger := logger.GetLogger(c)
		requestLogger = requestLogger.WithField("request_id", safeRequestID)
		c.Set(types.LoggerContextKey.String(), requestLogger)

		// Set request ID in the global context for logging
		c.Request = c.Request.WithContext(
			context.WithValue(
				context.WithValue(c.Request.Context(), types.RequestIDContextKey, requestID),
				types.LoggerContextKey, requestLogger,
			),
		)

		c.Next()
	}
}

// Logger middleware logs request details with request ID, input and output
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		isWikiStats := strings.HasPrefix(path, "/api/v1/knowledgebase/") && strings.HasSuffix(path, "/wiki/stats")
		if strings.HasPrefix(path, "/assets/") || isWikiStats {
			c.Next()
			return
		}

		// Browser traffic contains credentials, page content and screenshots.
		// Keep access metadata, but never read or buffer these request/response bodies.
		browserTraffic := strings.HasPrefix(path, "/api/v1/local-browser/") ||
			path == "/api/v1/me/browser" || strings.HasSuffix(path, "/local-browser")
		// Read the request body (before Next, since Next will consume the body).
		var requestBody string
		if !browserTraffic && (c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH") {
			requestBody = readRequestBody(c)
		}

		// Create the response body capturer.
		responseBody := &bytes.Buffer{}
		responseWriter := &loggerResponseBodyWriter{
			ResponseWriter: c.Writer,
			body:           responseBody,
		}
		if !browserTraffic {
			c.Writer = responseWriter
		}

		// Process request
		c.Next()

		// Get request ID from context
		requestID, exists := c.Get(types.RequestIDContextKey.String())
		requestIDStr := "unknown"
		if exists {
			if idStr, ok := requestID.(string); ok && idStr != "" {
				requestIDStr = idStr
			}
		}
		safeRequestID := secutils.SanitizeForLog(requestIDStr)

		// Calculate latency
		latency := time.Since(start)

		// Get client IP and status code
		clientIP := c.ClientIP()
		statusCode := c.Writer.Status()
		method := c.Request.Method

		if raw != "" {
			path = path + "?" + sanitizeQuery(raw)
		}

		// Read the response body
		responseBodyStr := ""
		if responseBody.Len() > 0 {
			contentType := c.Writer.Header().Get("Content-Type")
			if strings.Contains(contentType, "text/event-stream") {
				responseBodyStr = "[SSE stream response, skipped]"
			} else if strings.Contains(contentType, "application/json") ||
				strings.Contains(contentType, "text/") {
				bodyBytes := responseBody.Bytes()
				if len(bodyBytes) >= maxBodySize {
					responseBodyStr = string(bodyBytes[:maxBodySize]) + "... [content too long, truncated]"
				} else {
					responseBodyStr = string(bodyBytes)
				}
				responseBodyStr = sanitizeBody(responseBodyStr)
			} else {
				responseBodyStr = "[Non-text type, skipped]"
			}
		}

		// Build the log message
		logMsg := logger.GetLogger(c)
		logMsg = logMsg.WithFields(map[string]interface{}{
			"request_id":  safeRequestID,
			"method":      method,
			"path":        secutils.SanitizeForLog(path),
			"status_code": statusCode,
			"size":        c.Writer.Size(),
			"latency":     latency.String(),
			"client_ip":   secutils.SanitizeForLog(clientIP),
		})

		// Add the request body (if any)
		if requestBody != "" {
			logMsg = logMsg.WithField("request_body", secutils.SanitizeForLog(requestBody))
		}

		// Add the response body (if any)
		if responseBodyStr != "" {
			logMsg = logMsg.WithField("response_body", secutils.SanitizeForLog(responseBodyStr))
		}
		if last := c.Errors.Last(); last != nil && last.Err != nil {
			logMsg = logMsg.WithField("error", secutils.SanitizeForLog(last.Err.Error()))
		}
		switch {
		case statusCode >= 500:
			logMsg.Error()
		case statusCode >= 400:
			logMsg.Warn()
		default:
			logMsg.Info()
		}
	}
}
