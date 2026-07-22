package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.ParentIssue{}

func newParentIssueAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/69")
}

func TestParentIssue_Render_WritesTheParentsNumberTitleStateAndURL(t *testing.T) {
	parent := mustNewIssueSummary(t, 64, "Round of Tier 1 entries", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/64")
	p := valueobjects.NewParentIssue(newParentIssueAttribution(t), parent)

	var buf strings.Builder
	if err := p.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering parent issue: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"octocat\",\"created\":\"2026-07-02T14:19:40Z\",\"number\":64,\"title\":\"Round of Tier 1 entries\",\"state\":\"open\",\"url\":\"https://github.com/example/repo/issues/64\"}} -->\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestParentIssue_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	attribution := newParentIssueAttribution(t)
	parent := mustNewIssueSummary(t, 64, "title", valueobjects.IssueStateClosed, "https://github.com/example/repo/issues/64")
	p := valueobjects.NewParentIssue(attribution, parent)

	if !p.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", p.Attribution(), attribution)
	}
	if !p.Parent().Equals(parent) {
		t.Fatalf("Parent() = %#v, want %#v", p.Parent(), parent)
	}
}

func TestParentIssue_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	attribution := newParentIssueAttribution(t)
	parent := mustNewIssueSummary(t, 64, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/64")
	a := valueobjects.NewParentIssue(attribution, parent)
	b := valueobjects.NewParentIssue(attribution, parent)

	if !a.Equals(b) {
		t.Fatal("expected parent issues with matching fields to be equal")
	}
}

func TestParentIssue_Equals_TreatsDifferentParentAsNotEqual(t *testing.T) {
	attribution := newParentIssueAttribution(t)
	a := valueobjects.NewParentIssue(attribution, mustNewIssueSummary(t, 64, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/64"))
	b := valueobjects.NewParentIssue(attribution, mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65"))

	if a.Equals(b) {
		t.Fatal("expected parent issues with different parent to not be equal")
	}
}
