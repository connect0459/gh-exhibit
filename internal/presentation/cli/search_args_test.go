package cli

import (
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseArgs_NumberMode_LeavesCriteriaNil(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria != nil {
		t.Errorf("Criteria = %v, want nil in number mode", got.Criteria)
	}
}

func TestParseArgs_FilterMode_SelectedByAuthorFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria == nil {
		t.Fatal("Criteria = nil, want a non-nil SearchCriteria")
	}
	if want := []string{"octocat"}; !equalStrings(got.Criteria.Authors(), want) {
		t.Errorf("Authors() = %v, want %v", got.Criteria.Authors(), want)
	}
	if len(got.Numbers) != 0 {
		t.Errorf("Numbers = %v, want empty in filter mode", got.Numbers)
	}
}

func TestParseArgs_FilterMode_SelectedByDryRunAlone(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--dry-run"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria == nil {
		t.Fatal("Criteria = nil, want a non-nil SearchCriteria")
	}
	if !got.DryRun {
		t.Error("DryRun = false, want true")
	}
}

func TestParseArgs_FilterMode_ParsesACommaSeparatedAssigneeList(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--assignee", "octocat, monalisa"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []string{"octocat", "monalisa"}
	if !equalStrings(got.Criteria.Assignees(), want) {
		t.Errorf("Assignees() = %v, want %v", got.Criteria.Assignees(), want)
	}
}

func TestParseArgs_FilterMode_RejectsAnEmptyAuthorListEntry(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--author", "octocat,,monalisa"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty --author list entry")
	}
}

func TestParseArgs_FilterMode_RejectsAnEmptyAssigneeListEntry(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--assignee", "octocat,,monalisa"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty --assignee list entry")
	}
}

func TestParseArgs_FilterMode_RejectsAnEmptyKindListEntry(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--kind", "issue,,pr"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty --kind list entry")
	}
}

func TestParseArgs_FilterMode_DeduplicatesTheKindListInFirstSeenOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--kind", "issue,issue,pr"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	kinds := got.Criteria.Kinds()
	if len(kinds) != 2 || kinds[0] != valueobjects.IssueKindIssue || kinds[1] != valueobjects.IssueKindPullRequest {
		t.Errorf("Kinds() = %v, want [issue pr]", kinds)
	}
}

func TestParseArgs_FilterMode_RejectsAfterLaterThanBefore(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--after", "2024-06-01", "--before", "2024-01-01"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error when --after is later than --before")
	}
}

func TestParseArgs_FilterMode_DeduplicatesTheAuthorListInFirstSeenOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--author", "octocat,octocat,monalisa"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []string{"octocat", "monalisa"}
	if !equalStrings(got.Criteria.Authors(), want) {
		t.Errorf("Authors() = %v, want %v", got.Criteria.Authors(), want)
	}
}

func TestParseArgs_FilterMode_DefaultsKindToBothWhenOmitted(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got := got.Criteria.Kinds(); len(got) != 0 {
		t.Errorf("Kinds() = %v, want empty (both)", got)
	}
}

func TestParseArgs_FilterMode_ParsesTheKindList(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--kind", "pr"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	kinds := got.Criteria.Kinds()
	if len(kinds) != 1 || kinds[0] != valueobjects.IssueKindPullRequest {
		t.Errorf("Kinds() = %v, want [pr]", kinds)
	}
}

func TestParseArgs_FilterMode_RejectsAnInvalidKind(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--kind", "draft"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an invalid --kind value")
	}
}

func TestParseArgs_FilterMode_ParsesAfterAndBefore(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--after", "2024-01-01", "--before", "2024-06-01"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria.CreatedAfter() == nil || got.Criteria.CreatedAfter().Format("2006-01-02") != "2024-01-01" {
		t.Errorf("CreatedAfter() = %v, want 2024-01-01", got.Criteria.CreatedAfter())
	}
	if got.Criteria.CreatedBefore() == nil || got.Criteria.CreatedBefore().Format("2006-01-02") != "2024-06-01" {
		t.Errorf("CreatedBefore() = %v, want 2024-06-01", got.Criteria.CreatedBefore())
	}
}

func TestParseArgs_FilterMode_RejectsAMalformedDate(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--after", "01/01/2024"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a malformed date")
	}
}

func TestParseArgs_FilterMode_DefaultsLimitTo100(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria.Limit() != valueobjects.DefaultSearchLimit {
		t.Errorf("Limit() = %d, want %d", got.Criteria.Limit(), valueobjects.DefaultSearchLimit)
	}
}

func TestParseArgs_FilterMode_RejectsALimitAboveTheMax(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--limit", "101"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a limit above the max")
	}
}

func TestParseArgs_FilterMode_ReadsTheLimitFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--limit", "10"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria.Limit() != 10 {
		t.Errorf("Limit() = %d, want 10", got.Criteria.Limit())
	}
}

func TestParseArgs_FilterMode_DefaultsSortAndOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria.Sort() != valueobjects.SearchSortByCreated {
		t.Errorf("Sort() = %v, want created", got.Criteria.Sort())
	}
	if got.Criteria.Order() != valueobjects.SearchOrderDescending {
		t.Errorf("Order() = %v, want desc", got.Criteria.Order())
	}
}

func TestParseArgs_FilterMode_ReadsSortAndOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--sort", "comments", "--order", "asc"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria.Sort() != valueobjects.SearchSortByComments {
		t.Errorf("Sort() = %v, want comments", got.Criteria.Sort())
	}
	if got.Criteria.Order() != valueobjects.SearchOrderAscending {
		t.Errorf("Order() = %v, want asc", got.Criteria.Order())
	}
}

func TestParseArgs_FilterMode_RejectsAnInvalidSort(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--sort", "reactions"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an invalid --sort value")
	}
}

func TestParseArgs_FilterMode_RejectsAnInvalidOrder(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--order", "sideways"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an invalid --order value")
	}
}

func TestParseArgs_FilterMode_DefaultsDryRunToFalse(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.DryRun {
		t.Error("DryRun = true, want false")
	}
}

func TestParseArgs_RejectsCombiningAnExplicitNumberListWithAFilterFlag(t *testing.T) {
	_, err := ParseArgs([]string{"export", "123", "--author", "octocat"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error when combining a number list with a filter flag")
	}
}

func TestParseArgs_RejectsNeitherNumbersNorFilterFlags(t *testing.T) {
	_, err := ParseArgs([]string{"export"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error when neither a number list nor a filter flag is given")
	}
	if !strings.Contains(err.Error(), "--author") {
		t.Errorf("ParseArgs() error = %v, want it to mention filter flags as an alternative", err)
	}
}

func TestParseArgs_FilterMode_CarriesRepoOutputAndWithStdout(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--author", "octocat", "--repo", "octocat/hello-world", "--output", "/tmp/out", "--with-stdout"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Repo != "octocat/hello-world" {
		t.Errorf("Repo = %q, want %q", got.Repo, "octocat/hello-world")
	}
	if got.OutputDir != "/tmp/out" {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, "/tmp/out")
	}
	if !got.WithStdout {
		t.Error("WithStdout = false, want true")
	}
}
