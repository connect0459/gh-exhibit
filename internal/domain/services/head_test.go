package services

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPullRequestHeadSHA_ReturnsTheHeadCommitSHA(t *testing.T) {
	raw := json.RawMessage(`{"head": {"sha": "abc1234567890"}}`)

	got, err := PullRequestHeadSHA(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abc1234567890" {
		t.Fatalf("PullRequestHeadSHA() = %q, want %q", got, "abc1234567890")
	}
}

func TestPullRequestHeadSHA_ReturnsAnErrorWhenTheResourceFailsToUnmarshal(t *testing.T) {
	_, err := PullRequestHeadSHA(json.RawMessage(`{not valid json`))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("error = %q, want it to mention the unmarshal failure", err.Error())
	}
}

func TestPullRequestHeadSHA_ReturnsAnErrorWhenHeadSHAIsEmpty(t *testing.T) {
	_, err := PullRequestHeadSHA(json.RawMessage(`{"head": {"sha": ""}}`))
	if err == nil {
		t.Fatal("expected an error for an empty head sha, got nil")
	}
}
