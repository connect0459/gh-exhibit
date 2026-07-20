package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

func TestRewrite_SubstitutesADownloadedURLWithItsLocalPath(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	markdown := []byte("![alt](" + url + ")")
	resolutions := []services.Resolution{
		mustDownloaded(t, url, "./5/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f.png"),
	}

	got := services.Rewrite(markdown, resolutions)

	want := "![alt](./5/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f.png)"
	if string(got) != want {
		t.Fatalf("Rewrite() = %q, want %q", got, want)
	}
}

func TestRewrite_SubstitutesAFailedURLWithAPlaceholderNotingTheReason(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	markdown := []byte("![alt](" + url + ")")
	resolutions := []services.Resolution{
		mustFetchFailed(t, url, "404 Not Found"),
	}

	got := services.Rewrite(markdown, resolutions)

	want := "![alt](" + url + " (attachment unavailable: 404 Not Found))"
	if string(got) != want {
		t.Fatalf("Rewrite() = %q, want %q", got, want)
	}
}

func TestRewrite_LeavesAnUnresolvedURLUntouched(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	markdown := []byte("![alt](" + url + ")")

	got := services.Rewrite(markdown, nil)

	if string(got) != string(markdown) {
		t.Fatalf("Rewrite() = %q, want unchanged %q", got, markdown)
	}
}

func TestRewrite_ReplacesEveryOccurrenceOfARepeatedURL(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	markdown := []byte(url + " " + url)
	resolutions := []services.Resolution{
		mustDownloaded(t, url, "./5/assets/x.png"),
	}

	got := services.Rewrite(markdown, resolutions)

	want := "./5/assets/x.png ./5/assets/x.png"
	if string(got) != want {
		t.Fatalf("Rewrite() = %q, want %q", got, want)
	}
}

func TestRewrite_DistinguishesAFailedFetchWithAnEmptyReasonFromASuccess(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	markdown := []byte("![alt](" + url + ")")
	resolutions := []services.Resolution{
		mustFetchFailed(t, url, ""),
	}

	got := services.Rewrite(markdown, resolutions)

	want := "![alt](" + url + " (attachment unavailable: ))"
	if string(got) != want {
		t.Fatalf("Rewrite() = %q, want %q (an empty reason must still render as a failure placeholder, not a silently-emptied reference)", got, want)
	}
}

func TestRewrite_ResolvesMultipleDistinctURLsInASinglePass(t *testing.T) {
	first := "https://github.com/user-attachments/assets/00000000-0000-0000-0000-000000000001"
	second := "https://github.com/user-attachments/assets/00000000-0000-0000-0000-000000000002"
	markdown := []byte(first + " " + second)
	resolutions := []services.Resolution{
		mustDownloaded(t, first, "./5/assets/a.png"),
		mustFetchFailed(t, second, "timeout"),
	}

	got := services.Rewrite(markdown, resolutions)

	want := "./5/assets/a.png " + second + " (attachment unavailable: timeout)"
	if string(got) != want {
		t.Fatalf("Rewrite() = %q, want %q", got, want)
	}
}
