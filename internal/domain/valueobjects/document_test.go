package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func newDocumentAttribution(t *testing.T, url string) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "connect0459", time.Date(2025, 9, 19, 2, 31, 29, 0, time.UTC), url)
}

func TestNewDocument_RejectsEmptyTitle(t *testing.T) {
	_, err := valueobjects.NewDocument("", nil)

	if err == nil {
		t.Fatal("expected an error for an empty title, got nil")
	}
}

func TestNewDocument_AcceptsNoEntries(t *testing.T) {
	if _, err := valueobjects.NewDocument("Some title", nil); err != nil {
		t.Fatalf("unexpected error building a document with no entries: %v", err)
	}
}

func TestNewDocument_RejectsANilEntry(t *testing.T) {
	body := valueobjects.NewBody(newDocumentAttribution(t, "https://github.com/example/repo/issues/1"), "Issue description.", nil, nil)

	_, err := valueobjects.NewDocument("Some title", []valueobjects.Entry{body, nil})

	if err == nil {
		t.Fatal("expected an error for a nil entry, got nil")
	}
}

func TestDocument_Entries_MutatingTheReturnedSliceDoesNotAffectTheDocument(t *testing.T) {
	body := valueobjects.NewBody(newDocumentAttribution(t, "https://github.com/example/repo/issues/1"), "Issue description.", nil, nil)
	comment := valueobjects.NewIssueComment(
		newDocumentAttribution(t, "https://github.com/example/repo/issues/1#issuecomment-1"),
		"A follow-up comment.",
	)
	doc, err := valueobjects.NewDocument("Some title", []valueobjects.Entry{body, comment})
	if err != nil {
		t.Fatalf("unexpected error building document: %v", err)
	}

	doc.Entries()[0] = comment

	if !doc.Entries()[0].(valueobjects.Body).Equals(body) {
		t.Fatalf("Entries()[0] = %#v after mutating a previously returned slice, want it unchanged (%#v)", doc.Entries()[0], body)
	}
}

func TestNewDocument_MutatingTheCallerSliceAfterConstructionDoesNotAffectTheDocument(t *testing.T) {
	body := valueobjects.NewBody(newDocumentAttribution(t, "https://github.com/example/repo/issues/1"), "Issue description.", nil, nil)
	comment := valueobjects.NewIssueComment(
		newDocumentAttribution(t, "https://github.com/example/repo/issues/1#issuecomment-1"),
		"A follow-up comment.",
	)
	entries := []valueobjects.Entry{body}
	doc, err := valueobjects.NewDocument("Some title", entries)
	if err != nil {
		t.Fatalf("unexpected error building document: %v", err)
	}

	entries[0] = comment

	if !doc.Entries()[0].(valueobjects.Body).Equals(body) {
		t.Fatalf("Entries()[0] = %#v after mutating the caller's slice, want it unchanged (%#v)", doc.Entries()[0], body)
	}
}

func TestDocument_ExposesTheTitleAndEntriesItWasConstructedWith(t *testing.T) {
	body := valueobjects.NewBody(newDocumentAttribution(t, "https://github.com/example/repo/issues/1"), "Issue description.", nil, nil)
	entries := []valueobjects.Entry{body}

	doc, err := valueobjects.NewDocument("Some title", entries)
	if err != nil {
		t.Fatalf("unexpected error building document: %v", err)
	}

	if doc.Title() != "Some title" {
		t.Fatalf("Title() = %q, want %q", doc.Title(), "Some title")
	}
	if len(doc.Entries()) != 1 || !doc.Entries()[0].(valueobjects.Body).Equals(body) {
		t.Fatalf("Entries() = %#v, want a single entry equal to %#v", doc.Entries(), body)
	}
}

func TestDocument_Render_WritesTheTitleAsAnH1HeadingFollowedByASingleEntry(t *testing.T) {
	body := valueobjects.NewBody(newDocumentAttribution(t, "https://github.com/example/repo/issues/1"), "Issue description.", nil, nil)
	doc, err := valueobjects.NewDocument("Some title", []valueobjects.Entry{body})
	if err != nil {
		t.Fatalf("unexpected error building document: %v", err)
	}

	var buf strings.Builder
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering document: %v", err)
	}

	want := `# Some title

<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","url":"https://github.com/example/repo/issues/1"}} -->

Issue description.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestDocument_Render_JoinsMultipleEntriesWithASeparatorLine(t *testing.T) {
	body := valueobjects.NewBody(newDocumentAttribution(t, "https://github.com/example/repo/issues/1"), "Issue description.", nil, nil)
	comment := valueobjects.NewIssueComment(
		newDocumentAttribution(t, "https://github.com/example/repo/issues/1#issuecomment-1"),
		"A follow-up comment.",
	)
	doc, err := valueobjects.NewDocument("Some title", []valueobjects.Entry{body, comment})
	if err != nil {
		t.Fatalf("unexpected error building document: %v", err)
	}

	var buf strings.Builder
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering document: %v", err)
	}

	want := `# Some title

<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","url":"https://github.com/example/repo/issues/1"}} -->

Issue description.

------

<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","url":"https://github.com/example/repo/issues/1#issuecomment-1"}} -->

A follow-up comment.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestDocument_Render_JoinsAnEmptyBodyEntryWithASingleBlankLineBeforeTheSeparator(t *testing.T) {
	review := valueobjects.NewPullRequestReview(
		newDocumentAttribution(t, "https://github.com/example/repo/pull/1#pullrequestreview-1"),
		valueobjects.ReviewStateCommented,
		"",
	)
	comment := valueobjects.NewIssueComment(
		newDocumentAttribution(t, "https://github.com/example/repo/pull/1#issuecomment-1"),
		"A follow-up comment.",
	)
	doc, err := valueobjects.NewDocument("Some title", []valueobjects.Entry{review, comment})
	if err != nil {
		t.Fatalf("unexpected error building document: %v", err)
	}

	var buf strings.Builder
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering document: %v", err)
	}

	want := `# Some title

<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","state":"commented","url":"https://github.com/example/repo/pull/1#pullrequestreview-1"}} -->

------

<!-- {"meta":{"author":"connect0459","created":"2025-09-19T02:31:29Z","url":"https://github.com/example/repo/pull/1#issuecomment-1"}} -->

A follow-up comment.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestDocument_Render_WritesOnlyTheTitleWhenThereAreNoEntries(t *testing.T) {
	doc, err := valueobjects.NewDocument("Some title", nil)
	if err != nil {
		t.Fatalf("unexpected error building document: %v", err)
	}

	var buf strings.Builder
	if err := doc.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering document: %v", err)
	}

	want := "# Some title\n\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}
