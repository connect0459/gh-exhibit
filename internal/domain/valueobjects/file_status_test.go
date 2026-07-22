package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseFileStatus_ParsesEveryKnownGitHubFileStatus(t *testing.T) {
	cases := []struct {
		raw  string
		want valueobjects.FileStatus
	}{
		{"added", valueobjects.FileStatusAdded},
		{"removed", valueobjects.FileStatusRemoved},
		{"modified", valueobjects.FileStatusModified},
		{"renamed", valueobjects.FileStatusRenamed},
		{"copied", valueobjects.FileStatusCopied},
		{"changed", valueobjects.FileStatusChanged},
		{"unchanged", valueobjects.FileStatusUnchanged},
	}

	for _, c := range cases {
		got, err := valueobjects.ParseFileStatus(c.raw)
		if err != nil {
			t.Fatalf("ParseFileStatus(%q): unexpected error: %v", c.raw, err)
		}
		if got != c.want {
			t.Fatalf("ParseFileStatus(%q) = %v, want %v", c.raw, got, c.want)
		}
	}
}

func TestParseFileStatus_RejectsAnUnrecognizedStatus(t *testing.T) {
	_, err := valueobjects.ParseFileStatus("deleted")

	if err == nil {
		t.Fatal("expected an error for an unrecognized file status, got nil")
	}
}

func TestFileStatus_String_FallsBackForAnUnrecognizedNumericValue(t *testing.T) {
	unrecognized := valueobjects.FileStatus(99)

	got := unrecognized.String()

	if got != "FileStatus(99)" {
		t.Fatalf("String() = %q, want %q", got, "FileStatus(99)")
	}
}

func TestFileStatus_String_RoundTripsThroughParseFileStatus(t *testing.T) {
	statuses := []valueobjects.FileStatus{
		valueobjects.FileStatusAdded,
		valueobjects.FileStatusRemoved,
		valueobjects.FileStatusModified,
		valueobjects.FileStatusRenamed,
		valueobjects.FileStatusCopied,
		valueobjects.FileStatusChanged,
		valueobjects.FileStatusUnchanged,
	}

	for _, want := range statuses {
		got, err := valueobjects.ParseFileStatus(want.String())
		if err != nil {
			t.Fatalf("ParseFileStatus(%q): unexpected error: %v", want.String(), err)
		}
		if got != want {
			t.Fatalf("round trip through String() changed the status: got %v, want %v", got, want)
		}
	}
}
