package valueobjects

import (
	"fmt"
	"io"
)

// ClosureEvent is an issue or pull request being closed or reopened, sourced
// from the timeline's "closed"/"reopened" events. Like LabelEvent, GitHub's
// payload for this event kind carries no per-event html_url, so its
// attribution's url is the issue/PR's own html_url instead (see the services
// package). reason mirrors GitHub's state_reason field and is only ever
// populated for a "closed" action ("completed" or "not_planned"); a
// "reopened" action always carries an empty reason.
type ClosureEvent struct {
	attribution Attribution
	action      ClosureAction
	reason      string
}

// NewClosureEvent constructs a ClosureEvent from its attribution, action,
// and reason (empty for a "reopened" action, or when GitHub reports no
// state_reason for a "closed" one). A non-empty reason passed alongside a
// "reopened" action is normalized to empty, so Reason() always honors the
// invariant this type's own Godoc states regardless of what a caller
// happens to pass in.
func NewClosureEvent(attribution Attribution, action ClosureAction, reason string) ClosureEvent {
	if action != ClosureActionClosed {
		reason = ""
	}
	return ClosureEvent{attribution: attribution, action: action, reason: reason}
}

// Attribution returns who closed or reopened the issue/PR and when, and its
// own URL (see the ClosureEvent Godoc for why this isn't a per-event URL).
func (e ClosureEvent) Attribution() Attribution {
	return e.attribution
}

// Action returns whether the issue/PR was closed or reopened.
func (e ClosureEvent) Action() ClosureAction {
	return e.action
}

// Reason returns the closure's state_reason ("completed" or "not_planned"),
// or an empty string when GitHub reported none or the action is "reopened".
func (e ClosureEvent) Reason() string {
	return e.reason
}

// Equals reports whether e and other have the same attribution, action, and
// reason.
func (e ClosureEvent) Equals(other ClosureEvent) bool {
	return e.attribution.Equals(other.attribution) &&
		e.action == other.action &&
		e.reason == other.reason
}

// Render writes e's <!-- {"meta":...} --> line, followed by a plain-text
// description of the closure action so it's visible in a rendered Markdown
// preview, not just in the hidden meta comment.
func (e ClosureEvent) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Action string `json:"action"`
		Reason string `json:"reason"`
		URL    Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(e.attribution),
		Action:          e.action.String(),
		Reason:          e.reason,
		URL:             e.attribution.URL(),
	}

	var body string
	switch {
	case e.action == ClosureActionReopened:
		body = "Reopened"
	case e.action != ClosureActionClosed:
		body = e.action.String()
	case e.reason != "":
		body = fmt.Sprintf("Closed (%s)", e.reason)
	default:
		body = "Closed"
	}

	return writeMetaLine(w, meta, body)
}

func (ClosureEvent) entryNode() {}
