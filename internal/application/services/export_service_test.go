package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type fakeEvidenceFetcher struct {
	issue                 json.RawMessage
	issueErr              error
	timeline              []json.RawMessage
	timelineErr           error
	pullRequest           json.RawMessage
	pullRequestErr        error
	reviewComments        []json.RawMessage
	reviewCommentsErr     error
	pullRequestFiles      []json.RawMessage
	pullRequestFilesErr   error
	pullRequestCommits    []json.RawMessage
	pullRequestCommitsErr error
	subIssues             []json.RawMessage
	subIssuesErr          error
	parentIssue           json.RawMessage
	parentIssueErr        error
	checkRuns             []json.RawMessage
	checkRunsErr          error

	fetchPullRequestCalled        bool
	fetchReviewCommentsCalled     bool
	fetchPullRequestFilesCalled   bool
	fetchPullRequestCommitsCalled bool
	fetchSubIssuesCalled          bool
	fetchParentIssueCalled        bool
	fetchCheckRunsCalled          bool
	fetchCheckRunsCommitSHA       string

	// issueCalls counts FetchIssue invocations: the first is always for
	// ref itself (called synchronously before any concurrent fetch
	// starts), so only a second call — made, if at all, from within the
	// concurrent sub-issues/parent-issue branch — can be for the parent.
	issueCalls int

	// issueReferences/issueReferenceErrs serve a FetchIssue call made to
	// resolve a bare/cross-repo issue reference detected in the rendered
	// document, keyed by the referenced ref itself and checked before the
	// call-count-based logic above (which exists only to serve ref's own
	// fetch and, on a second call, its parent issue — an issue-reference
	// fetch is for neither, so it must not be confused with either by that
	// counter). issueReferenceCalls records how many times each such ref
	// was actually fetched, so a test can assert a repeated reference to
	// the same target is only fetched once.
	issueReferences     map[valueobjects.IssueRef]json.RawMessage
	issueReferenceErrs  map[valueobjects.IssueRef]error
	issueReferenceCalls map[valueobjects.IssueRef]int
}

func (f *fakeEvidenceFetcher) FetchIssue(_ context.Context, ref valueobjects.IssueRef) (json.RawMessage, error) {
	if _, ok := f.issueReferences[ref]; ok || f.issueReferenceErrs[ref] != nil {
		if f.issueReferenceCalls == nil {
			f.issueReferenceCalls = make(map[valueobjects.IssueRef]int)
		}
		f.issueReferenceCalls[ref]++
		return f.issueReferences[ref], f.issueReferenceErrs[ref]
	}

	f.issueCalls++
	if f.issueCalls == 1 {
		return f.issue, f.issueErr
	}
	f.fetchParentIssueCalled = true
	return f.parentIssue, f.parentIssueErr
}

func (f *fakeEvidenceFetcher) FetchTimeline(context.Context, valueobjects.IssueRef) ([]json.RawMessage, error) {
	return f.timeline, f.timelineErr
}

func (f *fakeEvidenceFetcher) FetchPullRequest(context.Context, valueobjects.IssueRef) (json.RawMessage, error) {
	f.fetchPullRequestCalled = true
	return f.pullRequest, f.pullRequestErr
}

func (f *fakeEvidenceFetcher) FetchReviewComments(context.Context, valueobjects.IssueRef) ([]json.RawMessage, error) {
	f.fetchReviewCommentsCalled = true
	return f.reviewComments, f.reviewCommentsErr
}

func (f *fakeEvidenceFetcher) FetchPullRequestFiles(context.Context, valueobjects.IssueRef) ([]json.RawMessage, error) {
	f.fetchPullRequestFilesCalled = true
	return f.pullRequestFiles, f.pullRequestFilesErr
}

func (f *fakeEvidenceFetcher) FetchPullRequestCommits(context.Context, valueobjects.IssueRef) ([]json.RawMessage, error) {
	f.fetchPullRequestCommitsCalled = true
	return f.pullRequestCommits, f.pullRequestCommitsErr
}

func (f *fakeEvidenceFetcher) FetchSubIssues(context.Context, valueobjects.IssueRef) ([]json.RawMessage, error) {
	f.fetchSubIssuesCalled = true
	return f.subIssues, f.subIssuesErr
}

func (f *fakeEvidenceFetcher) FetchCheckRuns(_ context.Context, _ valueobjects.IssueRef, commitSHA string) ([]json.RawMessage, error) {
	f.fetchCheckRunsCalled = true
	f.fetchCheckRunsCommitSHA = commitSHA
	return f.checkRuns, f.checkRunsErr
}

type fakeEvidenceWriter struct {
	issueErr              error
	timelineErr           error
	pullRequestErr        error
	reviewCommentsErr     error
	pullRequestFilesErr   error
	pullRequestCommitsErr error
	subIssuesErr          error
	parentIssueErr        error
	checkRunsErr          error

	wroteIssue                    json.RawMessage
	wroteTimeline                 []json.RawMessage
	wrotePullRequest              json.RawMessage
	wroteReviewComments           []json.RawMessage
	wrotePullRequestFiles         []json.RawMessage
	wrotePullRequestCommits       []json.RawMessage
	wroteSubIssues                []json.RawMessage
	wroteParentIssue              json.RawMessage
	wroteCheckRuns                []json.RawMessage
	writePullRequestCalled        bool
	writeReviewCommentsCalled     bool
	writePullRequestFilesCalled   bool
	writePullRequestCommitsCalled bool
	writeSubIssuesCalled          bool
	writeParentIssueCalled        bool
	writeCheckRunsCalled          bool
}

func (f *fakeEvidenceWriter) WriteIssue(_ context.Context, _ valueobjects.IssueRef, raw json.RawMessage) error {
	f.wroteIssue = raw
	return f.issueErr
}

func (f *fakeEvidenceWriter) WriteTimeline(_ context.Context, _ valueobjects.IssueRef, items []json.RawMessage) error {
	f.wroteTimeline = items
	return f.timelineErr
}

