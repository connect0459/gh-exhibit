package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// checkRunWire is the shape of one element of GET
// /repos/{owner}/{repo}/commits/{sha}/check-runs' own "check_runs" array.
// conclusion is empty until status reaches "completed".
type checkRunWire struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	HTMLURL    string `json:"html_url"`
}

// BuildPullRequestChecks constructs the PullRequestChecks Tier 1 entry from
// rawChecks (from EvidenceFetcher.FetchCheckRuns). attribution is reused
// as-is — typically the same Attribution BuildBody already derived from the
// issue/PR resource — since a check-run snapshot has no event of its own to
// attribute; headSHA and capturedAt are recorded alongside it (see
// valueobjects.PullRequestChecks' Godoc for why capturedAt is distinct from
// attribution's own created time). A check-run item that cannot be parsed
// is recorded as a SkipNote and skipped rather than aborting the whole
// call, matching BuildPullRequestCommits/BuildPullRequestDiff's handling of
// their own item lists.
func BuildPullRequestChecks(attribution valueobjects.Attribution, headSHA string, capturedAt time.Time, rawChecks []json.RawMessage) (valueobjects.PullRequestChecks, []SkipNote, error) {
	var skipped []SkipNote
	runs := make([]valueobjects.CheckRun, 0, len(rawChecks))
	for _, raw := range rawChecks {
		run, err := buildCheckRun(raw)
		if err != nil {
			skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
			continue
		}
		runs = append(runs, run)
	}

	return valueobjects.NewPullRequestChecks(attribution, headSHA, capturedAt, runs), skipped, nil
}

func buildCheckRun(raw json.RawMessage) (valueobjects.CheckRun, error) {
	var w checkRunWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return valueobjects.CheckRun{}, fmt.Errorf("unmarshal check run: %w", err)
	}

	outcome, err := valueobjects.ParseCheckOutcome(w.Status, w.Conclusion)
	if err != nil {
		return valueobjects.CheckRun{}, fmt.Errorf("check run outcome: %w", err)
	}

	run, err := valueobjects.NewCheckRun(w.Name, outcome, w.HTMLURL)
	if err != nil {
		return valueobjects.CheckRun{}, fmt.Errorf("check run: %w", err)
	}
	return run, nil
}
