package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// evidenceFetcher implements repositories.EvidenceFetcher against GitHub's
// REST API via go-gh.
type evidenceFetcher struct {
	client requester
	sleep  sleeper
}

// NewEvidenceFetcher builds a repositories.EvidenceFetcher backed by
// GitHub's REST API. Passing api.ClientOptions{} resolves host and auth
// token from the gh environment, matching api.DefaultRESTClient's behavior;
// tests override Host and Transport to point at a local fake server instead.
//
// opts.Transport is wrapped with a redirect-origin guard (see
// redirect_guard.go) before api.NewRESTClient sees it, so it takes
// precedence over the gh environment's http_unix_socket routing the same
// way any caller-supplied Transport does (api.ClientOptions.Transport's own
// documented behavior) — a limitation gh-exhibit does not otherwise use,
// since it never sets UnixDomainSocket itself.
func NewEvidenceFetcher(opts api.ClientOptions) (repositories.EvidenceFetcher, error) {
	opts.Transport = newRedirectGuardTransport(opts.Transport)

	client, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("create the GitHub REST client: %w", err)
	}

	return &evidenceFetcher{client: client, sleep: realSleep}, nil
}

func realSleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// FetchIssue implements repositories.EvidenceFetcher.
func (r *evidenceFetcher) FetchIssue(ctx context.Context, ref valueobjects.IssueRef) (json.RawMessage, error) {
	return r.fetchSingle(ctx, issuePath(ref))
}

// FetchPullRequest implements repositories.EvidenceFetcher.
func (r *evidenceFetcher) FetchPullRequest(ctx context.Context, ref valueobjects.IssueRef) (json.RawMessage, error) {
	return r.fetchSingle(ctx, pullPath(ref))
}

// FetchTimeline implements repositories.EvidenceFetcher.
func (r *evidenceFetcher) FetchTimeline(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error) {
	return r.fetchPaginated(ctx, issuePath(ref)+"/timeline")
}

// FetchReviewComments implements repositories.EvidenceFetcher.
func (r *evidenceFetcher) FetchReviewComments(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error) {
	return r.fetchPaginated(ctx, pullPath(ref)+"/comments")
}

// FetchPullRequestFiles implements repositories.EvidenceFetcher.
func (r *evidenceFetcher) FetchPullRequestFiles(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error) {
	return r.fetchPaginated(ctx, pullPath(ref)+"/files")
}

// FetchPullRequestCommits implements repositories.EvidenceFetcher.
func (r *evidenceFetcher) FetchPullRequestCommits(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error) {
	return r.fetchPaginated(ctx, pullPath(ref)+"/commits")
}

// FetchSubIssues implements repositories.EvidenceFetcher.
func (r *evidenceFetcher) FetchSubIssues(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error) {
	return r.fetchPaginated(ctx, issuePath(ref)+"/sub_issues")
}

func issuePath(ref valueobjects.IssueRef) string {
	return fmt.Sprintf("repos/%s/%s/issues/%d", ref.Owner(), ref.Repo(), ref.Number())
}

func pullPath(ref valueobjects.IssueRef) string {
	return fmt.Sprintf("repos/%s/%s/pulls/%d", ref.Owner(), ref.Repo(), ref.Number())
}

func (r *evidenceFetcher) fetchSingle(ctx context.Context, path string) (json.RawMessage, error) {
	resp, err := doWithRetry(pinRedirectOrigin(ctx), r.client, r.sleep, http.MethodGet, path)
	if err != nil {
		return nil, fmt.Errorf("fetch GitHub resource %s: %w", path, err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read GitHub response body for %s: %w", path, err)
	}

	return json.RawMessage(body), nil
}

// maxPaginationPages bounds fetchPaginated's Link-header-following loop
// against a misbehaving or misconfigured server whose Link header never
// stops claiming a next page. 1000 pages at GitHub's 100-item page-size
// ceiling (100,000 items) is far beyond any real issue/PR's timeline or
// review-comment count, so hitting it means the chain isn't legitimate.
const maxPaginationPages = 1000

// fetchPaginated returns one json.RawMessage per array element across all
// pages, following the Link header's "next" relation; the caller, not this
// fetcher, concatenates pages into a single persisted array.
func (r *evidenceFetcher) fetchPaginated(ctx context.Context, path string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	var expectedOrigin string

	for pages := 0; path != ""; pages++ {
		if pages == maxPaginationPages {
			return nil, fmt.Errorf("aborting GitHub pagination for %s after %d pages, Link header may be looping", path, maxPaginationPages)
		}

		resp, err := doWithRetry(pinRedirectOrigin(ctx), r.client, r.sleep, http.MethodGet, path)
		if err != nil {
			return nil, fmt.Errorf("fetch GitHub resource %s: %w", path, err)
		}
		if expectedOrigin == "" {
			expectedOrigin = requestOrigin(resp)
		}

		var page []json.RawMessage
		decodeErr := json.NewDecoder(resp.Body).Decode(&page)
		_ = resp.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode GitHub API page for %s: %w", path, decodeErr)
		}
		all = append(all, page...)

		next := nextPageURL(resp)
		if next != "" {
			if err := validatePaginationOrigin(next, expectedOrigin); err != nil {
				return nil, fmt.Errorf("fetch GitHub resource %s: %w", path, err)
			}
		}
		path = next
	}

	return all, nil
}
