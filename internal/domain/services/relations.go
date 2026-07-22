package services

import (
	"encoding/json"
	"fmt"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// issueSummaryWire is the shape shared by a parent issue resource (GET
// /issues/{number}, refetched for the ref named by parent_issue_url) and
// one element of GET /issues/{number}/sub_issues.
type issueSummaryWire struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
}

// BuildParentIssue constructs the ParentIssue Tier 1 entry from rawParent
// (the parent's own issue resource, refetched via EvidenceFetcher.FetchIssue
// using the ref parsed from IssueResource.ParentIssueRef). attribution is
// reused as-is — typically the same Attribution BuildBody already derived
// for the sub-issue itself — since a parent reference has no event of its
// own to attribute. Unlike BuildSubIssues, a malformed rawParent returns an
// error rather than a SkipNote: it names a single resource, not one item
// among many.
func BuildParentIssue(attribution valueobjects.Attribution, rawParent json.RawMessage) (valueobjects.ParentIssue, error) {
	summary, err := buildIssueSummary(rawParent)
	if err != nil {
		return valueobjects.ParentIssue{}, fmt.Errorf("parent issue: %w", err)
	}
	return valueobjects.NewParentIssue(attribution, summary), nil
}

// BuildSubIssues constructs the SubIssues Tier 1 entry from rawChildren
// (from EvidenceFetcher.FetchSubIssues). attribution is reused as-is,
// following the same reasoning as BuildParentIssue. A child item that
// cannot be parsed is recorded as a SkipNote and skipped rather than
// aborting the whole call, matching BuildPullRequestCommits/
// BuildPullRequestDiff's handling of their own item lists.
func BuildSubIssues(attribution valueobjects.Attribution, rawChildren []json.RawMessage) (valueobjects.SubIssues, []SkipNote, error) {
	var skipped []SkipNote
	children := make([]valueobjects.IssueSummary, 0, len(rawChildren))
	for _, raw := range rawChildren {
		summary, err := buildIssueSummary(raw)
		if err != nil {
			skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
			continue
		}
		children = append(children, summary)
	}

	return valueobjects.NewSubIssues(attribution, children), skipped, nil
}

func buildIssueSummary(raw json.RawMessage) (valueobjects.IssueSummary, error) {
	var w issueSummaryWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return valueobjects.IssueSummary{}, fmt.Errorf("unmarshal issue summary: %w", err)
	}

	state, err := valueobjects.ParseIssueState(w.State)
	if err != nil {
		return valueobjects.IssueSummary{}, fmt.Errorf("issue summary state: %w", err)
	}

	summary, err := valueobjects.NewIssueSummary(w.Number, w.Title, state, w.HTMLURL)
	if err != nil {
		return valueobjects.IssueSummary{}, fmt.Errorf("issue summary: %w", err)
	}
	return summary, nil
}
