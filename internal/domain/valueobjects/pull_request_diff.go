package valueobjects

import (
	"fmt"
	"io"
	"slices"
	"strings"
)

// PullRequestDiff is a pull request's changed files, sourced from GET
// /pulls/{number}/files. Unlike the other Tier 1 types, it has no event of
// its own, so its attribution reuses the pull request's own (author,
// created, url) rather than a per-event one. truncated reports whether
// every file's patch was suppressed because the pull request's total
// changed lines exceeded the builder's size threshold (see the services
// package); a file's own empty Patch() can also mean GitHub itself omitted
// it for being individually too large, which truncated does not
// distinguish from the threshold case.
type PullRequestDiff struct {
	attribution Attribution
	files       []ChangedFile
	additions   int
	deletions   int
	truncated   bool
}

// NewPullRequestDiff constructs a PullRequestDiff from its attribution,
// changed files, total additions/deletions, and whether every file's patch
// was suppressed for exceeding a size threshold.
func NewPullRequestDiff(attribution Attribution, files []ChangedFile, additions, deletions int, truncated bool) PullRequestDiff {
	// Cloned so a later mutation of the caller's slice can't silently
	// change this PullRequestDiff after construction (Immutable First).
	return PullRequestDiff{attribution: attribution, files: slices.Clone(files), additions: additions, deletions: deletions, truncated: truncated}
}

// Attribution returns the pull request's own author, creation time, and
// URL (see the PullRequestDiff Godoc for why this isn't a per-event
// attribution).
func (d PullRequestDiff) Attribution() Attribution {
	return d.attribution
}

// Files returns a copy of the pull request's changed files, so mutating
// the returned slice can't affect this PullRequestDiff (Immutable First).
func (d PullRequestDiff) Files() []ChangedFile {
	return slices.Clone(d.files)
}

// Additions returns the pull request's total added lines across every
// changed file.
func (d PullRequestDiff) Additions() int {
	return d.additions
}

// Deletions returns the pull request's total deleted lines across every
// changed file.
func (d PullRequestDiff) Deletions() int {
	return d.deletions
}

// Truncated reports whether every file's patch was suppressed for
// exceeding a size threshold (see the PullRequestDiff Godoc).
func (d PullRequestDiff) Truncated() bool {
	return d.truncated
}

// Equals reports whether d and other have the same attribution, files,
// additions, deletions, and truncated flag.
func (d PullRequestDiff) Equals(other PullRequestDiff) bool {
	return d.attribution.Equals(other.attribution) &&
		slices.EqualFunc(d.files, other.files, ChangedFile.Equals) &&
		d.additions == other.additions &&
		d.deletions == other.deletions &&
		d.truncated == other.truncated
}

// Render writes d's <!-- {"meta":...} --> line, a bullet list of every
// changed file, then each file's own diff hunk (when present) under a
// "**Diff: `filename`**" label in a fenced code block, satisfying Entry.
func (d PullRequestDiff) Render(w io.Writer) error {
	meta := struct {
		attributionMeta
		Files     int  `json:"files"`
		Additions int  `json:"additions"`
		Deletions int  `json:"deletions"`
		Truncated bool `json:"truncated,omitempty"`
		URL       Url  `json:"url"`
	}{
		attributionMeta: newAttributionMeta(d.attribution),
		Files:           len(d.files),
		Additions:       d.additions,
		Deletions:       d.deletions,
		Truncated:       d.truncated,
		URL:             d.attribution.URL(),
	}

	var list strings.Builder
	for _, f := range d.files {
		fmt.Fprintf(&list, "- %s\n", changedFileLine(f))
	}

	if err := writeMetaLine(w, meta, strings.TrimRight(list.String(), "\n")); err != nil {
		return err
	}

	for _, f := range d.files {
		patch := f.Patch()
		if patch == "" {
			continue
		}
		fence := diffFence(patch)
		if _, err := fmt.Fprintf(w, "\n**Diff: `%s`**\n\n%sdiff\n%s\n%s\n", f.Filename(), fence, patch, fence); err != nil {
			return err
		}
	}

	return nil
}

// changedFileLine formats one ChangedFile as a single bullet-list line,
// showing both names for a renamed file rather than only its new one. A
// filename is arbitrary, attacker-influenceable text (a contributor's own
// choice), so each one is fenced with titleCodeSpan rather than a fixed
// backtick pair, keeping it a single unbroken code span even when the
// filename itself contains a backtick.
func changedFileLine(f ChangedFile) string {
	name := titleCodeSpan(f.Filename())
	if f.Status() == FileStatusRenamed && f.PreviousFilename() != "" {
		name = fmt.Sprintf("%s -> %s", titleCodeSpan(f.PreviousFilename()), titleCodeSpan(f.Filename()))
	}
	return fmt.Sprintf("%s (%s, +%d/-%d)", name, f.Status().String(), f.Additions(), f.Deletions())
}

func (PullRequestDiff) entryNode() {}
