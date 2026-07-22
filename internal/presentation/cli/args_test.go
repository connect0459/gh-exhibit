package cli

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestParseArgs_AcceptsASingleIssueNumber(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_AcceptsACommaSeparatedListOfNumbers(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123,124,125"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123, 124, 125}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_TrimsSpacesAroundCommas(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123, 124 ,125"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123, 124, 125}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_DeduplicatesARepeatedNumberInFirstSeenOrder(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123,123,124"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123, 124}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_RejectsANonNumericEntry(t *testing.T) {
	if _, err := ParseArgs([]string{"export", "123,abc"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a non-numeric entry")
	}
}

func TestParseArgs_RejectsAZeroOrNegativeNumber(t *testing.T) {
	for _, in := range []string{"0", "-1", "123,0"} {
		if _, err := ParseArgs([]string{"export", in}); err == nil {
			t.Fatalf("ParseArgs([export %q]) error = nil, want an error for a non-positive number", in)
		}
	}
}

func TestParseArgs_RejectsABareNegativeNumberForTheRightReason(t *testing.T) {
	_, err := ParseArgs([]string{"export", "-1"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a non-positive number")
	}
	if strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("ParseArgs() error = %v, want the positive-number validation error, not a flag-parsing error", err)
	}
}

func TestParseArgs_RejectsACommaSeparatedListOfNegativeNumbersForTheRightReason(t *testing.T) {
	_, err := ParseArgs([]string{"export", "-1,-2"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a non-positive number")
	}
	if strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("ParseArgs() error = %v, want the positive-number validation error, not a flag-parsing error", err)
	}
}

func TestParseArgs_RejectsAnEmptyListEntry(t *testing.T) {
	if _, err := ParseArgs([]string{"export", "123,,124"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty list entry")
	}
}

func TestParseArgs_RejectsAMissingPositionalArgument(t *testing.T) {
	if _, err := ParseArgs([]string{"export"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error when no issue/PR number is given")
	}
}

func TestParseArgs_RejectsMultiplePositionalArguments(t *testing.T) {
	if _, err := ParseArgs([]string{"export", "123", "124"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for more than one positional argument")
	}
}

func TestParseArgs_DefaultsOutputDirToTheCurrentDirectory(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.OutputDir != "." {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, ".")
	}
}

func TestParseArgs_ReadsTheRepoFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--repo", "octocat/hello-world", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Repo != "octocat/hello-world" {
		t.Errorf("Repo = %q, want %q", got.Repo, "octocat/hello-world")
	}
}

func TestParseArgs_ReadsTheOutputFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--output", "/tmp/out", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.OutputDir != "/tmp/out" {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, "/tmp/out")
	}
}

func TestParseArgs_AcceptsTheNumberBeforeTheRepoFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123", "--repo", "octocat/hello-world"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Repo != "octocat/hello-world" {
		t.Errorf("Repo = %q, want %q", got.Repo, "octocat/hello-world")
	}
	if !equalInts(got.Numbers, []int{123}) {
		t.Errorf("Numbers = %v, want [123]", got.Numbers)
	}
}

