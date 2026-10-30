package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// RawEvidence is the verbatim GitHub REST evidence for one issue or pull
// request, one raw JSON element per item for list resources. Fields that
// do not apply to the ref's kind (or were never captured) are empty.
type RawEvidence struct {
	Issue              json.RawMessage
	PullRequest        json.RawMessage
	ReviewComments     []json.RawMessage
	PullRequestFiles   []json.RawMessage
	PullRequestCommits []json.RawMessage
	CheckRuns          []json.RawMessage
	SubIssues          []json.RawMessage
	ParentIssue        json.RawMessage
	Timeline           []json.RawMessage
}

// AssembleDocument classifies raw evidence into a Document: the body first,
// then the pull request's diff, commits and checks (or a plain issue's
// parent and sub-issues), then the timeline entries. capturedAt is the
// moment the pull request's check-run snapshot was observed. Individual
// unparsable items are returned as SkipNotes instead of failing the whole
// document; see BuildEntries.
func AssembleDocument(raw RawEvidence, capturedAt time.Time) (valueobjects.Document, []SkipNote, error) {
	issue, err := ParseIssueResource(raw.Issue)
	if err != nil {
		return valueobjects.Document{}, nil, fmt.Errorf("the issue/PR resource could not be parsed: %w", err)
	}
	if issue.IsPullRequest() && len(raw.PullRequest) == 0 {
		return valueobjects.Document{}, nil, errors.New("the pull request resource is missing for a pull request")
	}

	body, title, err := BuildBody(issue, raw.PullRequest)
	if err != nil {
		return valueobjects.Document{}, nil, fmt.Errorf("could not derive a title and body from the issue/PR resource: %w", err)
	}

	classified, skipped := BuildEntries(raw.Timeline, raw.ReviewComments, issue.HTMLURL())
	entries := []valueobjects.Entry{body}
	if issue.IsPullRequest() {
		diff, diffSkipped, err := BuildPullRequestDiff(body.Attribution(), raw.PullRequest, raw.PullRequestFiles)
		if err != nil {
			return valueobjects.Document{}, nil, fmt.Errorf("could not build the pull request diff: %w", err)
		}
		skipped = append(skipped, diffSkipped...)
		entries = append(entries, diff)

		commits, commitsSkipped, err := BuildPullRequestCommits(body.Attribution(), raw.PullRequestCommits)
		if err != nil {
			return valueobjects.Document{}, nil, fmt.Errorf("could not build the pull request commits: %w", err)
		}
		skipped = append(skipped, commitsSkipped...)
		entries = append(entries, commits)

		headSHA, err := PullRequestHeadSHA(raw.PullRequest)
		if err != nil {
			return valueobjects.Document{}, nil, fmt.Errorf("could not resolve the pull request's head commit sha: %w", err)
		}
		checks, checksSkipped, err := BuildPullRequestChecks(body.Attribution(), headSHA, capturedAt, raw.CheckRuns)
		if err != nil {
			return valueobjects.Document{}, nil, fmt.Errorf("could not build the pull request checks: %w", err)
		}
		skipped = append(skipped, checksSkipped...)
		if len(checks.Runs()) > 0 {
			entries = append(entries, checks)
		}
	} else {
		if len(raw.ParentIssue) > 0 {
			parent, err := BuildParentIssue(body.Attribution(), raw.ParentIssue)
			if err != nil {
				return valueobjects.Document{}, nil, fmt.Errorf("could not build the parent issue: %w", err)
			}
			entries = append(entries, parent)
		}

		subIssues, subIssuesSkipped, err := BuildSubIssues(body.Attribution(), raw.SubIssues)
		if err != nil {
			return valueobjects.Document{}, nil, fmt.Errorf("could not build the sub-issues: %w", err)
		}
		skipped = append(skipped, subIssuesSkipped...)
		if len(subIssues.Children()) > 0 {
			entries = append(entries, subIssues)
		}
	}
	entries = append(entries, classified...)

	doc, err := valueobjects.NewDocument(title, entries)
	if err != nil {
		return valueobjects.Document{}, nil, fmt.Errorf("assemble document: %w", err)
	}
	return doc, skipped, nil
}
