package services_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

const testIssueURL = "https://github.com/example/repo/issues/1"

func loadTestdata(t *testing.T, name string) json.RawMessage {
	t.Helper()
	raw, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("unexpected error reading testdata/%s: %v", name, err)
	}
	return raw
}

func reviewedEventRaw(id int64, login, body string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"id": %d,
		"event": "reviewed",
		"user": {"login": %q},
		"body": %q,
		"state": "commented",
		"submitted_at": "2026-07-02T14:19:40Z",
		"html_url": "https://github.com/example/repo/pull/1#pullrequestreview-%d"
	}`, id, login, body, id))
}

func commentedEventRaw(login, body, url string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"event": "commented",
		"user": {"login": %q},
		"body": %q,
		"created_at": "2026-07-01T10:00:00Z",
		"html_url": %q
	}`, login, body, url))
}

func reviewCommentRaw(reviewID int64, login, body, path string, line int) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"pull_request_review_id": %d,
		"user": {"login": %q},
		"body": %q,
		"path": %q,
		"line": %d,
		"diff_hunk": "@@ -1,3 +1,3 @@",
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": "https://github.com/example/repo/pull/1#discussion_r%d"
	}`, reviewID, login, body, path, line, line))
}

func TestBuildEntries_InsertsAnInlineCommentImmediatelyAfterItsParentReview(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{reviewCommentRaw(1001, "octocat", "Nit here.", "main.go", 10)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if _, ok := entries[0].(valueobjects.PullRequestReview); !ok {
		t.Fatalf("entries[0] = %#v, want PullRequestReview", entries[0])
	}
	if _, ok := entries[1].(valueobjects.InlineReviewComment); !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
}

func TestBuildEntries_PreservesOrderOfMultipleCommentsOnTheSameReview(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{
		reviewCommentRaw(1001, "octocat", "First nit.", "main.go", 10),
		reviewCommentRaw(1001, "octocat", "Second nit.", "main.go", 20),
	}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	first, ok := entries[1].(valueobjects.InlineReviewComment)
	if !ok || first.Body() != "First nit." {
		t.Fatalf("entries[1] = %#v, want the first inline comment", entries[1])
	}
	second, ok := entries[2].(valueobjects.InlineReviewComment)
	if !ok || second.Body() != "Second nit." {
		t.Fatalf("entries[2] = %#v, want the second inline comment", entries[2])
	}
}

func TestBuildEntries_StillRendersAReviewWithNoMatchingInlineComments(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Approved, nothing to add.")}

	entries, skipped := services.BuildEntries(rawTimeline, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if _, ok := entries[0].(valueobjects.PullRequestReview); !ok {
		t.Fatalf("entries[0] = %#v, want PullRequestReview", entries[0])
	}
}

func TestBuildEntries_StillRendersAnOrphanedInlineCommentAtTheEnd(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Approved.")}
	rawComments := []json.RawMessage{
		reviewCommentRaw(9999, "octocat", "Comment on a review we never fetched.", "main.go", 5),
	}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	if _, ok := entries[0].(valueobjects.PullRequestReview); !ok {
		t.Fatalf("entries[0] = %#v, want PullRequestReview", entries[0])
	}
	orphan, ok := entries[1].(valueobjects.InlineReviewComment)
	if !ok || orphan.Body() != "Comment on a review we never fetched." {
		t.Fatalf("entries[1] = %#v, want the orphaned inline comment", entries[1])
	}
}

func TestBuildEntries_RendersAReviewAndItsCommentsOnlyOnceWhenTheReviewedEventIsDuplicated(t *testing.T) {
	rawTimeline := []json.RawMessage{
		reviewedEventRaw(1001, "octocat", "Overall looks fine."),
		reviewedEventRaw(1001, "octocat", "Overall looks fine."),
	}
	rawComments := []json.RawMessage{reviewCommentRaw(1001, "octocat", "Nit here.", "main.go", 10)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1 (the duplicate reviewed event)", len(skipped))
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (one review, one comment, no duplication)", len(entries))
	}

	if _, ok := entries[0].(valueobjects.PullRequestReview); !ok {
		t.Fatalf("entries[0] = %#v, want PullRequestReview", entries[0])
	}
	if _, ok := entries[1].(valueobjects.InlineReviewComment); !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
}

func TestBuildEntries_RendersAReviewCommentOnlyOnceWhenItsOwnIDIsDuplicated(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	comment := json.RawMessage(`{
		"id": 9001,
		"pull_request_review_id": 1001,
		"user": {"login": "octocat"},
		"body": "Nit here.",
		"path": "main.go",
		"line": 10,
		"diff_hunk": "@@ -1,3 +1,3 @@",
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": "https://github.com/example/repo/pull/1#discussion_r9001"
	}`)
	rawComments := []json.RawMessage{comment, comment}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1 (the duplicate review comment)", len(skipped))
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (one review, one comment, no duplication)", len(entries))
	}

	if _, ok := entries[0].(valueobjects.PullRequestReview); !ok {
		t.Fatalf("entries[0] = %#v, want PullRequestReview", entries[0])
	}
	if _, ok := entries[1].(valueobjects.InlineReviewComment); !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
}

func TestBuildEntries_DoesNotJoinACommentToAMalformedReviewWithIDZero(t *testing.T) {
	rawTimeline := []json.RawMessage{
		reviewedEventRaw(0, "octocat", "A review missing its own id."),
		reviewedEventRaw(2002, "octocat", "A second, well-formed review."),
	}
	rawComments := []json.RawMessage{
		reviewCommentRaw(0, "octocat", "Unrelated comment whose review id also defaulted to zero.", "main.go", 5),
	}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}

	// If id=0 wrongly matched pull_request_review_id=0, the comment would
	// land at entries[1], immediately after the id-zero review, instead of
	// at the end as an orphan.
	if _, ok := entries[1].(valueobjects.PullRequestReview); !ok {
		t.Fatalf("entries[1] = %#v, want the second PullRequestReview, not the comment", entries[1])
	}
	orphan, ok := entries[2].(valueobjects.InlineReviewComment)
	if !ok || orphan.Body() != "Unrelated comment whose review id also defaulted to zero." {
		t.Fatalf("entries[2] = %#v, want the orphaned inline comment", entries[2])
	}
}

func TestBuildEntries_RendersAFileLevelReviewCommentWithoutALine(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{json.RawMessage(`{
		"pull_request_review_id": 1001,
		"user": {"login": "octocat"},
		"body": "This file as a whole needs a rewrite.",
		"path": "main.go",
		"subject_type": "file",
		"line": null,
		"original_line": null,
		"diff_hunk": "",
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": "https://github.com/example/repo/pull/1#discussion_r10"
	}`)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	comment, ok := entries[1].(valueobjects.InlineReviewComment)
	if !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
	if comment.Context().Line() != nil {
		t.Fatalf("Context().Line() = %v, want nil for a file-level comment", comment.Context().Line())
	}
	if comment.Context().Outdated() {
		t.Fatal("expected a file-level comment to not be marked outdated")
	}
}

func TestBuildEntries_FallsBackToOriginalLineForAnOutdatedReviewComment(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{json.RawMessage(`{
		"pull_request_review_id": 1001,
		"user": {"login": "octocat"},
		"body": "This diff has since changed underneath the comment.",
		"path": "main.go",
		"line": null,
		"original_line": 346,
		"diff_hunk": "",
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": "https://github.com/example/repo/pull/1#discussion_r10"
	}`)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	comment, ok := entries[1].(valueobjects.InlineReviewComment)
	if !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
	if comment.Context().Line() == nil || *comment.Context().Line() != 346 {
		t.Fatalf("Context().Line() = %v, want 346 (original_line)", comment.Context().Line())
	}
	if !comment.Context().Outdated() {
		t.Fatal("expected the comment's context to be marked outdated")
	}
}

func TestBuildEntries_RecordsAStartLineForARangeAnchoredReviewComment(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{json.RawMessage(`{
		"pull_request_review_id": 1001,
		"user": {"login": "octocat"},
		"body": "This whole block needs a rework.",
		"path": "main.go",
		"start_line": 10,
		"line": 15,
		"diff_hunk": "",
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": "https://github.com/example/repo/pull/1#discussion_r10"
	}`)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	comment, ok := entries[1].(valueobjects.InlineReviewComment)
	if !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
	if comment.Context().Line() == nil || *comment.Context().Line() != 15 {
		t.Fatalf("Context().Line() = %v, want 15", comment.Context().Line())
	}
	if comment.Context().StartLine() == nil || *comment.Context().StartLine() != 10 {
		t.Fatalf("Context().StartLine() = %v, want 10", comment.Context().StartLine())
	}
}

func TestBuildEntries_FallsBackToOriginalStartLineForAnOutdatedRangeReviewComment(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{json.RawMessage(`{
		"pull_request_review_id": 1001,
		"user": {"login": "octocat"},
		"body": "This diff has since changed underneath the comment.",
		"path": "main.go",
		"line": null,
		"original_line": 346,
		"start_line": null,
		"original_start_line": 340,
		"diff_hunk": "",
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": "https://github.com/example/repo/pull/1#discussion_r10"
	}`)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	comment, ok := entries[1].(valueobjects.InlineReviewComment)
	if !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
	if comment.Context().Line() == nil || *comment.Context().Line() != 346 {
		t.Fatalf("Context().Line() = %v, want 346 (original_line)", comment.Context().Line())
	}
	if comment.Context().StartLine() == nil || *comment.Context().StartLine() != 340 {
		t.Fatalf("Context().StartLine() = %v, want 340 (original_start_line)", comment.Context().StartLine())
	}
	if !comment.Context().Outdated() {
		t.Fatal("expected the comment's context to be marked outdated")
	}
}

func TestBuildEntries_AttributesAReviewCommentFromADeletedAccountToGhost(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{json.RawMessage(`{
		"pull_request_review_id": 1001,
		"user": null,
		"body": "This comment survives its author's account being deleted.",
		"path": "main.go",
		"line": 10,
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": "https://github.com/example/repo/pull/1#discussion_r10"
	}`)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	comment, ok := entries[1].(valueobjects.InlineReviewComment)
	if !ok {
		t.Fatalf("entries[1] = %#v, want InlineReviewComment", entries[1])
	}
	if comment.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", comment.Attribution().Author(), "ghost")
	}
}

func TestBuildEntries_SkipsAReviewCommentThatFailsToUnmarshal(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{json.RawMessage(`{"pull_request_review_id": "not-a-number"}`)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the review, without the unmarshalable comment)", len(entries))
	}
}

func TestBuildEntries_SkipsACommentedEventWithNoHTMLURL(t *testing.T) {
	raw := commentedEventRaw("octocat", "Looks good.", "")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "attribution") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the attribution failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsAReviewedEventWithNoHTMLURL(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 1001,
		"event": "reviewed",
		"user": {"login": "octocat"},
		"body": "Overall looks fine.",
		"state": "commented",
		"submitted_at": "2026-07-02T14:19:40Z",
		"html_url": ""
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "attribution") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the attribution failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsAReviewCommentWithNoHTMLURL(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{json.RawMessage(`{
		"pull_request_review_id": 1001,
		"user": {"login": "octocat"},
		"body": "Nit here.",
		"path": "main.go",
		"line": 10,
		"diff_hunk": "@@ -1,3 +1,3 @@",
		"created_at": "2026-07-02T14:19:39Z",
		"html_url": ""
	}`)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the review, without the URL-less comment)", len(entries))
	}
	if !strings.Contains(skipped[0].Reason, "attribution") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the attribution failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsAReviewCommentWithNoPath(t *testing.T) {
	rawTimeline := []json.RawMessage{reviewedEventRaw(1001, "octocat", "Overall looks fine.")}
	rawComments := []json.RawMessage{reviewCommentRaw(1001, "octocat", "Nit here.", "", 10)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the review, without the pathless comment)", len(entries))
	}
}

func TestBuildEntries_PreservesOverallOrderWhenCommentsAndReviewsInterleave(t *testing.T) {
	rawTimeline := []json.RawMessage{
		commentedEventRaw("alice", "First comment.", "https://github.com/example/repo/issues/1#issuecomment-1"),
		reviewedEventRaw(1001, "octocat", "Overall looks fine."),
		commentedEventRaw("bob", "Second comment.", "https://github.com/example/repo/issues/1#issuecomment-2"),
	}
	rawComments := []json.RawMessage{reviewCommentRaw(1001, "octocat", "Nit here.", "main.go", 10)}

	entries, skipped := services.BuildEntries(rawTimeline, rawComments, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(entries))
	}

	first, ok := entries[0].(valueobjects.IssueComment)
	if !ok || first.Body() != "First comment." {
		t.Fatalf("entries[0] = %#v, want the first issue comment", entries[0])
	}
	if _, ok := entries[1].(valueobjects.PullRequestReview); !ok {
		t.Fatalf("entries[1] = %#v, want PullRequestReview", entries[1])
	}
	if _, ok := entries[2].(valueobjects.InlineReviewComment); !ok {
		t.Fatalf("entries[2] = %#v, want InlineReviewComment", entries[2])
	}
	last, ok := entries[3].(valueobjects.IssueComment)
	if !ok || last.Body() != "Second comment." {
		t.Fatalf("entries[3] = %#v, want the second issue comment", entries[3])
	}
}

func TestBuildEntries_ClassifiesACommentedEventIntoAnIssueComment(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "commented_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.IssueComment)
	if !ok {
		t.Fatalf("entries[0] is not an IssueComment: %#v", entries[0])
	}

	attribution, err := valueobjects.NewAttribution(
		"connect0459",
		time.Date(2026, 6, 17, 9, 47, 30, 0, time.UTC),
		"https://github.com/connect0459/starlark-mbt/issues/218#issuecomment-4728347671",
	)
	if err != nil {
		t.Fatalf("unexpected error building expected attribution: %v", err)
	}
	want := valueobjects.NewIssueComment(attribution, "The bug has been fixed in PR#3690 of [moonbitlang/core](https://github.com/moonbitlang/core). Accordingly, the workaround applied in this repository will be removed.\n\n**Workaround removal schedule**\n\nPR#3690 was merged on 2026-06-17, but the current moon toolchain (`0.1.20260608`, built on 2026-06-08) does not yet include the fix. The workaround in `internal/std_math/std_math.mbt` will be removed once a new moon release that bundles the corrected `@math.cosh` is available.")

	if !got.Equals(want) {
		t.Fatalf("entries[0] = %#v, want %#v", got, want)
	}
}

func TestBuildEntries_ClassifiesAReviewedEventIntoAPullRequestReview(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "reviewed_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.PullRequestReview)
	if !ok {
		t.Fatalf("entries[0] is not a PullRequestReview: %#v", entries[0])
	}

	attribution, err := valueobjects.NewAttribution(
		"copilot-pull-request-reviewer[bot]",
		time.Date(2026, 6, 19, 1, 53, 17, 0, time.UTC),
		"https://github.com/connect0459/starlark-mbt/pull/277#pullrequestreview-4529659600",
	)
	if err != nil {
		t.Fatalf("unexpected error building expected attribution: %v", err)
	}
	want := valueobjects.NewPullRequestReview(
		attribution,
		valueobjects.ReviewStateCommented,
		"## Pull request overview\n\nThis PR prepares the `v0.3.1` release by bumping the module version and adding the `0.3.1` release notes + compare links to `CHANGELOG.md` (no implementation changes).\n\n**Changes:**\n- Bump `moon.mod` version from `0.3.0` to `0.3.1`.\n- Add `0.3.1` section to `CHANGELOG.md` with release notes and update bottom compare links.\n\n### Reviewed changes\n\nCopilot reviewed 2 out of 2 changed files in this pull request and generated 1 comment.\n\n| File | Description |\n| ---- | ----------- |\n| moon.mod | Version bump to `0.3.1` for the release. |\n| CHANGELOG.md | Adds `0.3.1` release notes and updates compare/reference links. |\n\n\n\n\n\n\n---\n\n💡 <a href=\"/connect0459/starlark-mbt/new/main?filename=.github/instructions/*.instructions.md\" class=\"Link--inTextBlock\" target=\"_blank\" rel=\"noopener noreferrer\">Add Copilot custom instructions</a> for smarter, more guided reviews. <a href=\"https://docs.github.com/en/copilot/customizing-copilot/adding-repository-custom-instructions-for-github-copilot\" class=\"Link--inTextBlock\" target=\"_blank\" rel=\"noopener noreferrer\">Learn how to get started</a>.",
	)

	if !got.Equals(want) {
		t.Fatalf("entries[0] = %#v, want %#v", got, want)
	}
}

func TestBuildEntries_AttributesACommentedEventFromADeletedAccountToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"event": "commented",
		"user": null,
		"body": "This comment survives its author's account being deleted.",
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1#issuecomment-1"
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	comment, ok := entries[0].(valueobjects.IssueComment)
	if !ok {
		t.Fatalf("entries[0] is not an IssueComment: %#v", entries[0])
	}
	if comment.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", comment.Attribution().Author(), "ghost")
	}
}

func TestBuildEntries_SkipsAnUnparsableTimelineItemAndContinuesWithTheRest(t *testing.T) {
	rawTimeline := []json.RawMessage{
		json.RawMessage(`{not valid json`),
		loadTestdata(t, "commented_event.json"),
	}

	entries, skipped := services.BuildEntries(rawTimeline, nil, testIssueURL)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the still-valid one)", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if string(skipped[0].Raw) != `{not valid json` {
		t.Fatalf("skipped[0].Raw = %q, want the offending raw JSON", skipped[0].Raw)
	}
}

func TestBuildEntries_SkipsACommentedEventThatFailsToUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{"event": "commented", "created_at": 12345}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "unmarshal") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the unmarshal failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsAReviewedEventWithAnUnrecognizedState(t *testing.T) {
	raw := json.RawMessage(`{
		"event": "reviewed",
		"user": {"login": "octocat"},
		"body": "x",
		"state": "dismissed",
		"submitted_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/pull/1#pullrequestreview-1"
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func TestBuildEntries_SkipsAReviewedEventThatFailsToUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{"event": "reviewed", "submitted_at": 12345}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "unmarshal") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the unmarshal failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsADuplicateCommentedEventSharingAnAlreadySeenID(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 5001,
		"event": "commented",
		"user": {"login": "octocat"},
		"body": "Looks good.",
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1#issuecomment-5001"
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw, raw}, nil, testIssueURL)

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the duplicate should be skipped)", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func TestBuildEntries_LeavesAnUnrecognizedEventKindUnclassifiedWithoutFailing(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "review_requested_event.json")}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries for an unrecognized event, want 0", len(entries))
	}
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items for an unrecognized-but-benign event, want 0: %#v", len(skipped), skipped)
	}
}

