package valueobjects

import (
	"fmt"
	"time"
)

// MaxSearchLimit is gh-exhibit's own ceiling on filter-mode's --limit flag —
// deliberately far below GitHub search's raw 1000-result cap, so that a
// filter-mode invocation with no author/assignee/date narrowing at all
// cannot become a de facto whole-repository sweep, the "--all" behavior
// this project's number-list mode has always deliberately lacked. It also
// happens to match GitHub search's own maximum page size, so a single
// underlying query is always exactly one HTTP request.
const MaxSearchLimit = 100

// DefaultSearchLimit is filter-mode's --limit default when the flag is
// omitted — the same as MaxSearchLimit, since GitHub search's own page
// size ceiling means there is no smaller "natural" default to prefer.
const DefaultSearchLimit = MaxSearchLimit

// SearchDateLayout is the calendar-day precision filter mode's created-date
// bounds use throughout: presentation/cli parses --after/--before with it,
// and infrastructure/github formats the same bounds into GitHub search's
// own "created:" qualifier with it — one shared constant rather than two
// independently-defined copies of the same layout string for two unrelated
// reasons that happen to agree today.
const SearchDateLayout = "2006-01-02"

// SearchCriteria is the validated, immutable shape of a criteria-based
// export selection: which issues/PRs to resolve via GitHub's search API
// rather than an explicit number list. Authors, assignees, and kinds may
// each name more than one value — GitHub's search query language has no OR
// semantics between repeated qualifiers of the same kind, so resolving a
// multi-valued criteria into underlying GitHub queries (one per
// author/assignee combination) is domain/services.BuildSearchQueries' job,
// not this type's.
type SearchCriteria struct {
	authors       []string
	assignees     []string
	kinds         []IssueKind
	createdAfter  *time.Time
	createdBefore *time.Time
	limit         int
	sort          SearchSortField
	order         SearchSortOrder
}

// NewSearchCriteria constructs a SearchCriteria from its filter values.
// authors and assignees, when non-empty, must each be a valid GitHub login
// (the same rule NewIssueRef applies to an owner). kinds, when non-empty,
// restricts matches to the named issue/PR kinds; empty means both. sort and
// order must each be one of their own package-level constants (e.g.
// SearchSortByCreated, SearchOrderDescending) — this is enforced here, not
// only by ParseSearchSortField/ParseSearchSortOrder, so an out-of-range
// value can never reach domain/services.MergeSearchResults regardless of
// how a SearchCriteria was built. createdAfter/createdBefore are inclusive
// bounds on the created-date range, either or both of which may be nil;
// when both are given, createdAfter must not be later than createdBefore.
// limit must be between 1 and MaxSearchLimit inclusive.
func NewSearchCriteria(authors, assignees []string, kinds []IssueKind, createdAfter, createdBefore *time.Time, limit int, sort SearchSortField, order SearchSortOrder) (SearchCriteria, error) {
	for _, author := range authors {
		if err := validateOwner(author, "search criteria author"); err != nil {
			return SearchCriteria{}, err
		}
	}
	for _, assignee := range assignees {
		if err := validateOwner(assignee, "search criteria assignee"); err != nil {
			return SearchCriteria{}, err
		}
	}
	for _, kind := range kinds {
		if !kind.valid() {
			return SearchCriteria{}, fmt.Errorf("search criteria kind %s is not a recognized value", kind)
		}
	}
	if !sort.valid() {
		return SearchCriteria{}, fmt.Errorf("search criteria sort %s is not a recognized value", sort)
	}
	if !order.valid() {
		return SearchCriteria{}, fmt.Errorf("search criteria order %s is not a recognized value", order)
	}
	if createdAfter != nil && createdBefore != nil && createdAfter.After(*createdBefore) {
		return SearchCriteria{}, fmt.Errorf("search criteria created-after %s must not be later than created-before %s", createdAfter, createdBefore)
	}
	if limit < 1 || limit > MaxSearchLimit {
		return SearchCriteria{}, fmt.Errorf("search criteria limit must be between 1 and %d, got %d", MaxSearchLimit, limit)
	}

	return SearchCriteria{
		authors:       append([]string(nil), authors...),
		assignees:     append([]string(nil), assignees...),
		kinds:         append([]IssueKind(nil), kinds...),
		createdAfter:  createdAfter,
		createdBefore: createdBefore,
		limit:         limit,
		sort:          sort,
		order:         order,
	}, nil
}

// Authors returns a defensive copy of the author logins to match (empty
// means unfiltered by author).
func (c SearchCriteria) Authors() []string {
	return append([]string(nil), c.authors...)
}

// Assignees returns a defensive copy of the assignee logins to match
// (empty means unfiltered by assignee).
func (c SearchCriteria) Assignees() []string {
	return append([]string(nil), c.assignees...)
}

// Kinds returns a defensive copy of the issue/PR kinds to match (empty
// means both).
func (c SearchCriteria) Kinds() []IssueKind {
	return append([]IssueKind(nil), c.kinds...)
}

// CreatedAfter returns the inclusive lower bound on the created-date range,
// or nil when unset.
func (c SearchCriteria) CreatedAfter() *time.Time {
	return c.createdAfter
}

// CreatedBefore returns the inclusive upper bound on the created-date
// range, or nil when unset.
func (c SearchCriteria) CreatedBefore() *time.Time {
	return c.createdBefore
}

// Limit returns the maximum number of matches to resolve.
func (c SearchCriteria) Limit() int {
	return c.limit
}

// Sort returns which field matches are ordered by.
func (c SearchCriteria) Sort() SearchSortField {
	return c.sort
}

// Order returns the ascending/descending direction matches are ordered in.
func (c SearchCriteria) Order() SearchSortOrder {
	return c.order
}
