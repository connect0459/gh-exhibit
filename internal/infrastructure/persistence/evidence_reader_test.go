package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

const readerIssueJSON = `{"title":"Something is broken","user":{"login":"octocat"}}`

func writeIssueEvidence(t *testing.T, baseDir string) {
	t.Helper()

	if err := NewEvidenceWriter(baseDir).WriteIssue(context.Background(), testIssueRef(t), json.RawMessage(readerIssueJSON)); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}
}

func TestEvidenceReader_ReadEvidence_ReturnsTheIssueResourceVerbatim(t *testing.T) {
	baseDir := t.TempDir()
	writeIssueEvidence(t, baseDir)

	got, err := NewEvidenceReader(baseDir).ReadEvidence(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("ReadEvidence() error = %v", err)
	}

	if string(got.Issue) != readerIssueJSON {
		t.Fatalf("Issue = %q, want %q", got.Issue, readerIssueJSON)
	}
}

func TestEvidenceReader_ReadEvidence_SplitsAConcatenatedArrayIntoItsElementsVerbatim(t *testing.T) {
	baseDir := t.TempDir()
	writeIssueEvidence(t, baseDir)
	items := []json.RawMessage{json.RawMessage(`{"id": 1}`), json.RawMessage(`{ "id":2 }`)}
	if err := NewEvidenceWriter(baseDir).WriteTimeline(context.Background(), testIssueRef(t), items); err != nil {
		t.Fatalf("WriteTimeline() error = %v", err)
	}

	got, err := NewEvidenceReader(baseDir).ReadEvidence(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("ReadEvidence() error = %v", err)
	}

	if len(got.Timeline) != 2 || string(got.Timeline[0]) != `{"id": 1}` || string(got.Timeline[1]) != `{ "id":2 }` {
		t.Fatalf("Timeline = %q, want the two elements written, whitespace preserved", got.Timeline)
	}
}

func TestEvidenceReader_ReadEvidence_ReadsEveryPullRequestFileThatWasWritten(t *testing.T) {
	baseDir := t.TempDir()
	writeIssueEvidence(t, baseDir)
	writer := NewEvidenceWriter(baseDir)
	ctx := context.Background()
	ref := testIssueRef(t)
	one := []json.RawMessage{json.RawMessage(`{"n":1}`)}
	if err := writer.WritePullRequest(ctx, ref, json.RawMessage(`{"merged_at":null}`)); err != nil {
		t.Fatalf("WritePullRequest() error = %v", err)
	}
	if err := writer.WriteReviewComments(ctx, ref, one); err != nil {
		t.Fatalf("WriteReviewComments() error = %v", err)
	}
	if err := writer.WritePullRequestFiles(ctx, ref, one); err != nil {
		t.Fatalf("WritePullRequestFiles() error = %v", err)
	}
	if err := writer.WritePullRequestCommits(ctx, ref, one); err != nil {
		t.Fatalf("WritePullRequestCommits() error = %v", err)
	}
	if err := writer.WriteCheckRuns(ctx, ref, one); err != nil {
		t.Fatalf("WriteCheckRuns() error = %v", err)
	}

	got, err := NewEvidenceReader(baseDir).ReadEvidence(ctx, ref)
	if err != nil {
		t.Fatalf("ReadEvidence() error = %v", err)
	}

	if string(got.PullRequest) != `{"merged_at":null}` {
		t.Fatalf("PullRequest = %q, want the pull resource verbatim", got.PullRequest)
	}
	for name, items := range map[string][]json.RawMessage{
		"ReviewComments":     got.ReviewComments,
		"PullRequestFiles":   got.PullRequestFiles,
		"PullRequestCommits": got.PullRequestCommits,
		"CheckRuns":          got.CheckRuns,
	} {
		if len(items) != 1 {
			t.Fatalf("%s has %d elements, want 1", name, len(items))
		}
	}
}

func TestEvidenceReader_ReadEvidence_LeavesAnAbsentOptionalFileEmpty(t *testing.T) {
	baseDir := t.TempDir()
	writeIssueEvidence(t, baseDir)

	got, err := NewEvidenceReader(baseDir).ReadEvidence(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("ReadEvidence() error = %v, want nil (only issue.json is required)", err)
	}

	if len(got.Timeline) != 0 || len(got.PullRequest) != 0 || len(got.ParentIssue) != 0 || len(got.SubIssues) != 0 {
		t.Fatalf("absent files should read as empty, got %+v", got)
	}
}

func TestEvidenceReader_ReadEvidence_ReturnsAnErrorWhenTheIssueResourceIsMissing(t *testing.T) {
	_, err := NewEvidenceReader(t.TempDir()).ReadEvidence(context.Background(), testIssueRef(t))

	if err == nil {
		t.Fatal("expected an error when issue.json does not exist, got nil")
	}
}

func TestEvidenceReader_ReadEvidence_ReturnsAnErrorWhenAnArrayFileIsNotAJSONArray(t *testing.T) {
	baseDir := t.TempDir()
	writeIssueEvidence(t, baseDir)
	path := filepath.Join(baseDir, "hello-world", "42", "evidence", "timeline.json")
	if err := os.WriteFile(path, []byte(`{"not":"an array"}`), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	_, err := NewEvidenceReader(baseDir).ReadEvidence(context.Background(), testIssueRef(t))

	if err == nil {
		t.Fatal("expected an error for a timeline.json that is not a JSON array, got nil")
	}
}

func TestEvidenceReader_ReadEvidence_ReturnsContextErrorWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	writeIssueEvidence(t, baseDir)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := NewEvidenceReader(baseDir).ReadEvidence(ctx, testIssueRef(t))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadEvidence() error = %v, want context.Canceled", err)
	}
}
