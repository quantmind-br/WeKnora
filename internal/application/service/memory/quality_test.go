package memory

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/stretchr/testify/require"
)

// This file covers the extraction-quality work: what the model is shown, how
// its answer is interpreted, and what is refused. Each test names the concrete
// failure it exists to prevent, because most of them were found by probing the
// running system rather than by reading the code.

// ---------------------------------------------------------------------------
// A memory the user deleted must not come back
// ---------------------------------------------------------------------------

// TestDeletedMemoryIsNotReExtracted is the one users would hit first. An
// explicit "remember X" is stored immediately, and the debounced distillation
// for that same turn reads the same message minutes later — so deleting the
// memory in between used to be undone by the system itself.
func TestDeletedMemoryIsNotReExtracted(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})

	stored, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Topic: "production database",
		Content: "the production database is PostgreSQL 17", Origin: types.MemoryOriginExplicit,
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteItem(ctx, stored.ID))

	messages.set("session-1", []*types.Message{
		{
			ID: "m1", SessionID: "session-1", Role: "user",
			Content: "remember: the production database is PostgreSQL 17", CreatedAt: time.Now().Add(-time.Minute),
		},
	})
	models.response = `{"memories":[{"action":"add","target":null,"kind":"fact",` +
		`"topic":"production database","content":"the production database is PostgreSQL 17","source":1}]}`
	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	_, total, err := svc.ListItems(ctx, "", 10, 0)
	require.NoError(t, err)
	require.Zero(t, total, "a memory the user deleted must not be re-extracted")
}

// TestRewordedMemoryDoesNotComeBack is the case the fingerprint alone misses:
// distillation re-reads the same message and words the statement slightly
// differently, so it hashes differently and used to slip straight through.
func TestRewordedMemoryDoesNotComeBack(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})

	stored, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Content: "our production database is PostgreSQL 17, deployed in Frankfurt",
		Origin: types.MemoryOriginExplicit, SourceMessageID: "m1", SourceSessionID: "session-1",
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteItem(ctx, stored.ID))

	messages.set("session-1", []*types.Message{
		{
			ID: "m1", SessionID: "session-1", Role: "user",
			Content:   "remember: our production database is PostgreSQL 17, deployed in Frankfurt",
			CreatedAt: time.Now().Add(-time.Minute),
		},
	})
	// Same fact, different wording: a different fingerprint.
	models.response = `{"memories":[{"action":"add","target":null,"kind":"fact",` +
		`"topic":"production database","content":"production database is PostgreSQL 17, deployed in Frankfurt","source":1}]}`
	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	_, total, err := svc.ListItems(ctx, "", 10, 0)
	require.NoError(t, err)
	require.Zero(t, total,
		"a re-worded restatement from the same rejected message must not come back")
}

// TestSayingItAgainLaterStillWorks keeps the suppression from turning into a
// permanent ban: the user asking again is not the system re-deriving.
func TestSayingItAgainLaterStillWorks(t *testing.T) {
	svc, tenantRepo, _, _, _ := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	stored, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Content: "the production database is PostgreSQL 17",
		Origin: types.MemoryOriginExplicit, SourceMessageID: "m1",
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteItem(ctx, stored.ID))

	// A later turn, a later message: this is the user asking again.
	again, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Content: "the production database really is PostgreSQL 17",
		Origin: types.MemoryOriginExplicit, SourceMessageID: "m9",
	})
	require.NoError(t, err)
	require.NotEmpty(t, again.ID)
}

func TestForgottenTopicsAreShownToTheModel(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})

	stored, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Topic: "home address", Content: "lives in Xihu District, Hangzhou",
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteItem(ctx, stored.ID))

	messages.set("session-1", []*types.Message{
		{ID: "m1", SessionID: "session-1", Role: "user", Content: "just chatting a bit", CreatedAt: time.Now()},
	})
	models.response = `{"memories":[]}`
	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	// The fingerprint check only catches an identical restatement, so the model
	// is also told which topics were rejected.
	require.Contains(t, models.lastPrompt, "home address")
	require.NotContains(t, models.lastPrompt, "lives in Xihu District, Hangzhou",
		"a tombstone must not retain the statement the user asked to forget")
}

