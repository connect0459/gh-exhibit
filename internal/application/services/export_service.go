// Package services implements gh-exhibit's application layer: orchestrating
// the domain and infrastructure layers into a single Export operation for
// one issue or pull request. Despite the shared package name, this is
// distinct from internal/domain/services, which holds domain-layer
// transformation logic.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// ExportService orchestrates the whole export flow for a single issue or
// pull request: fetch raw evidence, persist it verbatim, classify and
// assemble it into a Document, download any referenced attachments, and
// persist the rendered Markdown.
type ExportService struct {
	fetcher     repositories.EvidenceFetcher
	writer      repositories.EvidenceWriter
	docs        repositories.DocumentWriter
	attachments repositories.AttachmentFetcher
	assets      repositories.AttachmentWriter
	host        string
}

// NewExportService builds an ExportService from its five collaborating
// ports (dependency inversion — this constructor takes abstract types,
// not infrastructure-layer concrete implementations) plus host, the target
// repository's own host (e.g. "github.com" or a GitHub Enterprise Server
// hostname), used to recognize that host's own attachment URLs.
func NewExportService(fetcher repositories.EvidenceFetcher, writer repositories.EvidenceWriter, docs repositories.DocumentWriter, attachments repositories.AttachmentFetcher, assets repositories.AttachmentWriter, host string) *ExportService {
	return &ExportService{fetcher: fetcher, writer: writer, docs: docs, attachments: attachments, assets: assets, host: host}
}

// Export fetches, classifies, and renders the evidence for ref, returning
// any services.SkipNote recorded while classifying it (an individual
// unparsable item does not fail the whole export; see
// services.BuildEntries). Every fetch and build/validation step — including
// downloading every referenced attachment — runs before any write, so a
// failure anywhere in that phase leaves nothing on disk at all. The write
// phase itself has no rollback: a failure partway through it (e.g.
// WriteTimeline succeeding but WriteDocument failing) can still leave a
// partial evidence directory behind, since this project has no
// transactional/staged-write mechanism for local files. Any failure aborts
// the export and returns a wrapped error.
func (s *ExportService) Export(ctx context.Context, ref valueobjects.IssueRef) ([]services.SkipNote, error) {
	rawIssue, err := s.fetcher.FetchIssue(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve the issue/PR resource: %w", err)
	}

	issue, err := services.ParseIssueResource(rawIssue)
	if err != nil {
		return nil, fmt.Errorf("the issue/PR resource could not be parsed: %w", err)
	}

	fetched, err := s.fetchPullRequestChainAndTimeline(ctx, ref, issue)
	if err != nil {
		return nil, err
	}

	body, title, err := services.BuildBody(issue, fetched.pullRequest)
	if err != nil {
		return nil, fmt.Errorf("could not derive a title and body from the issue/PR resource: %w", err)
	}

	classified, skipped := services.BuildEntries(fetched.timeline, fetched.reviewComments)
	entries := append([]valueobjects.Entry{body}, classified...)

	doc, err := valueobjects.NewDocument(title, entries)
	if err != nil {
		return nil, fmt.Errorf("assemble document: %w", err)
	}

	var buf bytes.Buffer
	if err := doc.Render(&buf); err != nil {
		return nil, fmt.Errorf("could not render the document to Markdown: %w", err)
	}

	rendered, downloads, failureLog, err := s.resolveAttachments(ctx, ref, buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("could not resolve one or more attachments: %w", err)
	}

	if err := s.writer.WriteIssue(ctx, ref, rawIssue); err != nil {
		return nil, fmt.Errorf("could not persist the raw issue/PR resource: %w", err)
	}
	if issue.IsPullRequest() {
		if err := s.writer.WritePullRequest(ctx, ref, fetched.pullRequest); err != nil {
			return nil, fmt.Errorf("could not persist the raw pull request resource: %w", err)
		}
		if err := s.writer.WriteReviewComments(ctx, ref, fetched.reviewComments); err != nil {
			return nil, fmt.Errorf("could not persist the raw review comments: %w", err)
		}
	}
	if err := s.writer.WriteTimeline(ctx, ref, fetched.timeline); err != nil {
		return nil, fmt.Errorf("could not persist the raw timeline: %w", err)
	}
	for _, d := range downloads {
		if err := s.assets.WriteAsset(ctx, ref, d.filename, d.data); err != nil {
			return nil, fmt.Errorf("could not persist the downloaded attachment %s: %w", d.filename, err)
		}
	}
	// Always called, even when failureLog is empty: WriteFetchErrorLog
	// treats an empty log as "clear any existing one", so a stale log
	// from a prior run with failures doesn't survive a rerun where every
	// attachment (or none at all) now resolves successfully.
	if err := s.assets.WriteFetchErrorLog(ctx, ref, failureLog); err != nil {
		return nil, fmt.Errorf("could not persist the attachment fetch error log: %w", err)
	}
	if err := s.docs.WriteDocument(ctx, ref, rendered); err != nil {
		return nil, fmt.Errorf("could not persist the rendered document: %w", err)
	}

	return skipped, nil
}

// fetchedPullRequestAndTimeline groups fetchPullRequestChainAndTimeline's
// three raw results in named fields, rather than three positional returns —
// reviewComments and timeline share the same []json.RawMessage type, so a
// positional return gives the compiler nothing to catch a transposed
// assignment at either call site.
type fetchedPullRequestAndTimeline struct {
	pullRequest    json.RawMessage
	reviewComments []json.RawMessage
	timeline       []json.RawMessage
}

