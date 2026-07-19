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

func ParseReviewState(raw string) (ReviewState, error) {
	switch raw {
	case "approved":
		return ReviewStateApproved, nil
	case "changes_requested":
		return ReviewStateChangesRequested, nil
	case "commented":
		return ReviewStateCommented, nil
	default:
		return 0, fmt.Errorf("valueobjects: unrecognized review state %q", raw)
	}
}

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