func TestClearLeavesTombstones(t *testing.T) {
	svc, tenantRepo, _, _, _ := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	_, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Topic: "database", Content: "the production database is PostgreSQL",
	})
	require.NoError(t, err)
	_, err = svc.Clear(ctx)
	require.NoError(t, err)

	_, err = svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Topic: "database", Content: "the production database is PostgreSQL",
	})
	require.ErrorIs(t, err, ErrPreviouslyForgotten,
		"clearing is a rejection of everything, not just a bulk delete")
}

func TestForgettingDoesNotBlockGenuinelyNewInformation(t *testing.T) {
	svc, tenantRepo, _, _, _ := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	stored, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Topic: "database in use", Content: "the production database is MySQL",
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteItem(ctx, stored.ID))

	// Same topic, different statement: the user moved on, and suppressing this
	// would make deleting one memory quietly ban a subject forever.
	updated, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Topic: "database in use", Content: "the production database has moved to PostgreSQL",
	})
	require.NoError(t, err)
	require.Equal(t, "the production database has moved to PostgreSQL", updated.Content)
}

// ---------------------------------------------------------------------------
// Provenance
// ---------------------------------------------------------------------------

// TestProvenancePointsAtTheRightMessage covers the regression that arrived with
// multi-session runs: every memory used to be attributed to the turn that
// triggered the run, which could be a different conversation entirely.
func TestProvenancePointsAtTheRightMessage(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})

	base := time.Now().Add(-time.Hour)
	messages.set("session-a", []*types.Message{
		{
			ID: "msg-a1", SessionID: "session-a", Role: "user",
			Content: "I work in medical imaging", CreatedAt: base,
		},
	})
	messages.set("session-b", []*types.Message{
		{
			ID: "msg-b1", SessionID: "session-b", Role: "user",
			Content: "by the way, how is the weather", CreatedAt: base.Add(time.Second),
		},
	})
	models.response = `{"memories":[{"action":"add","target":null,"kind":"profile",` +
		`"topic":"occupation","content":"works in medical imaging","source":1}]}`

	svc.ScheduleExtraction(ctx, "session-a", "trigger-msg", "model-1")
	svc.ScheduleExtraction(ctx, "session-b", "trigger-msg", "model-1")
	drainExtractions(t, svc, enqueuer)

	items, _, err := svc.ListItems(ctx, types.MemoryStatusActive, 10, 0)
	require.NoError(t, err)
	require.NotEmpty(t, items)
	for _, item := range items {
		if item.Content != "works in medical imaging" {
			continue
		}
		require.Equal(t, "session-a", item.SourceSessionID)
		require.Equal(t, "msg-a1", item.SourceMessageID)
		return
	}
	t.Fatal("the extracted memory was not found")
}

func TestOutOfRangeSourceFallsBackInsideTheSegment(t *testing.T) {
	segment := transcriptSegment{lines: []transcriptLine{
		{sessionID: "s1", messageID: "m1", content: "first line"},
		{sessionID: "s1", messageID: "m2", content: "second line"},
	}}
	bogus := 99
	resolved := extractionDecision{Source: &bogus}.resolveSource(segment)
	require.Equal(t, "m1", resolved.messageID,
		"a hallucinated line number must still land inside the right conversation")
}

// ---------------------------------------------------------------------------
// Prior context
// ---------------------------------------------------------------------------

// TestPriorContextResolvesAReferringStatement: a run sees only what is new, so
// without a lead-in a turn like "let's just use the one from before" has nothing to resolve against.
func TestPriorContextResolvesAReferringStatement(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})
	models.response = `{"memories":[]}`

	base := time.Now().Add(-time.Hour)
	messages.set("session-1", []*types.Message{
		{
			ID: "m1", SessionID: "session-1", Role: "user",
			Content: "I am evaluating PostgreSQL and MySQL", CreatedAt: base,
		},
	})
	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	messages.set("session-1", []*types.Message{
		{
			ID: "m1", SessionID: "session-1", Role: "user",
			Content: "I am evaluating PostgreSQL and MySQL", CreatedAt: base,
		},
		{
			ID: "m2", SessionID: "session-1", Role: "user",
			Content: "let's just use the one from before", CreatedAt: base.Add(time.Minute),
		},
	})
	svc.ScheduleExtraction(ctx, "session-1", "m2", "model-1")
	drainExtractions(t, svc, enqueuer)

	require.Contains(t, models.lastPrompt, "I am evaluating",
		"a referring statement needs the turn it refers to")
	require.NotContains(t, transcriptBlock(models.lastPrompt), "I am evaluating",
		"context must not be extracted from a second time")
}

