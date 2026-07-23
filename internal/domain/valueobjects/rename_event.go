package valueobjects

import (
	"errors"
	"fmt"
	"io"
)

// RenameEvent is an issue or pull request's title being changed, sourced
// from the timeline's "renamed" event. Like LabelEvent, GitHub's payload for
// this event kind carries no per-event html_url, so its attribution's url
// is the issue/PR's own html_url instead (see the services package).
type RenameEvent struct {
	attribution Attribution
	from        string
	to          string
}

// NewRenameEvent constructs a RenameEvent from its attribution and the
// title's previous (from) and new (to) values. It returns an error if
// either from or to is empty: GitHub never allows an issue/PR title to be
// empty, so either one being empty indicates a malformed payload rather
// than a genuine title.
func NewRenameEvent(attribution Attribution, from, to string) (RenameEvent, error) {
	if from == "" {
		return RenameEvent{}, errors.New("rename event from-title must not be empty")
	}
	if to == "" {
		return RenameEvent{}, errors.New("rename event to-title must not be empty")
	}
	return RenameEvent{attribution: attribution, from: from, to: to}, nil
}

// Attribution returns who renamed the issue/PR and when, and its own URL
// (see the RenameEvent Godoc for why this isn't a per-event URL).
func (e RenameEvent) Attribution() Attribution {
	return e.attribution
}

// From returns the issue/PR's title before the rename.
func (e RenameEvent) From() string {
	return e.from
}

// To returns the issue/PR's title after the rename.
func (e RenameEvent) To() string {
	return e.to
}

// Equals reports whether e and other have the same attribution, from, and
// to.
func (e RenameEvent) Equals(other RenameEvent) bool {
	return e.attribution.Equals(other.attribution) &&
		e.from == other.from &&
		e.to == other.to
}

// Render writes e's <!-- {"meta":...} --> line, followed by a plain-text
// description of the rename so it's visible in a rendered Markdown preview,
// not just in the hidden meta comment.
func (e RenameEvent) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		From string `json:"from"`
		To   string `json:"to"`
		URL  Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(e.attribution),
		From:            e.from,
		To:              e.to,
		URL:             e.attribution.URL(),
	}

	return writeMetaLine(w, meta, fmt.Sprintf("Renamed from %q to %q", e.from, e.to))
}

func (RenameEvent) entryNode() {}
