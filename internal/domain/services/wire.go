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

// labelEventWire is the shape of a "labeled"/"unlabeled" timeline event.
// Unlike commentedEventWire/reviewedEventWire, its actor field is "actor"
// (not "user"), and it carries no html_url of its own — GitHub gives it
// only an API url (issues/events/{id}), not a page to link to.
type labelEventWire struct {
	ID        int64     `json:"id"`
	Actor     actorWire `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
	Label     struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	} `json:"label"`
}

// closureEventWire is the shape of a "closed"/"reopened" timeline event.
// Like labelEventWire, its actor field is "actor" (not "user"), and it
// carries no html_url of its own. state_reason mirrors GitHub's own field
// name and is only ever populated on a "closed" event.
type closureEventWire struct {
	ID          int64     `json:"id"`
	Actor       actorWire `json:"actor"`
	CreatedAt   time.Time `json:"created_at"`
	StateReason string    `json:"state_reason"`
}

// renameEventWire is the shape of a "renamed" timeline event. Like
// labelEventWire, its actor field is "actor" (not "user"), and it carries
// no html_url of its own.
type renameEventWire struct {
	ID        int64     `json:"id"`
	Actor     actorWire `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
	Rename    struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"rename"`
}

// milestoneEventWire is the shape of a "milestoned"/"demilestoned" timeline
// event. Like labelEventWire, its actor field is "actor" (not "user"), and
// it carries no html_url of its own.
type milestoneEventWire struct {
	ID        int64     `json:"id"`
	Actor     actorWire `json:"actor"`
	CreatedAt time.Time `json:"created_at"`
	Milestone struct {
		Title string `json:"title"`
	} `json:"milestone"`
}

// assignmentEventWire is the shape of an "assigned"/"unassigned" timeline
// event. Like labelEventWire, it carries no html_url of its own. Unlike the
// other actor-only events, it also carries an assignee distinct from the
// actor who performed the assignment.
type assignmentEventWire struct {
	ID        int64     `json:"id"`
	Actor     actorWire `json:"actor"`
	Assignee  actorWire `json:"assignee"`
	CreatedAt time.Time `json:"created_at"`
}

type reviewCommentWire struct {
	ID                  int64     `json:"id"`
	PullRequestReviewID int64     `json:"pull_request_review_id"`
	User                actorWire `json:"user"`
	Body                string    `json:"body"`
	Path                string    `json:"path"`
	StartLine           int       `json:"start_line"`
	OriginalStartLine   int       `json:"original_start_line"`
	Line                int       `json:"line"`
	OriginalLine        int       `json:"original_line"`
	DiffHunk            string    `json:"diff_hunk"`
	CreatedAt           time.Time `json:"created_at"`
	HTMLURL             string    `json:"html_url"`
}

// resolvedLine falls back to original_line/original_start_line, marked
// outdated, when line is null (0 after unmarshal) — GitHub clears a review
// comment's line once the diff it anchored to has changed, but
// original_line still records where it pointed when the comment was made.
// If both line and original_line are null/zero, the comment has no line at
// all (GitHub's subject_type "file", a comment on the whole file rather
// than a position within it), not an outdated one. startLine is nil unless
// the comment is anchored to a range of lines rather than a single one, in
// which case it is drawn from whichever of start_line/original_start_line
// pairs with the line/original_line that resolved line.
func (w reviewCommentWire) resolvedLine() (line, startLine *int, outdated bool) {
	if w.Line > 0 {
		v := w.Line
		return &v, resolvedStartLine(w.StartLine), false
	}
	if w.OriginalLine > 0 {
		v := w.OriginalLine
		return &v, resolvedStartLine(w.OriginalStartLine), true
	}
	return nil, nil, false
}

func resolvedStartLine(startLine int) *int {
	if startLine <= 0 {
		return nil
	}
	return &startLine
}
