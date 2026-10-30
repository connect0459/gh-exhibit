package persistence

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type viewWriter struct {
	baseDir string
}

// NewViewWriter builds a repositories.ViewWriter that persists each view as
// a self-contained index.html under baseDir, at {repo}/{number}/index.html,
// next to the rendered index.md.
func NewViewWriter(baseDir string) repositories.ViewWriter {
	return &viewWriter{baseDir: baseDir}
}

// WriteView implements repositories.ViewWriter.
func (w *viewWriter) WriteView(ctx context.Context, ref valueobjects.IssueRef, doc valueobjects.Document, rewrite func(markdown string) string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	rendered, err := renderView(ref, doc, rewrite)
	if err != nil {
		return fmt.Errorf("could not render the view: %w", err)
	}
	return writeFile(filepath.Join(issueDir(w.baseDir, ref), "index.html"), rendered)
}
