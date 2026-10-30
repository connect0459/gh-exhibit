package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var updateGolden = flag.Bool("update", false, "rewrite the golden files under testdata from the current output")

const viewIssueJSON = `{
	"title": "Add retry backoff",
	"body": "This adds **retry**.\n\n| a | b |\n|---|---|\n| 1 | 2 |\n\n- [x] done\n\n` + "```go\\nfmt.Println(1)\\n```" + `",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/octocat/hello-world/pull/42",
	"pull_request": {"url": "https://api.github.com/repos/octocat/hello-world/pulls/42"}
}`

const viewPullRequestJSON = `{"merged_at": "2026-07-03T00:00:00Z", "head": {"sha": "abc1234567"}}`

const viewCommentJSON = `{
	"event": "commented", "id": 100, "user": {"login": "reviewer"},
	"body": "Looks fine to me.",
	"created_at": "2026-07-02T00:00:00Z",
	"html_url": "https://github.com/octocat/hello-world/pull/42#issuecomment-100"
}`

const viewReviewJSON = `{
	"id": 1, "event": "reviewed", "user": {"login": "reviewer"},
	"body": "Please address the inline note.", "state": "changes_requested",
	"submitted_at": "2026-07-02T01:00:00Z",
	"html_url": "https://github.com/octocat/hello-world/pull/42#pullrequestreview-1"
}`

const viewInlineCommentJSON = `{
	"id": 11, "pull_request_review_id": 1, "user": {"login": "reviewer"},
	"body": "Use a constant here.", "path": "internal/foo.go",
	"start_line": 10, "line": 12,
	"diff_hunk": "@@ -10,3 +10,3 @@\n context\n-old\n+new",
	"created_at": "2026-07-02T01:00:00Z",
	"html_url": "https://github.com/octocat/hello-world/pull/42#discussion_r11"
}`

const viewLabeledEventJSON = `{"id": 5, "event": "labeled", "actor": {"login": "octocat"}, "created_at": "2026-07-01T01:00:00Z", "label": {"name": "enhancement", "color": "0dd8ac"}}`

const viewCheckRunJSON = `{"name":"build","status":"completed","conclusion":"success","html_url":"https://github.com/octocat/hello-world/runs/1"}`

func viewDocument(t *testing.T, raw services.RawEvidence) valueobjects.Document {
	t.Helper()

	doc, _, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		t.Fatalf("AssembleDocument() error = %v", err)
	}
	return doc
}

