package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// rewriteTransport redirects every outgoing request to target (an
// httptest.Server's host:port), regardless of the request's original host.
// api.NewRESTClient is pointed at the fixed hostname "github.localhost" (see
// newTestFetcher), which restURL resolves to a plain-HTTP URL with no
// real DNS/TLS involved; this transport is the seam that lands it on the
// local fake server instead.
type rewriteTransport struct {
	target string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Host = t.target
	req.Host = t.target
	return http.DefaultTransport.RoundTrip(req)
}

func newTestFetcher(t *testing.T, server *httptest.Server) repositories.EvidenceFetcher {
	t.Helper()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", server.URL, err)
	}

	fetcher, err := NewEvidenceFetcher(api.ClientOptions{
		Host:      "github.localhost",
		AuthToken: "test-token",
		Transport: &rewriteTransport{target: u.Host},
	})
	if err != nil {
		t.Fatalf("NewEvidenceFetcher() error = %v", err)
	}
	return fetcher
}

func testIssueRef(t *testing.T) valueobjects.IssueRef {
	t.Helper()

	ref, err := valueobjects.NewIssueRef("octocat", "hello-world", 42)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}
	return ref
}

func TestFetchIssue_ReturnsResponseBodyVerbatim(t *testing.T) {
	const body = `{"number":42,"title":"Some issue"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/repos/octocat/hello-world/issues/42" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchIssue() error = %v", err)
	}
	if string(got) != body {
		t.Fatalf("FetchIssue() = %q, want %q", got, body)
	}
}

func TestFetchPullRequest_ReturnsResponseBodyVerbatim(t *testing.T) {
	const body = `{"number":42,"title":"Some PR"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/repos/octocat/hello-world/pulls/42" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchPullRequest(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchPullRequest() error = %v", err)
	}
	if string(got) != body {
		t.Fatalf("FetchPullRequest() = %q, want %q", got, body)
	}
}

func TestFetchTimeline_FollowsLinkHeaderAndConcatenatesPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world/issues/42/timeline" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/octocat/hello-world/issues/42/timeline?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"id":1}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"id":2}]`))
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchTimeline() error = %v", err)
	}
	if len(got) != 2 || string(got[0]) != `{"id":1}` || string(got[1]) != `{"id":2}` {
		t.Fatalf("FetchTimeline() = %v, want [{\"id\":1} {\"id\":2}]", got)
	}
}

func TestFetchReviewComments_FollowsLinkHeaderAndConcatenatesPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world/pulls/42/comments" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/octocat/hello-world/pulls/42/comments?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"id":10}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"id":20}]`))
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchReviewComments(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchReviewComments() error = %v", err)
	}
	if len(got) != 2 || string(got[0]) != `{"id":10}` || string(got[1]) != `{"id":20}` {
		t.Fatalf("FetchReviewComments() = %v, want [{\"id\":10} {\"id\":20}]", got)
	}
}

func TestFetchPullRequestFiles_FollowsLinkHeaderAndConcatenatesPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world/pulls/42/files" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/octocat/hello-world/pulls/42/files?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"filename":"a.go"}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"filename":"b.go"}]`))
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchPullRequestFiles(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchPullRequestFiles() error = %v", err)
	}
	if len(got) != 2 || string(got[0]) != `{"filename":"a.go"}` || string(got[1]) != `{"filename":"b.go"}` {
		t.Fatalf("FetchPullRequestFiles() = %v, want [{\"filename\":\"a.go\"} {\"filename\":\"b.go\"}]", got)
	}
}

func TestFetchPullRequestCommits_FollowsLinkHeaderAndConcatenatesPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world/pulls/42/commits" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/octocat/hello-world/pulls/42/commits?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"sha":"aaa"}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"sha":"bbb"}]`))
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchPullRequestCommits(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchPullRequestCommits() error = %v", err)
	}
	if len(got) != 2 || string(got[0]) != `{"sha":"aaa"}` || string(got[1]) != `{"sha":"bbb"}` {
		t.Fatalf("FetchPullRequestCommits() = %v, want [{\"sha\":\"aaa\"} {\"sha\":\"bbb\"}]", got)
	}
}