func labelEventRaw(id int64, event, login, name, color string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"id": %d,
		"event": %q,
		"actor": {"login": %q},
		"created_at": "2026-07-01T10:00:00Z",
		"label": {"name": %q, "color": %q}
	}`, id, event, login, name, color))
}

func TestBuildEntries_ClassifiesALabeledEventIntoALabelEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "labeled_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.LabelEvent)
	if !ok {
		t.Fatalf("entries[0] is not a LabelEvent: %#v", entries[0])
	}

	attribution, err := valueobjects.NewAttribution(
		"tierninho",
		time.Date(2020, 2, 4, 17, 57, 45, 0, time.UTC),
		testIssueURL,
	)
	if err != nil {
		t.Fatalf("unexpected error building expected attribution: %v", err)
	}
	want, err := valueobjects.NewLabelEvent(attribution, valueobjects.LabelActionLabeled, "enhancement", "0dd8ac")
	if err != nil {
		t.Fatalf("unexpected error building expected label event: %v", err)
	}

	if !got.Equals(want) {
		t.Fatalf("entries[0] = %#v, want %#v", got, want)
	}
}

func TestBuildEntries_ClassifiesAnUnlabeledEventIntoALabelEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "unlabeled_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.LabelEvent)
	if !ok {
		t.Fatalf("entries[0] is not a LabelEvent: %#v", entries[0])
	}
	if got.Action() != valueobjects.LabelActionUnlabeled {
		t.Fatalf("Action() = %v, want %v", got.Action(), valueobjects.LabelActionUnlabeled)
	}
	if got.Name() != "epic: repo" {
		t.Fatalf("Name() = %q, want %q", got.Name(), "epic: repo")
	}
}

func TestBuildEntries_AttributesALabelEventToTheIssuesOwnURLNotAPerEventURL(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{labelEventRaw(1, "labeled", "octocat", "bug", "d73a4a")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.LabelEvent)
	if !ok {
		t.Fatalf("entries[0] is not a LabelEvent: %#v", entries[0])
	}
	if got.Attribution().URL().String() != testIssueURL {
		t.Fatalf("Attribution().URL() = %q, want the issue's own URL %q", got.Attribution().URL().String(), testIssueURL)
	}
}

func TestBuildEntries_PreservesChronologicalOrderOfLabelEventsAmongOtherTimelineItems(t *testing.T) {
	rawTimeline := []json.RawMessage{
		commentedEventRaw("alice", "First comment.", "https://github.com/example/repo/issues/1#issuecomment-1"),
		labelEventRaw(1, "labeled", "octocat", "bug", "d73a4a"),
		labelEventRaw(2, "unlabeled", "octocat", "wontfix", "ffffff"),
		commentedEventRaw("bob", "Second comment.", "https://github.com/example/repo/issues/1#issuecomment-2"),
	}

	entries, skipped := services.BuildEntries(rawTimeline, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(entries))
	}

	if _, ok := entries[0].(valueobjects.IssueComment); !ok {
		t.Fatalf("entries[0] = %#v, want IssueComment", entries[0])
	}
	labeled, ok := entries[1].(valueobjects.LabelEvent)
	if !ok || labeled.Action() != valueobjects.LabelActionLabeled {
		t.Fatalf("entries[1] = %#v, want a labeled LabelEvent", entries[1])
	}
	unlabeled, ok := entries[2].(valueobjects.LabelEvent)
	if !ok || unlabeled.Action() != valueobjects.LabelActionUnlabeled {
		t.Fatalf("entries[2] = %#v, want an unlabeled LabelEvent", entries[2])
	}
	if _, ok := entries[3].(valueobjects.IssueComment); !ok {
		t.Fatalf("entries[3] = %#v, want IssueComment", entries[3])
	}
}

func TestBuildEntries_AttributesALabelEventFromADeletedActorToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 1,
		"event": "labeled",
		"actor": null,
		"created_at": "2026-07-01T00:00:00Z",
		"label": {"name": "bug", "color": "d73a4a"}
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.LabelEvent)
	if !ok {
		t.Fatalf("entries[0] is not a LabelEvent: %#v", entries[0])
	}
	if got.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", got.Attribution().Author(), "ghost")
	}
}

func TestBuildEntries_SkipsALabelEventThatFailsToUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{"event": "labeled", "created_at": 12345}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "unmarshal") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the unmarshal failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsALabelEventWithAnEmptyLabelName(t *testing.T) {
	raw := labelEventRaw(1, "labeled", "octocat", "", "d73a4a")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func TestBuildEntries_SkipsADuplicateLabeledEventSharingAnAlreadySeenID(t *testing.T) {
	raw := labelEventRaw(1, "labeled", "octocat", "bug", "d73a4a")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw, raw}, nil, testIssueURL)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the duplicate should be skipped)", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func closureEventRaw(id int64, event, login, stateReason string) json.RawMessage {
	reasonField := ""
	if stateReason != "" {
		reasonField = fmt.Sprintf(`, "state_reason": %q`, stateReason)
	}
	return json.RawMessage(fmt.Sprintf(`{
		"id": %d,
		"event": %q,
		"actor": {"login": %q},
		"created_at": "2026-07-01T10:00:00Z"%s
	}`, id, event, login, reasonField))
}

func TestBuildEntries_ClassifiesAClosedEventIntoAClosureEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "closed_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.ClosureEvent)
	if !ok {
		t.Fatalf("entries[0] is not a ClosureEvent: %#v", entries[0])
	}

	attribution, err := valueobjects.NewAttribution(
		"tierninho",
		time.Date(2020, 2, 4, 18, 0, 0, 0, time.UTC),
		testIssueURL,
	)
	if err != nil {
		t.Fatalf("unexpected error building expected attribution: %v", err)
	}
	want := valueobjects.NewClosureEvent(attribution, valueobjects.ClosureActionClosed, "completed")

	if !got.Equals(want) {
		t.Fatalf("entries[0] = %#v, want %#v", got, want)
	}
}

func TestBuildEntries_ClassifiesAReopenedEventIntoAClosureEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "reopened_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.ClosureEvent)
	if !ok {
		t.Fatalf("entries[0] is not a ClosureEvent: %#v", entries[0])
	}
	if got.Action() != valueobjects.ClosureActionReopened {
		t.Fatalf("Action() = %v, want %v", got.Action(), valueobjects.ClosureActionReopened)
	}
	if got.Reason() != "" {
		t.Fatalf("Reason() = %q, want empty for a reopened event", got.Reason())
	}
}

func TestBuildEntries_AttributesAClosureEventToTheIssuesOwnURLNotAPerEventURL(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{closureEventRaw(1, "closed", "octocat", "completed")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.ClosureEvent)
	if !ok {
		t.Fatalf("entries[0] is not a ClosureEvent: %#v", entries[0])
	}
	if got.Attribution().URL().String() != testIssueURL {
		t.Fatalf("Attribution().URL() = %q, want the issue's own URL %q", got.Attribution().URL().String(), testIssueURL)
	}
}

func TestBuildEntries_AttributesAClosureEventFromADeletedActorToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 1,
		"event": "closed",
		"actor": null,
		"created_at": "2026-07-01T00:00:00Z",
		"state_reason": "not_planned"
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.ClosureEvent)
	if !ok {
		t.Fatalf("entries[0] is not a ClosureEvent: %#v", entries[0])
	}
	if got.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", got.Attribution().Author(), "ghost")
	}
}

func TestBuildEntries_SkipsAClosureEventThatFailsToUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{"event": "closed", "created_at": 12345}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "unmarshal") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the unmarshal failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsADuplicateClosedEventSharingAnAlreadySeenID(t *testing.T) {
	raw := closureEventRaw(1, "closed", "octocat", "completed")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw, raw}, nil, testIssueURL)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the duplicate should be skipped)", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func renameEventRaw(id int64, login, from, to string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"id": %d,
		"event": "renamed",
		"actor": {"login": %q},
		"created_at": "2026-07-01T10:00:00Z",
		"rename": {"from": %q, "to": %q}
	}`, id, login, from, to))
}

