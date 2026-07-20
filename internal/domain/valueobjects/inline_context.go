package valueobjects

import "errors"

// InlineContext is the (path, line, diff_hunk) triple that usually co-occurs
// on an inline PR review comment. Line is nil for a file-level comment
// (GitHub's subject_type "file"), which has no line at all rather than an
// outdated one. Outdated marks a context whose
// line was resolved from GitHub's original_line fallback because the diff
// changed and the comment's own line became null. StartLine is nil unless
// the comment is anchored to a range of lines rather than a single one, in
// which case it holds the first line of that range and line holds the last.
type InlineContext struct {
	path      string
	line      *int
	startLine *int
	diffHunk  string
	outdated  bool
}

// NewInlineContext constructs an InlineContext from path, an optional line
// (nil for a file-level comment), an optional startLine (non-nil only for a
// range-anchored comment, giving the first line of the range while line
// gives the last), diffHunk, and whether line was recovered from GitHub's
// original_line fallback (outdated). It returns an error if path is empty,
// if line is present but not positive, if outdated is set without a line,
// if startLine is present without a line, if startLine is present but not
// positive, or if startLine is not strictly less than line.
func NewInlineContext(path string, line *int, startLine *int, diffHunk string, outdated bool) (InlineContext, error) {
	if path == "" {
		return InlineContext{}, errors.New("inline context path must not be empty")
	}
	if line != nil && *line <= 0 {
		return InlineContext{}, errors.New("inline context line must be positive when present")
	}
	if line == nil && outdated {
		return InlineContext{}, errors.New("inline context cannot be outdated without a line")
	}
	if startLine != nil && line == nil {
		return InlineContext{}, errors.New("inline context cannot have a start line without a line")
	}
	if startLine != nil && *startLine <= 0 {
		return InlineContext{}, errors.New("inline context start line must be positive when present")
	}
	if startLine != nil && line != nil && *startLine >= *line {
		return InlineContext{}, errors.New("inline context start line must be less than line")
	}
	return InlineContext{path: path, line: line, startLine: startLine, diffHunk: diffHunk, outdated: outdated}, nil
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

// StartLine returns the first line of a range-anchored comment's span, or
// nil when the comment is anchored to a single line (or has no line at
// all).
func (c InlineContext) StartLine() *int {
	return c.startLine
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

// Equals reports whether c and other have the same path, line, start line,
// diff hunk, and outdated flag.
func (c InlineContext) Equals(other InlineContext) bool {
	return c.path == other.path &&
		equalIntPointers(c.line, other.line) &&
		equalIntPointers(c.startLine, other.startLine) &&
		c.diffHunk == other.diffHunk &&
		c.outdated == other.outdated
}

func equalIntPointers(a, b *int) bool {
	return equalPointers(a, b, func(x, y int) bool { return x == y })
}
