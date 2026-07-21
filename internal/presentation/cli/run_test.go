package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// fakeExporter is a hand-written in-memory fake for the Exporter port,
// mirroring internal/application/services/export_service_test.go's existing
// fake-port style (canned results keyed by ref number, no mocking library).
type fakeExporter struct {
	results       map[int]fakeExportResult
	calledNumbers []int
}

type fakeExportResult struct {
	skips []services.SkipNote
	err   error
}

func (f *fakeExporter) Export(_ context.Context, ref valueobjects.IssueRef) ([]services.SkipNote, error) {
	f.calledNumbers = append(f.calledNumbers, ref.Number())

	result, ok := f.results[ref.Number()]
	if !ok {
		return nil, fmt.Errorf("no fake result configured for #%d", ref.Number())
	}
	return result.skips, result.err
}

func TestRunExports_ReturnsZeroWhenEveryRefSucceeds(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {}, 2: {},
	}}
	var stdout, stderr bytes.Buffer

	got := RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2}, &stdout, &stderr)

	if got != 0 {
		t.Errorf("RunExports() = %d, want 0", got)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunExports_ReturnsOneWhenAnyRefFails(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {},
		2: {err: errors.New("boom")},
	}}
	var stdout, stderr bytes.Buffer

	got := RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2}, &stdout, &stderr)

	if got != 1 {
		t.Errorf("RunExports() = %d, want 1", got)
	}
}

func TestRunExports_ContinuesToTheRemainingRefsAfterAFailure(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {err: errors.New("boom")},
		2: {},
		3: {},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2, 3}, &stdout, &stderr)

	want := []int{1, 2, 3}
	if !equalInts(exporter.calledNumbers, want) {
		t.Errorf("calledNumbers = %v, want %v", exporter.calledNumbers, want)
	}
}

func TestRunExports_ReportsAnInvalidOwnerWithoutCallingExport(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}}}
	var stdout, stderr bytes.Buffer

	got := RunExports(context.Background(), exporter, "connect_0459", "hello-world", ".", []int{1}, &stdout, &stderr)

	if got != 1 {
		t.Errorf("RunExports() = %d, want 1", got)
	}
	if len(exporter.calledNumbers) != 0 {
		t.Errorf("Export was called %v times, want 0 (an invalid owner must be rejected before exporting)", exporter.calledNumbers)
	}
	if !strings.Contains(stderr.String(), "1") {
		t.Errorf("stderr = %q, want it to mention the failing ref number", stderr.String())
	}
}

func TestRunExports_PrintsAFailureLineToStderr(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {err: errors.New("boom")},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1}, &stdout, &stderr)

	if !strings.Contains(stderr.String(), "1") || !strings.Contains(stderr.String(), "boom") {
		t.Errorf("stderr = %q, want it to mention the failing ref number and the underlying error", stderr.String())
	}
}

func TestRunExports_PrintsASuccessLineToStdout(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		42: {},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{42}, &stdout, &stderr)

	if !strings.Contains(stdout.String(), "42") {
		t.Errorf("stdout = %q, want it to mention the exported ref number", stdout.String())
	}
}

func TestRunExports_ReflectsTheOutputDirInTheSuccessMessage(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		42: {},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", "/tmp/gh-exhibit-out", []int{42}, &stdout, &stderr)

	want := filepath.Join("/tmp/gh-exhibit-out", "hello-world", "42", "index.md")
	if !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout = %q, want it to mention the actual write path %q", stdout.String(), want)
	}
}

func TestRunExports_ReportsTheSkipNoteCountForARef(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		42: {skips: []services.SkipNote{{Reason: "unrecognized event"}, {Reason: "malformed comment"}}},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{42}, &stdout, &stderr)

	if !strings.Contains(stdout.String(), "2") {
		t.Errorf("stdout = %q, want it to mention the skip note count (2)", stdout.String())
	}
}
