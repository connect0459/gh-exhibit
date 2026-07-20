package valueobjects

import "io"

// IssueComment is a comment on an issue or pull request, sourced from the
// timeline's "commented" event.
type IssueComment struct {
	attribution Attribution
	body        string
}

// NewIssueComment constructs an IssueComment from its attribution and body
// text.
func NewIssueComment(attribution Attribution, body string) IssueComment {
	return IssueComment{attribution: attribution, body: body}
}

// Attribution returns who authored the comment and when, and its source URL.
func (c IssueComment) Attribution() Attribution {
	return c.attribution
}

// Body returns the comment's raw Markdown content.
func (c IssueComment) Body() string {
	return c.body
}

// Equals reports whether c and other have the same attribution and body.
func (c IssueComment) Equals(other IssueComment) bool {
	return c.attribution.Equals(other.attribution) && c.body == other.body
}

// Render writes c's meta:{...} line followed by its body, satisfying Entry.
func (c IssueComment) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		URL Url `json:"url"`
	}{
		attributionMeta: newAttributionMeta(c.attribution),
		URL:             c.attribution.URL(),
	}

	return writeMetaLine(w, meta, c.body)
}

func (IssueComment) entryNode() {}
