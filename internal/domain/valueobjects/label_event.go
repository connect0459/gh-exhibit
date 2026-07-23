package valueobjects

import (
	"errors"
	"fmt"
	"io"
)

// LabelEvent is a label added to or removed from an issue or pull request,
// sourced from the timeline's "labeled"/"unlabeled" events. Unlike the other
// Tier 1 types, GitHub's own event payload carries no html_url pointing at
// this specific event, so its attribution's url is the issue/PR's own
// html_url instead of a per-event permalink (see the services package).
type LabelEvent struct {
	attribution Attribution
	action      LabelAction
	name        string
	color       string
}

// NewLabelEvent constructs a LabelEvent from its attribution, action, and
// the affected label's name and color. It returns an error if name is empty.
func NewLabelEvent(attribution Attribution, action LabelAction, name, color string) (LabelEvent, error) {
	if name == "" {
		return LabelEvent{}, errors.New("label event name must not be empty")
	}
	return LabelEvent{attribution: attribution, action: action, name: name, color: color}, nil
}

// Attribution returns who performed the label action and when, and the
// issue/PR's own URL (see the LabelEvent Godoc for why this isn't a
// per-event URL).
func (e LabelEvent) Attribution() Attribution {
	return e.attribution
}

// Action returns whether the label was added or removed.
func (e LabelEvent) Action() LabelAction {
	return e.action
}

// Name returns the affected label's name.
func (e LabelEvent) Name() string {
	return e.name
}

// Color returns the affected label's color (a 6-character hex string,
// without a leading "#"), as GitHub reports it.
func (e LabelEvent) Color() string {
	return e.color
}

// Equals reports whether e and other have the same attribution, action,
// name, and color.
func (e LabelEvent) Equals(other LabelEvent) bool {
	return e.attribution.Equals(other.attribution) &&
		e.action == other.action &&
		e.name == other.name &&
		e.color == other.color
}

// Render writes e's <!-- {"meta":...} --> line, followed by a plain-text
// description of the label action so it's visible in a rendered Markdown
// preview, not just in the hidden meta comment.
func (e LabelEvent) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Action string `json:"action"`
		Label  string `json:"label"`
		Color  string `json:"color"`
		URL    Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(e.attribution),
		Action:          e.action.String(),
		Label:           e.name,
		Color:           e.color,
		URL:             e.attribution.URL(),
	}

	var verb string
	switch e.action {
	case LabelActionLabeled:
		verb = "Labeled"
	case LabelActionUnlabeled:
		verb = "Unlabeled"
	default:
		verb = e.action.String()
	}

	return writeMetaLine(w, meta, fmt.Sprintf("%s %s", verb, titleCodeSpan(e.name)))
}

func (LabelEvent) entryNode() {}
