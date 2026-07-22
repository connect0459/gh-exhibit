package services

import (
	"encoding/json"
	"fmt"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// maxDiffTotalLines bounds how many total changed lines (additions plus
// deletions, as reported by the pull request resource itself) a pull
// request may have before BuildPullRequestDiff suppresses every file's
// patch, keeping only the changed-file list (filename, status, additions,
// deletions).
const maxDiffTotalLines = 1000

// changedFileWire is the shape of one element of GET
// /pulls/{number}/files. previous_filename is only present for a renamed
// file; patch is absent for a file GitHub itself considers too large to
// diff.
type changedFileWire struct {
	Filename         string `json:"filename"`
	PreviousFilename string `json:"previous_filename"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Patch            string `json:"patch"`
}

// BuildPullRequestDiff constructs the PullRequestDiff Tier 1 entry from
// the pull request resource (for its total additions/deletions) and its
// changed-file list (rawFiles, from EvidenceFetcher.FetchPullRequestFiles).
// attribution is reused as-is — typically the same Attribution BuildBody
// already derived from the issue/PR resource — since a diff has no event
// of its own to attribute. Every file's patch is suppressed, while its
// filename/status/additions/deletions are still included, once the pull
// request's total changed lines exceed maxDiffTotalLines. A changed-file
// item that cannot be parsed is recorded as a SkipNote and skipped rather
// than aborting the whole call; a malformed rawPullRequest returns an
// error, matching BuildBody's own handling of the same resource.
func BuildPullRequestDiff(attribution valueobjects.Attribution, rawPullRequest json.RawMessage, rawFiles []json.RawMessage) (valueobjects.PullRequestDiff, []SkipNote, error) {
	var pw pullRequestResourceWire
	if err := json.Unmarshal(rawPullRequest, &pw); err != nil {
		return valueobjects.PullRequestDiff{}, nil, fmt.Errorf("unmarshal pull request resource: %w", err)
	}

	truncated := pw.Additions+pw.Deletions > maxDiffTotalLines

	var skipped []SkipNote
	files := make([]valueobjects.ChangedFile, 0, len(rawFiles))
	for _, raw := range rawFiles {
		file, err := buildChangedFile(raw, truncated)
		if err != nil {
			skipped = append(skipped, SkipNote{Reason: err.Error(), Raw: raw})
			continue
		}
		files = append(files, file)
	}

	return valueobjects.NewPullRequestDiff(attribution, files, pw.Additions, pw.Deletions, truncated), skipped, nil
}

func buildChangedFile(raw json.RawMessage, truncated bool) (valueobjects.ChangedFile, error) {
	var w changedFileWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return valueobjects.ChangedFile{}, fmt.Errorf("unmarshal pull request changed file: %w", err)
	}

	status, err := valueobjects.ParseFileStatus(w.Status)
	if err != nil {
		return valueobjects.ChangedFile{}, fmt.Errorf("pull request changed file status: %w", err)
	}

	patch := w.Patch
	if truncated {
		patch = ""
	}

	file, err := valueobjects.NewChangedFile(w.Filename, w.PreviousFilename, status, w.Additions, w.Deletions, patch)
	if err != nil {
		return valueobjects.ChangedFile{}, fmt.Errorf("pull request changed file: %w", err)
	}
	return file, nil
}
