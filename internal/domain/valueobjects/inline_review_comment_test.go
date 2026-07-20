package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.InlineReviewComment{}

func newInlineCommentAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "Copilot", time.Date(2026, 7, 2, 14, 19, 39, 0, time.UTC), "https://github.com/example/repo/pull/1#discussion_r1")
}

func TestInlineReviewComment_Render_IncludesPathAndLineInTheMetaLine(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "This looks off.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"Copilot","created":"2026-07-02T14:19:39Z","path":"docs/example.md","line":195,"url":"https://github.com/example/repo/pull/1#discussion_r1"}} -->

This looks off.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

// TestInlineReviewComment_Render_EscapesAPathThatWouldOtherwiseCloseTheSurroundingHTMLComment
// guards the actual invariant the hidden meta-line comment's safety
// depends on: encoding/json's default HTML escaping of "<", ">", and "&"
// (not any constraint on which characters InlineContext's path may
// contain, since NewInlineContext only rejects an empty one — a git path
// may contain almost any byte). A path containing a literal "-->" must
// never reach the rendered output unescaped, or it would close the
// surrounding <!-- ... --> comment early.
func TestInlineReviewComment_Render_EscapesAPathThatWouldOtherwiseCloseTheSurroundingHTMLComment(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("src/foo-->bar.go", intPtr(1), "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "This looks off.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"Copilot\",\"created\":\"2026-07-02T14:19:39Z\",\"path\":\"src/foo--\\u003ebar.go\",\"line\":1,\"url\":\"https://github.com/example/repo/pull/1#discussion_r1\"}} -->\n\nThis looks off.\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q (the path's \"-->\" must be json-escaped, not left literal, or it would close the surrounding HTML comment early)", buf.String(), want)
	}
}

func TestInlineReviewComment_Render_LabelsTheDiffHunkSeparatelyFromTheCommentBody(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), "@@ -169,7 +191,7 @@ There is no signing", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "This looks off.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"Copilot\",\"created\":\"2026-07-02T14:19:39Z\",\"path\":\"docs/example.md\",\"line\":195,\"url\":\"https://github.com/example/repo/pull/1#discussion_r1\"}} -->\n" +
		"\n" +
		"This looks off.\n" +
		"\n" +
		"**Diff:**\n" +
		"\n" +
		"```diff\n" +
		"@@ -169,7 +191,7 @@ There is no signing\n" +
		"```\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestInlineReviewComment_Render_WidensTheDiffFenceWhenTheHunkContainsBackticks(t *testing.T) {
	hunk := "@@ -1,3 +1,3 @@ README.md\n```go\nfunc foo() {}\n```"
	ctx, err := valueobjects.NewInlineContext("README.md", intPtr(3), hunk, false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "This looks off.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"Copilot\",\"created\":\"2026-07-02T14:19:39Z\",\"path\":\"README.md\",\"line\":3,\"url\":\"https://github.com/example/repo/pull/1#discussion_r1\"}} -->\n" +
		"\n" +
		"This looks off.\n" +
		"\n" +
		"**Diff:**\n" +
		"\n" +
		"````diff\n" +
		hunk + "\n" +
		"````\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestInlineReviewComment_Render_CollapsesTrailingNewlinesInTheBodyToASingleOne(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "This looks off.\n\n\n")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"Copilot","created":"2026-07-02T14:19:39Z","path":"docs/example.md","line":195,"url":"https://github.com/example/repo/pull/1#discussion_r1"}} -->

This looks off.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestInlineReviewComment_Render_NormalizesCRLFLineEndingsInTheBody(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "Line one.\r\nLine two.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"Copilot","created":"2026-07-02T14:19:39Z","path":"docs/example.md","line":195,"url":"https://github.com/example/repo/pull/1#discussion_r1"}} -->

Line one.
Line two.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestInlineReviewComment_Render_MarksAnOutdatedContextInTheMetaLine(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", intPtr(346), "", true)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "This looks off.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"Copilot","created":"2026-07-02T14:19:39Z","path":"docs/example.md","line":346,"outdated":true,"url":"https://github.com/example/repo/pull/1#discussion_r1"}} -->

This looks off.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestInlineReviewComment_Render_OmitsTheLineKeyForAFileLevelComment(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", nil, "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctx, "This file needs a rewrite.")

	var buf strings.Builder
	if err := comment.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering inline review comment: %v", err)
	}

	want := `<!-- {"meta":{"author":"Copilot","created":"2026-07-02T14:19:39Z","path":"docs/example.md","url":"https://github.com/example/repo/pull/1#discussion_r1"}} -->

This file needs a rewrite.
`
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestInlineReviewComment_ExposesTheAttributionContextAndBodyItWasConstructedWith(t *testing.T) {
	attribution := newInlineCommentAttribution(t)
	ctx, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	comment := valueobjects.NewInlineReviewComment(attribution, ctx, "This looks off.")

	if !comment.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", comment.Attribution(), attribution)
	}
	if !comment.Context().Equals(ctx) {
		t.Fatalf("Context() = %#v, want %#v", comment.Context(), ctx)
	}
	if comment.Body() != "This looks off." {
		t.Fatalf("Body() = %q, want %q", comment.Body(), "This looks off.")
	}
}

func TestInlineReviewComment_Equals_TreatsDifferentContextsAsNotEqual(t *testing.T) {
	ctxA, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	ctxB, err := valueobjects.NewInlineContext("docs/example.md", intPtr(200), "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	a := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctxA, "This looks off.")
	b := valueobjects.NewInlineReviewComment(newInlineCommentAttribution(t), ctxB, "This looks off.")

	if a.Equals(b) {
		t.Fatal("expected inline review comments with different contexts to not be equal")
	}
}
