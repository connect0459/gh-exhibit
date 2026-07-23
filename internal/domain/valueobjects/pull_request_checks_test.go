package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.PullRequestChecks{}

func newPullRequestChecksAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/pull/1")
}

func TestPullRequestChecks_Render_ListsEachCheckRunAlongsideTheCapturedTimestamp(t *testing.T) {
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
		mustNewCheckRun(t, "test", valueobjects.CheckOutcomeFailure, "https://github.com/example/repo/runs/2"),
	}
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	pc := valueobjects.NewPullRequestChecks(newPullRequestChecksAttribution(t), "abc1234567", capturedAt, runs)

	var buf strings.Builder
	if err := pc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request checks: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"octocat\",\"created\":\"2026-07-02T14:19:40Z\",\"head_sha\":\"abc1234567\",\"captured_at\":\"2026-07-22T09:30:00Z\",\"checks\":2,\"url\":\"https://github.com/example/repo/pull/1\"}} -->\n" +
		"\n" +
		"- `build`: success\n" +
		"- `test`: failure\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPullRequestChecks_Render_DoesNotLetACheckRunNameInjectMarkdownLinkSyntax(t *testing.T) {
	// A check run's name is arbitrary, attacker-influenceable text (a CI
	// job name, or a third-party Checks app's own naming) — the same
	// untrusted-string handling changedFileLine/commitLine/issueSummaryLine
	// already apply, none of which embed untrusted text inside a
	// "[text](url)" markdown link construct. A name shaped like
	// "click here](https://attacker.example)" would, if embedded that
	// way, close the real link early and splice in an attacker-chosen
	// URL; rendered verbatim inside a backtick span (this test's exact
	// expectation), it stays inert literal text instead.
	maliciousName := "click here](https://attacker.example)"
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, maliciousName, valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
	}
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	pc := valueobjects.NewPullRequestChecks(newPullRequestChecksAttribution(t), "abc1234567", capturedAt, runs)

	var buf strings.Builder
	if err := pc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request checks: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"octocat\",\"created\":\"2026-07-02T14:19:40Z\",\"head_sha\":\"abc1234567\",\"captured_at\":\"2026-07-22T09:30:00Z\",\"checks\":1,\"url\":\"https://github.com/example/repo/pull/1\"}} -->\n" +
		"\n" +
		"- `click here](https://attacker.example)`: success\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPullRequestChecks_Render_FencesACheckRunNameContainingABacktick(t *testing.T) {
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, "weird`build", valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
	}
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	pc := valueobjects.NewPullRequestChecks(newPullRequestChecksAttribution(t), "abc1234567", capturedAt, runs)

	var buf strings.Builder
	if err := pc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request checks: %v", err)
	}

	want := "- ``weird`build``: success\n"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("Render() should keep the whole check run name inside one unbroken code span, got:\n%s\nwant substring:\n%s", buf.String(), want)
	}
}

func TestPullRequestChecks_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	attribution := newPullRequestChecksAttribution(t)
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
	}
	pc := valueobjects.NewPullRequestChecks(attribution, "abc1234567", capturedAt, runs)

	if !pc.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", pc.Attribution(), attribution)
	}
	if pc.HeadSHA() != "abc1234567" {
		t.Fatalf("HeadSHA() = %q, want %q", pc.HeadSHA(), "abc1234567")
	}
	if !pc.CapturedAt().Equal(capturedAt) {
		t.Fatalf("CapturedAt() = %v, want %v", pc.CapturedAt(), capturedAt)
	}
	if len(pc.Runs()) != 1 || !pc.Runs()[0].Equals(runs[0]) {
		t.Fatalf("Runs() = %#v, want %#v", pc.Runs(), runs)
	}
}

func TestPullRequestChecks_Runs_MutatingTheReturnedSliceDoesNotAffectIt(t *testing.T) {
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
	}
	pc := valueobjects.NewPullRequestChecks(newPullRequestChecksAttribution(t), "abc1234567", time.Now(), runs)

	returned := pc.Runs()
	returned[0] = mustNewCheckRun(t, "tampered", valueobjects.CheckOutcomeFailure, "https://github.com/example/repo/runs/9")

	if pc.Runs()[0].Name() != "build" {
		t.Fatalf("mutating the returned slice affected the pull request checks' own state: got %q", pc.Runs()[0].Name())
	}
}

func TestPullRequestChecks_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	attribution := newPullRequestChecksAttribution(t)
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
	}
	a := valueobjects.NewPullRequestChecks(attribution, "abc1234567", capturedAt, runs)
	b := valueobjects.NewPullRequestChecks(attribution, "abc1234567", capturedAt, runs)

	if !a.Equals(b) {
		t.Fatal("expected pull request checks with matching fields to be equal")
	}
}

func TestPullRequestChecks_Equals_TreatsDifferentHeadSHAAsNotEqual(t *testing.T) {
	attribution := newPullRequestChecksAttribution(t)
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
	}
	a := valueobjects.NewPullRequestChecks(attribution, "abc1234567", capturedAt, runs)
	b := valueobjects.NewPullRequestChecks(attribution, "def7654321", capturedAt, runs)

	if a.Equals(b) {
		t.Fatal("expected pull request checks with different head shas to not be equal")
	}
}

func TestPullRequestChecks_Equals_TreatsDifferentCapturedAtAsNotEqual(t *testing.T) {
	attribution := newPullRequestChecksAttribution(t)
	runs := []valueobjects.CheckRun{
		mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/example/repo/runs/1"),
	}
	a := valueobjects.NewPullRequestChecks(attribution, "abc1234567", time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC), runs)
	b := valueobjects.NewPullRequestChecks(attribution, "abc1234567", time.Date(2026, 7, 22, 10, 30, 0, 0, time.UTC), runs)

	if a.Equals(b) {
		t.Fatal("expected pull request checks with different captured-at times to not be equal")
	}
}
