package github

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

// stubTransport returns resps in order, recording every request it's asked
// to round-trip so a test can assert whether next was ever reached.
type stubTransport struct {
	resps   []*http.Response
	n       int
	reqURLs []string
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.reqURLs = append(s.reqURLs, req.URL.String())
	resp := s.resps[s.n]
	s.n++
	return resp, nil
}

func newOKResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}
}

func mustGetRequest(t *testing.T, rawURL string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		t.Fatalf("http.NewRequest(%q) error = %v", rawURL, err)
	}
	return req
}

func TestRedirectGuardTransport_RejectsARequestWhoseContextHasNoPin(t *testing.T) {
	next := &stubTransport{resps: []*http.Response{newOKResponse()}}
	transport := newRedirectGuardTransport(next)

	_, err := transport.RoundTrip(mustGetRequest(t, "https://api.github.com/repos/octocat/hello-world/issues/42"))

	if err == nil {
		t.Fatal("RoundTrip() error = nil, want an error for a request with no origin pin (fail closed)")
	}
	if len(next.reqURLs) != 0 {
		t.Fatalf("next.RoundTrip called %d times, want 0 (the request must never reach the real transport without a pin)", len(next.reqURLs))
	}
}

func TestRedirectGuardTransport_PinsTheFirstRequestsOriginAndAllowsAMatchingSubsequentOrigin(t *testing.T) {
	next := &stubTransport{resps: []*http.Response{newOKResponse(), newOKResponse()}}
	transport := newRedirectGuardTransport(next)
	ctx := pinRedirectOrigin(context.Background())

	first := mustGetRequest(t, "https://api.github.com/repos/octocat/hello-world/issues/42").WithContext(ctx)
	if _, err := transport.RoundTrip(first); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil for the first request in a pinned chain", err)
	}

	second := mustGetRequest(t, "https://api.github.com/repos/octocat/hello-world/issues/42?page=2").WithContext(ctx)
	if _, err := transport.RoundTrip(second); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil for a same-origin second request", err)
	}

	if len(next.reqURLs) != 2 {
		t.Fatalf("next.RoundTrip called %d times, want 2", len(next.reqURLs))
	}
}

func TestRedirectGuardTransport_RejectsASubsequentRequestToADifferentOrigin(t *testing.T) {
	next := &stubTransport{resps: []*http.Response{newOKResponse(), newOKResponse()}}
	transport := newRedirectGuardTransport(next)
	ctx := pinRedirectOrigin(context.Background())

	first := mustGetRequest(t, "https://api.github.com/repos/octocat/hello-world/issues/42").WithContext(ctx)
	if _, err := transport.RoundTrip(first); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil for the first request", err)
	}

	redirected := mustGetRequest(t, "https://attacker.invalid/repos/octocat/hello-world/issues/42").WithContext(ctx)
	_, err := transport.RoundTrip(redirected)

	if err == nil {
		t.Fatal("RoundTrip() error = nil, want an error for a request whose origin differs from the pinned one")
	}
	if len(next.reqURLs) != 1 {
		t.Fatalf("next.RoundTrip called %d times, want 1 (the mismatched request must never reach the real transport)", len(next.reqURLs))
	}
}

func TestRedirectGuardTransport_AllowsASubsequentOriginDifferingOnlyByCase(t *testing.T) {
	next := &stubTransport{resps: []*http.Response{newOKResponse(), newOKResponse()}}
	transport := newRedirectGuardTransport(next)
	ctx := pinRedirectOrigin(context.Background())

	first := mustGetRequest(t, "https://api.github.com/repos/octocat/hello-world/issues/42").WithContext(ctx)
	if _, err := transport.RoundTrip(first); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil for the first request", err)
	}

	sameOriginDifferentCase := mustGetRequest(t, "https://API.GitHub.com/repos/octocat/hello-world/issues/42?page=2").WithContext(ctx)
	if _, err := transport.RoundTrip(sameOriginDifferentCase); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil — scheme and host are case-insensitive", err)
	}
}

func TestRedirectGuardTransport_DefaultsToHTTPDefaultTransportWhenGivenNoNext(t *testing.T) {
	transport := newRedirectGuardTransport(nil)

	guard, ok := transport.(*redirectGuardTransport)
	if !ok {
		t.Fatalf("newRedirectGuardTransport(nil) = %T, want *redirectGuardTransport", transport)
	}
	if guard.next != http.DefaultTransport {
		t.Fatalf("newRedirectGuardTransport(nil).next = %v, want http.DefaultTransport", guard.next)
	}
}

func TestPinRedirectOrigin_EachCallProducesAnIndependentPin(t *testing.T) {
	next := &stubTransport{resps: []*http.Response{newOKResponse(), newOKResponse()}}
	transport := newRedirectGuardTransport(next)

	ctxA := pinRedirectOrigin(context.Background())
	ctxB := pinRedirectOrigin(context.Background())

	reqA := mustGetRequest(t, "https://a.example.com/x").WithContext(ctxA)
	if _, err := transport.RoundTrip(reqA); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil for ctxA's first request", err)
	}

	reqB := mustGetRequest(t, "https://b.example.com/y").WithContext(ctxB)
	if _, err := transport.RoundTrip(reqB); err != nil {
		t.Fatalf("RoundTrip() error = %v, want nil — ctxB is an independent pin from ctxA, unaffected by ctxA's origin", err)
	}
}

func TestRedirectGuardTransport_ReturnsNextsErrorUnchangedOnAPinnedRequest(t *testing.T) {
	next := &erroringTransport{err: errors.New("network unreachable")}
	transport := newRedirectGuardTransport(next)
	ctx := pinRedirectOrigin(context.Background())

	_, err := transport.RoundTrip(mustGetRequest(t, "https://api.github.com/x").WithContext(ctx))

	if !errors.Is(err, next.err) {
		t.Fatalf("RoundTrip() error = %v, want the underlying transport's own error", err)
	}
}

type erroringTransport struct{ err error }

func (e *erroringTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, e.err
}