// ---------------------------------------------------------------------------
// Segmentation
// ---------------------------------------------------------------------------

// TestLongSilenceStartsANewSegment keeps one call from having to make sense of
// two unrelated situations at once.
func TestLongSilenceStartsANewSegment(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})
	models.response = `{"memories":[]}`

	base := time.Now().Add(-24 * time.Hour)
	messages.set("session-1", []*types.Message{
		{ID: "m1", SessionID: "session-1", Role: "user", Content: "what we discussed in the morning", CreatedAt: base},
		{
			ID: "m2", SessionID: "session-1", Role: "user", Content: "what we discussed in the evening",
			CreatedAt: base.Add(6 * time.Hour),
		},
	})
	svc.ScheduleExtraction(ctx, "session-1", "m2", "model-1")
	drainExtractions(t, svc, enqueuer)

	require.Equal(t, 2, models.calls, "a six-hour gap must split the run into two calls")
	seen := models.seenTranscripts()
	require.Contains(t, seen, "what we discussed in the morning")
	require.Contains(t, seen, "what we discussed in the evening")
}

func TestSeparateSessionsAreSeparateSegments(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})
	models.response = `{"memories":[]}`

	base := time.Now().Add(-time.Hour)
	messages.set("session-a", []*types.Message{
		{ID: "a1", SessionID: "session-a", Role: "user", Content: "words from session A", CreatedAt: base},
	})
	messages.set("session-b", []*types.Message{
		{
			ID: "b1", SessionID: "session-b", Role: "user", Content: "words from session B",
			CreatedAt: base.Add(time.Second),
		},
	})
	svc.ScheduleExtraction(ctx, "session-a", "a1", "model-1")
	svc.ScheduleExtraction(ctx, "session-b", "b1", "model-1")
	drainExtractions(t, svc, enqueuer)

	require.Equal(t, 2, models.calls)
	for _, prompt := range models.prompts {
		block := transcriptBlock(prompt)
		require.False(t, strings.Contains(block, "words from session A") && strings.Contains(block, "words from session B"),
			"two conversations must not be merged into one call")
	}
}

// TestSegmentCapStillCoversEverything: capping the calls one run makes must
// delay work, never lose it.
func TestSegmentCapStillCoversEverything(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})
	models.response = `{"memories":[]}`

	base := time.Now().Add(-100 * time.Hour)
	var transcript []*types.Message
	for i := 0; i < extractMaxSegmentsPerRun*2+1; i++ {
		transcript = append(transcript, &types.Message{
			ID: fmt.Sprintf("m%d", i), SessionID: "session-1", Role: "user",
			Content:   fmt.Sprintf("words from segment %d", i),
			CreatedAt: base.Add(time.Duration(i) * 3 * time.Hour),
		})
	}
	messages.set("session-1", transcript)
	svc.ScheduleExtraction(ctx, "session-1", "m0", "model-1")
	drainExtractions(t, svc, enqueuer)

	seen := models.seenTranscripts()
	for i := 0; i < extractMaxSegmentsPerRun*2+1; i++ {
		require.Contains(t, seen, fmt.Sprintf("words from segment %d", i),
			"segment %d was never read", i)
	}
}

// ---------------------------------------------------------------------------
// Decision handling
// ---------------------------------------------------------------------------

// TestEmptyActionIsIgnored: a truncated response used to be treated as add,
// which turned a broken model reply into a silent write.
func TestEmptyActionIsIgnored(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})
	messages.set("session-1", []*types.Message{
		{ID: "m1", SessionID: "session-1", Role: "user", Content: "just saying something", CreatedAt: time.Now()},
	})
	models.response = `{"memories":[{"kind":"fact","topic":"t","content":"some fact"},` +
		`{"action":"none","kind":"fact","topic":"t2","content":"another fact"}]}`

	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	_, total, err := svc.ListItems(ctx, "", 10, 0)
	require.NoError(t, err)
	require.Zero(t, total, "only an explicit add/update/delete may write")
}

