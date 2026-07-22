package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func mustIssueRef(t *testing.T, owner, repo string, number int) valueobjects.IssueRef {
	t.Helper()

	ref, err := valueobjects.NewIssueRef(owner, repo, number)
	if err != nil {
		t.Fatalf("NewIssueRef(%q, %q, %d) error = %v", owner, repo, number, err)
	}
	return ref
}

func TestDetectIssueReferences_FindsABareSameRepoReference(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("See #42 for context")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 1 {
		t.Fatalf("DetectIssueReferences() = %d references, want 1", len(got))
	}
	want := mustIssueRef(t, "connect0459", "gh-exhibit", 42)
	if !got[0].Ref().Equals(want) {
		t.Fatalf("Ref() = %+v, want %+v", got[0].Ref(), want)
	}
}

func TestDetectIssueReferences_FindsACrossRepoReference(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("See other-owner/other-repo#42 for context")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 1 {
		t.Fatalf("DetectIssueReferences() = %d references, want 1", len(got))
	}
	want := mustIssueRef(t, "other-owner", "other-repo", 42)
	if !got[0].Ref().Equals(want) {
		t.Fatalf("Ref() = %+v, want %+v", got[0].Ref(), want)
	}
}

func TestDetectIssueReferences_SkipsAReferenceInsideAFencedCodeBlock(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("before\n```\nsee #42 in this diff\n```\nafter")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (reference inside a fenced code block)", got)
	}
}

func TestDetectIssueReferences_SkipsAReferenceInsideALongerFence(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("before\n````\nsee #42, and a nested ```#43``` fence marker\n````\nafter")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (both references sit inside the outer 4-backtick fence)", got)
	}
}

func TestDetectIssueReferences_SkipsAReferenceInsideAnInlineCodeSpan(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("see `#42` for the raw form")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (reference inside an inline code span)", got)
	}
}

func TestDetectIssueReferences_SkipsAReferenceAlreadyFormattedAsAMarkdownLink(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("see [#42](https://github.com/connect0459/gh-exhibit/issues/42) already linked")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (reference already formatted as a link)", got)
	}
}

func TestDetectIssueReferences_SkipsAReferenceInsideAnHTMLComment(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte(`<!-- {"meta":{"note":"#42"}} -->` + "\nbody text")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (reference inside an HTML comment)", got)
	}
}

func TestDetectIssueReferences_RejectsANumberGluedToTrailingWordCharacters(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("version #42abc is unrelated")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (#42abc is not a reference)", got)
	}
}

func TestDetectIssueReferences_RejectsAHashGluedToPrecedingWordCharacters(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("build#42 is unrelated")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (build#42 is not a bare reference)", got)
	}
}

func TestDetectIssueReferences_RejectsANonPositiveNumber(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("see #0 for context")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty (#0 is not a valid issue number)", got)
	}
}

func TestDetectIssueReferences_FindsEachOccurrenceOfARepeatedReference(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)
	markdown := []byte("#42 and again #42")

	got := services.DetectIssueReferences(markdown, current)

	if len(got) != 2 {
		t.Fatalf("DetectIssueReferences() = %d references, want 2 (one per occurrence, not deduplicated)", len(got))
	}
}

func TestDetectIssueReferences_ReturnsNoReferencesWhenNoneArePresent(t *testing.T) {
	current := mustIssueRef(t, "connect0459", "gh-exhibit", 1)

	got := services.DetectIssueReferences([]byte("just plain text"), current)

	if len(got) != 0 {
		t.Fatalf("DetectIssueReferences() = %v, want empty", got)
	}
}
