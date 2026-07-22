package valueobjects

import "fmt"

// MilestoneAction is a MilestoneEvent's action, mirroring GitHub's REST
// timeline's "milestoned"/"demilestoned" event kinds.
type MilestoneAction int

const (
	MilestoneActionMilestoned MilestoneAction = iota
	MilestoneActionDemilestoned
)

// ParseMilestoneAction parses GitHub's REST timeline event field
// ("milestoned", "demilestoned") into a MilestoneAction. It returns an error
// for any other value.
func ParseMilestoneAction(raw string) (MilestoneAction, error) {
	switch raw {
	case "milestoned":
		return MilestoneActionMilestoned, nil
	case "demilestoned":
		return MilestoneActionDemilestoned, nil
	default:
		return 0, fmt.Errorf("unrecognized milestone action %q", raw)
	}
}

// String returns a's GitHub API spelling ("milestoned" or "demilestoned").
func (a MilestoneAction) String() string {
	switch a {
	case MilestoneActionMilestoned:
		return "milestoned"
	case MilestoneActionDemilestoned:
		return "demilestoned"
	default:
		return fmt.Sprintf("MilestoneAction(%d)", int(a))
	}
}
