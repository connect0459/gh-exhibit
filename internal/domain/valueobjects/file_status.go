package valueobjects

import "fmt"

// FileStatus is a ChangedFile's status, mirroring GitHub's REST
// GET /pulls/{number}/files "status" field.
type FileStatus int

const (
	FileStatusAdded FileStatus = iota
	FileStatusRemoved
	FileStatusModified
	FileStatusRenamed
	FileStatusCopied
	FileStatusChanged
	FileStatusUnchanged
)

// ParseFileStatus parses GitHub's REST changed-file status field ("added",
// "removed", "modified", "renamed", "copied", "changed", "unchanged") into
// a FileStatus. It returns an error for any other value.
func ParseFileStatus(raw string) (FileStatus, error) {
	switch raw {
	case "added":
		return FileStatusAdded, nil
	case "removed":
		return FileStatusRemoved, nil
	case "modified":
		return FileStatusModified, nil
	case "renamed":
		return FileStatusRenamed, nil
	case "copied":
		return FileStatusCopied, nil
	case "changed":
		return FileStatusChanged, nil
	case "unchanged":
		return FileStatusUnchanged, nil
	default:
		return 0, fmt.Errorf("unrecognized file status %q", raw)
	}
}

// String returns s's GitHub API spelling.
func (s FileStatus) String() string {
	switch s {
	case FileStatusAdded:
		return "added"
	case FileStatusRemoved:
		return "removed"
	case FileStatusModified:
		return "modified"
	case FileStatusRenamed:
		return "renamed"
	case FileStatusCopied:
		return "copied"
	case FileStatusChanged:
		return "changed"
	case FileStatusUnchanged:
		return "unchanged"
	default:
		return fmt.Sprintf("FileStatus(%d)", int(s))
	}
}
