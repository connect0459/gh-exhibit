package persistence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// evidenceWriter implements repositories.EvidenceWriter against the local
// filesystem.
type evidenceWriter struct {
	baseDir string
}

// NewEvidenceWriter builds a repositories.EvidenceWriter that persists raw
// evidence under baseDir, at {repo}/{number}/evidence/... (owner is
// deliberately not part of the path).
func NewEvidenceWriter(baseDir string) repositories.EvidenceWriter {
	return &evidenceWriter{baseDir: baseDir}
}

// WriteIssue implements repositories.EvidenceWriter.
func (w *evidenceWriter) WriteIssue(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(evidencePath(w.baseDir, ref, "issue.json"), raw)
}

// WriteTimeline implements repositories.EvidenceWriter.
func (w *evidenceWriter) WriteTimeline(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	joined, err := joinRawArray(items)
	if err != nil {
		return fmt.Errorf("could not combine the timeline pages into one array for %s/%d: %w", ref.Repo(), ref.Number(), err)
	}
	return writeFile(evidencePath(w.baseDir, ref, "timeline.json"), joined)
}

// WritePullRequest implements repositories.EvidenceWriter.
func (w *evidenceWriter) WritePullRequest(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(evidencePath(w.baseDir, ref, "pull.json"), raw)
}

// WriteReviewComments implements repositories.EvidenceWriter.
func (w *evidenceWriter) WriteReviewComments(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	joined, err := joinRawArray(items)
	if err != nil {
		return fmt.Errorf("could not combine the review comment pages into one array for %s/%d: %w", ref.Repo(), ref.Number(), err)
	}
	return writeFile(evidencePath(w.baseDir, ref, "review-comments.json"), joined)
}

// evidencePath builds the on-disk path for one of ref's raw evidence files
// named filename, under {repo}/{number}/evidence/ (owner is deliberately
// not part of the path).
func evidencePath(baseDir string, ref valueobjects.IssueRef, filename string) string {
	return filepath.Join(issueDir(baseDir, ref), "evidence", filename)
}

// joinRawArray concatenates items into a JSON array by splicing their raw
// bytes directly, rather than json.Marshal-ing the slice: encoding/json
// compacts each json.RawMessage element (stripping insignificant
// whitespace), which would break the verbatim-evidence guarantee.
func joinRawArray(items []json.RawMessage) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, item := range items {
		if len(item) == 0 {
			return nil, fmt.Errorf("item %d is empty", i)
		}
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(item)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

// writeFile persists data to path atomically: it writes to a temporary
// file in the same directory, syncs it to stable storage, and renames it
// into place. A rename replaces the directory entry in a single step, so
// a crash at any point during the write leaves the old file (untouched,
// under its original name) or the new file (complete, under the
// temporary name) — never a truncated file where the old one used to be.
func writeFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("could not save data to %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not save data to %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("could not save data to %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("could not save data to %s: %w", path, err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("could not save data to %s: %w", path, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("could not save data to %s: %w", path, err)
	}
	return nil
}
