package services

import (
	"fmt"
	"strings"
)

// Rewrite replaces every occurrence of a resolved attachment URL in
// markdown: a successful fetch substitutes the URL with its local path, so
// the surrounding Markdown/HTML image syntax keeps rendering; a failed
// fetch substitutes it with an inline placeholder noting the original URL
// and failure reason, so the evidence that an attachment existed is not
// silently lost (ADR-002). A URL with no entry in resolutions is left
// untouched. All substitutions run in a single pass over markdown via
// strings.Replacer, rather than one bytes.ReplaceAll scan per URL (which
// would rescan the whole buffer once per attachment — O(N×len(markdown))
// for N attachments instead of one pass).
func Rewrite(markdown []byte, resolutions map[string]Resolution) []byte {
	if len(resolutions) == 0 {
		return markdown
	}

	pairs := make([]string, 0, len(resolutions)*2)
	for url, res := range resolutions {
		var replacement string
		if res.ok() {
			replacement = res.localPath
		} else {
			replacement = fmt.Sprintf("%s (attachment unavailable: %s)", url, res.reason)
		}
		pairs = append(pairs, url, replacement)
	}

	return []byte(strings.NewReplacer(pairs...).Replace(string(markdown)))
}
