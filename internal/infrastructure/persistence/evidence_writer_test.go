package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"io"
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

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "issue.json"))
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

	if _, err := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "issue.json")); err != nil {
		t.Fatalf("expected file at hello-world/42/evidence/issue.json regardless of owner, stat error = %v", err)
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

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "pull.json"))
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
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "timeline.json"))
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

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "timeline.json"))
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
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "review-comments.json"))
	if got != want {
		t.Fatalf("WriteReviewComments() wrote %q, want %q", got, want)
	}
}

func TestWritePullRequestFiles_ConcatenatesItemsIntoOneArrayFileWithPullFilesSuffix(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	items := []json.RawMessage{
		json.RawMessage(`{"filename":"a.go"}`),
		json.RawMessage(`{"filename":"b.go"}`),
	}

	err := writer.WritePullRequestFiles(context.Background(), testIssueRef(t), items)
	if err != nil {
		t.Fatalf("WritePullRequestFiles() error = %v", err)
	}

	want := `[{"filename":"a.go"},{"filename":"b.go"}]`
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "pull-files.json"))
	if got != want {
		t.Fatalf("WritePullRequestFiles() wrote %q, want %q", got, want)
	}
}

func TestWritePullRequestCommits_ConcatenatesItemsIntoOneArrayFileWithPullCommitsSuffix(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	items := []json.RawMessage{
		json.RawMessage(`{"sha":"aaa"}`),
		json.RawMessage(`{"sha":"bbb"}`),
	}

	err := writer.WritePullRequestCommits(context.Background(), testIssueRef(t), items)
	if err != nil {
		t.Fatalf("WritePullRequestCommits() error = %v", err)
	}

	want := `[{"sha":"aaa"},{"sha":"bbb"}]`
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "pull-commits.json"))
	if got != want {
		t.Fatalf("WritePullRequestCommits() wrote %q, want %q", got, want)
	}
}

func TestWriteSubIssues_ConcatenatesItemsIntoOneArrayFileWithSubIssuesSuffix(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	items := []json.RawMessage{
		json.RawMessage(`{"number":65}`),
		json.RawMessage(`{"number":66}`),
	}

	err := writer.WriteSubIssues(context.Background(), testIssueRef(t), items)
	if err != nil {
		t.Fatalf("WriteSubIssues() error = %v", err)
	}

	want := `[{"number":65},{"number":66}]`
	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "sub-issues.json"))
	if got != want {
		t.Fatalf("WriteSubIssues() wrote %q, want %q", got, want)
	}
}

func TestWriteSubIssues_WritesAnEmptyArrayWhenGivenNoItems(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)

	err := writer.WriteSubIssues(context.Background(), testIssueRef(t), nil)
	if err != nil {
		t.Fatalf("WriteSubIssues() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "sub-issues.json"))
	if got != "[]" {
		t.Fatalf("WriteSubIssues() wrote %q, want \"[]\"", got)
	}
}

func TestWriteParentIssue_WritesResponseBodyVerbatimWithParentIssueSuffix(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	const body = `{"number":64,"title":"Parent issue"}`

	err := writer.WriteParentIssue(context.Background(), testIssueRef(t), json.RawMessage(body))
	if err != nil {
		t.Fatalf("WriteParentIssue() error = %v", err)
	}

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "parent-issue.json"))
	if got != body {
		t.Fatalf("WriteParentIssue() wrote %q, want %q", got, body)
	}
}

func TestWriteParentIssue_RemovesAnExistingFileWhenGivenNoRawData(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ref := testIssueRef(t)
	path := filepath.Join(baseDir, "hello-world", "42", "evidence", "parent-issue.json")

	if err := writer.WriteParentIssue(context.Background(), ref, json.RawMessage(`{"number":64}`)); err != nil {
		t.Fatalf("WriteParentIssue() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected parent-issue.json to exist after the first write, stat error = %v", err)
	}

	// A rerun where the issue no longer has a parent must remove the stale
	// file left by an earlier run, so the exported directory stays a
	// self-healing view of the issue's current state rather than keeping a
	// parent reference that no longer exists.
	if err := writer.WriteParentIssue(context.Background(), ref, nil); err != nil {
		t.Fatalf("WriteParentIssue() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected parent-issue.json to be removed once the parent is gone, stat error = %v", err)
	}
}

