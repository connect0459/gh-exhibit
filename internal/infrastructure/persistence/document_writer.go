package persistence

import (
	"context"
	"path/filepath"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// documentWriter implements repositories.DocumentWriter against the local
// filesystem.
type documentWriter struct {
	baseDir string
}

// NewDocumentWriter builds a repositories.DocumentWriter that persists
// rendered Markdown under baseDir, at {repo}/{number}/index.md.
func NewDocumentWriter(baseDir string) repositories.DocumentWriter {
	return &documentWriter{baseDir: baseDir}
}

// WriteDocument implements repositories.DocumentWriter.
func (w *documentWriter) WriteDocument(ctx context.Context, ref valueobjects.IssueRef, rendered []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(filepath.Join(issueDir(w.baseDir, ref), "index.md"), rendered)
}
