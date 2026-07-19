package valueobjects

import "io"

// IssueComment is a comment on an issue or pull request, sourced from the
// timeline's "commented" event.
type IssueComment struct {
	attribution Attribution
	body        string
}

func NewIssueComment(attribution Attribution, body string) IssueComment {
	return IssueComment{attribution: attribution, body: body}
}

func (c IssueComment) Attribution() Attribution {
	return c.attribution
}

func (c IssueComment) Body() string {
	return c.body
}

func (c IssueComment) Equals(other IssueComment) bool {
	return c.attribution.Equals(other.attribution) && c.body == other.body
}

func (c IssueComment) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		URL string `json:"url"`
	}{
		attributionMeta: newAttributionMeta(c.attribution),
		URL:             c.attribution.URL(),
	}

	return writeMetaLine(w, meta, c.body)
}

func (IssueComment) entryNode() {}
