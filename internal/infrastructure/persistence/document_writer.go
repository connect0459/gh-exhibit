package persistence

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// documentWriter implements repositories.DocumentWriter against the local
// filesystem. Unexported so callers depend only on the
// repositories.DocumentWriter interface, not this infrastructure-layer
// type.
type documentWriter struct {
	baseDir string
}

// NewDocumentWriter builds a repositories.DocumentWriter that persists
// rendered Markdown under baseDir, at {repo}/{number}.md.
func NewDocumentWriter(baseDir string) repositories.DocumentWriter {
	return &documentWriter{baseDir: baseDir}
}

// WriteDocument implements repositories.DocumentWriter.
func (w *documentWriter) WriteDocument(ctx context.Context, ref valueobjects.IssueRef, rendered []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(issuePath(w.baseDir, ref, "md"), rendered)
}
