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

## [0.1.0] - 2026-07-19

### Added

- `gh exhibit <number>[,<number>...]` — export a GitHub issue or pull
  request's full discussion (body, issue comments, pull request reviews,
  inline review comments) as offline-verifiable Markdown, alongside the raw
  JSON evidence it was rendered from.
- `--repo <owner>/<repo>` — target an explicit repository; defaults to the
  current repository's context when omitted.
- `-o`, `--output <dir>` — output directory the evidence is written under.
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

[Unreleased]: <https://github.com/connect0459/gh-exhibit/compare/v0.1.0...HEAD>
[0.1.0]: <https://github.com/connect0459/gh-exhibit/releases/tag/v0.1.0>
