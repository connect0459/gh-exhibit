package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// commitIdentityWire is the shape of GET /pulls/{number}/commits'
// commit.author or commit.committer object: a git-level name and
// timestamp, distinct from the top-level author/committer GitHub user
// objects (which can be null when the commit's email has no matched
// GitHub account).
type commitIdentityWire struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
}

// commitWire is the shape of one element of GET /pulls/{number}/commits.
type commitWire struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author    commitIdentityWire `json:"author"`
		Committer commitIdentityWire `json:"committer"`
		Message   string             `json:"message"`
	} `json:"commit"`
}

// BuildPullRequestCommits constructs the PullRequestCommits Tier 1 entry
// from rawCommits (from EvidenceFetcher.FetchPullRequestCommits).
// attribution is reused as-is — typically the same Attribution BuildBody
// already derived from the issue/PR resource — since a commit list has no
// event of its own to attribute. A commit item that cannot be parsed is
// recorded as a SkipNote and skipped rather than aborting the whole call.
func BuildPullRequestCommits(attribution valueobjects.Attribution, rawCommits []json.RawMessage) (valueobjects.PullRequestCommits, []SkipNote, error) {
	var skipped []SkipNote
	commits := make([]valueobjects.Commit, 0, len(rawCommits))
	for _, raw := range rawCommits {
		commit, err := buildCommit(raw)
		if err != nil {
			skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
			continue
		}
		commits = append(commits, commit)
	}

	return valueobjects.NewPullRequestCommits(attribution, commits), skipped, nil
}

func buildCommit(raw json.RawMessage) (valueobjects.Commit, error) {
	var w commitWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return valueobjects.Commit{}, fmt.Errorf("unmarshal pull request commit: %w", err)
	}

	commit, err := valueobjects.NewCommit(w.SHA, w.Commit.Author.Name, w.Commit.Author.Date, w.Commit.Committer.Name, w.Commit.Committer.Date, w.Commit.Message)
	if err != nil {
		return valueobjects.Commit{}, fmt.Errorf("pull request commit: %w", err)
	}
	return commit, nil
}
