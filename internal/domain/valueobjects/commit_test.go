package valueobjects_test

import (
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func mustNewCommit(t *testing.T, sha, authorName string, authoredAt time.Time, committerName string, committedAt time.Time, message string) valueobjects.Commit {
	t.Helper()
	commit, err := valueobjects.NewCommit(sha, authorName, authoredAt, committerName, committedAt, message)
	if err != nil {
		t.Fatalf("NewCommit(): unexpected error: %v", err)
	}
	return commit
}

func TestNewCommit_RejectsAnEmptySHA(t *testing.T) {
	_, err := valueobjects.NewCommit("", "octocat", time.Now(), "octocat", time.Now(), "a message")

	if err == nil {
		t.Fatal("expected an error for an empty sha, got nil")
	}
}

func TestCommit_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	authoredAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	committedAt := time.Date(2026, 7, 1, 0, 5, 0, 0, time.UTC)
	commit := mustNewCommit(t, "abc1234", "octocat", authoredAt, "web-flow", committedAt, "feat: add a thing\n\nSome body text.")

	if commit.SHA() != "abc1234" {
		t.Fatalf("SHA() = %q, want %q", commit.SHA(), "abc1234")
	}
	if commit.AuthorName() != "octocat" {
		t.Fatalf("AuthorName() = %q, want %q", commit.AuthorName(), "octocat")
	}
	if !commit.AuthoredAt().Equal(authoredAt) {
		t.Fatalf("AuthoredAt() = %v, want %v", commit.AuthoredAt(), authoredAt)
	}
	if commit.CommitterName() != "web-flow" {
		t.Fatalf("CommitterName() = %q, want %q", commit.CommitterName(), "web-flow")
	}
	if !commit.CommittedAt().Equal(committedAt) {
		t.Fatalf("CommittedAt() = %v, want %v", commit.CommittedAt(), committedAt)
	}
	if commit.Message() != "feat: add a thing\n\nSome body text." {
		t.Fatalf("Message() = %q, want %q", commit.Message(), "feat: add a thing\n\nSome body text.")
	}
}

func TestCommit_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	authoredAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	a := mustNewCommit(t, "abc1234", "octocat", authoredAt, "octocat", authoredAt, "message")
	b := mustNewCommit(t, "abc1234", "octocat", authoredAt, "octocat", authoredAt, "message")

	if !a.Equals(b) {
		t.Fatal("expected commits with matching fields to be equal")
	}
}

func TestCommit_Equals_TreatsDifferentCommitterNameAsNotEqual(t *testing.T) {
	authoredAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	a := mustNewCommit(t, "abc1234", "octocat", authoredAt, "octocat", authoredAt, "message")
	b := mustNewCommit(t, "abc1234", "octocat", authoredAt, "web-flow", authoredAt, "message")

	if a.Equals(b) {
		t.Fatal("expected commits with different committer names to not be equal")
	}
}

func TestCommit_Equals_TreatsDifferentMessageAsNotEqual(t *testing.T) {
	authoredAt := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	a := mustNewCommit(t, "abc1234", "octocat", authoredAt, "octocat", authoredAt, "message one")
	b := mustNewCommit(t, "abc1234", "octocat", authoredAt, "octocat", authoredAt, "message two")

	if a.Equals(b) {
		t.Fatal("expected commits with different messages to not be equal")
	}
}
