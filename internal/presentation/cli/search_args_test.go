package cli

import (
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestParseArgs_ExportSubcommand_LeavesCriteriaNil(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria != nil {
		t.Errorf("Criteria = %v, want nil for the export subcommand", got.Criteria)
	}
}

func TestParseArgs_ExportSubcommand_RejectsAFilterFlag(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--author", "octocat"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since export no longer defines filter flags")
	}
}

// A filter flag's value being omitted must not change which error export
// reports: it should still be rejected as unrecognized, not misreported as
// missing a value it was never defined to take in the first place.
func TestParseArgs_ExportSubcommand_RejectsAFilterFlagWithNoFollowingValueAsUnrecognized(t *testing.T) {
	_, err := ParseArgs([]string{"export", "123", "--author"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since export does not define --author")
	}
	if strings.Contains(err.Error(), "needs an argument") {
		t.Errorf("ParseArgs() error = %v, want an unrecognized-flag error, not a missing-argument error, since --author is not one of export's own flags", err)
	}
}

func TestParseArgs_ExportSubcommand_RejectsAFilterFlagImmediatelyFollowedByAnotherFlagAsUnrecognized(t *testing.T) {
	_, err := ParseArgs([]string{"export", "123", "--author", "--repo=x"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since export does not define --author")
	}
	if strings.Contains(err.Error(), "needs an argument") {
		t.Errorf("ParseArgs() error = %v, want an unrecognized-flag error, not a missing-argument error", err)
	}
}

func TestParseArgs_ExportSearchSubcommand_AcceptsNoFlagsAtAll(t *testing.T) {
	got, err := ParseArgs([]string{"export-search"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria == nil {
		t.Fatal("Criteria = nil, want a non-nil SearchCriteria matching everything up to the default limit")
	}
	if got.Criteria.Limit() != valueobjects.DefaultSearchLimit {
		t.Errorf("Limit() = %d, want %d", got.Criteria.Limit(), valueobjects.DefaultSearchLimit)
	}
}

func TestParseArgs_ExportSearchSubcommand_ReadsTheAuthorFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--author", "octocat"})
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
		t.Errorf("Numbers = %v, want empty for export-search", got.Numbers)
	}
}

func TestParseArgs_ExportSearchSubcommand_AcceptsDryRunAlone(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--dry-run"})
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

func TestParseArgs_ExportSearchSubcommand_ParsesACommaSeparatedAssigneeList(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--assignee", "octocat, monalisa"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []string{"octocat", "monalisa"}
	if !equalStrings(got.Criteria.Assignees(), want) {
		t.Errorf("Assignees() = %v, want %v", got.Criteria.Assignees(), want)
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnEmptyAuthorListEntry(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--author", "octocat,,monalisa"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty --author list entry")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnEmptyAssigneeListEntry(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--assignee", "octocat,,monalisa"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty --assignee list entry")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnEmptyKindListEntry(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--kind", "issue,,pr"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty --kind list entry")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnExplicitEmptyAuthorValue(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--author="})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an explicit empty --author value, not a silent fall-back to unfiltered")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnExplicitEmptyAssigneeValue(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--assignee="})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an explicit empty --assignee value, not a silent fall-back to unfiltered")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnExplicitEmptyKindValue(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--kind="})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an explicit empty --kind value, not a silent fall-back to both kinds")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnExplicitEmptyAfterValue(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--after="})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an explicit empty --after value, not a silent fall-back to unbounded")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnExplicitEmptyBeforeValue(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--before="})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an explicit empty --before value, not a silent fall-back to unbounded")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAllExplicitEmptyValuesGivenTogether(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--after=", "--before="})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for explicit empty --after and --before values")
	}
	if !strings.Contains(err.Error(), "after") || !strings.Contains(err.Error(), "before") {
		t.Errorf("ParseArgs() error = %v, want it to name both --after and --before, not just the last one visited", err)
	}
}

func TestParseArgs_ExportSearchSubcommand_DeduplicatesTheKindListInFirstSeenOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--kind", "issue,issue,pr"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	kinds := got.Criteria.Kinds()
	if len(kinds) != 2 || kinds[0] != valueobjects.IssueKindIssue || kinds[1] != valueobjects.IssueKindPullRequest {
		t.Errorf("Kinds() = %v, want [issue pr]", kinds)
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAfterLaterThanBefore(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--after", "2024-06-01", "--before", "2024-01-01"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error when --after is later than --before")
	}
}

func TestParseArgs_ExportSearchSubcommand_DeduplicatesTheAuthorListInFirstSeenOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--author", "octocat,octocat,monalisa"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []string{"octocat", "monalisa"}
	if !equalStrings(got.Criteria.Authors(), want) {
		t.Errorf("Authors() = %v, want %v", got.Criteria.Authors(), want)
	}
}

func TestParseArgs_ExportSearchSubcommand_DefaultsKindToBothWhenOmitted(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got := got.Criteria.Kinds(); len(got) != 0 {
		t.Errorf("Kinds() = %v, want empty (both)", got)
	}
}

func TestParseArgs_ExportSearchSubcommand_ParsesTheKindList(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--kind", "pr"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	kinds := got.Criteria.Kinds()
	if len(kinds) != 1 || kinds[0] != valueobjects.IssueKindPullRequest {
		t.Errorf("Kinds() = %v, want [pr]", kinds)
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnInvalidKind(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--kind", "draft"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an invalid --kind value")
	}
}

func TestParseArgs_ExportSearchSubcommand_ParsesAfterAndBefore(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--after", "2024-01-01", "--before", "2024-06-01"})
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

func TestParseArgs_ExportSearchSubcommand_RejectsAMalformedDate(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--after", "01/01/2024"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a malformed date")
	}
}

func TestParseArgs_ExportSearchSubcommand_DefaultsLimitTo100(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria.Limit() != valueobjects.DefaultSearchLimit {
		t.Errorf("Limit() = %d, want %d", got.Criteria.Limit(), valueobjects.DefaultSearchLimit)
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsALimitAboveTheMax(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--limit", "101"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a limit above the max")
	}
}

func TestParseArgs_ExportSearchSubcommand_ReadsTheLimitFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--limit", "10"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Criteria.Limit() != 10 {
		t.Errorf("Limit() = %d, want 10", got.Criteria.Limit())
	}
}

func TestParseArgs_ExportSearchSubcommand_DefaultsSortAndOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--author", "octocat"})
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

func TestParseArgs_ExportSearchSubcommand_ReadsSortAndOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--sort", "comments", "--order", "asc"})
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

func TestParseArgs_ExportSearchSubcommand_RejectsAnInvalidSort(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--sort", "reactions"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an invalid --sort value")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAnInvalidOrder(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "--order", "sideways"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an invalid --order value")
	}
}

func TestParseArgs_ExportSearchSubcommand_DefaultsDryRunToFalse(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--author", "octocat"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.DryRun {
		t.Error("DryRun = true, want false")
	}
}

func TestParseArgs_ExportSearchSubcommand_RejectsAPositionalArgument(t *testing.T) {
	_, err := ParseArgs([]string{"export-search", "123", "--author", "octocat"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since export-search takes no positional argument")
	}
}

func TestParseArgs_ExportSearchSubcommand_CarriesRepoOutputAndWithStdout(t *testing.T) {
	got, err := ParseArgs([]string{"export-search", "--author", "octocat", "--repo", "octocat/hello-world", "--output", "/tmp/out", "--with-stdout"})
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
