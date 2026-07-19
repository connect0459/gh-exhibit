package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseReviewState_ParsesEveryKnownGitHubReviewState(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.ReviewState
	}{
		{"approved", valueobjects.ReviewStateApproved},
		{"changes_requested", valueobjects.ReviewStateChangesRequested},
		{"commented", valueobjects.ReviewStateCommented},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseReviewState(c.raw)
		if err != nil {
			t.Fatalf("ParseReviewState(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseReviewState(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseReviewState_RejectsAnUnrecognizedState(t *testing.T) {
	_, err := valueobjects.ParseReviewState("dismissed")

	if err == nil {
		t.Fatal("expected an error for an unrecognized review state, got nil")
	}
}

func TestReviewState_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.ReviewState(99)

	got := unrecognized.String()

	if got != "ReviewState(99)" {
		t.Fatalf("String() = %q, want %q", got, "ReviewState(99)")
	}
}

func TestReviewState_String_RoundTripsThroughParseReviewState(t *testing.T) {
	states := []valueobjects.ReviewState{
		valueobjects.ReviewStateApproved,
		valueobjects.ReviewStateChangesRequested,
		valueobjects.ReviewStateCommented,
	}

	for _, want := range states {
		got, err := valueobjects.ParseReviewState(want.String())
		if err != nil {
			t.Fatalf("ParseReviewState(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the state: got %v, want %v", got, want)
		}
	}
}
