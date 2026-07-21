package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// hostScopedRewriteTransport rewrites a request whose Host is one of
// placeholderHosts to target's real address, and passes every other
// request through unrewritten. This is the seam that lets a test point
// NewExportService's real, production-wired fetchers at fake servers: an
// evidence request nominally addressed to "api.github.localhost" (go-gh's
// own REST URL prefix for its special-cased "github.localhost" test host)
// and an attachment request addressed to "github.localhost" itself both
// land on the same fake GitHub server, while a redirect hop already
// naming a second fake server's real address passes through untouched —
// letting that second hop actually reach a distinct origin, the shape a
// real GitHub attachment URL's redirect to its CDN takes.
type hostScopedRewriteTransport struct {
	placeholderHosts map[string]bool
	target           string
}

func (t *hostScopedRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	if t.placeholderHosts[req.URL.Host] {
		req.URL.Host = t.target
		req.Host = t.target
	}
	return http.DefaultTransport.RoundTrip(req)
}

// TestNewExportService_DownloadsAnAttachmentServedViaACrossOriginRedirect
// wires NewExportService's real production types — the same
// github.NewEvidenceFetcher/github.NewAttachmentFetcher/persistence
// writers registry.go itself constructs, not hand-written fakes — against
// two fake HTTP servers shaped like a real GitHub interaction: one serving
// the issue resource/timeline and the attachment's first hop, the other
// standing in for the cross-origin CDN (e.g. S3) that first hop redirects
// to. A regression at any layer between the two constructors and the
// final on-disk output — not just inside attachmentFetcher.Fetch in
// isolation — would surface here, which is what let v0.3.0's
// redirect-origin guard reach production despite that guard's own unit
// tests passing.
func TestNewExportService_DownloadsAnAttachmentServedViaACrossOriginRedirect(t *testing.T) {
	const attachmentBody = "not-actually-a-png"
	const attachmentPath = "/user-attachments/assets/abc-123"

	var cdnCalls int
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cdnCalls++
		if r.URL.Path != attachmentPath {
			t.Errorf("cdn received unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(attachmentBody))
	}))
	defer cdn.Close()

	var issueRequestAuth string
	githubAPI := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/octocat/hello-world/issues/42":
			issueRequestAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"title": "Test issue",
				"body": "<img src=\"http://github.localhost` + attachmentPath + `\" />",
				"user": {"login": "octocat"},
				"created_at": "2024-01-01T00:00:00Z",
				"html_url": "http://github.localhost/octocat/hello-world/issues/42",
				"closed_at": null
			}`))
		case "/repos/octocat/hello-world/issues/42/timeline":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[]`))
		case attachmentPath:
			http.Redirect(w, r, cdn.URL+attachmentPath, http.StatusFound)
		default:
			t.Errorf("githubAPI received unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer githubAPI.Close()

	githubAPIURL, err := url.Parse(githubAPI.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", githubAPI.URL, err)
	}

	outputDir := t.TempDir()
	exporter, err := NewExportService(Config{
		Host:      "github.localhost",
		OutputDir: outputDir,
		Version:   "test-version",
		Commit:    "test-commit",
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
		t.Fatalf("NewExportService() error = %v", err)
	}

	ref, err := valueobjects.NewIssueRef("octocat", "hello-world", 42)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}

	skipped, err := exporter.Export(context.Background(), ref)
	if err != nil {
		t.Fatalf("Export() error = %v, want the cross-origin-redirected attachment fetch to succeed", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("Export() skipped = %v, want none", skipped)
	}
	if cdnCalls != 1 {
		t.Fatalf("cdn received %d calls, want 1", cdnCalls)
	}
	if issueRequestAuth != "token test-token" {
		t.Fatalf("Authorization header sent for the issue request = %q, want %q — Config.AuthToken must reach the real request", issueRequestAuth, "token test-token")
	}

	documentPath := filepath.Join(outputDir, "hello-world", "42", "index.md")
	document, err := os.ReadFile(documentPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", documentPath, err)
	}
	if !strings.Contains(string(document), `src="assets/abc-123.png"`) {
		t.Fatalf("rendered document = %s, want it to reference the downloaded attachment's local path", document)
	}

	assetPath := filepath.Join(outputDir, "hello-world", "42", "assets", "abc-123.png")
	asset, err := os.ReadFile(assetPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", assetPath, err)
	}
	if string(asset) != attachmentBody {
		t.Fatalf("downloaded asset = %q, want %q", asset, attachmentBody)
	}

	errorLogPath := filepath.Join(outputDir, "hello-world", "42", "evidence", "fetch-errors.log")
	if _, err := os.Stat(errorLogPath); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want the attachment fetch to have no failure to log", errorLogPath, err)
	}
}