func (f *fakeEvidenceWriter) WritePullRequest(_ context.Context, _ valueobjects.IssueRef, raw json.RawMessage) error {
	f.writePullRequestCalled = true
	f.wrotePullRequest = raw
	return f.pullRequestErr
}

func (f *fakeEvidenceWriter) WriteReviewComments(_ context.Context, _ valueobjects.IssueRef, items []json.RawMessage) error {
	f.writeReviewCommentsCalled = true
	f.wroteReviewComments = items
	return f.reviewCommentsErr
}

func (f *fakeEvidenceWriter) WritePullRequestFiles(_ context.Context, _ valueobjects.IssueRef, items []json.RawMessage) error {
	f.writePullRequestFilesCalled = true
	f.wrotePullRequestFiles = items
	return f.pullRequestFilesErr
}

func (f *fakeEvidenceWriter) WritePullRequestCommits(_ context.Context, _ valueobjects.IssueRef, items []json.RawMessage) error {
	f.writePullRequestCommitsCalled = true
	f.wrotePullRequestCommits = items
	return f.pullRequestCommitsErr
}

func (f *fakeEvidenceWriter) WriteSubIssues(_ context.Context, _ valueobjects.IssueRef, items []json.RawMessage) error {
	f.writeSubIssuesCalled = true
	f.wroteSubIssues = items
	return f.subIssuesErr
}

func (f *fakeEvidenceWriter) WriteParentIssue(_ context.Context, _ valueobjects.IssueRef, raw json.RawMessage) error {
	f.writeParentIssueCalled = true
	f.wroteParentIssue = raw
	return f.parentIssueErr
}

func (f *fakeEvidenceWriter) WriteCheckRuns(_ context.Context, _ valueobjects.IssueRef, items []json.RawMessage) error {
	f.writeCheckRunsCalled = true
	f.wroteCheckRuns = items
	return f.checkRunsErr
}

type fakeDocumentWriter struct {
	err     error
	written []byte
}

func (f *fakeDocumentWriter) WriteDocument(_ context.Context, _ valueobjects.IssueRef, rendered []byte) error {
	f.written = rendered
	return f.err
}

type fakeProvenanceWriter struct {
	err     error
	written valueobjects.Provenance
}

func (f *fakeProvenanceWriter) WriteProvenance(_ context.Context, _ valueobjects.IssueRef, provenance valueobjects.Provenance) error {
	f.written = provenance
	return f.err
}

// fakeAttachmentFetcher is called concurrently by resolveAttachments'
// bounded worker pool, so its state must be safe for concurrent access —
// unlike every other fake in this file, whose collaborator is only ever
// called sequentially.
type fakeAttachmentFetcher struct {
	data        []byte
	contentType string
	err         error

	mu          sync.Mutex
	fetchedURLs []string
}

func (f *fakeAttachmentFetcher) Fetch(_ context.Context, attachment services.Attachment) ([]byte, string, error) {
	f.mu.Lock()
	f.fetchedURLs = append(f.fetchedURLs, attachment.URL().String())
	f.mu.Unlock()
	return f.data, f.contentType, f.err
}

type fakeAttachmentWriter struct {
	assetErr error
	logErr   error

	wroteAssets    map[string][]byte
	wroteLog       []byte
	logWriteCalled bool
}

func (f *fakeAttachmentWriter) WriteAsset(_ context.Context, _ valueobjects.IssueRef, filename valueobjects.AssetFilename, data []byte) error {
	if f.assetErr != nil {
		return f.assetErr
	}
	if f.wroteAssets == nil {
		f.wroteAssets = make(map[string][]byte)
	}
	f.wroteAssets[filename.String()] = data
	return nil
}

func (f *fakeAttachmentWriter) WriteFetchErrorLog(_ context.Context, _ valueobjects.IssueRef, log []byte) error {
	f.logWriteCalled = true
	f.wroteLog = log
	return f.logErr
}

// fakeClock implements repositories.Clock, returning a fixed now (the zero
// time.Time unless set).
type fakeClock struct {
	now time.Time
}

func (f fakeClock) Now() time.Time {
	return f.now
}

func testRef(t *testing.T) valueobjects.IssueRef {
	t.Helper()
	ref, err := valueobjects.NewIssueRef("octocat", "hello-world", 1)
	if err != nil {
		t.Fatalf("NewIssueRef() error = %v", err)
	}
	return ref
}

func refFor(t *testing.T, owner, repo string, number int) valueobjects.IssueRef {
	t.Helper()
	ref, err := valueobjects.NewIssueRef(owner, repo, number)
	if err != nil {
		t.Fatalf("NewIssueRef(%q, %q, %d) error = %v", owner, repo, number, err)
	}
	return ref
}

func testProvenance(t *testing.T) valueobjects.Provenance {
	t.Helper()
	p, err := valueobjects.NewProvenance("connect0459/gh-exhibit", "v0.1.0", "abc123")
	if err != nil {
		t.Fatalf("NewProvenance() error = %v", err)
	}
	return p
}

const plainIssueJSON = `{
	"title": "Something is broken",
	"body": "Steps to reproduce...",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1"
}`

const commentedEventJSON = `{
	"event": "commented",
	"id": 100,
	"user": {"login": "reviewer"},
	"body": "Looks fine to me.",
	"created_at": "2026-07-02T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1#issuecomment-100"
}`

const pullRequestIssueJSON = `{
	"title": "Add retry backoff",
	"body": "This adds retry.",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/2",
	"pull_request": {"url": "https://api.github.com/repos/example/repo/pulls/2"}
}`

const mergedPullRequestJSON = `{"merged_at": "2026-07-03T00:00:00Z", "head": {"sha": "abc1234567"}}`

