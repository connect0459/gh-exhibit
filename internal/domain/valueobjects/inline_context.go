package valueobjects

import "errors"

// InlineContext is the (path, line, diff_hunk) triple that usually co-occurs
// on an inline PR review comment. Line is nil for a file-level comment
// (GitHub's subject_type "file"), which has no line at all rather than an
// outdated one. Outdated marks a context whose
// line was resolved from GitHub's original_line fallback because the diff
// changed and the comment's own line became null.
type InlineContext struct {
	path     string
	line     *int
	diffHunk string
	outdated bool
}

// NewInlineContext constructs an InlineContext from path, an optional line
// (nil for a file-level comment), diffHunk, and whether line was recovered
// from GitHub's original_line fallback (outdated). It returns an error if
// path is empty, if line is present but not positive, or if outdated is set
// without a line.
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

// Path returns the file path the comment is anchored to.
func (c InlineContext) Path() string {
	return c.path
}

// Line returns the diff line the comment is anchored to, or nil for a
// file-level comment.
func (c InlineContext) Line() *int {
	return c.line
}

// DiffHunk returns the GitHub-supplied diff hunk surrounding the comment.
func (c InlineContext) DiffHunk() string {
	return c.diffHunk
}

// Outdated reports whether Line was recovered from GitHub's original_line
// fallback because the diff changed after the comment was made.
func (c InlineContext) Outdated() bool {
	return c.outdated
}

// Equals reports whether c and other have the same path, line, diff hunk,
// and outdated flag.
func (c InlineContext) Equals(other InlineContext) bool {
	return c.path == other.path &&
		equalIntPointers(c.line, other.line) &&
		c.diffHunk == other.diffHunk &&
		c.outdated == other.outdated
}

func equalIntPointers(a, b *int) bool {
	return equalPointers(a, b, func(x, y int) bool { return x == y })
}
