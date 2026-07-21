// Package registry wires gh-exhibit's infrastructure-layer implementations
// into the application layer's ExportService (dependency injection root),
// so the presentation layer depends only on the result, not on which
// concrete infrastructure packages exist.
package registry

import (
	"fmt"
	"net/http"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/connect0459/gh-exhibit/internal/application/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
	"github.com/connect0459/gh-exhibit/internal/infrastructure/github"
	"github.com/connect0459/gh-exhibit/internal/infrastructure/persistence"
)

// toolName identifies gh-exhibit itself in every export's
// evidence/provenance.json, distinguishing its own output from a similar
// tool's.
const toolName = "connect0459/gh-exhibit"

// Config holds NewExportService's parameters. A struct rather than several
// positional strings, deliberately: Host, OutputDir, Version, and Commit
// are all plain strings with no compiler-visible distinction between them,
// so a positional signature would let a caller transpose them silently;
// naming the fields at every call site removes that risk.
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
	// in every export's evidence/provenance.json alongside toolName.
	Version string
	Commit  string

	// AuthToken and Transport are passed through to both go-gh clients
	// this constructor builds. Production callers leave both unset, so
	// AuthToken resolves from the gh environment as usual and Transport
	// defaults to http.DefaultTransport; a test supplies both to point
	// NewExportService's real fetchers at a fake server instead of the
	// gh environment or the real GitHub host, the same seam
	// NewEvidenceFetcher/NewAttachmentFetcher's own tests already use.
	AuthToken string
	Transport http.RoundTripper
}

// NewExportService builds an ExportService backed by a go-gh REST/HTTP
// client scoped to cfg.Host and local filesystem storage rooted at
// cfg.OutputDir.
func NewExportService(cfg Config) (*services.ExportService, error) {
	fetcher, err := github.NewEvidenceFetcher(api.ClientOptions{Host: cfg.Host, AuthToken: cfg.AuthToken, Transport: cfg.Transport})
	if err != nil {
		return nil, fmt.Errorf("could not create the GitHub evidence client: %w", err)
	}

	attachments, err := github.NewAttachmentFetcher(api.ClientOptions{Host: cfg.Host, AuthToken: cfg.AuthToken, Transport: cfg.Transport})
	if err != nil {
		return nil, fmt.Errorf("could not create the GitHub attachment client: %w", err)
	}

	provenance, err := valueobjects.NewProvenance(toolName, cfg.Version, cfg.Commit)
	if err != nil {
		return nil, fmt.Errorf("could not build the export provenance: %w", err)
	}

	writer := persistence.NewEvidenceWriter(cfg.OutputDir)
	provenanceWriter := persistence.NewProvenanceWriter(cfg.OutputDir)
	docs := persistence.NewDocumentWriter(cfg.OutputDir)
	assets := persistence.NewAttachmentWriter(cfg.OutputDir)

	return services.NewExportService(fetcher, writer, provenanceWriter, docs, attachments, assets, cfg.Host, provenance), nil
}
