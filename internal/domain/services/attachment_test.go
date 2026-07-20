package services_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
)

func TestNewAttachment_URLReturnsTheURLItWasConstructedFrom(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"

	attachment, err := services.NewAttachment(url)
	if err != nil {
		t.Fatalf("NewAttachment(%q) error = %v", url, err)
	}

	if got := attachment.URL().String(); got != url {
		t.Fatalf("URL() = %q, want %q", got, url)
	}
}

func TestNewAttachment_RejectsAnEmptyURL(t *testing.T) {
	if _, err := services.NewAttachment(""); err == nil {
		t.Fatal("NewAttachment(\"\") error = nil, want an error for an empty url")
	}
}

func TestNewAttachment_RejectsAURLThatIsNotAGitHubAttachmentPath(t *testing.T) {
	if _, err := services.NewAttachment("https://github.com"); err == nil {
		t.Fatal("NewAttachment(\"https://github.com\") error = nil, want an error for a url that is not a user-attachments asset path")
	}
}

// newTestAttachment builds an Attachment from url, failing t immediately if
// construction errors — for tests exercising some other behavior of
// Attachment, not NewAttachment's own construction.
func newTestAttachment(t *testing.T, url string) services.Attachment {
	t.Helper()

	attachment, err := services.NewAttachment(url)
	if err != nil {
		t.Fatalf("NewAttachment(%q) error = %v", url, err)
	}
	return attachment
}
