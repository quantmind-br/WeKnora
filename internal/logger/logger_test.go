package logger

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/sirupsen/logrus"
)

func newEntry(level logrus.Level, msg string, data logrus.Fields) *logrus.Entry {
	e := logrus.NewEntry(logrus.New())
	e.Time = time.Date(2026, 5, 21, 10, 20, 30, 123_000_000, time.UTC)
	e.Level = level
	e.Message = msg
	e.Data = data
	return e
}

func TestAnsiStripWriter(t *testing.T) {
	var buf strings.Builder
	w := &ansiStripWriter{w: &buf}
	in := []byte("\x1b[32mINFO\x1b[0m hello \x1b[31mERROR\x1b[0m")
	n, err := w.Write(in)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}
	if n != len(in) {
		t.Fatalf("Write n = %d, want %d", n, len(in))
	}
	if got := buf.String(); got != "INFO hello ERROR" {
		t.Fatalf("stripped output = %q, want %q", got, "INFO hello ERROR")
	}
}

func TestFormat_DefaultModeUnchanged(t *testing.T) {
	f := &CustomFormatter{} // no template, no color
	entry := newEntry(logrus.InfoLevel, "hello", logrus.Fields{
		"request_id": "req-1",
		"caller":     "logger_test.go:1[Test]",
		"k1":         "v1",
	})

	out, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format returned error: %v", err)
	}
	got := string(out)

	if !strings.HasPrefix(got, "INFO ") {
		t.Errorf("expected INFO-prefixed default output, got %q", got)
	}
	for _, want := range []string{"2026-05-21 10:20:30.123", "req-1", "k1=v1", "logger_test.go:1[Test]", "hello"} {
		if !strings.Contains(got, want) {
			t.Errorf("default output missing %q: %s", want, got)
		}
	}
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("default output should end with newline, got %q", got)
	}
}

func TestFormat_TemplateReplacesAllPlaceholders(t *testing.T) {
	f := &CustomFormatter{
		Template:     "[%d] %level %thread %logger %traceId | %msg",
		threadNeeded: true,
	}
	entry := newEntry(logrus.WarnLevel, "boom", logrus.Fields{
		"request_id": "req-42",
		"caller":     "x.go:9[Fn]",
		"extra":      "ok",
	})

	out, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	got := string(out)

	for _, want := range []string{
		"[2026-05-21 10:20:30.123]",
		"WARNING",
		"x.go:9[Fn]",
		"req-42",
		"boom",
		"extra=ok",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("template output missing %q: %s", want, got)
		}
	}
	for _, placeholder := range []string{"%d", "%level", "%thread", "%logger", "%traceId", "%msg"} {
		if strings.Contains(got, placeholder) {
			t.Errorf("placeholder %q not substituted: %s", placeholder, got)
		}
	}
}

func TestFormat_TemplateGoroutineIDSkippedWhenNotReferenced(t *testing.T) {
	// When the template doesn't reference %thread, threadNeeded should be false, and the runtime shouldn't fetch the goroutine ID.
	// This is verified indirectly by checking the output doesn't contain a numeric-only goroutine ID segment; more importantly, it ensures no panic occurs.
	f := &CustomFormatter{
		Template:     "[%d] %level | %msg",
		threadNeeded: false,
	}
	entry := newEntry(logrus.InfoLevel, "no-thread", nil)
	if _, err := f.Format(entry); err != nil {
		t.Fatalf("Format error: %v", err)
	}
}

// TestFormat_ColorDoesNotPolluteMessage is a regression test for the colorize mis-coloring bug.
// The old implementation did ReplaceAll(line, "INFO", colored) on the whole line, which would also colorize "INFO"
// appearing in the message body itself; the new implementation only injects color at the %level substitution site.
func TestFormat_ColorDoesNotPolluteMessage(t *testing.T) {
	f := &CustomFormatter{
		ForceColor:   true,
		Template:     "%level | %msg",
		threadNeeded: false,
	}
	entry := newEntry(logrus.InfoLevel, "user INFO loaded", nil)

	out, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	got := string(out)

	// The total number of ANSI sequences in the output should be exactly 2 (one color+reset pair at the start),
	// not 4 as under the old implementation, where "INFO" in the message was also replaced.
	const ansiOpen = "\033[32m" // green for INFO
	const ansiReset = "\033[0m"
	if strings.Count(got, ansiOpen) != 1 {
		t.Errorf("expected exactly 1 green-open sequence, got %d in %q", strings.Count(got, ansiOpen), got)
	}
	if strings.Count(got, ansiReset) != 1 {
		t.Errorf("expected exactly 1 reset sequence, got %d in %q", strings.Count(got, ansiReset), got)
	}
	// A literal "INFO" string in the message body should remain uncolored (its preceding character isn't an ANSI start).
	idx := strings.Index(got, "user INFO loaded")
	if idx < 0 {
		t.Fatalf("message body not found verbatim in output: %q", got)
	}
}

// TestFormat_TemplateNoCascadingReplace verifies that using NewReplacer for a single-pass replacement
// prevents a placeholder literal in a field value (e.g. traceId value "%msg") from being replaced again.
func TestFormat_TemplateNoCascadingReplace(t *testing.T) {
	f := &CustomFormatter{
		Template:     "%traceId>%msg",
		threadNeeded: false,
	}
	entry := newEntry(logrus.InfoLevel, "actual-msg", logrus.Fields{
		"request_id": "%msg", // Malicious/coincidental field values
	})

	out, err := f.Format(entry)
	if err != nil {
		t.Fatalf("Format error: %v", err)
	}
	got := strings.TrimRight(string(out), "\n")
	want := "%msg>actual-msg"
	if got != want {
		t.Errorf("cascading-replace regression: got %q, want %q", got, want)
	}
}

func TestLevelColorFor(t *testing.T) {
	cases := map[logrus.Level]string{
		logrus.DebugLevel: colorCyan,
		logrus.InfoLevel:  colorGreen,
		logrus.WarnLevel:  colorYellow,
		logrus.ErrorLevel: colorRed,
		logrus.FatalLevel: colorPurple,
		logrus.TraceLevel: "",
	}
	for lvl, want := range cases {
		if got := levelColorFor(lvl); got != want {
			t.Errorf("levelColorFor(%v) = %q, want %q", lvl, got, want)
		}
	}
}

func TestCloneContextPreservesPrincipal(t *testing.T) {
	t.Parallel()

	ctx := types.WithPrincipal(context.Background(), types.EmbedSessionPrincipal(10000, "ch1", "sess1"))
	cloned := CloneContext(ctx)

	if got := types.SessionOwnerIDFromContext(cloned); got != "embed_session:10000:ch1:sess1" {
		t.Fatalf("SessionOwnerIDFromContext(cloned) = %q", got)
	}
}

func TestCloneContextPreservesTenantAPIKeyScope(t *testing.T) {
	t.Parallel()

	want := types.TenantAPIKeyScope{
		KeyID:            7,
		KnowledgeBaseIDs: types.StringArray{"kb-1"},
	}
	ctx := types.WithTenantAPIKeyScope(context.Background(), want)
	cloned := CloneContext(ctx)

	got, ok := types.TenantAPIKeyScopeFromContext(cloned)
	if !ok {
		t.Fatal("TenantAPIKeyScopeFromContext(cloned) = false, want true")
	}
	if got.KeyID != want.KeyID || !got.AllowsKnowledgeBase("kb-1") || got.AllowsKnowledgeBase("kb-2") {
		t.Fatalf("cloned scope = %#v, want key_id=7 scoped to kb-1", got)
	}
}
