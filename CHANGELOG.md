# Changelog

<!--
When cutting a new release, update THREE places in this file:

1. Rename [Unreleased] to [X.Y.Z] with today's date (above).
2. Update the reference links at the very bottom of this file:
    - Change [Unreleased] to compare the new tag against HEAD.
    - Add [X.Y.Z] comparing the new tag against the previous tag (or, for the first release, linking directly to the tag).
3. After the PR is merged, create a GitHub Release (this creates the remote tag). Pull main first so HEAD is the merge commit, then use `--target main` or pass the full 40-character SHA — the GitHub API rejects abbreviated SHAs:

    ```console
    git checkout main && git pull origin main
    gh release create vX.Y.Z --title "vX.Y.Z" \
      --notes-file path/to/gh-release-draft.md \
      --target main
    ```
-->

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.5.0] - 2026-07-22

### Added

- Issue/PR labels: a label added to or removed from the issue/PR is now rendered as a new `LabelEvent` Tier 1 entry, interleaved chronologically with the rest of the timeline the same way a comment or review already is.
- Issue/PR history: closing/reopening (`ClosureEvent`), a title rename (`RenameEvent`), a milestone being added or removed (`MilestoneEvent`), and an assignee being added or removed (`AssignmentEvent`) are now rendered as new Tier 1 entries, interleaved chronologically with the rest of the timeline.
- Pull requests: the full changed-file diff is now rendered as a new `PullRequestDiff` Tier 1 entry, sourced from `GET /pulls/{number}/files`. A file's patch is suppressed once the pull request's total changed lines exceed 1000; the changed-file list itself (filename, status, additions, deletions) still renders in full.
- Pull requests: the commit list is now rendered as a new `PullRequestCommits` Tier 1 entry, sourced from `GET /pulls/{number}/commits`, carrying each commit's distinct author and committer identity plus its full message.
- Issues: parent/child (sub-issue) relationships are now rendered as new `ParentIssue`/`SubIssues` Tier 1 entries, sourced from the issue resource's `parent_issue_url` and a new `GET /issues/{number}/sub_issues` endpoint; each renders only when it actually has content. A `SubIssues` bullet renders as `` `{title}` [#{number}](url) ({state}) ``.
- Pull requests: check-run status is now rendered as a new `PullRequestChecks` Tier 1 entry, sourced from `GET /repos/{owner}/{repo}/commits/{sha}/check-runs` against the pull request's head commit, labeled with an explicit `captured_at` snapshot timestamp so a reader does not mistake it for the pull request's current, possibly-since-changed state.
- A bare issue/PR reference (`#123`, `owner/repo#123`) in a comment's or body's text is now resolved and rewritten to a backtick-wrapped, title-first linked form (`` `{title}` [{original text}](url) ``), skipping a reference already linked, inside an HTML comment, a fenced code block, or an inline code span. An unresolvable target (deleted, private, a transient failure) is left exactly as originally written.

### Changed

- **Breaking**: the CLI now requires an explicit `export` subcommand (`gh exhibit export <number>[,<number>...]`); a bare `gh exhibit <number>` (no subcommand) is rejected instead of being treated as an implicit export. Root-level flags are now limited to `--version` and the automatic `--help`; `--repo`/`-o`/`--output` usage moved under `gh exhibit export --help`.

## [0.4.0] - 2026-07-21

### Changed

- **Breaking**: gh-exhibit's own provenance (tool, version, commit) no longer renders as a `<!-- {"tool":...} -->` line in `index.md`; it now lives in a new `evidence/provenance.json` file per issue/PR, written by a dedicated `ProvenanceWriter` rather than mixed into the rendered document. Anything relying on the old rendered line must read `evidence/provenance.json` instead, which carries the identical `{"tool":...,"version":...,"commit":...}` shape. GitHub's own per-entry `<!-- {"meta":...} -->` line is unaffected and stays in `index.md`.

### Fixed

- GitHub client: an `X-RateLimit-Reset` value that is negative or overflows `time.Duration`/`time.Unix`'s arithmetic is now rejected and falls back to fixed exponential backoff, instead of wrapping to a bogus wait duration that skipped the backoff entirely — matching the bound already enforced on `Retry-After`.
- CLI: a missing-value flag error now names the flag exactly as typed (e.g. `--repo`), instead of collapsing a long-form flag's error message to a single dash (`-repo`).

### Security

- Valueobjects: `IssueRef.repo` and `AssetFilename` now reject any all-dots segment (e.g. `...`, optionally trailing-spaced), not just the exact literals `.` and `..`, closing a narrower-than-intended path-safety guarantee. Defense-in-depth: no working escape was demonstrated through either type's actual production callers.

## [0.3.1] - 2026-07-21

### Changed

- Internal: `internal/registry`'s `Config` gains optional `AuthToken`/`Transport` fields, unused by every production caller, so an integration test can wire `NewExportService` against a fake server the same way `github.NewEvidenceFetcher`/`NewAttachmentFetcher`'s own tests already do one layer down. No behavior change for any existing caller.

### Fixed

- GitHub client: attachment fetches (`github.com/user-attachments/...`) now follow a redirect to a different origin (e.g. a signed S3 URL) again, restoring the `v0.1.0`-era download behavior that the redirect-origin guard added in `v0.3.0` had regressed for essentially every real issue/PR with a pasted image. The guard stays in place for `evidenceFetcher`'s REST API requests, where its original assumption still holds; `net/http` already strips the `Authorization`/`Cookie` headers on a cross-host redirect, so removing the guard here does not reintroduce a credential leak.

## [0.3.0] - 2026-07-21

### Changed

