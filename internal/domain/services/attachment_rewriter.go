package services

import "github.com/connect0459/gh-exhibit/internal/domain/valueobjects"

// NewAttachmentRewriter returns a function that rewrites every host
// attachment URL in a Markdown body: to its downloaded copy's local path
// when links records one, and to an "attachment unavailable" placeholder
// otherwise, so a view built from an export never hotlinks GitHub's CDN.
func NewAttachmentRewriter(host string, links []valueobjects.AttachmentLink) func(markdown string) string {
	localPaths := make(map[string]string, len(links))
	for _, l := range links {
		localPaths[l.URL().String()] = l.LocalPath()
	}

	return func(markdown string) string {
		attachments := Detect([]byte(markdown), host)
		if len(attachments) == 0 {
			return markdown
		}

		resolutions := make([]Resolution, 0, len(attachments))
		for _, a := range attachments {
			if path, ok := localPaths[a.URL().String()]; ok {
				resolutions = append(resolutions, Downloaded(a.URL(), path))
				continue
			}
			resolutions = append(resolutions, FetchFailed(a.URL(), "not downloaded by this export"))
		}
		return string(Rewrite([]byte(markdown), resolutions))
	}
}
