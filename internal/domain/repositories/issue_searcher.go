package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// IssueSearcher is the abstract port the application layer depends on to
// resolve a single SearchQuery into matching issue/PR numbers — the
// search-based counterpart to EvidenceFetcher's number-addressed fetches.
// A SearchCriteria naming multiple authors/assignees is expanded into
// multiple SearchQuery calls by domain/services.BuildSearchQueries; this
// port itself only ever sees one unambiguous query at a time.
type IssueSearcher interface {
	// Search executes query against GitHub's search API and returns its
	// matches, at most query.MaxResults() of them.
	Search(ctx context.Context, query valueobjects.SearchQuery) (valueobjects.SearchResult, error)
}
