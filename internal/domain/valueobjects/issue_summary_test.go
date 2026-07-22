package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func mustNewIssueSummary(t *testing.T, number int, title string, state valueobjects.IssueState, url string) valueobjects.IssueSummary {
	t.Helper()
	summary, err := valueobjects.NewIssueSummary(number, title, state, url)
	if err != nil {
		t.Fatalf("NewIssueSummary(): unexpected error: %v", err)
	}
	return summary
}

func TestNewIssueSummary_RejectsANonPositiveNumber(t *testing.T) {
	_, err := valueobjects.NewIssueSummary(0, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/1")

	if err == nil {
		t.Fatal("expected an error for a non-positive number, got nil")
	}
}

func TestNewIssueSummary_RejectsANonAbsoluteURL(t *testing.T) {
	_, err := valueobjects.NewIssueSummary(1, "title", valueobjects.IssueStateOpen, "not-a-url")

	if err == nil {
		t.Fatal("expected an error for a non-absolute url, got nil")
	}
}

func TestIssueSummary_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	summary := mustNewIssueSummary(t, 65, "Include issue/PR labels", valueobjects.IssueStateClosed, "https://github.com/example/repo/issues/65")

	if summary.Number() != 65 {
		t.Fatalf("Number() = %d, want %d", summary.Number(), 65)
	}
	if summary.Title() != "Include issue/PR labels" {
		t.Fatalf("Title() = %q, want %q", summary.Title(), "Include issue/PR labels")
	}
	if summary.State() != valueobjects.IssueStateClosed {
		t.Fatalf("State() = %v, want %v", summary.State(), valueobjects.IssueStateClosed)
	}
	if summary.URL().String() != "https://github.com/example/repo/issues/65" {
		t.Fatalf("URL() = %q, want %q", summary.URL().String(), "https://github.com/example/repo/issues/65")
	}
}

func TestIssueSummary_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	a := mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65")
	b := mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65")

	if !a.Equals(b) {
		t.Fatal("expected issue summaries with matching fields to be equal")
	}
}

func TestIssueSummary_Equals_TreatsDifferentStateAsNotEqual(t *testing.T) {
	a := mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateOpen, "https://github.com/example/repo/issues/65")
	b := mustNewIssueSummary(t, 65, "title", valueobjects.IssueStateClosed, "https://github.com/example/repo/issues/65")

	if a.Equals(b) {
		t.Fatal("expected issue summaries with different state to not be equal")
	}
}
