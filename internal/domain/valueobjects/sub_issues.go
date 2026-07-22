package valueobjects

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// SubIssues is this issue's list of sub-issues, sourced from GET
// /issues/{number}/sub_issues. Like PullRequestDiff/PullRequestCommits, it
// has no event of its own, so its attribution reuses the issue's own
// (author, created, url) rather than a per-event one. Present only when the
// exported ref is a plain issue that actually has at least one sub-issue.
type SubIssues struct {
	attribution Attribution
	children    []IssueSummary
}

// NewSubIssues constructs a SubIssues from its attribution and child issue
// summaries.
func NewSubIssues(attribution Attribution, children []IssueSummary) SubIssues {
	// Cloned so a later mutation of the caller's slice can't silently
	// change this SubIssues after construction (Immutable First).
	return SubIssues{attribution: attribution, children: slices.Clone(children)}
}

// Attribution returns the issue's own author, creation time, and URL (see
// the SubIssues Godoc for why this isn't a per-child attribution).
func (s SubIssues) Attribution() Attribution {
	return s.attribution
}

// Children returns a copy of this issue's sub-issues, so mutating the
// returned slice can't affect this SubIssues (Immutable First).
func (s SubIssues) Children() []IssueSummary {
	return slices.Clone(s.children)
}

// Equals reports whether s and other have the same attribution and
// children.
func (s SubIssues) Equals(other SubIssues) bool {
	return s.attribution.Equals(other.attribution) &&
		slices.EqualFunc(s.children, other.children, IssueSummary.Equals)
}

// Render writes s's <!-- {"meta":...} --> line, then a bullet list of every
// sub-issue with its own completion status, satisfying Entry.
func (s SubIssues) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		SubIssues int `json:"sub_issues"`
		URL       Url `json:"url"`
	}{
		attributionMeta: newAttributionMeta(s.attribution),
		SubIssues:       len(s.children),
		URL:             s.attribution.URL(),
	}

	var list strings.Builder
	for _, child := range s.children {
		fmt.Fprintf(&list, "- %s\n", issueSummaryLine(child))
	}

	return writeMetaLine(w, meta, strings.TrimRight(list.String(), "\n"))
}

func (SubIssues) entryNode() {}
