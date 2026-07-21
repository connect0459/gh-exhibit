// Package github implements repositories.EvidenceFetcher and
// repositories.AttachmentFetcher (internal/domain/repositories) against
// GitHub's REST API via go-gh. Every implementation type here is
// unexported, so callers depend only on the repositories interfaces
// (dependency inversion), never on these concrete types.
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

		nextURL := strings.Trim(strings.TrimSpace(parts[0]), "<>")
		for _, param := range parts[1:] {
			if strings.TrimSpace(param) == `rel="next"` {
				return nextURL
			}
		}
	}

	return ""
}

// requestOrigin returns the scheme and host actually used for resp's own
// request (e.g. "https://api.github.com"), or "" if unavailable. This is
// the trusted reference a subsequent "next" page URL is checked against,
// since it reflects where the request genuinely went rather than a value
// gh-exhibit only asked for.
func requestOrigin(resp *http.Response) string {
	if resp.Request == nil || resp.Request.URL == nil {
		return ""
	}
	return resp.Request.URL.Scheme + "://" + resp.Request.URL.Host
}

// validatePaginationOrigin rejects a next-page URL whose scheme and host
// (its origin) does not match expectedOrigin (the origin the current page
// was actually fetched from). A paginated GitHub endpoint's Link header
// always names the same origin across every page in legitimate use; a
// mismatch means either a malformed response or a server (compromised,
// misconfigured, or sitting behind a broken proxy) trying to redirect
// gh-exhibit's next request somewhere else — including a same-host scheme
// downgrade (https to http), which loses transport security even though
// the host itself didn't change. expectedOrigin being unknown (e.g.
// requestOrigin couldn't determine it) is treated as a mismatch too,
// failing closed rather than trusting an unverified destination.
//
// The comparison is case-insensitive: both a URL scheme (RFC 3986) and a
// hostname (DNS, and by extension HTTP's Host) are themselves
// case-insensitive, so a next-page URL differing from expectedOrigin only
// in letter case is the same origin, not a mismatch.
func validatePaginationOrigin(nextURL, expectedOrigin string) error {
	parsed, err := url.Parse(nextURL)
	if err != nil {
		return fmt.Errorf("parse next-page URL %q: %w", nextURL, err)
	}
	origin := parsed.Scheme + "://" + parsed.Host
	if expectedOrigin == "" || !strings.EqualFold(origin, expectedOrigin) {
		return fmt.Errorf("next-page URL %q origin %q does not match the expected origin %q", nextURL, origin, expectedOrigin)
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
