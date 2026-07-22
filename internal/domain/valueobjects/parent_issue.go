package valueobjects

import "io"

// ParentIssue is the issue this issue is a sub-issue of, sourced from GET
// /issues/{number}'s own parent_issue_url field (resolved to the parent's
// own resource via a second fetch of the same endpoint). Like
// PullRequestDiff/PullRequestCommits, it has no event of its own, so its
// attribution reuses the issue's own (author, created, url) rather than a
// per-event one. Present only when the exported ref is a plain issue that
// actually has a parent.
type ParentIssue struct {
	attribution Attribution
	parent      IssueSummary
}

// NewParentIssue constructs a ParentIssue from its attribution and the
// parent issue's summary.
func NewParentIssue(attribution Attribution, parent IssueSummary) ParentIssue {
	return ParentIssue{attribution: attribution, parent: parent}
}

// Attribution returns the issue's own author, creation time, and URL (see
// the ParentIssue Godoc for why this isn't the parent's own attribution).
func (p ParentIssue) Attribution() Attribution {
	return p.attribution
}

// Parent returns the parent issue's summary.
func (p ParentIssue) Parent() IssueSummary {
	return p.parent
}

// Equals reports whether p and other have the same attribution and parent.
func (p ParentIssue) Equals(other ParentIssue) bool {
	return p.attribution.Equals(other.attribution) &&
		p.parent.Equals(other.parent)
}

// Render writes p's <!-- {"meta":...} --> line, carrying the parent's
// number, title, state, and url. A ParentIssue has no body content beyond
// its meta line, satisfying Entry.
func (p ParentIssue) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Number int    `json:"number"`
		Title  string `json:"title"`
		State  string `json:"state"`
		URL    Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(p.attribution),
		Number:          p.parent.Number(),
		Title:           p.parent.Title(),
		State:           p.parent.State().String(),
		URL:             p.parent.URL(),
	}

	return writeMetaLine(w, meta, "")
}

func (ParentIssue) entryNode() {}