func TestWriteParentIssue_IsANoOpWhenGivenNoRawDataAndNoFileExists(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)

	err := writer.WriteParentIssue(context.Background(), testIssueRef(t), nil)
	if err != nil {
		t.Fatalf("WriteParentIssue() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "parent-issue.json")); !os.IsNotExist(err) {
		t.Fatalf("expected no parent-issue.json to be created, stat error = %v", err)
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
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "timeline.json")); !os.IsNotExist(statErr) {
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
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "review-comments.json")); !os.IsNotExist(statErr) {
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
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "issue.json")); !os.IsNotExist(statErr) {
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
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "timeline.json")); !os.IsNotExist(statErr) {
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
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "pull.json")); !os.IsNotExist(statErr) {
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
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "evidence", "review-comments.json")); !os.IsNotExist(statErr) {
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

func TestWriteIssue_LeavesAnAlreadyOpenReaderAbleToReadTheCompleteOldContentDuringARewrite(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ref := testIssueRef(t)
	const oldBody = `{"title":"first"}`

	if err := writer.WriteIssue(context.Background(), ref, json.RawMessage(oldBody)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}

	// A reader that opened the file before the rewrite must keep seeing the
	// old file's inode: a rename swaps the directory entry to a new inode
	// rather than truncating and overwriting the one this handle refers to.
	oldHandle, err := os.Open(filepath.Join(baseDir, "hello-world", "42", "evidence", "issue.json"))
	if err != nil {
		t.Fatalf("os.Open() error = %v", err)
	}
	defer func() { _ = oldHandle.Close() }()

	if err := writer.WriteIssue(context.Background(), ref, json.RawMessage(`{"title":"second"}`)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}

	got, err := io.ReadAll(oldHandle)
	if err != nil {
		t.Fatalf("io.ReadAll() error = %v", err)
	}
	if string(got) != oldBody {
		t.Fatalf("a reader open since before the rewrite saw %q, want the untouched old content %q", got, oldBody)
	}
}

func TestWriteIssue_LeavesNoTemporaryFilesBehindAfterASuccessfulWrite(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)

	if err := writer.WriteIssue(context.Background(), testIssueRef(t), json.RawMessage(`{}`)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}

	entries, err := os.ReadDir(filepath.Join(baseDir, "hello-world", "42", "evidence"))
	if err != nil {
		t.Fatalf("os.ReadDir() error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "issue.json" {
		names := make([]string, len(entries))
		for i, e := range entries {
			names[i] = e.Name()
		}
		t.Fatalf("directory contains %v after write, want only issue.json (no leaked temporary file)", names)
	}
}

func TestWriteIssue_LeavesNoTemporaryFileBehindWhenTheWriteFails(t *testing.T) {
	baseDir := t.TempDir()
	writer := NewEvidenceWriter(baseDir)
	ref := testIssueRef(t)

	if err := writer.WriteIssue(context.Background(), ref, json.RawMessage(`{"title":"first"}`)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}
	dir := filepath.Join(baseDir, "hello-world", "42", "evidence")
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatalf("os.Chmod() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	if err := writer.WriteIssue(context.Background(), ref, json.RawMessage(`{"title":"second"}`)); err == nil {
		t.Fatal("WriteIssue() error = nil, want an error when the directory forbids creating the temporary file")
	}

	got := readFile(t, filepath.Join(dir, "issue.json"))
	if got != `{"title":"first"}` {
		t.Fatalf("WriteIssue() left %q after a failed rewrite, want the untouched original %q", got, `{"title":"first"}`)
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

	got := readFile(t, filepath.Join(baseDir, "hello-world", "42", "evidence", "issue.json"))
	if got != `{"title":"second"}` {
		t.Fatalf("WriteIssue() wrote %q, want %q", got, `{"title":"second"}`)
	}
}
