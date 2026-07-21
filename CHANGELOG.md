# Changelog

<!--
When cutting a new release, update THREE places in this file:

1. Rename [Unreleased] to [X.Y.Z] with today's date (above).
2. Update the reference links at the very bottom of this file:
    - Change [Unreleased] to compare the new tag against HEAD.
    - Add [X.Y.Z] comparing the new tag against the previous tag (or, for
      the first release, linking directly to the tag).
3. After the PR is merged, create a GitHub Release (this creates the remote
   tag). Pull main first so HEAD is the merge commit, then use `--target main`
   or pass the full 40-character SHA — the GitHub API rejects abbreviated SHAs:

    ```console
    git checkout main && git pull origin main
    gh release create vX.Y.Z --title "vX.Y.Z" \
      --notes-file path/to/gh-release-draft.md \
      --target main
    ```
-->

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-07-21

### Added

- Rendered Markdown now carries a document-level provenance line
  (`<!-- {"tool":...,"version":...,"commit":...} -->`) right after the H1
  title, identifying the tool, version, and build commit that produced
  it — a self-reported identifier, not a tamper-resistant guarantee.

### Changed

- **Breaking**: the on-disk output layout dropped its fixed `issues/`
  segment; exported evidence and rendered Markdown now write directly
  under `{output}/{repo}/{number}...` instead of
  `{output}/issues/{repo}/{number}...`. `-o`/`--output` now owns all of
  "how to group this run's output on disk"; a caller who wants the old
  `issues/`-grouped shape gets it back by passing `-o .../issues`
  themselves.
- **Breaking**: each entry's `meta:{...}` line is now nested under a
  `"meta"` key and wrapped in an HTML comment
  (`<!-- {"meta":{...}} -->`), hiding it from a rendered Markdown preview
  while keeping it greppable as raw text. The `url` field stays
  undecorated.

### Fixed

- CLI: a value flag immediately adjacent to another flag token is now
  rejected instead of being silently swallowed as that other flag's
  value.
- GitHub client: a `Retry-After` header value that is negative or
  overflows is now rejected instead of driving an invalid wait.
- Persistence: `writeFile`'s rewrite of an existing file is now atomic
  (temp file + rename) instead of truncate-then-write, avoiding a
  corrupted file if the process is interrupted mid-write.
- Review comments: range-anchored inline review comments now resolve
  `start_line`/`original_start_line`, instead of only ever rendering the
  single-line end of the range.
- Services: an explicit `"pull_request": null` on the issue resource is
  now treated as "not a pull request," instead of being misread as
  present.

### Security

- Persistence: attachment filenames are now guaranteed path-safe via a
  dedicated `AssetFilename` value object, rejecting path-traversal-
  adjacent input at the boundary instead of trusting a derived filename.
- GitHub client: a paginated response's next-page URL is now rejected
  when its origin (scheme + host) differs from the current one, instead
  of being followed unconditionally.

## [0.1.2] - 2026-07-20

### Changed

- Internal: `internal/domain/services`'s attachment-download logic was
  refactored to remove Anemic Domain Model smells — decisions that
  belong on `Resolution`/`Attachment` (what Markdown text replaces an
  attachment reference, filename derivation) moved off `ExportService`
  and onto those types, and URL handling across `Attachment`/
  `Resolution`/`Attribution` was unified behind a new
  `valueobjects.Url` type. No behavior change and no on-disk output
  layout change.

## [0.1.1] - 2026-07-19

### Fixed

- Release workflow: the build-provenance attestation step now finds its
  subject binaries, instead of failing every release with "Could not
  find subject at path ...". Release automation only; no effect on the
  `gh-exhibit` binary itself.

## [0.1.0] - 2026-07-19

### Added

- `gh exhibit <number>[,<number>...]` — export a GitHub issue or pull
  request's full discussion (body, issue comments, pull request reviews,
  inline review comments) as offline-verifiable Markdown, alongside the raw
  JSON evidence it was rendered from.
- `--repo <owner>/<repo>` — target an explicit repository; defaults to the
  current repository's context when omitted.
- `-o`, `--output <dir>` — output directory the evidence is written under.
- `--version` — print the running binary's version, commit, and build date.
- Attachments referenced in a body/comment/review are downloaded locally
  (not hotlinked) and rewritten to a relative local path; a per-URL fetch
  failure is skipped with a placeholder and recorded in a
  `fetch-errors.log`, without failing the rest of the export.
- Rate-limit-aware fetching (`Retry-After` / `X-RateLimit-Reset` honored,
  exponential backoff otherwise) and pagination for long timelines/review
  comment lists.
- A comma-separated list of issue/PR numbers exports each independently; a
  failure on one does not stop the rest of the batch.
- Distributed as a `gh` CLI extension (`gh extension install
  connect0459/gh-exhibit`), with per-platform precompiled release binaries.

---

[Unreleased]: <https://github.com/connect0459/gh-exhibit/compare/v0.2.0...HEAD>
[0.2.0]: <https://github.com/connect0459/gh-exhibit/compare/v0.1.2...v0.2.0>
[0.1.2]: <https://github.com/connect0459/gh-exhibit/compare/v0.1.1...v0.1.2>
[0.1.1]: <https://github.com/connect0459/gh-exhibit/compare/v0.1.0...v0.1.1>
[0.1.0]: <https://github.com/connect0459/gh-exhibit/releases/tag/v0.1.0>
