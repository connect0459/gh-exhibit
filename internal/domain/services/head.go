package services

import (
	"encoding/json"
	"errors"
	"fmt"
)

type pullRequestHeadWire struct {
	Head struct {
		SHA string `json:"sha"`
	} `json:"head"`
}

// PullRequestHeadSHA extracts the pull request's current head commit sha
// from rawPullRequest (from EvidenceFetcher.FetchPullRequest), needed
// before EvidenceFetcher.FetchCheckRuns can be called: check runs are
// associated with a commit, not with the pull request itself.
func PullRequestHeadSHA(rawPullRequest json.RawMessage) (string, error) {
	var w pullRequestHeadWire
	if err := json.Unmarshal(rawPullRequest, &w); err != nil {
		return "", fmt.Errorf("unmarshal pull request resource: %w", err)
	}
	if w.Head.SHA == "" {
		return "", errors.New("pull request resource has no head commit sha")
	}
	return w.Head.SHA, nil
}
