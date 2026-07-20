package services

import (
	"errors"
	"fmt"
)

// Resolution is the outcome of attempting to fetch a single attachment URL:
// either the local path it was downloaded to, or a failure reason. Success
// is tracked by its own field rather than inferred from reason being
// empty — otherwise FetchFailed(url, "") (or a zero-value Resolution) would
// be indistinguishable from Downloaded(url, ""), and Rewrite would silently
// treat a failed fetch as a successful one, dropping the attachment
// reference instead of emitting ADR-002's placeholder. A zero Resolution is
// never constructed directly by a caller outside this package — use
// Downloaded or FetchFailed.
type Resolution struct {
	url       string
	localPath string
	reason    string
	succeeded bool
}

// Downloaded builds a Resolution for a successfully fetched attachment at
// url, identified by the path (relative to the rendered Markdown file, per
// ADR-002's assets/ layout) it was written to. It returns an error if url
// is empty.
func Downloaded(url, localPath string) (Resolution, error) {
	if url == "" {
		return Resolution{}, errors.New("resolution url must not be empty")
	}
	return Resolution{url: url, localPath: localPath, succeeded: true}, nil
}

// FetchFailed builds a Resolution for an attachment at url that could not
// be fetched, carrying reason for the placeholder Rewrite substitutes in
// its place (ADR-002: skip, continue, note the original URL and why). It
// returns an error if url is empty.
func FetchFailed(url, reason string) (Resolution, error) {
	if url == "" {
		return Resolution{}, errors.New("resolution url must not be empty")
	}
	return Resolution{url: url, reason: reason}, nil
}

// Substitute returns r's Markdown replacement text: on a successful fetch,
// its local path; on a failed fetch, an inline placeholder noting r's URL
// and the failure reason, so the evidence that an attachment existed is not
// silently lost (ADR-002).
func (r Resolution) Substitute() string {
	if r.succeeded {
		return r.localPath
	}
	return fmt.Sprintf("%s (attachment unavailable: %s)", r.url, r.reason)
}
