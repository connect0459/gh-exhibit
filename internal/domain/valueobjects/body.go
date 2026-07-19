package valueobjects

import (
	"io"
	"time"
)

// Body is an issue or pull request's own body content. Unlike the other
// three Tier 1 types, it is not sourced from the timeline array; closed/
// merged timestamps come from the issue/pull resource itself (ADR-001).
type Body struct {
	attribution Attribution
	content     string
	closedAt    *time.Time
	mergedAt    *time.Time
}

func NewBody(attribution Attribution, content string, closedAt, mergedAt *time.Time) Body {
	return Body{attribution: attribution, content: content, closedAt: closedAt, mergedAt: mergedAt}
}

func (b Body) Attribution() Attribution {
	return b.attribution
}

func (b Body) Content() string {
	return b.content
}

func (b Body) ClosedAt() *time.Time {
	return b.closedAt
}

func (b Body) MergedAt() *time.Time {
	return b.mergedAt
}

func (b Body) Equals(other Body) bool {
	return b.attribution.Equals(other.attribution) &&
		b.content == other.content &&
		equalTimePointers(b.closedAt, other.closedAt) &&
		equalTimePointers(b.mergedAt, other.mergedAt)
}

func equalTimePointers(a, b *time.Time) bool {
	return equalPointers(a, b, func(x, y time.Time) bool { return x.Equal(y) })
}

func (b Body) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Closed string `json:"closed,omitempty"`
		Merged string `json:"merged,omitempty"`
		URL    string `json:"url"`
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
