package services

import (
	"fmt"
	"sort"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// BuildSearchQueries expands criteria into one valueobjects.SearchQuery per
// author/assignee combination. GitHub's search query language rejects OR
// semantics between repeated qualifiers of the same kind ("author:a
// author:b" is AND, not OR), so a criteria naming multiple authors/
// assignees is resolved as separate underlying queries whose results
// MergeSearchResults unions afterward, not as one combined query. An
// unfiltered dimension (no authors, or no assignees) contributes a single
// "" slot, matching SearchQuery's own "empty means unfiltered" convention.
func BuildSearchQueries(owner, repo string, criteria valueobjects.SearchCriteria) ([]valueobjects.SearchQuery, error) {
	authors := criteria.Authors()
	if len(authors) == 0 {
		authors = []string{""}
	}
	assignees := criteria.Assignees()
	if len(assignees) == 0 {
		assignees = []string{""}
	}

	queries := make([]valueobjects.SearchQuery, 0, len(authors)*len(assignees))
	for _, author := range authors {
		for _, assignee := range assignees {
			query, err := valueobjects.NewSearchQuery(
				owner, repo, author, assignee, criteria.Kinds(),
				criteria.CreatedAfter(), criteria.CreatedBefore(),
				criteria.Sort(), criteria.Order(), criteria.Limit(),
			)
			if err != nil {
				return nil, fmt.Errorf("build search query for author %q assignee %q: %w", author, assignee, err)
			}
			queries = append(queries, query)
		}
	}
	return queries, nil
}

// MergeSearchResults merges results (one per BuildSearchQueries query, run
// against the same criteria) into the final ordered, deduplicated,
// limit-truncated issue/PR number list. matchedCount is the best known
// lower bound on the true number of distinct matches, before truncation:
// the deduplicated match count, or — when larger — a single result's own
// GitHub-reported TotalCount (a query returning fewer items than its own
// TotalCount means the true count for that query alone already exceeds
// what was deduplicated from every query combined, so reporting only the
// smaller deduplicated figure would understate the shortfall). exceededLimit
// is true when there is reason to believe more matches exist than numbers
// reflects — either a result's own TotalCount exceeded the matches it
// returned (that query's own results are already incomplete), or the
// deduplicated match count itself exceeded criteria.Limit() (truncated to
// fit) — both mean there is more real data than what is being returned,
// which must be surfaced rather than silently dropped.
func MergeSearchResults(results []valueobjects.SearchResult, criteria valueobjects.SearchCriteria) (numbers []int, matchedCount int, exceededLimit bool) {
	seen := make(map[int]bool)
	var matches []valueobjects.SearchMatch
	maxTotalCount := 0

	for _, result := range results {
		resultMatches := result.Matches()
		if result.TotalCount() > len(resultMatches) {
			exceededLimit = true
		}
		maxTotalCount = max(maxTotalCount, result.TotalCount())
		for _, match := range resultMatches {
			if seen[match.Number()] {
				continue
			}
			seen[match.Number()] = true
			matches = append(matches, match)
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		return lessSearchMatch(matches[i], matches[j], criteria.Sort(), criteria.Order())
	})

	matchedCount = max(len(matches), maxTotalCount)
	if len(matches) > criteria.Limit() {
		exceededLimit = true
		matches = matches[:criteria.Limit()]
	}

	numbers = make([]int, len(matches))
	for i, match := range matches {
		numbers[i] = match.Number()
	}
	return numbers, matchedCount, exceededLimit
}

// lessSearchMatch reports whether a should sort strictly before b under
// field/order, a strict weak ordering (equal values report false in both
// directions, so sort.SliceStable's own stability decides their relative
// order).
func lessSearchMatch(a, b valueobjects.SearchMatch, field valueobjects.SearchSortField, order valueobjects.SearchSortOrder) bool {
	cmp := compareSearchMatch(a, b, field)
	if order == valueobjects.SearchOrderAscending {
		return cmp < 0
	}
	return cmp > 0
}

// compareSearchMatch returns a negative number if a < b, a positive number
// if a > b, or zero if they are equal under field.
func compareSearchMatch(a, b valueobjects.SearchMatch, field valueobjects.SearchSortField) int {
	switch field {
	case valueobjects.SearchSortByCreated:
		return compareTime(a.CreatedAt(), b.CreatedAt())
	case valueobjects.SearchSortByUpdated:
		return compareTime(a.UpdatedAt(), b.UpdatedAt())
	case valueobjects.SearchSortByComments:
		return a.Comments() - b.Comments()
	default:
		return compareTime(a.CreatedAt(), b.CreatedAt())
	}
}

// compareTime returns a negative number if a < b, a positive number if
// a > b, or zero if they are equal.
func compareTime(a, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}
