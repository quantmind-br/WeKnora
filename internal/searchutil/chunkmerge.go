package searchutil

import (
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

// JoinChunkContent joins two current chunk bodies without relying on parser
// offsets. Exact containment is collapsed, a real suffix/prefix overlap is
// removed, and otherwise both bodies are retained with separator between
// them. The conservative fallback intentionally prefers small duplication
// over silently dropping edited content.
func JoinChunkContent(acc, next, separator string) string {
	if acc == "" {
		return next
	}
	if next == "" {
		return acc
	}
	if ContainsChunkContent(acc, next) {
		return acc
	}
	if ContainsChunkContent(next, acc) {
		return next
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)
	maxOverlap := minInt(len(accRunes), len(nextRunes))
	// Editable chunks may be much larger than parser-produced chunks. Bound
	// suffix matching so an adversarial 200 KB edit cannot turn retrieval into
	// quadratic work. Parser overlap windows are normally far below this cap;
	// larger unmatched overlap is safely retained as duplication.
	if maxOverlap > defaultSearchSpan {
		maxOverlap = defaultSearchSpan
	}
	for overlap := maxOverlap; overlap >= minOverlapRunes; overlap-- {
		if runeSlicesEqual(accRunes[len(accRunes)-overlap:], nextRunes[:overlap]) {
			return acc + string(nextRunes[overlap:])
		}
	}
	return acc + separator + next
}

// ContainsChunkContent reports whether the complete current body is safely
// represented by another body. Very short substrings are not treated as
// containment because common words and punctuation would create false drops.
func ContainsChunkContent(container, contained string) bool {
	if container == "" || contained == "" {
		return false
	}
	if container == contained {
		return true
	}
	return len([]rune(contained)) >= minOverlapRunes && strings.Contains(container, contained)
}

func runeSlicesEqual(left, right []rune) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

// Implements the common "overlap stitching" logic for chunk content here, reused by document reconstruction (reconstructContent),
// and knowledge graph content merging (graph mergeChunkContents). The chat retrieval path allows
// users to edit a Chunk, using JoinChunkContent above to avoid depending on original-text position coordinates.
//
// Historically, various places trimmed overlap using a "position-based" formula (offset = len(content) - (EndAt -
// lastEndAt), etc.), which assumes len([]rune(Content)) == EndAt-StartAt. But two kinds of
// data break this invariant, causing misaligned stitching, dropped characters, or duplication:
// 1. The parent-child splitter "re-adds table headers" for tables that got split apart; the re-added header is zero-width
// (start == end), position coordinates can't express it, so content is longer than EndAt-StartAt;
// 2. content may retain HTML entities (e.g. &#34; / &gt;), whose character count is longer than the original text span.
//
// So overlap here is instead matched "by text": search the window at the start of the next segment for the first occurrence
// of the merged text's suffix, and append from just after that position. Position info (StartAt/EndAt) is only used to estimate the search window
// size, and is no longer used for trimming.

const (
	// minOverlapRunes is the minimum suffix length that participates in matching. Too short (e.g., a table separator row |---|)
	// is prone to false matches, so it's ignored.
	minOverlapRunes = 12
	// defaultSearchSpan is the lower bound of the search window, ensuring that even when position info is missing/zero, it can
	// still detect real overlap within a certain range.
	defaultSearchSpan = 400
)

// AppendWithOverlap appends next after acc, removing the overlapping part between them.
//
// positionOverlap is the overlap amount estimated from StartAt/EndAt (lastEnd - curStart), used only to
// bound the search window size; the actual overlap is determined by text matching, which tolerates re-inserted headers and HTML entity length differences.
// If no text overlap is found, concatenate as-is (no trimming) — better to keep content than to break it.
//
// When positionOverlap <= 0, the two segments are strictly adjacent or disjoint in position, so there is no
// overlap to deduplicate. Running text matching in that case would, because of the headSlack floor of 320,
// falsely match a genuine content repeat of acc's suffix inside next's leading window (e.g. the same sentence
// appearing several times in the document) and delete the whole head of next as a re-inserted header,
// causing irreversible content loss. Concatenate directly and leave duplicate re-inserted headers to the
// caller's post-processing.
func AppendWithOverlap(acc, next string, positionOverlap int) string {
	if acc == "" {
		return next
	}
	if next == "" {
		return acc
	}
	if positionOverlap <= 0 {
		return acc + next
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)

	span := positionOverlap

	maxK := minInt(len(accRunes), len(nextRunes))
	if cap := maxInt(span*3, defaultSearchSpan); maxK > cap {
		maxK = cap
	}
	// The maximum prefix allowed to be skipped before the overlapping content (i.e., synthetic text such as re-inserted headers).
	headSlack := maxInt(span*2, 320)

	for k := maxK; k >= minOverlapRunes; k-- {
		needle := accRunes[len(accRunes)-k:]
		if pos := indexRunes(nextRunes, needle, headSlack); pos >= 0 {
			return acc + string(nextRunes[pos+k:])
		}
	}
	return acc + next
}

// AppendWithExactOverlap, when the caller has already confirmed the position coordinates are trustworthy, uses the exact overlap amount given by the coordinates
// to concatenate acc and next: verify that the last `overlap` characters of acc match the first `overlap` characters of next character-
// by-character; if equal, trim exactly, and if overlap is 0, concatenate directly.
//
// The difference from AppendWithOverlap is that this one "doesn't guess": the latter searches for the longest suffix match within the window to tolerate length
// deviations from re-inserted headers, HTML entities, etc., which can misjudge repeated periodic text (tables, logs) as overlap and cut real content.
// When the coordinates are trustworthy, the overlap amount is already known, so no search is needed.
//
// If validation fails, return ok=false, letting the caller decide whether to fall back to AppendWithOverlap.
func AppendWithExactOverlap(acc, next string, overlap int) (string, bool) {
	if acc == "" {
		return next, true
	}
	if next == "" {
		return acc, true
	}
	if overlap < 0 {
		return "", false
	}
	if overlap == 0 {
		return acc + next, true
	}

	accRunes := []rune(acc)
	nextRunes := []rune(next)
	if overlap > len(accRunes) || overlap > len(nextRunes) {
		return "", false
	}
	if !runeSlicesEqual(accRunes[len(accRunes)-overlap:], nextRunes[:overlap]) {
		return "", false
	}
	return acc + string(nextRunes[overlap:]), true
}

// MergeTextChunks sorts by StartAt (and by ChunkIndex for ties), then uses AppendWithOverlap
// to reconstruct the content of multiple chunks into complete text. gapSep is the separator used between two segments whose positions are not adjacent (i.e., have a gap)
// (e.g. "\n"); passing an empty string concatenates directly.
//
// The caller is responsible for type filtering beforehand (e.g., keeping only text chunks); this function is unaware of ChunkType.
func MergeTextChunks(chunks []*types.Chunk, gapSep string) string {
	if len(chunks) == 0 {
		return ""
	}

	sorted := make([]*types.Chunk, len(chunks))
	copy(sorted, chunks)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].StartAt == sorted[j].StartAt {
			return sorted[i].ChunkIndex < sorted[j].ChunkIndex
		}
		return sorted[i].StartAt < sorted[j].StartAt
	})

	merged := ""
	mergedEnd := -1
	for _, c := range sorted {
		if c == nil || c.Content == "" {
			continue
		}
		if merged == "" {
			merged = c.Content
			if c.EndAt > 0 {
				mergedEnd = c.EndAt
			}
			continue
		}

		// Gap / missing position info (EndAt==0): concatenated as an independent segment, with no overlap trimming.
		if c.StartAt > mergedEnd || c.EndAt == 0 {
			if gapSep != "" {
				merged += gapSep
			}
			merged += c.Content
			if c.EndAt > 0 {
				mergedEnd = c.EndAt
			}
			continue
		}

		// Partial overlap or adjacent ends: concatenated after removing overlap via text matching.
		if c.EndAt > mergedEnd {
			merged = AppendWithOverlap(merged, c.Content, mergedEnd-c.StartAt)
			mergedEnd = c.EndAt
		}
		// Otherwise it's fully covered by the previous segment and skipped.
	}

	return merged
}

// indexRunes finds the first occurrence of needle in haystack by rune index, with the start position not exceeding
// maxStart. Returns -1 if not found.
func indexRunes(haystack, needle []rune, maxStart int) int {
	if len(needle) == 0 || len(needle) > len(haystack) {
		return -1
	}
	limit := len(haystack) - len(needle)
	if maxStart < limit {
		limit = maxStart
	}
	for i := 0; i <= limit; i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
