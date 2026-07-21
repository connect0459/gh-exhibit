# gh-exhibit

[![CI](https://github.com/connect0459/gh-exhibit/actions/workflows/ci.yml/badge.svg)](https://github.com/connect0459/gh-exhibit/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/connect0459/gh-exhibit/blob/main/LICENSE)
[![GitHub CLI](https://img.shields.io/badge/gh-cli-blue.svg)](https://cli.github.com)

A `gh` CLI extension that exports a GitHub issue or pull request's full
discussion (body, comments, reviews, inline review comments, attachments)
as offline-verifiable Markdown alongside the raw JSON evidence it was
rendered from.

## Installation

```sh
gh extension install connect0459/gh-exhibit
```

## Usage

```sh
gh exhibit <number>[,<number>...] [--repo <owner>/<repo>] [-o|--output <dir>]
```

- `<number>[,<number>...]`: a single issue/PR number, or a comma-separated
  list of them.
- `--repo`: target repository as `owner/repo`; defaults to the current
  repository's context when omitted.
- `-o`, `--output`: output directory the evidence is written under;
  defaults to `.`.
- `--version`: print the version, commit, and build date, then exit.

### Examples

```sh
# Print the installed version
gh exhibit --version

# Export a single PR from the current repository
gh exhibit 10

# Export multiple issues/PRs from an explicit repository
gh exhibit 10,11,12 --repo connect0459/gh-exhibit -o ./evidence
```

## Documentation

- [Specification](docs/specs/README.md) — current behavior: CLI shape,
  domain model, on-disk layout, Markdown dialect, attachment and retry
  policy, coverage targets.
