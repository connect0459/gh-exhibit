package valueobjects

import "fmt"

// SearchSortField is which field a filter-mode search orders its matches
// by, mirroring the subset of GitHub search's own "sort" values gh-exhibit
// exposes.
type SearchSortField int

const (
	SearchSortByCreated SearchSortField = iota
	SearchSortByUpdated
	SearchSortByComments
)

// ParseSearchSortField parses gh-exhibit's own filter spelling ("created",
// "updated", "comments") into a SearchSortField. It returns an error for
// any other value.
func ParseSearchSortField(raw string) (SearchSortField, error) {
	switch raw {
	case "created":
		return SearchSortByCreated, nil
	case "updated":
		return SearchSortByUpdated, nil
	case "comments":
		return SearchSortByComments, nil
	default:
		return 0, fmt.Errorf("unrecognized search sort field %q", raw)
	}
}

// String returns f's gh-exhibit filter spelling (e.g. "comments").
func (f SearchSortField) String() string {
	switch f {
	case SearchSortByCreated:
		return "created"
	case SearchSortByUpdated:
		return "updated"
	case SearchSortByComments:
		return "comments"
	default:
		return fmt.Sprintf("SearchSortField(%d)", int(f))
	}
}

// valid reports whether f is one of SearchSortField's own defined
// constants, guarding against a value built by bypassing
// ParseSearchSortField (e.g. a raw int conversion).
func (f SearchSortField) valid() bool {
	switch f {
	case SearchSortByCreated, SearchSortByUpdated, SearchSortByComments:
		return true
	default:
		return false
	}
}

// SearchSortOrder is the ascending/descending direction a filter-mode
// search's matches are ordered in, independent of which field
// (SearchSortField) they are ordered by.
type SearchSortOrder int

const (
	SearchOrderDescending SearchSortOrder = iota
	SearchOrderAscending
)

// ParseSearchSortOrder parses gh-exhibit's own filter spelling ("asc",
// "desc") into a SearchSortOrder. It returns an error for any other value.
func ParseSearchSortOrder(raw string) (SearchSortOrder, error) {
	switch raw {
	case "asc":
		return SearchOrderAscending, nil
	case "desc":
		return SearchOrderDescending, nil
	default:
		return 0, fmt.Errorf("unrecognized search sort order %q", raw)
	}
}

// String returns o's gh-exhibit filter spelling (e.g. "asc").
func (o SearchSortOrder) String() string {
	switch o {
	case SearchOrderAscending:
		return "asc"
	case SearchOrderDescending:
		return "desc"
	default:
		return fmt.Sprintf("SearchSortOrder(%d)", int(o))
	}
}

// valid reports whether o is one of SearchSortOrder's own defined
// constants, guarding against a value built by bypassing
// ParseSearchSortOrder (e.g. a raw int conversion).
func (o SearchSortOrder) valid() bool {
	switch o {
	case SearchOrderAscending, SearchOrderDescending:
		return true
	default:
		return false
	}
}
