package services_test

import (
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func testSearchCriteria(t *testing.T, authors, assignees []string) valueobjects.SearchCriteria {
	t.Helper()
	criteria, err := valueobjects.NewSearchCriteria(authors, assignees, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}
	return criteria
}

func TestBuildSearchQueries_BuildsOneQueryPerAuthorAssigneeCombination(t *testing.T) {
	criteria := testSearchCriteria(t, []string{"octocat", "monalisa"}, []string{"hubot", "defunkt"})

	queries, err := services.BuildSearchQueries("connect0459", "gh-exhibit", criteria)
	if err != nil {
		t.Fatalf("unexpected error building search queries: %v", err)
	}

	if len(queries) != 4 {
		t.Fatalf("len(queries) = %d, want 4 (2 authors x 2 assignees)", len(queries))
	}

	seen := make(map[string]bool, len(queries))
	for _, q := range queries {
		if q.Owner() != "connect0459" || q.Repo() != "gh-exhibit" {
			t.Fatalf("query owner/repo = %s/%s, want connect0459/gh-exhibit", q.Owner(), q.Repo())
		}
		seen[q.Author()+"|"+q.Assignee()] = true
	}
	for _, want := range []string{"octocat|hubot", "octocat|defunkt", "monalisa|hubot", "monalisa|defunkt"} {
		if !seen[want] {
			t.Fatalf("expected a query for author|assignee combination %q, got %v", want, seen)
		}
	}
}

func TestBuildSearchQueries_BuildsASingleUnfilteredQueryWhenNoAuthorOrAssigneeGiven(t *testing.T) {
	criteria := testSearchCriteria(t, nil, nil)

	queries, err := services.BuildSearchQueries("connect0459", "gh-exhibit", criteria)
	if err != nil {
		t.Fatalf("unexpected error building search queries: %v", err)
	}

	if len(queries) != 1 {
		t.Fatalf("len(queries) = %d, want 1", len(queries))
	}
	if queries[0].Author() != "" || queries[0].Assignee() != "" {
		t.Fatalf("expected an unfiltered query, got author=%q assignee=%q", queries[0].Author(), queries[0].Assignee())
	}
}

func TestBuildSearchQueries_BuildsOneQueryPerAuthorWhenNoAssigneeGiven(t *testing.T) {
	criteria := testSearchCriteria(t, []string{"octocat", "monalisa"}, nil)

	queries, err := services.BuildSearchQueries("connect0459", "gh-exhibit", criteria)
	if err != nil {
		t.Fatalf("unexpected error building search queries: %v", err)
	}

	if len(queries) != 2 {
		t.Fatalf("len(queries) = %d, want 2", len(queries))
	}
	for _, q := range queries {
		if q.Assignee() != "" {
			t.Fatalf("expected an unfiltered assignee, got %q", q.Assignee())
		}
	}
}

func TestBuildSearchQueries_PropagatesSharedFieldsToEveryQuery(t *testing.T) {
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	kinds := []valueobjects.IssueKind{valueobjects.IssueKindPullRequest}
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, kinds, &after, &before, 7, valueobjects.SearchSortByComments, valueobjects.SearchOrderAscending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	queries, err := services.BuildSearchQueries("connect0459", "gh-exhibit", criteria)
	if err != nil {
		t.Fatalf("unexpected error building search queries: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("len(queries) = %d, want 1", len(queries))
	}

	q := queries[0]
	if got := q.Kinds(); len(got) != 1 || got[0] != valueobjects.IssueKindPullRequest {
		t.Fatalf("Kinds() = %v, want [pr]", got)
	}
	if q.CreatedAfter() == nil || !q.CreatedAfter().Equal(after) {
		t.Fatalf("CreatedAfter() = %v, want %v", q.CreatedAfter(), after)
	}
	if q.CreatedBefore() == nil || !q.CreatedBefore().Equal(before) {
		t.Fatalf("CreatedBefore() = %v, want %v", q.CreatedBefore(), before)
	}
	if q.Sort() != valueobjects.SearchSortByComments {
		t.Fatalf("Sort() = %v, want %v", q.Sort(), valueobjects.SearchSortByComments)
	}
	if q.Order() != valueobjects.SearchOrderAscending {
		t.Fatalf("Order() = %v, want %v", q.Order(), valueobjects.SearchOrderAscending)
	}
	if q.MaxResults() != 7 {
		t.Fatalf("MaxResults() = %d, want 7", q.MaxResults())
	}
}

func testSearchMatch(t *testing.T, number int, created, updated time.Time, comments int) valueobjects.SearchMatch {
	t.Helper()
	match, err := valueobjects.NewSearchMatch(number, created, updated, comments)
	if err != nil {
		t.Fatalf("unexpected error building search match: %v", err)
	}
	return match
}

func testSearchResult(t *testing.T, totalCount int, matches ...valueobjects.SearchMatch) valueobjects.SearchResult {
	t.Helper()
	result, err := valueobjects.NewSearchResult(matches, totalCount)
	if err != nil {
		t.Fatalf("unexpected error building search result: %v", err)
	}
	return result
}

func TestMergeSearchResults_DeduplicatesMatchingNumbersAcrossMultipleResults(t *testing.T) {
	now := time.Now()
	a := testSearchMatch(t, 1, now, now, 0)
	b := testSearchMatch(t, 2, now, now, 0)
	criteria := testSearchCriteria(t, nil, nil)

	numbers, matchedCount, exceededLimit := services.MergeSearchResults(
		[]valueobjects.SearchResult{testSearchResult(t, 1, a), testSearchResult(t, 2, a, b)},
		criteria,
	)

	if matchedCount != 2 {
		t.Fatalf("matchedCount = %d, want 2", matchedCount)
	}
	if exceededLimit {
		t.Fatal("expected exceededLimit to be false")
	}
	if len(numbers) != 2 || numbers[0] == numbers[1] {
		t.Fatalf("numbers = %v, want two distinct numbers", numbers)
	}
}

func TestMergeSearchResults_SortsByCreatedDescendingByDefault(t *testing.T) {
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	a := testSearchMatch(t, 1, older, older, 0)
	b := testSearchMatch(t, 2, newer, newer, 0)
	criteria := testSearchCriteria(t, nil, nil)

	numbers, _, _ := services.MergeSearchResults([]valueobjects.SearchResult{testSearchResult(t, 2, a, b)}, criteria)

	if len(numbers) != 2 || numbers[0] != 2 || numbers[1] != 1 {
		t.Fatalf("numbers = %v, want [2 1] (newest created first)", numbers)
	}
}

func TestMergeSearchResults_SortsAscendingWhenOrderIsAscending(t *testing.T) {
	older := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	a := testSearchMatch(t, 1, older, older, 0)
	b := testSearchMatch(t, 2, newer, newer, 0)
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderAscending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	numbers, _, _ := services.MergeSearchResults([]valueobjects.SearchResult{testSearchResult(t, 2, a, b)}, criteria)

	if len(numbers) != 2 || numbers[0] != 1 || numbers[1] != 2 {
		t.Fatalf("numbers = %v, want [1 2] (oldest created first)", numbers)
	}
}

func TestMergeSearchResults_SortsByComments(t *testing.T) {
	now := time.Now()
	a := testSearchMatch(t, 1, now, now, 1)
	b := testSearchMatch(t, 2, now, now, 5)
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, 100, valueobjects.SearchSortByComments, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	numbers, _, _ := services.MergeSearchResults([]valueobjects.SearchResult{testSearchResult(t, 2, a, b)}, criteria)

	if len(numbers) != 2 || numbers[0] != 2 || numbers[1] != 1 {
		t.Fatalf("numbers = %v, want [2 1] (most-commented first)", numbers)
	}
}

func TestMergeSearchResults_TruncatesToTheCriteriaLimitAndReportsExceededLimit(t *testing.T) {
	now := time.Now()
	a := testSearchMatch(t, 1, now, now, 0)
	b := testSearchMatch(t, 2, now, now, 0)
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, 1, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	numbers, matchedCount, exceededLimit := services.MergeSearchResults([]valueobjects.SearchResult{testSearchResult(t, 2, a, b)}, criteria)

	if len(numbers) != 1 {
		t.Fatalf("len(numbers) = %d, want 1 (truncated to the limit)", len(numbers))
	}
	if matchedCount != 2 {
		t.Fatalf("matchedCount = %d, want 2 (the true match count before truncation)", matchedCount)
	}
	if !exceededLimit {
		t.Fatal("expected exceededLimit to be true when the deduped match count exceeds the limit")
	}
}

func TestMergeSearchResults_ReportsExceededLimitWhenAResultsTotalCountExceedsItsOwnMatches(t *testing.T) {
	now := time.Now()
	a := testSearchMatch(t, 1, now, now, 0)
	criteria := testSearchCriteria(t, nil, nil)

	_, _, exceededLimit := services.MergeSearchResults([]valueobjects.SearchResult{testSearchResult(t, 50, a)}, criteria)

	if !exceededLimit {
		t.Fatal("expected exceededLimit to be true when a result's TotalCount exceeds its own returned matches")
	}
}

func TestMergeSearchResults_ReturnsNoNumbersForNoResults(t *testing.T) {
	criteria := testSearchCriteria(t, nil, nil)

	numbers, matchedCount, exceededLimit := services.MergeSearchResults(nil, criteria)

	if len(numbers) != 0 {
		t.Fatalf("numbers = %v, want empty", numbers)
	}
	if matchedCount != 0 {
		t.Fatalf("matchedCount = %d, want 0", matchedCount)
	}
	if exceededLimit {
		t.Fatal("expected exceededLimit to be false for no results")
	}
}
