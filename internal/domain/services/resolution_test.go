package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
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

func TestDownloaded_RejectsAnEmptyURL(t *testing.T) {
	if _, err := services.Downloaded("", "1/assets/abc-123.png"); err == nil {
		t.Fatal("Downloaded(\"\", ...) error = nil, want an error for an empty url")
	}
}

func TestFetchFailed_RejectsAnEmptyURL(t *testing.T) {
	if _, err := services.FetchFailed("", "404 Not Found"); err == nil {
		t.Fatal("FetchFailed(\"\", ...) error = nil, want an error for an empty url")
	}
}

// mustDownloaded builds a Resolution via Downloaded, failing t immediately
// if construction errors — for tests exercising some other behavior of
// Resolution, not Downloaded's own construction.
func mustDownloaded(t *testing.T, url, localPath string) services.Resolution {
	t.Helper()

	res, err := services.Downloaded(url, localPath)
	if err != nil {
		t.Fatalf("Downloaded(%q, %q) error = %v", url, localPath, err)
	}
	return res
}

// mustFetchFailed builds a Resolution via FetchFailed, failing t immediately
// if construction errors — for tests exercising some other behavior of
// Resolution, not FetchFailed's own construction.
func mustFetchFailed(t *testing.T, url, reason string) services.Resolution {
	t.Helper()

	res, err := services.FetchFailed(url, reason)
	if err != nil {
		t.Fatalf("FetchFailed(%q, %q) error = %v", url, reason, err)
	}
	return res
}
