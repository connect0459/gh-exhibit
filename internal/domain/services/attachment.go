package services

import "regexp"

// Attachment/Rewrite/Resolution below implement ADR-002's mandatory-
// local-download policy for GitHub `user-attachments` URLs inside already-
// rendered Markdown, independent of this package's timeline-classification
// half. Detection and rewriting run as a post-render pass over a Document's
// full output, so no Tier 1 type needs a content-mutation path of its own.

// Attachment identifies a single GitHub `user-attachments` URL referenced
// by a rendered Document. It carries the behavior that concept needs
// (Filename derivation) rather than spreading it across free functions each
// taking a bare url string.
type Attachment struct {
	url string
}

// NewAttachment constructs an Attachment from its GitHub URL.
func NewAttachment(url string) Attachment {
	return Attachment{url: url}
}

// URL returns the attachment's original GitHub URL.
func (a Attachment) URL() string {
	return a.url
}

// urlPattern matches host's user-attachments asset URLs, both bare
// (Markdown image syntax) and inside an HTML <img> tag's src attribute —
// the pattern targets the URL itself, not its surrounding syntax, so both
// forms are found by the same regexp. Both http and https are matched: a
// GitHub Enterprise Server host may be configured without TLS on an
// internal network, and rendered attachment URLs reflect whatever scheme
// that host actually used. The path segment after "assets/" is GitHub's
// own UUID, reused verbatim as the local asset's base filename. host is
// quoted so a literal `.` in it (e.g. "github.com") does not act as a
// regexp wildcard.
func urlPattern(host string) *regexp.Regexp {
	return regexp.MustCompile(`https?://` + regexp.QuoteMeta(host) + `/user-attachments/assets/[0-9A-Za-z-]+`)
}

// Detect returns the attachments referenced in markdown that point at host
// (the target repository's own host, e.g. "github.com" or a GitHub
// Enterprise Server hostname), deduplicated and in first-seen order.
func Detect(markdown []byte, host string) []Attachment {
	matches := urlPattern(host).FindAll(markdown, -1)

	seen := make(map[string]bool, len(matches))
	attachments := make([]Attachment, 0, len(matches))
	for _, m := range matches {
		url := string(m)
		if seen[url] {
			continue
		}
		seen[url] = true
		attachments = append(attachments, NewAttachment(url))
	}
	return attachments
}
