package valueobjects

import (
	"errors"
	"time"
)

// Commit is one commit included in a pull request, sourced from GET
// /pulls/{number}/commits. authorName/authoredAt are the commit's git
// author identity and timestamp (commit.author); committerName/committedAt
// are its committer identity and timestamp (commit.committer) — these can
// differ from the author's own, e.g. when GitHub's web UI or a
// rebase/squash operation re-commits an existing author's work.
type Commit struct {
	sha           string
	authorName    string
	authoredAt    time.Time
	committerName string
	committedAt   time.Time
	message       string
}

// NewCommit constructs a Commit from its sha, git author name and authored
// time, git committer name and committed time, and full message. It
// returns an error if sha is empty.
func NewCommit(sha, authorName string, authoredAt time.Time, committerName string, committedAt time.Time, message string) (Commit, error) {
	if sha == "" {
		return Commit{}, errors.New("commit sha must not be empty")
	}
	return Commit{
		sha:           sha,
		authorName:    authorName,
		authoredAt:    authoredAt,
		committerName: committerName,
		committedAt:   committedAt,
		message:       message,
	}, nil
}

// SHA returns the commit's full hash.
func (c Commit) SHA() string {
	return c.sha
}

// AuthorName returns the commit's git author name.
func (c Commit) AuthorName() string {
	return c.authorName
}

// AuthoredAt returns when the commit was authored.
func (c Commit) AuthoredAt() time.Time {
	return c.authoredAt
}

// CommitterName returns the commit's git committer name.
func (c Commit) CommitterName() string {
	return c.committerName
}

// CommittedAt returns when the commit was committed.
func (c Commit) CommittedAt() time.Time {
	return c.committedAt
}

// Message returns the commit's full message (subject and body).
func (c Commit) Message() string {
	return c.message
}

// Equals reports whether c and other have the same sha, author name,
// authored time, committer name, committed time, and message.
func (c Commit) Equals(other Commit) bool {
	return c.sha == other.sha &&
		c.authorName == other.authorName &&
		c.authoredAt.Equal(other.authoredAt) &&
		c.committerName == other.committerName &&
		c.committedAt.Equal(other.committedAt) &&
		c.message == other.message
}