func TestExportService_Export_WritesRawEvidenceAndRenderedDocumentForAPlainIssue(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, skipped, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(skipped) != 0 {
		t.Fatalf("got %d skip notes, want 0: %#v", len(skipped), skipped)
	}

	if repo.fetchPullRequestCalled {
		t.Fatal("FetchPullRequest was called for a plain issue")
	}
	if repo.fetchReviewCommentsCalled {
		t.Fatal("FetchReviewComments was called for a plain issue")
	}
	if repo.fetchPullRequestFilesCalled {
		t.Fatal("FetchPullRequestFiles was called for a plain issue")
	}
	if writer.writePullRequestCalled {
		t.Fatal("WritePullRequest was called for a plain issue")
	}
	if writer.writeReviewCommentsCalled {
		t.Fatal("WriteReviewComments was called for a plain issue")
	}
	if writer.writePullRequestFilesCalled {
		t.Fatal("WritePullRequestFiles was called for a plain issue")
	}

	if string(writer.wroteIssue) != plainIssueJSON {
		t.Fatalf("WriteIssue got %q, want the raw issue JSON verbatim", writer.wroteIssue)
	}
	if len(writer.wroteTimeline) != 1 {
		t.Fatalf("WriteTimeline got %d items, want 1", len(writer.wroteTimeline))
	}

	rendered := string(docs.written)
	if !strings.HasPrefix(rendered, "# Something is broken\n\n") {
		t.Fatalf("rendered document = %q, want it to start with the title", rendered)
	}
	if !strings.Contains(rendered, "Steps to reproduce...") {
		t.Fatalf("rendered document = %q, want it to contain the issue body", rendered)
	}
	if !strings.Contains(rendered, "Looks fine to me.") {
		t.Fatalf("rendered document = %q, want it to contain the classified comment", rendered)
	}
	if !strings.Contains(rendered, "\n------\n\n") {
		t.Fatalf("rendered document = %q, want a separator between entries", rendered)
	}
}

func TestExportService_Export_ReturnsTheRenderedDocumentBytesVerbatim(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	rendered, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !bytes.Equal(rendered, docs.written) {
		t.Fatalf("Export() returned rendered = %q, want it to match the bytes persisted via WriteDocument (%q)", rendered, docs.written)
	}
}

func TestExportService_Export_AlsoFetchesAndPersistsPullRequestEvidence(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		timeline:    nil,
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !repo.fetchPullRequestCalled {
		t.Fatal("FetchPullRequest was not called for a pull request")
	}
	if !repo.fetchReviewCommentsCalled {
		t.Fatal("FetchReviewComments was not called for a pull request")
	}
	if !writer.writePullRequestCalled {
		t.Fatal("WritePullRequest was not called for a pull request")
	}
	if !writer.writeReviewCommentsCalled {
		t.Fatal("WriteReviewComments was not called for a pull request")
	}
	if string(writer.wrotePullRequest) != mergedPullRequestJSON {
		t.Fatalf("WritePullRequest got %q, want the raw pull JSON verbatim", writer.wrotePullRequest)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, `"merged":"2026-07-03T00:00:00Z"`) {
		t.Fatalf("rendered document = %q, want it to reflect merged_at from the pull resource", rendered)
	}
}

const changedFileJSON = `{"filename":"internal/foo.go","status":"modified","additions":12,"deletions":3,"patch":"@@ -1,3 +1,3 @@"}`

