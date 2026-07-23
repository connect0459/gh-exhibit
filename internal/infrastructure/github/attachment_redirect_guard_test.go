package github

import (
	"context"
	"errors"
	"fmt"
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
	_, err := dialAttachmentRedirectHop(context.Background(), "127.0.0.1:443", "127.0.0.1:443", &fakeIPAddrResolver{}, dial.dial)
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

func TestNewAttachmentDialContext_DialsTheFirstHopUnvalidated(t *testing.T) {
	resolver := &fakeIPAddrResolver{err: errors.New("resolver must not be called for the first, unpinned-as-seen hop")}
	dial := &recordingDial{}
	dialContext := newAttachmentDialContext(resolver, dial.dial)

	ctx := pinAttachmentRedirectHops(context.Background())
	// A loopback target would be rejected on a redirect hop, but the
	// first dial within a pinned call is never validated: it always
	// targets the configured, trusted GitHub/GHES host, which may
	// legitimately be a private-network address for a self-hosted GHES
	// instance.
	if _, err := dialContext(ctx, "tcp", "127.0.0.1:443"); err != nil {
		t.Fatalf("dialContext() error = %v, want the first hop to be dialed unvalidated", err)
	}
	if want := []string{"127.0.0.1:443"}; len(dial.dialed) != 1 || dial.dialed[0] != want[0] {
		t.Fatalf("dial called with %v, want %v", dial.dialed, want)
	}
}

func TestNewAttachmentDialContext_ValidatesEveryDialAfterTheFirst(t *testing.T) {
	resolver := &fakeIPAddrResolver{ips: map[string][]net.IPAddr{
		"evil.example": ipAddrs("169.254.169.254"),
	}}
	dial := &recordingDial{}
	dialContext := newAttachmentDialContext(resolver, dial.dial)

	ctx := pinAttachmentRedirectHops(context.Background())
	if _, err := dialContext(ctx, "tcp", "github.localhost:443"); err != nil {
		t.Fatalf("dialContext() first hop error = %v, want nil", err)
	}
	if _, err := dialContext(ctx, "tcp", "evil.example:443"); err == nil {
		t.Fatal("dialContext() second hop error = nil, want a rejection for a redirect resolving to a cloud-metadata address")
	}
	if want := []string{"github.localhost:443"}; len(dial.dialed) != 1 || dial.dialed[0] != want[0] {
		t.Fatalf("dial calls = %v, want only the first hop to have been dialed", dial.dialed)
	}
}

func TestNewAttachmentGuardTransport_DialsAnUnpinnedRequestNormally(t *testing.T) {
	transport := newAttachmentGuardTransport()
	if transport.DialContext == nil {
		t.Fatal("newAttachmentGuardTransport().DialContext = nil, want the SSRF-safe dial context installed")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := &http.Client{Transport: transport}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("client.Get() error = %v, want a request with no redirect-hop pin to dial normally", err)
	}
	defer func() { _ = resp.Body.Close() }()
}

func TestNewAttachmentDialContext_DialsUnvalidatedWhenContextCarriesNoPin(t *testing.T) {
	resolver := &fakeIPAddrResolver{err: errors.New("resolver must not be called without a pin")}
	dial := &recordingDial{}
	dialContext := newAttachmentDialContext(resolver, dial.dial)

	if _, err := dialContext(context.Background(), "tcp", "127.0.0.1:443"); err != nil {
		t.Fatalf("dialContext() error = %v, want an unpinned context to dial unvalidated", err)
	}
}
