package valueobjects_test

import (
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func newAttribution(t *testing.T, author string, created time.Time, url string) valueobjects.Attribution {
	t.Helper()
	attribution, err := valueobjects.NewAttribution(author, created, url)
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}
	return attribution
}

func TestNewAttribution_RejectsEmptyAuthor(t *testing.T) {
	_, err := valueobjects.NewAttribution("", time.Now(), "https://github.com/example/repo/issues/1")

	if err == nil {
		t.Fatal("expected an error for an empty author, got nil")
	}
}

func TestNewAttribution_RejectsEmptyURL(t *testing.T) {
	_, err := valueobjects.NewAttribution("octocat", time.Now(), "")

	if err == nil {
		t.Fatal("expected an error for an empty url, got nil")
	}
}

func TestNewAttribution_RejectsANonAbsoluteURL(t *testing.T) {
	_, err := valueobjects.NewAttribution("octocat", time.Now(), "not-a-url")

	if err == nil {
		t.Fatal("expected an error for a non-absolute url, got nil")
	}
}

func TestAttribution_Equals_TreatsMatchingValuesAsEqual(t *testing.T) {
	created := time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC)
	a, err := valueobjects.NewAttribution("octocat", created, "https://github.com/example/repo/issues/1")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}
	b, err := valueobjects.NewAttribution("octocat", created, "https://github.com/example/repo/issues/1")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}

	if !a.Equals(b) {
		t.Fatal("expected attributions with matching author, created, and url to be equal")
	}
}

func TestAttribution_Equals_TreatsDifferentClockReadingsOfSameInstantAsEqual(t *testing.T) {
	wallClock := time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC)
	// Stripping through Unix()/time.Unix() discards the monotonic clock
	// reading that time.Now()-derived values carry, reproducing the case
	// where naive == would fail even though both values name the same
	// instant (documented behavior of the time package).
	monotonicStripped := time.Unix(wallClock.Unix(), 0).UTC()

	a, err := valueobjects.NewAttribution("octocat", wallClock, "https://github.com/example/repo/issues/1")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}
	b, err := valueobjects.NewAttribution("octocat", monotonicStripped, "https://github.com/example/repo/issues/1")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}

	if !a.Equals(b) {
		t.Fatal("expected attributions naming the same instant to be equal regardless of monotonic clock reading")
	}
}

func TestNewAttribution_RejectsAuthorContainingANonASCIICharacter(t *testing.T) {
	_, err := valueobjects.NewAttribution("octocät", time.Now(), "https://github.com/example/repo/issues/1")

	if err == nil {
		t.Fatal("expected an error for an author containing a non-ASCII character, got nil")
	}
}

func TestNewAttribution_RejectsAuthorContainingAUnicodeConfusableCharacter(t *testing.T) {
	// U+212A KELVIN SIGN case-folds to ASCII "k" under strings.EqualFold,
	// so admitting it into author would let two distinct real GitHub
	// logins collide in Attribution.Equals.
	_, err := valueobjects.NewAttribution("Kelvin", time.Now(), "https://github.com/example/repo/issues/1")

	if err == nil {
		t.Fatal("expected an error for an author containing a Unicode confusable character, got nil")
	}
}

func TestNewAttribution_AcceptsAuthorContainingBotAccountBrackets(t *testing.T) {
	// GitHub bot accounts (e.g. "dependabot[bot]", "github-actions[bot]")
	// carry a real, all-ASCII login shape that IssueRef's stricter
	// GitHub-username pattern would wrongly reject.
	_, err := valueobjects.NewAttribution("dependabot[bot]", time.Now(), "https://github.com/example/repo/issues/1")

	if err != nil {
		t.Fatalf("unexpected error for a bot account login: %v", err)
	}
}

func TestAttribution_Equals_TreatsDifferentlyCasedAuthorsAsEqual(t *testing.T) {
	created := time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC)
	a, err := valueobjects.NewAttribution("octocat", created, "https://github.com/example/repo/issues/1")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}
	b, err := valueobjects.NewAttribution("Octocat", created, "https://github.com/example/repo/issues/1")
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}

	if !a.Equals(b) {
		t.Fatal("expected attributions naming the same author to be equal regardless of letter case")
	}
}

func TestAttribution_Equals_TreatsDifferentCreatedTimesAsNotEqual(t *testing.T) {
	a, err := valueobjects.NewAttribution(
		"octocat",
		time.Date(2026, 7, 2, 14, 19, 40, 0, time.UTC),
		"https://github.com/example/repo/issues/1",
	)
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}
	b, err := valueobjects.NewAttribution(
		"octocat",
		time.Date(2026, 7, 3, 14, 19, 40, 0, time.UTC),
		"https://github.com/example/repo/issues/1",
	)
	if err != nil {
		t.Fatalf("unexpected error building attribution: %v", err)
	}

	if a.Equals(b) {
		t.Fatal("expected attributions with different created times to not be equal")
	}
}
