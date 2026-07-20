package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

func TestNewAttachment_URLReturnsTheURLItWasConstructedFrom(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"

	got := services.NewAttachment(url).URL()

	if got != url {
		t.Fatalf("URL() = %q, want %q", got, url)
	}
}
