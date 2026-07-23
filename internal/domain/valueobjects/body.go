package valueobjects

import (
	"io"
	"time"
)

// Body is an issue or pull request's own body content. Unlike the other
// three Tier 1 types, it is not sourced from the timeline array; closed/
// merged timestamps come from the issue/pull resource itself.
type Body struct {
	attribution Attribution
	content     string
	closedAt    *time.Time
	mergedAt    *time.Time
}

// NewBody constructs a Body from its attribution, content, and optional
// closed/merged timestamps (nil when not applicable).
func NewBody(attribution Attribution, content string, closedAt, mergedAt *time.Time) Body {
	return Body{attribution: attribution, content: content, closedAt: copyPointer(closedAt), mergedAt: copyPointer(mergedAt)}
}

// Attribution returns who authored the body and when, and its source URL.
func (b Body) Attribution() Attribution {
	return b.attribution
}

// Content returns the body's raw Markdown content.
func (b Body) Content() string {
	return b.content
}

// ClosedAt returns a defensive copy of when the issue/PR was closed, or nil
// if it is still open.
func (b Body) ClosedAt() *time.Time {
	return copyPointer(b.closedAt)
}

// MergedAt returns a defensive copy of when the pull request was merged, or
// nil if it was never merged (including for a plain issue, which has no
// merge concept).
func (b Body) MergedAt() *time.Time {
	return copyPointer(b.mergedAt)
}

// Equals reports whether b and other have the same attribution, content,
// and closed/merged timestamps.
func (b Body) Equals(other Body) bool {
	return b.attribution.Equals(other.attribution) &&
		b.content == other.content &&
		equalTimePointers(b.closedAt, other.closedAt) &&
		equalTimePointers(b.mergedAt, other.mergedAt)
}

func equalTimePointers(a, b *time.Time) bool {
	return equalPointers(a, b, func(x, y time.Time) bool { return x.Equal(y) })
}

// Render writes b's <!-- {"meta":...} --> line followed by its content,
// satisfying Entry.
func (b Body) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Closed string `json:"closed,omitempty"`
		Merged string `json:"merged,omitempty"`
		URL    Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(b.attribution),
		URL:             b.attribution.URL(),
	}
	if b.closedAt != nil {
		meta.Closed = b.closedAt.UTC().Format(time.RFC3339)
	}
	if b.mergedAt != nil {
		meta.Merged = b.mergedAt.UTC().Format(time.RFC3339)
	}

	return writeMetaLine(w, meta, b.content)
}

func (Body) entryNode() {}
