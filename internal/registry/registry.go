// Package registry wires gh-exhibit's infrastructure-layer implementations
// into the application layer's ExportService (dependency injection root),
// so the presentation layer depends only on the result, not on which
// concrete infrastructure packages exist.
package registry

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/application/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
	"github.com/connect0459/gh-exhibit/internal/infrastructure/github"
	"github.com/connect0459/gh-exhibit/internal/infrastructure/persistence"
)

// toolName identifies gh-exhibit itself in every Document's provenance
// line, distinguishing its own output from a similar tool's.
const toolName = "connect0459/gh-exhibit"

// Config holds NewExportService's parameters. A struct rather than two
// positional strings, deliberately: Host and OutputDir are both plain
// strings with no compiler-visible distinction between them, so a
// positional signature would let a caller swap them silently; naming the
// fields at every call site removes that risk instead of relying on a test
// that couldn't meaningfully catch it either (both constructions still
// "succeed" — reaching the wrong host or the wrong directory only shows up
// at runtime).
type Config struct {
	// Host is the target repository's host (e.g. "github.com"), scoping
	// both the GitHub REST and attachment-download clients this
	// constructor builds.
	Host string

	// OutputDir is the local filesystem directory this constructor's
	// writers persist raw evidence, rendered Markdown, and downloaded
	// attachments under.
	OutputDir string

	// Version and Commit identify the running gh-exhibit build, recorded
	// in every Document's provenance line alongside toolName.
	Version string
	Commit  string
}

// NewExportService builds an ExportService backed by a go-gh REST/HTTP
// client scoped to cfg.Host and local filesystem storage rooted at
// cfg.OutputDir.
func NewExportService(cfg Config) (*services.ExportService, error) {
	fetcher, err := github.NewEvidenceFetcher(api.ClientOptions{Host: cfg.Host})
	if err != nil {
		return nil, fmt.Errorf("could not create the GitHub evidence client: %w", err)
	}

	attachments, err := github.NewAttachmentFetcher(api.ClientOptions{Host: cfg.Host})
	if err != nil {
		return nil, fmt.Errorf("could not create the GitHub attachment client: %w", err)
	}

	provenance, err := valueobjects.NewProvenance(toolName, cfg.Version, cfg.Commit)
	if err != nil {
		return nil, fmt.Errorf("could not build the export provenance: %w", err)
	}

	writer := persistence.NewEvidenceWriter(cfg.OutputDir)
	docs := persistence.NewDocumentWriter(cfg.OutputDir)
	assets := persistence.NewAttachmentWriter(cfg.OutputDir)

	return services.NewExportService(fetcher, writer, docs, attachments, assets, cfg.Host, provenance), nil
}
