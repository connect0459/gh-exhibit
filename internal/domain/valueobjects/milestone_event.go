package valueobjects

import (
	"errors"
	"fmt"
	"io"
)

// MilestoneEvent is a milestone added to or removed from an issue or pull
// request, sourced from the timeline's "milestoned"/"demilestoned" events.
// Like LabelEvent, GitHub's payload for this event kind carries no
// per-event html_url, so its attribution's url is the issue/PR's own
// html_url instead (see the services package).
type MilestoneEvent struct {
	attribution Attribution
	action      MilestoneAction
	title       string
}

// NewMilestoneEvent constructs a MilestoneEvent from its attribution,
// action, and the affected milestone's title. It returns an error if title
// is empty.
func NewMilestoneEvent(attribution Attribution, action MilestoneAction, title string) (MilestoneEvent, error) {
	if title == "" {
		return MilestoneEvent{}, errors.New("milestone event title must not be empty")
	}
	return MilestoneEvent{attribution: attribution, action: action, title: title}, nil
}

// Attribution returns who performed the milestone action and when, and its
// own URL (see the MilestoneEvent Godoc for why this isn't a per-event URL).
func (e MilestoneEvent) Attribution() Attribution {
	return e.attribution
}

// Action returns whether the milestone was added or removed.
func (e MilestoneEvent) Action() MilestoneAction {
	return e.action
}

// Title returns the affected milestone's title.
func (e MilestoneEvent) Title() string {
	return e.title
}

// Equals reports whether e and other have the same attribution, action, and
// title.
func (e MilestoneEvent) Equals(other MilestoneEvent) bool {
	return e.attribution.Equals(other.attribution) &&
		e.action == other.action &&
		e.title == other.title
}

// Render writes e's <!-- {"meta":...} --> line, followed by a plain-text
// description of the milestone action so it's visible in a rendered
// Markdown preview, not just in the hidden meta comment.
func (e MilestoneEvent) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Action    string `json:"action"`
		Milestone string `json:"milestone"`
		URL       Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(e.attribution),
		Action:          e.action.String(),
		Milestone:       e.title,
		URL:             e.attribution.URL(),
	}

	var verb string
	switch e.action {
	case MilestoneActionMilestoned:
		verb = "Milestoned"
	case MilestoneActionDemilestoned:
		verb = "Demilestoned"
	default:
		verb = e.action.String()
	}

	return writeMetaLine(w, meta, fmt.Sprintf("%s %s", verb, titleCodeSpan(e.title)))
}

func (MilestoneEvent) entryNode() {}
