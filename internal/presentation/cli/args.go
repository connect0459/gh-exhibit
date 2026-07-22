// Package cli implements gh-exhibit's presentation layer: parsing process
// arguments, resolving the target repository, and driving ExportService
// across the requested issue/PR numbers.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

// Args is the parsed, validated shape of gh-exhibit's process arguments.
type Args struct {
	// Numbers is the ordered, de-duplicated-by-input-order list of issue/PR
	// numbers to export, parsed from a single positional argument that is
	// either one number or a comma-separated list of them. Populated only
	// in explicit-number mode; nil in filter mode (see Criteria).
	Numbers []int

	// Criteria is non-nil when any filter flag (--author, --assignee,
	// --kind, --created-after, --created-before, --limit, --sort, --order,
	// --dry-run) was given instead of an explicit issue/PR number list —
	// filter mode and explicit-number mode are mutually exclusive (see
	// parseExportArgs).
	Criteria *valueobjects.SearchCriteria

	// DryRun is true when --dry-run was given: the caller should report
	// Criteria's resolved match count and numbers without exporting
	// anything. Only meaningful when Criteria is non-nil.
	DryRun bool

	// Repo is the explicit --repo owner/repo override; empty when omitted,
	// in which case the caller resolves it from the current repository
	// context instead (see ResolveRepo).
	Repo string

	// OutputDir is the -o/--output base directory the exported evidence is
	// written under; defaults to "." (the current directory) when omitted.
	OutputDir string

	// WithStdout is true when --with-stdout was given; the caller should,
	// in addition to every file export already writes, also print each
	// exported ref's rendered document to standard output.
	WithStdout bool

	// Version is true when --version was given; the caller should print
	// the running binary's version and exit without requiring a
	// positional issue/PR number.
	Version bool
}

// exportSubcommand is the sole subcommand gh-exhibit currently recognizes.
const exportSubcommand = "export"

// ParseArgs parses and validates args (typically os.Args[1:]) into an Args
// value. The root level recognizes only --version (and the automatic
// -h/--help); everything else is dispatched to a subcommand, of which
// "export" is currently the only one defined. ParseArgs fails when no
// subcommand is given, when the given subcommand is not recognized, or when
// the "export" subcommand's own arguments are invalid (see
// parseExportArgs).
func ParseArgs(args []string) (Args, error) {
	rootFS := flag.NewFlagSet("gh-exhibit", flag.ContinueOnError)
	version := rootFS.Bool("version", false, "print the version and exit")

	if err := rootFS.Parse(args); err != nil {
		return Args{}, fmt.Errorf("parse flags: %w", err)
	}

	if *version {
		return Args{Version: true}, nil
	}

	remaining := rootFS.Args()
	if len(remaining) == 0 {
		return Args{}, fmt.Errorf("expected a subcommand (%q)", exportSubcommand)
	}

	subcommand, rest := remaining[0], remaining[1:]
	switch subcommand {
	case exportSubcommand:
		return parseExportArgs(rest)
	default:
		return Args{}, fmt.Errorf("unknown subcommand %q (expected %q)", subcommand, exportSubcommand)
	}
}

// filterFlagNames are parseExportArgs' filter-mode flags: giving any one of
// them, instead of the positional issue/PR number list, selects filter
// mode (see parseExportArgs).
var filterFlagNames = map[string]bool{
	"author": true, "assignee": true, "kind": true,
	"created-after": true, "created-before": true,
	"limit": true, "sort": true, "order": true, "dry-run": true,
}

