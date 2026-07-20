package persistence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestWriteDocument_WritesRenderedBytesVerbatimUnderRepoNumberWithMdSuffix(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewDocumentWriter(baseDir)
	const rendered = "# Some issue\n\nmeta:{\"author\":\"octocat\"}\n\nBody.\n"

	err := writer.WriteDocument(context.Background(), testIssueRef(t), []byte(rendered))
	if err != nil {
		t.Fatalf("WriteDocument() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.md"))
	if got != rendered {
		t.Fatalf("WriteDocument() wrote %q, want %q", got, rendered)
	}
}

func TestWriteDocument_OmitsOwnerFromThePath(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewDocumentWriter(baseDir)
	otherOwnerRef, err := valueobjects.NewIssueRef("some-other-owner", "hello-world", 42)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}

	if err := writer.WriteDocument(context.Background(), otherOwnerRef, []byte("# x\n")); err != nil {
		t.Fatalf("WriteDocument() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "hello-world", "42.md")); err != nil {
		t.Fatalf("expected file at hello-world/42.md regardless of owner, stat error = %v", err)
	}
}

func TestWriteDocument_OverwritesAnExistingFileForTheSameRef(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewDocumentWriter(baseDir)
	ref := testIssueRef(t)

	if err := writer.WriteDocument(context.Background(), ref, []byte("# first\n")); err != nil {
		t.Fatalf("WriteDocument() error = %v", err)
	}
	if err := writer.WriteDocument(context.Background(), ref, []byte("# second\n")); err != nil {
		t.Fatalf("WriteDocument() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.md"))
	if got != "# second\n" {
		t.Fatalf("WriteDocument() wrote %q, want %q", got, "# second\n")
	}
}

func TestWriteDocument_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewDocumentWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteDocument(ctx, testIssueRef(t), []byte("# x\n"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteDocument() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42.md")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteDocument() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}

func TestWriteDocument_ReturnsWrappedErrorWhenDirectoryCannotBeCreated(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(baseDir, "hello-world"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	writer := NewDocumentWriter(baseDir)

	err := writer.WriteDocument(context.Background(), testIssueRef(t), []byte("# x\n"))
	if err == nil {
		t.Fatal("WriteDocument() error = nil, want a directory-creation error")
	}
}
