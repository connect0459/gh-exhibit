package services_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func relationsAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	attribution, err := valueobjects.NewAttribution("octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/issues/69")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}
	return attribution
}

func issueSummaryRaw(number int, title, state, htmlURL string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"number": %d,
		"title": %q,
		"state": %q,
		"html_url": %q
	}`, number, title, state, htmlURL))
}

func TestBuildParentIssue_ParsesTheParentsFields(t *testing.T) {
	raw := issueSummaryRaw(64, "Round of Tier 1 entries", "open", "https://github.com/example/repo/issues/64")

	parent, err := services.BuildParentIssue(relationsAttribution(t), raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parent.Parent().Number() != 64 {
		t.Fatalf("Parent().Number() = %d, want %d", parent.Parent().Number(), 64)
	}
	if parent.Parent().Title() != "Round of Tier 1 entries" {
		t.Fatalf("Parent().Title() = %q, want %q", parent.Parent().Title(), "Round of Tier 1 entries")
	}
	if parent.Parent().State() != valueobjects.IssueStateOpen {
		t.Fatalf("Parent().State() = %v, want %v", parent.Parent().State(), valueobjects.IssueStateOpen)
	}
}

func TestBuildParentIssue_ReusesTheGivenAttribution(t *testing.T) {
	attribution := relationsAttribution(t)
	raw := issueSummaryRaw(64, "title", "open", "https://github.com/example/repo/issues/64")

	parent, err := services.BuildParentIssue(attribution, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !parent.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", parent.Attribution(), attribution)
	}
}

func TestBuildParentIssue_ReturnsAnErrorWhenTheParentResourceFailsToUnmarshal(t *testing.T) {
	_, err := services.BuildParentIssue(relationsAttribution(t), json.RawMessage(`{not valid json`))
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestBuildParentIssue_ReturnsAnErrorForAnUnrecognizedState(t *testing.T) {
	raw := issueSummaryRaw(64, "title", "archived", "https://github.com/example/repo/issues/64")

	_, err := services.BuildParentIssue(relationsAttribution(t), raw)
	if err == nil {
		t.Fatal("expected an error for an unrecognized state, got nil")
	}
}

func TestBuildSubIssues_ParsesEveryChildsFields(t *testing.T) {
	rawChildren := []json.RawMessage{
		issueSummaryRaw(65, "Include issue/PR labels", "closed", "https://github.com/example/repo/issues/65"),
		issueSummaryRaw(69, "Include parent/child issue relationships", "open", "https://github.com/example/repo/issues/69"),
	}

	subIssues, skipped, err := services.BuildSubIssues(relationsAttribution(t), rawChildren)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	children := subIssues.Children()
	if len(children) != 2 {
		t.Fatalf("got %d children, want 2", len(children))
	}
	if children[0].Number() != 65 || children[0].State() != valueobjects.IssueStateClosed {
		t.Fatalf("children[0] = %#v, want number=65 state=closed", children[0])
	}
	if children[1].Number() != 69 || children[1].State() != valueobjects.IssueStateOpen {
		t.Fatalf("children[1] = %#v, want number=69 state=open", children[1])
	}
}

func TestBuildSubIssues_SkipsAMalformedChildAndRecordsASkipNote(t *testing.T) {
	rawChildren := []json.RawMessage{
		issueSummaryRaw(65, "good", "open", "https://github.com/example/repo/issues/65"),
		issueSummaryRaw(66, "bad", "archived", "https://github.com/example/repo/issues/66"),
	}

	subIssues, skipped, err := services.BuildSubIssues(relationsAttribution(t), rawChildren)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if len(subIssues.Children()) != 1 || subIssues.Children()[0].Number() != 65 {
		t.Fatalf("Children() = %#v, want only the well-formed child", subIssues.Children())
	}
}

func TestBuildSubIssues_ReusesTheGivenAttribution(t *testing.T) {
	attribution := relationsAttribution(t)

	subIssues, _, err := services.BuildSubIssues(attribution, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !subIssues.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", subIssues.Attribution(), attribution)
	}
}
