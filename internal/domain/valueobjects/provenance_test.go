package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func newTestProvenance(t *testing.T) valueobjects.Provenance {
	t.Helper()

	p, err := valueobjects.NewProvenance("connect0459/gh-exhibit", "v0.1.0", "abc123")
	if err != nil {
		t.Fatalf("NewProvenance() error = %v", err)
	}
	return p
}

func TestNewProvenance_RejectsAnEmptyTool(t *testing.T) {
	if _, err := valueobjects.NewProvenance("", "v0.1.0", "abc123"); err == nil {
		t.Fatal("NewProvenance() error = nil, want an error for an empty tool")
	}
}

func TestNewProvenance_RejectsAnEmptyVersion(t *testing.T) {
	if _, err := valueobjects.NewProvenance("connect0459/gh-exhibit", "", "abc123"); err == nil {
		t.Fatal("NewProvenance() error = nil, want an error for an empty version")
	}
}

func TestNewProvenance_RejectsAnEmptyCommit(t *testing.T) {
	if _, err := valueobjects.NewProvenance("connect0459/gh-exhibit", "v0.1.0", ""); err == nil {
		t.Fatal("NewProvenance() error = nil, want an error for an empty commit")
	}
}

func TestProvenance_ExposesTheToolVersionAndCommitItWasConstructedWith(t *testing.T) {
	p := newTestProvenance(t)

	if p.Tool() != "connect0459/gh-exhibit" {
		t.Fatalf("Tool() = %q, want %q", p.Tool(), "connect0459/gh-exhibit")
	}
	if p.Version() != "v0.1.0" {
		t.Fatalf("Version() = %q, want %q", p.Version(), "v0.1.0")
	}
	if p.Commit() != "abc123" {
		t.Fatalf("Commit() = %q, want %q", p.Commit(), "abc123")
	}
}

func TestProvenance_Equals_TreatsMatchingValuesAsEqual(t *testing.T) {
	a := newTestProvenance(t)
	b := newTestProvenance(t)

	if !a.Equals(b) {
		t.Fatal("expected provenances constructed from the same tool, version, and commit to be equal")
	}
}

func TestProvenance_Equals_TreatsDifferentCommitsAsNotEqual(t *testing.T) {
	a := newTestProvenance(t)
	b, err := valueobjects.NewProvenance("connect0459/gh-exhibit", "v0.1.0", "def456")
	if err != nil {
		t.Fatalf("NewProvenance() error = %v", err)
	}

	if a.Equals(b) {
		t.Fatal("expected provenances with different commits to not be equal")
	}
}
