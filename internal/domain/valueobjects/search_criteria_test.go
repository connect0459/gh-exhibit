package valueobjects_test

import (
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestNewSearchCriteria_AcceptsAMinimalCriteria(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err != nil {
		t.Fatalf("unexpected error for a minimal (unfiltered) criteria: %v", err)
	}
}

func TestNewSearchCriteria_RejectsAnInvalidAuthorLogin(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria([]string{"owner/evil"}, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for an author login containing a slash, got nil")
	}
}

func TestNewSearchCriteria_RejectsAnInvalidAssigneeLogin(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, []string{""}, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for an empty assignee login, got nil")
	}
}

func TestNewSearchCriteria_RejectsCreatedAfterLaterThanCreatedBefore(t *testing.T) {
	after := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, &after, &before, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error when createdAfter is later than createdBefore, got nil")
	}
}

func TestNewSearchCriteria_AcceptsCreatedAfterEqualToCreatedBefore(t *testing.T) {
	same := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, &same, &same, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err != nil {
		t.Fatalf("unexpected error when createdAfter equals createdBefore: %v", err)
	}
}

func TestNewSearchCriteria_RejectsALimitOfZero(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, 0, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a limit of zero, got nil")
	}
}

func TestNewSearchCriteria_RejectsALimitAboveMaxSearchLimit(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, valueobjects.MaxSearchLimit+1, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a limit above MaxSearchLimit, got nil")
	}
}

func TestNewSearchCriteria_AcceptsALimitEqualToMaxSearchLimit(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, valueobjects.MaxSearchLimit, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err != nil {
		t.Fatalf("unexpected error for a limit equal to MaxSearchLimit: %v", err)
	}
}

func TestSearchCriteria_Accessors_ReturnTheConstructedValues(t *testing.T) {
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	kinds := []valueobjects.IssueKind{valueobjects.IssueKindPullRequest}

	criteria, err := valueobjects.NewSearchCriteria(
		[]string{"octocat"}, []string{"monalisa"}, kinds, &after, &before, 42,
		valueobjects.SearchSortByComments, valueobjects.SearchOrderAscending,
	)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	if got := criteria.Authors(); len(got) != 1 || got[0] != "octocat" {
		t.Fatalf("Authors() = %v, want [octocat]", got)
	}
	if got := criteria.Assignees(); len(got) != 1 || got[0] != "monalisa" {
		t.Fatalf("Assignees() = %v, want [monalisa]", got)
	}
	if got := criteria.Kinds(); len(got) != 1 || got[0] != valueobjects.IssueKindPullRequest {
		t.Fatalf("Kinds() = %v, want [pr]", got)
	}
	if got := criteria.CreatedAfter(); got == nil || !got.Equal(after) {
		t.Fatalf("CreatedAfter() = %v, want %v", got, after)
	}
	if got := criteria.CreatedBefore(); got == nil || !got.Equal(before) {
		t.Fatalf("CreatedBefore() = %v, want %v", got, before)
	}
	if got := criteria.Limit(); got != 42 {
		t.Fatalf("Limit() = %d, want 42", got)
	}
	if got := criteria.Sort(); got != valueobjects.SearchSortByComments {
		t.Fatalf("Sort() = %v, want %v", got, valueobjects.SearchSortByComments)
	}
	if got := criteria.Order(); got != valueobjects.SearchOrderAscending {
		t.Fatalf("Order() = %v, want %v", got, valueobjects.SearchOrderAscending)
	}
}

func TestSearchCriteria_Authors_ReturnsADefensiveCopy(t *testing.T) {
	criteria, err := valueobjects.NewSearchCriteria([]string{"octocat"}, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	authors := criteria.Authors()
	authors[0] = "mutated"

	if got := criteria.Authors(); got[0] != "octocat" {
		t.Fatalf("mutating the returned slice affected the criteria's own state: Authors() = %v", got)
	}
}

func TestSearchCriteria_Kinds_DefaultsToEmptyMeaningBoth(t *testing.T) {
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	if got := criteria.Kinds(); len(got) != 0 {
		t.Fatalf("Kinds() = %v, want empty (meaning both issue and PR)", got)
	}
}
