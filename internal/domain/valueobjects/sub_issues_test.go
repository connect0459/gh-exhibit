package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.SubIssues{}

func newSubIssuesAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/64")
}

func TestSubIssues_Render_ListsEachChildWithItsCompletionStatus(t *testing.T) {
	children := []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 65, "Include issue/PR labels", valueobjects.IssueStateClosed, "https://github.com/example/repo/issues/65"),
		mustNewIssueSummary(t, 69, "Include parent/child issue relationships", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/69"),
	}
	s := valueobjects.NewSubIssues(newSubIssuesAttribution(t), children)

	var buf strings.Builder
	if err := s.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering sub-issues: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"octocat\",\"created\":\"2026-07-02T14:19:40Z\",\"sub_issues\":2,\"url\":\"https://github.com/example/repo/issues/64\"}} -->\n" +
		"\n" +
		"- `Include issue/PR labels` [#65](https://github.com/example/repo/issues/65) (closed)\n" +
		"- `Include parent/child issue relationships` [#69](https://github.com/example/repo/issues/69) (open)\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestSubIssues_Render_UsesALongerFenceWhenAChildTitleContainsABacktick(t *testing.T) {
	children := []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 65, "Use `foo` here", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65"),
	}
	s := valueobjects.NewSubIssues(newSubIssuesAttribution(t), children)

	var buf strings.Builder
	if err := s.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering sub-issues: %v", err)
	}

	if !strings.Contains(buf.String(), "- ``Use `foo` here`` [#65](https://github.com/example/repo/issues/65) (open)\n") {
		t.Fatalf("Render() = %q, want a bullet with a double-backtick fence", buf.String())
	}
}

func TestSubIssues_Render_PadsWithASpaceWhenAChildTitleStartsWithABacktick(t *testing.T) {
	children := []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 65, "`code` in the title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65"),
	}
	s := valueobjects.NewSubIssues(newSubIssuesAttribution(t), children)

	var buf strings.Builder
	if err := s.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering sub-issues: %v", err)
	}

	if !strings.Contains(buf.String(), "- `` `code` in the title `` [#65](https://github.com/example/repo/issues/65) (open)\n") {
		t.Fatalf("Render() = %q, want a bullet padded around the leading backtick", buf.String())
	}
}

func TestSubIssues_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	attribution := newSubIssuesAttribution(t)
	children := []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65"),
	}
	s := valueobjects.NewSubIssues(attribution, children)

	if !s.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", s.Attribution(), attribution)
	}
	if len(s.Children()) != 1 || !s.Children()[0].Equals(children[0]) {
		t.Fatalf("Children() = %#v, want %#v", s.Children(), children)
	}
}

func TestSubIssues_Children_MutatingTheReturnedSliceDoesNotAffectIt(t *testing.T) {
	children := []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65"),
	}
	s := valueobjects.NewSubIssues(newSubIssuesAttribution(t), children)

	returned := s.Children()
	returned[0] = mustNewIssueSummary(t, 999, "tampered", valueobjects.IssueStateClosed, "https://github.com/example/repo/issues/999")

	if s.Children()[0].Number() != 65 {
		t.Fatalf("mutating the returned slice affected the sub-issues' own state: got %d", s.Children()[0].Number())
	}
}

func TestSubIssues_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	attribution := newSubIssuesAttribution(t)
	children := []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65"),
	}
	a := valueobjects.NewSubIssues(attribution, children)
	b := valueobjects.NewSubIssues(attribution, children)

	if !a.Equals(b) {
		t.Fatal("expected sub-issues with matching fields to be equal")
	}
}

func TestSubIssues_Equals_TreatsDifferentChildrenAsNotEqual(t *testing.T) {
	attribution := newSubIssuesAttribution(t)
	a := valueobjects.NewSubIssues(attribution, []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65"),
	})
	b := valueobjects.NewSubIssues(attribution, []valueobjects.IssueSummary{
		mustNewIssueSummary(t, 66, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/66"),
	})

	if a.Equals(b) {
		t.Fatal("expected sub-issues with different children to not be equal")
	}
}
