package valueobjects

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

// attributionMeta is the (author, created) prefix every Tier 1 entry's meta
// line shares. Embedding it first in a type's own meta struct reproduces
// that shared prefix in the marshaled JSON; url is declared separately per
// type since it's always last, after whatever type-specific fields sit
// between created and url.
type attributionMeta struct {
	Author  string `json:"author"`
	Created string `json:"created"`
}

func newAttributionMeta(a Attribution) attributionMeta {
	return attributionMeta{
		Author:  a.Author(),
		Created: a.CreatedAt().UTC().Format(time.RFC3339),
	}
}

// writeMetaLine writes meta as a single line-anchored `meta:{...}` JSON
// line followed by body, the shape every Tier 1 entry's Render() shares
// (ADR-001's Markdown dialect).
func writeMetaLine(w io.Writer, meta any, body string) error {
	line, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal meta: %w", err)
	}

	body = normalizeBody(body)
	if body == "" {
		_, err = fmt.Fprintf(w, "meta:%s\n", line)
		return err
	}

	_, err = fmt.Fprintf(w, "meta:%s\n\n%s\n", line, body)
	return err
}

// normalizeBody converts CRLF/CR line endings to LF and strips trailing
// newlines, so every Tier 1 entry's Render() output ends in exactly one
// newline regardless of the raw GitHub body's own line-ending convention or
// any trailing blank lines the author happened to type — otherwise the
// fixed single trailing "\n" this format already appends would double up
// with one the body already has, and a Document's "------" separator would
// pick up an extra blank line only for entries whose raw body happened to
// end in "\n".
func normalizeBody(body string) string {
	body = strings.ReplaceAll(body, "\r\n", "\n")
	body = strings.ReplaceAll(body, "\r", "\n")
	return strings.TrimRight(body, "\n")
}
