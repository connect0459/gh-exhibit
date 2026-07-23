package valueobjects_test

import (
	"strings"
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func newTestAssetFilename(t *testing.T, filename string) valueobjects.AssetFilename {
	t.Helper()

	f, err := valueobjects.NewAssetFilename(filename)
	if err != nil {
		t.Fatalf("NewAssetFilename(%q) error = %v", filename, err)
	}
	return f
}

func TestNewAssetFilename_AcceptsAnOrdinaryFilename(t *testing.T) {
	got := newTestAssetFilename(t, "abc-123.png")

	if got.String() != "abc-123.png" {
		t.Fatalf("String() = %q, want %q", got.String(), "abc-123.png")
	}
}

func TestNewAssetFilename_RejectsAnEmptyFilename(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename(""); err == nil {
		t.Fatal("NewAssetFilename(\"\") error = nil, want an error for an empty filename")
	}
}

func TestNewAssetFilename_RejectsDot(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename("."); err == nil {
		t.Fatal(`NewAssetFilename(".") error = nil, want an error for a filename equal to "."`)
	}
}

func TestNewAssetFilename_RejectsDotDot(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename(".."); err == nil {
		t.Fatal(`NewAssetFilename("..") error = nil, want an error for a filename equal to ".."`)
	}
}

func TestNewAssetFilename_RejectsAFilenameThatIsAllDots(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename("..."); err == nil {
		t.Fatal(`NewAssetFilename("...") error = nil, want an error for a filename that is entirely dots`)
	}
}

func TestNewAssetFilename_RejectsAFilenameThatIsDotsWithTrailingSpaces(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename(".. "); err == nil {
		t.Fatal(`NewAssetFilename(".. ") error = nil, want an error for a filename that is dots followed by trailing spaces`)
	}
}

func TestNewAssetFilename_RejectsAFilenameContainingADotDotSegment(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename("../../../../tmp/evil"); err == nil {
		t.Fatal("NewAssetFilename() error = nil, want an error for a filename containing a \"..\" segment")
	}
}

func TestNewAssetFilename_RejectsAFilenameContainingAPathSeparator(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename("sub/evil.png"); err == nil {
		t.Fatal("NewAssetFilename() error = nil, want an error for a filename containing a path separator")
	}
}

func TestNewAssetFilename_RejectsAnAbsolutePathFilename(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename("/etc/cron.d/evil"); err == nil {
		t.Fatal("NewAssetFilename() error = nil, want an error for an absolute-path filename")
	}
}

func TestNewAssetFilename_RejectsASingleSlash(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename("/"); err == nil {
		t.Fatal(`NewAssetFilename("/") error = nil, want an error for a filename equal to "/" (filepath.Base("/") == "/", a fixed point a naive Base comparison would miss)`)
	}
}

func TestNewAssetFilename_RejectsAFilenameContainingABackslash(t *testing.T) {
	if _, err := valueobjects.NewAssetFilename(`sub\evil.png`); err == nil {
		t.Fatal("NewAssetFilename() error = nil, want an error for a filename containing a backslash, regardless of the host OS's own path separator")
	}
}

func TestNewAssetFilename_AcceptsAFilenameAtTheMaximumLength(t *testing.T) {
	name := strings.Repeat("a", 255)

	got := newTestAssetFilename(t, name)

	if got.String() != name {
		t.Fatalf("String() = %q, want %q", got.String(), name)
	}
}

func TestNewAssetFilename_RejectsAFilenameExceedingTheMaximumLength(t *testing.T) {
	name := strings.Repeat("a", 256)

	if _, err := valueobjects.NewAssetFilename(name); err == nil {
		t.Fatal("NewAssetFilename() error = nil, want an error for a filename exceeding the filesystem's single-path-component limit")
	}
}
