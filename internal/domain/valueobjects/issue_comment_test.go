package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.IssueComment{}

func newIssueCommentAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/1#issuecomment-1")
}

func TestIssueComment_Render_WritesAnAnchoredMetaLineFollowedByTheBody(t *testing.T) {
	comment := valueobjects.NewIssueComment(newIssueCommentAttribution(t), "Looks good to me.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering issue comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","url":"https://github.com/example/repo/issues/1#issuecomment-1"}} -->

Looks good to me.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestIssueComment_Render_CollapsesTrailingNewlinesInTheBodyToASingleOne(t *testing.T) {
	comment := valueobjects.NewIssueComment(newIssueCommentAttribution(t), "Looks good to me.\n\n\n")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering issue comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","url":"https://github.com/example/repo/issues/1#issuecomment-1"}} -->

Looks good to me.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestIssueComment_Render_NormalizesCRLFLineEndingsInTheBody(t *testing.T) {
	comment := valueobjects.NewIssueComment(newIssueCommentAttribution(t), "Line one.\r\nLine two.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering issue comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","url":"https://github.com/example/repo/issues/1#issuecomment-1"}} -->

Line one.
Line two.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestIssueComment_Equals_TreatsMatchingAttributionAndBodyAsEqual(t *testing.T) {
	attribution := newIssueCommentAttribution(t)
	a := valueobjects.NewIssueComment(attribution, "Looks good to me.")
	b := valueobjects.NewIssueComment(attribution, "Looks good to me.")

	if !a.Equals(b) {
		t.Fatal("expected issue comments with matching attribution and body to be equal")
	}
}

func TestIssueComment_ExposesTheAttributionAndBodyItWasConstructedWith(t *testing.T) {
	attribution := newIssueCommentAttribution(t)
	comment := valueobjects.NewIssueComment(attribution, "Looks good to me.")

	if !comment.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", comment.Attribution(), attribution)
	}
	if comment.Body() != "Looks good to me." {
		t.Fatalf("Body() = %q, want %q", comment.Body(), "Looks good to me.")
	}
}

func TestIssueComment_Equals_TreatsDifferentBodiesAsNotEqual(t *testing.T) {
	attribution := newIssueCommentAttribution(t)
	a := valueobjects.NewIssueComment(attribution, "Looks good to me.")
	b := valueobjects.NewIssueComment(attribution, "Needs another look.")

	if a.Equals(b) {
		t.Fatal("expected issue comments with different bodies to not be equal")
	}
}
