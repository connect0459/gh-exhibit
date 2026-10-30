package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html"
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

const viewPullRequestJSON = `{"merged_at": "2026-07-03T00:00:00Z", "head": {"sha": "abc1234567"}, "additions": 1, "deletions": 1}`

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
		PullRequestFiles: []json.RawMessage{
			changedFileJSON(t, "internal/foo.go", "", "modified", 1, 1, "@@ -10,3 +10,3 @@\n context\n-old\n+new"),
		},
		PullRequestCommits: []json.RawMessage{
			commitJSON(t, "abc1234567890", "octocat", "feat: add retry backoff\n\nRetries now wait longer each time."),
		},
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

func TestViewWriter_WriteView_RendersEntriesWithoutADedicatedViewAsGenericMarkdown(t *testing.T) {
	doc := viewDocument(t, services.RawEvidence{
		Issue:     json.RawMessage(`{"title":"Epic","body":"b","user":{"login":"octocat"},"created_at":"2026-07-01T00:00:00Z","html_url":"https://github.com/octocat/hello-world/issues/42"}`),
		SubIssues: []json.RawMessage{json.RawMessage(`{"number":65,"title":"Include issue/PR labels","state":"closed","html_url":"https://github.com/octocat/hello-world/issues/65"}`)},
	})

	got := writeAndReadView(t, doc, identity)

	if !strings.Contains(got, `class="card kind-generic"`) || !strings.Contains(got, "Include issue/PR labels") {
		t.Fatalf("view does not render the sub-issues as a generic card naming the sub-issue")
	}
}

