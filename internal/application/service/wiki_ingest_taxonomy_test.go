package service

import (
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestFormatExistingTaxonomyForPrompt(t *testing.T) {
	got := formatExistingTaxonomyForPrompt([][]string{
		{"Spring Festival", "Traditional customs"},
		{"Spring Festival", "Cultural customs", "Festival customs"},
		{"Spring Festival customs"},
		{"Product positioning"},
	})
	want := "Product positioning\n" +
		"Spring Festival\n" +
		"  Cultural customs\n" +
		"    Festival customs\n" +
		"  Traditional customs\n" +
		"Spring Festival customs"
	if got != want {
		t.Fatalf("formatExistingTaxonomyForPrompt():\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatExistingTaxonomyForPromptEmpty(t *testing.T) {
	if got := formatExistingTaxonomyForPrompt(nil); got != "" {
		t.Fatalf("formatExistingTaxonomyForPrompt(nil) = %q, want empty", got)
	}
}

func TestParseTaxonomyAssignments(t *testing.T) {
	raw := "```json\n{\"assignments\":[" +
		"{\"slug\":\"entity/zhang-san\",\"path\":[\"People\"]}," +
		"{\"slug\":\"concept/spring\",\"path\":[\"Festivals\",\"Traditional festivals\"]}," +
		"{\"slug\":\"  \",\"path\":[\"X\"]}," +
		"{\"slug\":\"entity/unclassified\",\"path\":[]}" +
		"]}\n```"

	got := parseTaxonomyAssignments(raw)
	if len(got) != 3 {
		t.Fatalf("parseTaxonomyAssignments() returned %d entries, want 3 (blank slug dropped): %v", len(got), got)
	}
	if strings.Join(got["entity/zhang-san"], "/") != "People" {
		t.Fatalf("zhang-san path = %v, want [People]", got["entity/zhang-san"])
	}
	if strings.Join(got["concept/spring"], "/") != "Festivals/Traditional festivals" {
		t.Fatalf("spring path = %v, want [Festivals Traditional festivals]", got["concept/spring"])
	}
	if p, ok := got["entity/unclassified"]; !ok || len(p) != 0 {
		t.Fatalf("unclassified path = %v (ok=%v), want empty slice present", p, ok)
	}
}

func TestParseTaxonomyAssignmentsMalformed(t *testing.T) {
	if got := parseTaxonomyAssignments("not json at all"); got != nil {
		t.Fatalf("parseTaxonomyAssignments(garbage) = %v, want nil", got)
	}
}

func TestCosineSimilarity(t *testing.T) {
	if got := cosineSimilarity([]float32{1, 0}, []float32{1, 0}); got < 0.999 {
		t.Fatalf("identical vectors sim = %v, want ~1", got)
	}
	if got := cosineSimilarity([]float32{1, 0}, []float32{0, 1}); got != 0 {
		t.Fatalf("orthogonal vectors sim = %v, want 0", got)
	}
	if got := cosineSimilarity([]float32{1, 0}, nil); got != 0 {
		t.Fatalf("mismatched length sim = %v, want 0", got)
	}
	if got := cosineSimilarity([]float32{0, 0}, []float32{1, 1}); got != 0 {
		t.Fatalf("zero-norm sim = %v, want 0", got)
	}
}

func TestSelectFoldersByVectors(t *testing.T) {
	deeper := [][]string{
		{"AI", "Vendors"},       // 0
		{"AI", "Models"},        // 1
		{"Geography", "Cities"}, // 2
	}
	folderVecs := [][]float32{
		{1, 0, 0},
		{0.9, 0.1, 0},
		{0, 0, 1},
	}
	// Item closest to folder 0 (and 1 as runner-up); far from 2.
	itemVecs := [][]float32{{1, 0, 0}}

	got := selectFoldersByVectors(deeper, folderVecs, itemVecs, 2)
	if len(got) != 2 {
		t.Fatalf("selectFoldersByVectors() returned %d folders, want 2: %v", len(got), got)
	}
	// Input order preserved: folders 0 and 1, not the orthogonal 2.
	if strings.Join(got[0], "/") != "AI/Vendors" || strings.Join(got[1], "/") != "AI/Models" {
		t.Fatalf("selectFoldersByVectors() = %v, want [[AI Vendors] [AI Models]]", got)
	}
}

func TestSelectFoldersByVectorsGuards(t *testing.T) {
	if got := selectFoldersByVectors([][]string{{"a"}}, [][]float32{{1}, {2}}, [][]float32{{1}}, 1); got != nil {
		t.Fatalf("mismatched lengths should return nil, got %v", got)
	}
	if got := selectFoldersByVectors([][]string{{"a"}}, [][]float32{{1}}, nil, 1); got != nil {
		t.Fatalf("no items should return nil, got %v", got)
	}
}

func TestCollectTaxonomyItems(t *testing.T) {
	slugUpdates := map[string][]SlugUpdate{
		"entity/b": {{Type: types.WikiPageTypeEntity, Item: extractedItem{Name: "B", Description: "about B"}}},
		"entity/a": {{Type: types.WikiPageTypeConcept, Item: extractedItem{Name: "A"}}},
		"sum/x":    {{Type: "summary"}},
		"ret/y":    {{Type: "retract"}},
	}
	items := collectTaxonomyItems(slugUpdates)
	if len(items) != 2 {
		t.Fatalf("collectTaxonomyItems() = %d items, want 2 (summary/retract skipped): %+v", len(items), items)
	}
	// Deterministic slug order so chunk boundaries are stable.
	if items[0].slug != "entity/a" || items[1].slug != "entity/b" {
		t.Fatalf("collectTaxonomyItems() order = [%s, %s], want [entity/a, entity/b]", items[0].slug, items[1].slug)
	}
	if items[1].about != "about B" {
		t.Fatalf("items[1].about = %q, want %q", items[1].about, "about B")
	}
}
