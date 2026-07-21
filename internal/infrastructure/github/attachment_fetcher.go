package github

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

// doer is the subset of *http.Client's interface attachmentFetcher needs;
// *http.Client satisfies it directly, and tests substitute a fake.
type doer interface {
	Do(req *http.Request) (*http.Response, error)
}

// attachmentFetcher implements repositories.AttachmentFetcher against an
// attachment's own URL (e.g. github.com/user-attachments/assets/...), which
// is unrelated to the REST API host evidenceFetcher targets — so this type
// wraps a plain *http.Client (via api.NewHTTPClient) rather than
// evidenceFetcher's *api.RESTClient, which only knows how to build requests
// relative to its configured Host.
type attachmentFetcher struct {
	client   doer
	maxBytes int64
}

// maxAttachmentBytes bounds how much of a single attachment's response body
// Fetch reads into memory, a defense-in-depth cap independent of GitHub's
// own upload-size limits (which this project does not control and cannot
// rely on for a third-party or misconfigured host).
const maxAttachmentBytes = 100 * 1024 * 1024

// NewAttachmentFetcher builds a repositories.AttachmentFetcher backed by an
// authenticated *http.Client, required to fetch attachments on private
// repositories. Passing api.ClientOptions{} resolves host and auth token
// from the gh environment; tests override Host and Transport to point at a
// local fake server instead.
//
// Unlike NewEvidenceFetcher, opts.Transport is not wrapped with the
// redirect-origin guard from redirect_guard.go: a real attachment URL
// (e.g. github.com/user-attachments/assets/...) legitimately redirects
// cross-origin to serve the actual bytes (e.g. to a signed, time-limited
// S3 URL), so pinning the origin here would reject every such fetch. This
// stays safe without the guard because net/http itself strips the
// Authorization/Cookie headers on a redirect whose host differs from the
// original request's, so the credential this client attaches never
// reaches the redirect target.
func NewAttachmentFetcher(opts api.ClientOptions) (repositories.AttachmentFetcher, error) {
	client, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("create the GitHub-authenticated HTTP client: %w", err)
	}

	return &attachmentFetcher{client: client, maxBytes: maxAttachmentBytes}, nil
}

// Fetch implements repositories.AttachmentFetcher.
func (f *attachmentFetcher) Fetch(ctx context.Context, attachment services.Attachment) ([]byte, string, error) {
	url := attachment.URL().String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("build request for %s: %w", url, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, f.maxBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("read response body for %s: %w", url, err)
	}
	if int64(len(data)) > f.maxBytes {
		return nil, "", fmt.Errorf("attachment at %s exceeds the %d-byte size limit", url, f.maxBytes)
	}

	return data, resp.Header.Get("Content-Type"), nil
}
