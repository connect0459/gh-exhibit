package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	appservices "github.com/connect0459/gh-exhibit/internal/application/services"
	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type fakeViewer struct {
	results       map[int]fakeViewResult
	calledNumbers []int
}

type fakeViewResult struct {
	skips      []services.SkipNote
	unrecorded bool
	err        error
}

func (f *fakeViewer) View(_ context.Context, ref valueobjects.IssueRef) (appservices.ViewResult, error) {
	f.calledNumbers = append(f.calledNumbers, ref.Number())

	result, ok := f.results[ref.Number()]
	if !ok {
		return appservices.ViewResult{}, fmt.Errorf("no fake result configured for #%d", ref.Number())
	}
	return appservices.NewViewResult(result.skips, result.unrecorded), result.err
}

func TestParseArgs_AcceptsTheViewSubcommandWithNumbersRepoAndOutput(t *testing.T) {
	got, err := ParseArgs([]string{"view", "12,13", "--repo", "octocat/hello-world", "-o", "exhibits"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if !got.View || !equalInts(got.Numbers, []int{12, 13}) || got.Repo != "octocat/hello-world" || got.OutputDir != "exhibits" {
		t.Errorf("ParseArgs() = %+v, want View with numbers [12 13], the repo and output directory", got)
	}
}

func TestParseArgs_DefaultsTheViewOutputDirectoryToTheCurrentDirectory(t *testing.T) {
	got, err := ParseArgs([]string{"view", "12"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.OutputDir != "." {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, ".")
	}
}

func TestParseArgs_RejectsTheViewSubcommandWithoutExactlyOneNumberArgument(t *testing.T) {
	for _, args := range [][]string{{"view"}, {"view", "1", "2"}} {
		if _, err := ParseArgs(args); err == nil {
			t.Errorf("ParseArgs(%v) error = nil, want an error", args)
		}
	}
}

func TestParseArgs_RejectsAFlagThatTheViewSubcommandDoesNotDefine(t *testing.T) {
	for _, flag := range []string{"--dry-run", "--with-stdout"} {
		if _, err := ParseArgs([]string{"view", "1", flag}); err == nil {
			t.Errorf("ParseArgs(view 1 %s) error = nil, want an error", flag)
		}
	}
}

func TestRunViews_ReportsEachViewedRefsIndexHTMLPath(t *testing.T) {
	viewer := &fakeViewer{results: map[int]fakeViewResult{1: {}}}
	var stdout, stderr bytes.Buffer

	got := RunViews(context.Background(), viewer, "octocat", "hello-world", "exhibits", []int{1}, &stdout, &stderr)

	want := "viewed #1 -> " + filepath.Join("exhibits", "hello-world", "1", "index.html")
	if got != 0 || !strings.Contains(stdout.String(), want) {
		t.Errorf("RunViews() = %d with stdout %q, want 0 and a line containing %q", got, stdout.String(), want)
	}
}

func TestRunViews_ContinuesWithTheRemainingRefsAndReturnsOneWhenAnyRefFails(t *testing.T) {
	viewer := &fakeViewer{results: map[int]fakeViewResult{
		1: {err: errors.New("boom")},
		2: {},
	}}
	var stdout, stderr bytes.Buffer

	got := RunViews(context.Background(), viewer, "octocat", "hello-world", ".", []int{1, 2}, &stdout, &stderr)

	if got != 1 {
		t.Errorf("RunViews() = %d, want 1", got)
	}
	if !equalInts(viewer.calledNumbers, []int{1, 2}) {
		t.Errorf("viewer called for %v, want [1 2] (a failing ref must not stop the rest)", viewer.calledNumbers)
	}
	if !strings.Contains(stderr.String(), "failed #1: boom") {
		t.Errorf("stderr = %q, want it to report the failure of #1", stderr.String())
	}
}

func TestRunViews_ReportsHowManyEntriesWereSkipped(t *testing.T) {
	viewer := &fakeViewer{results: map[int]fakeViewResult{1: {skips: []services.SkipNote{{Reason: "bad"}, {Reason: "worse"}}}}}
	var stdout, stderr bytes.Buffer

	RunViews(context.Background(), viewer, "octocat", "hello-world", ".", []int{1}, &stdout, &stderr)

	if !strings.Contains(stdout.String(), "(skipped 2 entries)") {
		t.Errorf("stdout = %q, want it to report the 2 skipped entries", stdout.String())
	}
}

func TestRunViews_ReportsARefThatFailsValidationWithoutCallingTheViewer(t *testing.T) {
	viewer := &fakeViewer{}
	var stdout, stderr bytes.Buffer

	got := RunViews(context.Background(), viewer, "octocat", "hello-world", ".", []int{0}, &stdout, &stderr)

	if got != 1 || len(viewer.calledNumbers) != 0 {
		t.Errorf("RunViews() = %d, viewer called for %v, want 1 and no calls", got, viewer.calledNumbers)
	}
}

func TestRunViews_TellsTheUserToReExportWhenAnExportPredatesAttachmentLinks(t *testing.T) {
	viewer := &fakeViewer{results: map[int]fakeViewResult{7: {unrecorded: true}}}
	var stdout, stderr bytes.Buffer

	got := RunViews(context.Background(), viewer, "octocat", "hello-world", ".", []int{7}, &stdout, &stderr)

	if got != 0 {
		t.Errorf("RunViews() = %d, want 0 (the view was still written)", got)
	}
	if !strings.Contains(stderr.String(), "note: #7") || !strings.Contains(stderr.String(), "re-run export") {
		t.Errorf("stderr = %q, want a note that #7's attachments are unavailable until export is re-run", stderr.String())
	}
}

func TestRunViews_SaysNothingAboutAttachmentLinksForAnExportThatRecordedThem(t *testing.T) {
	viewer := &fakeViewer{results: map[int]fakeViewResult{7: {}}}
	var stdout, stderr bytes.Buffer

	RunViews(context.Background(), viewer, "octocat", "hello-world", ".", []int{7}, &stdout, &stderr)

	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want nothing", stderr.String())
	}
}
