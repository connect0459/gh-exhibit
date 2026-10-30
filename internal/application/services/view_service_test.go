package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	domainservices "github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type fakeEvidenceReader struct {
	evidence domainservices.RawEvidence
	err      error
}

func (f *fakeEvidenceReader) ReadEvidence(_ context.Context, _ valueobjects.IssueRef) (domainservices.RawEvidence, error) {
	return f.evidence, f.err
}

type fakeAttachmentLinkReader struct {
	links []valueobjects.AttachmentLink
	err   error
}

func (f *fakeAttachmentLinkReader) ReadAttachmentLinks(_ context.Context, _ valueobjects.IssueRef) ([]valueobjects.AttachmentLink, error) {
	return f.links, f.err
}

type fakeViewWriter struct {
	err error

	called  bool
	doc     valueobjects.Document
	rewrite func(string) string
}

func (f *fakeViewWriter) WriteView(_ context.Context, _ valueobjects.IssueRef, doc valueobjects.Document, rewrite func(string) string) error {
	f.called = true
	f.doc = doc
	f.rewrite = rewrite
	return f.err
}

func viewAttachmentLink(t *testing.T) valueobjects.AttachmentLink {
	t.Helper()

	u, err := valueobjects.NewUrl(attachmentURL)
	if err != nil {
		t.Fatalf("NewUrl() error = %v", err)
	}
	link, err := valueobjects.NewAttachmentLink(u, "assets/abc-123.png")
	if err != nil {
		t.Fatalf("NewAttachmentLink() error = %v", err)
	}
	return link
}

func plainEvidence() domainservices.RawEvidence {
	return domainservices.RawEvidence{Issue: json.RawMessage(plainIssueJSON)}
}

func TestViewService_View_HandsTheDocumentAssembledFromPersistedEvidenceToTheViewWriter(t *testing.T) {
	views := &fakeViewWriter{}
	svc := NewViewService(&fakeEvidenceReader{evidence: plainEvidence()}, &fakeAttachmentLinkReader{}, views)

	_, err := svc.View(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}

	if !views.called || views.doc.Title() != "Something is broken" {
		t.Fatalf("WriteView called=%v with title %q, want it called with the persisted issue's document", views.called, views.doc.Title())
	}
}

func TestViewService_View_RewritesALinkedAttachmentToItsLocalCopy(t *testing.T) {
	views := &fakeViewWriter{}
	svc := NewViewService(&fakeEvidenceReader{evidence: plainEvidence()}, &fakeAttachmentLinkReader{links: []valueobjects.AttachmentLink{viewAttachmentLink(t)}}, views)

	_, err := svc.View(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}

	if got := views.rewrite("![x](" + attachmentURL + ")"); got != "![x](assets/abc-123.png)" {
		t.Fatalf("rewrite() = %q, want the linked attachment replaced by its local path", got)
	}
}

func TestViewService_View_ReplacesAnUnlinkedAttachmentWithAPlaceholderInsteadOfHotlinkingIt(t *testing.T) {
	views := &fakeViewWriter{}
	svc := NewViewService(&fakeEvidenceReader{evidence: plainEvidence()}, &fakeAttachmentLinkReader{}, views)

	_, err := svc.View(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}

	got := views.rewrite("![x](" + attachmentURL + ")")
	if strings.Contains(got, "![x]("+attachmentURL+")") || !strings.Contains(got, "attachment unavailable") {
		t.Fatalf("rewrite() = %q, want a placeholder instead of a remote image reference", got)
	}
}

func TestViewService_View_ReturnsTheSkipNotesRecordedWhileAssemblingTheDocument(t *testing.T) {
	evidence := plainEvidence()
	evidence.Timeline = []json.RawMessage{json.RawMessage(`{"event":"commented","id":"not-a-number"}`)}
	svc := NewViewService(&fakeEvidenceReader{evidence: evidence}, &fakeAttachmentLinkReader{}, &fakeViewWriter{})

	skipped, err := svc.View(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("View() error = %v", err)
	}

	if len(skipped) != 1 {
		t.Fatalf("View() returned %d skip notes, want 1", len(skipped))
	}
}

func TestViewService_View_FailsWithoutWritingAViewWhenTheEvidenceCannotBeRead(t *testing.T) {
	views := &fakeViewWriter{}
	svc := NewViewService(&fakeEvidenceReader{err: errors.New("no such file")}, &fakeAttachmentLinkReader{}, views)

	_, err := svc.View(context.Background(), testRef(t))

	if err == nil || !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("View() error = %v, want it to wrap the read failure", err)
	}
	if views.called {
		t.Fatal("WriteView was called although the evidence could not be read")
	}
}

func TestViewService_View_FailsWithoutWritingAViewWhenAPullRequestLacksItsPullRequestResource(t *testing.T) {
	views := &fakeViewWriter{}
	svc := NewViewService(&fakeEvidenceReader{evidence: domainservices.RawEvidence{Issue: json.RawMessage(pullRequestIssueJSON)}}, &fakeAttachmentLinkReader{}, views)

	_, err := svc.View(context.Background(), testRef(t))

	if err == nil || views.called {
		t.Fatalf("View() error = %v, WriteView called = %v, want an error and no view written", err, views.called)
	}
}

func TestViewService_View_FailsWhenTheAttachmentLinksCannotBeRead(t *testing.T) {
	svc := NewViewService(&fakeEvidenceReader{evidence: plainEvidence()}, &fakeAttachmentLinkReader{err: errors.New("corrupt attachments.json")}, &fakeViewWriter{})

	_, err := svc.View(context.Background(), testRef(t))

	if err == nil || !strings.Contains(err.Error(), "corrupt attachments.json") {
		t.Fatalf("View() error = %v, want it to wrap the link read failure", err)
	}
}

func TestViewService_View_FailsWhenTheViewCannotBeWritten(t *testing.T) {
	svc := NewViewService(&fakeEvidenceReader{evidence: plainEvidence()}, &fakeAttachmentLinkReader{}, &fakeViewWriter{err: errors.New("disk full")})

	_, err := svc.View(context.Background(), testRef(t))

	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("View() error = %v, want it to wrap the write failure", err)
	}
}
