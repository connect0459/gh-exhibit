package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestNewSearchCriteria_AcceptsAMinimalCriteria(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err != nil {
		t.Fatalf("unexpected error for a minimal (unfiltered) criteria: %v", err)
	}
}

func TestNewSearchCriteria_RejectsAnInvalidAuthorLogin(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria([]string{"owner/evil"}, nil, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for an author login containing a slash, got nil")
	}
}

func TestNewSearchCriteria_RejectsAnInvalidAssigneeLogin(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, []string{""}, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for an empty assignee login, got nil")
	}
}

func TestNewSearchCriteria_RejectsAnInvalidReviewRequestedLogin(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, []string{"owner/evil"}, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a review-requested login containing a slash, got nil")
	}
}

func TestNewSearchCriteria_RejectsAnInvalidReviewedByLogin(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, []string{""}, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for an empty reviewed-by login, got nil")
	}
}

func TestNewSearchCriteria_InvalidAuthorErrorNamesSearchCriteriaNotIssueRef(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria([]string{"bad/login"}, nil, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for an author login containing a slash, got nil")
	}
	if !strings.Contains(err.Error(), "search criteria author") {
		t.Errorf("error = %v, want it to name \"search criteria author\"", err)
	}
	if strings.Contains(err.Error(), "issue ref") {
		t.Errorf("error = %v, want it not to leak IssueRef's own wording", err)
	}
}

func TestNewSearchCriteria_InvalidAssigneeErrorNamesSearchCriteriaNotIssueRef(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, []string{"bad/login"}, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for an assignee login containing a slash, got nil")
	}
	if !strings.Contains(err.Error(), "search criteria assignee") {
		t.Errorf("error = %v, want it to name \"search criteria assignee\"", err)
	}
	if strings.Contains(err.Error(), "issue ref") {
		t.Errorf("error = %v, want it not to leak IssueRef's own wording", err)
	}
}

func TestNewSearchCriteria_InvalidReviewRequestedErrorNamesSearchCriteriaNotIssueRef(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, []string{"bad/login"}, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a review-requested login containing a slash, got nil")
	}
	if !strings.Contains(err.Error(), "search criteria review-requested") {
		t.Errorf("error = %v, want it to name \"search criteria review-requested\"", err)
	}
	if strings.Contains(err.Error(), "issue ref") {
		t.Errorf("error = %v, want it not to leak IssueRef's own wording", err)
	}
}

func TestNewSearchCriteria_InvalidReviewedByErrorNamesSearchCriteriaNotIssueRef(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, []string{"bad/login"}, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a reviewed-by login containing a slash, got nil")
	}
	if !strings.Contains(err.Error(), "search criteria reviewed-by") {
		t.Errorf("error = %v, want it to name \"search criteria reviewed-by\"", err)
	}
	if strings.Contains(err.Error(), "issue ref") {
		t.Errorf("error = %v, want it not to leak IssueRef's own wording", err)
	}
}

func TestNewSearchCriteria_RejectsAnOutOfRangeSortField(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortField(99), valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a sort field built by bypassing ParseSearchSortField, got nil")
	}
}

func TestNewSearchCriteria_RejectsAnOutOfRangeOrder(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchSortOrder(99))

	if err == nil {
		t.Fatal("expected an error for an order built by bypassing ParseSearchSortOrder, got nil")
	}
}

func TestNewSearchCriteria_RejectsAnOutOfRangeKind(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, []valueobjects.IssueKind{valueobjects.IssueKind(99)}, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a kind built by bypassing ParseIssueKind, got nil")
	}
}

func TestNewSearchCriteria_RejectsCreatedAfterLaterThanCreatedBefore(t *testing.T) {
	after := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, &after, &before, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error when createdAfter is later than createdBefore, got nil")
	}
}

