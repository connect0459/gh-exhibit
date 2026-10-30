package valueobjects

import "errors"

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
// rendered document. It returns an error if localPath is empty.
func NewAttachmentLink(url Url, localPath string) (AttachmentLink, error) {
	if localPath == "" {
		return AttachmentLink{}, errors.New("attachment link local path must not be empty")
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
