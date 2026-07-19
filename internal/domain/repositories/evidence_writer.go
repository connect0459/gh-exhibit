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
// EvidenceFetcher's fetch shape; concatenating them into ADR-002's
// single persisted array per file is this port's implementation's job.
type EvidenceWriter interface {
	WriteIssue(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error
	WriteTimeline(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error
	WritePullRequest(ctx context.Context, ref valueobjects.IssueRef, raw json.RawMessage) error
	WriteReviewComments(ctx context.Context, ref valueobjects.IssueRef, items []json.RawMessage) error
}
