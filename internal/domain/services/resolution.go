package services

// Resolution is the outcome of attempting to fetch a single attachment URL:
// either the local path it was downloaded to, or a failure reason. Success
// is tracked by its own field rather than inferred from reason being
// empty — otherwise FetchFailed("") (or a zero-value Resolution) would be
// indistinguishable from Downloaded(""), and Rewrite would silently treat
// a failed fetch as a successful one, dropping the attachment reference
// instead of emitting ADR-002's placeholder. A zero Resolution is never
// constructed directly by a caller outside this package — use Downloaded
// or FetchFailed.
type Resolution struct {
	localPath string
	reason    string
	succeeded bool
}

// Downloaded builds a Resolution for a successfully fetched attachment,
// identified by the path (relative to the rendered Markdown file, per
// ADR-002's assets/ layout) it was written to.
func Downloaded(localPath string) Resolution {
	return Resolution{localPath: localPath, succeeded: true}
}

// FetchFailed builds a Resolution for an attachment that could not be
// fetched, carrying reason for the placeholder Rewrite substitutes in its
// place (ADR-002: skip, continue, note the original URL and why).
func FetchFailed(reason string) Resolution {
	return Resolution{reason: reason}
}

func (r Resolution) ok() bool {
	return r.succeeded
}
