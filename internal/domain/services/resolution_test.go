package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

func TestResolution_SubstituteReturnsTheLocalPathForADownloadedAttachment(t *testing.T) {
	res := services.Downloaded("https://github.com/user-attachments/assets/abc-123", "1/assets/abc-123.png")

	got := res.Substitute()

	want := "1/assets/abc-123.png"
	if got != want {
		t.Fatalf("Substitute() = %q, want %q", got, want)
	}
}

func TestResolution_SubstituteReturnsAPlaceholderNotingTheReasonForAFailedFetch(t *testing.T) {
	url := "https://github.com/user-attachments/assets/abc-123"
	res := services.FetchFailed(url, "404 Not Found")

	got := res.Substitute()

	want := url + " (attachment unavailable: 404 Not Found)"
	if got != want {
		t.Fatalf("Substitute() = %q, want %q", got, want)
	}
}

func TestResolution_SubstituteDistinguishesAFailedFetchWithAnEmptyReasonFromASuccess(t *testing.T) {
	url := "https://github.com/user-attachments/assets/abc-123"
	res := services.FetchFailed(url, "")

	got := res.Substitute()

	want := url + " (attachment unavailable: )"
	if got != want {
		t.Fatalf("Substitute() = %q, want %q (an empty reason must still render as a failure placeholder, not the bare URL)", got, want)
	}
}
