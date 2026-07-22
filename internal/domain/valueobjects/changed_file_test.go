package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func mustNewChangedFile(t *testing.T, filename, previousFilename string, status valueobjects.FileStatus, additions, deletions int, patch string) valueobjects.ChangedFile {
	t.Helper()
	file, err := valueobjects.NewChangedFile(filename, previousFilename, status, additions, deletions, patch)
	if err != nil {
		t.Fatalf("NewChangedFile(): unexpected error: %v", err)
	}
	return file
}

func TestNewChangedFile_RejectsAnEmptyFilename(t *testing.T) {
	_, err := valueobjects.NewChangedFile("", "", valueobjects.FileStatusAdded, 1, 0, "")

	if err == nil {
		t.Fatal("expected an error for an empty filename, got nil")
	}
}

func TestChangedFile_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	file := mustNewChangedFile(t, "internal/foo.go", "internal/old.go", valueobjects.FileStatusRenamed, 12, 3, "@@ -1,3 +1,3 @@")

	if file.Filename() != "internal/foo.go" {
		t.Fatalf("Filename() = %q, want %q", file.Filename(), "internal/foo.go")
	}
	if file.PreviousFilename() != "internal/old.go" {
		t.Fatalf("PreviousFilename() = %q, want %q", file.PreviousFilename(), "internal/old.go")
	}
	if file.Status() != valueobjects.FileStatusRenamed {
		t.Fatalf("Status() = %v, want %v", file.Status(), valueobjects.FileStatusRenamed)
	}
	if file.Additions() != 12 {
		t.Fatalf("Additions() = %d, want %d", file.Additions(), 12)
	}
	if file.Deletions() != 3 {
		t.Fatalf("Deletions() = %d, want %d", file.Deletions(), 3)
	}
	if file.Patch() != "@@ -1,3 +1,3 @@" {
		t.Fatalf("Patch() = %q, want %q", file.Patch(), "@@ -1,3 +1,3 @@")
	}
}

func TestChangedFile_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	a := mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch")
	b := mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch")

	if !a.Equals(b) {
		t.Fatal("expected changed files with matching fields to be equal")
	}
}

func TestChangedFile_Equals_TreatsDifferentPatchAsNotEqual(t *testing.T) {
	a := mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch")
	b := mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "")

	if a.Equals(b) {
		t.Fatal("expected changed files with different patch content to not be equal")
	}
}
