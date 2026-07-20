package services

import "encoding/json"

// SkipNote records a single timeline item or review comment that could not
// be classified, together with the raw JSON that caused it, so a caller can
// persist an audit trail of what was dropped instead of losing it silently.
type SkipNote struct {
	Reason string
	Raw    json.RawMessage
}
