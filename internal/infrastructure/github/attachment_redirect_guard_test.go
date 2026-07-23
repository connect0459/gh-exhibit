package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsDisallowedRedirectIP_RejectsLoopbackLinkLocalPrivateAndUnspecifiedAddresses(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"IPv4 loopback", "127.0.0.1", true},
		{"IPv6 loopback", "::1", true},
		{"AWS/GCP/Azure cloud-metadata address", "169.254.169.254", true},
		{"general IPv4 link-local", "169.254.1.1", true},
		{"IPv6 link-local", "fe80::1", true},
		{"RFC 1918 10.0.0.0/8", "10.0.0.5", true},
		{"RFC 1918 172.16.0.0/12", "172.16.0.1", true},
		{"RFC 1918 192.168.0.0/16", "192.168.1.1", true},
		{"IPv6 unique-local", "fc00::1", true},
		{"IPv4 unspecified", "0.0.0.0", true},
		{"public IPv4 address", "8.8.8.8", false},
		{"public IPv6 address", "2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("net.ParseIP(%q) = nil, want a valid IP", tt.ip)
			}
			if got := isDisallowedRedirectIP(ip); got != tt.want {
				t.Errorf("isDisallowedRedirectIP(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

// fakeIPAddrResolver is a test-only ipAddrResolver: it never touches real
// DNS, returning exactly the addresses (or error) configured for a given
// host.
type fakeIPAddrResolver struct {
	ips map[string][]net.IPAddr
	err error
}

func (r *fakeIPAddrResolver) LookupIPAddr(_ context.Context, host string) ([]net.IPAddr, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.ips[host], nil
}

// recordingDial is a test-only dialFunc: it records every address it was
// asked to dial and returns an in-memory net.Pipe() end instead of a real
// network connection.
type recordingDial struct {
	dialed []string
}

func (d *recordingDial) dial(_ context.Context, _, addr string) (net.Conn, error) {
	d.dialed = append(d.dialed, addr)
	conn, _ := net.Pipe()
	return conn, nil
}

func ipAddrs(ips ...string) []net.IPAddr {
	addrs := make([]net.IPAddr, len(ips))
	for i, ip := range ips {
		addrs[i] = net.IPAddr{IP: net.ParseIP(ip)}
	}
	return addrs
}

func TestDialAttachmentRedirectHop_RejectsALiteralDisallowedAddress(t *testing.T) {
	dial := &recordingDial{}
	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "127.0.0.1:443", &fakeIPAddrResolver{}, dial.dial)
	if err == nil {
		t.Fatal("dialAttachmentRedirectHop() error = nil, want a rejection for a literal loopback address")
	}
	if len(dial.dialed) != 0 {
		t.Fatalf("dial called with %v, want no dial attempt for a rejected target", dial.dialed)
	}
}

func TestDialAttachmentRedirectHop_RejectsAHostnameResolvingToADisallowedAddress(t *testing.T) {
	resolver := &fakeIPAddrResolver{ips: map[string][]net.IPAddr{
		"evil.example": ipAddrs("169.254.169.254"),
	}}
	dial := &recordingDial{}

	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "evil.example:443", resolver, dial.dial)
	if err == nil {
		t.Fatal("dialAttachmentRedirectHop() error = nil, want a rejection when the hostname resolves to a cloud-metadata address")
	}
	if len(dial.dialed) != 0 {
		t.Fatalf("dial called with %v, want no dial attempt for a rejected target", dial.dialed)
	}
}

func TestDialAttachmentRedirectHop_DialsTheResolvedAddressForAnOrdinaryHostname(t *testing.T) {
	resolver := &fakeIPAddrResolver{ips: map[string][]net.IPAddr{
		"cdn.example.test": ipAddrs("93.184.216.34"),
	}}
	dial := &recordingDial{}

	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "cdn.example.test:443", resolver, dial.dial)
	if err != nil {
		t.Fatalf("dialAttachmentRedirectHop() error = %v, want nil for a hostname resolving to a public address", err)
	}
	if want := []string{"93.184.216.34:443"}; len(dial.dialed) != 1 || dial.dialed[0] != want[0] {
		t.Fatalf("dial called with %v, want %v — the resolved address, not the original hostname", dial.dialed, want)
	}
}

func TestDialAttachmentRedirectHop_SkipsResolutionForALiteralPublicIP(t *testing.T) {
	resolver := &fakeIPAddrResolver{err: errors.New("resolver must not be called for a literal IP target")}
	dial := &recordingDial{}

	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "93.184.216.34:443", resolver, dial.dial)
	if err != nil {
		t.Fatalf("dialAttachmentRedirectHop() error = %v, want nil", err)
	}
	if want := []string{"93.184.216.34:443"}; len(dial.dialed) != 1 || dial.dialed[0] != want[0] {
		t.Fatalf("dial called with %v, want %v", dial.dialed, want)
	}
}