func TestBuildEntries_ClassifiesARenamedEventIntoARenameEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "renamed_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.RenameEvent)
	if !ok {
		t.Fatalf("entries[0] is not a RenameEvent: %#v", entries[0])
	}

	attribution, err := valueobjects.NewAttribution(
		"tierninho",
		time.Date(2020, 2, 4, 18, 5, 0, 0, time.UTC),
		testIssueURL,
	)
	if err != nil {
		t.Fatalf("unexpected error building expected attribution: %v", err)
	}
	want, err := valueobjects.NewRenameEvent(attribution, "Old title", "New title")
	if err != nil {
		t.Fatalf("unexpected error building expected rename event: %v", err)
	}

	if !got.Equals(want) {
		t.Fatalf("entries[0] = %#v, want %#v", got, want)
	}
}

func TestBuildEntries_AttributesARenameEventToTheIssuesOwnURLNotAPerEventURL(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{renameEventRaw(1, "octocat", "Old title", "New title")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.RenameEvent)
	if !ok {
		t.Fatalf("entries[0] is not a RenameEvent: %#v", entries[0])
	}
	if got.Attribution().URL().String() != testIssueURL {
		t.Fatalf("Attribution().URL() = %q, want the issue's own URL %q", got.Attribution().URL().String(), testIssueURL)
	}
}

