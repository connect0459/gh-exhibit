package valueobjects

import "fmt"

// ReviewState is a PullRequestReview's outcome, mirroring GitHub's REST
// "reviewed" timeline event's state field.
type ReviewState int

const (
	ReviewStateApproved ReviewState = iota
	ReviewStateChangesRequested
	ReviewStateCommented
)

// ParseReviewState parses GitHub's REST "reviewed" event's state field
// ("approved", "changes_requested", "commented") into a ReviewState. It
// returns an error for any other value.
func ParseReviewState(raw string) (ReviewState, error) {
	switch raw {
	case "approved":
		return ReviewStateApproved, nil
	case "changes_requested":
		return ReviewStateChangesRequested, nil
	case "commented":
		return ReviewStateCommented, nil
	default:
		return 0, fmt.Errorf("unrecognized review state %q", raw)
	}
}

// String returns s's GitHub API spelling (e.g. "changes_requested").
func (s ReviewState) String() string {
	switch s {
	case ReviewStateApproved:
		return "approved"
	case ReviewStateChangesRequested:
		return "changes_requested"
	case ReviewStateCommented:
		return "commented"
	default:
		return fmt.Sprintf("ReviewState(%d)", int(s))
	}
}
