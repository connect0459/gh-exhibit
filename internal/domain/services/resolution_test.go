package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func TestResolution_SubstituteReturnsTheLocalPathForADownloadedAttachment(t *testing.T) {
	res := mustDownloaded(t, "https://github.com/user-attachments/assets/abc-123", "1/assets/abc-123.png")

	got := res.Substitute()

	want := "1/assets/abc-123.png"
	if got != want {
		t.Fatalf("Substitute() = %q, want %q", got, want)
	}
}

func TestResolution_SubstituteReturnsAPlaceholderNotingTheReasonForAFailedFetch(t *testing.T) {
	url := "https://github.com/user-attachments/assets/abc-123"
	res := mustFetchFailed(t, url, "404 Not Found")

	got := res.Substitute()

	want := url + " (attachment unavailable: 404 Not Found)"
	if got != want {
		t.Fatalf("Substitute() = %q, want %q", got, want)
	}
}

func TestResolution_SubstituteDistinguishesAFailedFetchWithAnEmptyReasonFromASuccess(t *testing.T) {
	url := "https://github.com/user-attachments/assets/abc-123"
	res := mustFetchFailed(t, url, "")

	got := res.Substitute()

	want := url + " (attachment unavailable: )"
	if got != want {
		t.Fatalf("Substitute() = %q, want %q (an empty reason must still render as a failure placeholder, not the bare URL)", got, want)
	}
}

// mustDownloaded builds a Resolution via Downloaded, failing t immediately
// if url fails to parse — for tests exercising some other behavior of
// Resolution, not url parsing itself (see valueobjects.NewUrl's own tests
// for that).
func mustDownloaded(t *testing.T, rawURL, localPath string) services.Resolution {
	t.Helper()

	url, err := valueobjects.NewUrl(rawURL)
	if err != nil {
		t.Fatalf("NewUrl(%q) error = %v", rawURL, err)
	}
	return services.Downloaded(url, localPath)
}

// mustFetchFailed builds a Resolution via FetchFailed, failing t immediately
// if url fails to parse — for tests exercising some other behavior of
// Resolution, not url parsing itself (see valueobjects.NewUrl's own tests
// for that).
func mustFetchFailed(t *testing.T, rawURL, reason string) services.Resolution {
	t.Helper()

	url, err := valueobjects.NewUrl(rawURL)
	if err != nil {
		t.Fatalf("NewUrl(%q) error = %v", rawURL, err)
	}
	return services.FetchFailed(url, reason)
}
