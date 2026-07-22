package services

import (
	"bytes"
	"fmt"
	"strings"
)

// RewriteIssueReferences replaces each resolved reference's original
// [start, end) span in markdown with "`{title}` [{original text}]({url})"
// — title backtick-wrapped and placed before the link rather than inside
// its "[...]" text, so an issue/PR title (arbitrary, attacker-
// influenceable text) is never embedded inside this rewrite's own link
// syntax; the same "untrusted text is never placed inside a constructed
// [text](url) span" precedent checkRunLine/changedFileLine/commitLine/
// issueSummaryLine already establish, applied here to a link this rewrite
// constructs itself rather than avoiding a link altogether. Backticks
// (rather than no delimiter, or a quote character) give title a clear
// visual boundary against the surrounding prose it is spliced into —
// without one, a title that itself starts with a bracketed tag (e.g. an
// issue titled "[Feature] ...") reads as ambiguous with this rewrite's
// own inserted text — while matching the same backtick-wrapped-untrusted-
// text convention checkRunLine/changedFileLine/commitLine/issueSummaryLine
// already use elsewhere. A backtick inside title only ends its own code
// span early; unlike "[]"/"()", it cannot affect this rewrite's own link
// destination, which is built entirely from url, never from title. An
// unresolved reference's original span is copied through unchanged.
// resolutions must be given in the same left-to-right order
// DetectIssueReferences returned them in, since this runs as a single
// forward pass over markdown rather than a global substring replace (a
// bare reference's own text, e.g. "#123", is common enough to plausibly
// reappear inside content this rewrite must not touch, unlike an
// attachment URL, so Rewrite's blind strings.Replacer approach does not
// apply here).
func RewriteIssueReferences(markdown []byte, resolutions []ResolvedIssueReference) []byte {
	if len(resolutions) == 0 {
		return markdown
	}

	var buf bytes.Buffer
	cursor := 0
	for _, r := range resolutions {
		start, end := r.reference.start, r.reference.end
		buf.Write(markdown[cursor:start])
		if r.resolved {
			fmt.Fprintf(&buf, "%s [%s](%s)", titleCodeSpan(r.title), markdown[start:end], r.url)
		} else {
			buf.Write(markdown[start:end])
		}
		cursor = end
	}
	buf.Write(markdown[cursor:])
	return buf.Bytes()
}

// titleCodeSpan returns title wrapped in an inline code span, using a
// backtick fence one character longer than the longest run of backticks
// already inside title — the same longest-run-plus-one technique
// diffFence uses for a fenced diff hunk, adapted for an inline span:
// without it, a title containing its own backtick run as long as (or
// longer than) a fixed single-backtick fence would end the span early,
// splitting the rest of title out as ordinary (uncoded) Markdown text.
// When title starts or ends with a backtick, a single space is added on
// that side: otherwise the fence's own delimiter backticks would run
// together with title's, merging into one longer, wrongly-matched
// backtick run. CommonMark strips exactly one leading and trailing space
// from a code span's content when it has both, so this padding does not
// appear in the rendered result.
func titleCodeSpan(title string) string {
	fence := strings.Repeat("`", longestBacktickRun(title)+1)
	if strings.HasPrefix(title, "`") || strings.HasSuffix(title, "`") {
		return fence + " " + title + " " + fence
	}
	return fence + title + fence
}

// longestBacktickRun returns the length of the longest run of consecutive
// backtick characters in s.
func longestBacktickRun(s string) int {
	longest, current := 0, 0
	for _, r := range s {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	return longest
}
