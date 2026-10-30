package services

import (
	"context"
	"fmt"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// ViewService rebuilds a browsable view of an earlier export from its
// persisted evidence alone, entirely offline: read the raw evidence,
// assemble it into a Document, and hand that to a ViewWriter together with
// the rewrite that swaps attachment URLs for their downloaded copies.
type ViewService struct {
	evidence repositories.EvidenceReader
	links    repositories.AttachmentLinkReader
	views    repositories.ViewWriter
}

// NewViewService builds a ViewService from its three collaborating ports
// (dependency inversion — abstract types, not infrastructure-layer
// concrete implementations).
func NewViewService(evidence repositories.EvidenceReader, links repositories.AttachmentLinkReader, views repositories.ViewWriter) *ViewService {
	return &ViewService{evidence: evidence, links: links, views: views}
}

// View writes ref's view from its persisted evidence, returning any
// services.SkipNote recorded while assembling the document. Any failure
// aborts the view and returns a wrapped error, leaving no view written.
func (s *ViewService) View(ctx context.Context, ref valueobjects.IssueRef) ([]services.SkipNote, error) {
	raw, err := s.evidence.ReadEvidence(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("could not read the persisted evidence: %w", err)
	}

	doc, skipped, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		return nil, fmt.Errorf("could not assemble the document from the persisted evidence: %w", err)
	}

	links, err := s.links.ReadAttachmentLinks(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("could not read the attachment links: %w", err)
	}

	rewrite := services.NewAttachmentRewriter(hostOf(doc), links)
	if err := s.views.WriteView(ctx, ref, doc, rewrite); err != nil {
		return nil, fmt.Errorf("could not persist the view: %w", err)
	}
	return skipped, nil
}

func hostOf(doc valueobjects.Document) string {
	entries := doc.Entries()
	if len(entries) == 0 {
		return ""
	}
	body, ok := entries[0].(valueobjects.Body)
	if !ok {
		return ""
	}
	return body.Attribution().URL().Host()
}
