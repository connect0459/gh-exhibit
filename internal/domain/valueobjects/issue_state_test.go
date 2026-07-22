package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseIssueState_ParsesEveryKnownGitHubIssueState(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.IssueState
	}{
		{"open", valueobjects.IssueStateOpen},
		{"closed", valueobjects.IssueStateClosed},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseIssueState(c.raw)
		if err != nil {
			t.Fatalf("ParseIssueState(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseIssueState(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseIssueState_RejectsAnUnrecognizedState(t *testing.T) {
	_, err := valueobjects.ParseIssueState("archived")

	if err == nil {
		t.Fatal("expected an error for an unrecognized issue state, got nil")
	}
}

func TestIssueState_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.IssueState(99)

	got := unrecognized.String()

	if got != "IssueState(99)" {
		t.Fatalf("String() = %q, want %q", got, "IssueState(99)")
	}
}

func TestIssueState_String_RoundTripsThroughParseIssueState(t *testing.T) {
	states := []valueobjects.IssueState{
		valueobjects.IssueStateOpen,
		valueobjects.IssueStateClosed,
	}

	for _, want := range states {
		got, err := valueobjects.ParseIssueState(want.String())
		if err != nil {
			t.Fatalf("ParseIssueState(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the state: got %v, want %v", got, want)
		}
	}
}
