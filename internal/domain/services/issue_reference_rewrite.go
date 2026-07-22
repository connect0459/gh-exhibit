package services

import (
	"bytes"
	"fmt"
)

// RewriteIssueReferences replaces each resolved reference's original
// [start, end) span in markdown with "{title} [{original text}]({url})" —
// title placed before the link rather than inside its "[...]" text, so an
// issue/PR title (arbitrary, attacker-influenceable text) is never embedded
// inside this rewrite's own link syntax; the same "untrusted text is never
// placed inside a constructed [text](url) span" precedent
// checkRunLine/changedFileLine/commitLine/issueSummaryLine already
// establish, applied here to a link this rewrite constructs itself rather
// than avoiding a link altogether. An unresolved reference's original span
// is copied through unchanged. resolutions must be given in the same
// left-to-right order DetectIssueReferences returned them in, since this
// runs as a single forward pass over markdown rather than a global
// substring replace (a bare reference's own text, e.g. "#123", is common
// enough to plausibly reappear inside content this rewrite must not touch,
// unlike an attachment URL, so Rewrite's blind strings.Replacer approach
// does not apply here).
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
			fmt.Fprintf(&buf, "%s [%s](%s)", r.title, markdown[start:end], r.url)
		} else {
			buf.Write(markdown[start:end])
		}
		cursor = end
	}
	buf.Write(markdown[cursor:])
	return buf.Bytes()
}