// fetchPullRequestChainAndTimeline runs the pull-request chain
// (FetchPullRequest, then FetchReviewComments when issue is a pull request)
// concurrently with FetchTimeline: neither depends on the other's result,
// only on ref, so overlapping their round trips shortens Export's overall
// latency. The pull-request chain itself keeps its existing sequential
// short-circuit — FetchReviewComments is not called when FetchPullRequest
// fails — since only its result, not its invocation, is independent of
// FetchPullRequest's outcome.
//
// Both branches share a cancellable context derived from ctx: whichever
// branch fails first cancels it, so the sibling branch's in-flight fetch
// (which may otherwise be blocked in a rate-limit backoff wait up to an
// hour long, per internal/infrastructure/github/retry.go) is interrupted
// instead of being waited out to completion. The first genuine failure to
// occur (not the cancellation this triggers in the sibling branch) is the
// one returned — a fixed check-order priority would risk one branch's
// real error being masked by the other branch's collateral
// context.Canceled once cancellation is in play.
func (s *ExportService) fetchPullRequestChainAndTimeline(ctx context.Context, ref valueobjects.IssueRef, issue services.IssueResource) (fetchedPullRequestAndTimeline, error) {
	fetchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var result fetchedPullRequestAndTimeline
	var mu sync.Mutex
	var firstErr error
	fail := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		if firstErr == nil {
			firstErr = err
			cancel()
		}
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		timeline, err := s.fetcher.FetchTimeline(fetchCtx, ref)
		if err != nil {
			fail(fmt.Errorf("could not retrieve the timeline: %w", err))
			return
		}
		result.timeline = timeline
	}()

	if issue.IsPullRequest() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pullRequest, err := s.fetcher.FetchPullRequest(fetchCtx, ref)
			if err != nil {
				fail(fmt.Errorf("could not retrieve the pull request resource: %w", err))
				return
			}
			result.pullRequest = pullRequest

			reviewComments, err := s.fetcher.FetchReviewComments(fetchCtx, ref)
			if err != nil {
				fail(fmt.Errorf("could not retrieve the review comments: %w", err))
				return
			}
			result.reviewComments = reviewComments
		}()
	}
	wg.Wait()

	if firstErr != nil {
		return fetchedPullRequestAndTimeline{}, firstErr
	}
	return result, nil
}

// downloadedAsset pairs an attachment's local filename with its fetched
// bytes, deferred to the write phase (Export's write calls all happen after
// every fetch/build step, so a later failure never leaves a partial
// evidence directory behind).
type downloadedAsset struct {
	filename string
	data     []byte
}

// maxConcurrentAttachmentFetches bounds how many attachment URLs
// resolveAttachments fetches at once, so an issue with many attachments
// doesn't open an unbounded number of simultaneous connections to
// github.com.
const maxConcurrentAttachmentFetches = 4

// attachmentFetchResult carries one Attachment's Fetch outcome out of
// resolveAttachments' worker pool for sequential, deterministic handling
// once every fetch has finished.
type attachmentFetchResult struct {
	attachment  services.Attachment
	data        []byte
	contentType string
	err         error
}

// resolveAttachments downloads every attachment URL referenced in rendered,
// up to maxConcurrentAttachmentFetches at a time, returning the rewritten
// Markdown (local paths substituted for successful downloads, an inline
// placeholder for failed ones), the downloaded assets still awaiting a
// write, and this run's failure log (nil when nothing failed). An ordinary
// fetch failure (broken link, access denied) is recorded here, not
// propagated — a single broken attachment link must not abort the whole
// export. A context cancellation/deadline is different: it means the
// caller gave up on this Export call entirely, not that one attachment is
// unavailable, so it is returned as an error instead of a placeholder —
// matching how FetchIssue/FetchTimeline and the other fetch steps treat
// the same error.
func (s *ExportService) resolveAttachments(ctx context.Context, ref valueobjects.IssueRef, rendered []byte) ([]byte, []downloadedAsset, []byte, error) {
	attachments := services.Detect(rendered, s.host)
	if len(attachments) == 0 {
		return rendered, nil, nil, nil
	}

	results := make([]attachmentFetchResult, len(attachments))
	sem := make(chan struct{}, maxConcurrentAttachmentFetches)
	var wg sync.WaitGroup
	for i, a := range attachments {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, a services.Attachment) {
			defer wg.Done()
			defer func() { <-sem }()
			data, contentType, err := s.attachments.Fetch(ctx, a)
			results[i] = attachmentFetchResult{attachment: a, data: data, contentType: contentType, err: err}
		}(i, a)
	}
	wg.Wait()

	resolutions := make([]services.Resolution, 0, len(attachments))
	var downloads []downloadedAsset
	var failureLog bytes.Buffer
	for _, r := range results {
		if r.err != nil {
			if errors.Is(r.err, context.Canceled) || errors.Is(r.err, context.DeadlineExceeded) {
				return nil, nil, nil, fmt.Errorf("could not download the attachment at %s: %w", r.attachment.URL(), r.err)
			}
			resolutions = append(resolutions, services.FetchFailed(r.attachment.URL(), r.err.Error()))
			fmt.Fprintf(&failureLog, "%s: %s\n", r.attachment.URL(), r.err)
			continue
		}

		filename := r.attachment.Filename(r.contentType)
		downloads = append(downloads, downloadedAsset{filename: filename, data: r.data})
		resolutions = append(resolutions, services.Downloaded(r.attachment.URL(), ref.AssetPath(filename)))
	}

	return services.Rewrite(rendered, resolutions), downloads, failureLog.Bytes(), nil
}
