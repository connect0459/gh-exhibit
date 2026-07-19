// Package registry wires gh-exhibit's infrastructure-layer implementations
// into the application layer's ExportService (dependency injection root),
// so the presentation layer depends only on the result, not on which
// concrete infrastructure packages exist.
package registry

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/application/services"
	"github.com/connect0459/gh-exhibit/internal/infrastructure/github"
	"github.com/connect0459/gh-exhibit/internal/infrastructure/persistence"
)

// Config holds NewExportService's parameters. A struct rather than two
// positional strings, deliberately: Host and OutputDir are both plain
// strings with no compiler-visible distinction between them, so a
// positional signature would let a caller swap them silently; naming the
// fields at every call site removes that risk instead of relying on a test
// that couldn't meaningfully catch it either (both constructions still
// "succeed" — reaching the wrong host or the wrong directory only shows up
// at runtime).
type Config struct {
	// Host is the target repository's host (e.g. "github.com").
	Host string

	// OutputDir is the local filesystem directory evidence is written
	// under.
	OutputDir string
}

// NewExportService builds an ExportService backed by a go-gh REST/HTTP
// client scoped to cfg.Host and local filesystem storage rooted at
// cfg.OutputDir.
func NewExportService(cfg Config) (*services.ExportService, error) {
	fetcher, err := github.NewEvidenceFetcher(api.ClientOptions{Host: cfg.Host})
	if err != nil {
		return nil, fmt.Errorf("registry: could not create the GitHub evidence client: %w", err)
	}

	attachments, err := github.NewAttachmentFetcher(api.ClientOptions{Host: cfg.Host})
	if err != nil {
		return nil, fmt.Errorf("registry: could not create the GitHub attachment client: %w", err)
	}

	writer := persistence.NewEvidenceWriter(cfg.OutputDir)
	docs := persistence.NewDocumentWriter(cfg.OutputDir)
	assets := persistence.NewAttachmentWriter(cfg.OutputDir)

	return services.NewExportService(fetcher, writer, docs, attachments, assets), nil
}
