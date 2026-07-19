package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
)

func newTestAttachmentFetcher(t *testing.T, server *httptest.Server) repositories.AttachmentFetcher {
	t.Helper()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", server.URL, err)
	}

	fetcher, err := NewAttachmentFetcher(api.ClientOptions{
		Host:      "github.localhost",
		AuthToken: "test-token",
		Transport: &rewriteTransport{target: u.Host},
	})
	if err != nil {
		t.Fatalf("NewAttachmentFetcher() error = %v", err)
	}
	return fetcher
}

func TestFetch_ReturnsResponseBodyAndContentTypeVerbatim(t *testing.T) {
	const body = "not-actually-a-png"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/user-attachments/assets/abc-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(body))
	}))
	defer server.Close()

	fetcher := newTestAttachmentFetcher(t, server)
	data, contentType, err := fetcher.Fetch(context.Background(), "http://github.localhost/user-attachments/assets/abc-123")
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if string(data) != body {
		t.Fatalf("Fetch() data = %q, want %q", data, body)
	}
	if contentType != "image/png" {
		t.Fatalf("Fetch() contentType = %q, want %q", contentType, "image/png")
	}
}

func TestFetch_SendsAuthorizationHeaderFromTheConfiguredToken(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	fetcher := newTestAttachmentFetcher(t, server)
	if _, _, err := fetcher.Fetch(context.Background(), "http://github.localhost/user-attachments/assets/abc-123"); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if gotAuth != "token test-token" {
		t.Fatalf("Authorization header = %q, want %q", gotAuth, "token test-token")
	}
}

func TestFetch_ReturnsAnErrorForANonSuccessStatusCode(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	fetcher := newTestAttachmentFetcher(t, server)
	_, _, err := fetcher.Fetch(context.Background(), "http://github.localhost/user-attachments/assets/missing")
	if err == nil {
		t.Fatal("Fetch() error = nil, want an error for a 404 response")
	}
}

func TestFetch_ReturnsContextErrorWhenContextIsAlreadyCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("data"))
	}))
	defer server.Close()

	fetcher := newTestAttachmentFetcher(t, server)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := fetcher.Fetch(ctx, "http://github.localhost/user-attachments/assets/abc-123")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context.Canceled", err)
	}
}
