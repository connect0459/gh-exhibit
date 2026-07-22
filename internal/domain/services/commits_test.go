package services_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

func commitRaw(sha, authorName, authoredAt, committerName, committedAt, message string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"sha": %q,
		"commit": {
			"author": {"name": %q, "date": %q},
			"committer": {"name": %q, "date": %q},
			"message": %q
		}
	}`, sha, authorName, authoredAt, committerName, committedAt, message))
}

func TestBuildPullRequestCommits_ParsesEveryCommitField(t *testing.T) {
	rawCommits := []json.RawMessage{
		commitRaw("abc1234567", "octocat", "2026-07-01T00:00:00Z", "web-flow", "2026-07-01T00:05:00Z", "feat: add a thing"),
	}

	commits, skipped, err := services.BuildPullRequestCommits(diffAttribution(t), rawCommits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(commits.Commits()) != 1 {
		t.Fatalf("got %d commits, want 1", len(commits.Commits()))
	}
	got := commits.Commits()[0]
	if got.SHA() != "abc1234567" {
		t.Fatalf("SHA() = %q, want %q", got.SHA(), "abc1234567")
	}
	if got.AuthorName() != "octocat" {
		t.Fatalf("AuthorName() = %q, want %q", got.AuthorName(), "octocat")
	}
	if got.CommitterName() != "web-flow" {
		t.Fatalf("CommitterName() = %q, want %q", got.CommitterName(), "web-flow")
	}
	if got.Message() != "feat: add a thing" {
		t.Fatalf("Message() = %q, want %q", got.Message(), "feat: add a thing")
	}
	if got.AuthoredAt().Format("2006-01-02T15:04:05Z") != "2026-07-01T00:00:00Z" {
		t.Fatalf("AuthoredAt() = %v, want 2026-07-01T00:00:00Z", got.AuthoredAt())
	}
	if got.CommittedAt().Format("2006-01-02T15:04:05Z") != "2026-07-01T00:05:00Z" {
		t.Fatalf("CommittedAt() = %v, want 2026-07-01T00:05:00Z", got.CommittedAt())
	}
}

func TestBuildPullRequestCommits_SkipsAMalformedCommitAndRecordsASkipNote(t *testing.T) {
	rawCommits := []json.RawMessage{
		commitRaw("abc1234567", "octocat", "2026-07-01T00:00:00Z", "octocat", "2026-07-01T00:00:00Z", "good"),
		json.RawMessage(`{"sha": "", "commit": {"author": {}, "committer": {}}}`),
	}

	commits, skipped, err := services.BuildPullRequestCommits(diffAttribution(t), rawCommits)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if len(commits.Commits()) != 1 || commits.Commits()[0].SHA() != "abc1234567" {
		t.Fatalf("Commits() = %#v, want only the well-formed commit", commits.Commits())
	}
}

func TestBuildPullRequestCommits_ReusesTheGivenAttribution(t *testing.T) {
	attribution := diffAttribution(t)

	commits, _, err := services.BuildPullRequestCommits(attribution, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !commits.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", commits.Attribution(), attribution)
	}
}
