package github

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/services"
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

func newTestAttachment(t *testing.T, url string) services.Attachment {
	t.Helper()

	attachment, err := services.NewAttachment(url)
	if err != nil {
		t.Fatalf("NewAttachment(%q) error = %v", url, err)
	}
	return attachment
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
	attachment := newTestAttachment(t, "http://github.localhost/user-attachments/assets/abc-123")
	data, contentType, err := fetcher.Fetch(context.Background(), attachment)
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
	attachment := newTestAttachment(t, "http://github.localhost/user-attachments/assets/abc-123")
	if _, _, err := fetcher.Fetch(context.Background(), attachment); err != nil {
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
	attachment := newTestAttachment(t, "http://github.localhost/user-attachments/assets/missing")
	_, _, err := fetcher.Fetch(context.Background(), attachment)
	if err == nil {
		t.Fatal("Fetch() error = nil, want an error for a 404 response")
	}
}

func TestFetch_ReturnsAnErrorWhenTheResponseBodyExceedsTheSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("0123456789"))
	}))
	defer server.Close()

	fetcher := newTestAttachmentFetcher(t, server).(*attachmentFetcher)
	fetcher.maxBytes = 5

	attachment := newTestAttachment(t, "http://github.localhost/user-attachments/assets/abc-123")
	_, _, err := fetcher.Fetch(context.Background(), attachment)
	if err == nil {
		t.Fatal("Fetch() error = nil, want an error for a response body exceeding the size limit")
	}
	if !strings.Contains(err.Error(), "exceeds the 5-byte size limit") {
		t.Fatalf("Fetch() error = %v, want it to mention the size-limit violation", err)
	}
}

func TestFetch_AcceptsAResponseBodyExactlyAtTheSizeLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("01234"))
	}))
	defer server.Close()

	fetcher := newTestAttachmentFetcher(t, server).(*attachmentFetcher)
	fetcher.maxBytes = 5

	attachment := newTestAttachment(t, "http://github.localhost/user-attachments/assets/abc-123")
	data, _, err := fetcher.Fetch(context.Background(), attachment)
	if err != nil {
		t.Fatalf("Fetch() error = %v, want nil for a response body exactly at the size limit", err)
	}
	if string(data) != "01234" {
		t.Fatalf("Fetch() data = %q, want %q", data, "01234")
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

	attachment := newTestAttachment(t, "http://github.localhost/user-attachments/assets/abc-123")
	_, _, err := fetcher.Fetch(ctx, attachment)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Fetch() error = %v, want context.Canceled", err)
	}
}
