package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// Exporter is the narrow port RunExports depends on to export a single
// issue/PR (satisfied structurally by *services.ExportService on the
// production path); defined here so tests can inject a fake instead of
// exercising real network/filesystem I/O.
type Exporter interface {
	Export(ctx context.Context, ref valueobjects.IssueRef) ([]services.SkipNote, error)
}

// RunExports exports every ref in numbers (owner/repo/outputDir held fixed)
// via exporter, reporting one line per ref to stdout on success or stderr
// on failure. A failing ref does not stop the remaining ones from being
// attempted (this project's existing skip-and-continue precedent). Returns
// 0 if every ref succeeded, 1 if any failed. outputDir is only used to
// report the actual write location (matching docs/SPEC.md's
// issues/{repo}/{number}.md layout) — RunExports itself never touches the
// filesystem; exporter does.
func RunExports(ctx context.Context, exporter Exporter, owner, repo, outputDir string, numbers []int, stdout, stderr io.Writer) int {
	exitCode := 0

	for _, number := range numbers {
		ref, err := valueobjects.NewIssueRef(owner, repo, number)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "failed #%d: %v\n", number, err)
			exitCode = 1
			continue
		}

		skips, err := exporter.Export(ctx, ref)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "failed #%d: %v\n", number, err)
			exitCode = 1
			continue
		}

		documentPath := filepath.Join(outputDir, "issues", repo, fmt.Sprintf("%d.md", number))
		message := fmt.Sprintf("exported #%d -> %s", number, documentPath)
		if len(skips) > 0 {
			message += fmt.Sprintf(" (skipped %d entries)", len(skips))
		}
		_, _ = fmt.Fprintln(stdout, message)
	}

	return exitCode
}
