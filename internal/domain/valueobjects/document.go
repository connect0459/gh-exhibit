package valueobjects

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
)

// Document is the full rendered output for a single issue or pull request:
// an H1 title line, a hidden HTML-comment line recording the Provenance
// that produced it, then each Tier 1 entry's Render() output, joined by a
// "------" separator line.
type Document struct {
	title      string
	entries    []Entry
	provenance Provenance
}

// NewDocument constructs a Document from a non-empty title, an ordered
// list of entries, and the Provenance identifying which tool produced it.
// It returns an error if title is empty or if any entry is nil; provenance
// is trusted as already validated by NewProvenance, the same trust every
// other Value Object parameter here already gets.
func NewDocument(title string, entries []Entry, provenance Provenance) (Document, error) {
	if title == "" {
		return Document{}, errors.New("document title must not be empty")
	}
	for i, e := range entries {
		if e == nil {
			return Document{}, fmt.Errorf("document entry %d must not be nil", i)
		}
	}
	// Cloned so a later mutation of the caller's slice (or of a slice this
	// constructor was handed) can't silently change this Document after
	// construction (Immutable First).
	return Document{title: title, entries: slices.Clone(entries), provenance: provenance}, nil
}

// Title returns the issue/PR title rendered as the document's H1 line.
func (d Document) Title() string {
	return d.title
}

// Entries returns a copy, so mutating the returned slice can't affect this
// Document's own state (Immutable First).
func (d Document) Entries() []Entry {
	return slices.Clone(d.entries)
}

// Provenance returns which tool produced this Document.
func (d Document) Provenance() Provenance {
	return d.provenance
}

// Render writes the H1 title line, the hidden provenance comment line,
// then each entry's Render() output, separated by "------" lines.
func (d Document) Render(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# %s\n\n", d.title); err != nil {
		return err
	}

	if err := writeProvenanceLine(w, d.provenance); err != nil {
		return err
	}

	for i, e := range d.entries {
		if i > 0 {
			if _, err := io.WriteString(w, "\n------\n\n"); err != nil {
				return err
			}
		}
		if err := e.Render(w); err != nil {
			return err
		}
	}

	return nil
}

// writeProvenanceLine writes provenance as a single line-anchored
// `<!-- {"tool":...,"version":...,"commit":...} -->` HTML comment followed
// by a blank line — the same hidden-comment shape writeMetaLine gives each
// entry's own meta line, at the document level instead of per entry.
func writeProvenanceLine(w io.Writer, provenance Provenance) error {
	line, err := json.Marshal(struct {
		Tool    string `json:"tool"`
		Version string `json:"version"`
		Commit  string `json:"commit"`
	}{Tool: provenance.tool, Version: provenance.version, Commit: provenance.commit})
	if err != nil {
		return fmt.Errorf("marshal provenance: %w", err)
	}

	_, err = fmt.Fprintf(w, "<!-- %s -->\n\n", line)
	return err
}
