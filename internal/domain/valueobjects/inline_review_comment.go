package valueobjects

import (
	"fmt"
	"io"
	"strings"
)

// InlineReviewComment is a file/line-anchored comment on a pull request
// review, sourced from GET /pulls/{number}/comments and joined to its
// parent PullRequestReview via pull_request_review_id (ADR-001/ADR-002) —
// it is not classified out of the timeline array.
type InlineReviewComment struct {
	attribution Attribution
	context     InlineContext
	body        string
}

// NewInlineReviewComment constructs an InlineReviewComment from its
// attribution, file/line context, and body text.
func NewInlineReviewComment(attribution Attribution, context InlineContext, body string) InlineReviewComment {
	return InlineReviewComment{attribution: attribution, context: context, body: body}
}

// Attribution returns who authored the comment and when, and its source URL.
func (c InlineReviewComment) Attribution() Attribution {
	return c.attribution
}

// Context returns the file/line the comment is anchored to.
func (c InlineReviewComment) Context() InlineContext {
	return c.context
}

// Body returns the comment's raw Markdown content.
func (c InlineReviewComment) Body() string {
	return c.body
}

// Equals reports whether c and other have the same attribution, context,
// and body.
func (c InlineReviewComment) Equals(other InlineReviewComment) bool {
	return c.attribution.Equals(other.attribution) &&
		c.context.Equals(other.context) &&
		c.body == other.body
}

// Render writes c's meta:{...} line, its body, and — when present — its
// diff hunk under a "**Diff:**" label, satisfying Entry.
func (c InlineReviewComment) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Path     string `json:"path"`
		Line     *int   `json:"line,omitempty"`
		Outdated bool   `json:"outdated,omitempty"`
		URL      string `json:"url"`
	}{
		attributionMeta: newAttributionMeta(c.attribution),
		Path:            c.context.Path(),
		Line:            c.context.Line(),
		Outdated:        c.context.Outdated(),
		URL:             c.attribution.URL(),
	}

	if err := writeMetaLine(w, meta, c.body); err != nil {
		return err
	}

	// The diff hunk is GitHub-supplied context, not part of the human's
	// comment; labeling it avoids the two being mistaken for one another.
	hunk := c.context.DiffHunk()
	if hunk == "" {
		return nil
	}
	fence := diffFence(hunk)
	_, err := fmt.Fprintf(w, "\n**Diff:**\n\n%sdiff\n%s\n%s\n", fence, hunk, fence)
	return err
}

// diffFence returns a backtick fence one character longer than the longest
// run of backticks in content, the CommonMark-standard way of keeping a
// fenced code block from ending early when the fenced content itself
// contains a backtick run as long as (or longer than) the fence.
func diffFence(content string) string {
	longest, current := 0, 0
	for _, r := range content {
		if r == '`' {
			current++
			if current > longest {
				longest = current
			}
		} else {
			current = 0
		}
	}
	n := longest + 1
	if n < 3 {
		n = 3
	}
	return strings.Repeat("`", n)
}

func (InlineReviewComment) entryNode() {}
