package services

import "strings"

// Rewrite replaces every occurrence of a resolved attachment URL in
// markdown with its Resolution.Substitute text. A URL with no entry in
// resolutions is left untouched. All substitutions run in a single pass
// over markdown via strings.Replacer, rather than one bytes.ReplaceAll scan
// per URL (which would rescan the whole buffer once per attachment —
// O(N×len(markdown)) for N attachments instead of one pass).
func Rewrite(markdown []byte, resolutions map[string]Resolution) []byte {
	if len(resolutions) == 0 {
		return markdown
	}

	pairs := make([]string, 0, len(resolutions)*2)
	for url, res := range resolutions {
		pairs = append(pairs, url, res.Substitute(url))
	}

	return []byte(strings.NewReplacer(pairs...).Replace(string(markdown)))
}
