package reranking

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Tencent/WeKnora/internal/types"
)

func TestGetEnrichedPassageKeepsQuestionsFromEarlierRevision(t *testing.T) {
	metadata, err := json.Marshal(types.DocumentChunkMetadata{
		GeneratedQuestionsRevision: 1,
		GeneratedQuestions: []types.GeneratedQuestion{{
			ID: "old", Question: "question generated before the edit",
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	passage := EnrichedPassage(context.Background(), &types.SearchResult{
		Content:       "edited chunk body",
		ChunkMetadata: types.JSON(metadata),
	})
	if !strings.Contains(passage, "question generated before the edit") {
		t.Fatalf("earlier generated question was excluded from rerank passage: %q", passage)
	}
}

func TestGetEnrichedPassageKeepsCodeAndMathCandidates(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "code only",
			content: "```go\nfunc answer() int { return 42 }\n```",
			want:    "func answer() int { return 42 }",
		},
		{
			name:    "math only",
			content: "$$\nE = mc^2\n$$",
			want:    "E = mc^2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passage := EnrichedPassage(context.Background(), &types.SearchResult{Content: tt.content})
			if strings.TrimSpace(passage) == "" {
				t.Fatal("semantic-only candidate was removed from the rerank passage")
			}
			if !strings.Contains(passage, tt.want) {
				t.Fatalf("rerank passage %q does not preserve %q", passage, tt.want)
			}
		})
	}
}

func TestCleanPassageForRerank(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "plain text unchanged",
			input:  "This is a plain piece of text",
			expect: "This is a plain piece of text",
		},
		{
			name:   "remove markdown images",
			input:  "Before ![image caption](https://example.com/img.png) after",
			expect: "Before  after",
		},
		{
			name:   "convert markdown links to text",
			input:  "See [the official docs](https://docs.example.com) for details",
			expect: "See the official docs for details",
		},
		{
			name:   "remove standalone URLs",
			input:  "Visit https://example.com/path?q=1&b=2 for more info",
			expect: "Visit  for more info",
		},
		{
			name:   "unwrap code blocks",
			input:  "Sample code:\n```python\nprint('hello')\n```\nEnd of sample",
			expect: "Sample code:\nprint('hello')\nEnd of sample",
		},
		{
			name:   "unwrap LaTeX blocks",
			input:  "The formula is $$E=mc^2$$ where E is energy",
			expect: "The formula is E=mc^2 where E is energy",
		},
		{
			name:   "remove table separator rows and convert data rows",
			input:  "| Name | Value |\n| --- | --- |\n| A | 1 |",
			expect: "Name, Value\n\nA, 1",
		},
		{
			name:   "strip heading markers",
			input:  "## Chapter 2 Overview\n### 2.1 Background",
			expect: "Chapter 2 Overview\n2.1 Background",
		},
		{
			name:   "strip blockquote markers",
			input:  "> This is a quote\n> Second quoted line",
			expect: "This is a quote\nSecond quoted line",
		},
		{
			name:   "unwrap bold and italic",
			input:  "This is **bold** and *italic* and ***bold italic*** text",
			expect: "This is bold and italic and bold italic text",
		},
		{
			name:   "strip list markers",
			input:  "- Item one\n- Item two\n1. Ordered one\n2. Ordered two",
			expect: "Item one\nItem two\nOrdered one\nOrdered two",
		},
		{
			name:   "remove HTML tags",
			input:  "Text<br>break<div class=\"test\">content</div>end",
			expect: "Textbreakcontentend",
		},
		{
			name:   "collapse excessive newlines",
			input:  "Paragraph one\n\n\n\n\nParagraph two",
			expect: "Paragraph one\n\nParagraph two",
		},
		{
			name: "combined real-world passage",
			input: `## Product Overview

This is an **important** product. See [the product page](https://example.com/product).

![Product screenshot](images/product.png)

> User review: very easy to use

- Feature one
- Feature two

` + "```json\n{\"key\": \"value\"}\n```",
			expect: "Product Overview\n\nThis is an important product. See the product page.\n\nUser review: very easy to use\n\nFeature one\nFeature two\n\n{\"key\": \"value\"}",
		},
		{
			name:   "convert table data rows to plain text",
			input:  "| col1 | col2 | col3 |",
			expect: "col1, col2, col3",
		},
		{
			name:   "multi-row table fully converted",
			input:  "| Header1 | Header2 |\n| --- | --- |\n| data1 | data2 |\n| data3 | data4 |",
			expect: "Header1, Header2\n\ndata1, data2\ndata3, data4",
		},
		{
			name:   "table-only passage becomes empty after separator removal",
			input:  "| --- | --- |",
			expect: "",
		},
		{
			name:   "whitespace-only after cleaning",
			input:  "   \n\n   ",
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CleanPassage(tt.input)
			if got != tt.expect {
				t.Errorf("CleanPassage():\ngot:    %q\nexpect: %q", got, tt.expect)
			}
		})
	}
}
