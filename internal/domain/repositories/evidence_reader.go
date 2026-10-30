package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// EvidenceReader is the abstract port the application layer depends on to
// read back the raw evidence an earlier export persisted for an issue or
// pull request, symmetric to EvidenceWriter; infrastructure implements it
// (dependency inversion).
type EvidenceReader interface {
	// ReadEvidence reads every raw evidence file persisted for ref. Only the
	// issue resource is required; any other file that is absent reads as
	// empty, since which files exist depends on the ref's kind and on the
	// version that exported it. It returns an error if the issue resource
	// is missing or a list-shaped file (timeline, review comments, files,
	// commits, check runs, sub-issues) is not a JSON array. The shape of the
	// object-shaped files is not checked here; a malformed one fails when
	// the evidence is assembled into a Document.
	ReadEvidence(ctx context.Context, ref valueobjects.IssueRef) (services.RawEvidence, error)
}
