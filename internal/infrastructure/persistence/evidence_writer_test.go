package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func testIssueRef(t *testing.T) valueobjects.IssueRef {
	t.Helper()

	ref, err := valueobjects.NewIssueRef("octocat", "hello-world", 42)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}
	return ref
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return string(b)
}

func TestWriteIssue_WritesResponseBodyVerbatimUnderRepoNumber(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	const body = `{
  "number": 42,
  "title": "Some issue"
}`

	err := writer.WriteIssue(context.Background(), testIssueRef(t), json.RawMessage(body))
	if err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.json"))
	if got != body {
		t.Fatalf("WriteIssue() wrote %q, want %q", got, body)
	}
}

func TestWriteIssue_OmitsOwnerFromThePath(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	otherOwnerRef, err := valueobjects.NewIssueRef("some-other-owner", "hello-world", 42)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}

	if err := writer.WriteIssue(context.Background(), otherOwnerRef, json.RawMessage(`{}`)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "hello-world", "42.json")); err != nil {
		t.Fatalf("expected file at hello-world/42.json regardless of owner, stat error = %v", err)
	}
}

func TestWritePullRequest_WritesResponseBodyVerbatimWithPullSuffix(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	const body = `{"number":42,"title":"Some PR"}`

	err := writer.WritePullRequest(context.Background(), testIssueRef(t), json.RawMessage(body))
	if err != nil {
		t.Fatalf("WritePullRequest() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.pull.json"))
	if got != body {
		t.Fatalf("WritePullRequest() wrote %q, want %q", got, body)
	}
}

func TestWriteTimeline_ConcatenatesPagesIntoOneArrayFilePreservingEachItemVerbatim(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	pages := []json.RawMessage{
		json.RawMessage("{\n  \"id\": 1\n}"),
		json.RawMessage(`{"id":2}`),
	}

	err := writer.WriteTimeline(context.Background(), testIssueRef(t), pages)
	if err != nil {
		t.Fatalf("WriteTimeline() error = %v", err)
	}

	want := "[{\n  \"id\": 1\n},{\"id\":2}]"
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.timeline.json"))
	if got != want {
		t.Fatalf("WriteTimeline() wrote %q, want %q", got, want)
	}
}

func TestWriteTimeline_WritesAnEmptyArrayWhenGivenNoPages(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)

	err := writer.WriteTimeline(context.Background(), testIssueRef(t), nil)
	if err != nil {
		t.Fatalf("WriteTimeline() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.timeline.json"))
	if got != "[]" {
		t.Fatalf("WriteTimeline() wrote %q, want \"[]\"", got)
	}
}

func TestWriteReviewComments_ConcatenatesItemsIntoOneArrayFileWithReviewCommentsSuffix(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	items := []json.RawMessage{
		json.RawMessage(`{"id":10}`),
		json.RawMessage(`{"id":20}`),
	}

	err := writer.WriteReviewComments(context.Background(), testIssueRef(t), items)
	if err != nil {
		t.Fatalf("WriteReviewComments() error = %v", err)
	}

	want := `[{"id":10},{"id":20}]`
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.review-comments.json"))
	if got != want {
		t.Fatalf("WriteReviewComments() wrote %q, want %q", got, want)
	}
}

func TestWriteTimeline_ReturnsAnErrorInsteadOfWritingAMalformedArrayForAnEmptyElement(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	pages := []json.RawMessage{json.RawMessage(`{"id":1}`), json.RawMessage(nil)}

	err := writer.WriteTimeline(context.Background(), testIssueRef(t), pages)
	if err == nil {
		t.Fatal("WriteTimeline() error = nil, want an error for the empty element")
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42.timeline.json")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteTimeline() wrote a file despite the invalid element, stat error = %v", statErr)
	}
}

func TestWriteReviewComments_ReturnsAnErrorInsteadOfWritingAMalformedArrayForAnEmptyElement(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	items := []json.RawMessage{json.RawMessage(``), json.RawMessage(`{"id":20}`)}

	err := writer.WriteReviewComments(context.Background(), testIssueRef(t), items)
	if err == nil {
		t.Fatal("WriteReviewComments() error = nil, want an error for the empty element")
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42.review-comments.json")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteReviewComments() wrote a file despite the invalid element, stat error = %v", statErr)
	}
}

func TestWriteIssue_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteIssue(ctx, testIssueRef(t), json.RawMessage(`{}`))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteIssue() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42.json")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteIssue() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}

func TestWriteTimeline_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteTimeline(ctx, testIssueRef(t), []json.RawMessage{json.RawMessage(`{"id":1}`)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteTimeline() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42.timeline.json")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteTimeline() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}

func TestWritePullRequest_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WritePullRequest(ctx, testIssueRef(t), json.RawMessage(`{}`))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WritePullRequest() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42.pull.json")); !os.IsNotExist(statErr) {
		t.Fatalf("WritePullRequest() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}

func TestWriteReviewComments_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := writer.WriteReviewComments(ctx, testIssueRef(t), []json.RawMessage{json.RawMessage(`{"id":1}`)})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteReviewComments() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42.review-comments.json")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteReviewComments() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}

func TestWriteIssue_ReturnsWrappedErrorWhenDirectoryCannotBeCreated(t *testing.T) {
	baseDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(baseDir, "hello-world"), []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
	writer := NewEvidenceWriter(baseDir)

	err := writer.WriteIssue(context.Background(), testIssueRef(t), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("WriteIssue() error = nil, want a directory-creation error")
	}
}

func TestWriteIssue_ReturnsWrappedErrorWhenFileCannotBeWritten(t *testing.T) {
	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, "hello-world")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	writer := NewEvidenceWriter(baseDir)

	err := writer.WriteIssue(context.Background(), testIssueRef(t), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("WriteIssue() error = nil, want a file-write error")
	}
}

func TestWriteIssue_OverwritesAnExistingFileForTheSameRef(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ref := testIssueRef(t)

	if err := writer.WriteIssue(context.Background(), ref, json.RawMessage(`{"title":"first"}`)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}
	if err := writer.WriteIssue(context.Background(), ref, json.RawMessage(`{"title":"second"}`)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42.json"))
	if got != `{"title":"second"}` {
		t.Fatalf("WriteIssue() wrote %q, want %q", got, `{"title":"second"}`)
	}
}