func TestFetchSubIssues_FollowsLinkHeaderAndConcatenatesPages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/octocat/hello-world/issues/42/sub_issues" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/octocat/hello-world/issues/42/sub_issues?page=2>; rel="next"`, r.Host))
			_, _ = w.Write([]byte(`[{"number":65}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"number":66}]`))
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchSubIssues(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchSubIssues() error = %v", err)
	}
	if len(got) != 2 || string(got[0]) != `{"number":65}` || string(got[1]) != `{"number":66}` {
		t.Fatalf("FetchSubIssues() = %v, want [{\"number\":65} {\"number\":66}]", got)
	}
}

func TestFetchIssue_RetriesAfterRateLimitedResponseThenSucceeds(t *testing.T) {
	const body = `{"number":42,"title":"Some issue"}`
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchIssue() error = %v", err)
	}
	if string(got) != body {
		t.Fatalf("FetchIssue() = %q, want %q", got, body)
	}
	if calls != 2 {
		t.Fatalf("server received %d calls, want 2", calls)
	}
}

func TestFetchIssue_RetriesOn403WhenRateLimitRemainingIsZero(t *testing.T) {
	const body = `{"number":42,"title":"Some issue"}`
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", "0")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchIssue() error = %v", err)
	}
	if string(got) != body {
		t.Fatalf("FetchIssue() = %q, want %q", got, body)
	}
	if calls != 2 {
		t.Fatalf("server received %d calls, want 2", calls)
	}
}

func TestFetchIssue_DoesNotRetryPermissionDenied403(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))
	if err == nil {
		t.Fatal("FetchIssue() error = nil, want the permission-denied error")
	}
	if calls != 1 {
		t.Fatalf("server received %d calls, want 1 (no retry for an ordinary permission error)", calls)
	}
}

func TestFetchIssue_StopsAtMaxAttemptsAndReturnsTheLastError(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))
	if err == nil {
		t.Fatal("FetchIssue() error = nil, want the exhausted rate-limit error")
	}
	if calls != maxRetryAttempts {
		t.Fatalf("server received %d calls, want %d", calls, maxRetryAttempts)
	}
}

// fakeRequester and sleepSpy stand in for the real REST client and the real
// interruptible sleep, letting a test observe the exact wait duration doWithRetry
// computes and simulate a network-level error, neither of which a real
// httptest.Server can do without either a real sleep or a broken connection.
type fakeCall struct {
	resp *http.Response
	err  error
}

type fakeRequester struct {
	calls []fakeCall
	n     int
}

func (f *fakeRequester) RequestWithContext(_ context.Context, _, _ string, _ io.Reader) (*http.Response, error) {
	call := f.calls[f.n]
	f.n++
	return call.resp, call.err
}

type sleepSpy struct {
	waits []time.Duration
}

func (s *sleepSpy) sleep(_ context.Context, d time.Duration) error {
	s.waits = append(s.waits, d)
	return nil
}

func okResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

