package valueobjects_test

import (
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestNewSearchQuery_RejectsEmptyOwner(t *testing.T) {
	_, err := valueobjects.NewSearchQuery("", "gh-exhibit", "", "", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	if err == nil {
		t.Fatal("expected an error for an empty owner, got nil")
	}
}

func TestNewSearchQuery_RejectsEmptyRepo(t *testing.T) {
	_, err := valueobjects.NewSearchQuery("connect0459", "", "", "", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	if err == nil {
		t.Fatal("expected an error for an empty repo, got nil")
	}
}

func TestNewSearchQuery_RejectsANonPositiveMaxResults(t *testing.T) {
	_, err := valueobjects.NewSearchQuery("connect0459", "gh-exhibit", "", "", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 0)

	if err == nil {
		t.Fatal("expected an error for a non-positive max results, got nil")
	}
}

func TestNewSearchQuery_AcceptsAnEmptyAuthorAndAssigneeMeaningUnfiltered(t *testing.T) {
	_, err := valueobjects.NewSearchQuery("connect0459", "gh-exhibit", "", "", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	if err != nil {
		t.Fatalf("unexpected error for an unfiltered query: %v", err)
	}
}

func TestSearchQuery_Accessors_ReturnTheConstructedValues(t *testing.T) {
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	kinds := []valueobjects.IssueKind{valueobjects.IssueKindIssue}

	query, err := valueobjects.NewSearchQuery(
		"connect0459", "gh-exhibit", "octocat", "monalisa", kinds, &after, &before,
		valueobjects.SearchSortByUpdated, valueobjects.SearchOrderAscending, 42,
	)
	if err != nil {
		t.Fatalf("unexpected error building search query: %v", err)
	}

	if got := query.Owner(); got != "connect0459" {
		t.Fatalf("Owner() = %q, want %q", got, "connect0459")
	}
	if got := query.Repo(); got != "gh-exhibit" {
		t.Fatalf("Repo() = %q, want %q", got, "gh-exhibit")
	}
	if got := query.Author(); got != "octocat" {
		t.Fatalf("Author() = %q, want %q", got, "octocat")
	}
	if got := query.Assignee(); got != "monalisa" {
		t.Fatalf("Assignee() = %q, want %q", got, "monalisa")
	}
	if got := query.Kinds(); len(got) != 1 || got[0] != valueobjects.IssueKindIssue {
		t.Fatalf("Kinds() = %v, want [issue]", got)
	}
	if got := query.CreatedAfter(); got == nil || !got.Equal(after) {
		t.Fatalf("CreatedAfter() = %v, want %v", got, after)
	}
	if got := query.CreatedBefore(); got == nil || !got.Equal(before) {
		t.Fatalf("CreatedBefore() = %v, want %v", got, before)
	}
	if got := query.Sort(); got != valueobjects.SearchSortByUpdated {
		t.Fatalf("Sort() = %v, want %v", got, valueobjects.SearchSortByUpdated)
	}
	if got := query.Order(); got != valueobjects.SearchOrderAscending {
		t.Fatalf("Order() = %v, want %v", got, valueobjects.SearchOrderAscending)
	}
	if got := query.MaxResults(); got != 42 {
		t.Fatalf("MaxResults() = %d, want 42", got)
	}
}