func TestDialAttachmentRedirectHop_RejectsAMalformedDialAddress(t *testing.T) {
	dial := &recordingDial{}
	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "no-port-here", &fakeIPAddrResolver{}, dial.dial)
	if err == nil {
		t.Fatal("dialAttachmentRedirectHop() error = nil, want an error for an address with no port")
	}
	if len(dial.dialed) != 0 {
		t.Fatalf("dial called with %v, want no dial attempt for a malformed address", dial.dialed)
	}
}

func TestDialAttachmentRedirectHop_ReturnsTheUnderlyingErrorWhenEveryResolvedAddressFailsToDial(t *testing.T) {
	resolver := &fakeIPAddrResolver{ips: map[string][]net.IPAddr{
		"unreachable.example": ipAddrs("93.184.216.34"),
	}}
	failingDial := func(_ context.Context, _, addr string) (net.Conn, error) {
		return nil, fmt.Errorf("connection refused: %s", addr)
	}

	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "unreachable.example:443", resolver, failingDial)
	if err == nil {
		t.Fatal("dialAttachmentRedirectHop() error = nil, want the underlying dial error to be returned")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Fatalf("dialAttachmentRedirectHop() error = %v, want it to wrap the underlying dial failure", err)
	}
}

func TestDialAttachmentRedirectHop_RejectsWhenResolutionFails(t *testing.T) {
	resolver := &fakeIPAddrResolver{err: errors.New("no such host")}
	dial := &recordingDial{}

	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "unresolvable.example:443", resolver, dial.dial)
	if err == nil {
		t.Fatal("dialAttachmentRedirectHop() error = nil, want an error when resolution fails")
	}
	if len(dial.dialed) != 0 {
		t.Fatalf("dial called with %v, want no dial attempt when resolution fails", dial.dialed)
	}
}

func TestDialAttachmentRedirectHop_RejectsWhenResolutionReturnsNoAddresses(t *testing.T) {
	resolver := &fakeIPAddrResolver{ips: map[string][]net.IPAddr{}}
	dial := &recordingDial{}

	_, err := dialAttachmentRedirectHop(context.Background(), "tcp", "no-records.example:443", resolver, dial.dial)
	if err == nil {
		t.Fatal("dialAttachmentRedirectHop() error = nil, want an error when resolution returns no addresses")
	}
	if len(dial.dialed) != 0 {
		t.Fatalf("dial called with %v, want no dial attempt when resolution returns no addresses", dial.dialed)
	}
}

func TestNewAttachmentDialContext_DialsUnvalidatedWhenContextIsNotMarkedAsARedirectHop(t *testing.T) {
	resolver := &fakeIPAddrResolver{err: errors.New("resolver must not be called for an unmarked dial")}
	dial := &recordingDial{}
	dialContext := newAttachmentDialContext(resolver, dial.dial)

	// An address that would be rejected on a marked redirect hop dials
	// fine when the context carries no attachmentRedirectHopContextKey
	// marker — matching a call's first hop, which may legitimately be a
	// private-network address for a self-hosted GHES instance.
	if _, err := dialContext(context.Background(), "tcp", "127.0.0.1:443"); err != nil {
		t.Fatalf("dialContext() error = %v, want an unmarked dial to proceed unvalidated", err)
	}
	if want := []string{"127.0.0.1:443"}; len(dial.dialed) != 1 || dial.dialed[0] != want[0] {
		t.Fatalf("dial called with %v, want %v", dial.dialed, want)
	}
}

func TestNewAttachmentDialContext_ValidatesWhenContextIsMarkedAsARedirectHop(t *testing.T) {
	resolver := &fakeIPAddrResolver{ips: map[string][]net.IPAddr{
		"evil.example": ipAddrs("169.254.169.254"),
	}}
	dial := &recordingDial{}
	dialContext := newAttachmentDialContext(resolver, dial.dial)

	ctx := context.WithValue(context.Background(), attachmentRedirectHopContextKey{}, true)
	if _, err := dialContext(ctx, "tcp", "evil.example:443"); err == nil {
		t.Fatal("dialContext() error = nil, want a marked redirect hop resolving to a cloud-metadata address to be refused")
	}
	if len(dial.dialed) != 0 {
		t.Fatalf("dial called with %v, want no dial attempt for a rejected marked hop", dial.dialed)
	}
}

// fakeRoundTripper is a test-only http.RoundTripper: it records the last
// request it received (specifically, whether attachmentGuardRoundTripper
// marked its context) and returns a canned response.
type fakeRoundTripper struct {
	lastReqMarked bool
	calls         int
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	f.calls++
	f.lastReqMarked, _ = req.Context().Value(attachmentRedirectHopContextKey{}).(bool)
	return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody, Header: http.Header{}}, nil
}

func TestAttachmentGuardRoundTripper_DoesNotMarkAPinnedCallsFirstHop(t *testing.T) {
	next := &fakeRoundTripper{}
	guard := &attachmentGuardRoundTripper{next: next}

	req, _ := http.NewRequestWithContext(pinAttachmentRedirectHops(context.Background()), http.MethodGet, "http://example.test", nil)
	if _, err := guard.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	if next.lastReqMarked {
		t.Fatal("first hop was marked as a redirect hop, want it left unmarked")
	}
}

