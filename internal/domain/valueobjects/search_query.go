package valueobjects

import (
	"errors"
	"time"
)

// SearchQuery is a single, unambiguous GitHub search request: at most one
// author and one assignee (never a list). GitHub's search query language
// rejects OR semantics between repeated qualifiers of the same kind (e.g.
// "author:a author:b" is AND, not OR), so a SearchCriteria naming multiple
// authors/assignees is expanded into one SearchQuery per combination by
// domain/services.BuildSearchQueries before infrastructure ever sees one;
// this type is not built directly from CLI input.
type SearchQuery struct {
	owner         string
	repo          string
	author        string // "" means unfiltered by author
	assignee      string // "" means unfiltered by assignee
	kinds         []IssueKind
	createdAfter  *time.Time
	createdBefore *time.Time
	sort          SearchSortField
	order         SearchSortOrder
	maxResults    int
}

// NewSearchQuery constructs a SearchQuery. owner and repo must be
// non-empty; author and assignee may be empty, meaning that dimension is
// unfiltered. maxResults must be positive.
func NewSearchQuery(owner, repo, author, assignee string, kinds []IssueKind, createdAfter, createdBefore *time.Time, sort SearchSortField, order SearchSortOrder, maxResults int) (SearchQuery, error) {
	if owner == "" {
		return SearchQuery{}, errors.New("search query owner must not be empty")
	}
	if repo == "" {
		return SearchQuery{}, errors.New("search query repo must not be empty")
	}
	if maxResults <= 0 {
		return SearchQuery{}, errors.New("search query max results must be positive")
	}

	return SearchQuery{
		owner:         owner,
		repo:          repo,
		author:        author,
		assignee:      assignee,
		kinds:         append([]IssueKind(nil), kinds...),
		createdAfter:  createdAfter,
		createdBefore: createdBefore,
		sort:          sort,
		order:         order,
		maxResults:    maxResults,
	}, nil
}

// Owner returns the target repository's owner.
func (q SearchQuery) Owner() string {
	return q.owner
}

// Repo returns the target repository name.
func (q SearchQuery) Repo() string {
	return q.repo
}

// Author returns the single author login to filter by, or "" when
// unfiltered by author.
func (q SearchQuery) Author() string {
	return q.author
}

// Assignee returns the single assignee login to filter by, or "" when
// unfiltered by assignee.
func (q SearchQuery) Assignee() string {
	return q.assignee
}

// Kinds returns a defensive copy of the issue/PR kinds to match (empty
// means both).
func (q SearchQuery) Kinds() []IssueKind {
	return append([]IssueKind(nil), q.kinds...)
}

// CreatedAfter returns the inclusive lower bound on the created-date range,
// or nil when unset.
func (q SearchQuery) CreatedAfter() *time.Time {
	return q.createdAfter
}

// CreatedBefore returns the inclusive upper bound on the created-date
// range, or nil when unset.
func (q SearchQuery) CreatedBefore() *time.Time {
	return q.createdBefore
}

// Sort returns which field matches are ordered by.
func (q SearchQuery) Sort() SearchSortField {
	return q.sort
}

// Order returns the ascending/descending direction matches are ordered in.
func (q SearchQuery) Order() SearchSortOrder {
	return q.order
}

// MaxResults returns the maximum number of matches this single query
// should fetch.
func (q SearchQuery) MaxResults() int {
	return q.maxResults
}
