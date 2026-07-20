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
	WriteAsset(ctx context.Context, ref valueobjects.IssueRef, filename string, data []byte) error

	// WriteFetchErrorLog persists this run's attachment fetch-failure
	// summary to {repo}/{number}/fetch-errors.log, so a failure is
	// traceable even after the Markdown placeholder is the only inline
	// trace of it. An empty log removes any existing fetch-errors.log
	// instead of writing one, so a stale log from a prior failing run
	// does not survive a rerun where every attachment now succeeds.
	WriteFetchErrorLog(ctx context.Context, ref valueobjects.IssueRef, log []byte) error
}
