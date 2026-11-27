# gh-exhibit

[![CI](https://github.com/connect0459/gh-exhibit/actions/workflows/ci.yml/badge.svg)](https://github.com/connect0459/gh-exhibit/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/connect0459/gh-exhibit/blob/main/LICENSE)
[![GitHub CLI](https://img.shields.io/badge/gh-cli-blue.svg)](https://cli.github.com)

A `gh` CLI extension that exports the full discussion of a GitHub issue or pull request (body, comments, reviews, inline review comments, attachments) as offline-verifiable Markdown, alongside the raw JSON evidence it was rendered from.

## Motivation

A link to a GitHub issue or pull request is not evidence: it can 404, be edited, or become unreachable behind private-repository access. `gh-exhibit` exports the discussion as a self-contained file that a technical assessment, audit trail, or bug report can cite without depending on GitHub staying reachable.

- **Full fidelity, not a summary.** The raw REST API responses behind each export are stored verbatim next to the rendered Markdown, so no rendering choice made today loses information.
- **No hotlinking.** Attachments are downloaded into the export directory, so nothing in the output depends on GitHub's CDN.
- **One self-contained directory per issue/PR.** See [Output](#output) below.

## Installation

To install:

```sh
gh extension install connect0459/gh-exhibit
```

To upgrade:

```sh
gh extension upgrade connect0459/gh-exhibit
```

## Usage

Show the version:

```sh
gh exhibit --version
```

Show help:

```sh
gh exhibit --help

# Print usage for a subcommand
gh exhibit export --help
gh exhibit export-search --help
```

Export GitHub issues or pull requests by number:

```sh
gh exhibit export <number>[,<number>...] [--repo <owner>/<repo>] [-o|--output <dir>] [--with-stdout] [--dry-run]
```

Or export by filter criteria with the separate `export-search` subcommand:

```sh
gh exhibit export-search [--author <login>[,...]] [--assignee <login>[,...]] [--review-requested <login>[,...]] [--reviewed-by <login>[,...]] [--kind issue|pr[,...]] [--after <YYYY-MM-DD>] [--before <YYYY-MM-DD>] [--limit <n>] [--sort created|updated|comments] [--order asc|desc] [--dry-run] [--repo <owner>/<repo>] [-o|--output <dir>] [--with-stdout]
```

Render a saved export as a browsable page, offline:

```sh
gh exhibit view <number>[,<number>...] [--repo <owner>/<repo>] [-o|--output <dir>]
```

### Flags and Subcommands

- `export`: exports the given issue/PR(s) by number.
  - `<number>[,<number>...]`: a single issue/PR number, or a comma-separated list of them.
  - `--repo`: target repository as `owner/repo`; defaults to the current repository when omitted.
  - `-o`, `--output`: directory the evidence is written under; defaults to `.`.
  - `--with-stdout`: also print each exported ref's rendered document to standard output, in addition to the usual on-disk writes. When multiple refs are exported, each document is preceded by a `=== owner/repo#N ===` header line. The printed bytes are exactly what is written to `index.md`.
  - `--dry-run`: print each ref's destination path (e.g. `{output}/{repo}/{number}/index.md`) to stdout, without calling the GitHub API or writing anything to disk. `export` has no resolution step to preview because its numbers are already known, so this is a purely local, offline check, unlike `export-search --dry-run` below.
- `export-search`: resolves filter criteria into a list of issue/PR numbers via GitHub's search API, then exports every match. Takes no positional argument.
  - `--author`, `--assignee`: comma-separated GitHub login(s) to filter by.
  - `--review-requested`, `--reviewed-by`: comma-separated GitHub login(s) whose review is requested, or who have reviewed, respectively. Both map to GitHub's `review-requested:`/`reviewed-by:` search qualifiers and match only pull requests. Combining either with `--kind issue` yields no matches, because GitHub's search API returns zero results for that combination rather than an error.
  - `--kind`: comma-separated `issue`,`pr` to restrict the kind of ref; omitted (or both) means both.
  - `--after`, `--before`: an inclusive `YYYY-MM-DD` bound on the creation date.
  - `--limit`: maximum number of matches to resolve, `1`-`100` (default `100`). This is gh-exhibit's own conservative cap, well below GitHub search's 1000-result ceiling, so a call with no other filters can't become a de facto whole-repository export.
  - `--sort`, `--order`: the field (`created`/`updated`/`comments`) and direction (`asc`/`desc`) that order the matches; default `created`/`desc`.
  - `--dry-run`: runs the real search, then prints the match count and number list to stdout without exporting anything. It previews the search *result*, unlike `export --dry-run` above, which has no search to preview.
  - `--repo`, `-o`/`--output`, `--with-stdout`: same as `export`'s flags above.
- `view`: rebuilds a browsable `index.html` offline from the evidence a previous `export` saved, without calling the GitHub API. The page shows the title, state, and discussion as GitHub-style cards: the body, comments, reviews with their state, and inline review comments with their file, lines, and diff hunk. For a pull request, it also shows changed files, commits, and timeline events (assignments, closures, labels, milestones, renames). Authors are shown by initial rather than a downloaded avatar, and attachments link to their downloaded copies under `assets/`. Raw HTML in a body is sanitized: scripts, event handlers, forms, and any image that is not a file under `assets/` are dropped, so opening the page never contacts another host. Like `export`, `view` takes the repository from `--repo`, or from the current directory's git remote when `--repo` is omitted, so outside a git repository `--repo` is required. An export made before attachment links were recorded shows its attachments as unavailable; `view` says so, and re-running `export` records them.
  - `<number>[,<number>...]`: a single issue/PR number, or a comma-separated list of them.
  - `--repo`, `-o`, `--output`: same as `export`'s flags above; `--output` is where the earlier export wrote its evidence.
- `-h`, `--help`: print usage and exit. Run it at the root for the root-level flags, or as `gh exhibit export -h`/`gh exhibit export-search -h` for a subcommand's flags.
- `--version`: print the version, commit, and build date, then exit.

> [!NOTE]
> Every flag above works with either one or two leading dashes (`-repo` and `--repo` are the same flag, not separate short and long forms). The usage text always prints the single-dash spelling, which this project treats as each flag's base form; the double-dash spelling above is for readability. `-o` and `-h` are the only true single-letter shorthands for a separate long name (`--output`, `--help`).

### Examples

```sh
# Export a single PR from the current repository
gh exhibit export 10

# Export multiple issues/PRs from an explicit repository
gh exhibit export 10,11,12 --repo connect0459/gh-exhibit -o ./exhibits

# Preview where a number list would be written, without exporting anything
gh exhibit export 10,11,12 --repo connect0459/gh-exhibit -o ./exhibits --dry-run

# Preview every issue/PR a given author opened this year, without exporting anything
gh exhibit export-search --author connect0459 --after 2026-01-01 --limit 10 --dry-run

# Export every open pull request assigned to a given user
gh exhibit export-search --repo connect0459/gh-exhibit -o ./exhibits --assignee connect0459 --kind pr --limit 10
```

## Output

The multi-issue example above produces one self-contained directory per number, named `{repo}/{number}` (the owner is deliberately left out of the path):

```text
./exhibits/gh-exhibit/
├── 10/
│   ├── index.md            # rendered Markdown
│   ├── index.html          # browsable view, written by `view` (not by `export`)
│   ├── assets/             # downloaded attachments
│   └── evidence/           # raw JSON and export metadata
│       ├── issue.json      # verbatim GitHub REST response
│       ├── timeline.json   # verbatim GitHub REST response
│       └── provenance.json # which gh-exhibit tool/version/commit produced this export
├── 11/
└── 12/
```

See [Documentation](#documentation) for details.

## Documentation

- [Architecture](docs/ARCHITECTURE.md) — the onion architecture layers, the dependency rule between them, and why boundaries convert types instead of merely aligning dependency direction.
- [Specification](docs/specs/README.md) — current behavior: CLI shape, domain model, on-disk layout, Markdown dialect, attachment and retry policy, coverage targets.

## Contributing

See [CONTRIBUTING.md](https://github.com/connect0459/gh-exhibit/blob/main/CONTRIBUTING.md).

## License

[MIT](https://github.com/connect0459/gh-exhibit/blob/main/LICENSE)
