package services

import (
	"fmt"
	"regexp"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// Attachment/Rewrite/Resolution below implement the mandatory-local-
// download policy for GitHub `user-attachments` URLs inside already-
// rendered Markdown, independent of this package's timeline-classification
// half. Detection and rewriting run as a post-render pass over a Document's
// full output, so no Tier 1 type needs a content-mutation path of its own.

// Attachment identifies a single GitHub `user-attachments` URL referenced
// by a rendered Document. It carries the behavior that concept needs
// (Filename derivation) rather than spreading it across free functions each
// taking a bare url string.
type Attachment struct {
	url valueobjects.Url
}

// NewAttachment constructs an Attachment from its GitHub URL. It returns an
// error if rawURL is not a well-formed absolute http(s) URL (see
// valueobjects.NewUrl), or if its path does not match a GitHub
// user-attachments asset path (see attachmentPathPattern).
func NewAttachment(rawURL string) (Attachment, error) {
	url, err := valueobjects.NewUrl(rawURL)
	if err != nil {
		return Attachment{}, fmt.Errorf("attachment url: %w", err)
	}
	if !attachmentPathPattern.MatchString(url.Path()) {
		return Attachment{}, fmt.Errorf("attachment url %q must be a GitHub user-attachments asset URL", rawURL)
	}
	return Attachment{url: url}, nil
}

// URL returns the attachment's original GitHub URL.
func (a Attachment) URL() valueobjects.Url {
	return a.url
}

// attachmentPathRawPattern is a GitHub user-attachments asset path's shape,
// shared between urlPattern (host-scoped, used to find candidate URLs
// inside arbitrary markdown text) and attachmentPathPattern (host-agnostic,
// used to validate a single already-parsed URL's path in NewAttachment) —
// one definition for what this shape looks like, applied at two different
// points for two different reasons. The path segment after "assets/" is
// GitHub's own UUID, reused verbatim as the local asset's base filename.
const attachmentPathRawPattern = `/user-attachments/assets/[0-9A-Za-z-]+`

// attachmentPathPattern anchors attachmentPathRawPattern to the whole path,
// so NewAttachment rejects a URL whose path merely contains the shape as a
// substring (e.g. a longer, unrelated path) rather than matching it exactly.
var attachmentPathPattern = regexp.MustCompile(`^` + attachmentPathRawPattern + `$`)

// urlPattern matches host's user-attachments asset URLs, both bare
// (Markdown image syntax) and inside an HTML <img> tag's src attribute —
// the pattern targets the URL itself, not its surrounding syntax, so both
// forms are found by the same regexp. Both http and https are matched: a
// GitHub Enterprise Server host may be configured without TLS on an
// internal network, and rendered attachment URLs reflect whatever scheme
// that host actually used. host is quoted so a literal `.` in it (e.g.
// "github.com") does not act as a regexp wildcard.
func urlPattern(host string) *regexp.Regexp {
	return regexp.MustCompile(`https?://` + regexp.QuoteMeta(host) + attachmentPathRawPattern)
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
		// urlPattern reuses attachmentPathRawPattern (see its own comment),
		// so this can't actually fail; skipped rather than panicking,
		// matching this package's skip-and-continue handling elsewhere.
		attachment, err := NewAttachment(url)
		if err != nil {
			continue
		}
		attachments = append(attachments, attachment)
	}
	return attachments
}
