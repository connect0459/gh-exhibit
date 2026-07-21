package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// redirectOriginPinContextKey is the context.Value key redirectGuardTransport
// uses to find the origin pin threaded through by pinRedirectOrigin.
type redirectOriginPinContextKey struct{}

// redirectOriginPin holds the origin (scheme + host) of the first request
// issued for one logical call (one evidenceFetcher.fetchSingle call, or one
// page of evidenceFetcher.fetchPaginated), so every later request sharing
// the same pin — an HTTP redirect hop within that same *http.Client.Do
// call — can be checked against it. A pin is only ever read and written
// sequentially within the single logical call it was created for (Go's own
// redirect-following happens one hop at a time), so no synchronization is
// needed here.
type redirectOriginPin struct {
	origin string
}

// pinRedirectOrigin returns a context carrying a fresh, empty origin pin,
// to be passed into exactly one *http.Client.Do call (directly, or via
// api.RESTClient.RequestWithContext) so redirectGuardTransport can pin that
// call's first request's origin and reject any later hop that leaves it.
// Each call to pinRedirectOrigin produces an independent pin: two logical
// calls sharing a parent context never interfere with each other.
func pinRedirectOrigin(ctx context.Context) context.Context {
	return context.WithValue(ctx, redirectOriginPinContextKey{}, &redirectOriginPin{})
}

// newRedirectGuardTransport wraps next with an origin-pinning redirect
// guard, defaulting to http.DefaultTransport when next is nil. Only
// NewEvidenceFetcher installs this — NewAttachmentFetcher deliberately does
// not, since a real attachment URL legitimately redirects cross-origin to
// serve its bytes (see attachment_fetcher.go's Godoc).
//
// This exists because api.RESTClient (used by evidenceFetcher) does not
// expose a way to set http.Client.CheckRedirect, the usual mechanism for
// rejecting a cross-origin redirect — its underlying *http.Client is
// unexported. Installing this as ClientOptions.Transport instead reaches
// the same effect from one layer below.
//
// A request's Context is preserved by net/http across every hop of a
// redirect chain within one Do call (confirmed directly: the same *http.Request
// context, and therefore the same pin, is visible to RoundTrip on every
// hop) — but two independent Do calls sharing an ancestor context (e.g. a
// caller's own ctx, wrapped separately by pinRedirectOrigin per call) get
// independent pins, so this guard only ever compares origins within a
// single logical request, never across separate ones. Cross-call
// consistency (e.g. one paginated fetch's page 2 matching page 1's origin)
// is a distinct concern handled at the application layer instead (see
// validatePaginationOrigin in pagination.go), since a test harness or a
// real caller may legitimately address separate calls through different
// literal URLs that still resolve to the same real destination.
func newRedirectGuardTransport(next http.RoundTripper) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &redirectGuardTransport{next: next}
}

type redirectGuardTransport struct {
	next http.RoundTripper
}

// RoundTrip implements http.RoundTripper. It fails closed: a request whose
// context carries no origin pin (meaning some call site forgot to call
// pinRedirectOrigin) is rejected rather than silently let through
// unguarded, matching this project's existing fail-closed precedent for an
// unknown expected origin (validatePaginationOrigin in pagination.go).
func (t *redirectGuardTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	pin, ok := req.Context().Value(redirectOriginPinContextKey{}).(*redirectOriginPin)
	if !ok {
		return nil, fmt.Errorf("refusing to send a request to %s with no origin pin in its context", req.URL)
	}

	origin := req.URL.Scheme + "://" + req.URL.Host
	if pin.origin == "" {
		pin.origin = origin
	} else if !strings.EqualFold(pin.origin, origin) {
		return nil, fmt.Errorf("refusing to follow a redirect from origin %q to a different origin %q", pin.origin, origin)
	}

	return t.next.RoundTrip(req)
}
