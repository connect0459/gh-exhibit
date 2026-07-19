# TODO

## Design (open decisions)

All items below are formally recorded in
`docs/adrs/adr-002-language-and-domain-design.md`.

- [x] Decide the implementation language/stack — Go, with `go-gh` as the
      full-native acquisition layer (no `gh api` shellout). Driven by
      distribution ease (`gh extension create` scaffolding is Go-only) and
      consistency with the ADR-001 decision to distribute as a gh CLI
      extension.
- [x] Decide the Go type design for Tier 1 entries — all four entry types
      (issue/PR body, IssueComment, PullRequestReview, InlineReviewComment)
      are modeled as Value Objects (no identity-based tracking; re-fetch
      diffing is handled by git in the separate evidence repository, not by
      this tool). Rendering uses a sealed `Entry` interface with a
      polymorphic `Render()` method per concrete type, rather than a
      centralized type switch. Classification of timeline entries into
      concrete types still requires a two-pass unmarshal (peek discriminator,
      then dispatch), reinforced by the `exhaustive` lint check.
- [x] Define the on-disk layout for raw JSON evidence next to rendered
      Markdown — split by source endpoint rather than a consolidated
      wrapper: `issues/{repo}/{number}.json` (issue/pull resource),
      `issues/{repo}/{number}.timeline.json` (timeline, multi-page responses
      concatenated into one array), and `issues/{repo}/{number}.pull.json`
      (PRs only). Rendered Markdown stays `issues/{repo}/{number}.md` per the
      existing dialect.
- [x] Decide the attachment policy — local download is mandatory (not
      hotlinking), to uphold the README's offline-verifiability principle.
      Fetched via an authenticated request, extension derived from the
      response `Content-Type` header, stored under
      `issues/{repo}/{number}/assets/{filename}`. On fetch failure: skip and
      warn, keep processing, replace the Markdown reference with a
      placeholder noting the original URL and failure reason, and persist a
      failure summary file in the run's output directory (e.g.
      `issues/{repo}/{number}/fetch-errors.log`).
- [x] Discuss test coverage targets before implementation starts (required by
      AGENTS.md) — domain layer: C0 90% / C1 75-80% as a floor; boundary
      layer (HTTP, file I/O): qualitative branch coverage via mocks rather
      than a numeric target. C1 is not auto-gated in CI (Go's toolchain does
      not report it as a single metric); reviewed manually via
      `go tool cover -html`. No pre-agreed lower floor for backing off the
      domain-layer target — coverage breakdown is presented per package after
      implementation, decided case by case.

## Spike

- [x] Inspect real REST timeline responses (`reviewed`, `line-commented`,
      `commented`, `cross-referenced` events) to confirm where inline review
      comments carry `path`/`line`/`diff_hunk` — **finding: no
      `line-commented` event exists in the current timeline API; inline
      review comments come only from `GET /pulls/{number}/comments`,
      joined to their parent review via `pull_request_review_id`.**
      Verified against `cli/cli` PRs #13780 and #13084. ADR-001 and
      ADR-002 corrected accordingly (see their 2026-07-18 addenda).

## Implementation (after design closes)

- [ ] Acquisition: fetch issue/PR + timeline via REST, persist raw JSON
- [ ] Acquisition: fetch inline review comments via
      `GET /pulls/{number}/comments` for PRs, persist raw JSON
      (`{number}.review-comments.json`), join to parent review via
      `pull_request_review_id`
- [ ] Rendering: Tier 1 Markdown view (`meta:` lines, `------` separator,
      inline review comments with file/line context)
- [ ] Pagination handling for long timelines