func TestViewWriter_WriteView_UsesNoGenericCardForEntriesThatHaveADedicatedView(t *testing.T) {
	got := writeAndReadView(t, pullRequestDocument(t), identity)

	if n := strings.Count(got, `class="card kind-generic"`); n != 0 {
		t.Fatalf("view has %d generic cards, want 0 (the diff, commits, and events each have a dedicated view)", n)
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

func changedFileJSON(t *testing.T, filename, previous, status string, additions, deletions int, patch string) json.RawMessage {
	t.Helper()

	raw, err := json.Marshal(map[string]any{
		"filename": filename, "previous_filename": previous, "status": status,
		"additions": additions, "deletions": deletions, "patch": patch,
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

func pullRequestWithFiles(t *testing.T, additions, deletions int, files ...json.RawMessage) valueobjects.Document {
	t.Helper()

	return viewDocument(t, services.RawEvidence{
		Issue:            json.RawMessage(viewIssueJSON),
		PullRequest:      json.RawMessage(fmt.Sprintf(`{"merged_at": null, "head": {"sha": "abc1234567"}, "additions": %d, "deletions": %d}`, additions, deletions)),
		PullRequestFiles: files,
	})
}

func TestViewWriter_WriteView_ShowsAFilesChangedCardWithTheTotalsAndEachFilesName(t *testing.T) {
	doc := pullRequestWithFiles(t, 3, 1,
		changedFileJSON(t, "internal/foo.go", "", "modified", 2, 1, "@@ -1,2 +1,3 @@\n keep\n-old\n+new\n+newer"),
		changedFileJSON(t, "README.md", "", "modified", 1, 0, "@@ -1 +1,2 @@\n title\n+added"),
	)

	got := html.UnescapeString(writeAndReadView(t, doc, identity))

	for _, want := range []string{`class="card kind-files"`, "Files changed", "2 files", "+3", "−1", "internal/foo.go", "README.md"} {
		if !strings.Contains(got, want) {
			t.Fatalf("view lacks %q in its files changed card", want)
		}
	}
}

func TestViewWriter_WriteView_ColoursEachFilesDiffByAddedRemovedAndHunkLines(t *testing.T) {
	doc := pullRequestWithFiles(t, 1, 1, changedFileJSON(t, "a.go", "", "modified", 1, 1, "@@ -1 +1 @@\n-old\n+new"))

	got := html.UnescapeString(writeAndReadView(t, doc, identity))

	for _, want := range []string{`<div class="hunk">@@ -1 +1 @@</div>`, `<div class="del">-old</div>`, `<div class="add">+new</div>`} {
		if !strings.Contains(got, want) {
			t.Fatalf("view lacks %q in the file's diff", want)
		}
	}
}

func TestViewWriter_WriteView_ShowsBothNamesForARenamedFile(t *testing.T) {
	doc := pullRequestWithFiles(t, 0, 0, changedFileJSON(t, "new/name.go", "old/name.go", "renamed", 0, 0, ""))

	got := writeAndReadView(t, doc, identity)

	if !strings.Contains(got, "old/name.go → new/name.go") {
		t.Fatalf("view does not show a renamed file as old → new")
	}
}

func TestViewWriter_WriteView_ListsAFileWithoutADiffByNameOnly(t *testing.T) {
	doc := pullRequestWithFiles(t, 5, 0, changedFileJSON(t, "big.bin", "", "added", 5, 0, ""))

	got := writeAndReadView(t, doc, identity)

	if !strings.Contains(got, "big.bin") || strings.Contains(got, `class="diff"`) {
		t.Fatalf("view should list big.bin by name without a diff block")
	}
}

func TestViewWriter_WriteView_NotesThatDiffsWereOmittedForALargePullRequest(t *testing.T) {
	doc := pullRequestWithFiles(t, 1200, 0, changedFileJSON(t, "huge.go", "", "added", 1200, 0, "+x"))

	got := writeAndReadView(t, doc, identity)

	if !strings.Contains(got, "Diffs are omitted") {
		t.Fatalf("view does not note that diffs were omitted for a large pull request")
	}
}

func TestViewWriter_WriteView_OmitsTheFilesChangedCardWhenThereAreNoFiles(t *testing.T) {
	got := writeAndReadView(t, pullRequestWithFiles(t, 0, 0), identity)

	if strings.Contains(got, "kind-files") {
		t.Fatalf("view has a files changed card for a pull request without changed files")
	}
}

func TestViewWriter_WriteView_EscapesAChangedFilesNameAndPatchInsteadOfInterpretingThemAsHTML(t *testing.T) {
	doc := pullRequestWithFiles(t, 1, 0, changedFileJSON(t, "<script>alert(1)</script>.go", "", "added", 1, 0, "+<img src=x onerror=alert(2)>"))

	got := writeAndReadView(t, doc, identity)

	if strings.Contains(got, "<script>alert(1)") || strings.Contains(got, "<img src=x") {
		t.Fatalf("view interprets a changed file's name or patch as HTML")
	}
}

func commitJSON(t *testing.T, sha, author, message string) json.RawMessage {
	t.Helper()

	identity := map[string]string{"name": author, "date": "2026-07-01T09:30:00Z"}
	raw, err := json.Marshal(map[string]any{
		"sha":    sha,
		"commit": map[string]any{"author": identity, "committer": identity, "message": message},
	})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return raw
}

func pullRequestWithCommits(t *testing.T, commits ...json.RawMessage) valueobjects.Document {
	t.Helper()

	return viewDocument(t, services.RawEvidence{
		Issue:              json.RawMessage(viewIssueJSON),
		PullRequest:        json.RawMessage(viewPullRequestJSON),
		PullRequestCommits: commits,
	})
}

func TestViewWriter_WriteView_ShowsACommitsCardWithEachCommitsShortSHASubjectAndAuthor(t *testing.T) {
	doc := pullRequestWithCommits(t,
		commitJSON(t, "9f35e9c1234567890abcdef", "octocat", "ci: pin workflow actions"),
		commitJSON(t, "1111111222222233333334", "hubot", "fix: handle empty input"),
	)

	got := writeAndReadView(t, doc, identity)

	for _, want := range []string{`class="card kind-commits"`, "Commits", "2 commits", ">9f35e9c<", "ci: pin workflow actions", "octocat", "2026-07-01 09:30 UTC", ">1111111<", "hubot"} {
		if !strings.Contains(got, want) {
			t.Fatalf("view lacks %q in its commits card", want)
		}
	}
}

func TestViewWriter_WriteView_FoldsACommitMessageBodyBehindItsSubject(t *testing.T) {
	doc := pullRequestWithCommits(t,
		commitJSON(t, "9f35e9c1234567890abcdef", "octocat", "ci: pin workflow actions\n\nAlso drops redundant step names."),
		commitJSON(t, "1111111222222233333334", "hubot", "fix: handle empty input"),
	)

	got := writeAndReadView(t, doc, identity)

	if !strings.Contains(got, "<details><summary>ci: pin workflow actions</summary>") || !strings.Contains(got, "Also drops redundant step names.") {
		t.Fatalf("view does not fold the commit message body behind its subject")
	}
	if n := strings.Count(got, "<details>"); n != 1 {
		t.Fatalf("view has %d folded commit messages, want 1 (a subject-only commit has nothing to fold)", n)
	}
}

func TestViewWriter_WriteView_OmitsTheCommitsCardWhenThereAreNoCommits(t *testing.T) {
	got := writeAndReadView(t, pullRequestWithCommits(t), identity)

	if strings.Contains(got, "kind-commits") {
		t.Fatalf("view has a commits card for a pull request without commits")
	}
}

func TestViewWriter_WriteView_EscapesACommitsMessageAndAuthorInsteadOfInterpretingThemAsHTML(t *testing.T) {
	doc := pullRequestWithCommits(t, commitJSON(t, "9f35e9c1234567890abcdef", "<b>mallory</b>", "<script>alert(1)</script>\n\n<img src=x onerror=alert(2)>"))

	got := writeAndReadView(t, doc, identity)

	for _, bad := range []string{"<script>alert(1)", "<img src=x", "<b>mallory"} {
		if strings.Contains(got, bad) {
			t.Fatalf("view interprets a commit's message or author as HTML: found %q", bad)
		}
	}
}

func eventJSON(event, extra string) string {
	return fmt.Sprintf(`{"id": 9, "event": %q, "actor": {"login": "octocat"}, "created_at": "2026-07-01T02:00:00Z"%s}`, event, extra)
}

func TestViewWriter_WriteView_ShowsATimelineEventAsOneLineNamingTheActorAndWhen(t *testing.T) {
	doc := plainIssueWithBody(t, "t", "b", eventJSON("assigned", `, "assignee": {"login": "hubot"}`))

	got := writeAndReadView(t, doc, identity)

	for _, want := range []string{`<section class="event">`, "<strong>octocat</strong>", "assigned @hubot", "2026-07-01 02:00 UTC"} {
		if !strings.Contains(got, want) {
			t.Fatalf("view lacks %q in the assignment event row", want)
		}
	}
	if strings.Contains(got, "kind-generic") {
		t.Fatalf("view draws the event as a card, want a bare timeline row")
	}
}

func TestViewWriter_WriteView_DescribesEachKindOfTimelineEventInWords(t *testing.T) {
	tests := []struct {
		name  string
		event string
		want  string
	}{
		{"unassigned", eventJSON("unassigned", `, "assignee": {"login": "hubot"}`), "unassigned @hubot"},
		{"closed", eventJSON("closed", ""), "closed this"},
		{"closed with a reason", eventJSON("closed", `, "state_reason": "not_planned"`), "closed this (not_planned)"},
		{"reopened", eventJSON("reopened", ""), "reopened this"},
		{"labeled", eventJSON("labeled", `, "label": {"name": "bug", "color": "d73a4a"}`), "added the label bug"},
		{"unlabeled", eventJSON("unlabeled", `, "label": {"name": "bug", "color": "d73a4a"}`), "removed the label bug"},
		{"milestoned", eventJSON("milestoned", `, "milestone": {"title": "v1"}`), "added this to the milestone v1"},
		{"demilestoned", eventJSON("demilestoned", `, "milestone": {"title": "v1"}`), "removed this from the milestone v1"},
		{"renamed", eventJSON("renamed", `, "rename": {"from": "Old", "to": "New"}`), "renamed this from Old to New"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := writeAndReadView(t, plainIssueWithBody(t, "t", "b", tt.event), identity)

			if !strings.Contains(got, tt.want) {
				t.Fatalf("view lacks %q for a %s event", tt.want, tt.name)
			}
		})
	}
}

func TestViewWriter_WriteView_EscapesAnEventsLabelNameInsteadOfInterpretingItAsHTML(t *testing.T) {
	doc := plainIssueWithBody(t, "t", "b", eventJSON("labeled", `, "label": {"name": "<script>alert(1)</script>", "color": "d73a4a"}`))

	got := writeAndReadView(t, doc, identity)

	if strings.Contains(got, "<script>alert(1)") {
		t.Fatalf("view interprets an event's label name as HTML")
	}
}
