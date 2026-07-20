package valueobjects

import (
	"fmt"
	"net/url"
)

// Url is an absolute http or https URL, parsed and validated once at
// construction so every holder of a Url can trust it names such a resource
// without re-checking it at each use (Parse, don't validate). A plain
// non-empty-string check does not enforce this: net/url.Parse accepts
// almost any string as a relative reference (e.g. "not-a-url" parses
// without error), so NewUrl inspects the parsed result's own IsAbs/Scheme/
// Host rather than trusting a nil error alone.
type Url struct {
	raw    string
	scheme string
	host   string
	path   string
}

// NewUrl parses raw and validates it as an absolute http or https URL. It
// returns an error if raw fails to parse, is not absolute, uses a scheme
// other than http/https, or has no host.
func NewUrl(raw string) (Url, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return Url{}, fmt.Errorf("parse url %q: %w", raw, err)
	}
	if !parsed.IsAbs() {
		return Url{}, fmt.Errorf("url %q must be absolute", raw)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return Url{}, fmt.Errorf("url %q must use http or https, got %q", raw, parsed.Scheme)
	}
	if parsed.Host == "" {
		return Url{}, fmt.Errorf("url %q must have a host", raw)
	}
	return Url{raw: raw, scheme: parsed.Scheme, host: parsed.Host, path: parsed.Path}, nil
}

// String returns url's original, unmodified form.
func (u Url) String() string {
	return u.raw
}

// Scheme returns url's scheme ("http" or "https").
func (u Url) Scheme() string {
	return u.scheme
}

// Host returns url's host (e.g. "github.com").
func (u Url) Host() string {
	return u.host
}

// Path returns url's path component (e.g. "/user-attachments/assets/abc-123").
func (u Url) Path() string {
	return u.path
}

// Equals reports whether u and other were constructed from the same raw
// URL string.
func (u Url) Equals(other Url) bool {
	return u.raw == other.raw
}

// MarshalText renders url as its raw string, so a Url-typed struct field
// marshals to JSON identically to a plain string field (the meta:{...}
// line requires byte-exact output).
func (u Url) MarshalText() ([]byte, error) {
	return []byte(u.raw), nil
}