func TestBuildEntries_AttributesARenameEventFromADeletedActorToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 1,
		"event": "renamed",
		"actor": null,
		"created_at": "2026-07-01T00:00:00Z",
		"rename": {"from": "Old title", "to": "New title"}
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.RenameEvent)
	if !ok {
		t.Fatalf("entries[0] is not a RenameEvent: %#v", entries[0])
	}
	if got.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", got.Attribution().Author(), "ghost")
	}
}

func TestBuildEntries_SkipsARenamedEventThatFailsToUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{"event": "renamed", "created_at": 12345}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "unmarshal") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the unmarshal failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsARenamedEventWithAnEmptyToTitle(t *testing.T) {
	raw := renameEventRaw(1, "octocat", "Old title", "")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func TestBuildEntries_SkipsADuplicateRenamedEventSharingAnAlreadySeenID(t *testing.T) {
	raw := renameEventRaw(1, "octocat", "Old title", "New title")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw, raw}, nil, testIssueURL)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the duplicate should be skipped)", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func milestoneEventRaw(id int64, event, login, title string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"id": %d,
		"event": %q,
		"actor": {"login": %q},
		"created_at": "2026-07-01T10:00:00Z",
		"milestone": {"title": %q}
	}`, id, event, login, title))
}

func TestBuildEntries_ClassifiesAMilestonedEventIntoAMilestoneEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "milestoned_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.MilestoneEvent)
	if !ok {
		t.Fatalf("entries[0] is not a MilestoneEvent: %#v", entries[0])
	}

	attribution, err := valueobjects.NewAttribution(
		"tierninho",
		time.Date(2020, 2, 4, 18, 10, 0, 0, time.UTC),
		testIssueURL,
	)
	if err != nil {
		t.Fatalf("unexpected error building expected attribution: %v", err)
	}
	want, err := valueobjects.NewMilestoneEvent(attribution, valueobjects.MilestoneActionMilestoned, "v1.0")
	if err != nil {
		t.Fatalf("unexpected error building expected milestone event: %v", err)
	}

	if !got.Equals(want) {
		t.Fatalf("entries[0] = %#v, want %#v", got, want)
	}
}

func TestBuildEntries_ClassifiesADemilestonedEventIntoAMilestoneEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "demilestoned_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.MilestoneEvent)
	if !ok {
		t.Fatalf("entries[0] is not a MilestoneEvent: %#v", entries[0])
	}
	if got.Action() != valueobjects.MilestoneActionDemilestoned {
		t.Fatalf("Action() = %v, want %v", got.Action(), valueobjects.MilestoneActionDemilestoned)
	}
	if got.Title() != "v0.9" {
		t.Fatalf("Title() = %q, want %q", got.Title(), "v0.9")
	}
}

func TestBuildEntries_AttributesAMilestoneEventToTheIssuesOwnURLNotAPerEventURL(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{milestoneEventRaw(1, "milestoned", "octocat", "v1.0")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.MilestoneEvent)
	if !ok {
		t.Fatalf("entries[0] is not a MilestoneEvent: %#v", entries[0])
	}
	if got.Attribution().URL().String() != testIssueURL {
		t.Fatalf("Attribution().URL() = %q, want the issue's own URL %q", got.Attribution().URL().String(), testIssueURL)
	}
}

func TestBuildEntries_AttributesAMilestoneEventFromADeletedActorToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 1,
		"event": "milestoned",
		"actor": null,
		"created_at": "2026-07-01T00:00:00Z",
		"milestone": {"title": "v1.0"}
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.MilestoneEvent)
	if !ok {
		t.Fatalf("entries[0] is not a MilestoneEvent: %#v", entries[0])
	}
	if got.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", got.Attribution().Author(), "ghost")
	}
}

func TestBuildEntries_SkipsAMilestoneEventThatFailsToUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{"event": "milestoned", "created_at": 12345}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "unmarshal") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the unmarshal failure", skipped[0].Reason)
	}
}

func TestBuildEntries_SkipsAMilestoneEventWithAnEmptyTitle(t *testing.T) {
	raw := milestoneEventRaw(1, "milestoned", "octocat", "")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func TestBuildEntries_SkipsADuplicateMilestonedEventSharingAnAlreadySeenID(t *testing.T) {
	raw := milestoneEventRaw(1, "milestoned", "octocat", "v1.0")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw, raw}, nil, testIssueURL)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the duplicate should be skipped)", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func assignmentEventRaw(id int64, event, actorLogin, assigneeLogin string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"id": %d,
		"event": %q,
		"actor": {"login": %q},
		"assignee": {"login": %q},
		"created_at": "2026-07-01T10:00:00Z"
	}`, id, event, actorLogin, assigneeLogin))
}

