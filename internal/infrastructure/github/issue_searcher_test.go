package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func newTestSearcher(t *testing.T, server *httptest.Server) repositories.IssueSearcher {
	t.Helper()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", server.URL, err)
	}

	searcher, err := NewIssueSearcher(api.ClientOptions{
		Host:      "github.localhost",
		AuthToken: "test-token",
		Transport: &rewriteTransport{target: u.Host},
	})
	if err != nil {
		t.Fatalf("NewIssueSearcher() error = %v", err)
	}
	return searcher
}

func testSearchQuery(t *testing.T, author, assignee string, kinds []valueobjects.IssueKind, createdAfter, createdBefore *time.Time, sort valueobjects.SearchSortField, order valueobjects.SearchSortOrder, maxResults int) valueobjects.SearchQuery {
	t.Helper()

	query, err := valueobjects.NewSearchQuery("octocat", "hello-world", author, assignee, kinds, createdAfter, createdBefore, sort, order, maxResults)
	if err != nil {
		t.Fatalf("NewSearchQuery() error = %v", err)
	}
	return query
}

// searchQParam runs a Search call against a fake server that always
// responds with an empty result, and returns the "q" query-string
// parameter the server actually received — the exported entry point this
// package's Search-level tests assert query-string construction through,
// rather than calling the unexported buildSearchQueryString directly (this
// project's Evergreen Tests convention: a test names the exported unit it
// exercises, not an unexported one, since the exported entry point already
// reaches it).
func searchQParam(t *testing.T, query valueobjects.SearchQuery) string {
	t.Helper()

	var got string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query().Get("q")
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	defer server.Close()

	searcher := newTestSearcher(t, server)
	if _, err := searcher.Search(context.Background(), query); err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	return got
}

func TestSearch_SendsARepoOnlyQualifierWhenUnfiltered(t *testing.T) {
	query := testSearchQuery(t, "", "", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	if got != "repo:octocat/hello-world" {
		t.Fatalf("q = %q, want %q", got, "repo:octocat/hello-world")
	}
}

func TestSearch_SendsAuthorAndAssigneeQualifiers(t *testing.T) {
	query := testSearchQuery(t, "monalisa", "hubot", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	want := "repo:octocat/hello-world author:monalisa assignee:hubot"
	if got != want {
		t.Fatalf("q = %q, want %q", got, want)
	}
}

func TestSearch_OmitsTheIsQualifierWhenBothKindsAreRequested(t *testing.T) {
	kinds := []valueobjects.IssueKind{valueobjects.IssueKindIssue, valueobjects.IssueKindPullRequest}
	query := testSearchQuery(t, "", "", kinds, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	if got != "repo:octocat/hello-world" {
		t.Fatalf("q = %q, want no is: qualifier for both kinds", got)
	}
}

func TestSearch_AddsIsIssueWhenOnlyIssueKindIsRequested(t *testing.T) {
	kinds := []valueobjects.IssueKind{valueobjects.IssueKindIssue}
	query := testSearchQuery(t, "", "", kinds, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	want := "repo:octocat/hello-world is:issue"
	if got != want {
		t.Fatalf("q = %q, want %q", got, want)
	}
}

func TestSearch_AddsIsPrWhenOnlyPullRequestKindIsRequested(t *testing.T) {
	kinds := []valueobjects.IssueKind{valueobjects.IssueKindPullRequest}
	query := testSearchQuery(t, "", "", kinds, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	want := "repo:octocat/hello-world is:pr"
	if got != want {
		t.Fatalf("q = %q, want %q", got, want)
	}
}

func TestSearch_CombinesBothDateBoundsIntoARange(t *testing.T) {
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	before := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	query := testSearchQuery(t, "", "", nil, &after, &before, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	want := "repo:octocat/hello-world created:2024-01-01..2024-06-01"
	if got != want {
		t.Fatalf("q = %q, want %q", got, want)
	}
}

func TestSearch_UsesAOneSidedLowerBoundWhenOnlyCreatedAfterIsSet(t *testing.T) {
	after := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	query := testSearchQuery(t, "", "", nil, &after, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	want := "repo:octocat/hello-world created:>=2024-01-01"
	if got != want {
		t.Fatalf("q = %q, want %q", got, want)
	}
}

func TestSearch_UsesAOneSidedUpperBoundWhenOnlyCreatedBeforeIsSet(t *testing.T) {
	before := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	query := testSearchQuery(t, "", "", nil, nil, &before, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	got := searchQParam(t, query)

	want := "repo:octocat/hello-world created:<=2024-06-01"
	if got != want {
		t.Fatalf("q = %q, want %q", got, want)
	}
}

func TestSearch_SendsQSortOrderAndPerPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search/issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		if got := q.Get("q"); got != "repo:octocat/hello-world author:monalisa" {
			t.Errorf("q = %q, want %q", got, "repo:octocat/hello-world author:monalisa")
		}
		if got := q.Get("sort"); got != "comments" {
			t.Errorf("sort = %q, want %q", got, "comments")
		}
		if got := q.Get("order"); got != "asc" {
			t.Errorf("order = %q, want %q", got, "asc")
		}
		if got := q.Get("per_page"); got != "17" {
			t.Errorf("per_page = %q, want %q", got, "17")
		}
		_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
	}))
	defer server.Close()

	searcher := newTestSearcher(t, server)
	query := testSearchQuery(t, "monalisa", "", nil, nil, nil, valueobjects.SearchSortByComments, valueobjects.SearchOrderAscending, 17)

	_, err := searcher.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
}

func TestSearch_DecodesTotalCountAndItemsIntoASearchResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"total_count": 5,
			"items": [
				{"number": 1, "created_at": "2024-01-01T00:00:00Z", "updated_at": "2024-01-02T00:00:00Z", "comments": 3},
				{"number": 2, "created_at": "2024-02-01T00:00:00Z", "updated_at": "2024-02-02T00:00:00Z", "comments": 0}
			]
		}`))
	}))
	defer server.Close()

	searcher := newTestSearcher(t, server)
	query := testSearchQuery(t, "", "", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	result, err := searcher.Search(context.Background(), query)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if got := result.TotalCount(); got != 5 {
		t.Fatalf("TotalCount() = %d, want 5", got)
	}
	matches := result.Matches()
	if len(matches) != 2 {
		t.Fatalf("len(Matches()) = %d, want 2", len(matches))
	}
	if matches[0].Number() != 1 || matches[0].Comments() != 3 {
		t.Fatalf("Matches()[0] = %+v, want number 1 with 3 comments", matches[0])
	}
	if matches[1].Number() != 2 {
		t.Fatalf("Matches()[1].Number() = %d, want 2", matches[1].Number())
	}
	wantCreated := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !matches[0].CreatedAt().Equal(wantCreated) {
		t.Fatalf("Matches()[0].CreatedAt() = %v, want %v", matches[0].CreatedAt(), wantCreated)
	}
}

func TestSearch_ReturnsAnErrorOnAServerFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	searcher := newTestSearcher(t, server)
	query := testSearchQuery(t, "", "", nil, nil, nil, valueobjects.SearchSortByCreated, valueobjects.SearchOrderDescending, 100)

	_, err := searcher.Search(context.Background(), query)
	if err == nil {
		t.Fatal("expected an error for a server failure, got nil")
	}
}
