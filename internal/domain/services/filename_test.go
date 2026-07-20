package services_test

import (
	"testing"
)

func TestFilename_DerivesTheExtensionFromContentType(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"
	cases := []struct {
		contentType string
		want        string
	}{
		{"image/png", "9492692e-41a2-484f-8d3b-e149d5f2c20f.png"},
		{"image/jpeg", "9492692e-41a2-484f-8d3b-e149d5f2c20f.jpg"},
		{"image/gif", "9492692e-41a2-484f-8d3b-e149d5f2c20f.gif"},
		{"application/pdf", "9492692e-41a2-484f-8d3b-e149d5f2c20f.pdf"},
	}

	attachment := newTestAttachment(t, url)
	for _, c := range cases {
		got := attachment.Filename(c.contentType)
		if got != c.want {
			t.Fatalf("Filename(%q) = %q, want %q", c.contentType, got, c.want)
		}
	}
}

func TestFilename_IgnoresContentTypeParameters(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"

	got := newTestAttachment(t, url).Filename("image/png; charset=binary")

	want := "9492692e-41a2-484f-8d3b-e149d5f2c20f.png"
	if got != want {
		t.Fatalf("Filename() = %q, want %q", got, want)
	}
}

func TestFilename_OmitsTheExtensionForAnUnrecognizedContentType(t *testing.T) {
	url := "https://github.com/user-attachments/assets/9492692e-41a2-484f-8d3b-e149d5f2c20f"

	got := newTestAttachment(t, url).Filename("application/octet-stream")

	want := "9492692e-41a2-484f-8d3b-e149d5f2c20f"
	if got != want {
		t.Fatalf("Filename() = %q, want %q", got, want)
	}
}
