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
	fetcher          repositories.EvidenceFetcher
	writer           repositories.EvidenceWriter
	provenanceWriter repositories.ProvenanceWriter
	docs             repositories.DocumentWriter
	attachments      repositories.AttachmentFetcher
	assets           repositories.AttachmentWriter
	host             string
	provenance       valueobjects.Provenance
}

// NewExportService builds an ExportService from its six collaborating
// ports (dependency inversion — this constructor takes abstract types,
// not infrastructure-layer concrete implementations), host, the target
// repository's own host (e.g. "github.com" or a GitHub Enterprise Server
// hostname) used to recognize that host's own attachment URLs, and
// provenance, persisted via provenanceWriter for every ref this
// ExportService exports.
func NewExportService(fetcher repositories.EvidenceFetcher, writer repositories.EvidenceWriter, provenanceWriter repositories.ProvenanceWriter, docs repositories.DocumentWriter, attachments repositories.AttachmentFetcher, assets repositories.AttachmentWriter, host string, provenance valueobjects.Provenance) *ExportService {
	return &ExportService{fetcher: fetcher, writer: writer, provenanceWriter: provenanceWriter, docs: docs, attachments: attachments, assets: assets, host: host, provenance: provenance}
}

// Export fetches, classifies, and renders the evidence for ref, returning
// any services.SkipNote recorded while classifying it (an individual
// unparsable item does not fail the whole export; see
// services.BuildEntries). Any failure aborts the export and returns a
// wrapped error.
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

	classified, skipped := services.BuildEntries(fetched.timeline, fetched.reviewComments, issue.HTMLURL())
	entries := []valueobjects.Entry{body}
	if issue.IsPullRequest() {
		diff, diffSkipped, err := services.BuildPullRequestDiff(body.Attribution(), fetched.pullRequest, fetched.pullRequestFiles)
		if err != nil {
			return nil, fmt.Errorf("could not build the pull request diff: %w", err)
		}
		skipped = append(skipped, diffSkipped...)
		entries = append(entries, diff)

		commits, commitsSkipped, err := services.BuildPullRequestCommits(body.Attribution(), fetched.pullRequestCommits)
		if err != nil {
			return nil, fmt.Errorf("could not build the pull request commits: %w", err)
		}
		skipped = append(skipped, commitsSkipped...)
		entries = append(entries, commits)
	}
	entries = append(entries, classified...)

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

	// Every fetch/build/validation step above — including downloading every
	// attachment — completes before any write below, so a failure anywhere
	// above leaves nothing on disk. The write phase itself has no rollback:
	// a failure partway through (e.g. WriteTimeline succeeding but
	// WriteDocument failing) can still leave a partial evidence directory
	// behind, since this project has no transactional/staged-write
	// mechanism for local files.
	if err := s.writer.WriteIssue(ctx, ref, rawIssue); err != nil {
		return nil, fmt.Errorf("could not persist the raw issue/PR resource: %w", err)
	}
	if err := s.provenanceWriter.WriteProvenance(ctx, ref, s.provenance); err != nil {
		return nil, fmt.Errorf("could not persist which tool produced this export: %w", err)
	}
	if issue.IsPullRequest() {
		if err := s.writer.WritePullRequest(ctx, ref, fetched.pullRequest); err != nil {
			return nil, fmt.Errorf("could not persist the raw pull request resource: %w", err)
		}
		if err := s.writer.WriteReviewComments(ctx, ref, fetched.reviewComments); err != nil {
			return nil, fmt.Errorf("could not persist the raw review comments: %w", err)
		}
		if err := s.writer.WritePullRequestFiles(ctx, ref, fetched.pullRequestFiles); err != nil {
			return nil, fmt.Errorf("could not persist the raw pull request files: %w", err)
		}
		if err := s.writer.WritePullRequestCommits(ctx, ref, fetched.pullRequestCommits); err != nil {
			return nil, fmt.Errorf("could not persist the raw pull request commits: %w", err)
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
	// Always called, even when failureLog is empty — see
	// repositories.AttachmentWriter.WriteFetchErrorLog.
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
	pullRequest        json.RawMessage
	reviewComments     []json.RawMessage
	pullRequestFiles   []json.RawMessage
	pullRequestCommits []json.RawMessage
	timeline           []json.RawMessage
}

// fetchPullRequestChainAndTimeline runs the pull-request chain
// (FetchPullRequest, then FetchReviewComments, then FetchPullRequestFiles,
// then FetchPullRequestCommits, when issue is a pull request) concurrently
// with FetchTimeline, since neither depends on the other's result —
// overlapping their round trips shortens Export's overall latency.
func (s *ExportService) fetchPullRequestChainAndTimeline(ctx context.Context, ref valueobjects.IssueRef, issue services.IssueResource) (fetchedPullRequestAndTimeline, error) {
	fetchCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var result fetchedPullRequestAndTimeline
	var mu sync.Mutex
	var firstErr error
	// The first branch to fail cancels fetchCtx, interrupting the sibling
	// branch's in-flight fetch instead of waiting out its rate-limit
	// backoff (up to an hour, per internal/infrastructure/github/retry.go).
	// firstErr — not the cancellation this triggers in the sibling branch —
	// is what's returned, so one branch's collateral context.Canceled never
	// masks the other's real error.
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
		// FetchReviewComments is not called when FetchPullRequest fails:
		// only its result, not its invocation, is independent of
		// FetchPullRequest's outcome, so the existing sequential
		// short-circuit is kept even inside this concurrent branch.
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

			pullRequestFiles, err := s.fetcher.FetchPullRequestFiles(fetchCtx, ref)
			if err != nil {
				fail(fmt.Errorf("could not retrieve the pull request files: %w", err))
				return
			}
			result.pullRequestFiles = pullRequestFiles

			pullRequestCommits, err := s.fetcher.FetchPullRequestCommits(fetchCtx, ref)
			if err != nil {
				fail(fmt.Errorf("could not retrieve the pull request commits: %w", err))
				return
			}
			result.pullRequestCommits = pullRequestCommits
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
	filename valueobjects.AssetFilename
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
// write, and this run's failure log (nil when nothing failed).
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
			// An ordinary fetch failure (broken link, access denied) is
			// recorded here, not propagated — a single broken attachment
			// link must not abort the whole export. A context
			// cancellation/deadline means the caller gave up on this
			// Export call entirely, not that one attachment is
			// unavailable, so it is returned as an error instead —
			// matching FetchIssue/FetchTimeline and the other fetch steps.
			if errors.Is(r.err, context.Canceled) || errors.Is(r.err, context.DeadlineExceeded) {
				return nil, nil, nil, fmt.Errorf("could not download the attachment at %s: %w", r.attachment.URL(), r.err)
			}
			resolutions = append(resolutions, services.FetchFailed(r.attachment.URL(), r.err.Error()))
			fmt.Fprintf(&failureLog, "%s: %s\n", r.attachment.URL(), r.err)
			continue
		}

		filename, err := r.attachment.Filename(r.contentType)
		if err != nil {
			// See services.Filename's own Godoc for why this can't
			// actually happen; treated as an ordinary per-attachment
			// failure regardless, consistent with every other fetch-time
			// failure in this loop.
			resolutions = append(resolutions, services.FetchFailed(r.attachment.URL(), err.Error()))
			fmt.Fprintf(&failureLog, "%s: %s\n", r.attachment.URL(), err)
			continue
		}
		downloads = append(downloads, downloadedAsset{filename: filename, data: r.data})
		resolutions = append(resolutions, services.Downloaded(r.attachment.URL(), ref.AssetPath(filename)))
	}

	return services.Rewrite(rendered, resolutions), downloads, failureLog.Bytes(), nil
}
