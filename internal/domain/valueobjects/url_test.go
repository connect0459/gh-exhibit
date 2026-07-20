package valueobjects_test

import (
	"encoding/json"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func newTestUrl(t *testing.T, raw string) valueobjects.Url {
	t.Helper()

	u, err := valueobjects.NewUrl(raw)
	if err != nil {
		t.Fatalf("NewUrl(%q) error = %v", raw, err)
	}
	return u
}

func TestNewUrl_RejectsAnEmptyURL(t *testing.T) {
	if _, err := valueobjects.NewUrl(""); err == nil {
		t.Fatal("NewUrl(\"\") error = nil, want an error for an empty url")
	}
}

func TestNewUrl_RejectsARelativeReference(t *testing.T) {
	if _, err := valueobjects.NewUrl("not-a-url"); err == nil {
		t.Fatal(`NewUrl("not-a-url") error = nil, want an error for a relative reference`)
	}
}

func TestNewUrl_RejectsASchemeOtherThanHTTPOrHTTPS(t *testing.T) {
	if _, err := valueobjects.NewUrl("ftp://example.com/file"); err == nil {
		t.Fatal(`NewUrl("ftp://...") error = nil, want an error for a non-http(s) scheme`)
	}
}

func TestNewUrl_RejectsAnAbsoluteURLWithNoHost(t *testing.T) {
	if _, err := valueobjects.NewUrl("https:///path"); err == nil {
		t.Fatal(`NewUrl("https:///path") error = nil, want an error for an absolute url with no host`)
	}
}

func TestNewUrl_RejectsAMalformedURL(t *testing.T) {
	if _, err := valueobjects.NewUrl("https://[::1]:namedport"); err == nil {
		t.Fatal("NewUrl() error = nil, want an error for an unparsable url")
	}
}

func TestNewUrl_AcceptsAnHTTPURL(t *testing.T) {
	if _, err := valueobjects.NewUrl("http://github.localhost/user-attachments/assets/abc-123"); err != nil {
		t.Fatalf("NewUrl() error = %v, want nil for a well-formed http url", err)
	}
}

func TestUrl_StringReturnsTheURLItWasConstructedFrom(t *testing.T) {
	raw := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	u := newTestUrl(t, raw)

	if got := u.String(); got != raw {
		t.Fatalf("String() = %q, want %q", got, raw)
	}
}

func TestUrl_SchemeReturnsTheURLsScheme(t *testing.T) {
	u := newTestUrl(t, "https://github.com/user-attachments/assets/abc-123")

	if got := u.Scheme(); got != "https" {
		t.Fatalf("Scheme() = %q, want %q", got, "https")
	}
}

func TestUrl_HostReturnsTheURLsHost(t *testing.T) {
	u := newTestUrl(t, "https://github.com/user-attachments/assets/abc-123")

	if got := u.Host(); got != "github.com" {
		t.Fatalf("Host() = %q, want %q", got, "github.com")
	}
}

func TestUrl_PathReturnsTheURLsPath(t *testing.T) {
	u := newTestUrl(t, "https://github.com/user-attachments/assets/abc-123")

	if got := u.Path(); got != "/user-attachments/assets/abc-123" {
		t.Fatalf("Path() = %q, want %q", got, "/user-attachments/assets/abc-123")
	}
}

func TestUrl_Equals_TreatsMatchingValuesAsEqual(t *testing.T) {
	a := newTestUrl(t, "https://github.com/example/repo/issues/1")
	b := newTestUrl(t, "https://github.com/example/repo/issues/1")

	if !a.Equals(b) {
		t.Fatal("expected urls constructed from the same raw string to be equal")
	}
}

func TestUrl_Equals_TreatsDifferentURLsAsNotEqual(t *testing.T) {
	a := newTestUrl(t, "https://github.com/example/repo/issues/1")
	b := newTestUrl(t, "https://github.com/example/repo/issues/2")

	if a.Equals(b) {
		t.Fatal("expected urls constructed from different raw strings to not be equal")
	}
}

func TestUrl_MarshalTextProducesTheSameJSONStringAsARawString(t *testing.T) {
	raw := "https://github.com/example/repo/issues/1"
	u := newTestUrl(t, raw)

	got, err := json.Marshal(u)
	if err != nil {
		t.Fatalf("json.Marshal(Url) error = %v", err)
	}

	want, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("json.Marshal(string) error = %v", err)
	}

	if string(got) != string(want) {
		t.Fatalf("json.Marshal(Url) = %s, want %s (byte-identical to marshaling the raw string)", got, want)
	}
}
