package github

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
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

func TestRejectRedirectToADisallowedTarget_RefusesALiteralLoopbackOrPrivateAddressTarget(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"loopback", "http://127.0.0.1/evil"},
		{"cloud metadata", "http://169.254.169.254/latest/meta-data/iam/security-credentials/"},
		{"RFC 1918 private", "http://192.168.1.1/evil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			if err := rejectRedirectToADisallowedTarget(req, nil); err == nil {
				t.Fatalf("rejectRedirectToADisallowedTarget(%q) error = nil, want a rejection error", tt.url)
			}
		})
	}
}

func TestRejectRedirectToADisallowedTarget_AllowsAnOrdinaryHostnameTarget(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://objects.githubusercontent.com/evidence", nil)
	if err := rejectRedirectToADisallowedTarget(req, nil); err != nil {
		t.Fatalf("rejectRedirectToADisallowedTarget() error = %v, want nil for an ordinary hostname target", err)
	}
}

func TestRejectRedirectToADisallowedTarget_StopsAfterTenRedirects(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://objects.githubusercontent.com/evidence", nil)
	via := make([]*http.Request, 10)
	if err := rejectRedirectToADisallowedTarget(req, via); err == nil {
		t.Fatal("rejectRedirectToADisallowedTarget() error = nil, want the redirect-count cap to still apply")
	}
}

func TestFetch_RefusesARedirectToACloudMetadataAddress(t *testing.T) {
	firstHop := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data/iam/security-credentials/", http.StatusFound)
	}))
	defer firstHop.Close()

	u, err := url.Parse(firstHop.URL)
	if err != nil {
		t.Fatalf("url.Parse(%q) error = %v", firstHop.URL, err)
	}

	fetcher, err := NewAttachmentFetcher(api.ClientOptions{
		Host:      "github.localhost",
		AuthToken: "test-token",
		Transport: &rewriteTransport{target: u.Host},
	})
	if err != nil {
		t.Fatalf("NewAttachmentFetcher() error = %v", err)
	}

	attachment, err := services.NewAttachment("http://github.localhost/user-attachments/assets/abc-123")
	if err != nil {
		t.Fatalf("NewAttachment() error = %v", err)
	}

	_, _, err = fetcher.Fetch(context.Background(), attachment)
	if err == nil {
		t.Fatal("Fetch() error = nil, want a redirect into a cloud-metadata address to be refused")
	}
}
