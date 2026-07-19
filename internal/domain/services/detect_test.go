package services_test

import (
	"reflect"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

func TestDetect_FindsAMarkdownImageReference(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	markdown := []byte("before\n![alt](" + url + ")\nafter")

	got := services.Detect(markdown)

	want := []string{url}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Detect() = %v, want %v", got, want)
	}
}

func TestDetect_FindsAnHTMLImgTagReference(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	markdown := []byte(`<img width="1756" alt="Image" src="` + url + `" />`)

	got := services.Detect(markdown)

	want := []string{url}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Detect() = %v, want %v", got, want)
	}
}

func TestDetect_DeduplicatesARepeatedURLInFirstSeenOrder(t *testing.T) {
	first := "https://github.com/user-attachments/assets/00000000-0000-0000-0000-000000000001"
	second := "https://github.com/user-attachments/assets/00000000-0000-0000-0000-000000000002"
	markdown := []byte(first + "\n" + second + "\n" + first)

	got := services.Detect(markdown)

	want := []string{first, second}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Detect() = %v, want %v", got, want)
	}
}

func TestDetect_ReturnsNoURLsWhenNoneArePresent(t *testing.T) {
	got := services.Detect([]byte("just plain body text, no attachments here"))

	if len(got) != 0 {
		t.Fatalf("Detect() = %v, want empty", got)
	}
}

func TestDetect_IgnoresAGitHubURLThatIsNotAnAttachment(t *testing.T) {
	markdown := []byte("see https://github.com/connect0459/gh-exhibit/issues/5#issuecomment-123 for context")

	got := services.Detect(markdown)

	if len(got) != 0 {
		t.Fatalf("Detect() = %v, want empty (a non-attachment GitHub URL must not match)", got)
	}
}
