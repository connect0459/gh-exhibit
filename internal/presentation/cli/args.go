// Package cli implements gh-exhibit's presentation layer: parsing process
// arguments, resolving the target repository, and driving ExportService
// across the requested issue/PR numbers.
package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

// Args is the parsed, validated shape of gh-exhibit's process arguments.
type Args struct {
	// Numbers is the ordered, de-duplicated-by-input-order list of issue/PR
	// numbers to export, parsed from a single positional argument that is
	// either one number or a comma-separated list of them.
	Numbers []int

	// Repo is the explicit --repo owner/repo override; empty when omitted,
	// in which case the caller resolves it from the current repository
	// context instead (see ResolveRepo).
	Repo string

	// OutputDir is the -o/--output base directory the exported evidence is
	// written under; defaults to "." (the current directory) when omitted.
	OutputDir string

	// Version is true when --version was given; the caller should print
	// the running binary's version and exit without requiring a
	// positional issue/PR number.
	Version bool
}

// ParseArgs parses and validates args (typically os.Args[1:]) into an Args
// value. It fails on a missing or malformed issue/PR number, or on any
// number of positional arguments other than exactly one. Flags may appear
// before, after, or interleaved around the positional argument.
func ParseArgs(args []string) (Args, error) {
	fs := flag.NewFlagSet("gh-exhibit", flag.ContinueOnError)
	repo := fs.String("repo", "", "target repository as owner/repo (defaults to the current repository)")
	output := fs.String("output", ".", "output directory the evidence is written under")
	fs.StringVar(output, "o", ".", "shorthand for --output")
	version := fs.Bool("version", false, "print the version and exit")

	flagArgs, positional := splitFlagsAndPositional(args)

	if err := fs.Parse(flagArgs); err != nil {
		return Args{}, fmt.Errorf("parse flags: %w", err)
	}

	if *version {
		return Args{Version: true}, nil
	}

	if len(positional) != 1 {
		return Args{}, fmt.Errorf("expected exactly one issue/PR number argument (a single number or a comma-separated list), got %d", len(positional))
	}

	numbers, err := parseNumbers(positional[0])
	if err != nil {
		return Args{}, err
	}

	return Args{Numbers: numbers, Repo: *repo, OutputDir: *output}, nil
}

// valueFlags are gh-exhibit's flags that consume a following token as their
// value when not given in the attached "--flag=value" form.
var valueFlags = map[string]bool{"repo": true, "output": true, "o": true}

// splitFlagsAndPositional separates args into the tokens flag.FlagSet.Parse
// should see and the tokens that are gh-exhibit's own positional argument.
// This exists because flag.Parse stops scanning for flags at the first
// non-flag token, so without this split "gh-exhibit 123 --repo x" would
// misread "--repo" and "x" as extra positional arguments instead of a flag.
// A token shaped like a negative number or comma-separated list of them
// (e.g. "-1") is treated as positional rather than an unrecognized flag,
// since gh-exhibit's own numbers are the only thing that would ever look
// like that on the command line.
func splitFlagsAndPositional(args []string) (flagArgs, positional []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]

		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		if !strings.HasPrefix(a, "-") || a == "-" || looksLikeANegativeNumberList(a) {
			positional = append(positional, a)
			continue
		}

		name, _, hasInlineValue := strings.Cut(strings.TrimLeft(a, "-"), "=")
		if hasInlineValue || !valueFlags[name] {
			// Either the value is already attached ("--repo=x"), or this is
			// an unrecognized flag (including -h/--help) whose arity we
			// don't know — forwarded as-is so flag.Parse's own error/usage
			// handling applies to it.
			flagArgs = append(flagArgs, a)
			continue
		}

		flagArgs = append(flagArgs, a)
		if i+1 < len(args) {
			i++
			flagArgs = append(flagArgs, args[i])
		}
	}

	return flagArgs, positional
}

// looksLikeANegativeNumberList reports whether s is shaped like gh-exhibit's
// own number-or-comma-list positional argument gone negative (e.g. "-1",
// "-1,2"), as opposed to a flag.
func looksLikeANegativeNumberList(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	for _, r := range s[1:] {
		if r != ',' && r != ' ' && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

// parseNumbers splits raw on "," and parses each trimmed part as a positive
// issue/PR number, deduplicating repeats in first-seen order.
func parseNumbers(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	numbers := make([]int, 0, len(parts))
	seen := make(map[int]bool, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, fmt.Errorf("empty issue/PR number in list %q", raw)
		}

		n, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil, fmt.Errorf("%q is not a valid issue/PR number: %w", trimmed, err)
		}
		if n <= 0 {
			return nil, fmt.Errorf("issue/PR number %d must be positive", n)
		}
		if seen[n] {
			continue
		}
		seen[n] = true

		numbers = append(numbers, n)
	}

	return numbers, nil
}
