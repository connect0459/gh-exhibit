// Package repositories defines the abstract ports the domain layer exposes
// for fetching and persisting raw GitHub evidence, rendered documents, and
// downloaded attachments — infrastructure implements each interface
// (dependency inversion), so the application layer depends only on these
// abstractions, never on a concrete GitHub or filesystem implementation.
package repositories

import (
	"context"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

// AttachmentFetcher is the abstract port the application layer depends on
// to download a single attachment referenced by a rendered Document
// (ADR-002's mandatory-local-download policy); infrastructure implements it
// via an authenticated request (required for private-repository
// attachments). contentType is the response's Content-Type header, the
// only reliable source of the attachment's file extension — the
// user-attachments URL path does not encode one.
type AttachmentFetcher interface {
	// Fetch downloads attachment's URL and returns its body and Content-Type.
	Fetch(ctx context.Context, attachment services.Attachment) (data []byte, contentType string, err error)
}