func TestParseArgs_AcceptsFlagsOnBothSidesOfTheNumber(t *testing.T) {
	got, err := ParseArgs([]string{"export", "--repo", "octocat/hello-world", "123", "--output", "/tmp/out"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Repo != "octocat/hello-world" {
		t.Errorf("Repo = %q, want %q", got.Repo, "octocat/hello-world")
	}
	if got.OutputDir != "/tmp/out" {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, "/tmp/out")
	}
	if !equalInts(got.Numbers, []int{123}) {
		t.Errorf("Numbers = %v, want [123]", got.Numbers)
	}
}

func TestParseArgs_AcceptsAnAttachedFlagValueForm(t *testing.T) {
	got, err := ParseArgs([]string{"export", "123", "--repo=octocat/hello-world"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Repo != "octocat/hello-world" {
		t.Errorf("Repo = %q, want %q", got.Repo, "octocat/hello-world")
	}
}

func TestParseArgs_ReadsTheOutputShorthandFlag(t *testing.T) {
	got, err := ParseArgs([]string{"export", "-o", "/tmp/out", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.OutputDir != "/tmp/out" {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, "/tmp/out")
	}
}

func TestParseArgs_TreatsTokensAfterADoubleDashAsLiteralPositionalText(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--", "--repo"})
	if err == nil {
		t.Fatal(`ParseArgs() error = nil, want an error since "--repo" after -- is not a valid issue/PR number`)
	}
	if strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("ParseArgs() error = %v, want a number-parsing error, not a flag-parsing error (-- must force literal positional text)", err)
	}
}

func TestParseArgs_MissingValueErrorNamesALongFormFlagWithBothDashes(t *testing.T) {
	_, err := ParseArgs([]string{"export", "123", "--repo"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since --repo has no following value")
	}
	if !strings.Contains(err.Error(), "--repo") {
		t.Errorf("ParseArgs() error = %v, want it to name the flag as %q", err, "--repo")
	}
}

func TestParseArgs_MissingValueErrorNamesAShorthandFlagWithASingleDash(t *testing.T) {
	_, err := ParseArgs([]string{"export", "123", "-o"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since -o has no following value")
	}
	if !strings.Contains(err.Error(), "-o") || strings.Contains(err.Error(), "--o") {
		t.Errorf("ParseArgs() error = %v, want it to name the flag as %q", err, "-o")
	}
}

func TestParseArgs_RejectsAShorthandOutputFlagImmediatelyFollowedByAnAttachedFlag(t *testing.T) {
	_, err := ParseArgs([]string{"export", "-o", "--repo=x", "123"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since -o has no value before the next flag")
	}
}

func TestParseArgs_RejectsARepoFlagImmediatelyFollowedByAnAttachedFlag(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--repo", "--output=custom", "123"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since --repo has no value before the next flag")
	}
}

// A value flag immediately followed by "--" is rejected the same as one
// followed by any other flag-shaped token: "--" conventionally means "no
// more flags follow", so treating it as a flag's literal value would be
// the same silent-adjacency misparse this issue targets, just with the
// flag terminator instead of another named flag.
func TestParseArgs_RejectsARepoFlagImmediatelyFollowedByTheFlagTerminator(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--repo", "--", "123"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since --repo has no value before the flag terminator")
	}
}

func TestParseArgs_AcceptsTheVersionFlagWithoutAPositionalArgument(t *testing.T) {
	got, err := ParseArgs([]string{"--version"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if !got.Version {
		t.Errorf("Version = false, want true")
	}
}

func TestParseArgs_TheVersionFlagTakesPriorityOverASubcommand(t *testing.T) {
	got, err := ParseArgs([]string{"--version", "export", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}
	if !got.Version {
		t.Errorf("Version = false, want true")
	}
}

func TestParseArgs_WrapsFlagErrHelpForTheHelpFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("ParseArgs() error = %v, want it to wrap flag.ErrHelp", err)
	}
}

func TestParseArgs_ExportSubcommandWrapsFlagErrHelpForTheHelpFlag(t *testing.T) {
	_, err := ParseArgs([]string{"export", "--help"})
	if !errors.Is(err, flag.ErrHelp) {
		t.Errorf("ParseArgs() error = %v, want it to wrap flag.ErrHelp", err)
	}
}

func TestParseArgs_RejectsAMissingSubcommand(t *testing.T) {
	_, err := ParseArgs([]string{})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error when no subcommand is given")
	}
	if !strings.Contains(err.Error(), "export") {
		t.Errorf("ParseArgs() error = %v, want it to mention the %q subcommand", err, "export")
	}
}

func TestParseArgs_RejectsAnUnknownSubcommand(t *testing.T) {
	_, err := ParseArgs([]string{"frobnicate", "123"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an unrecognized subcommand")
	}
	if !strings.Contains(err.Error(), "frobnicate") {
		t.Errorf("ParseArgs() error = %v, want it to name the unrecognized subcommand %q", err, "frobnicate")
	}
}

// The bare-number invocation ("gh exhibit 123") this project supported
// before this issue is a breaking change removed outright: a positional
// number alone is no longer implicitly "export" and is now read as an
// (unrecognized) subcommand name.
func TestParseArgs_RejectsABareNumberWithoutTheExportSubcommand(t *testing.T) {
	_, err := ParseArgs([]string{"123"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since a bare number is no longer an implicit export")
	}
}

// This targets splitFlagsAndPositional directly rather than ParseArgs
// because a triple-dash token is always rejected by flag.FlagSet.Parse's
// own "bad flag syntax" check before it would ever try to consume a value
// for it — so ParseArgs's returned error is identical whether or not the
// pre-scanner over-consumes the next token. The over-consumption is only
// observable in what the pre-scanner itself classifies as positional.
func TestSplitFlagsAndPositional_DoesNotConsumeTheNextTokenForAThreeDashFlag(t *testing.T) {
	flagArgs, positional, err := splitFlagsAndPositional([]string{"123", "---repo", "456"})
	if err != nil {
		t.Fatalf("splitFlagsAndPositional() error = %v", err)
	}

	wantPositional := []string{"123", "456"}
	if !equalStrings(positional, wantPositional) {
		t.Errorf("positional = %v, want %v", positional, wantPositional)
	}

	wantFlagArgs := []string{"---repo"}
	if !equalStrings(flagArgs, wantFlagArgs) {
		t.Errorf("flagArgs = %v, want %v", flagArgs, wantFlagArgs)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
