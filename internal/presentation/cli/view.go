package cli

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strconv"

	"github.com/connect0459/gh-exhibit/internal/application/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// Viewer is the narrow port RunViews depends on to write a single issue/PR's
// view (satisfied structurally by *services.ViewService on the production
// path); defined here so tests can inject a fake instead of touching the
// filesystem.
type Viewer interface {
	View(ctx context.Context, ref valueobjects.IssueRef) (services.ViewResult, error)
}

// RunViews writes the view of every ref in numbers (owner/repo/outputDir
// held fixed) via viewer, reporting one line per ref to stdout on success
// or stderr on failure. A failing ref does not stop the remaining ones from
// being attempted, the same skip-and-continue precedent RunExports follows.
// Returns 0 if every ref succeeded, 1 if any failed. outputDir is only used
// to report the actual write location ({repo}/{number}/index.html) —
// RunViews itself never touches the filesystem; viewer does.
func RunViews(ctx context.Context, viewer Viewer, owner, repo, outputDir string, numbers []int, stdout, stderr io.Writer) int {
	exitCode := 0

	for _, number := range numbers {
		ref, err := valueobjects.NewIssueRef(owner, repo, number)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "failed #%d: %v\n", number, err)
			exitCode = 1
			continue
		}

		result, err := viewer.View(ctx, ref)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "failed #%d: %v\n", number, err)
			exitCode = 1
			continue
		}

		viewPath := filepath.Join(outputDir, repo, strconv.Itoa(number), "index.html")
		message := fmt.Sprintf("viewed #%d -> %s", number, viewPath)
		if skips := result.Skipped(); len(skips) > 0 {
			message += fmt.Sprintf(" (skipped %d entries)", len(skips))
		}
		_, _ = fmt.Fprintln(stdout, message)
		if result.AttachmentsUnrecorded() {
			_, _ = fmt.Fprintf(stderr, "note: #%d was exported before attachment links were recorded, so its attachments are shown as unavailable; re-run export to record them\n", number)
		}
	}

	return exitCode
}