func TestFetchIssue_WaitsTheDurationSpecifiedByRetryAfterBeforeRetrying(t *testing.T) {
	req := &fakeRequester{calls: []fakeCall{
		{err: &api.HTTPError{StatusCode: http.StatusTooManyRequests, Headers: http.Header{"Retry-After": []string{"2"}}}},
		{resp: okResponse()},
	}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err != nil {
		t.Fatalf("FetchIssue() error = %v, want nil", err)
	}
	if req.n != 2 {
		t.Fatalf("requester called %d times, want 2", req.n)
	}
	if len(spy.waits) != 1 || spy.waits[0] != 2*time.Second {
		t.Fatalf("waits = %v, want [2s]", spy.waits)
	}
}

func TestFetchIssue_UsesFixedBackoffWhenNeitherRateLimitHeaderIsPresent(t *testing.T) {
	req := &fakeRequester{calls: []fakeCall{
		{err: &api.HTTPError{StatusCode: http.StatusTooManyRequests, Headers: http.Header{}}},
		{resp: okResponse()},
	}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err != nil {
		t.Fatalf("FetchIssue() error = %v, want nil", err)
	}
	if req.n != 2 {
		t.Fatalf("requester called %d times, want 2", req.n)
	}
	if len(spy.waits) != 1 || spy.waits[0] != fixedBackoffBase {
		t.Fatalf("waits = %v, want [%v] (the fixed-backoff fallback for attempt 0)", spy.waits, fixedBackoffBase)
	}
}

func TestFetchIssue_UsesFixedBackoffWhenRateLimitHeadersAreMalformed(t *testing.T) {
	req := &fakeRequester{calls: []fakeCall{
		{err: &api.HTTPError{StatusCode: http.StatusTooManyRequests, Headers: http.Header{
			"Retry-After":       []string{"not-a-number"},
			"X-RateLimit-Reset": []string{"also-not-a-number"},
		}}},
		{resp: okResponse()},
	}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err != nil {
		t.Fatalf("FetchIssue() error = %v, want nil", err)
	}
	if len(spy.waits) != 1 || spy.waits[0] != fixedBackoffBase {
		t.Fatalf("waits = %v, want [%v] (malformed headers must fall through to the fixed-backoff fallback)", spy.waits, fixedBackoffBase)
	}
}

func TestFetchIssue_UsesFixedBackoffWhenRetryAfterIsNegative(t *testing.T) {
	req := &fakeRequester{calls: []fakeCall{
		{err: &api.HTTPError{StatusCode: http.StatusTooManyRequests, Headers: http.Header{
			"Retry-After": []string{"-5"},
		}}},
		{resp: okResponse()},
	}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err != nil {
		t.Fatalf("FetchIssue() error = %v, want nil", err)
	}
	if len(spy.waits) != 1 || spy.waits[0] != fixedBackoffBase {
		t.Fatalf("waits = %v, want [%v] (a negative Retry-After must fall through to the fixed-backoff fallback)", spy.waits, fixedBackoffBase)
	}
}

func TestFetchIssue_UsesFixedBackoffWhenRetryAfterOverflowsDuration(t *testing.T) {
	req := &fakeRequester{calls: []fakeCall{
		{err: &api.HTTPError{StatusCode: http.StatusTooManyRequests, Headers: http.Header{
			"Retry-After": []string{"9223372037"},
		}}},
		{resp: okResponse()},
	}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err != nil {
		t.Fatalf("FetchIssue() error = %v, want nil", err)
	}
	if len(spy.waits) != 1 || spy.waits[0] != fixedBackoffBase {
		t.Fatalf("waits = %v, want [%v] (a Retry-After that overflows time.Duration must fall through to the fixed-backoff fallback)", spy.waits, fixedBackoffBase)
	}
}

func TestFetchIssue_UsesFixedBackoffWhenRateLimitResetOverflows(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Reset", "9223372036854775807")
	req := &fakeRequester{calls: []fakeCall{
		{err: &api.HTTPError{StatusCode: http.StatusTooManyRequests, Headers: headers}},
		{resp: okResponse()},
	}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err != nil {
		t.Fatalf("FetchIssue() error = %v, want nil", err)
	}
	if len(spy.waits) != 1 || spy.waits[0] != fixedBackoffBase {
		t.Fatalf("waits = %v, want [%v] (an X-RateLimit-Reset that overflows time.Unix/time.Until must fall through to the fixed-backoff fallback)", spy.waits, fixedBackoffBase)
	}
}

func TestFetchIssue_UsesFixedBackoffWhenRateLimitResetIsNegative(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-RateLimit-Reset", "-5")
	req := &fakeRequester{calls: []fakeCall{
		{err: &api.HTTPError{StatusCode: http.StatusTooManyRequests, Headers: headers}},
		{resp: okResponse()},
	}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err != nil {
		t.Fatalf("FetchIssue() error = %v, want nil", err)
	}
	if len(spy.waits) != 1 || spy.waits[0] != fixedBackoffBase {
		t.Fatalf("waits = %v, want [%v] (a negative X-RateLimit-Reset must fall through to the fixed-backoff fallback)", spy.waits, fixedBackoffBase)
	}
}

func TestFetchIssue_DoesNotRetryANetworkLevelError(t *testing.T) {
	wantErr := errors.New("connection refused")
	req := &fakeRequester{calls: []fakeCall{{err: wantErr}}}
	spy := &sleepSpy{}
	fetcher := &evidenceFetcher{client: req, sleep: spy.sleep}

	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if !errors.Is(err, wantErr) {
		t.Fatalf("FetchIssue() error = %v, want %v", err, wantErr)
	}
	if req.n != 1 {
		t.Fatalf("requester called %d times, want 1 (no retry)", req.n)
	}
	if len(spy.waits) != 0 {
		t.Fatalf("waits = %v, want none", spy.waits)
	}
}

// alwaysNextRequester simulates a misbehaving or misconfigured server whose
// Link header always claims there is a next page, to verify fetchPaginated
// does not follow it forever.
type alwaysNextRequester struct {
	calls int
}

func (r *alwaysNextRequester) RequestWithContext(_ context.Context, _, _ string, _ io.Reader) (*http.Response, error) {
	r.calls++
	header := http.Header{}
	header.Set("Link", `<http://example.invalid/next>; rel="next"`)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Request:    &http.Request{URL: &url.URL{Scheme: "http", Host: "example.invalid", Path: "/next"}},
		Body:       io.NopCloser(strings.NewReader(`[]`)),
	}, nil
}

func TestFetchTimeline_StopsAfterOnePageWhenLinkHeaderIsAbsent(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchTimeline() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("server received %d calls, want 1 (no Link header means no next page)", calls)
	}
	if len(got) != 1 || string(got[0]) != `{"id":1}` {
		t.Fatalf("FetchTimeline() = %v, want [{\"id\":1}]", got)
	}
}

func TestFetchTimeline_FollowsTheNextRelAmongMultipleRels(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(
				`<http://%s/repos/octocat/hello-world/issues/42/timeline?page=1>; rel="prev", `+
					`<http://%s/repos/octocat/hello-world/issues/42/timeline?page=2>; rel="next", `+
					`<http://%s/repos/octocat/hello-world/issues/42/timeline?page=5>; rel="last"`,
				r.Host, r.Host, r.Host))
			_, _ = w.Write([]byte(`[{"id":1}]`))
		case "2":
			_, _ = w.Write([]byte(`[{"id":2}]`))
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchTimeline() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("server received %d calls, want 2 (only the next rel should be followed)", calls)
	}
	if len(got) != 2 || string(got[0]) != `{"id":1}` || string(got[1]) != `{"id":2}` {
		t.Fatalf("FetchTimeline() = %v, want [{\"id\":1} {\"id\":2}]", got)
	}
}

func TestFetchTimeline_StopsAfterOnePageWhenOnlyPrevAndLastRelsArePresent(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Link", fmt.Sprintf(
			`<http://%s/repos/octocat/hello-world/issues/42/timeline?page=1>; rel="prev", `+
				`<http://%s/repos/octocat/hello-world/issues/42/timeline?page=5>; rel="last"`,
			r.Host, r.Host))
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchTimeline() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("server received %d calls, want 1 (this is the last page)", calls)
	}
	if len(got) != 1 || string(got[0]) != `{"id":1}` {
		t.Fatalf("FetchTimeline() = %v, want [{\"id\":1}]", got)
	}
}

