package valueobjects

import (
	"errors"
	"fmt"
	"strings"
)

const assetsDirPrefix = "assets/"

// AttachmentLink pairs an attachment's original URL with the relative path
// of its downloaded copy, so a later reader of an export can recover which
// local file stands in for which URL without re-deriving the file name,
// which depends on the Content-Type of the response at download time.
type AttachmentLink struct {
	url       Url
	localPath string
}

// NewAttachmentLink constructs an AttachmentLink from the attachment's
// original url and the localPath of its downloaded copy, relative to the
// rendered document. It returns an error unless localPath names a single
// file directly under assets/, so a link read back from an edited
// attachments.json cannot point outside the export's own assets.
func NewAttachmentLink(url Url, localPath string) (AttachmentLink, error) {
	if localPath == "" {
		return AttachmentLink{}, errors.New("attachment link local path must not be empty")
	}
	name, underAssets := strings.CutPrefix(localPath, assetsDirPrefix)
	if !underAssets {
		return AttachmentLink{}, fmt.Errorf("attachment link local path %q must be directly under %s", localPath, assetsDirPrefix)
	}
	if _, err := NewAssetFilename(name); err != nil {
		return AttachmentLink{}, fmt.Errorf("attachment link local path %q: %w", localPath, err)
	}
	return AttachmentLink{url: url, localPath: localPath}, nil
}

// URL returns the attachment's original URL.
func (l AttachmentLink) URL() Url {
	return l.url
}

// LocalPath returns the relative path of the attachment's downloaded copy.
func (l AttachmentLink) LocalPath() string {
	return l.localPath
}
