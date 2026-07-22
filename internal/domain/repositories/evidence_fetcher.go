package repositories

import (
	"context"
	"encoding/json"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// EvidenceFetcher is the abstract port the application layer depends on
// to fetch raw evidence for an issue or pull request; infrastructure
// implements it (dependency inversion). Timeline and review comment
// results are one raw JSON element per item, matching the shape
// services.BuildEntries already consumes, rather than a single
// concatenated blob.
type EvidenceFetcher interface {
	// FetchIssue fetches ref's issue or pull request resource.
	FetchIssue(ctx context.Context, ref valueobjects.IssueRef) (json.RawMessage, error)
	// FetchTimeline fetches ref's timeline, one raw JSON element per item
	// across all pages.
	FetchTimeline(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error)
	// FetchPullRequest fetches ref's pull request resource. Callers should
	// only call this once ref is known to be a pull request.
	FetchPullRequest(ctx context.Context, ref valueobjects.IssueRef) (json.RawMessage, error)
	// FetchReviewComments fetches ref's inline review comments, one raw
	// JSON element per item across all pages.
	FetchReviewComments(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error)
	// FetchPullRequestFiles fetches ref's changed files, one raw JSON
	// element per item across all pages. Callers should only call this
	// once ref is known to be a pull request.
	FetchPullRequestFiles(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error)
	// FetchPullRequestCommits fetches ref's commit list, one raw JSON
	// element per item across all pages. Callers should only call this
	// once ref is known to be a pull request.
	FetchPullRequestCommits(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error)
	// FetchSubIssues fetches ref's sub-issues, one raw JSON element per item
	// across all pages. Callers should only call this once ref is known to
	// be a plain issue: a pull request always has no sub-issues.
	FetchSubIssues(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error)
	// FetchCheckRuns fetches the check runs associated with commitSHA (a
	// pull request's head commit), one raw JSON element per item across all
	// pages. Callers should only call this once ref is known to be a pull
	// request and commitSHA has been resolved from its head commit.
	FetchCheckRuns(ctx context.Context, ref valueobjects.IssueRef, commitSHA string) ([]json.RawMessage, error)
}