// parseExportArgs parses and validates the "export" subcommand's own
// arguments (everything after the "export" token) into an Args value.
// Supplying the positional issue/PR number (or comma-separated list)
// selects explicit-number mode; supplying any filter flag (see
// filterFlagNames) instead selects filter mode. Supplying both is a parse
// error — the two modes are mutually exclusive. Flags may appear before,
// after, or interleaved around the positional argument.
func parseExportArgs(args []string) (Args, error) {
	fs := flag.NewFlagSet("gh-exhibit export", flag.ContinueOnError)
	repo := fs.String("repo", "", "target repository as owner/repo (defaults to the current repository)")
	output := fs.String("output", ".", "output directory the evidence is written under")
	fs.StringVar(output, "o", ".", "shorthand for --output")
	withStdout := fs.Bool("with-stdout", false, "also print each exported ref's rendered document to standard output")

	author := fs.String("author", "", "filter mode: comma-separated GitHub login(s) to match as author")
	assignee := fs.String("assignee", "", "filter mode: comma-separated GitHub login(s) to match as assignee")
	kind := fs.String("kind", "", "filter mode: comma-separated issue,pr to restrict the ref kind (default: both)")
	createdAfter := fs.String("created-after", "", "filter mode: only match refs created on or after this date (YYYY-MM-DD)")
	createdBefore := fs.String("created-before", "", "filter mode: only match refs created on or before this date (YYYY-MM-DD)")
	limit := fs.Int("limit", valueobjects.DefaultSearchLimit, fmt.Sprintf("filter mode: maximum number of matches to resolve (1-%d)", valueobjects.MaxSearchLimit))
	sortFlag := fs.String("sort", "created", "filter mode: sort matches by created, updated, or comments")
	order := fs.String("order", "desc", "filter mode: sort order, asc or desc")
	dryRun := fs.Bool("dry-run", false, "filter mode: report the resolved match count and numbers without exporting anything")

	flagArgs, positional, err := splitFlagsAndPositional(args)
	if err != nil {
		return Args{}, fmt.Errorf("parse flags: %w", err)
	}

	if err := fs.Parse(flagArgs); err != nil {
		return Args{}, fmt.Errorf("parse flags: %w", err)
	}

	filterFlagGiven := false
	fs.Visit(func(f *flag.Flag) {
		if filterFlagNames[f.Name] {
			filterFlagGiven = true
		}
	})

	if filterFlagGiven {
		if len(positional) > 0 {
			return Args{}, errors.New("cannot combine an explicit issue/PR number list with filter flags")
		}
		criteria, err := parseSearchCriteria(*author, *assignee, *kind, *createdAfter, *createdBefore, *limit, *sortFlag, *order)
		if err != nil {
			return Args{}, err
		}
		return Args{Criteria: &criteria, DryRun: *dryRun, Repo: *repo, OutputDir: *output, WithStdout: *withStdout}, nil
	}

	if len(positional) != 1 {
		return Args{}, fmt.Errorf("expected exactly one issue/PR number argument (a single number or a comma-separated list), or a filter flag (--author, --assignee, --kind, --created-after, --created-before, --limit, --sort, --order, --dry-run); got %d positional argument(s) and no filter flag", len(positional))
	}

	numbers, err := parseNumbers(positional[0])
	if err != nil {
		return Args{}, err
	}

	return Args{Numbers: numbers, Repo: *repo, OutputDir: *output, WithStdout: *withStdout}, nil
}

// parseSearchCriteria builds a valueobjects.SearchCriteria from
// parseExportArgs' own raw filter-flag values.
func parseSearchCriteria(rawAuthor, rawAssignee, rawKind, rawCreatedAfter, rawCreatedBefore string, limit int, rawSort, rawOrder string) (valueobjects.SearchCriteria, error) {
	authors, err := parseLogins(rawAuthor)
	if err != nil {
		return valueobjects.SearchCriteria{}, fmt.Errorf("--author: %w", err)
	}
	assignees, err := parseLogins(rawAssignee)
	if err != nil {
		return valueobjects.SearchCriteria{}, fmt.Errorf("--assignee: %w", err)
	}
	kinds, err := parseKinds(rawKind)
	if err != nil {
		return valueobjects.SearchCriteria{}, fmt.Errorf("--kind: %w", err)
	}
	createdAfter, err := parseSearchDate(rawCreatedAfter)
	if err != nil {
		return valueobjects.SearchCriteria{}, fmt.Errorf("--created-after: %w", err)
	}
	createdBefore, err := parseSearchDate(rawCreatedBefore)
	if err != nil {
		return valueobjects.SearchCriteria{}, fmt.Errorf("--created-before: %w", err)
	}
	sortField, err := valueobjects.ParseSearchSortField(rawSort)
	if err != nil {
		return valueobjects.SearchCriteria{}, fmt.Errorf("--sort: %w", err)
	}
	order, err := valueobjects.ParseSearchSortOrder(rawOrder)
	if err != nil {
		return valueobjects.SearchCriteria{}, fmt.Errorf("--order: %w", err)
	}

	criteria, err := valueobjects.NewSearchCriteria(authors, assignees, kinds, createdAfter, createdBefore, limit, sortField, order)
	if err != nil {
		return valueobjects.SearchCriteria{}, err
	}
	return criteria, nil
}

// parseLogins splits raw on "," and trims each part, deduplicating repeats
// in first-seen order; "" (the flag's unset default) returns nil, meaning
// unfiltered by that dimension.
func parseLogins(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	logins := make([]string, 0, len(parts))
	seen := make(map[string]bool, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, fmt.Errorf("empty login in list %q", raw)
		}
		if seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		logins = append(logins, trimmed)
	}
	return logins, nil
}

// parseKinds splits raw on "," and parses each trimmed part as an
// valueobjects.IssueKind, deduplicating repeats in first-seen order; "" (the
// flag's unset default) returns nil, meaning both kinds.
func parseKinds(raw string) ([]valueobjects.IssueKind, error) {
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	kinds := make([]valueobjects.IssueKind, 0, len(parts))
	seen := make(map[valueobjects.IssueKind]bool, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			return nil, fmt.Errorf("empty kind in list %q", raw)
		}
		kind, err := valueobjects.ParseIssueKind(trimmed)
		if err != nil {
			return nil, err
		}
		if seen[kind] {
			continue
		}
		seen[kind] = true
		kinds = append(kinds, kind)
	}
	return kinds, nil
}

