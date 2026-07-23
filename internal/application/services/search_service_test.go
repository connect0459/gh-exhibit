package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type fakeIssueSearcher struct {
	results map[string]valueobjects.SearchResult
	err     error

	calls []valueobjects.SearchQuery
}

func (f *fakeIssueSearcher) Search(_ context.Context, query valueobjects.SearchQuery) (valueobjects.SearchResult, error) {
	f.calls = append(f.calls, query)
	if f.err != nil {
		return valueobjects.SearchResult{}, f.err
	}
	return f.results[query.Author()+"|"+query.Assignee()], nil
}

func testSearchCriteria(t *testing.T, authors, assignees []string, limit int) valueobjects.SearchCriteria {
	t.Helper()
	criteria, err := valueobjects.NewSearchCriteria(authors, assignees, nil, nil, nil, limit, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}
	return criteria
}

func testSearchResult(t *testing.T, totalCount int, numbers ...int) valueobjects.SearchResult {
	t.Helper()
	matches := make([]valueobjects.SearchMatch, 0, len(numbers))
	for _, n := range numbers {
		match, err := valueobjects.NewSearchMatch(n, time.Now(), time.Now(), 0)
		if err != nil {
			t.Fatalf("unexpected error building search match: %v", err)
		}
		matches = append(matches, match)
	}
	result, err := valueobjects.NewSearchResult(matches, totalCount)
	if err != nil {
		t.Fatalf("unexpected error building search result: %v", err)
	}
	return result
}

func TestSearchService_Search_ResolvesASingleUnfilteredQueryIntoItsNumbers(t *testing.T) {
	searcher := &fakeIssueSearcher{
		results: map[string]valueobjects.SearchResult{
			"|": testSearchResult(t, 2, 1, 2),
		},
	}
	service := NewSearchService(searcher)
	criteria := testSearchCriteria(t, nil, nil, 100)

	outcome, err := service.Search(context.Background(), "connect0459", "gh-exhibit", criteria)

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(searcher.calls) != 1 {
		t.Fatalf("len(calls) = %d, want 1", len(searcher.calls))
	}
	if len(outcome.Numbers) != 2 {
		t.Fatalf("Numbers = %v, want 2 numbers", outcome.Numbers)
	}
	if outcome.MatchedCount != 2 {
		t.Fatalf("MatchedCount = %d, want 2", outcome.MatchedCount)
	}
	if outcome.ExceededLimit {
		t.Fatal("expected ExceededLimit to be false")
	}
}

func TestSearchService_Search_QueriesOncePerAuthorAssigneeCombinationAndMergesResults(t *testing.T) {
	searcher := &fakeIssueSearcher{
		results: map[string]valueobjects.SearchResult{
			"octocat|":  testSearchResult(t, 1, 1),
			"monalisa|": testSearchResult(t, 1, 2),
		},
	}
	service := NewSearchService(searcher)
	criteria := testSearchCriteria(t, []string{"octocat", "monalisa"}, nil, 100)

	outcome, err := service.Search(context.Background(), "connect0459", "gh-exhibit", criteria)

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(searcher.calls) != 2 {
		t.Fatalf("len(calls) = %d, want 2 (one per author)", len(searcher.calls))
	}
	if len(outcome.Numbers) != 2 {
		t.Fatalf("Numbers = %v, want 2 distinct numbers merged from both queries", outcome.Numbers)
	}
}

func TestSearchService_Search_ReportsExceededLimitWhenMergedMatchesExceedTheCriteriaLimit(t *testing.T) {
	searcher := &fakeIssueSearcher{
		results: map[string]valueobjects.SearchResult{
			"|": testSearchResult(t, 3, 1, 2, 3),
		},
	}
	service := NewSearchService(searcher)
	criteria := testSearchCriteria(t, nil, nil, 1)

	outcome, err := service.Search(context.Background(), "connect0459", "gh-exhibit", criteria)

	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !outcome.ExceededLimit {
		t.Fatal("expected ExceededLimit to be true when merged matches exceed the criteria limit")
	}
	if len(outcome.Numbers) != 1 {
		t.Fatalf("len(Numbers) = %d, want 1 (truncated to the limit)", len(outcome.Numbers))
	}
}

func TestSearchService_Search_ReturnsAWrappedErrorWhenTheSearcherFails(t *testing.T) {
	searcher := &fakeIssueSearcher{err: errors.New("network failure")}
	service := NewSearchService(searcher)
	criteria := testSearchCriteria(t, nil, nil, 100)

	_, err := service.Search(context.Background(), "connect0459", "gh-exhibit", criteria)

	if err == nil {
		t.Fatal("expected an error when the searcher fails, got nil")
	}
}

func TestSearchService_Search_ErrorNamesTheFailingQuerysAuthorAndAssignee(t *testing.T) {
	searcher := &fakeIssueSearcher{err: errors.New("network failure")}
	service := NewSearchService(searcher)
	criteria := testSearchCriteria(t, []string{"octocat"}, nil, 100)

	_, err := service.Search(context.Background(), "connect0459", "gh-exhibit", criteria)

	if err == nil {
		t.Fatal("expected an error when the searcher fails, got nil")
	}
	if !strings.Contains(err.Error(), "octocat") {
		t.Errorf("error = %v, want it to name which author/assignee combination failed, unlike infrastructure/github's own generic wrap", err)
	}
}
