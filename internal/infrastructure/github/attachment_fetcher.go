package github

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
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
// relative to its configured Host. Unexported so callers depend only on the
// repositories.AttachmentFetcher interface, not this infrastructure-layer
// type.
type attachmentFetcher struct {
	client doer
}

// NewAttachmentFetcher builds a repositories.AttachmentFetcher backed by an
// authenticated *http.Client, required to fetch attachments on private
// repositories (ADR-002). Passing api.ClientOptions{} resolves host and auth
// token from the gh environment; tests override Host and Transport to point
// at a local fake server instead.
func NewAttachmentFetcher(opts api.ClientOptions) (repositories.AttachmentFetcher, error) {
	client, err := api.NewHTTPClient(opts)
	if err != nil {
		return nil, fmt.Errorf("github: new HTTP client: %w", err)
	}

	return &attachmentFetcher{client: client}, nil
}

func (f *attachmentFetcher) Fetch(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("github: build request for %s: %w", url, err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("github: fetch %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("github: fetch %s: unexpected status %d", url, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("github: read response body for %s: %w", url, err)
	}

	return data, resp.Header.Get("Content-Type"), nil
}
