// Package persistence implements gh-exhibit's domain-layer repository
// ports (EvidenceWriter, DocumentWriter, AttachmentWriter) against the
// local filesystem, per docs/specs/README.md's on-disk layout.
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
// local filesystem, per docs/specs/README.md's on-disk layout. Unexported
// so callers depend only on the repositories.AttachmentWriter interface,
// not this infrastructure-layer type.
type attachmentWriter struct {
	baseDir string
}

// NewAttachmentWriter builds a repositories.AttachmentWriter that persists
// fetched attachments and this run's failure log under baseDir, following
// docs/specs/README.md's issues/{repo}/{number}/assets/{filename} and
// issues/{repo}/{number}/fetch-errors.log layout.
func NewAttachmentWriter(baseDir string) repositories.AttachmentWriter {
	return &attachmentWriter{baseDir: baseDir}
}

// WriteAsset implements repositories.AttachmentWriter.
func (w *attachmentWriter) WriteAsset(ctx context.Context, ref valueobjects.IssueRef, filename string, data []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(filepath.Join(issueDir(w.baseDir, ref), "assets", filename), data)
}

// WriteFetchErrorLog persists log verbatim, except an empty log removes any
// existing fetch-errors.log instead of writing one: the evidence directory
// is a regenerable view, so a rerun where every attachment now resolves
// successfully must not leave a prior run's failure log behind.
func (w *attachmentWriter) WriteFetchErrorLog(ctx context.Context, ref valueobjects.IssueRef, log []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	path := filepath.Join(issueDir(w.baseDir, ref), "fetch-errors.log")
	if len(log) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove %s: %w", path, err)
		}
		return nil
	}
	return writeFile(path, log)
}

// issueDir builds the on-disk directory for ref's per-issue attachment
// artifacts (owner is deliberately not part of the path, matching
// issuePath's own precedent).
func issueDir(baseDir string, ref valueobjects.IssueRef) string {
	return filepath.Join(baseDir, "issues", ref.Repo(), strconv.Itoa(ref.Number()))
}
