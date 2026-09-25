package service

import (
	"regexp"
	"strings"
	"testing"
)

func TestSanitizeOCRText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "whitespace only",
			input: "   \n\t  ",
			want:  "",
		},
		{
			name:  "pure HTML skeleton with no text",
			input: `<html><body><div class="image"><img/></div></body></html>`,
			want:  "",
		},
		{
			name:  "HTML with only whitespace text",
			input: "<html><body>  \n  </body></html>",
			want:  "",
		},
		{
			name:  "valid markdown passes through",
			input: "# Title\n\nThis is a body paragraph with some content.\n\n| Col 1 | Col 2 |\n| --- | --- |\n| Data 1 | Data 2 |",
			want:  "# Title\n\nThis is a body paragraph with some content.\n\n| Col 1 | Col 2 |\n| --- | --- |\n| Data 1 | Data 2 |",
		},
		{
			name:  "code block wrapper stripped",
			input: "```markdown\n# Document Title\n\nThe body content goes here.\n```",
			want:  "# Document Title\n\nThe body content goes here.",
		},
		{
			name:  "html code block wrapper stripped",
			input: "```html\n<p>This is some content</p>\n```",
			want:  "This is some content",
		},
		{
			name:  "HTML document converted to markdown",
			input: "<html><body><h1>Title</h1><p>This is a long body paragraph used to test HTML to Markdown conversion.</p></body></html>",
			want:  "# Title\n\nThis is a long body paragraph used to test HTML to Markdown conversion.",
		},
		{
			name:  "known empty reply - Chinese",
			input: "无文字内容",
			want:  "",
		},
		{
			name:  "known empty reply - no text",
			input: "No text",
			want:  "",
		},
		{
			name:  "known empty reply - Chinese no text in image",
			input: "图片中没有文字",
			want:  "",
		},
		{
			name:  "plain text with minimal HTML not converted",
			input: "This is normal text, price <100 yuan.",
			want:  "This is normal text, price <100 yuan.",
		},
		{
			name:  "multiple blank lines collapsed",
			input: "Paragraph one\n\n\n\n\nParagraph two",
			want:  "Paragraph one\n\nParagraph two",
		},
		{
			name:  "HTML with substantial text content is converted",
			input: "<div><h2>Report Summary</h2><p>Revenue grew 15% year over year this quarter, and net profit reached 230 million yuan.</p><table><tr><th>Metric</th><th>Value</th></tr><tr><td>Revenue</td><td>1 billion</td></tr></table></div>",
			want:  "", // placeholder; will be checked for non-empty
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeOCRText(tt.input)

			if tt.name == "HTML with substantial text content is converted" {
				if got == "" {
					t.Errorf("sanitizeOCRText() returned empty for substantial HTML content")
				}
				if got == tt.input {
					t.Errorf("sanitizeOCRText() did not convert HTML, got original")
				}
				return
			}

			if got != tt.want {
				t.Errorf("sanitizeOCRText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStripMarkdownCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no code block",
			input: "just normal text",
			want:  "just normal text",
		},
		{
			name:  "markdown code block",
			input: "```markdown\n# Title\nContent here\n```",
			want:  "# Title\nContent here",
		},
		{
			name:  "html code block",
			input: "```html\n<p>hello</p>\n```",
			want:  "<p>hello</p>",
		},
		{
			name:  "plain code block",
			input: "```\nsome text\n```",
			want:  "some text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripMarkdownCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("stripMarkdownCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLooksLikeHTML(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "HTML document",
			input: "<html><body><p>text</p></body></html>",
			want:  true,
		},
		{
			name:  "DOCTYPE",
			input: "<!DOCTYPE html><html><body></body></html>",
			want:  true,
		},
		{
			name:  "body tag",
			input: "<body><p>content</p></body>",
			want:  true,
		},
		{
			name:  "plain markdown",
			input: "# Title\n\nSome paragraph text",
			want:  false,
		},
		{
			name:  "text with minor HTML",
			input: "This is mostly text with a <b>bold</b> word.",
			want:  false,
		},
		{
			name:  "heavy HTML tags",
			input: "<div><p><span>x</span></p></div><div><p><span>y</span></p></div>",
			want:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeHTML(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeHTML() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSanitizeOCRText_ConvertsInlineHTMLTable guards against a regression where
// an HTML <table> embedded in a markdown body survived sanitization untouched:
// the mixed content does not satisfy looksLikeHTML, so the old HTML-to-markdown
// path never ran and the raw <table> markup reached the chunker.
func TestSanitizeOCRText_ConvertsInlineHTMLTable(t *testing.T) {
	para := "The company's overall operations remained stable this quarter, and both core metrics, revenue and profit, grew year over year. " +
		"To help management quickly understand the operating results, the table below summarizes the main financial data for the reporting period, " +
		"for reference in subsequent operating analysis, budgeting and annual performance reviews. " +
		"Please judge it together with the actual business situation and never interpret any single figure out of its business context."
	tail := "All of the above data comes from the official statements audited by the finance department, and the statistical basis is consistent with the previous reporting period; " +
		"there were no changes in accounting policy or retrospective adjustments. If you have questions, please confirm with the finance department. " +
		"The finance department reserves the right of final interpretation."
	input := "# Report\n\n" + para + "\n\n" +
		`<table><tr><th>Metric</th><th>Value</th></tr>` +
		`<tr><td>Revenue</td><td>1 billion</td></tr>` +
		`<tr><td>Profit</td><td>230 million</td></tr></table>` +
		"\n\n" + tail + "\n\n"

	if looksLikeHTML(input) {
		t.Fatalf("test input unexpectedly looks like HTML; the regression test must exercise the mixed-content path")
	}

	got := sanitizeOCRText(input)

	if strings.Contains(got, "<table") {
		t.Fatalf("expected inline HTML table to be converted, got:\n%s", got)
	}
	if !strings.Contains(got, "# Report") || !strings.Contains(got, "The finance department reserves the right of final interpretation.") {
		t.Fatalf("expected surrounding markdown to be preserved, got:\n%s", got)
	}
	for _, want := range []string{"Metric", "Value", "Revenue", "1 billion", "Profit", "230 million"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected converted table to retain %q, got:\n%s", want, got)
		}
	}
	if !regexp.MustCompile(`\|[-:]{3,}\|`).MatchString(got) {
		t.Fatalf("expected a GFM table separator row, got:\n%s", got)
	}
}

func TestIsKnownEmptyReply(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"无文字内容", true},
		{"无法识别", true},
		{"no text", true},
		{"No Text", true},
		{"NO CONTENT", true},
		{"empty", true},
		{"This is normal content", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isKnownEmptyReply(tt.input)
			if got != tt.want {
				t.Errorf("isKnownEmptyReply(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
