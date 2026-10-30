package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// AttachmentLinkReader is the abstract port the application layer depends
// on to read back which downloaded file stands in for which original
// attachment URL, symmetric to AttachmentWriter.WriteAttachmentLinks.
type AttachmentLinkReader interface {
	// ReadAttachmentLinks reads ref's recorded attachment links. recorded
	// reports whether the export recorded any at all: an export that
	// predates attachment links reads as no links and recorded false, which
	// is not an error, but is distinguishable from an export that recorded
	// an empty list (recorded true), where an unlinked attachment means the
	// download failed.
	ReadAttachmentLinks(ctx context.Context, ref valueobjects.IssueRef) (links []valueobjects.AttachmentLink, recorded bool, err error)
}
