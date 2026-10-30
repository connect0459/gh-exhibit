package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/connect0459/gh-exhibit/internal/domain/repositories"
	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type attachmentLinkReader struct {
	baseDir string
}

// NewAttachmentLinkReader builds a repositories.AttachmentLinkReader that
// reads the links an attachmentWriter persisted under baseDir, at
// {repo}/{number}/evidence/attachments.json.
func NewAttachmentLinkReader(baseDir string) repositories.AttachmentLinkReader {
	return &attachmentLinkReader{baseDir: baseDir}
}

// ReadAttachmentLinks implements repositories.AttachmentLinkReader.
func (r *attachmentLinkReader) ReadAttachmentLinks(ctx context.Context, ref valueobjects.IssueRef) ([]valueobjects.AttachmentLink, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(evidencePath(r.baseDir, ref, "attachments.json"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("could not read attachments.json: %w", err)
	}

	var wire []struct {
		URL  string `json:"url"`
		Path string `json:"path"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return nil, fmt.Errorf("attachments.json is not a list of url/path pairs: %w", err)
	}

	links := make([]valueobjects.AttachmentLink, 0, len(wire))
	for i, w := range wire {
		url, err := valueobjects.NewUrl(w.URL)
		if err != nil {
			return nil, fmt.Errorf("attachments.json entry %d is invalid: %w", i, err)
		}
		link, err := valueobjects.NewAttachmentLink(url, w.Path)
		if err != nil {
			return nil, fmt.Errorf("attachments.json entry %d is invalid: %w", i, err)
		}
		links = append(links, link)
	}
	return links, nil
}
