package docparser

import (
	"strings"
	"testing"
)

func TestNormalizeHTMLTables_ConvertsStyledTableToMarkdown(t *testing.T) {
	// Mirrors PaddleOCR-VL output: a data table where every cell carries a
	// text-align style that wastes tokens.
	input := `# Report

<table><tr><td style="text-align:center;">Metric</td><td style="text-align:center;">Value</td></tr>` +
		`<tr><td style="text-align:center;">Revenue</td><td style="text-align:right;">1B</td></tr>` +
		`<tr><td style="text-align:center;">Profit</td><td style="text-align:right;">230M</td></tr></table>

The end.`

	got := NormalizeHTMLTables(input)

	if strings.Contains(got, "<table") {
		t.Fatalf("expected HTML table to be converted away, got:\n%s", got)
	}
	if strings.Contains(got, "text-align") {
		t.Fatalf("expected style attributes removed, got:\n%s", got)
	}
	if !markdownTableSeparatorPattern.MatchString(got) {
		t.Fatalf("expected a Markdown table separator row, got:\n%s", got)
	}
	for _, want := range []string{"Metric", "Value", "Revenue", "1B", "Profit", "230M"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected converted table to retain %q, got:\n%s", want, got)
		}
	}
	if !strings.Contains(got, "# Report") || !strings.Contains(got, "The end.") {
		t.Fatalf("expected surrounding markdown to be preserved, got:\n%s", got)
	}
}

func TestNormalizeHTMLTables_NoTableUnchanged(t *testing.T) {
	input := "# Title\n\nA plain paragraph without a table.\n\n| a | b |\n| --- | --- |\n| 1 | 2 |"
	if got := NormalizeHTMLTables(input); got != input {
		t.Fatalf("expected content without HTML tables to be unchanged, got:\n%s", got)
	}
}

func TestNormalizeHTMLTables_SpanValueOneIsConvertible(t *testing.T) {
	// rowspan=1 / colspan=1 merge nothing, so the table must still become GFM.
	// Covers quoted, unquoted and space-padded attribute forms.
	input := `<table><tr>` +
		`<td rowspan="1" colspan="1" style="text-align:center;">Metric</td>` +
		`<td rowspan=1 colspan=1>Value</td></tr>` +
		`<tr><td rowspan="1" colspan='1'>Revenue</td>` +
		`<td rowspan = 1 colspan = 1>1B</td></tr></table>`

	got := NormalizeHTMLTables(input)

	if strings.Contains(got, "<table") {
		t.Fatalf("expected span=1 table to be converted away, got:\n%s", got)
	}
	if !markdownTableSeparatorPattern.MatchString(got) {
		t.Fatalf("expected a Markdown table separator row, got:\n%s", got)
	}
	for _, want := range []string{"Metric", "Value", "Revenue", "1B"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected converted table to retain %q, got:\n%s", want, got)
		}
	}
}

func TestNormalizeHTMLTables_RealSpanKeepsHTMLWithRowNewlines(t *testing.T) {
	// Genuine merges (colspan>1 / rowspan>1) cannot be GFM, so the table stays
	// HTML — but each row must start on its own line so the chunker can split it.
	input := `<table><tr><td colspan="6" style="text-align:center;">Total</td></tr>` +
		`<tr><td rowspan="3">A</td><td>1</td></tr>` +
		`<tr><td rowspan = 2>B</td><td>2</td></tr></table>`

	got := NormalizeHTMLTables(input)

	if !strings.Contains(got, "<table") {
		t.Fatalf("expected real span table to remain HTML, got:\n%s", got)
	}
	if !strings.Contains(got, `colspan="6"`) || !strings.Contains(got, `rowspan="3"`) {
		t.Fatalf("expected structural span attrs preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "\n<tr") {
		t.Fatalf("expected each <tr> to start on a new line, got:\n%q", got)
	}
	if !strings.Contains(got, "\n\n<table") {
		t.Fatalf("expected block to be padded at the head with blank lines, got:\n%q", got)
	}
	if !strings.HasSuffix(got, "</table>\n\n") {
		t.Fatalf("expected block to be padded at the tail with blank lines, got:\n%q", got)
	}
	if n := strings.Count(got, "\n<tr"); n != 3 {
		t.Fatalf("expected 3 rows each preceded by a newline, got %d in:\n%q", n, got)
	}
}

func assertHTMLRowsSplittable(t *testing.T, got string) {
	t.Helper()
	if !strings.Contains(got, "<tr") {
		t.Fatalf("expected remaining HTML table rows, got:\n%s", got)
	}
	for i := 0; ; {
		idx := strings.Index(got[i:], "<tr")
		if idx < 0 {
			return
		}
		abs := i + idx
		if abs == 0 || got[abs-1] != '\n' {
			t.Fatalf("remaining <tr> at offset %d is not preceded by a newline:\n%s", abs, got)
		}
		i = abs + 1
	}
}

func TestNormalizeHTMLTables_StripsAttrsOnSpanTables(t *testing.T) {
	// rowspan/colspan cannot be expressed in Markdown, so the table stays HTML
	// but its presentational attributes are stripped and rows become splittable.
	input := `<table><tr>` +
		`<td colspan="2" style="text-align:center;" class="hdr">Total</td></tr>` +
		`<tr><td style="text-align:left;">A</td><td width="80">B</td></tr></table>`

	got := NormalizeHTMLTables(input)

	if !strings.Contains(got, "<table") {
		t.Fatalf("expected span table to remain HTML, got:\n%s", got)
	}
	if !strings.Contains(got, `colspan="2"`) {
		t.Fatalf("expected colspan to be preserved, got:\n%s", got)
	}
	for _, banned := range []string{"text-align", "class=", "width="} {
		if strings.Contains(got, banned) {
			t.Fatalf("expected %q to be stripped, got:\n%s", banned, got)
		}
	}
	assertHTMLRowsSplittable(t, got)
}

func TestNormalizeHTMLTables_BRCellsStaySplittableHTML(t *testing.T) {
	// <br> inside a cell makes the table plugin give up (it emits a newline).
	// The HTML fallback must still put each <tr> on its own line.
	input := `<table><tr><td>Metric<br/>Name</td><td>Value<br/>Unit</td></tr>` +
		`<tr><td>Tensile strength<br/>MPa</td><td>Pass<br/>Grade A</td></tr></table>`

	got := NormalizeHTMLTables(input)

	if !strings.Contains(got, "<table") {
		t.Fatalf("expected <br> table to remain HTML rather than flattened text, got:\n%s", got)
	}
	assertHTMLRowsSplittable(t, got)
	for _, want := range []string{"Metric", "Name", "Tensile strength", "Pass"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected fallback HTML to retain %q, got:\n%s", want, got)
		}
	}
}

