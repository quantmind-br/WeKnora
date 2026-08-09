package chat

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

		// Check whether it's the end marker
		if line == "data: [DONE]" {
			return &SSEEvent{Done: true}, nil
		}

		// Parse the data line
		if strings.HasPrefix(line, "data: ") {
			jsonStr := line[6:]
			return &SSEEvent{Data: []byte(jsonStr)}, nil
		}

		// Improve compatibility (no space right after "data:")
		if strings.HasPrefix(line, "data:") {
			jsonStr := line[5:]
			return &SSEEvent{Data: []byte(jsonStr)}, nil
		}

		// Other lines (e.g. event:, id:, etc.) are skipped
	}

	if err := r.scanner.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}