func TestExportService_Export_IncludesAPullRequestDiffEntryForAPullRequest(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:            json.RawMessage(pullRequestIssueJSON),
		pullRequest:      json.RawMessage(mergedPullRequestJSON),
		pullRequestFiles: []json.RawMessage{json.RawMessage(changedFileJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !repo.fetchPullRequestFilesCalled {
		t.Fatal("FetchPullRequestFiles was not called for a pull request")
	}
	if !writer.writePullRequestFilesCalled {
		t.Fatal("WritePullRequestFiles was not called for a pull request")
	}
	if string(writer.wrotePullRequestFiles[0]) != changedFileJSON {
		t.Fatalf("WritePullRequestFiles got %q, want the raw changed file JSON verbatim", writer.wrotePullRequestFiles)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, `"files":1`) {
		t.Fatalf("rendered document = %q, want a PullRequestDiff entry reporting 1 changed file", rendered)
	}
	if !strings.Contains(rendered, "internal/foo.go") {
		t.Fatalf("rendered document = %q, want it to list the changed file's name", rendered)
	}
}

const commitJSON = `{"sha":"abc1234567","commit":{"author":{"name":"octocat","date":"2026-07-01T00:00:00Z"},"committer":{"name":"octocat","date":"2026-07-01T00:00:00Z"},"message":"feat: add retry backoff"}}`

func TestExportService_Export_IncludesAPullRequestCommitsEntryForAPullRequest(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:              json.RawMessage(pullRequestIssueJSON),
		pullRequest:        json.RawMessage(mergedPullRequestJSON),
		pullRequestCommits: []json.RawMessage{json.RawMessage(commitJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !repo.fetchPullRequestCommitsCalled {
		t.Fatal("FetchPullRequestCommits was not called for a pull request")
	}
	if !writer.writePullRequestCommitsCalled {
		t.Fatal("WritePullRequestCommits was not called for a pull request")
	}
	if string(writer.wrotePullRequestCommits[0]) != commitJSON {
		t.Fatalf("WritePullRequestCommits got %q, want the raw commit JSON verbatim", writer.wrotePullRequestCommits)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, `"commits":1`) {
		t.Fatalf("rendered document = %q, want a PullRequestCommits entry reporting 1 commit", rendered)
	}
	if !strings.Contains(rendered, "feat: add retry backoff") {
		t.Fatalf("rendered document = %q, want it to contain the commit's message", rendered)
	}
}

const checkRunJSON = `{"name":"build","status":"completed","conclusion":"success","html_url":"https://github.com/example/repo/runs/1"}`

func TestExportService_Export_IncludesAPullRequestChecksEntryForAPullRequest(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
		checkRuns:   []json.RawMessage{json.RawMessage(checkRunJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	capturedAt := time.Date(2026, 7, 22, 9, 30, 0, 0, time.UTC)
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{now: capturedAt})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !repo.fetchCheckRunsCalled {
		t.Fatal("FetchCheckRuns was not called for a pull request")
	}
	if repo.fetchCheckRunsCommitSHA != "abc1234567" {
		t.Fatalf("FetchCheckRuns commitSHA = %q, want the pull request's own head sha %q", repo.fetchCheckRunsCommitSHA, "abc1234567")
	}
	if !writer.writeCheckRunsCalled {
		t.Fatal("WriteCheckRuns was not called for a pull request")
	}
	if string(writer.wroteCheckRuns[0]) != checkRunJSON {
		t.Fatalf("WriteCheckRuns got %q, want the raw check run JSON verbatim", writer.wroteCheckRuns)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, `"checks":1`) {
		t.Fatalf("rendered document = %q, want a PullRequestChecks entry reporting 1 check", rendered)
	}
	if !strings.Contains(rendered, `"captured_at":"2026-07-22T09:30:00Z"`) {
		t.Fatalf("rendered document = %q, want it to record the clock's captured-at time", rendered)
	}
	if !strings.Contains(rendered, "`build`: success") {
		t.Fatalf("rendered document = %q, want it to list the check run's name and outcome", rendered)
	}
}

func TestExportService_Export_OmitsPullRequestChecksEntryWhenThereAreNoCheckRuns(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !writer.writeCheckRunsCalled {
		t.Fatal("WriteCheckRuns should still be called (writing an empty array) even with no check runs")
	}
	if strings.Contains(string(docs.written), `"checks"`) {
		t.Fatalf("rendered document = %q, want no PullRequestChecks entry when there are no check runs", docs.written)
	}
}

func TestExportService_Export_DoesNotFetchCheckRunsForAPlainIssue(t *testing.T) {
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if repo.fetchCheckRunsCalled {
		t.Fatal("FetchCheckRuns was called for a plain issue")
	}
	if writer.writeCheckRunsCalled {
		t.Fatal("WriteCheckRuns was called for a plain issue")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchCheckRunsFails(t *testing.T) {
	wantErr := errors.New("fetch check runs failed")
	repo := &fakeEvidenceFetcher{
		issue:        json.RawMessage(pullRequestIssueJSON),
		pullRequest:  json.RawMessage(mergedPullRequestJSON),
		checkRunsErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteCheckRunsFails(t *testing.T) {
	wantErr := errors.New("write check runs failed")
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{checkRunsErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenTheHeadCommitSHACannotBeResolved(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(`{"merged_at": null}`),
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err == nil {
		t.Fatal("Export() error = nil, want an error when the pull request resource has no head commit sha")
	}
	if repo.fetchCheckRunsCalled {
		t.Fatal("FetchCheckRuns was called despite the head commit sha failing to resolve")
	}
}

const subIssueJSON = `{"number":65,"title":"Include issue/PR labels","state":"closed","html_url":"https://github.com/example/repo/issues/65"}`

func TestExportService_Export_FetchesAndPersistsSubIssuesForAPlainIssue(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:     json.RawMessage(plainIssueJSON),
		subIssues: []json.RawMessage{json.RawMessage(subIssueJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !repo.fetchSubIssuesCalled {
		t.Fatal("FetchSubIssues was not called for a plain issue")
	}
	if !writer.writeSubIssuesCalled {
		t.Fatal("WriteSubIssues was not called for a plain issue")
	}
	if string(writer.wroteSubIssues[0]) != subIssueJSON {
		t.Fatalf("WriteSubIssues got %q, want the raw sub-issue JSON verbatim", writer.wroteSubIssues)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, `"sub_issues":1`) {
		t.Fatalf("rendered document = %q, want a SubIssues entry reporting 1 child", rendered)
	}
	if !strings.Contains(rendered, "Include issue/PR labels") {
		t.Fatalf("rendered document = %q, want it to list the child issue's title", rendered)
	}
}

func TestExportService_Export_OmitsSubIssuesEntryWhenThereAreNoChildren(t *testing.T) {
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !writer.writeSubIssuesCalled {
		t.Fatal("WriteSubIssues should still be called (writing an empty array) even with no children")
	}
	if strings.Contains(string(docs.written), `"sub_issues"`) {
		t.Fatalf("rendered document = %q, want no SubIssues entry when there are no children", docs.written)
	}
}

func TestExportService_Export_DoesNotFetchSubIssuesForAPullRequest(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if repo.fetchSubIssuesCalled {
		t.Fatal("FetchSubIssues was called for a pull request")
	}
	if writer.writeSubIssuesCalled {
		t.Fatal("WriteSubIssues was called for a pull request")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchSubIssuesFails(t *testing.T) {
	wantErr := errors.New("fetch sub-issues failed")
	repo := &fakeEvidenceFetcher{
		issue:        json.RawMessage(plainIssueJSON),
		subIssuesErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

const issueWithParentJSON = `{
	"title": "Sub-issue",
	"body": "x",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/69",
	"parent_issue_url": "https://api.github.com/repos/octocat/hello-world/issues/64"
}`

const parentIssueJSON = `{"number":64,"title":"Round of Tier 1 entries","state":"open","html_url":"https://github.com/example/repo/issues/64"}`

func TestExportService_Export_FetchesAndIncludesTheParentIssueWhenPresent(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(issueWithParentJSON),
		parentIssue: json.RawMessage(parentIssueJSON),
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !repo.fetchParentIssueCalled {
		t.Fatal("FetchIssue was not called a second time for the parent issue")
	}
	if !writer.writeParentIssueCalled {
		t.Fatal("WriteParentIssue was not called")
	}
	if string(writer.wroteParentIssue) != parentIssueJSON {
		t.Fatalf("WriteParentIssue got %q, want the raw parent issue JSON verbatim", writer.wroteParentIssue)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, `"number":64`) {
		t.Fatalf("rendered document = %q, want a ParentIssue entry naming the parent's number", rendered)
	}
	if !strings.Contains(rendered, "Round of Tier 1 entries") {
		t.Fatalf("rendered document = %q, want it to include the parent's title", rendered)
	}
}

func TestExportService_Export_OmitsParentIssueEntryWhenAbsentButStillWritesToRemoveAnyStaleFile(t *testing.T) {
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if repo.fetchParentIssueCalled {
		t.Fatal("FetchIssue was called a second time despite no parent_issue_url")
	}
	if !writer.writeParentIssueCalled {
		t.Fatal("WriteParentIssue should still be called (to remove any stale file) even with no parent")
	}
	if writer.wroteParentIssue != nil {
		t.Fatalf("WriteParentIssue got %q, want nil/empty when there is no parent", writer.wroteParentIssue)
	}
	if strings.Contains(string(docs.written), `"number":64`) {
		t.Fatalf("rendered document = %q, want no ParentIssue entry when there is no parent", docs.written)
	}
}

func TestExportService_Export_DoesNotFetchParentIssueForAPullRequest(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if repo.fetchParentIssueCalled {
		t.Fatal("FetchIssue was called a second time for a pull request")
	}
	if writer.writeParentIssueCalled {
		t.Fatal("WriteParentIssue was called for a pull request")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchParentIssueFails(t *testing.T) {
	wantErr := errors.New("fetch parent issue failed")
	repo := &fakeEvidenceFetcher{
		issue:          json.RawMessage(issueWithParentJSON),
		parentIssueErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorForAMalformedParentIssueURL(t *testing.T) {
	rawIssue := json.RawMessage(`{
		"title": "Sub-issue",
		"body": "x",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/69",
		"parent_issue_url": "https://api.github.com/not-the-expected-shape"
	}`)
	repo := &fakeEvidenceFetcher{issue: rawIssue}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err == nil {
		t.Fatal("Export() error = nil, want an error for a malformed parent_issue_url")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteSubIssuesFails(t *testing.T) {
	wantErr := errors.New("write sub-issues failed")
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{subIssuesErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteParentIssueFails(t *testing.T) {
	wantErr := errors.New("write parent issue failed")
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{parentIssueErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchPullRequestCommitsFails(t *testing.T) {
	wantErr := errors.New("fetch pull request commits failed")
	repo := &fakeEvidenceFetcher{
		issue:                 json.RawMessage(pullRequestIssueJSON),
		pullRequest:           json.RawMessage(mergedPullRequestJSON),
		pullRequestCommitsErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWritePullRequestCommitsFails(t *testing.T) {
	wantErr := errors.New("write pull request commits failed")
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{pullRequestCommitsErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchPullRequestFilesFails(t *testing.T) {
	wantErr := errors.New("fetch pull request files failed")
	repo := &fakeEvidenceFetcher{
		issue:               json.RawMessage(pullRequestIssueJSON),
		pullRequest:         json.RawMessage(mergedPullRequestJSON),
		pullRequestFilesErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWritePullRequestFilesFails(t *testing.T) {
	wantErr := errors.New("write pull request files failed")
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{pullRequestFilesErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want %v", err, wantErr)
	}
}

// barrierEvidenceFetcher wraps fakeEvidenceFetcher, blocking FetchPullRequest
// and FetchTimeline on a shared barrier so a test can prove both actually run
// concurrently: each sends to reached, then waits on release, so a fully
// sequential caller would deadlock waiting for the second reached send
// before ever unblocking the first.
type barrierEvidenceFetcher struct {
	fakeEvidenceFetcher
	reached chan struct{}
	release chan struct{}
}

func (f *barrierEvidenceFetcher) FetchPullRequest(ctx context.Context, ref valueobjects.IssueRef) (json.RawMessage, error) {
	f.reached <- struct{}{}
	<-f.release
	return f.fakeEvidenceFetcher.FetchPullRequest(ctx, ref)
}

func (f *barrierEvidenceFetcher) FetchTimeline(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error) {
	f.reached <- struct{}{}
	<-f.release
	return f.fakeEvidenceFetcher.FetchTimeline(ctx, ref)
}

func TestExportService_Export_FetchesThePullRequestChainAndTheTimelineConcurrently(t *testing.T) {
	fetcher := &barrierEvidenceFetcher{
		fakeEvidenceFetcher: fakeEvidenceFetcher{
			issue:       json.RawMessage(pullRequestIssueJSON),
			pullRequest: json.RawMessage(mergedPullRequestJSON),
			timeline:    []json.RawMessage{json.RawMessage(commentedEventJSON)},
		},
		reached: make(chan struct{}),
		release: make(chan struct{}),
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(fetcher, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})
	ref := testRef(t)

	done := make(chan error, 1)
	go func() {
		_, _, err := svc.Export(context.Background(), ref)
		done <- err
	}()

	for i := 0; i < 2; i++ {
		select {
		case <-fetcher.reached:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for both FetchPullRequest and FetchTimeline to start — they are not running concurrently")
		}
	}
	close(fetcher.release)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Export() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Export did not complete after releasing both fetches")
	}
}

// blockingTimelineFetcher wraps fakeEvidenceFetcher, blocking FetchTimeline
// until its ctx is cancelled — simulating a timeline fetch stuck in a
// real rate-limit backoff wait (which can be up to an hour, per
// internal/infrastructure/github/retry.go) while a sibling fetch fails
// immediately.
type blockingTimelineFetcher struct {
	fakeEvidenceFetcher
}

func (f *blockingTimelineFetcher) FetchTimeline(ctx context.Context, _ valueobjects.IssueRef) ([]json.RawMessage, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestExportService_Export_ReturnsPromptlyWhenThePullRequestChainFailsWhileTimelineIsStillFetching(t *testing.T) {
	wantErr := errors.New("boom")
	fetcher := &blockingTimelineFetcher{
		fakeEvidenceFetcher: fakeEvidenceFetcher{
			issue:          json.RawMessage(pullRequestIssueJSON),
			pullRequestErr: wantErr,
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(fetcher, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	done := make(chan error, 1)
	go func() {
		_, _, err := svc.Export(context.Background(), testRef(t))
		done <- err
	}()

	select {
	case err := <-done:
		if !errors.Is(err, wantErr) {
			t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
		}
	case <-time.After(time.Second):
		t.Fatal("Export did not return promptly when the pull-request chain failed — it waited for the still-fetching timeline instead of cancelling it")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchIssueFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{issueErr: wantErr}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if writer.wroteIssue != nil {
		t.Fatal("WriteIssue was called despite FetchIssue failing")
	}
}

func TestExportService_Export_WritesNothingWhenBuildBodyFailsAfterAllFetchesSucceed(t *testing.T) {
	// An empty html_url fails valueobjects.NewAttribution inside BuildBody, after
	// FetchIssue has already succeeded. No write should happen in this
	// case — otherwise a partial evidence directory (raw JSON on disk,
	// but no timeline/document) would be left behind.
	raw := json.RawMessage(`{
		"title": "Missing url",
		"body": "x",
		"user": {"login": "octocat"},
		"created_at": "2026-07-01T00:00:00Z",
		"html_url": ""
	}`)
	repo := &fakeEvidenceFetcher{issue: raw}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err == nil {
		t.Fatal("Export() error = nil, want an error from BuildBody's attribution validation")
	}
	if writer.wroteIssue != nil {
		t.Fatal("WriteIssue was called despite BuildBody failing — this would leave a partial evidence directory")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteIssueFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{issueErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestExportService_Export_PersistsProvenanceAlongsideTheRawEvidence(t *testing.T) {
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	provenance := testProvenance(t)
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", provenance, fakeClock{})

	if _, _, err := svc.Export(context.Background(), testRef(t)); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if !provenanceWriter.written.Equals(provenance) {
		t.Fatalf("WriteProvenance got %#v, want %#v", provenanceWriter.written, provenance)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteProvenanceFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{err: wantErr}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite WriteProvenance failing")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchPullRequestFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{
		issue:          json.RawMessage(pullRequestIssueJSON),
		pullRequestErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if repo.fetchReviewCommentsCalled {
		t.Fatal("FetchReviewComments was called despite FetchPullRequest failing")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWritePullRequestFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{pullRequestErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if !repo.fetchReviewCommentsCalled {
		t.Fatal("FetchReviewComments was not called — the whole fetch phase should complete before any write is attempted")
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite WritePullRequest failing")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchReviewCommentsFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{
		issue:             json.RawMessage(pullRequestIssueJSON),
		pullRequest:       json.RawMessage(mergedPullRequestJSON),
		reviewCommentsErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteReviewCommentsFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(pullRequestIssueJSON),
		pullRequest: json.RawMessage(mergedPullRequestJSON),
	}
	writer := &fakeEvidenceWriter{reviewCommentsErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteTimelineFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{timelineErr: wantErr}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite WriteTimeline failing")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenFetchTimelineFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{
		issue:       json.RawMessage(plainIssueJSON),
		timelineErr: wantErr,
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite FetchTimeline failing")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteDocumentFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{err: wantErr}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
}

func TestExportService_Export_ReturnsSkipNotesFromClassificationWithoutFailingTheWholeExport(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue: json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{
			json.RawMessage(`{not valid json`),
			json.RawMessage(commentedEventJSON),
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, skipped, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v, want nil (skips should not fail the export)", err)
	}
	if len(skipped) != 1 {
		t.Fatalf("got %d skip notes, want 1", len(skipped))
	}
}

const commentedEventWithAttachmentJSON = `{
	"event": "commented",
	"id": 100,
	"user": {"login": "reviewer"},
	"body": "See ![screenshot](https://github.com/user-attachments/assets/abc-123)",
	"created_at": "2026-07-02T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1#issuecomment-100"
}`

const attachmentURL = "https://github.com/user-attachments/assets/abc-123"

func TestExportService_Export_DownloadsAnAttachmentReferencedInTheRenderedDocumentAndRewritesItsURL(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{data: []byte("png-bytes"), contentType: "image/png"}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if len(attachments.fetchedURLs) != 1 || attachments.fetchedURLs[0] != attachmentURL {
		t.Fatalf("Fetch was called with %v, want [%s]", attachments.fetchedURLs, attachmentURL)
	}
	if string(assets.wroteAssets["abc-123.png"]) != "png-bytes" {
		t.Fatalf("WriteAsset wrote %q for abc-123.png, want %q", assets.wroteAssets["abc-123.png"], "png-bytes")
	}
	if !assets.logWriteCalled || assets.wroteLog != nil {
		t.Fatalf("WriteFetchErrorLog called=%v with %q, want it called with an empty log (nothing failed, but any stale log must still be cleared)", assets.logWriteCalled, assets.wroteLog)
	}

	rendered := string(docs.written)
	if strings.Contains(rendered, attachmentURL) {
		t.Fatalf("rendered document = %q, want the attachment URL rewritten to its local path", rendered)
	}
	if !strings.Contains(rendered, "assets/abc-123.png") {
		t.Fatalf("rendered document = %q, want it to reference the downloaded asset's local path", rendered)
	}
}

func TestExportService_Export_DownloadsAnAttachmentOnAGitHubEnterpriseServerHost(t *testing.T) {
	const ghesAttachmentURL = "https://github.example.com/user-attachments/assets/abc-123"
	const commentedEventWithGHESAttachmentJSON = `{
		"event": "commented",
		"id": 100,
		"user": {"login": "reviewer"},
		"body": "See ![screenshot](https://github.example.com/user-attachments/assets/abc-123)",
		"created_at": "2026-07-02T00:00:00Z",
		"html_url": "https://github.example.com/example/repo/issues/1#issuecomment-100"
	}`

	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithGHESAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{data: []byte("png-bytes"), contentType: "image/png"}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.example.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if len(attachments.fetchedURLs) != 1 || attachments.fetchedURLs[0] != ghesAttachmentURL {
		t.Fatalf("Fetch was called with %v, want [%s]", attachments.fetchedURLs, ghesAttachmentURL)
	}
}

func TestExportService_Export_ReplacesAFailedAttachmentFetchWithAPlaceholderAndPersistsAFailureLog(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{err: errors.New("404 not found")}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v, want nil (an attachment fetch failure should not fail the export)", err)
	}

	if len(assets.wroteAssets) != 0 {
		t.Fatalf("WriteAsset was called despite the fetch failing: %v", assets.wroteAssets)
	}
	if !strings.Contains(string(assets.wroteLog), attachmentURL) || !strings.Contains(string(assets.wroteLog), "404 not found") {
		t.Fatalf("WriteFetchErrorLog got %q, want it to mention the URL and failure reason", assets.wroteLog)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, attachmentURL) {
		t.Fatalf("rendered document = %q, want the original URL preserved in the placeholder", rendered)
	}
	if !strings.Contains(rendered, "attachment unavailable: 404 not found") {
		t.Fatalf("rendered document = %q, want a placeholder noting the failure reason", rendered)
	}
}

func TestExportService_Export_ReplacesAnOversizedAttachmentIDWithAPlaceholderInsteadOfAbortingTheExport(t *testing.T) {
	oversizedID := strings.Repeat("a", 300)
	oversizedAttachmentURL := "https://github.example.com/user-attachments/assets/" + oversizedID
	commentedEventWithOversizedAttachmentJSON := `{
		"event": "commented",
		"id": 100,
		"user": {"login": "reviewer"},
		"body": "See ![screenshot](` + oversizedAttachmentURL + `)",
		"created_at": "2026-07-02T00:00:00Z",
		"html_url": "https://github.example.com/example/repo/issues/1#issuecomment-100"
	}`

	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithOversizedAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{data: []byte("png-bytes"), contentType: "image/png"}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.example.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v, want nil (an oversized attachment id should not fail the export)", err)
	}

	if len(assets.wroteAssets) != 0 {
		t.Fatalf("WriteAsset was called despite the derived filename being invalid: %v", assets.wroteAssets)
	}
	if !strings.Contains(string(assets.wroteLog), oversizedAttachmentURL) {
		t.Fatalf("WriteFetchErrorLog got %q, want it to mention the oversized attachment's URL", assets.wroteLog)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, oversizedAttachmentURL) {
		t.Fatalf("rendered document = %q, want the original URL preserved in the placeholder", rendered)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteAssetFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{data: []byte("png-bytes"), contentType: "image/png"}
	assets := &fakeAttachmentWriter{assetErr: wantErr}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite WriteAsset failing")
	}
}

func TestExportService_Export_PropagatesAnErrorWhenWriteFetchErrorLogFails(t *testing.T) {
	wantErr := errors.New("boom")
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{err: errors.New("404 not found")}
	assets := &fakeAttachmentWriter{logErr: wantErr}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, wantErr) {
		t.Fatalf("Export() error = %v, want it to wrap %v", err, wantErr)
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite WriteFetchErrorLog failing")
	}
}

func TestExportService_Export_DoesNotFetchAnyAttachmentWhenTheRenderedDocumentReferencesNone(t *testing.T) {
	repo := &fakeEvidenceFetcher{issue: json.RawMessage(plainIssueJSON)}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if len(attachments.fetchedURLs) != 0 {
		t.Fatalf("Fetch was called %v, want no calls", attachments.fetchedURLs)
	}
	if !assets.logWriteCalled || assets.wroteLog != nil {
		t.Fatalf("WriteFetchErrorLog called=%v with %q, want it called with an empty log, so a stale log from a prior run with attachments is still cleared", assets.logWriteCalled, assets.wroteLog)
	}
}

func TestExportService_Export_DownloadsEveryAttachmentWhenTheRenderedDocumentReferencesMultiple(t *testing.T) {
	const commentedEventWithTwoAttachmentsJSON = `{
		"event": "commented",
		"id": 100,
		"user": {"login": "reviewer"},
		"body": "See ![a](https://github.com/user-attachments/assets/aaa-1) and ![b](https://github.com/user-attachments/assets/bbb-2)",
		"created_at": "2026-07-02T00:00:00Z",
		"html_url": "https://github.com/example/repo/issues/1#issuecomment-100"
	}`
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithTwoAttachmentsJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{data: []byte("bytes"), contentType: "image/png"}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if len(assets.wroteAssets) != 2 {
		t.Fatalf("got %d written assets, want 2: %v", len(assets.wroteAssets), assets.wroteAssets)
	}
	if _, ok := assets.wroteAssets["aaa-1.png"]; !ok {
		t.Fatalf("asset aaa-1.png was not written: %v", assets.wroteAssets)
	}
	if _, ok := assets.wroteAssets["bbb-2.png"]; !ok {
		t.Fatalf("asset bbb-2.png was not written: %v", assets.wroteAssets)
	}
}

func TestExportService_Export_AbortsWhenAnAttachmentFetchIsCancelled(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{err: context.Canceled}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Export() error = %v, want it to wrap context.Canceled", err)
	}
	if writer.wroteIssue != nil {
		t.Fatal("WriteIssue was called despite the attachment fetch being cancelled")
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite the attachment fetch being cancelled")
	}
	if len(assets.wroteAssets) != 0 || assets.logWriteCalled {
		t.Fatal("an asset or fetch-error log was written despite the attachment fetch being cancelled")
	}
}

func TestExportService_Export_AbortsWhenAnAttachmentFetchExceedsItsDeadline(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventWithAttachmentJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{err: context.DeadlineExceeded}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Export() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite the attachment fetch exceeding its deadline")
	}
}

const issueWithBareReferenceJSON = `{
	"title": "Something is broken",
	"body": "See #42 for context",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1"
}`

const issueWithRepeatedBareReferenceJSON = `{
	"title": "Something is broken",
	"body": "See #42, and again #42 for context",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1"
}`

const issueWithCrossRepoReferenceJSON = `{
	"title": "Something is broken",
	"body": "See other-owner/other-repo#7 for context",
	"user": {"login": "octocat"},
	"created_at": "2026-07-01T00:00:00Z",
	"html_url": "https://github.com/example/repo/issues/1"
}`

const referencedIssueJSON = `{
	"title": "The referenced issue",
	"body": "",
	"user": {"login": "octocat"},
	"created_at": "2026-06-01T00:00:00Z",
	"html_url": "https://github.com/octocat/hello-world/issues/42"
}`

const crossRepoReferencedIssueJSON = `{
	"title": "Cross repo issue",
	"body": "",
	"user": {"login": "someone"},
	"created_at": "2026-06-01T00:00:00Z",
	"html_url": "https://github.com/other-owner/other-repo/issues/7"
}`

func TestExportService_Export_LinksABareIssueReferenceWithItsResolvedTitle(t *testing.T) {
	ref42 := refFor(t, "octocat", "hello-world", 42)
	repo := &fakeEvidenceFetcher{
		issue: json.RawMessage(issueWithBareReferenceJSON),
		issueReferences: map[valueobjects.IssueRef]json.RawMessage{
			ref42: json.RawMessage(referencedIssueJSON),
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	rendered := string(docs.written)
	want := "`The referenced issue` [#42](https://github.com/octocat/hello-world/issues/42)"
	if !strings.Contains(rendered, want) {
		t.Fatalf("rendered document = %q, want it to contain %q", rendered, want)
	}
}

func TestExportService_Export_LinksACrossRepoIssueReferenceWithItsResolvedTitle(t *testing.T) {
	ref7 := refFor(t, "other-owner", "other-repo", 7)
	repo := &fakeEvidenceFetcher{
		issue: json.RawMessage(issueWithCrossRepoReferenceJSON),
		issueReferences: map[valueobjects.IssueRef]json.RawMessage{
			ref7: json.RawMessage(crossRepoReferencedIssueJSON),
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	rendered := string(docs.written)
	want := "`Cross repo issue` [other-owner/other-repo#7](https://github.com/other-owner/other-repo/issues/7)"
	if !strings.Contains(rendered, want) {
		t.Fatalf("rendered document = %q, want it to contain %q", rendered, want)
	}
}

func TestExportService_Export_LeavesAnUnresolvableIssueReferenceUnchanged(t *testing.T) {
	ref42 := refFor(t, "octocat", "hello-world", 42)
	repo := &fakeEvidenceFetcher{
		issue: json.RawMessage(issueWithBareReferenceJSON),
		issueReferenceErrs: map[valueobjects.IssueRef]error{
			ref42: errors.New("404 Not Found"),
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v, want nil (an unresolvable issue reference should not fail the export)", err)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, "See #42 for context") {
		t.Fatalf("rendered document = %q, want the original reference text left untouched", rendered)
	}
}

func TestExportService_Export_FetchesARepeatedIssueReferenceOnlyOnce(t *testing.T) {
	ref42 := refFor(t, "octocat", "hello-world", 42)
	repo := &fakeEvidenceFetcher{
		issue: json.RawMessage(issueWithRepeatedBareReferenceJSON),
		issueReferences: map[valueobjects.IssueRef]json.RawMessage{
			ref42: json.RawMessage(referencedIssueJSON),
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if repo.issueReferenceCalls[ref42] != 1 {
		t.Fatalf("FetchIssue was called %d times for the same reference, want 1", repo.issueReferenceCalls[ref42])
	}

	rendered := string(docs.written)
	if strings.Count(rendered, "`The referenced issue` [#42]") != 2 {
		t.Fatalf("rendered document = %q, want both occurrences linked", rendered)
	}
}

func TestExportService_Export_PropagatesAnErrorWhenIssueReferenceResolutionIsCancelled(t *testing.T) {
	ref42 := refFor(t, "octocat", "hello-world", 42)
	repo := &fakeEvidenceFetcher{
		issue: json.RawMessage(issueWithBareReferenceJSON),
		issueReferenceErrs: map[valueobjects.IssueRef]error{
			ref42: context.Canceled,
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Export() error = %v, want it to wrap context.Canceled", err)
	}
	if docs.written != nil {
		t.Fatal("WriteDocument was called despite issue-reference resolution being cancelled")
	}
}

func TestExportService_Export_DoesNotTreatAnAttachmentURLEmbeddedInAReferencedIssuesTitleAsAnAttachment(t *testing.T) {
	ref42 := refFor(t, "octocat", "hello-world", 42)
	maliciousTitleJSON := `{
		"title": "See ` + attachmentURL + ` for details",
		"body": "",
		"user": {"login": "attacker"},
		"created_at": "2026-06-01T00:00:00Z",
		"html_url": "https://github.com/octocat/hello-world/issues/42"
	}`
	repo := &fakeEvidenceFetcher{
		issue: json.RawMessage(issueWithBareReferenceJSON),
		issueReferences: map[valueobjects.IssueRef]json.RawMessage{
			ref42: json.RawMessage(maliciousTitleJSON),
		},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	attachments := &fakeAttachmentFetcher{data: []byte("png-bytes"), contentType: "image/png"}
	assets := &fakeAttachmentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, attachments, assets, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if len(attachments.fetchedURLs) != 0 {
		t.Fatalf("Fetch was called with %v, want none — a referenced issue's title text must never be treated as an attachment source", attachments.fetchedURLs)
	}
	if len(assets.wroteAssets) != 0 {
		t.Fatalf("WriteAsset was called despite the URL only appearing in a referenced issue's title: %v", assets.wroteAssets)
	}

	rendered := string(docs.written)
	if !strings.Contains(rendered, attachmentURL) {
		t.Fatalf("rendered document = %q, want the referenced title's literal text preserved verbatim", rendered)
	}
}

func TestExportService_Export_DoesNotFetchAnythingExtraWhenNoIssueReferencesArePresent(t *testing.T) {
	repo := &fakeEvidenceFetcher{
		issue:    json.RawMessage(plainIssueJSON),
		timeline: []json.RawMessage{json.RawMessage(commentedEventJSON)},
	}
	writer := &fakeEvidenceWriter{}
	provenanceWriter := &fakeProvenanceWriter{}
	docs := &fakeDocumentWriter{}
	svc := NewExportService(repo, writer, provenanceWriter, docs, &fakeAttachmentFetcher{}, &fakeAttachmentWriter{}, "github.com", testProvenance(t), fakeClock{})

	_, _, err := svc.Export(context.Background(), testRef(t))
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if repo.issueCalls != 1 {
		t.Fatalf("FetchIssue was called %d times, want 1 (no reference to resolve)", repo.issueCalls)
	}
}
