package services

import "time"

type actorWire struct {
	Login string `json:"login"`
}

// resolvedLogin falls back to "ghost" when GitHub returns a null user
// (a deleted account) — GitHub's own sentinel login for this case, so
// comments and reviews from deleted accounts stay attributable instead of
// failing Attribution's non-empty-author invariant.
func (a actorWire) resolvedLogin() string {
	if a.Login == "" {
		return "ghost"
	}
	return a.Login
}

type commentedEventWire struct {
	ID        int64     `json:"id"`
	User      actorWire `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	HTMLURL   string    `json:"html_url"`
}

type reviewedEventWire struct {
	ID          int64     `json:"id"`
	User        actorWire `json:"user"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	SubmittedAt time.Time `json:"submitted_at"`
	HTMLURL     string    `json:"html_url"`
}

type reviewCommentWire struct {
	ID                  int64     `json:"id"`
	PullRequestReviewID int64     `json:"pull_request_review_id"`
	User                actorWire `json:"user"`
	Body                string    `json:"body"`
	Path                string    `json:"path"`
	Line                int       `json:"line"`
	OriginalLine        int       `json:"original_line"`
	DiffHunk            string    `json:"diff_hunk"`
	CreatedAt           time.Time `json:"created_at"`
	HTMLURL             string    `json:"html_url"`
}

// resolvedLine falls back to original_line, marked outdated, when line is
// null (0 after unmarshal) — GitHub clears a review comment's line once the
// diff it anchored to has changed, but original_line still records where it
// pointed when the comment was made. If both are null/zero, the comment has
// no line at all (GitHub's subject_type "file", a comment on the whole file
// rather than a position within it), not an outdated one.
func (w reviewCommentWire) resolvedLine() (line *int, outdated bool) {
	if w.Line > 0 {
		v := w.Line
		return &v, false
	}
	if w.OriginalLine > 0 {
		v := w.OriginalLine
		return &v, true
	}
	return nil, false
}
