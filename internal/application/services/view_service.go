package services

import (
	"context"
	"fmt"
	"slices"
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

// ViewResult is what a successful ViewService.View reports beyond the view
// itself: the services.SkipNote recorded while assembling the document, and
// whether the view shows attachments as unavailable only because the export
// predates attachment links.
type ViewResult struct {
	skipped               []services.SkipNote
	attachmentsUnrecorded bool
}

// NewViewResult constructs a ViewResult from the skip notes recorded while
// assembling the document and whether the export's attachment links were
// missing while the document held at least one attachment.
func NewViewResult(skipped []services.SkipNote, attachmentsUnrecorded bool) ViewResult {
	return ViewResult{skipped: slices.Clone(skipped), attachmentsUnrecorded: attachmentsUnrecorded}
}

// Skipped returns a copy of the skip notes recorded while assembling the
// document.
func (r ViewResult) Skipped() []services.SkipNote {
	return slices.Clone(r.skipped)
}

// AttachmentsUnrecorded reports whether the export predates attachment
// links and the document holds an attachment, so the view shows it as
// unavailable although a downloaded copy may exist; re-running export
// records the links.
func (r ViewResult) AttachmentsUnrecorded() bool {
	return r.attachmentsUnrecorded
}

// View writes ref's view from its persisted evidence. Any failure aborts
// the view and returns a wrapped error, leaving no view written.
func (s *ViewService) View(ctx context.Context, ref valueobjects.IssueRef) (ViewResult, error) {
	raw, err := s.evidence.ReadEvidence(ctx, ref)
	if err != nil {
		return ViewResult{}, fmt.Errorf("could not read the persisted evidence: %w", err)
	}

	doc, skipped, err := services.AssembleDocument(raw, time.Time{})
	if err != nil {
		return ViewResult{}, fmt.Errorf("could not assemble the document from the persisted evidence: %w", err)
	}

	links, recorded, err := s.links.ReadAttachmentLinks(ctx, ref)
	if err != nil {
		return ViewResult{}, fmt.Errorf("could not read the attachment links: %w", err)
	}

	rewriteAttachments := services.NewAttachmentRewriter(hostOf(doc), links)
	sawAttachment := false
	rewrite := func(markdown string) string {
		rewritten := rewriteAttachments(markdown)
		if rewritten != markdown {
			sawAttachment = true
		}
		return rewritten
	}

	if err := s.views.WriteView(ctx, ref, doc, rewrite); err != nil {
		return ViewResult{}, fmt.Errorf("could not persist the view: %w", err)
	}
	return NewViewResult(skipped, !recorded && sawAttachment), nil
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
