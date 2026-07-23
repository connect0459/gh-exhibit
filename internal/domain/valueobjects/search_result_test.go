package valueobjects_test

import (
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestNewSearchMatch_RejectsANonPositiveNumber(t *testing.T) {
	_, err := valueobjects.NewSearchMatch(0, time.Now(), time.Now(), 0)

	if err == nil {
		t.Fatal("expected an error for a non-positive number, got nil")
	}
}

func TestNewSearchMatch_RejectsNegativeComments(t *testing.T) {
	_, err := valueobjects.NewSearchMatch(1, time.Now(), time.Now(), -1)

	if err == nil {
		t.Fatal("expected an error for a negative comment count, got nil")
	}
}

func TestSearchMatch_Accessors_ReturnTheConstructedValues(t *testing.T) {
	created := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	updated := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)

	match, err := valueobjects.NewSearchMatch(42, created, updated, 3)
	if err != nil {
		t.Fatalf("unexpected error building search match: %v", err)
	}

	if got := match.Number(); got != 42 {
		t.Fatalf("Number() = %d, want 42", got)
	}
	if got := match.CreatedAt(); !got.Equal(created) {
		t.Fatalf("CreatedAt() = %v, want %v", got, created)
	}
	if got := match.UpdatedAt(); !got.Equal(updated) {
		t.Fatalf("UpdatedAt() = %v, want %v", got, updated)
	}
	if got := match.Comments(); got != 3 {
		t.Fatalf("Comments() = %d, want 3", got)
	}
}

func TestNewSearchResult_RejectsANegativeTotalCount(t *testing.T) {
	_, err := valueobjects.NewSearchResult(nil, -1)

	if err == nil {
		t.Fatal("expected an error for a negative total count, got nil")
	}
}

func TestNewSearchResult_RejectsATotalCountLessThanTheMatchCount(t *testing.T) {
	match, err := valueobjects.NewSearchMatch(1, time.Now(), time.Now(), 0)
	if err != nil {
		t.Fatalf("unexpected error building search match: %v", err)
	}

	_, err = valueobjects.NewSearchResult([]valueobjects.SearchMatch{match, match}, 1)

	if err == nil {
		t.Fatal("expected an error when total count is less than the number of matches, got nil")
	}
}

func TestNewSearchResult_AcceptsATotalCountExceedingTheMatchCount(t *testing.T) {
	match, err := valueobjects.NewSearchMatch(1, time.Now(), time.Now(), 0)
	if err != nil {
		t.Fatalf("unexpected error building search match: %v", err)
	}

	result, err := valueobjects.NewSearchResult([]valueobjects.SearchMatch{match}, 50)
	if err != nil {
		t.Fatalf("unexpected error when total count exceeds the match count: %v", err)
	}
	if got := result.TotalCount(); got != 50 {
		t.Fatalf("TotalCount() = %d, want 50", got)
	}
}

func TestSearchResult_Matches_ReturnsADefensiveCopy(t *testing.T) {
	match, err := valueobjects.NewSearchMatch(1, time.Now(), time.Now(), 0)
	if err != nil {
		t.Fatalf("unexpected error building search match: %v", err)
	}
	result, err := valueobjects.NewSearchResult([]valueobjects.SearchMatch{match}, 1)
	if err != nil {
		t.Fatalf("unexpected error building search result: %v", err)
	}

	matches := result.Matches()
	matches[0], err = valueobjects.NewSearchMatch(999, time.Now(), time.Now(), 0)
	if err != nil {
		t.Fatalf("unexpected error building search match: %v", err)
	}

	if got := result.Matches()[0].Number(); got != 1 {
		t.Fatalf("mutating the returned slice affected the result's own state: Number() = %d", got)
	}
}
