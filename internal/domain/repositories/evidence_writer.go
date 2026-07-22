package repositories

import (
	"context"
	"encoding/json"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// EvidenceWriter is the abstract port the application layer depends on to
// persist raw evidence for an issue or pull request to local storage;
// infrastructure implements it (dependency inversion), symmetric to
// EvidenceFetcher on the acquisition side. Timeline and review comment
// pages arrive as one raw JSON element per item, matching
// EvidenceFetcher's fetch shape; concatenating them into a single
// persisted array per file is this port's implementation's job.
type EvidenceWriter interface {
	// WriteIssue persists ref's raw issue or pull request resource
	// verbatim.
	WriteIssue(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error
	// WriteTimeline persists ref's timeline items, concatenated into a
	// single JSON array.
	WriteTimeline(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error
	// WritePullRequest persists ref's raw pull request resource verbatim.
	WritePullRequest(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error
	// WriteReviewComments persists ref's review comment items,
	// concatenated into a single JSON array.
	WriteReviewComments(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error
	// WritePullRequestFiles persists ref's changed-file items, concatenated
	// into a single JSON array.
	WritePullRequestFiles(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error
	// WritePullRequestCommits persists ref's commit items, concatenated
	// into a single JSON array.
	WritePullRequestCommits(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error
	// WriteSubIssues persists ref's sub-issue items, concatenated into a
	// single JSON array.
	WriteSubIssues(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error
	// WriteParentIssue persists ref's parent issue resource verbatim. raw
	// is nil/empty when ref has no parent, in which case any file left by
	// an earlier run (from when ref did have one) is removed instead, so a
	// rerun stays a self-healing view of ref's current state.
	WriteParentIssue(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error
}
