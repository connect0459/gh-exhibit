package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseAssignmentAction_ParsesEveryKnownGitHubAssignmentAction(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.AssignmentAction
	}{
		{"assigned", valueobjects.AssignmentActionAssigned},
		{"unassigned", valueobjects.AssignmentActionUnassigned},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseAssignmentAction(c.raw)
		if err != nil {
			t.Fatalf("ParseAssignmentAction(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseAssignmentAction(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseAssignmentAction_RejectsAnUnrecognizedAction(t *testing.T) {
	_, err := valueobjects.ParseAssignmentAction("reassigned")

	if err == nil {
		t.Fatal("expected an error for an unrecognized assignment action, got nil")
	}
}

func TestAssignmentAction_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.AssignmentAction(99)

	got := unrecognized.String()

	if got != "AssignmentAction(99)" {
		t.Fatalf("String() = %q, want %q", got, "AssignmentAction(99)")
	}
}

func TestAssignmentAction_String_RoundTripsThroughParseAssignmentAction(t *testing.T) {
	actions := []valueobjects.AssignmentAction{
		valueobjects.AssignmentActionAssigned,
		valueobjects.AssignmentActionUnassigned,
	}

	for _, want := range actions {
		got, err := valueobjects.ParseAssignmentAction(want.String())
		if err != nil {
			t.Fatalf("ParseAssignmentAction(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the action: got %v, want %v", got, want)
		}
	}
}
