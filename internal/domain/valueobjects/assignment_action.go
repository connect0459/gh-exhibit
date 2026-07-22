package valueobjects

import "fmt"

// AssignmentAction is an AssignmentEvent's action, mirroring GitHub's REST
// timeline's "assigned"/"unassigned" event kinds.
type AssignmentAction int

const (
	AssignmentActionAssigned AssignmentAction = iota
	AssignmentActionUnassigned
)

// ParseAssignmentAction parses GitHub's REST timeline event field
// ("assigned", "unassigned") into an AssignmentAction. It returns an error
// for any other value.
func ParseAssignmentAction(raw string) (AssignmentAction, error) {
	switch raw {
	case "assigned":
		return AssignmentActionAssigned, nil
	case "unassigned":
		return AssignmentActionUnassigned, nil
	default:
		return 0, fmt.Errorf("unrecognized assignment action %q", raw)
	}
}

// String returns a's GitHub API spelling ("assigned" or "unassigned").
func (a AssignmentAction) String() string {
	switch a {
	case AssignmentActionAssigned:
		return "assigned"
	case AssignmentActionUnassigned:
		return "unassigned"
	default:
		return fmt.Sprintf("AssignmentAction(%d)", int(a))
	}
}
