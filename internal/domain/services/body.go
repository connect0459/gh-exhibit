package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type issueResourceWire struct {
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	User        actorWire       `json:"user"`
	CreatedAt   time.Time       `json:"created_at"`
	HTMLURL     string          `json:"html_url"`
	ClosedAt    *time.Time      `json:"closed_at"`
	PullRequest json.RawMessage `json:"pull_request,omitempty"`
}

type pullRequestResourceWire struct {
	MergedAt *time.Time `json:"merged_at"`
}

// IssueResource is the parsed issue/PR resource (from
// EvidenceFetcher.FetchIssue), unmarshaled once so a caller can check
// IsPullRequest and later pass the same parse result into BuildBody
// without unmarshaling rawIssue a second time.
type IssueResource struct {
	wire issueResourceWire
}

// ParseIssueResource unmarshals rawIssue once, ready for IsPullRequest and
// BuildBody to use.
func ParseIssueResource(rawIssue json.RawMessage) (IssueResource, error) {
	var w issueResourceWire
	if err := json.Unmarshal(rawIssue, &w); err != nil {
		return IssueResource{}, fmt.Errorf("unmarshal issue resource: %w", err)
	}
	return IssueResource{wire: w}, nil
}

// IsPullRequest reports whether the issue resource represents a pull
// request, detected via presence of its own "pull_request" key — GitHub's
// issues endpoint serves both issues and PRs, and only a PR's response
// carries this key.
func (r IssueResource) IsPullRequest() bool {
	return len(r.wire.PullRequest) > 0
}

// BuildBody constructs the document title and the Body Tier 1 entry from
// issue (already parsed via ParseIssueResource). rawPullRequest is the
// pull resource (from EvidenceFetcher.FetchPullRequest) and should be
// nil/empty for a plain issue: merged_at is never present on the issues
// endpoint's own response, so it is only ever sourced from
// rawPullRequest when given.
func BuildBody(issue IssueResource, rawPullRequest json.RawMessage) (body valueobjects.Body, title string, err error) {
	w := issue.wire

	attribution, err := valueobjects.NewAttribution(w.User.resolvedLogin(), w.CreatedAt, w.HTMLURL)
	if err != nil {
		return valueobjects.Body{}, "", fmt.Errorf("issue resource attribution: %w", err)
	}

	var mergedAt *time.Time
	if len(rawPullRequest) > 0 {
		var pw pullRequestResourceWire
		if err := json.Unmarshal(rawPullRequest, &pw); err != nil {
			return valueobjects.Body{}, "", fmt.Errorf("unmarshal pull request resource: %w", err)
		}
		mergedAt = pw.MergedAt
	}

	return valueobjects.NewBody(attribution, w.Body, w.ClosedAt, mergedAt), w.Title, nil
}
