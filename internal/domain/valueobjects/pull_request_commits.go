package valueobjects

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

// PullRequestCommits is a pull request's commit list, sourced from GET
// /pulls/{number}/commits. Like PullRequestDiff, it has no event of its
// own, so its attribution reuses the pull request's own (author, created,
// url) rather than a per-event one.
type PullRequestCommits struct {
	attribution Attribution
	commits     []Commit
}

// NewPullRequestCommits constructs a PullRequestCommits from its
// attribution and commit list.
func NewPullRequestCommits(attribution Attribution, commits []Commit) PullRequestCommits {
	// Cloned so a later mutation of the caller's slice can't silently
	// change this PullRequestCommits after construction (Immutable First).
	return PullRequestCommits{attribution: attribution, commits: slices.Clone(commits)}
}

// Attribution returns the pull request's own author, creation time, and
// URL (see the PullRequestCommits Godoc for why this isn't a per-event
// attribution).
func (c PullRequestCommits) Attribution() Attribution {
	return c.attribution
}

// Commits returns a copy of the pull request's commits, so mutating the
// returned slice can't affect this PullRequestCommits (Immutable First).
func (c PullRequestCommits) Commits() []Commit {
	return slices.Clone(c.commits)
}

// Equals reports whether c and other have the same attribution and commits.
func (c PullRequestCommits) Equals(other PullRequestCommits) bool {
	return c.attribution.Equals(other.attribution) &&
		slices.EqualFunc(c.commits, other.commits, Commit.Equals)
}

// Render writes c's <!-- {"meta":...} --> line, a bullet list of every
// commit, then each commit's own full message under a "**Commit `sha`**"
// label in a fenced code block, satisfying Entry.
func (c PullRequestCommits) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Commits int `json:"commits"`
		URL     Url `json:"url"`
	}{
		attributionMeta: newAttributionMeta(c.attribution),
		Commits:         len(c.commits),
		URL:             c.attribution.URL(),
	}

	var list strings.Builder
	for _, commit := range c.commits {
		fmt.Fprintf(&list, "- %s\n", commitLine(commit))
	}

	if err := writeMetaLine(w, meta, strings.TrimRight(list.String(), "\n")); err != nil {
		return err
	}

	for _, commit := range c.commits {
		message := commit.Message()
		if message == "" {
			continue
		}
		fence := diffFence(message)
		if _, err := fmt.Fprintf(w, "\n**Commit `%s`**\n\n%s\n%s\n%s\n", shortSHA(commit.SHA()), fence, message, fence); err != nil {
			return err
		}
	}

	return nil
}

// commitLine formats one Commit as a single bullet-list line, showing both
// its author's and committer's name and timestamp — these can differ, e.g.
// when GitHub's web UI or a rebase/squash operation re-commits an existing
// author's work.
func commitLine(c Commit) string {
	return fmt.Sprintf("`%s` %s (authored %s, committed %s by %s)",
		shortSHA(c.SHA()), c.AuthorName(), c.AuthoredAt().UTC().Format(time.RFC3339),
		c.CommittedAt().UTC().Format(time.RFC3339), c.CommitterName())
}

// shortSHA truncates sha to its conventional 7-character short form,
// leaving a shorter value (such as a test fixture's) untouched.
func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

func (PullRequestCommits) entryNode() {}
