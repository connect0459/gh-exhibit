package services

import (
	"fmt"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// Resolution is the outcome of attempting to fetch a single attachment URL:
// either the local path it was downloaded to, or a failure reason. Success
// is tracked by its own field rather than inferred from reason being
// empty — otherwise FetchFailed(url, "") (or a zero-value Resolution) would
// be indistinguishable from Downloaded(url, ""), and Rewrite would silently
// treat a failed fetch as a successful one, dropping the attachment
// reference instead of emitting the failure placeholder. A zero Resolution
// is never constructed directly by a caller outside this package — use
// Downloaded or FetchFailed. url is a valueobjects.Url, not a bare string:
// every caller already holds one (an already-fetched Attachment's own URL),
// so Resolution carries that proof forward instead of re-validating it.
type Resolution struct {
	url       valueobjects.Url
	localPath string
	reason    string
	succeeded bool
}

// Downloaded builds a Resolution for a successfully fetched attachment at
// url, identified by the path (relative to the rendered Markdown file) it
// was written to.
func Downloaded(url valueobjects.Url, localPath string) Resolution {
	return Resolution{url: url, localPath: localPath, succeeded: true}
}

// FetchFailed builds a Resolution for an attachment at url that could not
// be fetched, carrying reason for the placeholder Rewrite substitutes in
// its place: skip, continue, note the original URL and why.
func FetchFailed(url valueobjects.Url, reason string) Resolution {
	return Resolution{url: url, reason: reason}
}

// Substitute returns r's Markdown replacement text: on a successful fetch,
// its local path; on a failed fetch, an inline placeholder noting r's URL
// and the failure reason, so the evidence that an attachment existed is not
// silently lost.
func (r Resolution) Substitute() string {
	if r.succeeded {
		return r.localPath
	}
	return fmt.Sprintf("%s (attachment unavailable: %s)", r.url, r.reason)
}
