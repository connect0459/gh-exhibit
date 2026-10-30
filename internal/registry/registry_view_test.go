package registry

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
	"github.com/connect0459/gh-exhibit/internal/infrastructure/persistence"
)

func TestNewViewService_WritesAnIndexHTMLFromAnEarlierExportsPersistedEvidence(t *testing.T) {
	dir := t.TempDir()
	ref, err := valueobjects.NewIssueRef("octocat", "hello-world", 7)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}
	issue := json.RawMessage(`{"title":"Something is broken","body":"Steps","user":{"login":"octocat"},"created_at":"2026-07-01T00:00:00Z","html_url":"https://github.com/octocat/hello-world/issues/7"}`)
	if err := persistence.NewEvidenceWriter(dir).WriteIssue(context.Background(), ref, issue); err != nil {
		t.Fatalf("WriteIssue() error = %v", err)
	}

	svc := NewViewService(Config{OutputDir: dir})
	if _, err := svc.View(context.Background(), ref); err != nil {
		t.Fatalf("View() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "hello-world", "7", "index.html"))
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}
	if !strings.Contains(string(got), "Something is broken") {
		t.Fatalf("index.html = %q, want it to contain the issue title", string(got))
	}
}
