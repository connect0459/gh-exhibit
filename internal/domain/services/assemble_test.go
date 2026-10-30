package services_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

const assemblePlainIssueJSON = `{
	"title": "Something is broken",
	"body": "Steps to reproduce...",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1"
}`

const assemblePullRequestIssueJSON = `{
	"title": "Add retry backoff",
	"body": "This adds retry.",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/2",
	"pull_request": {"url": "https://api.github.com/repos/example/repo/pulls/2"}
}`

const assembleCommentedEventJSON = `{
	"event": "commented",
	"id": 100,
	"user": {"login": "reviewer"},
	"body": "Looks fine to me.",
	"created_at": "2026-07-02T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1#issuecomment-100"
}`

const assembleMergedPullRequestJSON = `{"merged_at": "2026-07-03T00:00:00Z", "head": {"sha": "abc1234567"}}`

const assembleCheckRunJSON = `{"name":"build","status":"completed","conclusion":"success","html_url":"https://github.com/example/repo/runs/1"}`

const assembleSubIssueJSON = `{"number":65,"title":"Include issue/PR labels","state":"closed","html_url":"https://github.com/example/repo/issues/65"}`

func renderToString(t *testing.T, render func(*bytes.Buffer) error) string {
	t.Helper()
	var buf bytes.Buffer
	if err := render(&buf); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buf.String()
}

func TestAssembleDocument_TitlesTheDocumentWithTheIssueTitle(t *testing.T) {
	raw := services.RawEvidence{Issue: json.RawMessage(assemblePlainIssueJSON)}

	doc, _, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() error = %v", err)
	}

	if doc.Title() != "Something is broken" {
		t.Fatalf("Title() = %q, want %q", doc.Title(), "Something is broken")
	}
}

func TestAssembleDocument_PutsTheBodyFirstThenTimelineEntriesInOrder(t *testing.T) {
	raw := services.RawEvidence{
		Issue:    json.RawMessage(assemblePlainIssueJSON),
		Timeline: []json.RawMessage{json.RawMessage(assembleCommentedEventJSON)},
	}

	doc, _, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() error = %v", err)
	}

	rendered := renderToString(t, func(b *bytes.Buffer) error { return doc.Render(b) })
	body := strings.Index(rendered, "Steps to reproduce...")
	comment := strings.Index(rendered, "Looks fine to me.")
	if body < 0 || comment < 0 || body > comment {
		t.Fatalf("rendered = %q, want the issue body before the comment", rendered)
	}
}

func TestAssembleDocument_ReflectsThePullRequestResourceForAPullRequest(t *testing.T) {
	raw := services.RawEvidence{
		Issue:       json.RawMessage(assemblePullRequestIssueJSON),
		PullRequest: json.RawMessage(assembleMergedPullRequestJSON),
	}

	doc, _, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() error = %v", err)
	}

	rendered := renderToString(t, func(b *bytes.Buffer) error { return doc.Render(b) })
	if !strings.Contains(rendered, `"merged":"2026-07-03T00:00:00Z"`) {
		t.Fatalf("rendered = %q, want it to reflect merged_at from the pull request resource", rendered)
	}
}

func TestAssembleDocument_RejectsAPullRequestWithoutItsPullRequestResource(t *testing.T) {
	raw := services.RawEvidence{Issue: json.RawMessage(assemblePullRequestIssueJSON)}

	_, _, err := services.AssembleDocument(raw, time.Time{})

	if err == nil {
		t.Fatal("expected an error for a pull request whose pull request resource is missing, got nil")
	}
}

func TestAssembleDocument_IncludesChecksOnlyWhenThePullRequestHasCheckRuns(t *testing.T) {
	withoutChecks := services.RawEvidence{
		Issue:       json.RawMessage(assemblePullRequestIssueJSON),
		PullRequest: json.RawMessage(assembleMergedPullRequestJSON),
	}
	withChecks := withoutChecks
	withChecks.CheckRuns = []json.RawMessage{json.RawMessage(assembleCheckRunJSON)}

	docWithout, _, err := services.AssembleDocument(withoutChecks, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() without check runs error = %v", err)
	}
	docWith, _, err := services.AssembleDocument(withChecks, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() with check runs error = %v", err)
	}

	if len(docWith.Entries()) != len(docWithout.Entries())+1 {
		t.Fatalf("entries with check runs = %d, without = %d, want exactly one more entry", len(docWith.Entries()), len(docWithout.Entries()))
	}
}

func TestAssembleDocument_StampsTheChecksWithTheGivenObservationTime(t *testing.T) {
	observedAt := time.Date(2026, 7, 4, 12, 30, 0, 0, time.UTC)
	raw := services.RawEvidence{
		Issue:       json.RawMessage(assemblePullRequestIssueJSON),
		PullRequest: json.RawMessage(assembleMergedPullRequestJSON),
		CheckRuns:   []json.RawMessage{json.RawMessage(assembleCheckRunJSON)},
	}

	doc, _, err := services.AssembleDocument(raw, observedAt)
	if err != nil {
		t.Fatalf("AssembleDocument() error = %v", err)
	}

	rendered := renderToString(t, func(b *bytes.Buffer) error { return doc.Render(b) })
	if !strings.Contains(rendered, "2026-07-04T12:30:00Z") {
		t.Fatalf("rendered = %q, want the observation time 2026-07-04T12:30:00Z", rendered)
	}
}

func TestAssembleDocument_IncludesSubIssuesForAPlainIssue(t *testing.T) {
	raw := services.RawEvidence{
		Issue:     json.RawMessage(assemblePlainIssueJSON),
		SubIssues: []json.RawMessage{json.RawMessage(assembleSubIssueJSON)},
	}

	doc, _, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() error = %v", err)
	}

	rendered := renderToString(t, func(b *bytes.Buffer) error { return doc.Render(b) })
	if !strings.Contains(rendered, "Include issue/PR labels") {
		t.Fatalf("rendered = %q, want it to list the sub-issue", rendered)
	}
}

func TestAssembleDocument_ReportsAnUnparsableTimelineItemAsASkipNoteInsteadOfFailing(t *testing.T) {
	raw := services.RawEvidence{
		Issue:    json.RawMessage(assemblePlainIssueJSON),
		Timeline: []json.RawMessage{json.RawMessage(`{"event":"commented","id":"not-a-number"}`)},
	}

	_, skipped, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() error = %v, want nil (a single bad item must not fail the whole document)", err)
	}

	if len(skipped) != 1 {
		t.Fatalf("skipped = %d notes, want 1", len(skipped))
	}
}

func TestAssembleDocument_RejectsAnUnparsableIssueResource(t *testing.T) {
	raw := services.RawEvidence{Issue: json.RawMessage(`not json`)}

	_, _, err := services.AssembleDocument(raw, time.Time{})

	if err == nil {
		t.Fatal("expected an error for an unparsable issue resource, got nil")
	}
}
