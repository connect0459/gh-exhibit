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
	FetchIssue(ctx context.Context, ref valueobjects.IssueRef) (json.RawMessage, error)
	FetchTimeline(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error)
	FetchPullRequest(ctx context.Context, ref valueobjects.IssueRef) (json.RawMessage, error)
	FetchReviewComments(ctx context.Context, ref valueobjects.IssueRef) ([]json.RawMessage, error)
}
