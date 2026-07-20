package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.PullRequestReview{}

func newReviewAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/pull/1#pullrequestreview-1")
}

func TestPullRequestReview_Render_IncludesTheReviewStateInTheMetaLine(t *testing.T) {
	cases := []struct {
		name  string
		state valueobjects.ReviewState
		want  string
	}{
		{"approved", valueobjects.ReviewStateApproved, "approved"},
		{"changes requested", valueobjects.ReviewStateChangesRequested, "changes_requested"},
		{"commented", valueobjects.ReviewStateCommented, "commented"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			review := valueobjects.NewPullRequestReview(newReviewAttribution(t), c.state, "Looks solid overall.")

			var buf strings.Builder
			if err := review.Render(&buf); err != nil {
				t.Fatalf("unexpected error rendering review: %v", err)
			}

			want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","state":"` + c.want + `","url":"https://github.com/example/repo/pull/1#pullrequestreview-1"}} -->

Looks solid overall.
`
			if buf.String() != want {
				t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
			}
		})
	}
}

func TestPullRequestReview_Render_AcceptsAnEmptyBody(t *testing.T) {
	review := valueobjects.NewPullRequestReview(newReviewAttribution(t), valueobjects.ReviewStateApproved, "")

	var buf strings.Builder
	if err := review.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering review: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","state":"approved","url":"https://github.com/example/repo/pull/1#pullrequestreview-1"}} -->
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPullRequestReview_Render_CollapsesTrailingNewlinesInTheBodyToASingleOne(t *testing.T) {
	review := valueobjects.NewPullRequestReview(newReviewAttribution(t), valueobjects.ReviewStateApproved, "Looks solid overall.\n\n\n")

	var buf strings.Builder
	if err := review.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering review: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","state":"approved","url":"https://github.com/example/repo/pull/1#pullrequestreview-1"}} -->

Looks solid overall.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPullRequestReview_Render_NormalizesCRLFLineEndingsInTheBody(t *testing.T) {
	review := valueobjects.NewPullRequestReview(newReviewAttribution(t), valueobjects.ReviewStateApproved, "Line one.\r\nLine two.")

	var buf strings.Builder
	if err := review.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering review: %v", err)
	}

	want := `<!-- {"meta":{"author":"octocat","created":"2026-07-02T14:19:40Z","state":"approved","url":"https://github.com/example/repo/pull/1#pullrequestreview-1"}} -->

Line one.
Line two.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPullRequestReview_ExposesTheAttributionStateAndBodyItWasConstructedWith(t *testing.T) {
	attribution := newReviewAttribution(t)
	review := valueobjects.NewPullRequestReview(attribution, valueobjects.ReviewStateChangesRequested, "Please address the nits.")

	if !review.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", review.Attribution(), attribution)
	}
	if review.State() != valueobjects.ReviewStateChangesRequested {
		t.Fatalf("State() = %v, want %v", review.State(), valueobjects.ReviewStateChangesRequested)
	}
	if review.Body() != "Please address the nits." {
		t.Fatalf("Body() = %q, want %q", review.Body(), "Please address the nits.")
	}
}

func TestPullRequestReview_Equals_TreatsDifferentStatesAsNotEqual(t *testing.T) {
	a := valueobjects.NewPullRequestReview(newReviewAttribution(t), valueobjects.ReviewStateApproved, "Looks solid overall.")
	b := valueobjects.NewPullRequestReview(newReviewAttribution(t), valueobjects.ReviewStateChangesRequested, "Looks solid overall.")

	if a.Equals(b) {
		t.Fatal("expected reviews with different states to not be equal")
	}
}
