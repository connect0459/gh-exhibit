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
	// is missing or any file is not valid JSON of its expected shape.
	ReadEvidence(ctx context.Context, ref valueobjects.IssueRef) (services.RawEvidence, error)
}
