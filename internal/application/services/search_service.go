package services

import (
	"context"
	"fmt"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// SearchOutcome is Search's result: Numbers is the final, ordered,
// deduplicated, limit-truncated issue/PR number list; MatchedCount is the
// true number of distinct matches found before truncation; ExceededLimit
// is true when there is reason to believe more matches exist than Numbers
// reflects (see domain/services.MergeSearchResults).
type SearchOutcome struct {
	Numbers       []int
	MatchedCount  int
	ExceededLimit bool
}

// SearchService resolves a criteria-based filter-mode selection into the
// concrete issue/PR numbers it matches, the search-based counterpart to
// ExportService's number-addressed export.
type SearchService struct {
	searcher repositories.IssueSearcher
}

// NewSearchService builds a SearchService from its one collaborating port
// (dependency inversion — this constructor takes an abstract type, not an
// infrastructure-layer concrete implementation).
func NewSearchService(searcher repositories.IssueSearcher) *SearchService {
	return &SearchService{searcher: searcher}
}

// Search resolves criteria against owner/repo: it expands criteria into
// one underlying IssueSearcher.Search call per author/assignee combination
// (see services.BuildSearchQueries), then merges every call's results into
// a single SearchOutcome (see services.MergeSearchResults). Any single
// underlying call failing aborts the whole search and returns a wrapped
// error.
func (s *SearchService) Search(ctx context.Context, owner, repo string, criteria valueobjects.SearchCriteria) (SearchOutcome, error) {
	queries, err := services.BuildSearchQueries(owner, repo, criteria)
	if err != nil {
		return SearchOutcome{}, fmt.Errorf("build search queries: %w", err)
	}

	results := make([]valueobjects.SearchResult, 0, len(queries))
	for _, query := range queries {
		result, err := s.searcher.Search(ctx, query)
		if err != nil {
			return SearchOutcome{}, fmt.Errorf("search GitHub issues/PRs: %w", err)
		}
		results = append(results, result)
	}

	numbers, matchedCount, exceededLimit := services.MergeSearchResults(results, criteria)
	return SearchOutcome{Numbers: numbers, MatchedCount: matchedCount, ExceededLimit: exceededLimit}, nil
}
