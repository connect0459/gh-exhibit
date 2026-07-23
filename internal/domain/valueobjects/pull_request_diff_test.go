package valueobjects_test

import (
	"strings"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

var _ valueobjects.Entry = valueobjects.PullRequestDiff{}

func newPullRequestDiffAttribution(t *testing.T) valueobjects.Attribution {
	t.Helper()
	return newAttribution(t, "octocat", time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC), "https://github.com/example/repo/pull/1")
}

func TestPullRequestDiff_Render_ListsEachFileAndItsDiffWhenNotTruncated(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 12, 3, "@@ -1,3 +1,3 @@\n-old\n+new"),
		mustNewChangedFile(t, "internal/bar.go", "", valueobjects.FileStatusAdded, 1, 0, "@@ -0,0 +1 @@\n+bar"),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 13, 3, false)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	want := "<!-- {\"meta\":{\"author\":\"octocat\",\"created\":\"2026-07-02T14:19:40Z\",\"files\":2,\"additions\":13,\"deletions\":3,\"url\":\"https://github.com/example/repo/pull/1\"}} -->\n" +
		"\n" +
		"- `internal/foo.go` (modified, +12/-3)\n" +
		"- `internal/bar.go` (added, +1/-0)\n" +
		"\n" +
		"**Diff: `internal/foo.go`**\n" +
		"\n" +
		"```diff\n" +
		"@@ -1,3 +1,3 @@\n" +
		"-old\n" +
		"+new\n" +
		"```\n" +
		"\n" +
		"**Diff: `internal/bar.go`**\n" +
		"\n" +
		"```diff\n" +
		"@@ -0,0 +1 @@\n" +
		"+bar\n" +
		"```\n"
	if buf.String() != want {
		t.Fatalf("Render() =\n%q\nwant\n%q", buf.String(), want)
	}
}

func TestPullRequestDiff_Render_OmitsDiffBlockForAFileWithNoPatch(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/huge.go", "", valueobjects.FileStatusModified, 5000, 4000, ""),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 9000, 4000, false)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	if strings.Contains(buf.String(), "**Diff:") {
		t.Fatalf("Render() should not include a diff block for a file with no patch, got:\n%s", buf.String())
	}
	if !strings.Contains(buf.String(), "internal/huge.go") {
		t.Fatalf("Render() should still list the file's name, got:\n%s", buf.String())
	}
}

func TestPullRequestDiff_Render_IncludesTruncatedInTheMetaLineWhenTruncated(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 5000, 4000, ""),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 9000, 4000, true)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	if !strings.Contains(buf.String(), `"truncated":true`) {
		t.Fatalf("Render() should report truncated:true in the meta line, got:\n%s", buf.String())
	}
}

func TestPullRequestDiff_Render_OmitsTruncatedFromTheMetaLineWhenNotTruncated(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "@@ -1 +1 @@"),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 1, 1, false)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	if strings.Contains(buf.String(), "truncated") {
		t.Fatalf("Render() should not mention truncated when false, got:\n%s", buf.String())
	}
}

func TestPullRequestDiff_Render_FencesADiffHeadingFilenameContainingABacktick(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "weird`file.go", "", valueobjects.FileStatusModified, 1, 1, "@@ -1 +1 @@\n-old\n+new"),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 1, 1, false)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	want := "**Diff: ``weird`file.go``**\n"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("Render() should keep the whole filename inside one unbroken code span in the diff heading, got:\n%s\nwant substring:\n%s", buf.String(), want)
	}
}

func TestPullRequestDiff_Render_FencesAFilenameContainingABacktick(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "weird`file.go", "", valueobjects.FileStatusModified, 1, 1, ""),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 1, 1, false)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	want := "- ``weird`file.go`` (modified, +1/-1)\n"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("Render() should keep the whole filename inside one unbroken code span, got:\n%s\nwant substring:\n%s", buf.String(), want)
	}
}

