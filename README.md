# gh-exhibit

[![CI](https://github.com/connect0459/gh-exhibit/actions/workflows/ci.yml/badge.svg)](https://github.com/connect0459/gh-exhibit/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/connect0459/gh-exhibit/blob/main/LICENSE)
[![GitHub CLI](https://img.shields.io/badge/gh-cli-blue.svg)](https://cli.github.com)

A `gh` CLI extension that exports a GitHub issue or pull request's full
discussion (body, comments, reviews, inline review comments, attachments)
as offline-verifiable Markdown alongside the raw JSON evidence it was
rendered from.

## Motivation

A link back to a GitHub issue or pull request is not evidence — it can
404, get edited, or become unreachable behind private-repository access.
`gh-exhibit` exports the discussion as a self-contained file: the kind of
record a technical assessment, an audit trail, or a bug report can point
to and trust independent of GitHub staying reachable.

- **Full-fidelity, not a summary.** The raw REST API responses backing
  each export are stored verbatim alongside the rendered Markdown, so
  nothing is lost to a rendering choice made today.
- **No hotlinking.** Attachments are downloaded into the export directory
  itself; nothing in the output depends on GitHub's CDN staying up.
- **One self-contained directory per issue/PR** — see [Output](#output)
  below.

## Installation

```sh
gh extension install connect0459/gh-exhibit
```

To upgrade:

```sh
gh extension upgrade connect0459/gh-exhibit
```

## Usage

Get version:

```sh
gh exhibit --version
```

Get help:

```sh
gh exhibit --help

# Print usage for the export subcommand
gh exhibit export --help
```

Export GitHub issues or pull requests:

```sh
gh exhibit export <number>[,<number>...] [--repo <owner>/<repo>] [-o|--output <dir>]
```

## Flags and Subcommands

- `export`: exports the given issue/PR(s).
  - `<number>[,<number>...]`: a single issue/PR number, or a comma-separated
    list of them.
  - `--repo`: target repository as `owner/repo`; defaults to the current
    repository's context when omitted.
  - `-o`, `--output`: output directory the evidence is written under;
    defaults to `.`.
- `-h`, `--help`: print usage and exit. Run at the root for the
  root-level flags, or `gh exhibit export -h` for `export`'s own flags.
- `--version`: print the version, commit, and build date, then exit.

> [!NOTE]
> Every flag above works with either one or two leading dashes (`-repo` and `--repo` are the same flag, not separate short/long forms) — `-h`/`--help`'s
> own usage text always prints the single-dash spelling, which this project treats as each flag's base form;
> the double-dash spelling shown above is written for readability. `-o` and `-h` are the only flags that are true
> single-letter shorthands for a separate long name (`--output`, `--help`).

### Examples

```sh
# Export a single PR from the current repository
gh exhibit export 10

# Export multiple issues/PRs from an explicit repository
gh exhibit export 10,11,12 --repo connect0459/gh-exhibit -o ./exhibits
```

## Output

The multi-issue example above produces one self-contained directory per
number, named by `{repo}/{number}` (the owner is deliberately not part of
the path):

```text
./exhibits/gh-exhibit/
├── 10/
│   ├── index.md          rendered Markdown — the exhibit itself
│   ├── assets/            downloaded attachments
│   └── evidence/          raw JSON and export metadata
│       ├── issue.json          verbatim GitHub REST response
│       ├── timeline.json       verbatim GitHub REST response
│       └── provenance.json     which gh-exhibit tool/version/commit produced this export
├── 11/
└── 12/
```

See [Documentation](#documentation) for details.

## Documentation

- [Specification](docs/specs/README.md) — current behavior: CLI shape,
  domain model, on-disk layout, Markdown dialect, attachment and retry
  policy, coverage targets.

## Contributing

See [CONTRIBUTING.md](https://github.com/connect0459/gh-exhibit/blob/main/CONTRIBUTING.md).

## License

[MIT](https://github.com/connect0459/gh-exhibit/blob/main/LICENSE)
