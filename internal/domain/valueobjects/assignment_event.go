package valueobjects

import (
	"errors"
	"fmt"
	"io"
)

// AssignmentEvent is an assignee added to or removed from an issue or pull
// request, sourced from the timeline's "assigned"/"unassigned" events. Like
// LabelEvent, GitHub's payload for this event kind carries no per-event
// html_url, so its attribution's url is the issue/PR's own html_url instead
// (see the services package).
type AssignmentEvent struct {
	attribution Attribution
	action      AssignmentAction
	assignee    string
}

// NewAssignmentEvent constructs an AssignmentEvent from its attribution,
// action, and the affected assignee's login. It returns an error if
// assignee is empty.
func NewAssignmentEvent(attribution Attribution, action AssignmentAction, assignee string) (AssignmentEvent, error) {
	if assignee == "" {
		return AssignmentEvent{}, errors.New("assignment event assignee must not be empty")
	}
	return AssignmentEvent{attribution: attribution, action: action, assignee: assignee}, nil
}

// Attribution returns who performed the assignment action and when, and its
// own URL (see the AssignmentEvent Godoc for why this isn't a per-event
// URL).
func (e AssignmentEvent) Attribution() Attribution {
	return e.attribution
}

// Action returns whether the assignee was added or removed.
func (e AssignmentEvent) Action() AssignmentAction {
	return e.action
}

// Assignee returns the affected assignee's GitHub login.
func (e AssignmentEvent) Assignee() string {
	return e.assignee
}

// Equals reports whether e and other have the same attribution, action, and
// assignee.
func (e AssignmentEvent) Equals(other AssignmentEvent) bool {
	return e.attribution.Equals(other.attribution) &&
		e.action == other.action &&
		e.assignee == other.assignee
}

// Render writes e's <!-- {"meta":...} --> line, followed by a plain-text
// description of the assignment action so it's visible in a rendered
// Markdown preview, not just in the hidden meta comment.
func (e AssignmentEvent) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Action   string `json:"action"`
		Assignee string `json:"assignee"`
		URL      Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(e.attribution),
		Action:          e.action.String(),
		Assignee:        e.assignee,
		URL:             e.attribution.URL(),
	}

	var verb string
	switch e.action {
	case AssignmentActionAssigned:
		verb = "Assigned"
	case AssignmentActionUnassigned:
		verb = "Unassigned"
	default:
		verb = e.action.String()
	}

	return writeMetaLine(w, meta, fmt.Sprintf("%s @%s", verb, e.assignee))
}

func (AssignmentEvent) entryNode() {}