func TestNormalizeHTMLTables_ListInCellFallsBackToSplittableHTML(t *testing.T) {
	input := `<table><tr><td><ul><li>a</li><li>b</li></ul></td><td>x</td></tr>` +
		`<tr><td>c</td><td>d</td></tr></table>`

	got := NormalizeHTMLTables(input)

	if !strings.Contains(got, "<table") {
		t.Fatalf("expected list-in-cell table to remain HTML, got:\n%s", got)
	}
	assertHTMLRowsSplittable(t, got)
}

func TestNormalizeHTMLTables_SingleColumnConvertsToGFM(t *testing.T) {
	input := `<table><tr><td>Contents</td></tr><tr><td>Chapter 1</td></tr><tr><td>Chapter 2</td></tr></table>`

	got := NormalizeHTMLTables(input)

	if strings.Contains(got, "<table") {
		t.Fatalf("expected single-column table to convert to GFM, got:\n%s", got)
	}
	if !markdownTableSeparatorPattern.MatchString(got) {
		t.Fatalf("expected a Markdown table separator row, got:\n%s", got)
	}
	for _, want := range []string{"Contents", "Chapter 1", "Chapter 2"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected converted table to retain %q, got:\n%s", want, got)
		}
	}
}

func TestNormalizeHTMLTables_ImgInSingleColumnBecomesGFM(t *testing.T) {
	input := `<table><tr><td><img src="local://images/profile.png" alt="profile"/></td></tr></table>`

	got := NormalizeHTMLTables(input)

	if strings.Contains(got, "<table") {
		t.Fatalf("expected single-column image table to convert to GFM, got:\n%s", got)
	}
	if !strings.Contains(got, "![profile](local://images/profile.png)") {
		t.Fatalf("expected markdown image to be preserved, got:\n%s", got)
	}
}

func TestNormalizeHTMLTables_MarkdownImageInCellNotEscaped(t *testing.T) {
	input := `<table><tr><td>![cover](local://images/cover.png)</td><td>Notes</td></tr></table>`

	got := NormalizeHTMLTables(input)

	if strings.Contains(got, `!\[`) {
		t.Fatalf("markdown image syntax escaped:\n%s", got)
	}
	if !strings.Contains(got, "![cover](local://images/cover.png)") {
		t.Fatalf("expected markdown image to survive conversion, got:\n%s", got)
	}
}

func TestNormalizeHTMLTables_CodeFenceTableLeftAlone(t *testing.T) {
	input := "Example:\n\n```html\n<table><tr><td>a</td><td>b</td></tr></table>\n```\n"

	got := NormalizeHTMLTables(input)

	if got != input {
		t.Fatalf("expected fenced HTML table to be unchanged, got:\n%s", got)
	}
}

func TestNormalizeHTMLTables_SpanTableIdempotent(t *testing.T) {
	input := `<table><tr><td colspan="6">Summary</td></tr><tr><td colspan="6">Remarks</td></tr></table>`
	once := NormalizeHTMLTables(input)
	twice := NormalizeHTMLTables(once)
	if once != twice {
		t.Fatalf("NormalizeHTMLTables is not idempotent for span tables:\n1=%q\n2=%q", once, twice)
	}
}
