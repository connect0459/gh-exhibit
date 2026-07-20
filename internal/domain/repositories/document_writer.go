package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// DocumentWriter is the abstract port the application layer depends on to
// persist the rendered Markdown document for an issue or pull request;
// infrastructure implements it (dependency inversion), separate from
// EvidenceWriter, which is scoped to raw JSON evidence only — raw JSON is
// the evidentiary source of truth and the rendered Markdown is a
// regenerable view of it, deliberately different concerns.
type DocumentWriter interface {
	// WriteDocument persists ref's fully rendered Markdown document.
	WriteDocument(ctx context.Context, ref valueobjects.IssueRef, rendered []byte) error
}
