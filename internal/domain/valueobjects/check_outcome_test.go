package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseCheckOutcome_ParsesEveryKnownPreCompletionStatus(t *testing.T) {
	cases := []struct {
		status string
		want   valueobjects.CheckOutcome
	}{
		{"queued", valueobjects.CheckOutcomeQueued},
		{"in_progress", valueobjects.CheckOutcomeInProgress},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseCheckOutcome(c.status, "")
		if err != nil {
			t.Fatalf("ParseCheckOutcome(%q, \"\"): unexpected error: %v", c.status, err)
		}
		if got != c.want {
			t.Fatalf("ParseCheckOutcome(%q, \"\") = %v, want %v", c.status, got, c.want)
		}
	}
}

func TestParseCheckOutcome_ParsesEveryKnownConclusionWhenStatusIsCompleted(t *testing.T) {
	cases := []struct {
		conclusion string
		want       valueobjects.CheckOutcome
	}{
		{"success", valueobjects.CheckOutcomeSuccess},
		{"failure", valueobjects.CheckOutcomeFailure},
		{"neutral", valueobjects.CheckOutcomeNeutral},
		{"cancelled", valueobjects.CheckOutcomeCancelled},
		{"skipped", valueobjects.CheckOutcomeSkipped},
		{"timed_out", valueobjects.CheckOutcomeTimedOut},
		{"action_required", valueobjects.CheckOutcomeActionRequired},
		{"stale", valueobjects.CheckOutcomeStale},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseCheckOutcome("completed", c.conclusion)
		if err != nil {
			t.Fatalf("ParseCheckOutcome(\"completed\", %q): unexpected error: %v", c.conclusion, err)
		}
		if got != c.want {
			t.Fatalf("ParseCheckOutcome(\"completed\", %q) = %v, want %v", c.conclusion, got, c.want)
		}
	}
}

func TestParseCheckOutcome_RejectsACompletedStatusWithNoConclusion(t *testing.T) {
	_, err := valueobjects.ParseCheckOutcome("completed", "")

	if err == nil {
		t.Fatal("expected an error for a completed status with an empty conclusion, got nil")
	}
}

func TestParseCheckOutcome_RejectsAnUnrecognizedConclusion(t *testing.T) {
	_, err := valueobjects.ParseCheckOutcome("completed", "bogus")

	if err == nil {
		t.Fatal("expected an error for an unrecognized conclusion, got nil")
	}
}

func TestParseCheckOutcome_RejectsAnUnrecognizedStatus(t *testing.T) {
	_, err := valueobjects.ParseCheckOutcome("bogus", "")

	if err == nil {
		t.Fatal("expected an error for an unrecognized status, got nil")
	}
}

func TestCheckOutcome_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.CheckOutcome(99)

	got := unrecognized.String()

	if got != "CheckOutcome(99)" {
		t.Fatalf("String() = %q, want %q", got, "CheckOutcome(99)")
	}
}

func TestCheckOutcome_String_RoundTripsThroughParseCheckOutcomeAsAConclusion(t *testing.T) {
	conclusions := []valueobjects.CheckOutcome{
		valueobjects.CheckOutcomeSuccess,
		valueobjects.CheckOutcomeFailure,
		valueobjects.CheckOutcomeNeutral,
		valueobjects.CheckOutcomeCancelled,
		valueobjects.CheckOutcomeSkipped,
		valueobjects.CheckOutcomeTimedOut,
		valueobjects.CheckOutcomeActionRequired,
		valueobjects.CheckOutcomeStale,
	}

	for _, want := range conclusions {
		got, err := valueobjects.ParseCheckOutcome("completed", want.String())
		if err != nil {
			t.Fatalf("ParseCheckOutcome(\"completed\", %q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the outcome: got %v, want %v", got, want)
		}
	}
}
