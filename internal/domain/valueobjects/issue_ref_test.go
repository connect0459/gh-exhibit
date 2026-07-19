package valueobjects_test

import (
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestNewIssueRef_RejectsEmptyOwner(t *testing.T) {
	_, err := valueobjects.NewIssueRef("", "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an empty owner, got nil")
	}
}

func TestNewIssueRef_RejectsEmptyRepo(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", "", 1)

	if err == nil {
		t.Fatal("expected an error for an empty repo, got nil")
	}
}

func TestNewIssueRef_RejectsNonPositiveNumber(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", "gh-exhibit", 0)

	if err == nil {
		t.Fatal("expected an error for a non-positive number, got nil")
	}
}

func TestNewIssueRef_AcceptsOwnerWithHyphen(t *testing.T) {
	_, err := valueobjects.NewIssueRef("octo-cat", "gh-exhibit", 1)

	if err != nil {
		t.Fatalf("unexpected error for a hyphenated owner: %v", err)
	}
}

func TestNewIssueRef_RejectsOwnerContainingSlash(t *testing.T) {
	_, err := valueobjects.NewIssueRef("owner/evil", "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an owner containing a slash, got nil")
	}
}

func TestNewIssueRef_RejectsOwnerContainingBackslash(t *testing.T) {
	_, err := valueobjects.NewIssueRef(`owner\evil`, "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an owner containing a backslash, got nil")
	}
}

func TestNewIssueRef_RejectsOwnerWithLeadingHyphen(t *testing.T) {
	_, err := valueobjects.NewIssueRef("-connect0459", "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an owner with a leading hyphen, got nil")
	}
}

func TestNewIssueRef_RejectsOwnerWithTrailingHyphen(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459-", "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an owner with a trailing hyphen, got nil")
	}
}

func TestNewIssueRef_RejectsOwnerWithConsecutiveHyphens(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect--0459", "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an owner with consecutive hyphens, got nil")
	}
}

func TestNewIssueRef_RejectsOwnerContainingUnderscore(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect_0459", "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an owner containing an underscore, got nil")
	}
}

func TestNewIssueRef_RejectsOwnerExceedingMaxLength(t *testing.T) {
	_, err := valueobjects.NewIssueRef(strings.Repeat("a", 40), "gh-exhibit", 1)

	if err == nil {
		t.Fatal("expected an error for an owner exceeding GitHub's 39-character limit, got nil")
	}
}

func TestNewIssueRef_AcceptsRepoWithPeriodsAndUnderscoresAndHyphens(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", "gh.exhibit_v2-beta", 1)

	if err != nil {
		t.Fatalf("unexpected error for a repo with periods, underscores, and hyphens: %v", err)
	}
}

func TestNewIssueRef_RejectsRepoContainingSlash(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", "evil/../../etc", 1)

	if err == nil {
		t.Fatal("expected an error for a repo containing a slash, got nil")
	}
}

func TestNewIssueRef_RejectsRepoContainingBackslash(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", `evil\..\..\etc`, 1)

	if err == nil {
		t.Fatal("expected an error for a repo containing a backslash, got nil")
	}
}

func TestNewIssueRef_RejectsRepoEqualToDotDot(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", "..", 1)

	if err == nil {
		t.Fatal("expected an error for a repo equal to \"..\", got nil")
	}
}

func TestNewIssueRef_RejectsRepoEqualToDot(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", ".", 1)

	if err == nil {
		t.Fatal("expected an error for a repo equal to \".\", got nil")
	}
}

func TestNewIssueRef_RejectsRepoContainingInvalidCharacter(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", "gh exhibit!", 1)

	if err == nil {
		t.Fatal("expected an error for a repo containing an invalid character, got nil")
	}
}

func TestNewIssueRef_RejectsRepoExceedingMaxLength(t *testing.T) {
	_, err := valueobjects.NewIssueRef("connect0459", strings.Repeat("a", 101), 1)

	if err == nil {
		t.Fatal("expected an error for a repo exceeding GitHub's 100-character limit, got nil")
	}
}

func TestIssueRef_Equals_TreatsMatchingValuesAsEqual(t *testing.T) {
	a, err := valueobjects.NewIssueRef("connect0459", "gh-exhibit", 1)
	if err != nil {
		t.Fatalf("unexpected error building issue ref: %v", err)
	}
	b, err := valueobjects.NewIssueRef("connect0459", "gh-exhibit", 1)
	if err != nil {
		t.Fatalf("unexpected error building issue ref: %v", err)
	}

	if !a.Equals(b) {
		t.Fatal("expected issue refs with matching owner, repo, and number to be equal")
	}
}

func TestIssueRef_Equals_TreatsDifferentlyCasedOwnerAndRepoAsEqual(t *testing.T) {
	a, err := valueobjects.NewIssueRef("connect0459", "gh-exhibit", 1)
	if err != nil {
		t.Fatalf("unexpected error building issue ref: %v", err)
	}
	b, err := valueobjects.NewIssueRef("Connect0459", "Gh-Exhibit", 1)
	if err != nil {
		t.Fatalf("unexpected error building issue ref: %v", err)
	}

	if !a.Equals(b) {
		t.Fatal("expected issue refs naming the same owner and repo to be equal regardless of letter case")
	}
}

func TestIssueRef_Equals_TreatsDifferentNumberAsNotEqual(t *testing.T) {
	a, err := valueobjects.NewIssueRef("connect0459", "gh-exhibit", 1)
	if err != nil {
		t.Fatalf("unexpected error building issue ref: %v", err)
	}
	b, err := valueobjects.NewIssueRef("connect0459", "gh-exhibit", 2)
	if err != nil {
		t.Fatalf("unexpected error building issue ref: %v", err)
	}

	if a.Equals(b) {
		t.Fatal("expected issue refs with different numbers to not be equal")
	}
}
