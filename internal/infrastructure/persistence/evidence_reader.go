package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/services"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// evidenceReader implements repositories.EvidenceReader against the local
// filesystem.
type evidenceReader struct {
	baseDir string
}

// NewEvidenceReader builds a repositories.EvidenceReader that reads the
// raw evidence an evidenceWriter persisted under baseDir, at
// {repo}/{number}/evidence/.
func NewEvidenceReader(baseDir string) repositories.EvidenceReader {
	return &evidenceReader{baseDir: baseDir}
}

// ReadEvidence implements repositories.EvidenceReader.
func (r *evidenceReader) ReadEvidence(ctx context.Context, ref valueobjects.IssueRef) (services.RawEvidence, error) {
	if err := ctx.Err(); err != nil {
		return services.RawEvidence{}, err
	}

	issue, err := os.ReadFile(evidencePath(r.baseDir, ref, "issue.json"))
	if err != nil {
		return services.RawEvidence{}, fmt.Errorf("could not read the issue resource: %w", err)
	}

	raw := services.RawEvidence{Issue: issue}
	if raw.PullRequest, err = r.readOptional(ref, "pull.json"); err != nil {
		return services.RawEvidence{}, err
	}
	if raw.ParentIssue, err = r.readOptional(ref, "parent-issue.json"); err != nil {
		return services.RawEvidence{}, err
	}

	arrays := []struct {
		filename string
		into     *[]json.RawMessage
	}{
		{"timeline.json", &raw.Timeline},
		{"review-comments.json", &raw.ReviewComments},
		{"pull-files.json", &raw.PullRequestFiles},
		{"pull-commits.json", &raw.PullRequestCommits},
		{"sub-issues.json", &raw.SubIssues},
		{"check-runs.json", &raw.CheckRuns},
	}
	for _, a := range arrays {
		if *a.into, err = r.readOptionalArray(ref, a.filename); err != nil {
			return services.RawEvidence{}, err
		}
	}
	return raw, nil
}

func (r *evidenceReader) readOptional(ref valueobjects.IssueRef, filename string) ([]byte, error) {
	data, err := os.ReadFile(evidencePath(r.baseDir, ref, filename))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read %s: %w", filename, err)
	}
	return data, nil
}

func (r *evidenceReader) readOptionalArray(ref valueobjects.IssueRef, filename string) ([]json.RawMessage, error) {
	data, err := r.readOptional(ref, filename)
	if err != nil || data == nil {
		return nil, err
	}

	var items []json.RawMessage
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("%s is not a JSON array: %w", filename, err)
	}
	return items, nil
}
