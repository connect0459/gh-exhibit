package valueobjects

import "fmt"

// IssueState is an issue's open/closed state, mirroring GitHub's REST issue
// resource's own "state" field.
type IssueState int

const (
	IssueStateOpen IssueState = iota
	IssueStateClosed
)

// ParseIssueState parses GitHub's REST issue state field ("open", "closed")
// into an IssueState. It returns an error for any other value.
func ParseIssueState(raw string) (IssueState, error) {
	switch raw {
	case "open":
		return IssueStateOpen, nil
	case "closed":
		return IssueStateClosed, nil
	default:
		return 0, fmt.Errorf("unrecognized issue state %q", raw)
	}
}

// String returns s's GitHub API spelling.
func (s IssueState) String() string {
	switch s {
	case IssueStateOpen:
		return "open"
	case IssueStateClosed:
		return "closed"
	default:
		return fmt.Sprintf("IssueState(%d)", int(s))
	}
}
