package github

import (
	"net/http"
	"net/url"
	"testing"
)

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", raw, err)
	}
	return u
}

func TestRequestOrigin_ReturnsSchemeAndHostFromTheResponsesOwnRequest(t *testing.T) {
	resp := &http.Response{Request: &http.Request{URL: mustParseURL(t, "https://api.github.com/repos/octocat/hello-world/issues/42")}}

	got := requestOrigin(resp)

	want := "https://api.github.com"
	if got != want {
		t.Fatalf("requestOrigin() = %q, want %q", got, want)
	}
}

func TestRequestOrigin_ReturnsEmptyWhenRequestIsNil(t *testing.T) {
	resp := &http.Response{}

	if got := requestOrigin(resp); got != "" {
		t.Fatalf("requestOrigin() = %q, want empty", got)
	}
}

func TestRequestOrigin_ReturnsEmptyWhenRequestURLIsNil(t *testing.T) {
	resp := &http.Response{Request: &http.Request{}}

	if got := requestOrigin(resp); got != "" {
		t.Fatalf("requestOrigin() = %q, want empty", got)
	}
}

func TestValidatePaginationOrigin_AcceptsAMatchingOrigin(t *testing.T) {
	err := validatePaginationOrigin("https://api.github.com/repos/octocat/hello-world/issues/42?page=2", "https://api.github.com")
	if err != nil {
		t.Fatalf("validatePaginationOrigin() error = %v, want nil for a matching origin", err)
	}
}

func TestValidatePaginationOrigin_RejectsADifferentHost(t *testing.T) {
	err := validatePaginationOrigin("https://attacker.invalid/next", "https://api.github.com")
	if err == nil {
		t.Fatal("validatePaginationOrigin() error = nil, want an error for a different host")
	}
}

func TestValidatePaginationOrigin_RejectsADifferentScheme(t *testing.T) {
	err := validatePaginationOrigin("http://api.github.com/repos/octocat/hello-world/issues/42?page=2", "https://api.github.com")
	if err == nil {
		t.Fatal("validatePaginationOrigin() error = nil, want an error for a scheme downgrade even on the same host")
	}
}

func TestValidatePaginationOrigin_RejectsAnUnknownExpectedOrigin(t *testing.T) {
	err := validatePaginationOrigin("https://api.github.com/next", "")
	if err == nil {
		t.Fatal("validatePaginationOrigin() error = nil, want an error when the expected origin could not be determined (fail closed)")
	}
}

func TestValidatePaginationOrigin_RejectsAMalformedNextURL(t *testing.T) {
	err := validatePaginationOrigin("://not-a-valid-url", "https://api.github.com")
	if err == nil {
		t.Fatal("validatePaginationOrigin() error = nil, want an error for a malformed next-page URL")
	}
}
