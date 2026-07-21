package valueobjects

import "fmt"

// LabelAction is a LabelEvent's action, mirroring GitHub's REST timeline's
// "labeled"/"unlabeled" event kinds.
type LabelAction int

const (
	LabelActionLabeled LabelAction = iota
	LabelActionUnlabeled
)

// ParseLabelAction parses GitHub's REST timeline event field ("labeled",
// "unlabeled") into a LabelAction. It returns an error for any other value.
func ParseLabelAction(raw string) (LabelAction, error) {
	switch raw {
	case "labeled":
		return LabelActionLabeled, nil
	case "unlabeled":
		return LabelActionUnlabeled, nil
	default:
		return 0, fmt.Errorf("unrecognized label action %q", raw)
	}
}

// String returns a's GitHub API spelling ("labeled" or "unlabeled").
func (a LabelAction) String() string {
	switch a {
	case LabelActionLabeled:
		return "labeled"
	case LabelActionUnlabeled:
		return "unlabeled"
	default:
		return fmt.Sprintf("LabelAction(%d)", int(a))
	}
}
