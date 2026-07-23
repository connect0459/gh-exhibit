package github

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

// newAttachmentGuardTransport builds the http.RoundTripper
// NewAttachmentFetcher installs as its client's Transport when its caller
// leaves api.ClientOptions.Transport nil (always true in real usage,
// since neither cmd/gh-exhibit nor internal/registry ever sets it).
//
// This guard splits across two layers deliberately, not out of
// convenience: attachmentGuardRoundTripper tracks, once per logical HTTP
// request/hop, whether a call has already had its first hop — this must
// live at the RoundTripper layer (invoked exactly once per hop,
// regardless of what the underlying connection does) rather than be
// inferred from how many times DialContext itself has been called, since
// net/http.Transport reuses a pooled keep-alive connection for a request
// to an already-open host without ever calling DialContext. Because
// attachmentFetcher builds its *http.Client once and reuses it across
// every Fetch call, a second Fetch to the same trusted host routinely
// reuses the first Fetch's pooled connection — meaning DialContext
// itself is never invoked for that "first hop", so a dial-invocation-count
// based check would wrongly treat that request's own redirect hop (the
// next DialContext call actually made) as the trusted first hop instead,
// skipping validation on exactly the request that most needs it. Tracking
// at the RoundTrip layer, which fires once per hop unconditionally,
// avoids this miscount entirely.
//
// newAttachmentDialContext still performs the actual address resolution
// and validation, and it must still happen at the exact point of
// dialing (see its own Godoc) — attachmentGuardRoundTripper only decides
// whether a given hop needs that validation, by marking its request's
// Context before handing it to the wrapped Transport.
//
// A caller-supplied Transport (as every test in this package substitutes,
// to point at a local fake server) bypasses this guard entirely, the
// same way it already bypasses go-gh's own default transport.
func newAttachmentGuardTransport() http.RoundTripper {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = newAttachmentDialContext(net.DefaultResolver, nil)
	return &attachmentGuardRoundTripper{next: transport}
}

// attachmentRedirectPinContextKey is the context.Value key for
// attachmentRedirectPin, threaded through by pinAttachmentRedirectHops.
type attachmentRedirectPinContextKey struct{}

// attachmentRedirectPin tracks, across every hop of one logical
// attachmentFetcher.Fetch call, whether the call's first RoundTrip has
// already happened. The first hop always targets the configured,
// trusted GitHub/GHES host (validated far upstream by
// services.NewAttachment's URL-shape check) and is deliberately never
// address-validated: a self-hosted GHES instance may legitimately sit on
// a private-network address. Every hop after the first belongs to a
// redirect — an attacker-influenceable Location header — and its dial is
// validated. Mutated in place without synchronization: Go processes one
// hop of a redirect chain at a time, sequentially, within a single
// http.Client.Do call, and a request's Context (carrying this same pin)
// is preserved across every hop of that call.
type attachmentRedirectPin struct {
	seenFirstHop bool
}

// pinAttachmentRedirectHops returns a context carrying a fresh pin, to be
// passed into exactly one attachmentFetcher.Fetch call's
// http.NewRequestWithContext, so attachmentGuardRoundTripper can tell
// that call's first hop apart from a later redirect hop's.
func pinAttachmentRedirectHops(ctx context.Context) context.Context {
	return context.WithValue(ctx, attachmentRedirectPinContextKey{}, &attachmentRedirectPin{})
}

// attachmentRedirectHopContextKey is the context.Value key
// attachmentGuardRoundTripper sets on a redirect hop's request, for
// newAttachmentDialContext to read.
type attachmentRedirectHopContextKey struct{}

// attachmentGuardRoundTripper marks a redirect hop's request context
// (every hop after a pinned call's first) so newAttachmentDialContext
// knows to validate its dial, then delegates to next unconditionally —
// see newAttachmentGuardTransport's own Godoc for why this tracking must
// happen here rather than inside DialContext itself.
type attachmentGuardRoundTripper struct {
	next http.RoundTripper
}

