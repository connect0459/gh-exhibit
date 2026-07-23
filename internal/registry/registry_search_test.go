package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// TestNewSearchService_ResolvesMatchesViaTheRealGitHubSearchAPI wires
// NewSearchService's real production types (github.NewIssueSearcher, not a
// hand-written fake) against a fake GitHub search/issues endpoint, the same
// "exercise the real registry-to-response path" precedent
// TestNewExportService_DownloadsAnAttachmentServedViaACrossOriginRedirect
// already establishes for NewExportService.
func TestNewSearchService_ResolvesMatchesViaTheRealGitHubSearchAPI(t *testing.T) {
	var requestedPath string
	var requestedQuery url.Values
	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"total_count": 1,
			"items": [{"number": 42, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-02T00:00:00Z", "comments": 3}]
		}`))
	}))
	defer githubAPI.Close()

	githubAPIURL, err := url.Parse(githubAPI.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", githubAPI.URL, err)
	}

	service, err := NewSearchService(Config{
		Host:      "github.localhost",
		AuthToken: "test-token",
		Transport: &hostScopedRewriteTransport{
			placeholderHosts: map[string]bool{
				"github.localhost":     true,
				"api.github.localhost": true,
			},
			target: githubAPIURL.Host,
		},
	})
	if err != nil {
		t.Fatalf("NewSearchService() error = %v", err)
	}

	criteria, err := valueobjects.NewSearchCriteria([]string{"octocat"}, nil, nil, nil, nil, 100, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending)
	if err != nil {
		t.Fatalf("unexpected error building search criteria: %v", err)
	}

	outcome, err := service.Search(context.Background(), "octocat", "hello-world", criteria)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if requestedPath != "/search/issues" {
		t.Fatalf("requested path = %q, want %q", requestedPath, "/search/issues")
	}
	if got := requestedQuery.Get("q"); got != "repo:octocat/hello-world author:octocat" {
		t.Fatalf("requested q = %q, want %q", got, "repo:octocat/hello-world author:octocat")
	}
	if len(outcome.Numbers) != 1 || outcome.Numbers[0] != 42 {
		t.Fatalf("Numbers = %v, want [42]", outcome.Numbers)
	}
}
