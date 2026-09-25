package memory

import (
	"context"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/stretchr/testify/require"
)

// The point of phase two is that memory changes what gets retrieved, not only
// what the answer prompt says. These tests pin the behaviours that make that
// true, and the ones that keep it from becoming a way to assert wrong things
// about a person or to read somebody else's data.

func TestRetrievalContextCarriesWhoIsAsking(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	_, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindProfile, Content: "works on the backend for medical imaging", Importance: 4,
	})
	require.NoError(t, err)

	memCtx := svc.RetrievalContextFor(ctx)
	require.False(t, memCtx.Empty())
	require.Contains(t, memCtx.Background, "medical imaging")
	require.NotEmpty(t, memCtx.Items, "the UI has to be able to show what shaped the search")
}

func TestRetrievalContextIsEmptyWhenConditioningIsOff(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	_, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindProfile, Content: "works on the backend for medical imaging", Importance: 4,
	})
	require.NoError(t, err)

	off := false
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled:               true,
		WriteMode:             types.MemoryWriteAuto,
		RetrievalConditioning: &off,
	})

	require.True(t, svc.RetrievalContextFor(ctx).Empty())
	// The answer prompt is a separate switch, and turning off conditioning
	// must not quietly turn off memory itself.
	require.NotEmpty(t, svc.Recall(ctx, "medical imaging").Items)
}

func TestPendingMemoriesNeverReachAPrompt(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	_, err := svc.Remember(ctx, types.MemoryItem{
		Kind:       types.MemoryKindProfile,
		Content:    "possibly in charge of shift scheduling for a store chain",
		Importance: 3,
		Origin:     types.MemoryOriginExtracted,
		Inferred:   true,
	})
	require.NoError(t, err)

	require.Empty(t, svc.Recall(ctx, "receiving process").Items,
		"a guess about the user must not be asserted before they confirm it")
	require.True(t, svc.RetrievalContextFor(ctx).Empty(),
		"an unconfirmed guess must not steer retrieval either")

	items, total, err := svc.ListItems(ctx, types.MemoryStatusPending, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total, "but it must be visible so the user can decide")

	confirmed, err := svc.ConfirmItem(ctx, items[0].ID)
	require.NoError(t, err)
	require.Equal(t, types.MemoryStatusActive, confirmed.Status)
	require.NotEmpty(t, svc.Recall(ctx, "receiving process").Items)
}

func TestRejectingAGuessStopsItComingBack(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	stored, err := svc.Remember(ctx, types.MemoryItem{
		Kind:       types.MemoryKindProfile,
		Content:    "possibly in charge of shift scheduling for a store chain",
		Importance: 3,
		Origin:     types.MemoryOriginExtracted,
		Inferred:   true,
	})
	require.NoError(t, err)
	require.NoError(t, svc.RejectItem(ctx, stored.ID))

	_, err = svc.Remember(ctx, types.MemoryItem{
		Kind:       types.MemoryKindProfile,
		Content:    "possibly in charge of shift scheduling for a store chain",
		Importance: 3,
		Origin:     types.MemoryOriginExtracted,
		Inferred:   true,
	})
	require.ErrorIs(t, err, ErrPreviouslyForgotten,
		"declining a guess has to be remembered, or the same guess returns next week")
}

func TestATopicBecomesAnInterestOnlyWhenItRecurs(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, InterestThreshold: 3,
	})

	require.Empty(t, svc.ObserveQuestionTopics(ctx, []string{"store shift scheduling"}),
		"one question is a passing curiosity, not a fact about the person")
	require.Empty(t, svc.ObserveQuestionTopics(ctx, []string{"store shift scheduling"}))

	promoted := svc.ObserveQuestionTopics(ctx, []string{"store shift scheduling"})
	require.Equal(t, []string{"store shift scheduling"}, promoted,
		"the same subject across conversations is a signal worth keeping")

	memCtx := svc.RetrievalContextFor(ctx)
	require.Contains(t, memCtx.Interests, "store shift scheduling")

	require.Empty(t, svc.ObserveQuestionTopics(ctx, []string{"store shift scheduling"}),
		"and it is promoted once, not on every question thereafter")
}

func TestInterestsDoNotCrossBetweenPeople(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	alice := enabledCtx(t, tenantRepo, 1, "alice")
	bob := enabledCtx(t, tenantRepo, 1, "bob")
	tenantRepo.set(1, &types.MemoryConfig{
		Enabled: true, WriteMode: types.MemoryWriteAuto, InterestThreshold: 2,
	})

	svc.ObserveQuestionTopics(alice, []string{"medical image segmentation"})
	require.Empty(t, svc.ObserveQuestionTopics(bob, []string{"medical image segmentation"}),
		"bob asking once must not inherit alice's count")
	require.Empty(t, svc.RetrievalContextFor(bob).Interests)
}

