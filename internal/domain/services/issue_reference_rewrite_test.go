package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func mustReferenceURL(t *testing.T, raw string) valueobjects.Url {
	t.Helper()

	url, err := valueobjects.NewUrl(raw)
	if err != nil {
		t.Fatalf("NewUrl(%q) error = %v", raw, err)
	}
	return url
}

func TestRewriteIssueReferences_SubstitutesAResolvedReferenceWithTitleBeforeTheLink(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("see #42 for context")
	refs := services.DetectIssueReferences(markdown, current)
	url := mustReferenceURL(t, "https://github.com/connect0459/gh-exhibit/issues/42")

	got := services.RewriteIssueReferences(markdown, []services.ResolvedIssueReference{
		services.Resolved(refs[0], "Fix the thing", url),
	})

	want := "see Fix the thing [#42](https://github.com/connect0459/gh-exhibit/issues/42) for context"
	if string(got) != want {
		t.Fatalf("RewriteIssueReferences() = %q, want %q", got, want)
	}
}

func TestRewriteIssueReferences_UsesTheOriginalMatchedTextAsTheLinkLabel(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("see other-owner/other-repo#42 for context")
	refs := services.DetectIssueReferences(markdown, current)
	url := mustReferenceURL(t, "https://github.com/other-owner/other-repo/issues/42")

	got := services.RewriteIssueReferences(markdown, []services.ResolvedIssueReference{
		services.Resolved(refs[0], "Fix the thing", url),
	})

	want := "see Fix the thing [other-owner/other-repo#42](https://github.com/other-owner/other-repo/issues/42) for context"
	if string(got) != want {
		t.Fatalf("RewriteIssueReferences() = %q, want %q", got, want)
	}
}

func TestRewriteIssueReferences_LeavesAnUnresolvedReferenceUnchanged(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("see #42 for context")
	refs := services.DetectIssueReferences(markdown, current)

	got := services.RewriteIssueReferences(markdown, []services.ResolvedIssueReference{
		services.Unresolved(refs[0]),
	})

	if string(got) != string(markdown) {
		t.Fatalf("RewriteIssueReferences() = %q, want unchanged %q", got, markdown)
	}
}

func TestRewriteIssueReferences_ReturnsMarkdownUnchangedWhenGivenNoResolutions(t *testing.T) {
	markdown := []byte("see #42 for context")

	got := services.RewriteIssueReferences(markdown, nil)

	if string(got) != string(markdown) {
		t.Fatalf("RewriteIssueReferences() = %q, want unchanged %q", got, markdown)
	}
}

func TestRewriteIssueReferences_HandlesMultipleReferencesInASinglePass(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("#1 then #2")
	refs := services.DetectIssueReferences(markdown, current)
	url1 := mustReferenceURL(t, "https://github.com/connect0459/gh-exhibit/issues/1")

	got := services.RewriteIssueReferences(markdown, []services.ResolvedIssueReference{
		services.Resolved(refs[0], "First", url1),
		services.Unresolved(refs[1]),
	})

	want := "First [#1](https://github.com/connect0459/gh-exhibit/issues/1) then #2"
	if string(got) != want {
		t.Fatalf("RewriteIssueReferences() = %q, want %q", got, want)
	}
}
