package services

import "strings"

// Rewrite replaces every occurrence of each resolution's attachment URL in
// markdown with its Resolution.Substitute text. A URL with no corresponding
// resolution is left untouched. All substitutions run in a single pass over
// markdown via strings.Replacer, rather than one bytes.ReplaceAll scan per
// URL (which would rescan the whole buffer once per attachment —
// O(N×len(markdown)) for N attachments instead of one pass).
func Rewrite(markdown []byte, resolutions []Resolution) []byte {
	if len(resolutions) == 0 {
		return markdown
	}

	pairs := make([]string, 0, len(resolutions)*2)
	for _, res := range resolutions {
		pairs = append(pairs, res.url.String(), res.Substitute())
	}

	return []byte(strings.NewReplacer(pairs...).Replace(string(markdown)))
}
