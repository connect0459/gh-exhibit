package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseIssueKind_ParsesEveryKnownFilterKind(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.IssueKind
	}{
		{"issue", valueobjects.IssueKindIssue},
		{"pr", valueobjects.IssueKindPullRequest},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseIssueKind(c.raw)
		if err != nil {
			t.Fatalf("ParseIssueKind(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseIssueKind(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseIssueKind_RejectsAnUnrecognizedKind(t *testing.T) {
	_, err := valueobjects.ParseIssueKind("draft")

	if err == nil {
		t.Fatal("expected an error for an unrecognized issue kind, got nil")
	}
}

func TestIssueKind_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.IssueKind(99)

	got := unrecognized.String()

	if got != "IssueKind(99)" {
		t.Fatalf("String() = %q, want %q", got, "IssueKind(99)")
	}
}

func TestIssueKind_String_RoundTripsThroughParseIssueKind(t *testing.T) {
	kinds := []valueobjects.IssueKind{
		valueobjects.IssueKindIssue,
		valueobjects.IssueKindPullRequest,
	}

	for _, want := range kinds {
		got, err := valueobjects.ParseIssueKind(want.String())
		if err != nil {
			t.Fatalf("ParseIssueKind(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the kind: got %v, want %v", got, want)
		}
	}
}