func (t *attachmentGuardRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	pin, _ := req.Context().Value(attachmentRedirectPinContextKey{}).(*attachmentRedirectPin)
	if pin != nil {
		if pin.seenFirstHop {
			req = req.Clone(context.WithValue(req.Context(), attachmentRedirectHopContextKey{}, true))
		} else {
			pin.seenFirstHop = true
		}
	}
	return t.next.RoundTrip(req)
}

// ipAddrResolver is the subset of *net.Resolver newAttachmentDialContext
// needs; tests substitute a fake to control DNS resolution without
// depending on real DNS.
type ipAddrResolver interface {
	LookupIPAddr(ctx context.Context, host string) ([]net.IPAddr, error)
}

// dialFunc dials network/addr; tests substitute a fake to avoid a real
// network connection while still exercising newAttachmentDialContext's
// own resolve-then-validate logic.
type dialFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// newAttachmentDialContext returns the function newAttachmentGuardTransport
// installs as its wrapped *http.Transport's DialContext. dial defaults to
// a plain *net.Dialer's DialContext when nil.
//
// A dial whose context was not marked by attachmentGuardRoundTripper (an
// unpinned call's dial, or a pinned call's first hop) proceeds
// unvalidated. A dial whose context was marked (every hop after a pinned
// call's first) is resolved and validated here, at the exact point of
// connecting, rather than earlier (e.g. in http.Client.CheckRedirect,
// which fires once per redirect but before any address is resolved) or
// via a separate lookup performed ahead of the real dial: either of
// those would check an address that may differ from the one actually
// connected to moments later — for a hostname target, each independent
// net.Resolver.LookupIPAddr call is free to return a different answer
// (e.g. from a misconfigured or malicious authoritative DNS server
// deliberately racing its own responses), the same "DNS-rebinding"
// technique that lets a naive check-then-dial design be bypassed.
// Resolving and validating right here, immediately before dialing the
// exact address just validated (with no second resolution in between),
// closes that gap structurally.
//
// A target that is already a literal IP address skips resolution
// entirely. Any resolved (or literal) address that is loopback,
// link-local, or private-network (see isDisallowedRedirectIP) refuses
// the dial outright; otherwise the first address dial succeeds against
// is used.
func newAttachmentDialContext(resolver ipAddrResolver, dial dialFunc) func(ctx context.Context, network, addr string) (net.Conn, error) {
	if dial == nil {
		dial = (&net.Dialer{}).DialContext
	}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		if marked, _ := ctx.Value(attachmentRedirectHopContextKey{}).(bool); marked {
			return dialAttachmentRedirectHop(ctx, network, addr, resolver, dial)
		}
		return dial(ctx, network, addr)
	}
}

// dialAttachmentRedirectHop resolves addr's host (skipping resolution
// when it is already a literal IP address), refuses to proceed if any
// resolved address is loopback, link-local, or private-network, and
// otherwise dials the first address dial succeeds against.
func dialAttachmentRedirectHop(ctx context.Context, network, addr string, resolver ipAddrResolver, dial dialFunc) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("parse attachment redirect dial address %q: %w", addr, err)
	}

	var ips []net.IP
	if literal := net.ParseIP(host); literal != nil {
		ips = []net.IP{literal}
	} else {
		resolved, err := resolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, fmt.Errorf("resolve attachment redirect target %s: %w", host, err)
		}
		for _, a := range resolved {
			ips = append(ips, a.IP)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("resolve attachment redirect target %s: no addresses found", host)
	}
	for _, ip := range ips {
		if isDisallowedRedirectIP(ip) {
			return nil, fmt.Errorf("attachment redirect to %s refused: %s is a loopback, link-local, or private-network address", host, ip)
		}
	}

	var lastErr error
	for _, ip := range ips {
		conn, err := dial(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

// isDisallowedRedirectIP reports whether ip falls in a range
// dialAttachmentRedirectHop refuses to dial an attachment redirect into:
// loopback, link-local unicast (covers the 169.254.169.254
// cloud-metadata address), RFC 1918 / IPv6 unique-local private ranges,
// or the unspecified address.
func isDisallowedRedirectIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() || ip.IsUnspecified()
}
