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

func NewPullRequestReview(attribution Attribution, state ReviewState, body string) PullRequestReview {
	return PullRequestReview{attribution: attribution, state: state, body: body}
}

func (r PullRequestReview) Attribution() Attribution {
	return r.attribution
}

func (r PullRequestReview) State() ReviewState {
	return r.state
}

func (r PullRequestReview) Body() string {
	return r.body
}

func (r PullRequestReview) Equals(other PullRequestReview) bool {
	return r.attribution.Equals(other.attribution) &&
		r.state == other.state &&
		r.body == other.body
}

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
