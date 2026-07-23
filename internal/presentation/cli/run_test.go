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
	rendered []byte
	skips    []services.SkipNote
	err      error
}

func (f *fakeExporter) Export(_ context.Context, ref valueobjects.IssueRef) ([]byte, []services.SkipNote, error) {
	f.calledNumbers = append(f.calledNumbers, ref.Number())

	result, ok := f.results[ref.Number()]
	if !ok {
		return nil, nil, fmt.Errorf("no fake result configured for #%d", ref.Number())
	}
	return result.rendered, result.skips, result.err
}

func TestRunExports_ReturnsZeroWhenEveryRefSucceeds(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {}, 2: {},
	}}
	var stdout, stderr bytes.Buffer

	got := RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2}, false, false, &stdout, &stderr)

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

	got := RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2}, false, false, &stdout, &stderr)

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

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2, 3}, false, false, &stdout, &stderr)

	want := []int{1, 2, 3}
	if !equalInts(exporter.calledNumbers, want) {
		t.Errorf("calledNumbers = %v, want %v", exporter.calledNumbers, want)
	}
}

func TestRunExports_ReportsAnInvalidOwnerWithoutCallingExport(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}}}
	var stdout, stderr bytes.Buffer

	got := RunExports(context.Background(), exporter, "connect_0459", "hello-world", ".", []int{1}, false, false, &stdout, &stderr)

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

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1}, false, false, &stdout, &stderr)

	if !strings.Contains(stderr.String(), "1") || !strings.Contains(stderr.String(), "boom") {
		t.Errorf("stderr = %q, want it to mention the failing ref number and the underlying error", stderr.String())
	}
}

func TestRunExports_PrintsASuccessLineToStdout(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		42: {},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{42}, false, false, &stdout, &stderr)

	if !strings.Contains(stdout.String(), "42") {
		t.Errorf("stdout = %q, want it to mention the exported ref number", stdout.String())
	}
}

func TestRunExports_ReflectsTheOutputDirInTheSuccessMessage(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		42: {},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", "/tmp/gh-exhibit-out", []int{42}, false, false, &stdout, &stderr)

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

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{42}, false, false, &stdout, &stderr)

	if !strings.Contains(stdout.String(), "2") {
		t.Errorf("stdout = %q, want it to mention the skip note count (2)", stdout.String())
	}
}

func TestRunExports_PrintsTheRenderedDocumentToStdoutWhenWithStdoutIsEnabled(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		42: {rendered: []byte("# Title\n\nBody")},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{42}, false, true, &stdout, &stderr)

	if !strings.Contains(stdout.String(), "# Title\n\nBody") {
		t.Errorf("stdout = %q, want it to contain the rendered document", stdout.String())
	}
}

func TestRunExports_DoesNotPrintTheRenderedDocumentWhenWithStdoutIsDisabled(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		42: {rendered: []byte("# Title\n\nBody")},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{42}, false, false, &stdout, &stderr)

	if strings.Contains(stdout.String(), "# Title\n\nBody") {
		t.Errorf("stdout = %q, want it to not contain the rendered document when --with-stdout was not given", stdout.String())
	}
}

func TestRunExports_PrintsAHeaderNamingEachRefBeforeItsDocument(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {rendered: []byte("doc one")},
		2: {rendered: []byte("doc two")},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2}, false, true, &stdout, &stderr)

	out := stdout.String()
	header1 := strings.Index(out, "=== octocat/hello-world#1 ===")
	doc1 := strings.Index(out, "doc one")
	header2 := strings.Index(out, "=== octocat/hello-world#2 ===")
	doc2 := strings.Index(out, "doc two")
	if header1 == -1 || doc1 == -1 || header2 == -1 || doc2 == -1 {
		t.Fatalf("stdout = %q, want a header and document for both refs", out)
	}
	if header1 >= doc1 || doc1 >= header2 || header2 >= doc2 {
		t.Errorf("stdout = %q, want each ref's header immediately before its own document, in ref order", out)
	}
}

func TestRunExports_PrintsNoDocumentForARefThatFails(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {err: errors.New("boom")},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1}, false, true, &stdout, &stderr)

	if strings.Contains(stdout.String(), "===") {
		t.Errorf("stdout = %q, want no document header for a ref that failed to export", stdout.String())
	}
}

func TestRunExports_DryRunDoesNotCallExport(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{
		1: {}, 2: {},
	}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2}, true, false, &stdout, &stderr)

	if len(exporter.calledNumbers) != 0 {
		t.Errorf("Export was called %v times, want 0 (--dry-run must never call Export)", exporter.calledNumbers)
	}
}

func TestRunExports_DryRunPrintsTheWouldBeDestinationPathForEachRef(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{42: {}}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", "/tmp/gh-exhibit-out", []int{42}, true, false, &stdout, &stderr)

	want := filepath.Join("/tmp/gh-exhibit-out", "hello-world", "42", "index.md")
	if !strings.Contains(stdout.String(), want) {
		t.Errorf("stdout = %q, want it to mention the would-be write path %q", stdout.String(), want)
	}
}

func TestRunExports_DryRunReturnsZeroWhenEveryRefIsValid(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}, 2: {}}}
	var stdout, stderr bytes.Buffer

	got := RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{1, 2}, true, false, &stdout, &stderr)

	if got != 0 {
		t.Errorf("RunExports() = %d, want 0", got)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunExports_DryRunReportsAnInvalidOwnerAsAFailureWithoutCallingExport(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{1: {}}}
	var stdout, stderr bytes.Buffer

	got := RunExports(context.Background(), exporter, "connect_0459", "hello-world", ".", []int{1}, true, false, &stdout, &stderr)

	if got != 1 {
		t.Errorf("RunExports() = %d, want 1", got)
	}
	if len(exporter.calledNumbers) != 0 {
		t.Errorf("Export was called %v times, want 0 (an invalid owner must be rejected before previewing)", exporter.calledNumbers)
	}
	if !strings.Contains(stderr.String(), "1") {
		t.Errorf("stderr = %q, want it to mention the failing ref number", stderr.String())
	}
}

func TestRunExports_DryRunDoesNotPrintASuccessOrDocumentLine(t *testing.T) {
	exporter := &fakeExporter{results: map[int]fakeExportResult{42: {rendered: []byte("# Title")}}}
	var stdout, stderr bytes.Buffer

	RunExports(context.Background(), exporter, "octocat", "hello-world", ".", []int{42}, true, true, &stdout, &stderr)

	if strings.Contains(stdout.String(), "exported") || strings.Contains(stdout.String(), "===") {
		t.Errorf("stdout = %q, want no exported-success line or document header during a dry run", stdout.String())
	}
}
