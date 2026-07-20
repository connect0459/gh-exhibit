package valueobjects

import (
	"errors"
	"fmt"
	"io"
	"slices"
)

// Document is the full rendered output for a single issue or pull request:
// an H1 title line followed by each Tier 1 entry's Render() output, joined
// by a "------" separator line (see docs/specs/README.md's Markdown
// dialect).
type Document struct {
	title   string
	entries []Entry
}

// NewDocument constructs a Document from a non-empty title and an ordered
// list of entries. It returns an error if title is empty or if any entry is
// nil.
func NewDocument(title string, entries []Entry) (Document, error) {
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
	return Document{title: title, entries: slices.Clone(entries)}, nil
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

// Render writes the H1 title line followed by each entry's Render() output,
// separated by "------" lines (see docs/specs/README.md's Markdown
// dialect).
func (d Document) Render(w io.Writer) error {
	if _, err := fmt.Fprintf(w, "# %s\n\n", d.title); err != nil {
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