func TestFetchTimeline_StopsAfterOnePageOnAMalformedLinkHeader(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Link", "not a valid link header")
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchTimeline() error = %v", err)
	}
	if calls != 1 {
		t.Fatalf("server received %d calls, want 1 (a malformed Link header should not be followed)", calls)
	}
	if len(got) != 1 || string(got[0]) != `{"id":1}` {
		t.Fatalf("FetchTimeline() = %v, want [{\"id\":1}]", got)
	}
}

func TestFetchTimeline_FollowsANextURLContainingAnUnescapedCommaInItsQuery(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Query().Get("filter") {
		case "":
			w.Header().Set("Link", fmt.Sprintf(
				`<http://%s/repos/octocat/hello-world/issues/42/timeline?filter=a,b>; rel="next", `+
					`<http://%s/repos/octocat/hello-world/issues/42/timeline?page=5>; rel="last"`,
				r.Host, r.Host))
			_, _ = w.Write([]byte(`[{"id":1}]`))
		case "a,b":
			_, _ = w.Write([]byte(`[{"id":2}]`))
		default:
			t.Errorf("unexpected filter: %s", r.URL.Query().Get("filter"))
		}
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	got, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("FetchTimeline() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("server received %d calls, want 2 (the comma-bearing next URL should still be followed)", calls)
	}
	if len(got) != 2 || string(got[0]) != `{"id":1}` || string(got[1]) != `{"id":2}` {
		t.Fatalf("FetchTimeline() = %v, want [{\"id\":1} {\"id\":2}]", got)
	}
}