// parseSearchDate parses raw as a YYYY-MM-DD date; "" (the flag's unset
// default) returns nil, meaning that bound is unset.
func parseSearchDate(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}

	parsed, err := time.Parse(valueobjects.SearchDateLayout, raw)
	if err != nil {
		return nil, fmt.Errorf("%q is not a valid date (want YYYY-MM-DD): %w", raw, err)
	}
	return &parsed, nil
}

// valueFlags are gh-exhibit's flags that consume a following token as their
// value when not given in the attached "--flag=value" form.
var valueFlags = map[string]bool{
	"repo": true, "output": true, "o": true,
	"author": true, "assignee": true, "kind": true,
	"created-after": true, "created-before": true,
	"limit": true, "sort": true, "order": true,
}

// splitFlagsAndPositional separates args into the tokens flag.FlagSet.Parse
// should see and the tokens that are the "export" subcommand's own
// positional argument. This exists because flag.Parse stops scanning for
// flags at the first non-flag token, so without this split
// "gh-exhibit export 123 --repo x" would misread "--repo" and "x" as extra
// positional arguments instead of a flag.
// A token shaped like a negative number or comma-separated list of them
// (e.g. "-1") is treated as positional rather than an unrecognized flag,
// since gh-exhibit's own numbers are the only thing that would ever look
// like that on the command line.
//
// It returns an error, rather than deferring to flag.FlagSet.Parse, when a
// value-taking flag (-o/--output/--repo) is not immediately followed by a
// usable value: flag.FlagSet.Parse would otherwise unconditionally consume
// whatever token comes next — including one shaped like another flag — and
// silently misassign it as the value instead of reporting a missing
// argument.
func splitFlagsAndPositional(args []string) (flagArgs, positional []string, err error) {
	for i := 0; i < len(args); i++ {
		a := args[i]

		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}

		if !isFlagShaped(a) {
			positional = append(positional, a)
			continue
		}

		name, dashes, hasInlineValue := flagNameAndInlineValue(a)
		if hasInlineValue || !valueFlags[name] {
			// Either the value is already attached ("--repo=x"), or this is
			// an unrecognized flag (including -h/--help) whose arity we
			// don't know — forwarded as-is so flag.Parse's own error/usage
			// handling applies to it.
			flagArgs = append(flagArgs, a)
			continue
		}

		if i+1 >= len(args) || isFlagShaped(args[i+1]) {
			return nil, nil, fmt.Errorf("flag needs an argument: %s%s", dashes, name)
		}

		i++
		flagArgs = append(flagArgs, a, args[i])
	}

	return flagArgs, positional, nil
}

// isFlagShaped reports whether s should be scanned as a flag token rather
// than gh-exhibit's positional argument.
func isFlagShaped(s string) bool {
	return strings.HasPrefix(s, "-") && s != "-" && !looksLikeANegativeNumberList(s)
}

// flagNameAndInlineValue extracts a flag token's name, the dash prefix it was
// given with ("-" or "--"), and whether it carries an attached "=value". It
// strips at most two leading dashes, so a token with three or more (e.g.
// "---repo") does not collapse onto a recognized flag name and is left for
// flag.Parse's own rejection.
func flagNameAndInlineValue(a string) (name, dashes string, hasInlineValue bool) {
	trimmed := strings.TrimPrefix(a, "--")
	dashes = "--"
	if trimmed == a {
		trimmed = strings.TrimPrefix(a, "-")
		dashes = "-"
	}
	name, _, hasInlineValue = strings.Cut(trimmed, "=")
	return name, dashes, hasInlineValue
}

// looksLikeANegativeNumberList reports whether s is shaped like gh-exhibit's
// own number-or-comma-list positional argument gone negative (e.g. "-1",
// "-1,2", "-1,-2"), as opposed to a flag. This is a loose heuristic, not a
// full grammar check: any digit/comma/space/dash mix containing at least
// one digit passes (so does, e.g., "-1-2"), leaving the stricter shape
// check to parseNumbers — the point here is only to keep such a token out
// of flag.FlagSet's own parsing, not to fully validate it. A "-"-only
// token (e.g. "--", the flag terminator) is deliberately excluded by
// requiring at least one digit somewhere in s, since a dash alone never
// carries a number.
func looksLikeANegativeNumberList(s string) bool {
	if len(s) < 2 || s[0] != '-' {
		return false
	}
	sawDigit := false
	for _, r := range s[1:] {
		switch {
		case r == ',' || r == ' ' || r == '-':
			continue
		case r >= '0' && r <= '9':
			sawDigit = true
		default:
			return false
		}
	}
	return sawDigit
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