func TestPullRequestDiff_Render_FencesARenamedFilesFromAndToEachContainingABacktick(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "new`file.go", "old`file.go", valueobjects.FileStatusRenamed, 0, 0, ""),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 0, 0, false)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	want := "- ``old`file.go`` -> ``new`file.go`` (renamed, +0/-0)\n"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("Render() should keep each renamed filename inside its own unbroken code span, got:\n%s\nwant substring:\n%s", buf.String(), want)
	}
}

func TestPullRequestDiff_Render_ShowsRenamedFilesFromAndTo(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/new.go", "internal/old.go", valueobjects.FileStatusRenamed, 0, 0, ""),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 0, 0, false)

	var buf strings.Builder
	if err := diff.Render(&buf); err != nil {
		t.Fatalf("unexpected error rendering pull request diff: %v", err)
	}

	want := "- `internal/old.go` -> `internal/new.go` (renamed, +0/-0)\n"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("Render() should show the rename's from and to, got:\n%s\nwant substring:\n%s", buf.String(), want)
	}
}

func TestPullRequestDiff_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	attribution := newPullRequestDiffAttribution(t)
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch"),
	}
	diff := valueobjects.NewPullRequestDiff(attribution, files, 1, 1, true)

	if !diff.Attribution().Equals(attribution) {
		t.Fatalf("Attribution() = %#v, want %#v", diff.Attribution(), attribution)
	}
	if len(diff.Files()) != 1 || !diff.Files()[0].Equals(files[0]) {
		t.Fatalf("Files() = %#v, want %#v", diff.Files(), files)
	}
	if diff.Additions() != 1 {
		t.Fatalf("Additions() = %d, want %d", diff.Additions(), 1)
	}
	if diff.Deletions() != 1 {
		t.Fatalf("Deletions() = %d, want %d", diff.Deletions(), 1)
	}
	if !diff.Truncated() {
		t.Fatal("Truncated() = false, want true")
	}
}

func TestPullRequestDiff_Files_MutatingTheReturnedSliceDoesNotAffectTheDiff(t *testing.T) {
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch"),
	}
	diff := valueobjects.NewPullRequestDiff(newPullRequestDiffAttribution(t), files, 1, 1, false)

	returned := diff.Files()
	returned[0] = mustNewChangedFile(t, "internal/tampered.go", "", valueobjects.FileStatusAdded, 9, 9, "tampered")

	if diff.Files()[0].Filename() != "internal/foo.go" {
		t.Fatalf("mutating the returned slice affected the diff's own state: got %q", diff.Files()[0].Filename())
	}
}

func TestPullRequestDiff_Equals_TreatsMatchingFieldsAsEqual(t *testing.T) {
	attribution := newPullRequestDiffAttribution(t)
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch"),
	}
	a := valueobjects.NewPullRequestDiff(attribution, files, 1, 1, false)
	b := valueobjects.NewPullRequestDiff(attribution, files, 1, 1, false)

	if !a.Equals(b) {
		t.Fatal("expected pull request diffs with matching fields to be equal")
	}
}

func TestPullRequestDiff_Equals_TreatsDifferentFilesAsNotEqual(t *testing.T) {
	attribution := newPullRequestDiffAttribution(t)
	a := valueobjects.NewPullRequestDiff(attribution, []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch"),
	}, 1, 1, false)
	b := valueobjects.NewPullRequestDiff(attribution, []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/bar.go", "", valueobjects.FileStatusModified, 1, 1, "patch"),
	}, 1, 1, false)

	if a.Equals(b) {
		t.Fatal("expected pull request diffs with different files to not be equal")
	}
}

func TestPullRequestDiff_Equals_TreatsDifferentTruncatedAsNotEqual(t *testing.T) {
	attribution := newPullRequestDiffAttribution(t)
	files := []valueobjects.ChangedFile{
		mustNewChangedFile(t, "internal/foo.go", "", valueobjects.FileStatusModified, 1, 1, "patch"),
	}
	a := valueobjects.NewPullRequestDiff(attribution, files, 1, 1, false)
	b := valueobjects.NewPullRequestDiff(attribution, files, 1, 1, true)

	if a.Equals(b) {
		t.Fatal("expected pull request diffs with different truncated values to not be equal")
	}
}