// TestUpdateAddressesTheNoteByIndex is the anti-hallucination measure: a model
// that mis-types a topic would otherwise silently create a duplicate.
func TestUpdateAddressesTheNoteByIndex(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})

	_, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindFact, Topic: "database in use", Content: "the production database is MySQL",
	})
	require.NoError(t, err)

	messages.set("session-1", []*types.Message{
		{
			ID: "m1", SessionID: "session-1", Role: "user",
			Content: "we moved to PostgreSQL", CreatedAt: time.Now(),
		},
	})
	// The topic is deliberately misspelled; the index is what must win.
	models.response = `{"memories":[{"action":"delete","target":0,"kind":"fact",` +
		`"topic":"databse in use","content":"no longer uses MySQL","source":1}]}`

	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	_, active, err := svc.ListItems(ctx, types.MemoryStatusActive, 10, 0)
	require.NoError(t, err)
	require.Zero(t, active, "delete must find the note by index despite the wrong topic")
}

func TestDuplicateTopicsInOneResponseDoNotChurn(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})
	messages.set("session-1", []*types.Message{
		{ID: "m1", SessionID: "session-1", Role: "user", Content: "said it twice", CreatedAt: time.Now()},
	})
	models.response = `{"memories":[
		{"action":"add","target":null,"kind":"fact","topic":"database","content":"uses PostgreSQL","source":1},
		{"action":"add","target":null,"kind":"fact","topic":"database","content":"uses PostgreSQL 17","source":1}]}`

	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	_, superseded, err := svc.ListItems(ctx, types.MemoryStatusSuperseded, 10, 0)
	require.NoError(t, err)
	require.Zero(t, superseded, "one run must not supersede its own output")
}

// ---------------------------------------------------------------------------
// Expiry
// ---------------------------------------------------------------------------

func TestExpiredTaskLeavesTheContext(t *testing.T) {
	svc, tenantRepo, _, _, _ := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(48 * time.Hour)
	_, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindTask, Topic: "expired task", Content: "weekly report due last week",
		ExpiresAt: &past,
	})
	require.NoError(t, err)
	_, err = svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindTask, Topic: "current work", Content: "weekly report due this week",
		ExpiresAt: &future,
	})
	require.NoError(t, err)

	prompt := svc.Recall(ctx, "how is the weekly report going").Prompt
	require.Contains(t, prompt, "weekly report due this week")
	require.NotContains(t, prompt, "weekly report due last week",
		"an expired task must stop being recalled")
}

func TestParseExpiryRejectsUnusableDates(t *testing.T) {
	require.Nil(t, parseExpiry(""))
	require.Nil(t, parseExpiry("null"))
	require.Nil(t, parseExpiry("next Friday"))
	require.Nil(t, parseExpiry("2020-01-01"), "a date already past would be stored and archived at once")
	require.NotNil(t, parseExpiry(time.Now().Add(72*time.Hour).Format("2006-01-02")))
}

// ---------------------------------------------------------------------------
// Workspace instructions
// ---------------------------------------------------------------------------

func TestWorkspaceInstructionsReachThePrompt(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
		ExtractInstructions: "Never record customer names",
	})
	models.response = `{"memories":[]}`
	messages.set("session-1", []*types.Message{
		{ID: "m1", SessionID: "session-1", Role: "user", Content: "just chatting", CreatedAt: time.Now()},
	})

	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)
	require.Contains(t, models.lastPrompt, "Never record customer names")
}

// ---------------------------------------------------------------------------
// Structured output
// ---------------------------------------------------------------------------

func TestExtractionRequestsStructuredOutput(t *testing.T) {
	svc, tenantRepo, messages, models, enqueuer := newExtractionHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, ExtractDelaySeconds: 5,
	})
	models.response = `{"memories":[]}`
	messages.set("session-1", []*types.Message{
		{ID: "m1", SessionID: "session-1", Role: "user", Content: "just chatting", CreatedAt: time.Now()},
	})

	svc.ScheduleExtraction(ctx, "session-1", "m1", "model-1")
	drainExtractions(t, svc, enqueuer)

	require.NotEmpty(t, models.lastFormat,
		"the response schema must be sent, not just described in prose")
	require.Contains(t, string(models.lastFormat), "memories")
}

var _ = context.Background