func TestBuildEntries_ClassifiesAnAssignedEventIntoAnAssignmentEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "assigned_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.AssignmentEvent)
	if !ok {
		t.Fatalf("entries[0] is not an AssignmentEvent: %#v", entries[0])
	}

	attribution, err := valueobjects.NewAttribution(
		"tierninho",
		time.Date(2020, 2, 4, 18, 15, 0, 0, time.UTC),
		testIssueURL,
	)
	if err != nil {
		t.Fatalf("unexpected error building expected attribution: %v", err)
	}
	want, err := valueobjects.NewAssignmentEvent(attribution, valueobjects.AssignmentActionAssigned, "billygriffin")
	if err != nil {
		t.Fatalf("unexpected error building expected assignment event: %v", err)
	}

	if !got.Equals(want) {
		t.Fatalf("entries[0] = %#v, want %#v", got, want)
	}
}

func TestBuildEntries_ClassifiesAnUnassignedEventIntoAnAssignmentEvent(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{loadTestdata(t, "unassigned_event.json")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.AssignmentEvent)
	if !ok {
		t.Fatalf("entries[0] is not an AssignmentEvent: %#v", entries[0])
	}
	if got.Action() != valueobjects.AssignmentActionUnassigned {
		t.Fatalf("Action() = %v, want %v", got.Action(), valueobjects.AssignmentActionUnassigned)
	}
	if got.Assignee() != "tierninho" {
		t.Fatalf("Assignee() = %q, want %q", got.Assignee(), "tierninho")
	}
}

