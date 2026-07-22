package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// issueSearcher implements repositories.IssueSearcher against GitHub's
// REST search API via go-gh.
type issueSearcher struct {
	client requester
	sleep  sleeper
}

// NewIssueSearcher builds a repositories.IssueSearcher backed by GitHub's
// REST search API, mirroring NewEvidenceFetcher's own construction (same
// redirect-guard wrapping, same requester/sleeper seams for testability).
func NewIssueSearcher(opts api.ClientOptions) (repositories.IssueSearcher, error) {
	opts.Transport = newRedirectGuardTransport(opts.Transport)

	client, err := api.NewRESTClient(opts)
	if err != nil {
		return nil, fmt.Errorf("create the GitHub REST client: %w", err)
	}

	return &issueSearcher{client: client, sleep: realSleep}, nil
}

// Search implements repositories.IssueSearcher. query's MaxResults is at
// most valueobjects.MaxSearchLimit (100), which is also GitHub search's own
// maximum page size, so a single request always covers it — no pagination
// loop is needed here, unlike evidenceFetcher's paginated fetches.
func (s *issueSearcher) Search(ctx context.Context, query valueobjects.SearchQuery) (valueobjects.SearchResult, error) {
	path := "search/issues?" + searchRequestValues(query).Encode()

	resp, err := doWithRetry(pinRedirectOrigin(ctx), s.client, s.sleep, http.MethodGet, path)
	if err != nil {
		return valueobjects.SearchResult{}, fmt.Errorf("search GitHub issues/PRs: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return valueobjects.SearchResult{}, fmt.Errorf("read GitHub search response body: %w", err)
	}

	return decodeSearchResponse(body)
}

// searchRequestValues builds the full set of query-string parameters for a
// GitHub search/issues request: the search qualifiers themselves under
// "q", plus "sort", "order", and "per_page".
func searchRequestValues(query valueobjects.SearchQuery) url.Values {
	values := url.Values{}
	values.Set("q", buildSearchQueryString(query))
	values.Set("sort", query.Sort().String())
	values.Set("order", query.Order().String())
	values.Set("per_page", strconv.Itoa(query.MaxResults()))
	return values
}

// buildSearchQueryString builds GitHub search's own "q" qualifier string
// for query: a "repo:" qualifier is always present; "author:"/"assignee:"
// are added only when query names one (GitHub's search query language has
// no OR semantics between repeated qualifiers of the same kind, so a
// multi-valued filter is never represented here — see
// domain/services.BuildSearchQueries); "is:issue"/"is:pr" is added only
// when query restricts to exactly one kind (omitted entirely, meaning
// both, when it names both or neither); the created-date range collapses
// to a single two-sided "created:after..before" qualifier when both bounds
// are set, or a one-sided ">="/"<=" qualifier when only one is.
func buildSearchQueryString(query valueobjects.SearchQuery) string {
	parts := []string{fmt.Sprintf("repo:%s/%s", query.Owner(), query.Repo())}

	if author := query.Author(); author != "" {
		parts = append(parts, "author:"+author)
	}
	if assignee := query.Assignee(); assignee != "" {
		parts = append(parts, "assignee:"+assignee)
	}
	if kinds := query.Kinds(); len(kinds) == 1 {
		switch kinds[0] {
		case valueobjects.IssueKindIssue:
			parts = append(parts, "is:issue")
		case valueobjects.IssueKindPullRequest:
			parts = append(parts, "is:pr")
		}
	}

	after, before := query.CreatedAfter(), query.CreatedBefore()
	switch {
	case after != nil && before != nil:
		parts = append(parts, fmt.Sprintf("created:%s..%s", after.Format(valueobjects.SearchDateLayout), before.Format(valueobjects.SearchDateLayout)))
	case after != nil:
		parts = append(parts, "created:>="+after.Format(valueobjects.SearchDateLayout))
	case before != nil:
		parts = append(parts, "created:<="+before.Format(valueobjects.SearchDateLayout))
	}

	return strings.Join(parts, " ")
}

// searchResponseItem mirrors one element of GitHub search/issues' own
// "items" array — only the fields domain/services.MergeSearchResults needs
// to merge, sort, and deduplicate matches.
type searchResponseItem struct {
	Number    int       `json:"number"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Comments  int       `json:"comments"`
}

// searchResponse mirrors GitHub search/issues' own top-level response
// shape.
type searchResponse struct {
	TotalCount int                  `json:"total_count"`
	Items      []searchResponseItem `json:"items"`
}

// decodeSearchResponse decodes body (a GitHub search/issues response) into
// a valueobjects.SearchResult.
func decodeSearchResponse(body []byte) (valueobjects.SearchResult, error) {
	var decoded searchResponse
	if err := json.Unmarshal(body, &decoded); err != nil {
		return valueobjects.SearchResult{}, fmt.Errorf("decode GitHub search response: %w", err)
	}

	matches := make([]valueobjects.SearchMatch, 0, len(decoded.Items))
	for _, item := range decoded.Items {
		match, err := valueobjects.NewSearchMatch(item.Number, item.CreatedAt, item.UpdatedAt, item.Comments)
		if err != nil {
			return valueobjects.SearchResult{}, fmt.Errorf("decode GitHub search result item #%d: %w", item.Number, err)
		}
		matches = append(matches, match)
	}

	result, err := valueobjects.NewSearchResult(matches, decoded.TotalCount)
	if err != nil {
		return valueobjects.SearchResult{}, fmt.Errorf("build GitHub search result: %w", err)
	}
	return result, nil
}