func TestAttachmentGuardRoundTripper_MarksEveryHopAfterThePinnedCallsFirst(t *testing.T) {
	next := &fakeRoundTripper{}
	guard := &attachmentGuardRoundTripper{next: next}

	ctx := pinAttachmentRedirectHops(context.Background())
	firstReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	if _, err := guard.RoundTrip(firstReq); err != nil {
		t.Fatalf("RoundTrip() first hop error = %v", err)
	}
	if next.lastReqMarked {
		t.Fatal("first hop was marked as a redirect hop, want it left unmarked")
	}

	secondReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/redirected", nil)
	if _, err := guard.RoundTrip(secondReq); err != nil {
		t.Fatalf("RoundTrip() second hop error = %v", err)
	}
	if !next.lastReqMarked {
		t.Fatal("second hop was not marked as a redirect hop, want it marked for dial-time validation")
	}
}

func TestAttachmentGuardRoundTripper_NeverMarksAnUnpinnedRequest(t *testing.T) {
	next := &fakeRoundTripper{}
	guard := &attachmentGuardRoundTripper{next: next}

	for i := 0; i < 2; i++ {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test", nil)
		if _, err := guard.RoundTrip(req); err != nil {
			t.Fatalf("RoundTrip() call %d error = %v", i, err)
		}
		if next.lastReqMarked {
			t.Fatalf("call %d was marked as a redirect hop, want an unpinned request never marked", i)
		}
	}
}

// TestAttachmentGuardTransport_RejectsARedirectOnASecondFetchReusingAPooledConnection
// reproduces the exact regression a local review found in an earlier
// version of this guard: net/http.Transport reuses a pooled keep-alive
// connection for a second request to an already-open host without ever
// calling DialContext, so tracking "have I seen this pinned call's first
// hop" by counting DialContext's own invocations wrongly treats a
// pooled-connection Fetch's own redirect hop as if it were the trusted
// first hop, skipping validation on exactly the request that most needs
// it. This test exercises the real production types
// (attachmentGuardRoundTripper + newAttachmentDialContext) against two
// sequential requests to one real httptest.Server, draining and closing
// the first response's body (as attachment_fetcher.go's Fetch always
// does) so its connection is eligible for reuse, then confirms via an
// instrumented dial that only one real dial occurred (proving reuse
// actually happened) while the second request's redirect to a loopback
// address is still refused.
func TestAttachmentGuardTransport_RejectsARedirectOnASecondFetchReusingAPooledConnection(t *testing.T) {
	var evilHits int
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		evilHits++
		_, _ = w.Write([]byte("evil"))
	}))
	defer evil.Close()

	redirectOnSecondRequest := false
	trusted := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if redirectOnSecondRequest {
			http.Redirect(w, r, evil.URL, http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer trusted.Close()

	var dialCount int
	countingDial := func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialCount++
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	}
	baseTransport := http.DefaultTransport.(*http.Transport).Clone()
	baseTransport.DialContext = newAttachmentDialContext(net.DefaultResolver, countingDial)
	client := &http.Client{Transport: &attachmentGuardRoundTripper{next: baseTransport}}

	// First Fetch-equivalent call: an ordinary response, its body fully
	// drained and closed so the connection returns to the pool — exactly
	// what attachment_fetcher.go's Fetch does via io.ReadAll.
	ctx1 := pinAttachmentRedirectHops(context.Background())
	req1, _ := http.NewRequestWithContext(ctx1, http.MethodGet, trusted.URL, nil)
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("first Fetch-equivalent call error = %v", err)
	}
	_, _ = io.Copy(io.Discard, resp1.Body)
	_ = resp1.Body.Close()

	// Second Fetch-equivalent call to the same trusted host: its own
	// first hop is expected to reuse the pooled connection from above (no
	// DialContext call), and the trusted host now redirects to evil, a
	// loopback address that must still be refused.
	redirectOnSecondRequest = true
	ctx2 := pinAttachmentRedirectHops(context.Background())
	req2, _ := http.NewRequestWithContext(ctx2, http.MethodGet, trusted.URL, nil)
	if _, err := client.Do(req2); err == nil {
		t.Fatal("second Fetch-equivalent call error = nil, want the redirect to a loopback address to be refused even when the first hop reused a pooled connection")
	}
	if evilHits != 0 {
		t.Fatalf("evil server hit %d times, want 0 — the redirect should have been refused before ever connecting to it", evilHits)
	}
	if dialCount != 1 {
		t.Fatalf("real dials performed = %d, want exactly 1 — this test only proves what it claims if the second call's first hop actually reused a pooled connection", dialCount)
	}
}

func TestNewAttachmentGuardTransport_DialsAnUnpinnedRequestNormally(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := &http.Client{Transport: newAttachmentGuardTransport()}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("client.Get() error = %v, want a request with no redirect-hop pin to dial normally", err)
	}
	defer func() { _ = resp.Body.Close() }()
}
