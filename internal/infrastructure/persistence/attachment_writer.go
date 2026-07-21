// Package persistence implements gh-exhibit's domain-layer repository
// ports (EvidenceWriter, DocumentWriter, AttachmentWriter, ProvenanceWriter)
// against the local filesystem. Every implementation type here is
// unexported, so callers depend only on the repositories interfaces
// (dependency inversion), never on these concrete types.
package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// attachmentWriter implements repositories.AttachmentWriter against the
// local filesystem.
type attachmentWriter struct {
	baseDir string
}

// NewAttachmentWriter builds a repositories.AttachmentWriter that persists
// fetched attachments and this run's failure log under baseDir, at
// {repo}/{number}/assets/{filename} and
// {repo}/{number}/evidence/fetch-errors.log respectively.
func NewAttachmentWriter(baseDir string) repositories.AttachmentWriter {
	return &attachmentWriter{baseDir: baseDir}
}

// WriteAsset implements repositories.AttachmentWriter.
func (w *attachmentWriter) WriteAsset(ctx context.Context, ref valueobjects.IssueRef, filename valueobjects.AssetFilename, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(filepath.Join(issueDir(w.baseDir, ref), "assets", filename.String()), data)
}

// WriteFetchErrorLog implements repositories.AttachmentWriter.
func (w *attachmentWriter) WriteFetchErrorLog(ctx context.Context, ref valueobjects.IssueRef, log []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path := filepath.Join(issueDir(w.baseDir, ref), "evidence", "fetch-errors.log")
	if len(log) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		return nil
	}
	return writeFile(path, log)
}

// issueDir builds the on-disk directory for ref's per-issue artifacts
// (owner is deliberately not part of the path), shared by every writer in
// this package: attachmentWriter uses it directly, evidenceWriter builds
// on it via evidencePath, and documentWriter joins it with index.md.
func issueDir(baseDir string, ref valueobjects.IssueRef) string {
	return filepath.Join(baseDir, ref.Repo(), strconv.Itoa(ref.Number()))
}