- **Breaking**: each issue/PR's output now nests under a single `{number}/` page-bundle directory instead of scattering a rendered `{number}.md` file and flat, number-prefixed raw evidence files alongside a same-named `{number}/assets/` directory. The new layout:

  ```text
  {output}/{repo}/{number}/
  ├── index.md
  ├── assets/{filename}
  └── evidence/
      ├── issue.json
      ├── timeline.json
      ├── pull.json
      ├── review-comments.json
      └── fetch-errors.log
  ```

  A rendered document's link to its own attachments changes from `{number}/assets/{filename}` to `assets/{filename}`.

### Fixed

- CLI: a comma-separated list containing more than one negative number (e.g. `-1,-2`) is now recognized as a negative-number list and reaches the intended domain-validation error, instead of falling through to a generic "flag provided but not defined" error.

### Security

- GitHub client: a redirect whose target origin differs from the request's own origin is now rejected before the request is sent, closing a gap that could otherwise have let a redirected first page poison the already-shipped pagination-origin check's trusted reference.

## [0.2.0] - 2026-07-21

### Added

- Rendered Markdown now carries a document-level provenance line (`<!-- {"tool":...,"version":...,"commit":...} -->`) right after the H1 title, identifying the tool, version, and build commit that produced it — a self-reported identifier, not a tamper-resistant guarantee.

### Changed

- **Breaking**: the on-disk output layout dropped its fixed `issues/` segment; exported evidence and rendered Markdown now write directly under `{output}/{repo}/{number}...` instead of `{output}/issues/{repo}/{number}...`. `-o`/`--output` now owns all of "how to group this run's output on disk"; a caller who wants the old `issues/`-grouped shape gets it back by passing `-o .../issues` themselves.
- **Breaking**: each entry's `meta:{...}` line is now nested under a `"meta"` key and wrapped in an HTML comment (`<!-- {"meta":{...}} -->`), hiding it from a rendered Markdown preview while keeping it greppable as raw text. The `url` field stays undecorated.

### Fixed

- CLI: a value flag immediately adjacent to another flag token is now rejected instead of being silently swallowed as that other flag's value.
- GitHub client: a `Retry-After` header value that is negative or overflows is now rejected instead of driving an invalid wait.
- Persistence: `writeFile`'s rewrite of an existing file is now atomic (temp file + rename) instead of truncate-then-write, avoiding a corrupted file if the process is interrupted mid-write.
- Review comments: range-anchored inline review comments now resolve `start_line`/`original_start_line`, instead of only ever rendering the single-line end of the range.
- Services: an explicit `"pull_request": null` on the issue resource is now treated as "not a pull request," instead of being misread as present.

### Security

- Persistence: attachment filenames are now guaranteed path-safe via a dedicated `AssetFilename` value object, rejecting path-traversal- adjacent input at the boundary instead of trusting a derived filename.
- GitHub client: a paginated response's next-page URL is now rejected when its origin (scheme + host) differs from the current one, instead of being followed unconditionally.

## [0.1.2] - 2026-07-20

### Changed

- Internal: `internal/domain/services`'s attachment-download logic was refactored to remove Anemic Domain Model smells — decisions that belong on `Resolution`/`Attachment` (what Markdown text replaces an attachment reference, filename derivation) moved off `ExportService` and onto those types, and URL handling across `Attachment`/ `Resolution`/`Attribution` was unified behind a new `valueobjects.Url` type. No behavior change and no on-disk output layout change.

## [0.1.1] - 2026-07-19

### Fixed

- Release workflow: the build-provenance attestation step now finds its subject binaries, instead of failing every release with "Could not find subject at path ...". Release automation only; no effect on the `gh-exhibit` binary itself.

## [0.1.0] - 2026-07-19

### Added

- `gh exhibit <number>[,<number>...]` — export a GitHub issue or pull request's full discussion (body, issue comments, pull request reviews, inline review comments) as offline-verifiable Markdown, alongside the raw JSON evidence it was rendered from.
- `--repo <owner>/<repo>` — target an explicit repository; defaults to the current repository's context when omitted.
- `-o`, `--output <dir>` — output directory the evidence is written under.
- `--version` — print the running binary's version, commit, and build date.
- Attachments referenced in a body/comment/review are downloaded locally (not hotlinked) and rewritten to a relative local path; a per-URL fetch failure is skipped with a placeholder and recorded in a `fetch-errors.log`, without failing the rest of the export.
- Rate-limit-aware fetching (`Retry-After` / `X-RateLimit-Reset` honored, exponential backoff otherwise) and pagination for long timelines/review comment lists.
- A comma-separated list of issue/PR numbers exports each independently; a failure on one does not stop the rest of the batch.
- Distributed as a `gh` CLI extension (`gh extension install connect0459/gh-exhibit`), with per-platform precompiled release binaries.

---

[Unreleased]: <https://github.com/connect0459/gh-exhibit/compare/v0.5.0...HEAD>
[0.5.0]: <https://github.com/connect0459/gh-exhibit/compare/v0.4.0...v0.5.0>
[0.4.0]: <https://github.com/connect0459/gh-exhibit/compare/v0.3.1...v0.4.0>
[0.3.1]: <https://github.com/connect0459/gh-exhibit/compare/v0.3.0...v0.3.1>
[0.3.0]: <https://github.com/connect0459/gh-exhibit/compare/v0.2.0...v0.3.0>
[0.2.0]: <https://github.com/connect0459/gh-exhibit/compare/v0.1.2...v0.2.0>
[0.1.2]: <https://github.com/connect0459/gh-exhibit/compare/v0.1.1...v0.1.2>
[0.1.1]: <https://github.com/connect0459/gh-exhibit/compare/v0.1.0...v0.1.1>
[0.1.0]: <https://github.com/connect0459/gh-exhibit/releases/tag/v0.1.0>
