package services

import "regexp"

// Detect/Rewrite/Filename/Resolution below implement ADR-002's mandatory-
// local-download policy for GitHub `user-attachments` URLs inside already-
// rendered Markdown, independent of this package's timeline-classification
// half. Detection and rewriting run as a post-render pass over a Document's
// full output, so no Tier 1 type needs a content-mutation path of its own.

// urlPattern matches GitHub's user-attachments asset URLs, both bare
// (Markdown image syntax) and inside an HTML <img> tag's src attribute —
// the pattern targets the URL itself, not its surrounding syntax, so both
// forms are found by the same regexp. The path segment after "assets/" is
// GitHub's own UUID, reused verbatim as the local asset's base filename.
var urlPattern = regexp.MustCompile(`https://github\.com/user-attachments/assets/[0-9A-Za-z-]+`)

// Detect returns the attachment URLs referenced in markdown, deduplicated
// and in first-seen order.
func Detect(markdown []byte) []string {
	matches := urlPattern.FindAll(markdown, -1)

	seen := make(map[string]bool, len(matches))
	urls := make([]string, 0, len(matches))
	for _, m := range matches {
		url := string(m)
		if seen[url] {
			continue
		}
		seen[url] = true
		urls = append(urls, url)
	}
	return urls
}