func TestDocumentAffinityGrowsWithUseAndStaysPerPerson(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	alice := enabledCtx(t, tenantRepo, 1, "alice")
	bob := enabledCtx(t, tenantRepo, 1, "bob")

	refs := []types.MemoryDocAffinity{{KnowledgeID: "doc-1", Title: "Segmentation model tuning handbook"}}
	svc.RecordAnswerSources(alice, refs)
	svc.RecordAnswerSources(alice, refs)

	require.Equal(t, map[string]int{"doc-1": 2}, svc.DocumentAffinity(alice, []string{"doc-1"}))
	require.Empty(t, svc.DocumentAffinity(bob, []string{"doc-1"}),
		"what alice reads must not reorder bob's results")

	// Two sightings is a habit worth telling the rewriter about; one is not.
	require.Contains(t, svc.RetrievalContextFor(alice).Documents, "Segmentation model tuning handbook")
}

func TestMemoryIsNotSharedAcrossWorkspaces(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	first := enabledCtx(t, tenantRepo, 1, "alice")
	second := enabledCtx(t, tenantRepo, 2, "alice")

	svc.RecordAnswerSources(first, []types.MemoryDocAffinity{
		{KnowledgeID: "doc-1", Title: "Internal pricing notes"},
	})
	require.Empty(t, svc.DocumentAffinity(second, []string{"doc-1"}))
	require.Empty(t, svc.RetrievalContextFor(second).Documents)
}

func TestStaleTasksAreDemotedRatherThanDeleted(t *testing.T) {
	svc, db, tenantRepo := newMemoryHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	stored, err := svc.Remember(ctx, types.MemoryItem{
		Kind: types.MemoryKindTask, Content: "Refactor the payment flow, planned to finish this week", Importance: 4,
	})
	require.NoError(t, err)

	old := stored.ValidFrom.Add(-staleTaskAge - staleTaskAge)
	require.NoError(t, db.Model(&types.MemoryItem{}).
		Where("id = ?", stored.ID).
		Updates(map[string]interface{}{"valid_from": old, "last_used_at": nil}).Error)

	scope := scopeFor(t, ctx)
	items, _, err := svc.repo.ListItems(ctx, scope, types.MemoryStatusActive, 50, 0)
	require.NoError(t, err)
	require.Equal(t, 1, svc.demoteStaleTasks(ctx, scope, items))

	after, err := svc.repo.GetItem(ctx, scope, stored.ID)
	require.NoError(t, err)
	require.Equal(t, 1, after.Importance)
	require.Equal(t, types.MemoryStatusActive, after.Status,
		"the user never said they finished it, so it is demoted rather than deleted")
}

func TestOnlySimilarMemoriesAreEverMerged(t *testing.T) {
	// Merging two memories that merely looked alike destroys something the
	// user actually told us. A missed merge only leaves the store redundant,
	// so the threshold is deliberately lopsided.
	items := []*types.MemoryItem{
		{ID: "a", Kind: types.MemoryKindPreference, Topic: "回答风格", Content: "回答直接给结论不要铺垫"},
		{ID: "b", Kind: types.MemoryKindPreference, Topic: "回答风格", Content: "回答直接给结论不用铺垫"},
		{ID: "c", Kind: types.MemoryKindPreference, Topic: "输出语言", Content: "始终使用中文回复我"},
	}
	clusters := clusterSimilar(items)
	require.Len(t, clusters, 1)
	require.Len(t, clusters[0], 2)
	require.ElementsMatch(t, []string{"a", "b"},
		[]string{clusters[0][0].ID, clusters[0][1].ID})
}

func TestMemoriesOfDifferentKindsAreNeverMerged(t *testing.T) {
	items := []*types.MemoryItem{
		{ID: "a", Kind: types.MemoryKindTask, Topic: "payment refactor", Content: "refactor the payment flow this week"},
		{ID: "b", Kind: types.MemoryKindFact, Topic: "payment refactor", Content: "refactor the payment flow this week"},
	}
	require.Empty(t, clusterSimilar(items),
		"what someone is doing and what is true of their system are different claims")
}

func scopeFor(t *testing.T, ctx context.Context) interfaces.MemoryScope {
	t.Helper()
	scope, err := ResolveScope(ctx)
	require.NoError(t, err)
	return scope
}

// The same guess arriving twice must not stack up two copies in the review
// list. Deduplication originally looked only at active memories, so every
// re-derivation of an inference added another row the user had to decline
// separately.
func TestARepeatedGuessDoesNotStackUpInTheInbox(t *testing.T) {
	svc, _, tenantRepo := newMemoryHarness(t)
	ctx := enabledCtx(t, tenantRepo, 1, "alice")

	guess := types.MemoryItem{
		Kind:       types.MemoryKindProfile,
		Topic:      "possible role",
		Content:    "possibly in charge of shift scheduling for a store chain",
		Importance: 2,
		Origin:     types.MemoryOriginExtracted,
		Inferred:   true,
	}
	first, err := svc.Remember(ctx, guess)
	require.NoError(t, err)
	second, err := svc.Remember(ctx, guess)
	require.NoError(t, err)
	require.Equal(t, first.ID, second.ID)

	_, total, err := svc.ListItems(ctx, types.MemoryStatusPending, 10, 0)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
}
