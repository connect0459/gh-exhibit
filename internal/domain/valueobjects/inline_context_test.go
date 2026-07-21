package valueobjects_test

import (
	"testing"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

func intPtr(v int) *int {
	return &v
}

func TestNewInlineContext_RejectsAnEmptyPath(t *testing.T) {
	_, err := valueobjects.NewInlineContext("", intPtr(195), nil, "@@ -1,3 +1,3 @@", false)

	if err == nil {
		t.Fatal("expected an error for an empty path, got nil")
	}
}

func TestNewInlineContext_RejectsANonPositiveLine(t *testing.T) {
	_, err := valueobjects.NewInlineContext("docs/example.md", intPtr(0), nil, "@@ -1,3 +1,3 @@", false)

	if err == nil {
		t.Fatal("expected an error for a non-positive line, got nil")
	}
}

func TestNewInlineContext_AcceptsANilLineForAFileLevelComment(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", nil, nil, "", false)

	if err != nil {
		t.Fatalf("expected a nil line to be accepted, got error: %v", err)
	}
	if ctx.Line() != nil {
		t.Fatalf("Line() = %v, want nil", ctx.Line())
	}
}

func TestNewInlineContext_RejectsOutdatedWithoutALine(t *testing.T) {
	_, err := valueobjects.NewInlineContext("docs/example.md", nil, nil, "", true)

	if err == nil {
		t.Fatal("expected an error for outdated=true with no line, got nil")
	}
}

func TestNewInlineContext_AcceptsAnEmptyDiffHunkForAnOutdatedComment(t *testing.T) {
	_, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "", true)

	if err != nil {
		t.Fatalf("expected an empty diff hunk to be accepted, got error: %v", err)
	}
}

func TestNewInlineContext_AcceptsAStartLineLessThanLineForARangeComment(t *testing.T) {
	ctx, err := valueobjects.NewInlineContext("docs/example.md", intPtr(15), intPtr(10), "@@ -1,3 +1,3 @@", false)

	if err != nil {
		t.Fatalf("expected a start line less than line to be accepted, got error: %v", err)
	}
	if ctx.StartLine() == nil || *ctx.StartLine() != 10 {
		t.Fatalf("StartLine() = %v, want 10", ctx.StartLine())
	}
}

func TestNewInlineContext_RejectsAStartLineWithoutALine(t *testing.T) {
	_, err := valueobjects.NewInlineContext("docs/example.md", nil, intPtr(10), "", false)

	if err == nil {
		t.Fatal("expected an error for a start line with no line, got nil")
	}
}

func TestNewInlineContext_RejectsANonPositiveStartLine(t *testing.T) {
	_, err := valueobjects.NewInlineContext("docs/example.md", intPtr(15), intPtr(0), "@@ -1,3 +1,3 @@", false)

	if err == nil {
		t.Fatal("expected an error for a non-positive start line, got nil")
	}
}

func TestNewInlineContext_RejectsAStartLineNotLessThanLine(t *testing.T) {
	_, err := valueobjects.NewInlineContext("docs/example.md", intPtr(15), intPtr(15), "@@ -1,3 +1,3 @@", false)

	if err == nil {
		t.Fatal("expected an error for a start line equal to line, got nil")
	}
}

func TestInlineContext_Outdated_ReturnsWhetherTheContextWasConstructedAsOutdated(t *testing.T) {
	current, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	if current.Outdated() {
		t.Fatal("expected a non-outdated context to report Outdated() == false")
	}

	outdated, err := valueobjects.NewInlineContext("docs/example.md", intPtr(346), nil, "", true)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	if !outdated.Outdated() {
		t.Fatal("expected an outdated context to report Outdated() == true")
	}
}

func TestInlineContext_Equals_TreatsMatchingValuesAsEqual(t *testing.T) {
	a, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	b, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}

	if !a.Equals(b) {
		t.Fatal("expected inline contexts with matching path, line, diff hunk, and outdated flag to be equal")
	}
}

func TestInlineContext_Equals_TreatsDifferentLinesAsNotEqual(t *testing.T) {
	a, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	b, err := valueobjects.NewInlineContext("docs/example.md", intPtr(196), nil, "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}

	if a.Equals(b) {
		t.Fatal("expected inline contexts with different lines to not be equal")
	}
}

func TestInlineContext_Equals_TreatsANilLineAndAPresentLineAsNotEqual(t *testing.T) {
	a, err := valueobjects.NewInlineContext("docs/example.md", nil, nil, "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	b, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}

	if a.Equals(b) {
		t.Fatal("expected a context with no line to not equal one with a line")
	}
}

func TestInlineContext_Equals_TreatsDifferentOutdatedFlagsAsNotEqual(t *testing.T) {
	a, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	b, err := valueobjects.NewInlineContext("docs/example.md", intPtr(195), nil, "@@ -1,3 +1,3 @@", true)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}

	if a.Equals(b) {
		t.Fatal("expected inline contexts with different outdated flags to not be equal")
	}
}

func TestInlineContext_Equals_TreatsDifferentStartLinesAsNotEqual(t *testing.T) {
	a, err := valueobjects.NewInlineContext("docs/example.md", intPtr(15), intPtr(10), "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}
	b, err := valueobjects.NewInlineContext("docs/example.md", intPtr(15), intPtr(11), "@@ -1,3 +1,3 @@", false)
	if err != nil {
		t.Fatalf("unexpected error building inline context: %v", err)
	}

	if a.Equals(b) {
		t.Fatal("expected inline contexts with different start lines to not be equal")
	}
}
