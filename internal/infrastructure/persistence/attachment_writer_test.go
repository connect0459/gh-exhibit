package persistence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAsset_WritesDataVerbatimUnderIssuesRepoNumberAssets(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)
	const data = "binary-ish content"

	err := writer.WriteAsset(context.Background(), testIssueRef(t), "abc-123.png", []byte(data))
	if err != nil {
		t.Fatalf("WriteAsset() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "issues", "hello-world", "42", "assets", "abc-123.png"))
	if got != data {
		t.Fatalf("WriteAsset() wrote %q, want %q", got, data)
	}
}

func TestWriteAsset_OmitsOwnerFromThePath(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)

	err := writer.WriteAsset(context.Background(), testIssueRef(t), "abc-123.png", []byte("data"))
	if err != nil {
		t.Fatalf("WriteAsset() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "issues", "hello-world", "42", "assets", "abc-123.png")); err != nil {
		t.Fatalf("expected file at issues/hello-world/42/assets/abc-123.png, stat error = %v", err)
	}
}

func TestWriteAsset_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteAsset(ctx, testIssueRef(t), "abc-123.png", []byte("data"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteAsset() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "issues", "hello-world", "42", "assets", "abc-123.png")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteAsset() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}

func TestWriteFetchErrorLog_WritesLogVerbatimUnderIssuesRepoNumber(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)
	const log = "https://github.com/user-attachments/assets/abc: 404 not found\n"

	err := writer.WriteFetchErrorLog(context.Background(), testIssueRef(t), []byte(log))
	if err != nil {
		t.Fatalf("WriteFetchErrorLog() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "issues", "hello-world", "42", "fetch-errors.log"))
	if got != log {
		t.Fatalf("WriteFetchErrorLog() wrote %q, want %q", got, log)
	}
}

func TestWriteFetchErrorLog_RemovesAnExistingLogWhenGivenAnEmptyLog(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)
	ref := testIssueRef(t)
	if err := writer.WriteFetchErrorLog(context.Background(), ref, []byte("stale failure\n")); err != nil {
		t.Fatalf("WriteFetchErrorLog() error = %v", err)
	}

	if err := writer.WriteFetchErrorLog(context.Background(), ref, nil); err != nil {
		t.Fatalf("WriteFetchErrorLog() error = %v, want a rerun with no failures to clear the stale log without error", err)
	}

	if _, statErr := os.Stat(filepath.Join(baseDir, "issues", "hello-world", "42", "fetch-errors.log")); !os.IsNotExist(statErr) {
		t.Fatalf("fetch-errors.log still exists after WriteFetchErrorLog was given an empty log, stat error = %v", statErr)
	}
}

func TestWriteFetchErrorLog_ReturnsWrappedErrorWhenTheExistingLogCannotBeRemoved(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)
	ref := testIssueRef(t)
	if err := writer.WriteFetchErrorLog(context.Background(), ref, []byte("stale\n")); err != nil {
		t.Fatalf("WriteFetchErrorLog() error = %v", err)
	}

	dir := issueDir(baseDir, ref)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := writer.WriteFetchErrorLog(context.Background(), ref, nil); err == nil {
		t.Fatal("WriteFetchErrorLog() error = nil, want an error when the existing log cannot be removed")
	}
}

func TestWriteFetchErrorLog_DoesNothingWhenGivenAnEmptyLogAndNoExistingLogExists(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)

	if err := writer.WriteFetchErrorLog(context.Background(), testIssueRef(t), nil); err != nil {
		t.Fatalf("WriteFetchErrorLog() error = %v, want no error when there was never a log to clear", err)
	}

	if _, statErr := os.Stat(filepath.Join(baseDir, "issues", "hello-world", "42", "fetch-errors.log")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteFetchErrorLog() created a file for an empty log, stat error = %v", statErr)
	}
}

func TestWriteFetchErrorLog_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewAttachmentWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteFetchErrorLog(ctx, testIssueRef(t), []byte("log"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteFetchErrorLog() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "issues", "hello-world", "42", "fetch-errors.log")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteFetchErrorLog() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}
