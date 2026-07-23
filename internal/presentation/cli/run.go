package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// Exporter is the narrow port RunExports depends on to export a single
// issue/PR (satisfied structurally by *services.ExportService on the
// production path); defined here so tests can inject a fake instead of
// exercising real network/filesystem I/O.
type Exporter interface {
	Export(ctx context.Context, ref valueobjects.IssueRef) (rendered []byte, skips []services.SkipNote, err error)
}

// RunExports exports every ref in numbers (owner/repo/outputDir held fixed)
// via exporter, reporting one line per ref to stdout on success or stderr
// on failure. A failing ref does not stop the remaining ones from being
// attempted (this project's existing skip-and-continue precedent). Returns
// 0 if every ref succeeded, 1 if any failed. outputDir is only used to
// report the actual write location ({repo}/{number}/index.md) —
// RunExports itself never touches the filesystem; exporter does. When
// withStdout is true, each successfully exported ref's rendered document
// is additionally printed to stdout, preceded by a "=== owner/repo#N ==="
// header line so multiple refs' documents can be told apart in the
// combined stream; the printed bytes are exactly what exporter wrote to
// disk, byte for byte. A ref that fails has no document printed, since
// exporter never produced one for it.
//
// When dryRun is true, exporter.Export is never called: each ref is only
// validated into a valueobjects.IssueRef and its would-be destination path
// is reported, entirely offline (unlike export-search's own --dry-run,
// export has no resolution step to preview — its numbers are already
// known, so the only meaningful preview is this local, no-I/O one). A ref
// that fails validation is still reported as a per-ref failure, the same
// as a real run's export failure. withStdout has no effect combined with
// dryRun, since no document is ever rendered.
func RunExports(ctx context.Context, exporter Exporter, owner, repo, outputDir string, numbers []int, dryRun, withStdout bool, stdout, stderr io.Writer) int {
	exitCode := 0

	for _, number := range numbers {
		ref, err := valueobjects.NewIssueRef(owner, repo, number)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "failed #%d: %v\n", number, err)
			exitCode = 1
			continue
		}

		documentPath := filepath.Join(outputDir, repo, strconv.Itoa(number), "index.md")

		if dryRun {
			_, _ = fmt.Fprintf(stdout, "would export #%d -> %s\n", number, documentPath)
			continue
		}

		rendered, skips, err := exporter.Export(ctx, ref)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "failed #%d: %v\n", number, err)
			exitCode = 1
			continue
		}

		message := fmt.Sprintf("exported #%d -> %s", number, documentPath)
		if len(skips) > 0 {
			message += fmt.Sprintf(" (skipped %d entries)", len(skips))
		}
		_, _ = fmt.Fprintln(stdout, message)

		if withStdout {
			_, _ = fmt.Fprintf(stdout, "=== %s/%s#%d ===\n", owner, repo, number)
			_, _ = stdout.Write(rendered)
			_, _ = fmt.Fprintln(stdout)
		}
	}

	return exitCode
}
