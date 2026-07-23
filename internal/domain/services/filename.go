package services

import (
	"mime"
	"path"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// extensionsByContentType is an explicit, hermetic lookup rather than
// mime.ExtensionsByType, whose result set is drawn from the host's own
// mime database and is not guaranteed stable across platforms — this
// project's own no-flaky-tests precedent (see the retry/pagination
// packages' fake-boundary tests) argues against relying on it here.
var extensionsByContentType = map[string]string{
	"image/png":       ".png",
	"image/jpeg":      ".jpg",
	"image/gif":       ".gif",
	"image/webp":      ".webp",
	"image/svg+xml":   ".svg",
	"video/mp4":       ".mp4",
	"application/pdf": ".pdf",
	"text/plain":      ".txt",
	"application/zip": ".zip",
}

// Filename derives a's local asset filename: the id GitHub assigns in its
// URL path (unique and stable, a UUID on github.com itself) as the base
// name, plus an extension resolved from the response's Content-Type header
// — the URL path itself does not reliably encode one. An unrecognized
// content type yields no extension rather than a guessed one. The error
// return is real, not defensive: a's own URL is only validated by
// NewAttachment to match a GitHub user-attachments asset path's shape, and
// on an untrusted GitHub Enterprise Server host that id segment is fully
// server-controlled and can fail valueobjects.NewAssetFilename, e.g. by
// exceeding its maximum length.
func (a Attachment) Filename(contentType string) (valueobjects.AssetFilename, error) {
	id := path.Base(a.url.Path())

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return valueobjects.NewAssetFilename(id)
	}

	ext, ok := extensionsByContentType[mediaType]
	if !ok {
		return valueobjects.NewAssetFilename(id)
	}
	return valueobjects.NewAssetFilename(id + ext)
}
