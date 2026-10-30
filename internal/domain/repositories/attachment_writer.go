package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// AttachmentWriter is the abstract port the application layer depends on to
// persist a fetched attachment and this run's failure summary to local
// storage; infrastructure implements it, symmetric to
// EvidenceWriter/DocumentWriter but for a different artifact shape (binary
// assets, not JSON or Markdown).
type AttachmentWriter interface {
	// WriteAsset writes a single downloaded attachment's data under
	// {repo}/{number}/assets/{filename}.
	WriteAsset(ctx context.Context, ref valueobjects.IssueRef, filename valueobjects.AssetFilename, data []byte) error

	// WriteFetchErrorLog persists this run's attachment fetch-failure
	// summary to {repo}/{number}/evidence/fetch-errors.log, so a failure
	// is traceable even after the Markdown placeholder is the only inline
	// trace of it. An empty log removes any existing fetch-errors.log
	// instead of writing one, so a stale log from a prior failing run
	// does not survive a rerun where every attachment now succeeds.
	WriteFetchErrorLog(ctx context.Context, ref valueobjects.IssueRef, log []byte) error

	// WriteAttachmentLinks persists which downloaded file stands in for
	// which original attachment URL to
	// {repo}/{number}/evidence/attachments.json, so a later reader of the
	// export can resolve an attachment URL in the raw evidence to its local
	// copy. An empty links still writes an empty array, so an export without
	// attachments stays distinguishable from one that predates this file,
	// and a stale file from a prior run is replaced.
	WriteAttachmentLinks(ctx context.Context, ref valueobjects.IssueRef, links []valueobjects.AttachmentLink) error
}