func TestBuildEntries_AttributesAnAssignmentEventToTheIssuesOwnURLNotAPerEventURL(t *testing.T) {
	entries, skipped := services.BuildEntries([]json.RawMessage{assignmentEventRaw(1, "assigned", "octocat", "hubot")}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.AssignmentEvent)
	if !ok {
		t.Fatalf("entries[0] is not an AssignmentEvent: %#v", entries[0])
	}
	if got.Attribution().URL().String() != testIssueURL {
		t.Fatalf("Attribution().URL() = %q, want the issue's own URL %q", got.Attribution().URL().String(), testIssueURL)
	}
}

func TestBuildEntries_AttributesAnAssignmentEventFromADeletedActorToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 1,
		"event": "assigned",
		"actor": null,
		"assignee": {"login": "hubot"},
		"created_at": "2026-07-01T00:00:00Z"
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.AssignmentEvent)
	if !ok {
		t.Fatalf("entries[0] is not an AssignmentEvent: %#v", entries[0])
	}
	if got.Attribution().Author() != "ghost" {
		t.Fatalf("Attribution().Author() = %q, want %q", got.Attribution().Author(), "ghost")
	}
}

func TestBuildEntries_SkipsAnAssignmentEventThatFailsToUnmarshal(t *testing.T) {
	raw := json.RawMessage(`{"event": "assigned", "created_at": 12345}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
	if !strings.Contains(skipped[0].Reason, "unmarshal") {
		t.Fatalf("skipped[0].Reason = %q, want it to mention the unmarshal failure", skipped[0].Reason)
	}
}

// TestBuildEntries_AttributesAnAssignmentEventFromADeletedAssigneeToGhost
// guards the same deleted-account case classifyLabelEvent/
// classifyClosureEvent/etc. already guard for their actor field: GitHub
// nulls out any user reference (not just the actor who performed an
// action) once that account is deleted, so the assignee itself must fall
// back to "ghost" the same way the acting actor already does, rather than
// being treated as malformed input.
func TestBuildEntries_AttributesAnAssignmentEventFromADeletedAssigneeToGhost(t *testing.T) {
	raw := json.RawMessage(`{
		"id": 1,
		"event": "assigned",
		"actor": {"login": "octocat"},
		"assignee": null,
		"created_at": "2026-07-01T00:00:00Z"
	}`)

	entries, skipped := services.BuildEntries([]json.RawMessage{raw}, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}

	got, ok := entries[0].(valueobjects.AssignmentEvent)
	if !ok {
		t.Fatalf("entries[0] is not an AssignmentEvent: %#v", entries[0])
	}
	if got.Assignee() != "ghost" {
		t.Fatalf("Assignee() = %q, want %q", got.Assignee(), "ghost")
	}
}

func TestBuildEntries_SkipsADuplicateAssignedEventSharingAnAlreadySeenID(t *testing.T) {
	raw := assignmentEventRaw(1, "assigned", "octocat", "hubot")

	entries, skipped := services.BuildEntries([]json.RawMessage{raw, raw}, nil, testIssueURL)
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1 (the duplicate should be skipped)", len(entries))
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skipped items, want 1", len(skipped))
	}
}

func TestBuildEntries_PreservesChronologicalOrderOfClosureEventsAmongOtherTimelineItems(t *testing.T) {
	rawTimeline := []json.RawMessage{
		commentedEventRaw("alice", "First comment.", "https://github.com/example/repo/issues/1#issuecomment-1"),
		closureEventRaw(1, "closed", "octocat", "completed"),
		closureEventRaw(2, "reopened", "octocat", ""),
		commentedEventRaw("bob", "Second comment.", "https://github.com/example/repo/issues/1#issuecomment-2"),
	}

	entries, skipped := services.BuildEntries(rawTimeline, nil, testIssueURL)
	if len(skipped) != 0 {
		t.Fatalf("got %d skipped items, want 0: %#v", len(skipped), skipped)
	}
	if len(entries) != 4 {
		t.Fatalf("got %d entries, want 4", len(entries))
	}

	if _, ok := entries[0].(valueobjects.IssueComment); !ok {
		t.Fatalf("entries[0] = %#v, want IssueComment", entries[0])
	}
	closed, ok := entries[1].(valueobjects.ClosureEvent)
	if !ok || closed.Action() != valueobjects.ClosureActionClosed {
		t.Fatalf("entries[1] = %#v, want a closed ClosureEvent", entries[1])
	}
	reopened, ok := entries[2].(valueobjects.ClosureEvent)
	if !ok || reopened.Action() != valueobjects.ClosureActionReopened {
		t.Fatalf("entries[2] = %#v, want a reopened ClosureEvent", entries[2])
	}
	if _, ok := entries[3].(valueobjects.IssueComment); !ok {
		t.Fatalf("entries[3] = %#v, want IssueComment", entries[3])
	}
}
