// Package github implements repositories.EvidenceFetcher (internal/domain/
// repositories) against GitHub's REST API via go-gh.
package github

import (
	"fmt"
	"net/http"
	"net/url"
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

// requestHost returns the host actually used for resp's own request, or ""
// if unavailable. This is the trusted reference a subsequent "next" page
// URL is checked against, since it reflects where the request genuinely
// went rather than a value gh-exhibit only asked for.
func requestHost(resp *http.Response) string {
	if resp.Request == nil || resp.Request.URL == nil {
		return ""
	}
	return resp.Request.URL.Host
}

// validatePaginationHost rejects a next-page URL whose host does not match
// expectedHost (the host the current page was actually fetched from). A
// paginated GitHub endpoint's Link header always names the same host across
// every page in legitimate use; a mismatch means either a malformed
// response or a server (compromised, misconfigured, or sitting behind a
// broken proxy) trying to redirect gh-exhibit's next request somewhere
// else. expectedHost being unknown (e.g. requestHost couldn't determine it)
// is treated as a mismatch too, failing closed rather than trusting an
// unverified destination.
func validatePaginationHost(nextURL, expectedHost string) error {
	parsed, err := url.Parse(nextURL)
	if err != nil {
		return fmt.Errorf("parse next-page URL %q: %w", nextURL, err)
	}
	if expectedHost == "" || parsed.Host != expectedHost {
		return fmt.Errorf("next-page URL %q host %q does not match the expected host %q", nextURL, parsed.Host, expectedHost)
	}
	return nil
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
