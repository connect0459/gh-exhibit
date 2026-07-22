package valueobjects

import "fmt"

// CheckOutcome is a check run's displayable outcome, unifying GitHub's REST
// check-run "status" field (queued, in_progress, completed) with its
// "conclusion" field (populated only once status is completed) into a
// single value: a run not yet completed reports its status, a completed one
// reports its conclusion.
type CheckOutcome int

const (
	CheckOutcomeQueued CheckOutcome = iota
	CheckOutcomeInProgress
	CheckOutcomeSuccess
	CheckOutcomeFailure
	CheckOutcomeNeutral
	CheckOutcomeCancelled
	CheckOutcomeSkipped
	CheckOutcomeTimedOut
	CheckOutcomeActionRequired
	CheckOutcomeStale
)

// ParseCheckOutcome parses GitHub's REST check-run status ("queued",
// "in_progress", "completed") and, when status is "completed", its
// conclusion ("success", "failure", "neutral", "cancelled", "skipped",
// "timed_out", "action_required", "stale") into a CheckOutcome. It returns
// an error for an unrecognized status, an unrecognized conclusion, or a
// "completed" status paired with an empty conclusion.
func ParseCheckOutcome(status, conclusion string) (CheckOutcome, error) {
	switch status {
	case "queued":
		return CheckOutcomeQueued, nil
	case "in_progress":
		return CheckOutcomeInProgress, nil
	case "completed":
		return parseCheckConclusion(conclusion)
	default:
		return 0, fmt.Errorf("unrecognized check run status %q", status)
	}
}

func parseCheckConclusion(conclusion string) (CheckOutcome, error) {
	switch conclusion {
	case "success":
		return CheckOutcomeSuccess, nil
	case "failure":
		return CheckOutcomeFailure, nil
	case "neutral":
		return CheckOutcomeNeutral, nil
	case "cancelled":
		return CheckOutcomeCancelled, nil
	case "skipped":
		return CheckOutcomeSkipped, nil
	case "timed_out":
		return CheckOutcomeTimedOut, nil
	case "action_required":
		return CheckOutcomeActionRequired, nil
	case "stale":
		return CheckOutcomeStale, nil
	default:
		return 0, fmt.Errorf("unrecognized check run conclusion %q", conclusion)
	}
}

// String returns o's GitHub API spelling (its status spelling when not yet
// completed, its conclusion spelling once completed).
func (o CheckOutcome) String() string {
	switch o {
	case CheckOutcomeQueued:
		return "queued"
	case CheckOutcomeInProgress:
		return "in_progress"
	case CheckOutcomeSuccess:
		return "success"
	case CheckOutcomeFailure:
		return "failure"
	case CheckOutcomeNeutral:
		return "neutral"
	case CheckOutcomeCancelled:
		return "cancelled"
	case CheckOutcomeSkipped:
		return "skipped"
	case CheckOutcomeTimedOut:
		return "timed_out"
	case CheckOutcomeActionRequired:
		return "action_required"
	case CheckOutcomeStale:
		return "stale"
	default:
		return fmt.Sprintf("CheckOutcome(%d)", int(o))
	}
}
