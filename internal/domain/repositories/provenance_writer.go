package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// ProvenanceWriter is the abstract port the application layer depends on to
// persist which tool, version, and commit produced an export;
// infrastructure implements it (dependency inversion). Kept separate from
// EvidenceWriter rather than added as a fifth method on it: EvidenceWriter
// persists raw, verbatim, GitHub-origin data, while a Provenance is
// gh-exhibit's own self-reported record of itself — a different kind of
// thing than the data it is bundled alongside.
type ProvenanceWriter interface {
	// WriteProvenance persists which tool, version, and commit produced
	// ref's export.
	WriteProvenance(ctx context.Context, ref valueobjects.IssueRef, provenance valueobjects.Provenance) error
}
