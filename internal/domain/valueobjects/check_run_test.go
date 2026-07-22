package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func mustNewCheckRun(t *testing.T, name string, outcome valueobjects.CheckOutcome, rawURL string) valueobjects.CheckRun {
	t.Helper()
	run, err := valueobjects.NewCheckRun(name, outcome, rawURL)
	if err != nil {
		t.Fatalf("NewCheckRun(): unexpected error: %v", err)
	}
	return run
}

func TestNewCheckRun_RejectsAnEmptyName(t *testing.T) {
	_, err := valueobjects.NewCheckRun("", valueobjects.CheckOutcomeSuccess, "https://github.com/octocat/hello-world/runs/1")

	if err == nil {
		t.Fatal("expected an error for an empty name, got nil")
	}
}

func TestNewCheckRun_RejectsAMalformedURL(t *testing.T) {
	_, err := valueobjects.NewCheckRun("build", valueobjects.CheckOutcomeSuccess, "not a url")

	if err == nil {
		t.Fatal("expected an error for a malformed url, got nil")
	}
}

func TestCheckRun_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	run := mustNewCheckRun(t, "build", valueobjects.CheckOutcomeFailure, "https://github.com/octocat/hello-world/runs/1")

	if run.Name() != "build" {
		t.Fatalf("Name() = %q, want %q", run.Name(), "build")
	}
	if run.Outcome() != valueobjects.CheckOutcomeFailure {
		t.Fatalf("Outcome() = %v, want %v", run.Outcome(), valueobjects.CheckOutcomeFailure)
	}
	if run.URL().String() != "https://github.com/octocat/hello-world/runs/1" {
		t.Fatalf("URL() = %q, want %q", run.URL().String(), "https://github.com/octocat/hello-world/runs/1")
	}
}

func TestCheckRun_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	a := mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/octocat/hello-world/runs/1")
	b := mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/octocat/hello-world/runs/1")

	if !a.Equals(b) {
		t.Fatal("expected check runs with matching fields to be equal")
	}
}

func TestCheckRun_Equals_TreatsDifferentOutcomeAsNotEqual(t *testing.T) {
	a := mustNewCheckRun(t, "build", valueobjects.CheckOutcomeSuccess, "https://github.com/octocat/hello-world/runs/1")
	b := mustNewCheckRun(t, "build", valueobjects.CheckOutcomeFailure, "https://github.com/octocat/hello-world/runs/1")

	if a.Equals(b) {
		t.Fatal("expected check runs with different outcomes to not be equal")
	}
}
