package valueobjects

import (
	"fmt"
	"strings"
)

// AssetFilename identifies a downloaded attachment's on-disk filename
// under an issue/PR's assets directory. A smart constructor guarantees it
// is always a single, path-safe segment, so no caller needs to
// re-validate it before joining it into a filesystem path or a rendered
// Markdown link.
type AssetFilename struct {
	value string
}

// NewAssetFilename validates filename and returns an AssetFilename. It
// rejects anything that isn't a single path-safe segment: empty, "." or
// "..", or containing a path separator — any of which could otherwise
// escape the intended assets directory once joined into a filesystem
// path.
//
// The separator check scans directly for '/' and '\', not a comparison
// against filepath.Base(filename): Base has a fixed point at "/" itself
// (filepath.Base("/") == "/"), which a "did Base change it" comparison
// would miss entirely, and checking only the host OS's own
// filepath.Separator would miss a backslash on a non-Windows build even
// though gh-exhibit is distributed for Windows too.
func NewAssetFilename(filename string) (AssetFilename, error) {
	if filename == "" {
		return AssetFilename{}, fmt.Errorf("attachment filename must not be empty")
	}
	if isAllDotsWithOptionalTrailingSpaces(filename) {
		return AssetFilename{}, fmt.Errorf("attachment filename must not be %q", filename)
	}
	if strings.ContainsAny(filename, `/\`) {
		return AssetFilename{}, fmt.Errorf("attachment filename %q must not contain a path separator", filename)
	}
	return AssetFilename{value: filename}, nil
}

// isAllDotsWithOptionalTrailingSpaces reports whether s, once any trailing
// spaces are stripped, consists entirely of '.' characters — generalizing
// the exact "."/".." rejection to the same "resolves to a traversal-like
// segment" property this type's constructor already claims to guarantee.
// Trailing dots and spaces in a path component are documented as
// significant on some Win32 file-handling code paths, which gh-exhibit's
// Windows distribution target makes relevant here even though "." and
// ".." are the only two segments POSIX itself treats specially.
func isAllDotsWithOptionalTrailingSpaces(s string) bool {
	trimmed := strings.TrimRight(s, " ")
	if trimmed == "" {
		return false
	}
	return strings.Trim(trimmed, ".") == ""
}

// String returns filename's raw value.
func (f AssetFilename) String() string {
	return f.value
}
