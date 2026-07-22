package valueobjects

import "fmt"

// ClosureAction is a ClosureEvent's action, mirroring GitHub's REST
// timeline's "closed"/"reopened" event kinds.
type ClosureAction int

const (
	ClosureActionClosed ClosureAction = iota
	ClosureActionReopened
)

// ParseClosureAction parses GitHub's REST timeline event field ("closed",
// "reopened") into a ClosureAction. It returns an error for any other value.
func ParseClosureAction(raw string) (ClosureAction, error) {
	switch raw {
	case "closed":
		return ClosureActionClosed, nil
	case "reopened":
		return ClosureActionReopened, nil
	default:
		return 0, fmt.Errorf("unrecognized closure action %q", raw)
	}
}

// String returns a's GitHub API spelling ("closed" or "reopened").
func (a ClosureAction) String() string {
	switch a {
	case ClosureActionClosed:
		return "closed"
	case ClosureActionReopened:
		return "reopened"
	default:
		return fmt.Sprintf("ClosureAction(%d)", int(a))
	}
}
