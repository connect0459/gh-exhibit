package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseSearchSortField_ParsesEveryKnownSortField(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.SearchSortField
	}{
		{"created", valueobjects.SearchSortByCreated},
		{"updated", valueobjects.SearchSortByUpdated},
		{"comments", valueobjects.SearchSortByComments},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseSearchSortField(c.raw)
		if err != nil {
			t.Fatalf("ParseSearchSortField(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseSearchSortField(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseSearchSortField_RejectsAnUnrecognizedField(t *testing.T) {
	_, err := valueobjects.ParseSearchSortField("reactions")

	if err == nil {
		t.Fatal("expected an error for an unrecognized sort field, got nil")
	}
}

func TestSearchSortField_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.SearchSortField(99)

	got := unrecognized.String()

	if got != "SearchSortField(99)" {
		t.Fatalf("String() = %q, want %q", got, "SearchSortField(99)")
	}
}

func TestSearchSortField_String_RoundTripsThroughParseSearchSortField(t *testing.T) {
	fields := []valueobjects.SearchSortField{
		valueobjects.SearchSortByCreated,
		valueobjects.SearchSortByUpdated,
		valueobjects.SearchSortByComments,
	}

	for _, want := range fields {
		got, err := valueobjects.ParseSearchSortField(want.String())
		if err != nil {
			t.Fatalf("ParseSearchSortField(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the sort field: got %v, want %v", got, want)
		}
	}
}

func TestParseSearchSortOrder_ParsesEveryKnownOrder(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.SearchSortOrder
	}{
		{"asc", valueobjects.SearchOrderAscending},
		{"desc", valueobjects.SearchOrderDescending},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseSearchSortOrder(c.raw)
		if err != nil {
			t.Fatalf("ParseSearchSortOrder(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseSearchSortOrder(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseSearchSortOrder_RejectsAnUnrecognizedOrder(t *testing.T) {
	_, err := valueobjects.ParseSearchSortOrder("random")

	if err == nil {
		t.Fatal("expected an error for an unrecognized sort order, got nil")
	}
}

func TestSearchSortOrder_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.SearchSortOrder(99)

	got := unrecognized.String()

	if got != "SearchSortOrder(99)" {
		t.Fatalf("String() = %q, want %q", got, "SearchSortOrder(99)")
	}
}

func TestSearchSortOrder_String_RoundTripsThroughParseSearchSortOrder(t *testing.T) {
	orders := []valueobjects.SearchSortOrder{
		valueobjects.SearchOrderAscending,
		valueobjects.SearchOrderDescending,
	}

	for _, want := range orders {
		got, err := valueobjects.ParseSearchSortOrder(want.String())
		if err != nil {
			t.Fatalf("ParseSearchSortOrder(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the sort order: got %v, want %v", got, want)
		}
	}
}
