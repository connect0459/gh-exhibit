package services

import (
	"bytes"
	"regexp"
)

// byteRange is a half-open [start, end) byte span within a markdown
// buffer.
type byteRange struct {
	start, end int
}

// overlapsAny reports whether [start, end) intersects any range in ranges.
func overlapsAny(start, end int, ranges []byteRange) bool {
	for _, r := range ranges {
		if start < r.end && end > r.start {
			return true
		}
	}
	return false
}

// htmlCommentPattern matches an HTML comment (e.g. a Tier 1 entry's own
// <!-- {"meta":...} --> line), non-greedy so two adjacent comments are not
// merged into one span. (?s) makes "." match a newline, since a comment
// may itself span multiple lines.
var htmlCommentPattern = regexp.MustCompile(`(?s)<!--.*?-->`)

// markdownLinkPattern matches an already-formatted markdown link or image
// ("[text](url)"/"![text](url)"), so DetectIssueReferences does not
// re-linkify a reference an author (or an earlier Tier 1 render step) has
// already wrapped in link syntax. It does not handle a link whose own text
// or destination contains a nested, unescaped "]"/")" — a known
// simplification, not a full CommonMark link parser.
var markdownLinkPattern = regexp.MustCompile(`\[[^\]\n]*\]\([^)\n]*\)`)

// inlineCodeSpanPattern matches a single-backtick-delimited inline code
// span on one line. CommonMark inline code spans may also be delimited by
// a longer, matching run of backtick characters (used when the span's own
// content contains a single backtick), which would require backreference
// support Go's RE2-based regexp engine does not have; a multi-backtick
// span is therefore only partially protected by this pattern, a known,
// accepted simplification given how rarely multi-backtick inline code
// appears in practice.
var inlineCodeSpanPattern = regexp.MustCompile("`[^`\n]*`")

// protectedRanges returns every byte range in markdown that
// DetectIssueReferences must not treat as a candidate reference: an HTML
// comment, a fenced code block (see fencedCodeBlockRanges), an inline code
// span, or an already-formatted markdown link.
func protectedRanges(markdown []byte) []byteRange {
	var ranges []byteRange
	for _, m := range htmlCommentPattern.FindAllIndex(markdown, -1) {
		ranges = append(ranges, byteRange{m[0], m[1]})
	}
	for _, m := range markdownLinkPattern.FindAllIndex(markdown, -1) {
		ranges = append(ranges, byteRange{m[0], m[1]})
	}
	for _, m := range inlineCodeSpanPattern.FindAllIndex(markdown, -1) {
		ranges = append(ranges, byteRange{m[0], m[1]})
	}
	ranges = append(ranges, fencedCodeBlockRanges(markdown)...)
	return ranges
}

// fencedCodeBlockRanges returns the byte range of every fenced code block
// in markdown (a line starting, after up to leading spaces, with 3 or more
// "`" or "~" characters, closed by a later line consisting of at least
// that many of the same fence character) — the shape a diff patch's or
// commit message's own fenced block uses (see diffFence), plus whatever
// fence style an issue/PR body or comment's own author markdown may use.
// An unterminated fence (no matching closing line before the end of
// markdown) protects through the end of markdown, rather than leaving the
// rest of a broken fence's content unprotected.
func fencedCodeBlockRanges(markdown []byte) []byteRange {
	var ranges []byteRange
	inFence := false
	var fenceChar byte
	var fenceLen int
	var blockStart int

	lineStart := 0
	for lineStart <= len(markdown) {
		rel := bytes.IndexByte(markdown[lineStart:], '\n')
		var line []byte
		var nextLineStart int
		atEOF := rel == -1
		if atEOF {
			line = markdown[lineStart:]
			nextLineStart = len(markdown)
		} else {
			line = markdown[lineStart : lineStart+rel]
			nextLineStart = lineStart + rel + 1
		}

		trimmed := bytes.TrimLeft(line, " ")
		switch {
		case !inFence:
			if ch, n, ok := fenceOpen(trimmed); ok {
				inFence = true
				fenceChar = ch
				fenceLen = n
				blockStart = lineStart
			}
		case isFenceClose(trimmed, fenceChar, fenceLen):
			ranges = append(ranges, byteRange{blockStart, nextLineStart})
			inFence = false
		}

		if atEOF {
			break
		}
		lineStart = nextLineStart
	}

	if inFence {
		ranges = append(ranges, byteRange{blockStart, len(markdown)})
	}
	return ranges
}

// fenceOpen reports whether trimmed (a line with its leading spaces
// already stripped) opens a fenced code block, returning the fence
// character and its run length.
func fenceOpen(trimmed []byte) (ch byte, n int, ok bool) {
	if len(trimmed) < 3 {
		return 0, 0, false
	}
	ch = trimmed[0]
	if ch != '`' && ch != '~' {
		return 0, 0, false
	}
	for n < len(trimmed) && trimmed[n] == ch {
		n++
	}
	if n < 3 {
		return 0, 0, false
	}
	return ch, n, true
}

// isFenceClose reports whether trimmed (a line with its leading spaces
// already stripped) closes a fence of ch repeated at least minLen times —
// CommonMark's own rule that a closing fence must be at least as long as
// its opening fence.
func isFenceClose(trimmed []byte, ch byte, minLen int) bool {
	trimmed = bytes.TrimRight(trimmed, " \r")
	if len(trimmed) < minLen {
		return false
	}
	for _, b := range trimmed {
		if b != ch {
			return false
		}
	}
	return true
}
