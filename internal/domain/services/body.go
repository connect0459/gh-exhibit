package services

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type issueResourceWire struct {
	Title          string          `json:"title"`
	Body           string          `json:"body"`
	User           actorWire       `json:"user"`
	CreatedAt      time.Time       `json:"created_at"`
	HTMLURL        string          `json:"html_url"`
	ClosedAt       *time.Time      `json:"closed_at"`
	PullRequest    json.RawMessage `json:"pull_request,omitempty"`
	ParentIssueURL string          `json:"parent_issue_url"`
}

type pullRequestResourceWire struct {
	MergedAt  *time.Time `json:"merged_at"`
	Additions int        `json:"additions"`
	Deletions int        `json:"deletions"`
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
// carries this key. An explicit JSON null is treated the same as an
// absent key, since it carries no pull-request data either.
func (r IssueResource) IsPullRequest() bool {
	return len(r.wire.PullRequest) > 0 && string(r.wire.PullRequest) != "null"
}

// HTMLURL returns the issue/PR resource's own html_url. Timeline events
// with no per-event permalink of their own (e.g. "labeled"/"unlabeled")
// fall back to this as their attribution's url.
func (r IssueResource) HTMLURL() string {
	return r.wire.HTMLURL
}

// ParentIssueRef reports the ref this issue is a sub-issue of, resolved
// from the issue resource's own parent_issue_url. ok is false when
// parent_issue_url is absent — an issue with no parent, or any pull
// request, since GitHub never populates this field for one. err is
// non-nil only when parent_issue_url is present but its path does not end
// in .../repos/{owner}/{repo}/issues/{number} (see parseIssueResourceURL
// for why only the path's tail is checked, not its whole shape).
func (r IssueResource) ParentIssueRef() (ref valueobjects.IssueRef, ok bool, err error) {
	if r.wire.ParentIssueURL == "" {
		return valueobjects.IssueRef{}, false, nil
	}
	ref, err = parseIssueResourceURL(r.wire.ParentIssueURL)
	if err != nil {
		return valueobjects.IssueRef{}, false, fmt.Errorf("parse parent issue url: %w", err)
	}
	return ref, true, nil
}

// parseIssueResourceURL extracts an IssueRef from raw, a GitHub REST issue
// resource URL ending in .../repos/{owner}/{repo}/issues/{number}. Only the
// path's trailing 5 segments are inspected, rather than requiring the whole
// path to be exactly 5 segments: a GitHub Enterprise Server installation
// serves its REST API under an additional /api/v3/ prefix (matching go-gh's
// own restPrefix for outgoing requests), so a GHES-origin URL's path has
// two extra leading segments a github.com-origin one never has.
func parseIssueResourceURL(raw string) (valueobjects.IssueRef, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return valueobjects.IssueRef{}, fmt.Errorf("parse url %q: %w", raw, err)
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 5 {
		return valueobjects.IssueRef{}, fmt.Errorf("url %q does not match the expected .../repos/{owner}/{repo}/issues/{number} shape", raw)
	}
	tail := parts[len(parts)-5:]
	if tail[0] != "repos" || tail[3] != "issues" {
		return valueobjects.IssueRef{}, fmt.Errorf("url %q does not match the expected .../repos/{owner}/{repo}/issues/{number} shape", raw)
	}
	number, err := strconv.Atoi(tail[4])
	if err != nil {
		return valueobjects.IssueRef{}, fmt.Errorf("url %q has a non-numeric issue number: %w", raw, err)
	}
	return valueobjects.NewIssueRef(tail[1], tail[2], number)
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
