package valueobjects

import "io"

// PullRequestReview is a top-level review left on a pull request, sourced
// from the timeline's "reviewed" event. Its inline comments, if any, are
// joined separately (see the services package), not embedded here.
type PullRequestReview struct {
	attribution Attribution
	state       ReviewState
	body        string
}

// NewPullRequestReview constructs a PullRequestReview from its attribution,
// outcome state, and body text.
func NewPullRequestReview(attribution Attribution, state ReviewState, body string) PullRequestReview {
	return PullRequestReview{attribution: attribution, state: state, body: body}
}

// Attribution returns who left the review and when, and its source URL.
func (r PullRequestReview) Attribution() Attribution {
	return r.attribution
}

// State returns the review's outcome (approved, changes requested, or
// commented).
func (r PullRequestReview) State() ReviewState {
	return r.state
}

// Body returns the review's raw Markdown summary text.
func (r PullRequestReview) Body() string {
	return r.body
}

// Equals reports whether r and other have the same attribution, state, and
// body.
func (r PullRequestReview) Equals(other PullRequestReview) bool {
	return r.attribution.Equals(other.attribution) &&
		r.state == other.state &&
		r.body == other.body
}

// Render writes r's meta:{...} line followed by its body, satisfying Entry.
func (r PullRequestReview) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		State string `json:"state"`
		URL   string `json:"url"`
	}{
		attributionMeta: newAttributionMeta(r.attribution),
		State:           r.state.String(),
		URL:             r.attribution.URL(),
	}

	return writeMetaLine(w, meta, r.body)
}

func (PullRequestReview) entryNode() {}
