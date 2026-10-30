package services_test

import (
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

const rewriterAttachmentURL = "https://github.com/user-attachments/assets/abc-123"

func mustAttachmentLink(t *testing.T, rawURL, localPath string) valueobjects.AttachmentLink {
	t.Helper()
	u, err := valueobjects.NewUrl(rawURL)
	if err != nil {
		t.Fatalf("NewUrl(%q): unexpected error: %v", rawURL, err)
	}
	link, err := valueobjects.NewAttachmentLink(u, localPath)
	if err != nil {
		t.Fatalf("NewAttachmentLink(): unexpected error: %v", err)
	}
	return link
}

func TestNewAttachmentRewriter_ReplacesALinkedAttachmentURLWithItsLocalPath(t *testing.T) {
	rewrite := services.NewAttachmentRewriter("github.com", []valueobjects.AttachmentLink{
		mustAttachmentLink(t, rewriterAttachmentURL, "assets/abc-123.png"),
	})

	got := rewrite("see ![shot](" + rewriterAttachmentURL + ") here")

	if got != "see ![shot](assets/abc-123.png) here" {
		t.Fatalf("rewrite() = %q, want the URL replaced by its local path", got)
	}
}

func TestNewAttachmentRewriter_ReplacesAnHTMLImgSrcWithItsLocalPath(t *testing.T) {
	rewrite := services.NewAttachmentRewriter("github.com", []valueobjects.AttachmentLink{
		mustAttachmentLink(t, rewriterAttachmentURL, "assets/abc-123.png"),
	})

	got := rewrite(`<img src="` + rewriterAttachmentURL + `" width="100">`)

	if got != `<img src="assets/abc-123.png" width="100">` {
		t.Fatalf("rewrite() = %q, want the img src replaced by its local path", got)
	}
}

func TestNewAttachmentRewriter_ReplacesAnUnlinkedAttachmentURLWithAnUnavailablePlaceholder(t *testing.T) {
	rewrite := services.NewAttachmentRewriter("github.com", nil)

	got := rewrite("![shot](" + rewriterAttachmentURL + ")")

	if !strings.Contains(got, rewriterAttachmentURL) || !strings.Contains(got, "attachment unavailable") {
		t.Fatalf("rewrite() = %q, want a placeholder naming the URL instead of a hotlinked image", got)
	}
	if strings.Contains(got, "![shot]("+rewriterAttachmentURL+")") {
		t.Fatalf("rewrite() = %q, want the attachment no longer referenced as an image", got)
	}
}

func TestNewAttachmentRewriter_LeavesTextWithoutAttachmentsUnchanged(t *testing.T) {
	rewrite := services.NewAttachmentRewriter("github.com", nil)

	const text = "plain text with a [link](https://example.com/page)"

	if got := rewrite(text); got != text {
		t.Fatalf("rewrite() = %q, want %q", got, text)
	}
}

func TestNewAttachmentRewriter_IgnoresAnAttachmentOnADifferentHost(t *testing.T) {
	rewrite := services.NewAttachmentRewriter("github.example.com", nil)

	const text = "![shot](" + rewriterAttachmentURL + ")"

	if got := rewrite(text); got != text {
		t.Fatalf("rewrite() = %q, want %q (not this host's attachment)", got, text)
	}
}
