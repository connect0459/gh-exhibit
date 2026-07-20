package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.Body{}

func newBodyAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "connect0459", time.Date(2025, 9, 19, 2, 31, 29, 0, time.UTC), "https://github.com/example/repo/issues/1")
}

func TestBody_Render_OmitsClosedAndMergedForAnOpenIssue(t *testing.T) {
	body := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", nil, nil)

	var buf strings.Builder
	if err := body.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering body: %v", err)
	}

	want := `<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","url":"https://github.com/example/repo/issues/1"}} -->

Issue description.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestBody_Render_IncludesClosedTimestampForAClosedIssue(t *testing.T) {
	closedAt := time.Date(2026, 3, 16, 4, 27, 50, 0, time.UTC)
	body := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", &closedAt, nil)

	var buf strings.Builder
	if err := body.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering body: %v", err)
	}

	want := `<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","closed":"2026-03-16T04:27:50Z","url":"https://github.com/example/repo/issues/1"}} -->

Issue description.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestBody_Render_IncludesBothClosedAndMergedTimestampsForAMergedPullRequest(t *testing.T) {
	closedAt := time.Date(2026, 3, 16, 4, 27, 50, 0, time.UTC)
	mergedAt := time.Date(2026, 3, 16, 4, 27, 50, 0, time.UTC)
	body := valueobjects.NewBody(newBodyAttribution(t), "PR description.", &closedAt, &mergedAt)

	var buf strings.Builder
	if err := body.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering body: %v", err)
	}

	want := `<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","closed":"2026-03-16T04:27:50Z","merged":"2026-03-16T04:27:50Z","url":"https://github.com/example/repo/issues/1"}} -->

PR description.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestBody_Render_CollapsesTrailingNewlinesInTheContentToASingleOne(t *testing.T) {
	body := valueobjects.NewBody(newBodyAttribution(t), "Issue description.\n\n\n", nil, nil)

	var buf strings.Builder
	if err := body.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering body: %v", err)
	}

	want := `<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","url":"https://github.com/example/repo/issues/1"}} -->

Issue description.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestBody_Render_NormalizesCRLFLineEndingsInTheContent(t *testing.T) {
	body := valueobjects.NewBody(newBodyAttribution(t), "Line one.\r\nLine two.", nil, nil)

	var buf strings.Builder
	if err := body.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering body: %v", err)
	}

	want := `<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","url":"https://github.com/example/repo/issues/1"}} -->

Line one.
Line two.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestBody_ExposesTheAttributionContentAndTimestampsItWasConstructedWith(t *testing.T) {
	attribution := newBodyAttribution(t)
	closedAt := time.Date(2026, 3, 16, 4, 27, 50, 0, time.UTC)
	mergedAt := time.Date(2026, 3, 16, 4, 27, 50, 0, time.UTC)
	body := valueobjects.NewBody(attribution, "PR description.", &closedAt, &mergedAt)

	if !body.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", body.Attribution(), attribution)
	}
	if body.Content() != "PR description." {
		t.Fatalf("Content() = %q, want %q", body.Content(), "PR description.")
	}
	if body.ClosedAt() == nil || !body.ClosedAt().Equal(closedAt) {
		t.Fatalf("ClosedAt() = %v, want %v", body.ClosedAt(), closedAt)
	}
	if body.MergedAt() == nil || !body.MergedAt().Equal(mergedAt) {
		t.Fatalf("MergedAt() = %v, want %v", body.MergedAt(), mergedAt)
	}
}

func TestBody_ClosedAtAndMergedAt_AreNilForAnOpenIssue(t *testing.T) {
	body := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", nil, nil)

	if body.ClosedAt() != nil {
		t.Fatalf("ClosedAt() = %v, want nil", body.ClosedAt())
	}
	if body.MergedAt() != nil {
		t.Fatalf("MergedAt() = %v, want nil", body.MergedAt())
	}
}

func TestBody_Equals_TreatsMatchingValuesAsEqual(t *testing.T) {
	closedAt := time.Date(2026, 3, 16, 4, 27, 50, 0, time.UTC)
	a := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", &closedAt, nil)
	b := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", &closedAt, nil)

	if !a.Equals(b) {
		t.Fatal("expected bodies with matching attribution, content, and timestamps to be equal")
	}
}

func TestBody_Equals_TreatsTwoOpenBodiesAsEqual(t *testing.T) {
	a := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", nil, nil)
	b := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", nil, nil)

	if !a.Equals(b) {
		t.Fatal("expected two open bodies (both ClosedAt/MergedAt nil) to be equal")
	}
}

func TestBody_Equals_TreatsOneClosedAndOneOpenAsNotEqual(t *testing.T) {
	closedAt := time.Date(2026, 3, 16, 4, 27, 50, 0, time.UTC)
	a := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", &closedAt, nil)
	b := valueobjects.NewBody(newBodyAttribution(t), "Issue description.", nil, nil)

	if a.Equals(b) {
		t.Fatal("expected a closed body and an open body to not be equal")
	}
}