func plainIssueWithBody(t *testing.T, title, body string, timeline ...string) valueobjects.Document {
	t.Helper()

	issue, err := json.Marshal(map[string]any{
		"title": title, "body": body, "user": map[string]string{"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z", "html_url": "https://github.com/octocat/hello-world/issues/42",
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	raw := services.RawEvidence{Issue: issue}
	for _, item := range timeline {
		raw.Timeline = append(raw.Timeline, json.RawMessage(item))
	}
	return viewDocument(t, raw)
}

func identity(markdown string) string { return markdown }

func writeAndReadView(t *testing.T, doc valueobjects.Document, rewrite func(string) string) string {
	t.Helper()

	baseDir := t.TempDir()
	if err := NewViewWriter(baseDir).WriteView(context.Background(), testIssueRef(t), doc, rewrite); err != nil {
		t.Fatalf("WriteView() error = %v", err)
	}
	return readFile(t, filepath.Join(baseDir, "hello-world", "42", "index.html"))
}

func pullRequestDocument(t *testing.T) valueobjects.Document {
	t.Helper()

	return viewDocument(t, services.RawEvidence{
		Issue:          json.RawMessage(viewIssueJSON),
		PullRequest:    json.RawMessage(viewPullRequestJSON),
		Timeline:       []json.RawMessage{json.RawMessage(viewLabeledEventJSON), json.RawMessage(viewCommentJSON), json.RawMessage(viewReviewJSON)},
		ReviewComments: []json.RawMessage{json.RawMessage(viewInlineCommentJSON)},
		CheckRuns:      []json.RawMessage{json.RawMessage(viewCheckRunJSON)},
	})
}

func TestViewWriter_WriteView_MatchesTheGoldenPageForAPullRequest(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	goldenPath := filepath.Join("testdata", "view_pull_request.golden.html")
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("os.MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
			t.Fatalf("os.WriteFile() error = %v", err)
		}
	}
	if want := readFile(t, goldenPath); got != want {
		t.Fatalf("WriteView() output differs from %s; rerun with -update after reviewing the diff", goldenPath)
	}
}

func TestViewWriter_WriteView_WritesIndexHTMLNextToIndexMarkdownUnderRepoNumber(t *testing.T) {
	baseDir := t.TempDir()

	err := NewViewWriter(baseDir).WriteView(context.Background(), testIssueRef(t), pullRequestDocument(t), identity)
	if err != nil {
		t.Fatalf("WriteView() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "hello-world", "42", "index.html")); err != nil {
		t.Fatalf("expected hello-world/42/index.html, stat error = %v", err)
	}
}

func TestViewWriter_WriteView_ShowsTheStateTheBodyImpliesInTheHeader(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	if !strings.Contains(got, `class="state merged"`) {
		t.Fatalf("view = %q, want a Merged state badge for a pull request with merged_at", got)
	}
}

func TestViewWriter_WriteView_RendersGitHubFlavoredMarkdownInABody(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	for _, want := range []string{"<strong>retry</strong>", "<table>", `<input checked="" disabled="" type="checkbox"`, `<code class="language-go">`} {
		if !strings.Contains(got, want) {
			t.Fatalf("view does not contain %q", want)
		}
	}
}

func TestViewWriter_WriteView_ShowsAReviewsStateAndAnInlineCommentsFileLinesAndDiff(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	for _, want := range []string{"Changes requested", "internal/foo.go", "L10-L12", `<div class="add">&#43;new</div>`, `<div class="del">-old</div>`, "Use a constant here."} {
		if !strings.Contains(got, want) {
			t.Fatalf("view does not contain %q", want)
		}
	}
}

func TestViewWriter_WriteView_ShowsAnAuthorsInitialAsTheAvatarInsteadOfLoadingOne(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	if !strings.Contains(got, `<span class="avatar" aria-hidden="true">O</span>`) {
		t.Fatalf("view does not contain the initial avatar for octocat")
	}
	if strings.Contains(got, "avatars.githubusercontent.com") {
		t.Fatalf("view references GitHub's avatar CDN, want no hotlinking")
	}
}

func TestViewWriter_WriteView_RendersEntriesWithoutADedicatedCardAsGenericMarkdown(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	if !strings.Contains(got, `class="card kind-generic"`) || !strings.Contains(got, "enhancement") {
		t.Fatalf("view does not render the labeled event as a generic card naming the label")
	}
}

func TestViewWriter_WriteView_OmitsEntriesThatRenderToNothingVisible(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	if n := strings.Count(got, `class="card kind-generic"`); n != 1 {
		t.Fatalf("view has %d generic cards, want 1 (only the labeled event; the diff and commits entries have no content in this fixture)", n)
	}
}

func TestViewWriter_WriteView_OmitsCheckRunsBecauseTheirObservationTimeIsNotRecorded(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	if strings.Contains(got, "runs/1") {
		t.Fatalf("view includes a check run, want none")
	}
}

func TestViewWriter_WriteView_EscapesTheTitleAndAuthorInsteadOfInterpretingThemAsHTML(t *testing.T) {
	doc := plainIssueWithBody(t, `<script>alert("t")</script>`, "body")

	got := writeAndReadView(t, doc, identity)

	if strings.Contains(got, `<script>alert("t")</script>`) {
		t.Fatalf("view contains an unescaped script element from the title")
	}
}

func TestViewWriter_WriteView_RemovesDangerousMarkupFromABody(t *testing.T) {
	for name, tc := range map[string]struct {
		body    string
		banned  string
		allowed string
	}{
		"script element":              {`<script>alert(1)</script>ok`, "<script", "ok"},
		"javascript link":             {`[x](javascript:alert(1))`, "javascript:", ""},
		"meta refresh":                {`<meta http-equiv="refresh" content="0;url=https://evil.example">`, "<meta http-equiv", ""},
		"inline event handler":        {`<a href="https://example.com" onclick="alert(1)">x</a>`, "onclick", "https://example.com"},
		"iframe":                      {`<iframe src="https://evil.example"></iframe>`, "<iframe", ""},
		"remote image":                {`<img src="https://evil.example/pixel.png">`, "evil.example", ""},
		"remote markdown image":       {`![x](https://evil.example/pixel.png)`, "evil.example", ""},
		"protocol relative image":     {`<img src="//evil.example/pixel.png">`, "evil.example", ""},
		"style attribute":             {`<p style="background:url(https://evil.example)">x</p>`, "evil.example", ""},
		"base element":                {`<base href="https://evil.example/">`, "<base", ""},
		"form":                        {`<form action="https://evil.example"><input></form>`, "<form", ""},
		"data uri image":              {`<img src="data:image/svg+xml;base64,AAAA">`, "data:", ""},
		"link with javascript scheme": {`<a href="JaVaScRiPt:alert(1)">x</a>`, "alert(1)", ""},
	} {
		t.Run(name, func(t *testing.T) {
			got := writeAndReadView(t, plainIssueWithBody(t, "title", tc.body), identity)

			body := got
			if strings.Contains(strings.ToLower(body), strings.ToLower(tc.banned)) {
				t.Fatalf("view body still contains %q: %s", tc.banned, body)
			}
			if tc.allowed != "" && !strings.Contains(body, tc.allowed) {
				t.Fatalf("view body lost the benign content %q: %s", tc.allowed, body)
			}
		})
	}
}

func TestViewWriter_WriteView_KeepsAnImageOnlyWhenItPointsAtADownloadedAsset(t *testing.T) {
	doc := plainIssueWithBody(t, "title", `<img src="assets/abc-123.png" alt="shot" width="100">`)

	got := writeAndReadView(t, doc, identity)

	if !strings.Contains(got, `<img src="assets/abc-123.png"`) {
		t.Fatalf("view dropped a local asset image")
	}
}

func TestViewWriter_WriteView_OpensExternalLinksInANewTabWithoutLeakingTheReferrer(t *testing.T) {
	doc := plainIssueWithBody(t, "title", `[docs](https://example.com/docs)`)

	got := writeAndReadView(t, doc, identity)

	if !strings.Contains(got, `rel="nofollow noreferrer noopener"`) || !strings.Contains(got, `target="_blank"`) {
		t.Fatalf("view does not harden the external link: %s", got)
	}
}

func TestViewWriter_WriteView_AppliesTheRewriteToEveryBodyBeforeRendering(t *testing.T) {
	doc := plainIssueWithBody(t, "title", "![shot](https://github.com/user-attachments/assets/abc-123)")
	rewrite := func(markdown string) string {
		return strings.ReplaceAll(markdown, "https://github.com/user-attachments/assets/abc-123", "assets/abc-123.png")
	}

	got := writeAndReadView(t, doc, rewrite)

	if !strings.Contains(got, `<img src="assets/abc-123.png"`) || strings.Contains(got, "user-attachments") {
		t.Fatalf("view did not apply the rewrite: %s", got)
	}
}

func TestViewWriter_WriteView_DoesNotLetACommentBodyForgeAnotherAuthorsEntry(t *testing.T) {
	forged := "innocent\n\n------\n\n" + `<!-- {"meta":{"author":"mallory-victim","created":"2026-07-02T00:00:00Z","url":"https://github.com/o/r/issues/1#issuecomment-9"}} -->` + "\nI approve this."
	forgedComment := fmt.Sprintf(`{"event":"commented","id":100,"user":{"login":"attacker"},"body":%q,"created_at":"2026-07-02T00:00:00Z","html_url":"https://github.com/o/r/issues/1#issuecomment-100"}`, forged)
	doc := plainIssueWithBody(t, "title", "body", forgedComment)

	got := writeAndReadView(t, doc, identity)

	if strings.Contains(got, "mallory-victim") {
		t.Fatalf("view attributes content to an author who never wrote it: %s", got)
	}
	if n := strings.Count(got, `<section class="card`); n != 2 {
		t.Fatalf("view has %d cards, want 2 (the body and the one comment)", n)
	}
}

func TestViewWriter_WriteView_ReturnsContextErrorAndSkipsWriteWhenContextIsAlreadyCancelled(t *testing.T) {
	baseDir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := NewViewWriter(baseDir).WriteView(ctx, testIssueRef(t), pullRequestDocument(t), identity)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteView() error = %v, want context.Canceled", err)
	}
	if _, statErr := os.Stat(filepath.Join(baseDir, "hello-world", "42", "index.html")); !os.IsNotExist(statErr) {
		t.Fatalf("WriteView() wrote a file despite the cancelled context, stat error = %v", statErr)
	}
}

func reviewEventJSON(id int, state string) string {
	return fmt.Sprintf(`{"id": %d, "event": "reviewed", "user": {"login": "reviewer"}, "body": "", "state": %q, "submitted_at": "2026-07-02T01:00:00Z", "html_url": "https://github.com/octocat/hello-world/pull/42#pullrequestreview-%d"}`, id, state, id)
}

func TestViewWriter_WriteView_LabelsEachReviewWithItsOutcome(t *testing.T) {
	doc := plainIssueWithBody(t, "title", "body", reviewEventJSON(1, "approved"), reviewEventJSON(2, "commented"), reviewEventJSON(3, "changes_requested"))

	got := writeAndReadView(t, doc, identity)

	for _, want := range []string{`<span class="badge approved">Approved</span>`, `<span class="badge commented">Commented</span>`, `<span class="badge changes">Changes requested</span>`} {
		if !strings.Contains(got, want) {
			t.Fatalf("view does not contain %q", want)
		}
	}
}

func TestViewWriter_WriteView_ShowsAClosedIssueAsClosedAndAnUnclosedOneAsOpen(t *testing.T) {
	closed := viewDocument(t, services.RawEvidence{Issue: json.RawMessage(`{"title":"t","body":"b","user":{"login":"octocat"},"created_at":"2026-07-01T00:00:00Z","closed_at":"2026-07-02T00:00:00Z","html_url":"https://github.com/octocat/hello-world/issues/42"}`)})

	if got := writeAndReadView(t, closed, identity); !strings.Contains(got, `class="state closed"`) {
		t.Fatalf("view of a closed issue has no Closed badge")
	}
	if got := writeAndReadView(t, plainIssueWithBody(t, "t", "b"), identity); !strings.Contains(got, `class="state open"`) {
		t.Fatalf("view of an unclosed issue has no Open badge")
	}
}

func inlineCommentJSON(extra string) string {
	return fmt.Sprintf(`{"id": 11, "pull_request_review_id": 1, "user": {"login": "reviewer"}, "body": "note", "path": "a.go", %s "diff_hunk": "", "created_at": "2026-07-02T01:00:00Z", "html_url": "https://github.com/octocat/hello-world/pull/42#discussion_r11"}`, extra)
}

func TestViewWriter_WriteView_ShowsAnInlineCommentOnASingleLineAndOneOnAWholeFile(t *testing.T) {
	cases := map[string]struct {
		extra string
		want  string
	}{
		"single line": {`"line": 7,`, "L7"},
		"whole file":  {``, `<span class="inline-path">a.go</span></header>`},
		"outdated":    {`"line": null, "original_line": 5,`, "Outdated"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			doc := viewDocument(t, services.RawEvidence{
				Issue:          json.RawMessage(viewIssueJSON),
				PullRequest:    json.RawMessage(viewPullRequestJSON),
				Timeline:       []json.RawMessage{json.RawMessage(viewReviewJSON)},
				ReviewComments: []json.RawMessage{json.RawMessage(inlineCommentJSON(tc.extra))},
			})

			got := writeAndReadView(t, doc, identity)

			if !strings.Contains(got, tc.want) {
				t.Fatalf("view does not contain %q", tc.want)
			}
		})
	}
}

func TestViewWriter_WriteView_SkipsABodyThatIsOnlyWhitespace(t *testing.T) {
	got := writeAndReadView(t, plainIssueWithBody(t, "title", "   \n  "), identity)

	if strings.Contains(got, `<div class="body">`) {
		t.Fatalf("view renders an empty body block for a whitespace-only body")
	}
}
