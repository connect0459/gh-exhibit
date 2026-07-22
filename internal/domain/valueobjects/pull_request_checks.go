package valueobjects

import (
	"fmt"
	"io"
	"slices"
	"strings"
	"time"
)

// PullRequestChecks is the check runs associated with a pull request's head
// commit, sourced from GET /repos/{owner}/{repo}/commits/{sha}/check-runs.
// Like PullRequestDiff/PullRequestCommits, it has no event of its own, so
// its attribution reuses the pull request's own (author, created, url)
// rather than a per-event one. Unlike every other Tier 1 type, the content
// it captures can keep changing after the export is taken — a check can be
// re-run, or a pending one can later resolve — so capturedAt records the
// wall-clock time this snapshot was taken, distinguishing it from
// attribution's created (the pull request's own creation time), so a reader
// does not mistake this entry for the pull request's current, possibly
// since-changed check state.
type PullRequestChecks struct {
	attribution Attribution
	headSHA     string
	capturedAt  time.Time
	runs        []CheckRun
}

// NewPullRequestChecks constructs a PullRequestChecks from its attribution,
// the head commit sha the check runs belong to, the wall-clock time this
// snapshot was captured, and the check runs themselves.
func NewPullRequestChecks(attribution Attribution, headSHA string, capturedAt time.Time, runs []CheckRun) PullRequestChecks {
	// Cloned so a later mutation of the caller's slice can't silently
	// change this PullRequestChecks after construction (Immutable First).
	return PullRequestChecks{attribution: attribution, headSHA: headSHA, capturedAt: capturedAt, runs: slices.Clone(runs)}
}

// Attribution returns the pull request's own author, creation time, and
// URL (see the PullRequestChecks Godoc for why this isn't a per-event
// attribution).
func (c PullRequestChecks) Attribution() Attribution {
	return c.attribution
}

// HeadSHA returns the commit sha these check runs belong to.
func (c PullRequestChecks) HeadSHA() string {
	return c.headSHA
}

// CapturedAt returns the wall-clock time this check-run snapshot was taken
// (see the PullRequestChecks Godoc for why this differs from
// Attribution().CreatedAt()).
func (c PullRequestChecks) CapturedAt() time.Time {
	return c.capturedAt
}

// Runs returns a copy of the pull request's check runs, so mutating the
// returned slice can't affect this PullRequestChecks (Immutable First).
func (c PullRequestChecks) Runs() []CheckRun {
	return slices.Clone(c.runs)
}

// Equals reports whether c and other have the same attribution, head sha,
// captured-at time, and check runs.
func (c PullRequestChecks) Equals(other PullRequestChecks) bool {
	return c.attribution.Equals(other.attribution) &&
		c.headSHA == other.headSHA &&
		c.capturedAt.Equal(other.capturedAt) &&
		slices.EqualFunc(c.runs, other.runs, CheckRun.Equals)
}

// Render writes c's <!-- {"meta":...} --> line, carrying the head sha,
// the captured-at timestamp, and a check count, then a bullet list of every
// check run's name (linked to its own url) and outcome, satisfying Entry.
func (c PullRequestChecks) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		HeadSHA    string `json:"head_sha"`
		CapturedAt string `json:"captured_at"`
		Checks     int    `json:"checks"`
		URL        Url    `json:"url"`
	}{
		attributionMeta: newAttributionMeta(c.attribution),
		HeadSHA:         c.headSHA,
		CapturedAt:      c.capturedAt.UTC().Format(time.RFC3339),
		Checks:          len(c.runs),
		URL:             c.attribution.URL(),
	}

	var list strings.Builder
	for _, run := range c.runs {
		fmt.Fprintf(&list, "- %s\n", checkRunLine(run))
	}

	return writeMetaLine(w, meta, strings.TrimRight(list.String(), "\n"))
}

// checkRunLine formats one CheckRun as a single bullet-list line, linking
// its name to its own url.
func checkRunLine(r CheckRun) string {
	return fmt.Sprintf("[%s](%s): %s", r.Name(), r.URL().String(), r.Outcome().String())
}

func (PullRequestChecks) entryNode() {}
