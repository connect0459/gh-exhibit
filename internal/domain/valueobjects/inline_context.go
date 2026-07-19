package valueobjects

import "errors"

// InlineContext is the (path, line, diff_hunk) triple that usually co-occurs
// on an inline PR review comment, restoring the file/line context ADR-001
// identifies as lost by the current hand-maintained export format. Line is
// nil for a file-level comment (GitHub's subject_type "file"), which has no
// line at all rather than an outdated one. Outdated marks a context whose
// line was resolved from GitHub's original_line fallback because the diff
// changed and the comment's own line became null.
type InlineContext struct {
	path     string
	line     *int
	diffHunk string
	outdated bool
}

func NewInlineContext(path string, line *int, diffHunk string, outdated bool) (InlineContext, error) {
	if path == "" {
		return InlineContext{}, errors.New("inline context path must not be empty")
	}
	if line != nil && *line <= 0 {
		return InlineContext{}, errors.New("inline context line must be positive when present")
	}
	if line == nil && outdated {
		return InlineContext{}, errors.New("inline context cannot be outdated without a line")
	}
	return InlineContext{path: path, line: line, diffHunk: diffHunk, outdated: outdated}, nil
}

func (c InlineContext) Path() string {
	return c.path
}

func (c InlineContext) Line() *int {
	return c.line
}

func (c InlineContext) DiffHunk() string {
	return c.diffHunk
}

func (c InlineContext) Outdated() bool {
	return c.outdated
}

func (c InlineContext) Equals(other InlineContext) bool {
	return c.path == other.path &&
		equalIntPointers(c.line, other.line) &&
		c.diffHunk == other.diffHunk &&
		c.outdated == other.outdated
}

func equalIntPointers(a, b *int) bool {
	return equalPointers(a, b, func(x, y int) bool { return x == y })
}
