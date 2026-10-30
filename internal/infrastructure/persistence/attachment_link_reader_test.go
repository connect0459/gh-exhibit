package persistence

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func writeAttachmentsJSON(t *testing.T, baseDir, content string) {
	t.Helper()

	path := filepath.Join(baseDir, "hello-world", "42", "evidence", "attachments.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}

func TestAttachmentLinkReader_ReadAttachmentLinks_ReturnsTheLinksTheWriterPersisted(t *testing.T) {
	baseDir := t.TempDir()
	written := []valueobjects.AttachmentLink{
		testAttachmentLink(t, "https://github.com/user-attachments/assets/abc", "assets/abc-123.png"),
		testAttachmentLink(t, "https://github.com/user-attachments/assets/def", "assets/def-456.png"),
	}
	if err := NewAttachmentWriter(baseDir).WriteAttachmentLinks(context.Background(), testIssueRef(t), written); err != nil {
		t.Fatalf("WriteAttachmentLinks() error = %v", err)
	}

	got, recorded, err := NewAttachmentLinkReader(baseDir).ReadAttachmentLinks(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("ReadAttachmentLinks() error = %v", err)
	}

	if !recorded {
		t.Fatal("ReadAttachmentLinks() recorded = false, want true for an export that wrote attachments.json")
	}
	if len(got) != 2 || !got[0].URL().Equals(written[0].URL()) || got[0].LocalPath() != "assets/abc-123.png" || got[1].LocalPath() != "assets/def-456.png" {
		t.Fatalf("ReadAttachmentLinks() = %v, want the two links written, in order", got)
	}
}

func TestAttachmentLinkReader_ReadAttachmentLinks_ReportsAnExportThatPredatesAttachmentLinksAsNotRecorded(t *testing.T) {
	got, recorded, err := NewAttachmentLinkReader(t.TempDir()).ReadAttachmentLinks(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("ReadAttachmentLinks() error = %v, want nil when attachments.json does not exist", err)
	}

	if recorded || len(got) != 0 {
		t.Fatalf("ReadAttachmentLinks() = %v, recorded %v, want no links and recorded false", got, recorded)
	}
}

func TestAttachmentLinkReader_ReadAttachmentLinks_ReportsAnExportThatRecordedNoAttachmentsAsRecordedAndEmpty(t *testing.T) {
	baseDir := t.TempDir()
	writeAttachmentsJSON(t, baseDir, `[]`)

	got, recorded, err := NewAttachmentLinkReader(baseDir).ReadAttachmentLinks(context.Background(), testIssueRef(t))
	if err != nil {
		t.Fatalf("ReadAttachmentLinks() error = %v", err)
	}

	if !recorded || len(got) != 0 {
		t.Fatalf("ReadAttachmentLinks() = %v, recorded %v, want no links and recorded true", got, recorded)
	}
}

func TestAttachmentLinkReader_ReadAttachmentLinks_RejectsAPathThatEscapesTheAssetsDirectory(t *testing.T) {
	baseDir := t.TempDir()
	writeAttachmentsJSON(t, baseDir, `[{"url":"https://github.com/user-attachments/assets/abc","path":"assets/../../etc/passwd"}]`)

	_, _, err := NewAttachmentLinkReader(baseDir).ReadAttachmentLinks(context.Background(), testIssueRef(t))

	if err == nil {
		t.Fatal("expected an error for an attachments.json path that escapes assets/, got nil")
	}
}

func TestAttachmentLinkReader_ReadAttachmentLinks_ReturnsAnErrorForMalformedContent(t *testing.T) {
	for name, content := range map[string]string{
		"not a list":     `{"url":"https://example.com/a","path":"assets/a.png"}`,
		"relative url":   `[{"url":"not-a-url","path":"assets/a.png"}]`,
		"empty path":     `[{"url":"https://example.com/a","path":""}]`,
		"invalid syntax": `[{`,
	} {
		t.Run(name, func(t *testing.T) {
			baseDir := t.TempDir()
			writeAttachmentsJSON(t, baseDir, content)

			_, _, err := NewAttachmentLinkReader(baseDir).ReadAttachmentLinks(context.Background(), testIssueRef(t))

			if err == nil {
				t.Fatalf("expected an error for attachments.json content %q, got nil", content)
			}
		})
	}
}

func TestAttachmentLinkReader_ReadAttachmentLinks_ReturnsContextErrorWhenContextIsAlreadyCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := NewAttachmentLinkReader(t.TempDir()).ReadAttachmentLinks(ctx, testIssueRef(t))

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ReadAttachmentLinks() error = %v, want context.Canceled", err)
	}
}
