package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// AttachmentLinkReader is the abstract port the application layer depends
// on to read back which downloaded file stands in for which original
// attachment URL, symmetric to AttachmentWriter.WriteAttachmentLinks.
type AttachmentLinkReader interface {
	// ReadAttachmentLinks reads ref's recorded attachment links. An export
	// that predates attachment links, or recorded none, reads as empty
	// rather than as an error.
	ReadAttachmentLinks(ctx context.Context, ref valueobjects.IssueRef) ([]valueobjects.AttachmentLink, error)
}
