package cli

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/application/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// fakeSearcher is a hand-written in-memory fake for the Searcher port,
// mirroring run_test.go's own fakeExporter style (canned result, no mocking
// library).
type fakeSearcher struct {
	outcome services.SearchOutcome
	err     error

	calledOwner, calledRepo string
}

func (f *fakeSearcher) Search(_ context.Context, owner, repo string, _ valueobjects.SearchCriteria) (services.SearchOutcome, error) {
	f.calledOwner, f.calledRepo = owner, repo
	if f.err != nil {
		return services.SearchOutcome{}, f.err
	}
	return f.outcome, nil
}

func testCriteriaWithLimit(t *testing.T, limit int) valueobjects.SearchCriteria {
	t.Helper()
	criteria, err := valueobjects.NewSearchCriteria(nil, nil, nil, nil, nil, limit, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}
	return criteria
}

func TestRunSearchExport_ReturnsOneAndPrintsTheErrorWhenSearchFails(t *testing.T) {
	searcher := &fakeSearcher{err: errors.New("search boom")}
	exporter := &fakeExporter{}
	var stdout, stderr bytes.Buffer

	got := RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 100), false, false, &stdout, &stderr)

	if got != 1 {
		t.Errorf("RunSearchExport() = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "search boom") {
		t.Errorf("stderr = %q, want it to contain the search error", stderr.String())
	}
}

func TestRunSearchExport_DryRun_ReturnsZeroWithoutCallingExport(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{Numbers: []int{1, 2}, MatchedCount: 2}}
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}, 2: {}}}
	var stdout, stderr bytes.Buffer

	got := RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 100), true, false, &stdout, &stderr)

	if got != 0 {
		t.Errorf("RunSearchExport() = %d, want 0", got)
	}
	if len(exporter.calledNumbers) != 0 {
		t.Errorf("exporter.calledNumbers = %v, want empty in dry-run mode", exporter.calledNumbers)
	}
}

func TestRunSearchExport_DryRun_PrintsTheMatchedCountAndNumbers(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{Numbers: []int{1, 2}, MatchedCount: 2}}
	exporter := &fakeExporter{}
	var stdout, stderr bytes.Buffer

	RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 100), true, false, &stdout, &stderr)

	out := stdout.String()
	if !strings.Contains(out, "2") {
		t.Errorf("stdout = %q, want it to mention the matched count 2", out)
	}
	if !strings.Contains(out, "#1") || !strings.Contains(out, "#2") {
		t.Errorf("stdout = %q, want it to list #1 and #2", out)
	}
}

func TestRunSearchExport_DryRun_PrintsTheTrueMatchedCountEvenWhenTruncated(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{Numbers: []int{1}, MatchedCount: 5, ExceededLimit: true}}
	exporter := &fakeExporter{}
	var stdout, stderr bytes.Buffer

	RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 1), true, false, &stdout, &stderr)

	out := stdout.String()
	if !strings.Contains(out, "matched 5 issue/PR number(s)") {
		t.Errorf("stdout = %q, want the headline to report the true matched count 5, not the truncated list length 1", out)
	}
}

func TestRunSearchExport_NonDryRun_DelegatesTheResolvedNumbersToRunExports(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{Numbers: []int{1, 2}, MatchedCount: 2}}
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}, 2: {}}}
	var stdout, stderr bytes.Buffer

	got := RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 100), false, false, &stdout, &stderr)

	if got != 0 {
		t.Errorf("RunSearchExport() = %d, want 0", got)
	}
	want := []int{1, 2}
	if len(exporter.calledNumbers) != len(want) || exporter.calledNumbers[0] != want[0] || exporter.calledNumbers[1] != want[1] {
		t.Errorf("exporter.calledNumbers = %v, want %v", exporter.calledNumbers, want)
	}
}

func TestRunSearchExport_NonDryRun_ReturnsOneWhenAResolvedRefFails(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{Numbers: []int{1}, MatchedCount: 1}}
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {err: errors.New("export boom")}}}
	var stdout, stderr bytes.Buffer

	got := RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 100), false, false, &stdout, &stderr)

	if got != 1 {
		t.Errorf("RunSearchExport() = %d, want 1", got)
	}
}

func TestRunSearchExport_PrintsAWarningToStderrWhenExceededLimit(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{Numbers: []int{1}, MatchedCount: 5, ExceededLimit: true}}
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}}}
	var stdout, stderr bytes.Buffer

	RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 1), false, false, &stdout, &stderr)

	if stderr.Len() == 0 {
		t.Fatal("stderr is empty, want an explicit warning when ExceededLimit is true")
	}
}

func TestRunSearchExport_PrintsNoWarningWhenNotExceededLimit(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{Numbers: []int{1}, MatchedCount: 1}}
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}}}
	var stdout, stderr bytes.Buffer

	RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 100), false, false, &stdout, &stderr)

	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty when ExceededLimit is false", stderr.String())
	}
}

func TestRunSearchExport_PassesOwnerAndRepoToTheSearcher(t *testing.T) {
	searcher := &fakeSearcher{outcome: services.SearchOutcome{}}
	exporter := &fakeExporter{}
	var stdout, stderr bytes.Buffer

	RunSearchExport(context.Background(), searcher, exporter, "octocat", "hello-world", ".", testCriteriaWithLimit(t, 100), false, false, &stdout, &stderr)

	if searcher.calledOwner != "octocat" || searcher.calledRepo != "hello-world" {
		t.Errorf("searcher called with owner=%q repo=%q, want octocat/hello-world", searcher.calledOwner, searcher.calledRepo)
	}
}
