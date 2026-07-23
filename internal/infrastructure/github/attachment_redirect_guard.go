package github

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

// newAttachmentGuardTransport builds the *http.Transport
// NewAttachmentFetcher installs as its client's Transport when its caller
// leaves api.ClientOptions.Transport nil (always true in real usage,
// since neither cmd/gh-exhibit nor internal/registry ever sets it). It
// mirrors http.DefaultTransport's own settings except for DialContext,
// which is replaced by newAttachmentDialContext's SSRF-safe dial.
//
// This guard is a property of the concrete *http.Transport actually
// performing the raw dial, not of http.Client.CheckRedirect (which fires
// before any address is resolved) or a wrapping http.RoundTripper (which
// can observe a redirect's target hostname but not the address it
// resolves to at the moment of connecting) — see
// newAttachmentDialContext's own Godoc for why a hostname target must be
// resolved and validated at the exact point of dialing. Consequently, a
// caller-supplied Transport (as every test in this package substitutes,
// to point at a local fake server) bypasses this guard entirely, the
// same way it already bypasses go-gh's own default transport.
func newAttachmentGuardTransport() *http.Transport {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = newAttachmentDialContext(net.DefaultResolver, nil)
	return transport
}

// attachmentRedirectPinContextKey is the context.Value key for
// attachmentRedirectPin, threaded through by pinAttachmentRedirectHops.
type attachmentRedirectPinContextKey struct{}

// attachmentRedirectPin tracks, across every dial within one logical
// attachmentFetcher.Fetch call, whether the call's first dial has
// already happened. The first dial always targets the configured,
// trusted GitHub/GHES host (validated far upstream by
// services.NewAttachment's URL-shape check) and is deliberately never
// address-validated: a self-hosted GHES instance may legitimately sit on
// a private-network address. Every dial after the first belongs to a
// redirect hop — an attacker-influenceable Location header — and is
// validated by dialAttachmentRedirectHop before connecting. Mutated in
// place without synchronization: Go dials one hop of a redirect chain at
// a time, sequentially, within a single http.Client.Do call, and a
// request's Context (carrying this same pin) is preserved across every
// hop of that call.
type attachmentRedirectPin struct {
	seenFirstDial bool
}

// pinAttachmentRedirectHops returns a context carrying a fresh pin, to be
// passed into exactly one attachmentFetcher.Fetch call's
// http.NewRequestWithContext, so newAttachmentDialContext's returned
// function can tell that call's first dial apart from a later redirect
// hop's.
func pinAttachmentRedirectHops(ctx context.Context) context.Context {
	return context.WithValue(ctx, attachmentRedirectPinContextKey{}, &attachmentRedirectPin{})
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
// installs as *http.Transport.DialContext. dial defaults to a plain
// *net.Dialer's DialContext when nil.
//
// A dial whose context carries no pin (from pinAttachmentRedirectHops) —
// or whose pin has not yet seen a first dial — proceeds unvalidated,
// matching this function's contract for a call's first, trusted hop.
// Every later dial within the same pinned call is resolved and validated
// here, at the exact point of connecting, rather than earlier (e.g. in
// http.Client.CheckRedirect, which fires once per redirect but before any
// address is resolved) or via a separate lookup performed ahead of the
// real dial: either of those would check an address that may differ from
// the one actually connected to moments later — for a hostname target,
// each independent net.Resolver.LookupIPAddr call is free to return a
// different answer (e.g. from a misconfigured or malicious authoritative
// DNS server deliberately racing its own responses), the same
// "DNS-rebinding" technique that lets a naive check-then-dial design be
// bypassed. Resolving and validating right here, immediately before
// dialing the exact address just validated (with no second resolution in
// between), closes that gap structurally.
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
		pin, _ := ctx.Value(attachmentRedirectPinContextKey{}).(*attachmentRedirectPin)
		if pin == nil || !pin.seenFirstDial {
			if pin != nil {
				pin.seenFirstDial = true
			}
			return dial(ctx, network, addr)
		}
		return dialAttachmentRedirectHop(ctx, network, addr, resolver, dial)
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
