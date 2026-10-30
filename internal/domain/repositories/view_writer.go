package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// ViewWriter is the abstract port the application layer depends on to
// persist a browsable view of an issue or pull request's Document;
// infrastructure implements it (dependency inversion). Like the rendered
// Markdown, the view is a regenerable rendering of the raw evidence, not
// evidence itself.
type ViewWriter interface {
	// WriteView persists doc's view for ref. rewrite is applied to every
	// Markdown body before it is rendered, so the view can substitute
	// downloaded copies for remote attachment URLs.
	WriteView(ctx context.Context, ref valueobjects.IssueRef, doc valueobjects.Document, rewrite func(markdown string) string) error
}
