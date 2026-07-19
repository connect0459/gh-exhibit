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
// filesystem, per ADR-002's on-disk layout. Unexported so callers depend
// only on the repositories.EvidenceWriter interface, not this
// infrastructure-layer type.
type evidenceWriter struct {
	baseDir string
}

// NewEvidenceWriter builds a repositories.EvidenceWriter that persists raw
// evidence under baseDir, following ADR-002's issues/{repo}/{number}...
// layout (owner is deliberately not part of the path, matching the
// hand-maintained export directory ADR-001 modeled this format on).
func NewEvidenceWriter(baseDir string) repositories.EvidenceWriter {
	return &evidenceWriter{baseDir: baseDir}
}

func (w *evidenceWriter) WriteIssue(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(issuePath(w.baseDir, ref, "json"), raw)
}

func (w *evidenceWriter) WriteTimeline(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	joined, err := joinRawArray(items)
	if err != nil {
		return fmt.Errorf("persistence: could not combine the timeline pages into one array for %s/%d: %w", ref.Repo(), ref.Number(), err)
	}
	return writeFile(issuePath(w.baseDir, ref, "timeline.json"), joined)
}

func (w *evidenceWriter) WritePullRequest(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeFile(issuePath(w.baseDir, ref, "pull.json"), raw)
}

func (w *evidenceWriter) WriteReviewComments(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	joined, err := joinRawArray(items)
	if err != nil {
		return fmt.Errorf("persistence: could not combine the review comment pages into one array for %s/%d: %w", ref.Repo(), ref.Number(), err)
	}
	return writeFile(issuePath(w.baseDir, ref, "review-comments.json"), joined)
}

// issuePath builds the ADR-002 on-disk path for ref's evidence file with
// the given suffix, shared by evidenceWriter and documentWriter (owner is
// deliberately not part of the path, matching the hand-maintained export
// directory ADR-002 was modeled on).
func issuePath(baseDir string, ref valueobjects.IssueRef, suffix string) string {
	return filepath.Join(baseDir, "issues", ref.Repo(), fmt.Sprintf("%d.%s", ref.Number(), suffix))
}

// joinRawArray concatenates items into a JSON array by splicing their raw
// bytes directly, rather than json.Marshal-ing the slice: encoding/json
// compacts each json.RawMessage element (stripping insignificant
// whitespace), which would break ADR-001's verbatim-evidence guarantee.
func joinRawArray(items []json.RawMessage) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('[')
	for i, item := range items {
		if len(item) == 0 {
			return nil, fmt.Errorf("persistence: item %d is empty", i)
		}
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(item)
	}
	buf.WriteByte(']')
	return buf.Bytes(), nil
}

func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("persistence: create directory for %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("persistence: could not save data to %s: %w", path, err)
	}
	return nil
}
