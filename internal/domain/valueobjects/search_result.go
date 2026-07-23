package valueobjects

import (
	"errors"
	"time"
)

// SearchMatch is one item GitHub's search endpoint matched, carrying just
// enough for domain/services to merge, sort, and deduplicate matches from
// multiple SearchQuery calls before handing a final number list back to the
// existing per-ref export pipeline. Unlike EvidenceFetcher's raw
// json.RawMessage results, this is already decoded: a search match is never
// persisted to evidence/*.json — the resolved number is re-fetched through
// EvidenceFetcher like any other export — so there is no verbatim-JSON
// fidelity requirement to preserve here.
type SearchMatch struct {
	number    int
	createdAt time.Time
	updatedAt time.Time
	comments  int
}

// NewSearchMatch constructs a SearchMatch. number must be positive and
// comments must not be negative.
func NewSearchMatch(number int, createdAt, updatedAt time.Time, comments int) (SearchMatch, error) {
	if number <= 0 {
		return SearchMatch{}, errors.New("search match number must be positive")
	}
	if comments < 0 {
		return SearchMatch{}, errors.New("search match comments must not be negative")
	}
	return SearchMatch{number: number, createdAt: createdAt, updatedAt: updatedAt, comments: comments}, nil
}

// Number returns the matched issue/PR number.
func (m SearchMatch) Number() int {
	return m.number
}

// CreatedAt returns the matched issue/PR's creation time.
func (m SearchMatch) CreatedAt() time.Time {
	return m.createdAt
}

// UpdatedAt returns the matched issue/PR's last-updated time.
func (m SearchMatch) UpdatedAt() time.Time {
	return m.updatedAt
}

// Comments returns the matched issue/PR's comment count.
func (m SearchMatch) Comments() int {
	return m.comments
}

// SearchResult is one SearchQuery's outcome. TotalCount is GitHub's own
// reported match count for that query, which can exceed len(Matches) when
// more issues/PRs matched than the query's own MaxResults asked for.
type SearchResult struct {
	matches    []SearchMatch
	totalCount int
}

// NewSearchResult constructs a SearchResult. totalCount must not be
// negative, and must not be less than len(matches) — GitHub's own reported
// total can never be smaller than the matches actually returned for it.
func NewSearchResult(matches []SearchMatch, totalCount int) (SearchResult, error) {
	if totalCount < 0 {
		return SearchResult{}, errors.New("search result total count must not be negative")
	}
	if totalCount < len(matches) {
		return SearchResult{}, errors.New("search result total count must not be less than the number of matches")
	}
	return SearchResult{matches: append([]SearchMatch(nil), matches...), totalCount: totalCount}, nil
}

// Matches returns a defensive copy of the matched items (at most the
// originating query's MaxResults).
func (r SearchResult) Matches() []SearchMatch {
	return append([]SearchMatch(nil), r.matches...)
}

// TotalCount returns GitHub's own reported match count for the originating
// query.
func (r SearchResult) TotalCount() int {
	return r.totalCount
}
