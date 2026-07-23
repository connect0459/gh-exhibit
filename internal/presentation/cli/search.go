package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/connect0459/gh-exhibit/internal/application/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// Searcher is the narrow port RunSearchExport depends on to resolve a
// SearchCriteria into matching issue/PR numbers (satisfied structurally by
// *services.SearchService on the production path); defined here so tests
// inject a fake instead of exercising real network I/O — the same seam
// Exporter (run.go) establishes for ExportService.
type Searcher interface {
	Search(ctx context.Context, owner, repo string, criteria valueobjects.SearchCriteria) (services.SearchOutcome, error)
}

// RunSearchExport resolves criteria via searcher — filter mode's
// counterpart to RunExports' explicit-number mode — then either reports
// the match (dry-run) or hands the resolved numbers straight into the
// existing RunExports, unchanged. Returns the same exit-code convention
// RunExports uses: 0 when the search (and, unless dryRun, every resolved
// ref) succeeded, 1 otherwise.
func RunSearchExport(ctx context.Context, searcher Searcher, exporter Exporter, owner, repo, outputDir string, criteria valueobjects.SearchCriteria, dryRun, withStdout bool, stdout, stderr io.Writer) int {
	outcome, err := searcher.Search(ctx, owner, repo, criteria)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	if outcome.ExceededLimit {
		_, _ = fmt.Fprintf(stderr, "warning: found %d matching issue/PR number(s) or more, but only %d will be used (gh-exhibit's own --limit is %d); narrow the filter or raise --limit to see more\n", outcome.MatchedCount, len(outcome.Numbers), criteria.Limit())
	}

	if dryRun {
		_, _ = fmt.Fprintf(stdout, "matched %d issue/PR number(s):\n", outcome.MatchedCount)
		for _, number := range outcome.Numbers {
			_, _ = fmt.Fprintf(stdout, "  #%d\n", number)
		}
		return 0
	}

	return RunExports(ctx, exporter, owner, repo, outputDir, outcome.Numbers, withStdout, stdout, stderr)
}
