package cli

import (
	"errors"
	"flag"
	"strings"
	"testing"
)

func TestParseArgs_AcceptsASingleIssueNumber(t *testing.T) {
	got, err := ParseArgs([]string{"123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_AcceptsACommaSeparatedListOfNumbers(t *testing.T) {
	got, err := ParseArgs([]string{"123,124,125"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123, 124, 125}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_TrimsSpacesAroundCommas(t *testing.T) {
	got, err := ParseArgs([]string{"123, 124 ,125"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123, 124, 125}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_DeduplicatesARepeatedNumberInFirstSeenOrder(t *testing.T) {
	got, err := ParseArgs([]string{"123,123,124"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	want := []int{123, 124}
	if !equalInts(got.Numbers, want) {
		t.Errorf("Numbers = %v, want %v", got.Numbers, want)
	}
}

func TestParseArgs_RejectsANonNumericEntry(t *testing.T) {
	if _, err := ParseArgs([]string{"123,abc"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a non-numeric entry")
	}
}

func TestParseArgs_RejectsAZeroOrNegativeNumber(t *testing.T) {
	for _, in := range []string{"0", "-1", "123,0"} {
		if _, err := ParseArgs([]string{in}); err == nil {
			t.Fatalf("ParseArgs([%q]) error = nil, want an error for a non-positive number", in)
		}
	}
}

func TestParseArgs_RejectsABareNegativeNumberForTheRightReason(t *testing.T) {
	_, err := ParseArgs([]string{"-1"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for a non-positive number")
	}
	if strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("ParseArgs() error = %v, want the positive-number validation error, not a flag-parsing error", err)
	}
}

func TestParseArgs_RejectsAnEmptyListEntry(t *testing.T) {
	if _, err := ParseArgs([]string{"123,,124"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for an empty list entry")
	}
}

func TestParseArgs_RejectsAMissingPositionalArgument(t *testing.T) {
	if _, err := ParseArgs([]string{}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error when no issue/PR number is given")
	}
}

func TestParseArgs_RejectsMultiplePositionalArguments(t *testing.T) {
	if _, err := ParseArgs([]string{"123", "124"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error for more than one positional argument")
	}
}

func TestParseArgs_DefaultsOutputDirToTheCurrentDirectory(t *testing.T) {
	got, err := ParseArgs([]string{"123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.OutputDir != "." {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, ".")
	}
}

func TestParseArgs_ReadsTheRepoFlag(t *testing.T) {
	got, err := ParseArgs([]string{"--repo", "octocat/hello-world", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Repo != "octocat/hello-world" {
		t.Errorf("Repo = %q, want %q", got.Repo, "octocat/hello-world")
	}
}

func TestParseArgs_ReadsTheOutputFlag(t *testing.T) {
	got, err := ParseArgs([]string{"--output", "/tmp/out", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.OutputDir != "/tmp/out" {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, "/tmp/out")
	}
}

func TestParseArgs_AcceptsTheNumberBeforeTheRepoFlag(t *testing.T) {
	got, err := ParseArgs([]string{"123", "--repo", "octocat/hello-world"})
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
	got, err := ParseArgs([]string{"--repo", "octocat/hello-world", "123", "--output", "/tmp/out"})
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
	got, err := ParseArgs([]string{"123", "--repo=octocat/hello-world"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.Repo != "octocat/hello-world" {
		t.Errorf("Repo = %q, want %q", got.Repo, "octocat/hello-world")
	}
}

func TestParseArgs_ReadsTheOutputShorthandFlag(t *testing.T) {
	got, err := ParseArgs([]string{"-o", "/tmp/out", "123"})
	if err != nil {
		t.Fatalf("ParseArgs() error = %v", err)
	}

	if got.OutputDir != "/tmp/out" {
		t.Errorf("OutputDir = %q, want %q", got.OutputDir, "/tmp/out")
	}
}

func TestParseArgs_TreatsTokensAfterADoubleDashAsLiteralPositionalText(t *testing.T) {
	_, err := ParseArgs([]string{"--", "--repo"})
	if err == nil {
		t.Fatal(`ParseArgs() error = nil, want an error since "--repo" after -- is not a valid issue/PR number`)
	}
	if strings.Contains(err.Error(), "flag provided but not defined") {
		t.Errorf("ParseArgs() error = %v, want a number-parsing error, not a flag-parsing error (-- must force literal positional text)", err)
	}
}

func TestParseArgs_ReturnsAnErrorWhenAValueFlagIsTheLastToken(t *testing.T) {
	if _, err := ParseArgs([]string{"123", "--repo"}); err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since --repo has no following value")
	}
}

func TestParseArgs_RejectsAShorthandOutputFlagImmediatelyFollowedByAnAttachedFlag(t *testing.T) {
	_, err := ParseArgs([]string{"-o", "--repo=x", "123"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since -o has no value before the next flag")
	}
}

func TestParseArgs_RejectsARepoFlagImmediatelyFollowedByAnAttachedFlag(t *testing.T) {
	_, err := ParseArgs([]string{"--repo", "--output=custom", "123"})
	if err == nil {
		t.Fatal("ParseArgs() error = nil, want an error since --repo has no value before the next flag")
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

func TestParseArgs_TheVersionFlagTakesPriorityOverAMissingPositionalArgument(t *testing.T) {
	got, err := ParseArgs([]string{"--repo", "octocat/hello-world", "--version"})
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
