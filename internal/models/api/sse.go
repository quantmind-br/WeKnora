package api

import (
	"bufio"
	"io"
	"strings"
)

// SSEEvent represents a Server-Sent Events event
type SSEEvent struct {
	Data []byte
	Done bool
}

// SSEReader is used to read an SSE stream
type SSEReader struct {
	scanner *bufio.Scanner
}

// NewSSEReader creates an SSE reader
func NewSSEReader(reader io.Reader) *SSEReader {
	scanner := bufio.NewScanner(reader)
	// Sets a larger buffer to handle long lines (chain-of-thought content can be long)
	buf := make([]byte, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	return &SSEReader{scanner: scanner}
}

// ReadEvent reads the next SSE event
func (r *SSEReader) ReadEvent() (*SSEEvent, error) {
	for r.scanner.Scan() {
		line := r.scanner.Text()

		// Empty line, skip
		if line == "" {
			continue
		}

		// Parse the data line. The SSE spec only requires the "data:" prefix; the single
		// space after the colon is optional, so both forms must be accepted.
		if !strings.HasPrefix(line, "data:") {
			// Other lines (e.g. event:, id:, etc.) are skipped
			continue
		}
		payload := strings.TrimPrefix(line[len("data:"):], " ")

		// Check whether it's the end marker. Some gateways write "data:[DONE]" or leave
		// trailing whitespace, so compare the trimmed content to avoid parsing the sentinel as JSON.
		if strings.TrimSpace(payload) == "[DONE]" {
			return &SSEEvent{Done: true}, nil
		}

		return &SSEEvent{Data: []byte(payload)}, nil

		// Other lines (e.g. event:, id:, etc.) are skipped
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}
