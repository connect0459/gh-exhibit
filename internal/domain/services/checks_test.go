package services_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func checkRunRaw(name, status, conclusion, htmlURL string) json.RawMessage {
	raw, _ := json.Marshal(struct {
		Name       string `json:"name"`
		Status     string `json:"status"`
		Conclusion string `json:"conclusion,omitempty"`
		HTMLURL    string `json:"html_url"`
	}{Name: name, Status: status, Conclusion: conclusion, HTMLURL: htmlURL})
	return raw
}

func TestBuildPullRequestChecks_ParsesEveryCheckRunField(t *testing.T) {
	rawChecks := []json.RawMessage{
		checkRunRaw("build", "completed", "success", "https://github.com/example/repo/runs/1"),
	}
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)

	checks, skipped, err := services.BuildPullRequestChecks(diffAttribution(t), "abc1234567", capturedAt, rawChecks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(checks.Runs()) != 1 {
		t.Fatalf("got %d runs, want 1", len(checks.Runs()))
	}
	got := checks.Runs()[0]
	if got.Name() != "build" {
		t.Fatalf("Name() = %q, want %q", got.Name(), "build")
	}
	if got.Outcome() != valueobjects.CheckOutcomeSuccess {
		t.Fatalf("Outcome() = %v, want %v", got.Outcome(), valueobjects.CheckOutcomeSuccess)
	}
	if got.URL().String() != "https://github.com/example/repo/runs/1" {
		t.Fatalf("URL() = %q, want %q", got.URL().String(), "https://github.com/example/repo/runs/1")
	}
}

func TestBuildPullRequestChecks_SkipsAMalformedCheckRunAndRecordsASkipNote(t *testing.T) {
	rawChecks := []json.RawMessage{
		checkRunRaw("build", "completed", "success", "https://github.com/example/repo/runs/1"),
		checkRunRaw("broken", "not-a-real-status", "", "https://github.com/example/repo/runs/2"),
	}

	checks, skipped, err := services.BuildPullRequestChecks(diffAttribution(t), "abc1234567", time.Now(), rawChecks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if len(checks.Runs()) != 1 || checks.Runs()[0].Name() != "build" {
		t.Fatalf("Runs() = %#v, want only the well-formed check run", checks.Runs())
	}
}

func TestBuildPullRequestChecks_ReusesTheGivenAttributionHeadSHAAndCapturedAt(t *testing.T) {
	attribution := diffAttribution(t)
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)

	checks, _, err := services.BuildPullRequestChecks(attribution, "abc1234567", capturedAt, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !checks.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", checks.Attribution(), attribution)
	}
	if checks.HeadSHA() != "abc1234567" {
		t.Fatalf("HeadSHA() = %q, want %q", checks.HeadSHA(), "abc1234567")
	}
	if !checks.CapturedAt().Equal(capturedAt) {
		t.Fatalf("CapturedAt() = %v, want %v", checks.CapturedAt(), capturedAt)
	}
}
