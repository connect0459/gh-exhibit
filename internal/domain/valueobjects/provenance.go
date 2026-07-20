package valueobjects

import "errors"

// Provenance identifies which tool, version, and commit produced a
// Document, so a Document taken out of its own repository/directory
// context still carries a record of its own origin. This is a
// self-reported identifier, not a tamper-resistant guarantee: nothing
// prevents a different tool, or a hand-written file, from claiming the
// same values.
type Provenance struct {
	tool    string
	version string
	commit  string
}

// NewProvenance constructs a Provenance from tool (e.g. "owner/repo"),
// version, and commit. It returns an error if any of the three is empty.
func NewProvenance(tool, version, commit string) (Provenance, error) {
	if tool == "" {
		return Provenance{}, errors.New("provenance tool must not be empty")
	}
	if version == "" {
		return Provenance{}, errors.New("provenance version must not be empty")
	}
	if commit == "" {
		return Provenance{}, errors.New("provenance commit must not be empty")
	}
	return Provenance{tool: tool, version: version, commit: commit}, nil
}

// Tool returns the identifier (e.g. "owner/repo") of the tool that
// produced a Document.
func (p Provenance) Tool() string {
	return p.tool
}

// Version returns the tool's version string.
func (p Provenance) Version() string {
	return p.version
}

// Commit returns the tool's build commit hash.
func (p Provenance) Commit() string {
	return p.commit
}

// Equals reports whether p and other were constructed from the same tool,
// version, and commit.
func (p Provenance) Equals(other Provenance) bool {
	return p.tool == other.tool && p.version == other.version && p.commit == other.commit
}
