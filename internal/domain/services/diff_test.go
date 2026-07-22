package services_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func diffAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	attribution, err := valueobjects.NewAttribution("octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/pull/1")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}
	return attribution
}

func changedFileRaw(filename, status, patch string, additions, deletions int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"filename": %q,
		"status": %q,
		"additions": %d,
		"deletions": %d,
		"patch": %q
	}`, filename, status, additions, deletions, patch))
}

func pullRequestResourceRaw(additions, deletions int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{"additions": %d, "deletions": %d}`, additions, deletions))
}

func TestBuildPullRequestDiff_ParsesFilesAndTotalsFromThePullResource(t *testing.T) {
	rawFiles := []json.RawMessage{
		changedFileRaw("internal/foo.go", "modified", "@@ -1,3 +1,3 @@", 12, 3),
	}

	diff, skipped, err := services.BuildPullRequestDiff(diffAttribution(t), pullRequestResourceRaw(12, 3), rawFiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if diff.Additions() != 12 || diff.Deletions() != 3 {
		t.Fatalf("Additions()/Deletions() = %d/%d, want 12/3", diff.Additions(), diff.Deletions())
	}
	if diff.Truncated() {
		t.Fatal("Truncated() = true, want false")
	}
	if len(diff.Files()) != 1 || diff.Files()[0].Patch() != "@@ -1,3 +1,3 @@" {
		t.Fatalf("Files() = %#v, want a single file with its patch preserved", diff.Files())
	}
}

func TestBuildPullRequestDiff_SuppressesPatchesWhenTotalChangedLinesExceedThreshold(t *testing.T) {
	rawFiles := []json.RawMessage{
		changedFileRaw("internal/huge.go", "modified", "@@ -1,3 +1,3 @@", 600, 500),
	}

	diff, _, err := services.BuildPullRequestDiff(diffAttribution(t), pullRequestResourceRaw(600, 500), rawFiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.Truncated() {
		t.Fatal("Truncated() = false, want true")
	}
	if len(diff.Files()) != 1 {
		t.Fatalf("got %d files, want 1 (the file list itself must still be present)", len(diff.Files()))
	}
	if diff.Files()[0].Patch() != "" {
		t.Fatalf("Files()[0].Patch() = %q, want empty (patch must be suppressed once truncated)", diff.Files()[0].Patch())
	}
	if diff.Files()[0].Filename() != "internal/huge.go" {
		t.Fatalf("Files()[0].Filename() = %q, want %q", diff.Files()[0].Filename(), "internal/huge.go")
	}
}

func TestBuildPullRequestDiff_SkipsAMalformedChangedFileAndRecordsASkipNote(t *testing.T) {
	rawFiles := []json.RawMessage{
		changedFileRaw("internal/foo.go", "modified", "@@ -1,3 +1,3 @@", 1, 1),
		changedFileRaw("internal/bad.go", "deleted-from-the-future", "", 0, 0),
	}

	diff, skipped, err := services.BuildPullRequestDiff(diffAttribution(t), pullRequestResourceRaw(1, 1), rawFiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if len(diff.Files()) != 1 || diff.Files()[0].Filename() != "internal/foo.go" {
		t.Fatalf("Files() = %#v, want only the well-formed file", diff.Files())
	}
}

func TestBuildPullRequestDiff_ReturnsAnErrorForAMalformedPullRequestResource(t *testing.T) {
	_, _, err := services.BuildPullRequestDiff(diffAttribution(t), json.RawMessage(`not json`), nil)

	if err == nil {
		t.Fatal("expected an error for a malformed pull request resource, got nil")
	}
}

func TestBuildPullRequestDiff_ReusesTheGivenAttribution(t *testing.T) {
	attribution := diffAttribution(t)

	diff, _, err := services.BuildPullRequestDiff(attribution, pullRequestResourceRaw(0, 0), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !diff.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", diff.Attribution(), attribution)
	}
}