func TestFetchTimeline_StopsFollowingAnUnboundedLinkHeaderChain(t *testing.T) {
	req := &alwaysNextRequester{}
	fetcher := &evidenceFetcher{client: req, sleep: func(context.Context, time.Duration) error { return nil }}

	_, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))

	if err == nil {
		t.Fatal("FetchTimeline() error = nil, want a pagination-limit error")
	}
	if req.calls != maxPaginationPages {
		t.Fatalf("requester called %d times, want %d (the pagination page cap)", req.calls, maxPaginationPages)
	}
}

func TestFetchIssue_RefusesToFollowARedirectToADifferentOrigin(t *testing.T) {
	attackerCalls := 0
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attackerCalls++
		_, _ = w.Write([]byte(`{"number":42,"title":"attacker-controlled"}`))
	}))
	defer attacker.Close()

	legit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, attacker.URL+"/attacker-path", http.StatusFound)
	}))
	defer legit.Close()

	fetcher := newTestFetcher(t, legit)
	_, err := fetcher.FetchIssue(context.Background(), testIssueRef(t))

	if err == nil {
		t.Fatal("FetchIssue() error = nil, want an error for a response redirecting to a different origin")
	}
	if attackerCalls != 0 {
		t.Fatalf("attacker server received %d calls, want 0 (the redirect must never be followed)", attackerCalls)
	}
}

func TestFetchTimeline_RefusesToFollowARedirectOnTheFirstPageToADifferentOrigin(t *testing.T) {
	attackerCalls := 0
	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attackerCalls++
		// A misbehaving/attacker-controlled page 1 that both serves content
		// and claims a self-referential next page, matching the report's
		// "first hop redirected poisons expectedOrigin" scenario: if the
		// redirect guard didn't exist, this Link header alone would have
		// been enough to make fetchPaginated treat the attacker's own
		// origin as trusted for the rest of the pagination chain.
		w.Header().Set("Link", fmt.Sprintf(`<http://%s/attacker-path?page=2>; rel="next"`, r.Host))
		_, _ = w.Write([]byte(`[{"id":"attacker-page"}]`))
	}))
	defer attacker.Close()

	legit := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, attacker.URL+"/attacker-path", http.StatusFound)
	}))
	defer legit.Close()

	fetcher := newTestFetcher(t, legit)
	_, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))

	if err == nil {
		t.Fatal("FetchTimeline() error = nil, want an error for the first page redirecting to a different origin")
	}
	if attackerCalls != 0 {
		t.Fatalf("attacker server received %d calls, want 0 (the redirect must never be followed, and must not poison the expected origin for later pages)", attackerCalls)
	}
}

func TestFetchTimeline_RefusesToFollowANextPageURLPointingToADifferentHost(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Link", `<http://attacker.invalid/repos/octocat/hello-world/issues/42/timeline?page=2>; rel="next"`)
		}
		_, _ = w.Write([]byte(`[{"id":1}]`))
	}))
	defer server.Close()

	fetcher := newTestFetcher(t, server)
	_, err := fetcher.FetchTimeline(context.Background(), testIssueRef(t))

	if err == nil {
		t.Fatal("FetchTimeline() error = nil, want a host-mismatch error")
	}
	if calls != 1 {
		t.Fatalf("server received %d calls, want 1 (a next-page URL naming a different host must not be followed)", calls)
	}
}
