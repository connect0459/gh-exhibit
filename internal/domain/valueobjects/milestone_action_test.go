package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseMilestoneAction_ParsesEveryKnownGitHubMilestoneAction(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.MilestoneAction
	}{
		{"milestoned", valueobjects.MilestoneActionMilestoned},
		{"demilestoned", valueobjects.MilestoneActionDemilestoned},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseMilestoneAction(c.raw)
		if err != nil {
			t.Fatalf("ParseMilestoneAction(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseMilestoneAction(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseMilestoneAction_RejectsAnUnrecognizedAction(t *testing.T) {
	_, err := valueobjects.ParseMilestoneAction("remilestoned")

	if err == nil {
		t.Fatal("expected an error for an unrecognized milestone action, got nil")
	}
}

func TestMilestoneAction_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.MilestoneAction(99)

	got := unrecognized.String()

	if got != "MilestoneAction(99)" {
		t.Fatalf("String() = %q, want %q", got, "MilestoneAction(99)")
	}
}

func TestMilestoneAction_String_RoundTripsThroughParseMilestoneAction(t *testing.T) {
	actions := []valueobjects.MilestoneAction{
		valueobjects.MilestoneActionMilestoned,
		valueobjects.MilestoneActionDemilestoned,
	}

	for _, want := range actions {
		got, err := valueobjects.ParseMilestoneAction(want.String())
		if err != nil {
			t.Fatalf("ParseMilestoneAction(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the action: got %v, want %v", got, want)
		}
	}
}
