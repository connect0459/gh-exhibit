package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.PullRequestCommits{}

func newPullRequestCommitsAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/pull/1")
}

func TestPullRequestCommits_Render_ListsEachCommitAndItsFullMessage(t *testing.T) {
	commits := []valueobjects.Commit{
		mustNewCommit(t, "abc1234567", "octocat", time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC), "octocat", time.Date(2026, 7, 1, 0, 5, 0, 0, time.UTC), "feat: add a thing\n\nSome body text."),
		mustNewCommit(t, "def7654321", "octocat", time.Date(2026, 7, 1, 1, 0, 0, 0, time.UTC), "web-flow", time.Date(2026, 7, 1, 2, 0, 0, 0, time.UTC), "fix: correct a thing"),
	}
	pc := valueobjects.NewPullRequestCommits(newPullRequestCommitsAttribution(t), commits)

	var buf strings.Builder
	if err := pc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request commits: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"octocat\",\"created\":\"2026-07-02T14:19:40Z\",\"commits\":2,\"url\":\"https://github.com/example/repo/pull/1\"}} -->\n" +
		"\n" +
		"- `abc1234` octocat (authored 2026-07-01T00:00:00Z, committed 2026-07-01T00:05:00Z by octocat)\n" +
		"- `def7654` octocat (authored 2026-07-01T01:00:00Z, committed 2026-07-01T02:00:00Z by web-flow)\n" +
		"\n" +
		"**Commit `abc1234`**\n" +
		"\n" +
		"```\n" +
		"feat: add a thing\n" +
		"\n" +
		"Some body text.\n" +
		"```\n" +
		"\n" +
		"**Commit `def7654`**\n" +
		"\n" +
		"```\n" +
		"fix: correct a thing\n" +
		"```\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPullRequestCommits_Render_OmitsMessageBlockForACommitWithNoMessage(t *testing.T) {
	commits := []valueobjects.Commit{
		mustNewCommit(t, "abc1234567", "octocat", time.Now(), "octocat", time.Now(), ""),
	}
	pc := valueobjects.NewPullRequestCommits(newPullRequestCommitsAttribution(t), commits)

	var buf strings.Builder
	if err := pc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request commits: %v", err)
	}

	if strings.Contains(buf.String(), "**Commit") {
		t.Fatalf("Render() should not include a message block for a commit with no message, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "abc1234") {
		t.Fatalf("Render() should still list the commit's sha, got:\n%s", buf.String())
	}
}

func TestPullRequestCommits_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	attribution := newPullRequestCommitsAttribution(t)
	commits := []valueobjects.Commit{
		mustNewCommit(t, "abc1234567", "octocat", time.Now(), "octocat", time.Now(), "message"),
	}
	pc := valueobjects.NewPullRequestCommits(attribution, commits)

	if !pc.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", pc.Attribution(), attribution)
	}
	if len(pc.Commits()) != 1 || !pc.Commits()[0].Equals(commits[0]) {
		t.Fatalf("Commits() = %#v, want %#v", pc.Commits(), commits)
	}
}

func TestPullRequestCommits_Commits_MutatingTheReturnedSliceDoesNotAffectIt(t *testing.T) {
	commits := []valueobjects.Commit{
		mustNewCommit(t, "abc1234567", "octocat", time.Now(), "octocat", time.Now(), "message"),
	}
	pc := valueobjects.NewPullRequestCommits(newPullRequestCommitsAttribution(t), commits)

	returned := pc.Commits()
	returned[0] = mustNewCommit(t, "tampered", "tampered", time.Now(), "tampered", time.Now(), "tampered")

	if pc.Commits()[0].SHA() != "abc1234567" {
		t.Fatalf("mutating the returned slice affected the pull request commits' own state: got %q", pc.Commits()[0].SHA())
	}
}

func TestPullRequestCommits_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	attribution := newPullRequestCommitsAttribution(t)
	commits := []valueobjects.Commit{
		mustNewCommit(t, "abc1234567", "octocat", time.Now(), "octocat", time.Now(), "message"),
	}
	a := valueobjects.NewPullRequestCommits(attribution, commits)
	b := valueobjects.NewPullRequestCommits(attribution, commits)

	if !a.Equals(b) {
		t.Fatal("expected pull request commits with matching fields to be equal")
	}
}

func TestPullRequestCommits_Equals_TreatsDifferentCommitsAsNotEqual(t *testing.T) {
	attribution := newPullRequestCommitsAttribution(t)
	a := valueobjects.NewPullRequestCommits(attribution, []valueobjects.Commit{
		mustNewCommit(t, "abc1234567", "octocat", time.Now(), "octocat", time.Now(), "message"),
	})
	b := valueobjects.NewPullRequestCommits(attribution, []valueobjects.Commit{
		mustNewCommit(t, "def7654321", "octocat", time.Now(), "octocat", time.Now(), "message"),
	})

	if a.Equals(b) {
		t.Fatal("expected pull request commits with different commits to not be equal")
	}
}
