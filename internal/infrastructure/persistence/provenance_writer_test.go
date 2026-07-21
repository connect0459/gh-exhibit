package persistence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func testProvenance(t *testing.T) valueobjects.Provenance {
	t.Helper()
	p, err := valueobjects.NewProvenance("connect0459/gh-exhibit", "v0.1.0", "abc123")
	if err != nil {
		t.Fatalf("NewProvenance() error = %v", err)
	}
	return p
}

func TestWriteProvenance_WritesToolVersionAndCommitAsJSONUnderEvidence(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewProvenanceWriter(baseDir)

	err := writer.WriteProvenance(context.Background(), testIssueRef(t), testProvenance(t))
	if err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}

	want := `{"tool":"connect0459/gh-exhibit","version":"v0.1.0","commit":"abc123"}`
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "provenance.json"))
	if got != want {
		t.Fatalf("WriteProvenance() wrote %q, want %q", got, want)
	}
}

func TestWriteProvenance_OmitsOwnerFromThePath(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewProvenanceWriter(baseDir)
	otherOwnerRef, err := valueobjects.NewIssueRef("some-other-owner", "hello-world", 42)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}

	if err := writer.WriteProvenance(context.Background(), otherOwnerRef, testProvenance(t)); err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "provenance.json")); err != nil {
		t.Fatalf("expected file at hello-world/42/evidence/provenance.json regardless of owner, stat error = %v", err)
	}
}

func TestWriteProvenance_OverwritesAnExistingFileForTheSameRef(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewProvenanceWriter(baseDir)
	ref := testIssueRef(t)
	older, err := valueobjects.NewProvenance("connect0459/gh-exhibit", "v0.0.9", "old456")
	if err != nil {
		t.Fatalf("NewProvenance() error = %v", err)
	}

	if err := writer.WriteProvenance(context.Background(), ref, older); err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}
	if err := writer.WriteProvenance(context.Background(), ref, testProvenance(t)); err != nil {
		t.Fatalf("WriteProvenance() error = %v", err)
	}

	want := `{"tool":"connect0459/gh-exhibit","version":"v0.1.0","commit":"abc123"}`
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "provenance.json"))
	if got != want {
		t.Fatalf("WriteProvenance() wrote %q, want %q", got, want)
	}
}

func TestWriteProvenance_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewProvenanceWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteProvenance(ctx, testIssueRef(t), testProvenance(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteProvenance() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "provenance.json")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteProvenance() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}
