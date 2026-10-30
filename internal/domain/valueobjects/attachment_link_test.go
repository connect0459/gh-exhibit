package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func mustNewURL(t *testing.T, raw string) valueobjects.Url {
	t.Helper()
	u, err := valueobjects.NewUrl(raw)
	if err != nil {
		t.Fatalf("NewUrl(%q): unexpected error: %v", raw, err)
	}
	return u
}

func TestNewAttachmentLink_RejectsAnEmptyLocalPath(t *testing.T) {
	_, err := valueobjects.NewAttachmentLink(mustNewURL(t, "https://github.com/user-attachments/assets/abc"), "")

	if err == nil {
		t.Fatal("expected an error for an empty local path, got nil")
	}
}

func TestAttachmentLink_ExposesTheFieldsItWasConstructedWith(t *testing.T) {
	source := mustNewURL(t, "https://github.com/user-attachments/assets/abc")
	link, err := valueobjects.NewAttachmentLink(source, "assets/abc-123.png")
	if err != nil {
		t.Fatalf("NewAttachmentLink(): unexpected error: %v", err)
	}

	if !link.URL().Equals(source) {
		t.Fatalf("URL() = %q, want %q", link.URL(), source)
	}
	if link.LocalPath() != "assets/abc-123.png" {
		t.Fatalf("LocalPath() = %q, want %q", link.LocalPath(), "assets/abc-123.png")
	}
}

func TestNewAttachmentLink_RejectsALocalPathThatIsNotASingleFileDirectlyUnderAssets(t *testing.T) {
	tests := []struct {
		name      string
		localPath string
	}{
		{"parent traversal", "assets/../../etc/passwd"},
		{"traversal from the document directory", "../secret.png"},
		{"a nested path", "assets/a/b.png"},
		{"another directory", "other/abc.png"},
		{"the assets directory itself", "assets/"},
		{"an absolute path", "/etc/passwd"},
		{"a backslash separator", `assets\abc.png`},
		{"a bare dot-dot file name", "assets/.."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := valueobjects.NewAttachmentLink(mustNewURL(t, "https://github.com/user-attachments/assets/abc"), tt.localPath)

			if err == nil {
				t.Fatalf("expected an error for local path %q, got nil", tt.localPath)
			}
		})
	}
}
