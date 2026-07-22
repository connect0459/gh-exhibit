package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseClosureAction_ParsesEveryKnownGitHubClosureAction(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.ClosureAction
	}{
		{"closed", valueobjects.ClosureActionClosed},
		{"reopened", valueobjects.ClosureActionReopened},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseClosureAction(c.raw)
		if err != nil {
			t.Fatalf("ParseClosureAction(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseClosureAction(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseClosureAction_RejectsAnUnrecognizedAction(t *testing.T) {
	_, err := valueobjects.ParseClosureAction("merged")

	if err == nil {
		t.Fatal("expected an error for an unrecognized closure action, got nil")
	}
}

func TestClosureAction_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.ClosureAction(99)

	got := unrecognized.String()

	if got != "ClosureAction(99)" {
		t.Fatalf("String() = %q, want %q", got, "ClosureAction(99)")
	}
}

func TestClosureAction_String_RoundTripsThroughParseClosureAction(t *testing.T) {
	actions := []valueobjects.ClosureAction{
		valueobjects.ClosureActionClosed,
		valueobjects.ClosureActionReopened,
	}

	for _, want := range actions {
		got, err := valueobjects.ParseClosureAction(want.String())
		if err != nil {
			t.Fatalf("ParseClosureAction(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the action: got %v, want %v", got, want)
		}
	}
}
