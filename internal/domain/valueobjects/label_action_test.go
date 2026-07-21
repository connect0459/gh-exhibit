package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseLabelAction_ParsesEveryKnownGitHubLabelAction(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.LabelAction
	}{
		{"labeled", valueobjects.LabelActionLabeled},
		{"unlabeled", valueobjects.LabelActionUnlabeled},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseLabelAction(c.raw)
		if err != nil {
			t.Fatalf("ParseLabelAction(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseLabelAction(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseLabelAction_RejectsAnUnrecognizedAction(t *testing.T) {
	_, err := valueobjects.ParseLabelAction("relabeled")

	if err == nil {
		t.Fatal("expected an error for an unrecognized label action, got nil")
	}
}

func TestLabelAction_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.LabelAction(99)

	got := unrecognized.String()

	if got != "LabelAction(99)" {
		t.Fatalf("String() = %q, want %q", got, "LabelAction(99)")
	}
}

func TestLabelAction_String_RoundTripsThroughParseLabelAction(t *testing.T) {
	actions := []valueobjects.LabelAction{
		valueobjects.LabelActionLabeled,
		valueobjects.LabelActionUnlabeled,
	}

	for _, want := range actions {
		got, err := valueobjects.ParseLabelAction(want.String())
		if err != nil {
			t.Fatalf("ParseLabelAction(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the action: got %v, want %v", got, want)
		}
	}
}
