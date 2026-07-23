package github

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

// rejectRedirectToADisallowedTarget is installed as attachmentFetcher's
// http.Client.CheckRedirect. It refuses to follow a redirect whose
// Location names a loopback, link-local (including the 169.254.169.254
// cloud-metadata address every major cloud provider exposes), or
// private-network (RFC 1918 / IPv6 unique-local) literal IP address,
// while still allowing a redirect to any hostname or public IP —
// including a cross-origin one, which a real GitHub attachment fetch
// legitimately requires (see NewAttachmentFetcher's own Godoc for why no
// origin-pinning guard is installed here instead).
//
// This intentionally does not resolve a hostname target's DNS address to
// apply the same check: doing so here would only validate an address
// that may differ from the one the transport actually dials moments
// later (a TOCTOU gap), and is out of scope for this guard. A redirect
// to an arbitrary external, attacker-controlled hostname remains
// possible after this check — that residual risk is documented as an
// accepted, unmitigated trade-off in SECURITY.md; this guard closes only
// the narrower SSRF-into-internal-network edge of the gap.
//
// Setting CheckRedirect at all replaces net/http's own default (which
// only caps the redirect chain at 10 hops), so that cap is reimplemented
// here to preserve the existing bound.
func rejectRedirectToADisallowedTarget(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}
	if ip := net.ParseIP(req.URL.Hostname()); ip != nil && isDisallowedRedirectIP(ip) {
		return fmt.Errorf("attachment redirect to %s refused: %s is a loopback, link-local, or private-network address", req.URL, ip)
	}
	return nil
}

// isDisallowedRedirectIP reports whether ip falls in a range
// rejectRedirectToADisallowedTarget refuses to redirect an attachment
// fetch into: loopback, link-local unicast (covers the 169.254.169.254
// cloud-metadata address), RFC 1918 / IPv6 unique-local private ranges,
// or the unspecified address.
func isDisallowedRedirectIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() || ip.IsUnspecified()
}