func TestNewSearchCriteria_AcceptsCreatedAfterEqualToCreatedBefore(t *testing.T) {
	same := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, &same, &same, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err != nil {
		t.Fatalf("unexpected error when createdAfter equals createdBefore: %v", err)
	}
}

func TestNewSearchCriteria_RejectsALimitOfZero(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, nil, 0, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a limit of zero, got nil")
	}
}

func TestNewSearchCriteria_RejectsALimitAboveMaxSearchLimit(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, nil, valueobjects.MaxSearchLimit+1, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err == nil {
		t.Fatal("expected an error for a limit above MaxSearchLimit, got nil")
	}
}

func TestNewSearchCriteria_AcceptsALimitEqualToMaxSearchLimit(t *testing.T) {
	_, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, nil, valueobjects.MaxSearchLimit, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)

	if err != nil {
		t.Fatalf("unexpected error for a limit equal to MaxSearchLimit: %v", err)
	}
}

func TestSearchCriteria_Accessors_ReturnTheConstructedValues(t *testing.T) {
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	kinds := []valueobjects.IssueKind{valueobjects.IssueKindPullRequest}

	criteria, err := valueobjects.NewSearchCriteria(
		[]string{"octocat"}, []string{"monalisa"}, []string{"hubot"}, []string{"github-actions"},
		kinds, &after, &before, 42,
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
	if got := criteria.ReviewRequested(); len(got) != 1 || got[0] != "hubot" {
		t.Fatalf("ReviewRequested() = %v, want [hubot]", got)
	}
	if got := criteria.ReviewedBy(); len(got) != 1 || got[0] != "github-actions" {
		t.Fatalf("ReviewedBy() = %v, want [github-actions]", got)
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
	criteria, err := valueobjects.NewSearchCriteria([]string{"octocat"}, nil, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	authors := criteria.Authors()
	authors[0] = "mutated"

	if got := criteria.Authors(); got[0] != "octocat" {
		t.Fatalf("mutating the returned slice affected the criteria's own state: Authors() = %v", got)
	}
}

func TestSearchCriteria_ReviewRequested_ReturnsADefensiveCopy(t *testing.T) {
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, []string{"octocat"}, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	reviewRequested := criteria.ReviewRequested()
	reviewRequested[0] = "mutated"

	if got := criteria.ReviewRequested(); got[0] != "octocat" {
		t.Fatalf("mutating the returned slice affected the criteria's own state: ReviewRequested() = %v", got)
	}
}

func TestSearchCriteria_ReviewedBy_ReturnsADefensiveCopy(t *testing.T) {
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, []string{"octocat"}, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	reviewedBy := criteria.ReviewedBy()
	reviewedBy[0] = "mutated"

	if got := criteria.ReviewedBy(); got[0] != "octocat" {
		t.Fatalf("mutating the returned slice affected the criteria's own state: ReviewedBy() = %v", got)
	}
}

func TestSearchCriteria_CreatedAfter_ReturnsADefensiveCopy(t *testing.T) {
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, &after, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	got := criteria.CreatedAfter()
	*got = got.Add(24 * time.Hour)

	if want := criteria.CreatedAfter(); !want.Equal(after) {
		t.Fatalf("mutating the returned pointer affected the criteria's own state: CreatedAfter() = %v, want %v", want, after)
	}
}

func TestNewSearchCriteria_DoesNotAliasTheCallersCreatedBeforePointer(t *testing.T) {
	original := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	before := original
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, &before, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	before = before.Add(24 * time.Hour)

	if got := criteria.CreatedBefore(); got == nil || !got.Equal(original) {
		t.Fatalf("mutating the caller's own pointer after construction affected the criteria's state: CreatedBefore() = %v, want %v", got, original)
	}
}

func TestSearchCriteria_Kinds_DefaultsToEmptyMeaningBoth(t *testing.T) {
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	if got := criteria.Kinds(); len(got) != 0 {
		t.Fatalf("Kinds() = %v, want empty (meaning both issue and PR)", got)
	}
}
