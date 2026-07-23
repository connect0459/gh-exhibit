package valueobjects

import "fmt"

// IssueKind restricts a SearchCriteria's ref-kind filter to an issue or a
// pull request, distinct from GitHub's own "issue type" feature (a
// user-defined classification such as Bug/Feature), which this project does
// not filter on.
type IssueKind int

const (
	IssueKindIssue IssueKind = iota
	IssueKindPullRequest
)

// ParseIssueKind parses gh-exhibit's own filter spelling ("issue", "pr")
// into an IssueKind. It returns an error for any other value.
func ParseIssueKind(raw string) (IssueKind, error) {
	switch raw {
	case "issue":
		return IssueKindIssue, nil
	case "pr":
		return IssueKindPullRequest, nil
	default:
		return 0, fmt.Errorf("unrecognized issue kind %q", raw)
	}
}

// String returns k's gh-exhibit filter spelling (e.g. "pr").
func (k IssueKind) String() string {
	switch k {
	case IssueKindIssue:
		return "issue"
	case IssueKindPullRequest:
		return "pr"
	default:
		return fmt.Sprintf("IssueKind(%d)", int(k))
	}
}

// valid reports whether k is one of IssueKind's own defined constants,
// guarding against a value built by bypassing ParseIssueKind (e.g. a raw
// int conversion).
func (k IssueKind) valid() bool {
	switch k {
	case IssueKindIssue, IssueKindPullRequest:
		return true
	default:
		return false
	}
}
