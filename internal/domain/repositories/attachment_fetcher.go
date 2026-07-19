package repositories

import "context"

// AttachmentFetcher is the abstract port the application layer depends on
// to download a single attachment referenced by a rendered Document
// (ADR-002's mandatory-local-download policy); infrastructure implements it
// via an authenticated request (required for private-repository
// attachments). contentType is the response's Content-Type header, the
// only reliable source of the attachment's file extension — the
// user-attachments URL path does not encode one.
type AttachmentFetcher interface {
	Fetch(ctx context.Context, url string) (data []byte, contentType string, err error)
}
