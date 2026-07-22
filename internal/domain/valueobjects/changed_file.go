package valueobjects

import "errors"

// ChangedFile is one file changed by a pull request, sourced from GET
// /pulls/{number}/files. previousFilename is only populated for a
// FileStatusRenamed file; patch is GitHub's unified diff hunk for this
// file, empty when GitHub itself omits it for an individually oversized
// file, or when PullRequestDiff's builder suppresses it for a pull request
// whose total changed lines exceed its size threshold.
type ChangedFile struct {
	filename         string
	previousFilename string
	status           FileStatus
	additions        int
	deletions        int
	patch            string
}

// NewChangedFile constructs a ChangedFile from its filename, previous
// filename (empty unless status is FileStatusRenamed), status, additions,
// deletions, and patch. It returns an error if filename is empty.
func NewChangedFile(filename, previousFilename string, status FileStatus, additions, deletions int, patch string) (ChangedFile, error) {
	if filename == "" {
		return ChangedFile{}, errors.New("changed file name must not be empty")
	}
	return ChangedFile{
		filename:         filename,
		previousFilename: previousFilename,
		status:           status,
		additions:        additions,
		deletions:        deletions,
		patch:            patch,
	}, nil
}

// Filename returns the file's current path.
func (f ChangedFile) Filename() string {
	return f.filename
}

// PreviousFilename returns the file's prior path, or an empty string
// unless Status is FileStatusRenamed.
func (f ChangedFile) PreviousFilename() string {
	return f.previousFilename
}

// Status returns whether the file was added, removed, modified, renamed,
// copied, changed, or left unchanged.
func (f ChangedFile) Status() FileStatus {
	return f.status
}

// Additions returns the number of lines added to this file.
func (f ChangedFile) Additions() int {
	return f.additions
}

// Deletions returns the number of lines deleted from this file.
func (f ChangedFile) Deletions() int {
	return f.deletions
}

// Patch returns this file's unified diff hunk, or an empty string when
// omitted (see the ChangedFile Godoc for why).
func (f ChangedFile) Patch() string {
	return f.patch
}

// Equals reports whether f and other have the same filename, previous
// filename, status, additions, deletions, and patch.
func (f ChangedFile) Equals(other ChangedFile) bool {
	return f.filename == other.filename &&
		f.previousFilename == other.previousFilename &&
		f.status == other.status &&
		f.additions == other.additions &&
		f.deletions == other.deletions &&
		f.patch == other.patch
}
