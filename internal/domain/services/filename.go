package services

import (
	"mime"
	"path"
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

// Filename derives the local asset filename for an attachment URL: the
// UUID GitHub assigns in the URL path (unique and stable) as the base
// name, plus an extension resolved from the response's Content-Type header
// (ADR-002) — the URL path itself does not reliably encode one. An
// unrecognized content type yields no extension rather than a guessed one.
func Filename(url, contentType string) string {
	id := path.Base(url)

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return id
	}

	ext, ok := extensionsByContentType[mediaType]
	if !ok {
		return id
	}
	return id + ext
}
