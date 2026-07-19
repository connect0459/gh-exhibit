// Package github implements repositories.EvidenceFetcher (internal/domain/
// repositories) against GitHub's REST API via go-gh, per ADR-002's
// full-native acquisition decision.
package github

import (
	"net/http"
	"strings"
)

// nextPageURL returns the "next" relation URL from resp's Link header (RFC
// 5988, the format GitHub's paginated REST endpoints use), or "" if there is
// no next page or the header is absent/malformed.
func nextPageURL(resp *http.Response) string {
	for _, segment := range splitLinkHeader(resp.Header.Get("Link")) {
		parts := strings.Split(segment, ";")
		if len(parts) < 2 {
			continue
		}

		url := strings.Trim(strings.TrimSpace(parts[0]), "<>")
		for _, param := range parts[1:] {
			if strings.TrimSpace(param) == `rel="next"` {
				return url
			}
		}
	}

	return ""
}

// splitLinkHeader splits a Link header into its comma-separated <url>;
// param=value entries, without splitting on a comma that appears inside a
// <...> URL — RFC 3986 allows unescaped commas in a URL's query string, so a
// naive strings.Split(header, ",") would break a URL like
// "?filter=a,b" into two bogus segments.
func splitLinkHeader(header string) []string {
	var segments []string
	depth := 0
	start := 0

	for i, r := range header {
		switch r {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				segments = append(segments, header[start:i])
				start = i + 1
			}
		}
	}
	segments = append(segments, header[start:])

	return segments
}
