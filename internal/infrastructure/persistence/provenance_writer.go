package persistence

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// provenanceWriter implements repositories.ProvenanceWriter against the
// local filesystem.
type provenanceWriter struct {
	baseDir string
}

// NewProvenanceWriter builds a repositories.ProvenanceWriter that persists
// which tool, version, and commit produced an export under baseDir, at
// {repo}/{number}/evidence/provenance.json.
func NewProvenanceWriter(baseDir string) repositories.ProvenanceWriter {
	return &provenanceWriter{baseDir: baseDir}
}

// WriteProvenance implements repositories.ProvenanceWriter.
func (w *provenanceWriter) WriteProvenance(ctx context.Context, ref valueobjects.IssueRef, provenance valueobjects.Provenance) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, err := json.Marshal(struct {
		Tool    string `json:"tool"`
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}{Tool: provenance.Tool(), Version: provenance.Version(), Commit: provenance.Commit()})
	if err != nil {
		return fmt.Errorf("marshal provenance: %w", err)
	}
	return writeFile(evidencePath(w.baseDir, ref, "provenance.json"), encoded)
}
