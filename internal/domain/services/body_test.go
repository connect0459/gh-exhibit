package services

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func mustParseIssueResource(t *testing.T, raw json.RawMessage) IssueResource {
	t.Helper()
	issue, err := ParseIssueResource(raw)
	if err != nil {
		t.Fatalf("ParseIssueResource() error = %v", err)
	}
	return issue
}

func TestParseIssueResource_ReturnsAnErrorWhenTheIssueResourceFailsToUnmarshal(t *testing.T) {
	_, err := ParseIssueResource(json.RawMessage(`{not valid json`))
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("error = %q, want it to mention the unmarshal failure", err.Error())
	}
}

func TestIssueResource_IsPullRequest_TrueWhenTheIssueResourceCarriesAPullRequestKey(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Add retry backoff",
		"pull_request": {"url": "https://api.github.com/repos/example/repo/pulls/1"}
	}`)

	if !mustParseIssueResource(t, raw).IsPullRequest() {
		t.Fatalf("IsPullRequest() = false, want true for a resource carrying pull_request")
	}
}

func TestIssueResource_IsPullRequest_FalseForAPlainIssueResource(t *testing.T) {
	raw := json.RawMessage(`{"title": "Something is broken"}`)

	if mustParseIssueResource(t, raw).IsPullRequest() {
		t.Fatalf("IsPullRequest() = true, want false for a resource with no pull_request key")
	}
}

func TestIssueResource_IsPullRequest_FalseWhenThePullRequestKeyIsExplicitlyNull(t *testing.T) {
	raw := json.RawMessage(`{"title": "Something is broken", "pull_request": null}`)

	if mustParseIssueResource(t, raw).IsPullRequest() {
		t.Fatalf("IsPullRequest() = true, want false for a resource with an explicit null pull_request")
	}
}

func TestIssueResource_HTMLURL_ReturnsTheIssueResourcesOwnHTMLURL(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Something is broken",
		"html_url": "https://github.com/example/repo/issues/1"
	}`)

	if got := mustParseIssueResource(t, raw).HTMLURL(); got != "https://github.com/example/repo/issues/1" {
		t.Fatalf("HTMLURL() = %q, want %q", got, "https://github.com/example/repo/issues/1")
	}
}

func TestIssueResource_ParentIssueRef_FalseWhenParentIssueURLIsAbsent(t *testing.T) {
	raw := json.RawMessage(`{"title": "Something is broken"}`)

	_, ok, err := mustParseIssueResource(t, raw).ParentIssueRef()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("ok = true, want false when parent_issue_url is absent")
	}
}

func TestIssueResource_ParentIssueRef_ParsesTheParentIssueURL(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Sub-issue",
		"parent_issue_url": "https://api.github.com/repos/octocat/hello-world/issues/64"
	}`)

	ref, ok, err := mustParseIssueResource(t, raw).ParentIssueRef()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("ok = false, want true when parent_issue_url is present")
	}
	if ref.Owner() != "octocat" || ref.Repo() != "hello-world" || ref.Number() != 64 {
		t.Fatalf("ParentIssueRef() = %+v, want owner=octocat repo=hello-world number=64", ref)
	}
}

func TestIssueResource_ParentIssueRef_ReturnsAnErrorForAMalformedParentIssueURL(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Sub-issue",
		"parent_issue_url": "https://api.github.com/not-the-expected-shape"
	}`)

	_, _, err := mustParseIssueResource(t, raw).ParentIssueRef()
	if err == nil {
		t.Fatal("expected an error for a malformed parent_issue_url, got nil")
	}
}

func TestBuildBody_BuildsTitleAndBodyFromAPlainIssueResource(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Something is broken",
		"body": "Steps to reproduce...",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1",
		"closed_at": null
	}`)

	body, title, err := BuildBody(mustParseIssueResource(t, raw), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title != "Something is broken" {
		t.Fatalf("title = %q, want %q", title, "Something is broken")
	}
	if body.Content() != "Steps to reproduce..." {
		t.Fatalf("Content() = %q, want %q", body.Content(), "Steps to reproduce...")
	}
	if body.Attribution().Author() != "octocat" {
		t.Fatalf("Attribution().Author() = %q, want %q", body.Attribution().Author(), "octocat")
	}
	if body.ClosedAt() != nil {
		t.Fatalf("ClosedAt() = %v, want nil", body.ClosedAt())
	}
	if body.MergedAt() != nil {
		t.Fatalf("MergedAt() = %v, want nil for a plain issue", body.MergedAt())
	}
}

func TestBuildBody_SetsClosedAtFromTheIssueResourceWhenPresent(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Fixed now",
		"body": "x",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1",
		"closed_at": "2026-07-02T00:00:00Z"
	}`)

	body, _, err := BuildBody(mustParseIssueResource(t, raw), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC)
	if body.ClosedAt() == nil || !body.ClosedAt().Equal(want) {
		t.Fatalf("ClosedAt() = %v, want %v", body.ClosedAt(), want)
	}
}

func TestBuildBody_SetsMergedAtFromThePullRequestResourceWhenGiven(t *testing.T) {
	rawIssue := json.RawMessage(`{
		"title": "Add retry backoff",
		"body": "x",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1",
		"pull_request": {"url": "https://api.github.com/repos/example/repo/pulls/1"}
	}`)
	rawPull := json.RawMessage(`{"merged_at": "2026-07-03T00:00:00Z"}`)

	body, _, err := BuildBody(mustParseIssueResource(t, rawIssue), rawPull)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	if body.MergedAt() == nil || !body.MergedAt().Equal(want) {
		t.Fatalf("MergedAt() = %v, want %v", body.MergedAt(), want)
	}
}

func TestBuildBody_LeavesMergedAtNilWhenThePullRequestResourceHasNotMergedYet(t *testing.T) {
	rawIssue := json.RawMessage(`{
		"title": "Add retry backoff",
		"body": "x",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1",
		"pull_request": {"url": "https://api.github.com/repos/example/repo/pulls/1"}
	}`)
	rawPull := json.RawMessage(`{"merged_at": null}`)

	body, _, err := BuildBody(mustParseIssueResource(t, rawIssue), rawPull)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.MergedAt() != nil {
		t.Fatalf("MergedAt() = %v, want nil", body.MergedAt())
	}
}

func TestBuildBody_AttributesAnIssueFromADeletedAccountToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Orphaned issue",
		"body": "x",
		"user": null,
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1"
	}`)

	body, _, err := BuildBody(mustParseIssueResource(t, raw), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", body.Attribution().Author(), "ghost")
	}
}

func TestBuildBody_ReturnsAnErrorWhenTheIssueResourceHasNoHTMLURL(t *testing.T) {
	raw := json.RawMessage(`{
		"title": "Missing url",
		"body": "x",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": ""
	}`)

	_, _, err := BuildBody(mustParseIssueResource(t, raw), nil)
	if err == nil {
		t.Fatalf("expected an error propagated from valueobjects.NewAttribution, got nil")
	}
}

func TestBuildBody_ReturnsAnErrorWhenThePullRequestResourceFailsToUnmarshal(t *testing.T) {
	rawIssue := json.RawMessage(`{
		"title": "Add retry backoff",
		"body": "x",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1",
		"pull_request": {"url": "https://api.github.com/repos/example/repo/pulls/1"}
	}`)

	_, _, err := BuildBody(mustParseIssueResource(t, rawIssue), json.RawMessage(`{not valid json`))
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "unmarshal") {
		t.Fatalf("error = %q, want it to mention the unmarshal failure", err.Error())
	}
}
