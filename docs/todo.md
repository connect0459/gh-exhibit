# TODO

## Design (open decisions)

All items below were originally recorded in this project's now-removed ADR
documents (`docs/adrs/adr-001-initial-plan.md`,
`docs/adrs/adr-002-language-and-domain-design.md`); their still-valid
conclusions live in `docs/specs/README.md` now, and their full original reasoning
remains readable via `git log --follow -- docs/adrs/` or
`git show <commit>:docs/adrs/adr-002-language-and-domain-design.md` on a
commit before their removal. See "ADRs replaced by docs/specs/README.md" below for
why.

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
- [x] Decide a case-normalization policy for GitHub identifiers held by
      value objects (`valueobjects.Attribution.author`,
      `valueobjects.IssueRef`'s `owner`/`repo` — named `entry.Attribution`/
      `repositories.IssueRef` at the time this item was flagged, before the
      domain layer package reorganization below moved both into
      `valueobjects`) — **flagged by local review (2026-07-18), resolved
      (2026-07-19)**. See "Case-insensitive identifier equality
      (2026-07-19)" below for what was built.
- [x] Reconcile `CLAUDE.md`'s Evergreen Tests wording ("test names must
      not reference implementation details") with this project's actual,
      consistently-applied test-naming convention across every existing
      test file — **flagged by local review (2026-07-18) of
      `internal/infrastructure/github`, revisited and resolved
      (2026-07-19)**. See "Test-naming and error-message convention
      resolution (2026-07-19)" below for what was decided and built.
- [x] Decide whether `repositories.NewIssueRef` should reject `owner`/`repo`
      values containing path separators or `..` segments — **flagged by
      local review (2026-07-18) of `internal/infrastructure/persistence`,
      revisited and fixed (2026-07-19) now that Slice 5's CLI argument
      layer constructs `IssueRef` values from user-supplied `--repo`/
      positional-number input**. See "IssueRef owner/repo validation
      (2026-07-19)" below for what was built.
- [x] Decide whether `ExportService.Export`'s independent fetches
      (`FetchPullRequest`+`FetchReviewComments` vs. `FetchTimeline`) should
      run concurrently — **revisited and fixed (2026-07-19)** now that
      Slice 5's CLI wiring exists, closing the "no wired-up caller"
      deferral reason recorded here previously. See "Concurrent
      pull-request-chain/timeline fetch (2026-07-19)" below for what was
      built.

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

Broken into slices, each following the domain slice's own precedent: Red/
Green TDD, one concern per commit, `todo.md` updated in its own commit
separate from implementation. Package names below follow this user's
onion-architecture convention (`~/.agents/agent-docs/architecture/
onion-architecture.md`): domain defines abstract types, infrastructure
implements them, application orchestrates across layers, presentation is
the CLI entrypoint. Names are a starting proposal, not final — confirm at
each slice's own design step, same as the domain slice went through
`EnterPlanMode` before coding.

### Slice 1: Domain layer (Tier 1 entries + timeline classification) — done

See "Domain layer status" below.

### Slice 2: Acquisition (go-gh boundary + raw JSON persistence)

- [x] `internal/domain/repositories`: define the abstract port the
      application layer depends on to fetch raw evidence (dependency
      inversion — domain owns the interface, infrastructure implements
      it), e.g. `FetchIssue`/`FetchTimeline`/`FetchPullRequest`/
      `FetchReviewComments` returning `json.RawMessage`/`[]json.RawMessage`.
- [x] `internal/infrastructure/github`: `go-gh`-backed implementation.
      See "go-gh REST client (2026-07-18)" below for what was built.
      - [x] Issue/PR resource: `GET /repos/{owner}/{repo}/issues/{number}`
      - [x] Timeline, paginated: `GET .../issues/{number}/timeline` —
            **correction while implementing**: `EvidenceRepository.
            FetchTimeline` returns one `json.RawMessage` per item across
            all pages (matching `repositories.EvidenceRepository`'s
            already-existing interface contract), not one concatenated
            array; concatenating into ADR-002's single persisted array
            file is `internal/infrastructure/persistence`'s job, still
            pending below, not this item's. This is where the original
            todo's "pagination handling for long timelines" item lives; it
            is not a separate implementation step.
      - [x] Pull request resource: `GET /repos/{owner}/{repo}/pulls/{number}`
            (PRs only)
      - [x] Inline review comments: `GET .../pulls/{number}/comments`
            (PRs only). Joining these to their parent review via
            `pull_request_review_id` is already implemented on the domain
            side (`internal/domain/timeline.BuildEntries`); this item was
            only the REST fetch, not the join logic itself. Raw-JSON
            persistence is a separate item, still pending below.
      - [x] Rate-limit retry/backoff — neither `gh api` nor `go-gh`
            provides this (ADR-002 consequence); it is gh-exhibit's own
            responsibility. Design decided 2026-07-18: on 403/429, honor
            `Retry-After` (seconds) or `X-RateLimit-Reset` (epoch seconds)
            when either header is present; fall back to fixed exponential
            backoff only when neither header is present. Retry ceiling: 3
            attempts, then surface the error to the caller. No proactive
            `GET /rate_limit` check before a batch — reactive-only
            (respond to headers on each call's own response), revisited
            once Slice 5's argument shape (single/list/range/all)
            clarifies whether a "batch" large enough to justify a
            pre-flight check actually exists (YAGNI: the concept doesn't
            exist yet to design around).
- [x] `internal/infrastructure/persistence`: write raw JSON to the ADR-002
      layout (`issues/{repo}/{number}.json`, `.timeline.json`,
      `.pull.json`, `.review-comments.json`). See "Raw JSON persistence
      writer (2026-07-18)" below for what was built.
- [x] Boundary-layer tests: Detroit-school, exercised against a real
      `t.TempDir()` (deterministic, fast, no reason to mock a local
      filesystem). Qualitative branch coverage per ADR-002, no numeric
      floor for this layer.

### Slice 3: Document-level Tier 1 Markdown assembly — done

Per-entry `Render()` is already implemented for all four Tier 1 types
(`internal/domain/entry`: Body, IssueComment, PullRequestReview,
InlineReviewComment), each producing its `meta:{...}` line and content.
What remains is the document-level view (`meta:` lines already exist;
`------` separator and H1 title do not yet):

- [x] A document-assembly function (likely `internal/domain/entry` or a
      sibling package) combining the issue/PR's own title (not currently
      modeled on any Tier 1 entry — sourced from the raw issue/pull
      resource) with an ordered `[]entry.Entry`: H1 title line, then each
      entry's `Render()` output joined by `------`. See "Document
      assembly (2026-07-18)" below for what was built.
- [x] `internal/application/services`: orchestrate acquisition ->
      `timeline.BuildEntries` -> document assembly -> write
      `issues/{repo}/{number}.md`, tying Slice 2's output to Slice 1's
      domain logic. See "Export service orchestration (2026-07-18)" below
      for what was built.

### Slice 4: Attachment handling (mandatory local download)

- [x] **Design question resolved (2026-07-18), confirmed with the user**:
      detection/rewriting runs as a third option neither of todo.md's
      original two candidates named — a post-render transform over a
      `Document`'s fully rendered Markdown bytes, run by `ExportService`
      after `doc.Render(&buf)` and before `WriteDocument`. This was chosen
      over (a) rewriting raw body/comment text before entry construction
      or (b) adding a per-type content-mutation method to each of the four
      Tier 1 Value Objects: both would have required changing
      `timeline.BuildEntries`/`BuildBody`'s signatures or the sealed
      `Entry` interface, risking regressions in domain-layer code already
      hardened across six review rounds. Operating on the rendered bytes
      instead needs no such change, since the `user-attachments` URL
      pattern is structurally distinct from the `meta:{...}` line's own
      `url` field (an `issues/{number}#...` path, not
      `user-attachments/assets/...`), so a scoped regex over the full
      rendered text cannot collide with it. Accepted trade-off: a URL-
      shaped substring occurring inside a diff hunk would be rewritten
      too, indistinguishable at this layer from a genuine attachment
      reference — deliberately left unguarded against, no different in
      kind from this project's other "unobserved in practice" acceptances
      (e.g. `markSeen`'s `id <= 0` case).
- [x] `internal/domain/attachment` — pure detection/rewriting, no I/O:
      `Detect(markdown) []string` (deduplicated, first-seen order),
      `Resolution` (`Downloaded`/`FetchFailed`), `Rewrite(markdown,
      resolutions)`, and `Filename(url, contentType)` (UUID from the URL
      path plus an extension from an explicit content-type table — not
      `mime.ExtensionsByType`, whose result set is drawn from the host's
      own mime database and isn't guaranteed stable across platforms).
      C0: 96.6%.
- [x] **New ports, confirmed with the user**: dedicated
      `repositories.AttachmentFetcher` (HTTP fetch) and
      `repositories.AttachmentWriter` (`WriteAsset`/`WriteFetchErrorLog`)
      rather than extending `EvidenceRepository`/`EvidenceWriter` — a
      fetched attachment is a different artifact shape (binary blob) than
      either existing port's JSON concern. No dedicated test file,
      mirroring `EvidenceRepository`/`EvidenceWriter`'s own precedent.
- [x] `internal/infrastructure/github`: implement `AttachmentFetcher`
      against the authenticated `go-gh` HTTP client. See "Attachment
      fetcher and writer (2026-07-19)" below for what was built.
- [x] `internal/infrastructure/persistence`: implement `AttachmentWriter`,
      writing `issues/{repo}/{number}/assets/{filename}` and
      `issues/{repo}/{number}/fetch-errors.log`.
- [x] `internal/application/services`: wire `attachment.Detect`/`Rewrite`
      plus the two new ports into `ExportService.Export`, running after
      `doc.Render(&buf)` and before `WriteDocument`. On a per-URL fetch
      failure: skip, continue processing the rest (don't fail the whole
      export), rewrite that one reference to a placeholder, and persist
      the accumulated failure summary.

### Slice 5: CLI / gh extension entrypoint

- [x] `cmd/gh-exhibit/main.go` and `internal/presentation/cli`, per the
      onion-architecture doc's presentation layer, wiring Slice 2-4's
      pieces together via `internal/registry`-style DI. See "CLI
      entrypoint (2026-07-19)" below for what was built.
- [x] **Open design question resolved (2026-07-19), confirmed with the
      user**: argument shape is a single positional argument that is
      either one issue/PR number or a comma-separated list of them
      (`gh exhibit 123` / `gh exhibit 123,124,125`); no range/`--all`
      syntax in this first version (YAGNI, matches this project's
      existing deferral pattern). A list's per-ref failures are reported
      and the rest still run (confirmed with the user), rather than
      aborting the whole batch on the first error.
- [x] Target repository argument, defaulting to the current repo's context
      (matching gh extension conventions) when omitted.
- [x] Output directory argument and default (`-o`/`--output`, default
      `.`).
- [x] Distribution: `gh extension create` scaffolding (Go-only, per
      ADR-001) and a release workflow producing per-platform binaries.
      See "Distribution scaffolding (2026-07-19)" below for what was
      built.

### Domain layer status (2026-07-18)

Implemented on `feat/initial-implementation` via Red/Green TDD, one
concern per commit: `Attribution`, `ReviewState`, `InlineContext` value
objects; the sealed `Entry` interface; all four concrete Tier 1 types; and
`internal/domain/timeline`'s classify/join logic. No I/O — pure domain
logic per the onion-architecture boundary ADR-002 draws. C0 coverage:
`internal/domain/entry` 94.0%, `internal/domain/timeline` 96.9%, both above
ADR-002's 90% floor; remaining gaps are the sealed interface's no-op
`entryNode` markers and defensive `json.Marshal` error branches on
always-marshalable meta structs (not meaningfully testable). Two schema
details not evidenced by any real sample were proposed and confirmed with
the user: `PullRequestReview`'s meta line includes `state`; the diff hunk on
`InlineReviewComment` is rendered under an explicit `**Diff:**` label,
separate from the comment body, to avoid confusing GitHub-supplied context
with the author's own words. Next: acquisition (go-gh REST calls, raw JSON
persistence) and document-level Markdown assembly.

### Code review fixes (2026-07-18)

A local review of the domain layer found 3 correctness bugs and 3 cleanup
items, all addressed on `feat/domain-layer-implementation`:

- **Single-item failures no longer abort a whole batch.** `classify` and
  `BuildEntries` previously returned an error (aborting the entire call) the
  moment one timeline item or review comment failed to parse or violated a
  value object's invariant. Both now record a `SkipNote` (reason + raw
  JSON) for the offending item and continue, mirroring the attachment
  fetch-errors.log precedent (ADR-002) instead of letting one bad item take
  the whole run down.
- **Deleted-account comments** (`user: null`) are attributed to `"ghost"`
  (GitHub's own sentinel login) instead of failing `Attribution`'s
  non-empty-author invariant.
- **Outdated inline review comments** (`line: null`, common — roughly half
  of one real sampled PR's inline comments were in this state) fall back to
  `original_line` and render an `"outdated"` meta flag, instead of failing
  `InlineContext`'s positive-line invariant and being dropped.
- Verified against GitHub's REST docs that review dismissal is a separate
  timeline event type, not a `state` value on the `reviewed` event itself —
  the reviewer's cited "unrecognized review state" scenario is therefore
  unlikely to be reachable via that path in practice, though the new
  per-item skip mechanism still guards it defensively.
- Cleanup: removed the unused `commentedEventWire.ID` field; extracted a
  shared `writeMetaLine` helper so the four Tier 1 types no longer
  duplicate their meta-marshal logic; trimmed several code comments added
  without prior explicit permission (package/exported-type doc comments
  retroactively approved by the user; others judged individually).

C0 after these fixes: `internal/domain/entry` 98.6%, `internal/domain/timeline`
94.9%.

### Second review round (2026-07-18)

A follow-up local review (workflow effort=high) of the same diff found one
more correctness bug and 3 smaller issues, all addressed on
`feat/domain-layer-implementation`:

- **File-level review comments** (`subject_type: "file"`, a comment on the
  whole file rather than a diff position) have both `line` and
  `original_line` null — a variant the first round's outdated-line fix
  didn't cover, so `resolvedLine()` still fell through to `(0, outdated:
  true)` and the comment was silently dropped. `InlineContext.line` is now
  `*int` (mirroring `Body.ClosedAt`/`MergedAt`'s existing optional-field
  pattern): nil means no line applies at all, distinct from `outdated`
  (a historical line recovered via `original_line`). No live example was
  found across ~100 sampled `cli/cli` PRs, but `subject_type: "file"` is a
  real, currently-documented GitHub API value, so this was fixed
  defensively rather than left unverified.
- **`knownReviewIDs` id<=0 collision**: a malformed "reviewed" event
  missing its own `id` unmarshals to 0, which was registered as a "known"
  review id; a review comment whose own `pull_request_review_id` likewise
  defaulted to 0 would wrongly match it instead of falling through to the
  orphan path. Only ids `> 0` are registered now.
- **Dead `error` return removed**: `classify`/`BuildEntries` never returned
  a non-nil error once every failure path became a `SkipNote` (confirmed
  independently by two review rounds); both now return just
  `(results, skipped)`.
- Reaffirmed keeping `inline_review_comment.go`'s diff-hunk-label comment
  (a method-body comment, not a package/exported-type doc comment) — this
  was already an intentional, individually-judged exception from the first
  round, not an oversight.

C0 after this round: `internal/domain/entry` 98.7%, `internal/domain/timeline`
96.3%.

### Third review round (2026-07-18)

A third local review found one more correctness bug and 2 cleanup items,
all addressed on `feat/domain-layer-implementation`:

- **Duplicate `reviewed` events duplicated inline comments**: `BuildEntries`
  buckets review comments by numeric review id via `byReview[id]`. Two
  `reviewed` timeline items sharing the same id (e.g. overlapping
  pagination delivering the same event twice) each independently appended
  the review plus the *entire* comment list bucketed under that id, so the
  rendered Markdown showed every inline comment on that review twice.
  `classify` now tracks seen review ids (`id > 0` only, consistent with the
  earlier id<=0 guard) and records a `SkipNote` for a repeat instead of
  adding a second `reviewCandidate` for it.
- Cleanup: extracted a shared `attributionMeta{Author, Created}` struct,
  embedded first in each of the four Tier 1 types' meta struct, removing
  the duplicated `Author()`/`CreatedAt().UTC().Format(time.RFC3339)`
  construction from each `Render()` (output unchanged — the existing
  byte-for-byte tests caught any regression). Unified
  `equalTimePointers`/`equalIntPointers` into a single generic
  `equalPointers[T any](a, b *T, eq func(T, T) bool) bool`, so a future
  value object with another optional field reuses it directly instead of
  writing another nil-safe comparison from scratch.

C0 after this round: `internal/domain/entry` 98.8%, `internal/domain/timeline`
96.6%.

### Fourth review round (2026-07-18)

A fourth local review (workflow effort=high) found the third round's
duplicate-event fix had only been generalized to one of three symmetric
cases, plus 2 process items, all addressed on `feat/domain-layer-implementation`:

- **`commented` events and review comments shared the same duplicate-delivery
  risk as `reviewed` events, but only `reviewed` had been fixed.** The third
  round's dedup guard covered duplicate `reviewed` timeline items but not
  duplicate `commented` items (`commentedEventWire` had no `id` field to key
  on — one was previously removed as unused in the first round's cleanup,
  now reintroduced for this purpose) nor duplicate review comments from
  `GET /pulls/{number}/comments` (`reviewCommentWire` had no `id` field
  either, only `pull_request_review_id`, which identifies the parent review,
  not the comment itself). Both now carry their own `id` and are deduped the
  same way as `reviewed` events.
- **Generalized the dedup pattern instead of copy-pasting it a third time**:
  extracted `markSeen(seen map[int64]bool, id int64) bool`, used at all three
  call sites (`classify`'s `commented` and `reviewed` branches, and
  `BuildEntries`'s review-comment loop). `classify` now returns its
  `seenReviewIDs` set directly as `knownReviewIDs`, so `BuildEntries` no
  longer re-derives the same set by re-scanning `items` — removing both the
  redundant `O(n)` pass and the risk of its `id > 0` filter drifting from
  `classify`'s own copy. The `id <= 0` guard rationale (previously stated
  inline at the `BuildEntries` call site) is now centralized once in
  `markSeen`'s doc comment instead of being restated per call site.
- **Comment governance**: the reviewer flagged two WHY comments
  (dedup rationale in `classify` and in the now-removed `BuildEntries`
  re-scan) as added without an individually recorded approval, unlike the
  `inline_review_comment.go` diff-hunk-label exception this project already
  recorded. `CLAUDE.local.md`'s comment policy was broadened in the interim
  to permit non-obvious WHY/WHY-NOT comments generally, not only Godoc — the
  flagged comments (and `markSeen`'s new doc comment) fall under that
  standing policy rather than needing individual sign-off. The specific
  comment at the old `BuildEntries` re-scan site no longer exists; its code
  was eliminated by the generalization above.

C0 after this round: `internal/domain/entry` 98.8%, `internal/domain/timeline`
96.8%.

### Fifth review round (2026-07-18)

A fifth local review (workflow effort=high) found one scope question and
one cleanup item; only the cleanup was implemented on
`feat/domain-layer-implementation`.

- **`markSeen`'s `id <= 0` exclusion: reaffirmed as an accepted limitation,
  not a bug.** The reviewer noted `id <= 0` is never treated as a duplicate,
  so two malformed events sharing a defaulted id (missing/malformed id,
  itself never observed in ~100 sampled `cli/cli` PRs across four review
  rounds) that also happen to be delivered twice via overlapping pagination
  would render twice. This requires two independently rare conditions to
  co-occur, and the id-based scheme cannot structurally distinguish "the
  same event redelivered" from "two different malformed events that both
  defaulted to id 0" — only a content-hash key (on the raw JSON payload
  instead of the domain id) could, and that would be a different dedup
  model, not an extension of the current one, requiring either applying it
  uniformly to all events or running two dedup schemes side by side. Given
  no observed real-world occurrence of `id <= 0` at all, let alone a
  duplicated one, this is deliberately left as-is — decided with the user,
  not to be re-flagged as an unfixed bug in future rounds.
- Cleanup: `body_test.go`/`inline_review_comment_test.go`/
  `pull_request_review_test.go` each had a near-identical
  `newXAttribution(t) entry.Attribution` helper differing only in fixture
  values. Each now delegates to a single `newAttribution(t, author,
  created, url)` in `attribution_test.go`, added there rather than as a
  flat replacement at all 21 call sites — inlining the same three literal
  arguments 21 times would have traded helper-definition duplication for
  call-site duplication instead of removing it.

C0 after this round: `internal/domain/entry` 98.8%, `internal/domain/timeline`
96.8%.

### Acquisition port (2026-07-18)

`internal/domain/repositories` now defines the abstract port Slice 2's
remaining items build on, on `feat/acquisition-repositories-port`:

- `IssueRef` (owner, repo, number) — a new value object identifying which
  issue/PR to fetch, following the existing `Attribution`/`InlineContext`
  pattern (validating constructor, `Equals`). Nothing in the domain
  modeled this concept before.
- `EvidenceRepository` interface: `FetchIssue`/`FetchPullRequest` return a
  single `json.RawMessage`; `FetchTimeline`/`FetchReviewComments` return
  `[]json.RawMessage` (one element per item) rather than a concatenated
  blob, matching `timeline.BuildEntries`'s existing parameter shape so no
  translation step is needed downstream.
- No test file for the interface itself (no logic to exercise), mirroring
  `entry.Entry`'s sealed interface, which likewise has no dedicated test
  file of its own.
- Rate-limit retry/backoff (the one open design question under this
  slice) is still undecided and remains a separate item below; this port
  definition does not depend on that decision.
- A local review of this slice (workflow effort=high, 4 finders + 1
  verifier) flagged `IssueRef.Equals`'s case-sensitive owner/repo
  comparison (GitHub identifiers are case-insensitive for uniqueness).
  Verdict PLAUSIBLE, not fixed here: no production code calls any domain
  value object's `Equals` yet (test-only), and the same gap already
  exists in the already-merged `entry.Attribution`, so this is recorded
  as its own open design item in this file's "Design" section rather
  than patched in isolation.

### Sixth review round (2026-07-18)

A sixth local review found one more correctness bug and one leftover
generalization gap, both addressed on `feat/domain-layer-implementation`:

- **`InlineReviewComment.Render` broke on a diff hunk containing a
  triple-backtick run**: the diff hunk was embedded in a fixed ` ```diff `
  fence. A hunk whose content itself contains three or more consecutive
  backticks (e.g. a diff touching a Markdown file whose context lines
  include a fenced code block) closes the outer fence early, corrupting
  the rendered output. `diffFence` now measures the longest backtick run
  in the hunk and picks a fence one backtick longer (minimum 3), the
  CommonMark-standard technique for embedding arbitrary content in a
  fenced block.
- **`issue_comment_test.go` was the one file the `newAttribution` cleanup
  missed**: the tidy commit `fa7e5cf` unified the duplicated
  `NewAttribution`-plus-error-handling boilerplate in `body_test.go`,
  `inline_review_comment_test.go`, and `pull_request_review_test.go`
  behind a shared `newAttribution(t, author, created, url)`, but
  `issue_comment_test.go`'s four call sites were left as the original
  copy-pasted blocks. It now has its own `newIssueCommentAttribution(t)`
  thin wrapper, matching the other three files' pattern.

C0 after this round: `internal/domain/entry` 98.9%, `internal/domain/timeline`
96.8%.

### GitHub issue/PR templates (2026-07-18)

Added `.github/PULL_REQUEST_TEMPLATE.md` and `.github/ISSUE_TEMPLATE/
{BUG_REPORT,FEATURE_REQUEST}.md`, mirroring the section structure
`connect0459/starlark-mbt` uses for its own templates (Related Links,
`[Required] Overview`, Scope of Change, Breaking Changes, Deferred Items
and TODOs, Test Items, `[Required] Quality Checklist`). Fields specific to
MoonBit (target backend, `.mbti` diff, `moon info`) are replaced with this
project's own layout: Scope/Affected Package options list the onion-
architecture layers (`internal/domain`, `internal/infrastructure`,
`internal/application`, CLI/`cmd/gh-exhibit`) instead of MoonBit packages,
and the PR Quality Checklist references this repo's actual pre-commit
hooks (`golangci-lint`, `gofmt`, `go test ./...`) instead of a manually
triggered CI workflow run. This formalizes the structure PR #2's body
already followed by hand, so it is pre-filled for future issues/PRs
instead of copied each time.

### go-gh REST client (2026-07-18)

`internal/infrastructure/github` now implements
`repositories.EvidenceRepository` against GitHub's REST API via `go-gh`,
on `feat/acquisition-github-client`:

- `EvidenceRepository` wraps `*api.RESTClient` behind an unexported
  `requester` interface (matching `*api.RESTClient`'s own method
  signature exactly, so the real client satisfies it with no adapter),
  the same dependency-inversion seam this project already uses for
  `repositories.EvidenceRepository` itself — applied here to make the
  retry layer unit-testable without real HTTP.
- Rate-limit retry (`retry.go`) implements the design decided in this
  file's Slice 2 checklist: `errors.As` against `*api.HTTPError`
  (go-gh's own error type, which carries `StatusCode` and `Headers`) to
  decide retryability, `Retry-After`/`X-RateLimit-Reset` header-driven
  wait when present, fixed exponential fallback otherwise, 3-attempt
  ceiling. **Refinement found while implementing, not previously
  discussed**: a 403 is retried only when its headers identify it as a
  rate limit (`Retry-After` present or `X-RateLimit-Remaining: 0`) —
  GitHub returns 403 for both permission errors and rate limiting, and
  only the headers distinguish them, so retrying every 403 unconditionally
  would burn 3 attempts (and any accompanying wait) on an ordinary
  permission-denied response before surfacing it.
- Pagination (`pagination.go`) follows the `Link` response header's
  `rel="next"` relation; `FetchTimeline`/`FetchReviewComments` loop pages
  and return one `json.RawMessage` per item across all of them, matching
  `repositories.EvidenceRepository`'s pre-existing interface contract
  (see the corrected Slice 2 checklist entry above) rather than a single
  concatenated array — that concatenation is `internal/infrastructure/
  persistence`'s responsibility, not this package's.
- Boundary-layer tests (Detroit-school): `retry_test.go` uses a fake
  `requester` (no real HTTP) for the retry-decision branches;
  `evidence_repository_test.go` uses a real `httptest.Server` plus a
  custom `http.RoundTripper` that redirects `api.NewRESTClient`'s traffic
  to it — verified directly against `go-gh v2.13.0`'s source
  (`rest_client.go`, `client_options.go`, `http_client.go`) that setting
  `Host: "github.localhost"` (go-gh's own plain-HTTP-scheme hostname) with
  an explicit `AuthToken` and `Transport` skips `gh` config/auth
  resolution entirely, so these tests need no `gh` CLI or login. C0:
  `internal/infrastructure/github` 87.2% (qualitative target per
  ADR-002's boundary-layer policy, no numeric floor).
- `internal/infrastructure/persistence` (raw JSON file writer) is
  unstarted — out of scope for this branch, per this project's
  one-slice-item-per-branch precedent.

### Local review of internal/infrastructure/github (2026-07-18)

A local review found 2 correctness bugs and 1 cleanup-style finding,
addressed on `feat/acquisition-github-client`:

- **`fetchPaginated` trusted an unbounded chain of Link-header "next"
  URLs.** A misbehaving or misconfigured server (or a proxy rewriting
  responses) could make the loop fetch forever, with no ctx deadline
  guarding it in this package alone. Capped at `maxPaginationPages` (1000
  — 100,000 items at GitHub's page-size ceiling, far beyond any real
  issue/PR's timeline or review-comment count), returning an error
  instead of looping once exceeded.
- **`nextPageURL` split the `Link` header on every comma**, but RFC 3986
  permits an unescaped comma inside a URL's query string, so a URL like
  `?filter=a,b` would be split mid-URL, silently returning a garbage
  fragment as the "next" page instead of the real one. GitHub's current
  timeline/review-comments endpoints don't use comma-bearing queries, so
  this had no live impact, but the parser should be correct regardless.
  `splitLinkHeader` now tracks `<...>` bracket depth and only splits on a
  comma outside it.
- **Test-naming convention question, not fixed here**: the review flagged
  `TestNextPageURL_...`/`TestDoWithRetry_...` for naming an unexported
  function, citing `CLAUDE.md`'s Evergreen Tests wording. Checked against
  every existing test in the project (`TestClassify_...`,
  `TestBuildEntries_...`, `TestNewAttribution_...`, `TestNewIssueRef_...`,
  etc.) and found this package's tests match the project's actual,
  consistent convention exactly — every test names the (often unexported)
  unit under test. Renaming only this package's tests would trade one
  inconsistency for another. Recorded as a project-wide open design
  question in this file's Design section instead of fixed piecemeal.

C0 after these fixes: `internal/infrastructure/github` 89.0%.

### Encapsulation fix: infra-layer struct no longer exported (2026-07-18)

`NewEvidenceRepository` returned `*EvidenceRepository` (an exported concrete
type defined in `internal/infrastructure/github`), so a caller could type its
own variable as the infra-layer struct directly instead of the
`repositories.EvidenceRepository` port — an onion-architecture leak even
though every existing caller already happened to use the interface. Fixed on
`fix/github-evidence-repository-encapsulation`:

- `EvidenceRepository` renamed to unexported `evidenceRepository`;
  `NewEvidenceRepository` now returns `repositories.EvidenceRepository`, so
  the concrete type cannot be named outside this package.
- The standalone `var _ repositories.EvidenceRepository = (*EvidenceRepository)(nil)`
  assertion was removed — the constructor's return type now enforces the
  same guarantee, making the separate assertion redundant.
- Swept the rest of the repo for the same two patterns (exported
  infra-layer structs, `var _` assertions symptomatic of a leaked
  abstraction). No other instance found: `internal/infrastructure/github`'s
  other files (`retry.go`, `pagination.go`) export no types, and the only
  remaining `var _` occurrences are `internal/domain/entry`'s
  `var _ entry.Entry = entry.<Concrete>{}` compile-time checks, which are
  intra-layer (verifying the domain's own Value Object types satisfy the
  domain's own sealed `Entry` interface) rather than an architecture
  violation, so left as-is.

### Raw JSON persistence writer (2026-07-18)

Slice 2's remaining item is done, on `feat/acquisition-persistence-writer`:

- `internal/domain/repositories.EvidenceWriter` — a new port symmetric to
  the acquisition-side `EvidenceRepository`, so the application layer
  depends only on the abstract type for persisting raw evidence, not on
  how or where it is stored. No dedicated test file, mirroring
  `EvidenceRepository`'s own precedent (an interface has no logic to
  exercise).
- `internal/infrastructure/persistence` implements it against the local
  filesystem: `WriteIssue`/`WritePullRequest` write a single raw response
  verbatim; `WriteTimeline`/`WriteReviewComments` splice multiple pages'
  raw bytes directly into one JSON array (`[`, each item, `,` between,
  `]`) instead of `json.Marshal`-ing the `[]json.RawMessage` slice —
  `encoding/json` compacts each element's whitespace during marshaling,
  which would silently break ADR-001's verbatim-evidence guarantee for
  multi-page files even though single-file writes stayed byte-exact.
- Path scheme follows ADR-002 literally: `issues/{repo}/{number}.json`
  (plus `.timeline.json`/`.pull.json`/`.review-comments.json`). Owner is
  deliberately not part of the path — confirmed against the hand-maintained
  export directory this format was modeled on (ADR-001), which has no
  owner segment either.
- Boundary-layer tests (Detroit-school): exercised against a real
  `t.TempDir()` rather than a mocked filesystem, per ADR-002's boundary-
  layer policy — no reason to fake local file I/O when a real temp
  directory is just as fast and deterministic. C0:
  `internal/infrastructure/persistence` 89.5% (qualitative target, no
  numeric floor).

### Local review of internal/infrastructure/persistence (2026-07-18)

A local review (workflow effort=high, 4 finders + 1 verifier; the
workflow's final synthesis step itself failed on a structured-output
retry cap, so results were recovered from the run's journal rather than
its summary) found 4 correctness/robustness issues and 1 reachable-but-
currently-inert issue, addressed on `feat/acquisition-persistence-writer`
except where noted:

- **`WriteTimeline`'s `pages` parameter contradicted its own doc comment**
  (it holds one already-flattened item per element, same shape as
  `WriteReviewComments`' `items`) — renamed to `items` for consistency;
  the old name invited a future caller to pass unflattened per-page
  bodies, which `joinRawArray` would have spliced into a nested
  array-of-arrays with no compiler signal.
- **`joinRawArray` silently produced malformed JSON for an empty
  element** (e.g. `[,{"id":2}]`) — now returns an error instead of
  writing.
- **`ctx` was discarded by all four `EvidenceWriter` methods** — each now
  checks `ctx.Err()` before touching the filesystem, so a cancelled or
  expired context is honored instead of ignored.
- **`writeFile`'s two error-wrapping branches had no test forcing
  either to fail** — added coverage using a real temp directory (a
  plain file blocking `MkdirAll`, a read-only directory blocking
  `WriteFile`), consistent with this layer's no-mocking policy.
- **Path traversal via unsanitized `ref.Repo()`**: flagged as
  PLAUSIBLE, not fixed here — recorded as an open design question in
  this file's Design section instead, since `NewIssueRef` currently has
  no production caller and the right fix location is the domain-layer
  constructor's invariant, not a persistence-layer patch.
- Verified as working as designed, not a bug: the path scheme keys
  files by `repo`+`number` only, omitting `owner` — this is ADR-002's
  literal specification (`issues/{repo}/{number}.json`), confirmed by
  this diff's own `TestWriteIssue_OmitsOwnerFromThePath`.

C0 after this round: `internal/infrastructure/persistence` 94.3%.

### Document assembly (2026-07-18)

Slice 3's first item is done, on `feat/document-assembly`:

- `internal/domain/entry.Document` — a new value object combining a title
  string with an ordered `[]entry.Entry`, mirroring `Entry`'s own
  `Render(io.Writer) error` shape rather than a bare function, and staying
  in the `entry` package (design confirmed with the user) so the
  `------`-separator knowledge stays alongside the per-entry `meta:{...}`
  rendering it complements, instead of a new sibling package speculating
  about future Tier 2+ needs that don't exist yet.
- `NewDocument` validates only that `title` is non-empty (confirmed with
  the user), matching `NewAttribution`/`NewIssueRef`'s existing
  constructor-validation style; `entries` may be empty (e.g. `nil`) since
  "a Body entry always exists" is an application-layer composition
  invariant, not something the `Document` type itself should assume.
- Verified against the hand-maintained reference export that the separator
  is exactly `"\n------\n\n"` between one entry's `Render()` output (which
  already ends in its own trailing `\n`) and the next entry's `meta:{...}`
  line — reproducing the title/blank/meta/blank/content/blank/`------`/
  blank/meta pattern the existing hand-maintained directory uses.
- `internal/application/services` orchestration (acquisition ->
  `timeline.BuildEntries` -> document assembly -> write the rendered
  Markdown file) is unstarted — out of scope for this branch, per this
  project's one-slice-item-per-branch precedent; it also introduces the
  project's first `application` layer package, so it likely warrants its
  own design pass before coding starts.

C0: `internal/domain/entry` 96.3% (`Document.Render`'s error-propagation
branches after a successful write are the only new gap — the same
"defensive, not meaningfully testable without a fake failing io.Writer"
shape as this package's existing uncovered `json.Marshal`-error branches,
not a new kind of gap).

### Local review of Document (2026-07-18)

A local review (workflow effort=high) found 2 correctness/robustness
issues, both addressed on `feat/document-assembly`:

- **`Document` was mutable despite being a value object**: `NewDocument`
  stored the caller's `entries` slice by reference, and `Entries()`
  returned that same internal slice — mutating either after construction
  silently changed a `Document`'s rendered output, violating this
  project's Immutable First principle. Both now go through `slices.Clone`
  (constructor stores a clone of the input; `Entries()` returns a clone of
  the stored slice), so no path lets a caller mutate a `Document`'s state
  after construction.
- **Nil entries went unvalidated**: a nil element in `entries` panicked
  inside `Render()`'s `e.Render(w)` instead of failing at construction.
  `NewDocument` now rejects a nil entry with an error identifying its
  index, mirroring the array-element validation `evidence_writer.go`'s
  `joinRawArray` already does for empty `json.RawMessage` elements.

C0 after this round: `internal/domain/entry` 96.4%.

### Export service orchestration (2026-07-18)

Slice 3's remaining item is done, on `feat/application-export-service`,
closing out Slice 3:

- `internal/domain/timeline/body.go` — two new functions filling the one
  gap left after Document assembly: `IsPullRequest(rawIssue)` detects a
  pull request via the issue resource's own `pull_request` key (GitHub's
  issues endpoint serves both issues and PRs), and
  `BuildBody(rawIssue, rawPullRequest)` builds the document title and the
  `Body` Tier 1 entry from the issue/PR resource. `merged_at` is never
  present on the issues endpoint's own response, so it is only ever
  sourced from `rawPullRequest` (nil/empty for a plain issue). Placed in
  `internal/domain/timeline` rather than a new package — confirmed with
  the user — since this package already treats itself as "wherever raw
  GitHub JSON becomes an `entry.Entry` value" (`InlineReviewComment` is
  already built there from a non-timeline endpoint).
- `internal/domain/repositories.DocumentWriter` — a new port, separate
  from `EvidenceWriter`, confirmed with the user: `EvidenceWriter` is
  explicitly scoped to raw JSON evidence, while the rendered Markdown is
  ADR-002's "regenerable view," a deliberately different concern.
  `WriteDocument(ctx, ref, rendered []byte)` takes already-rendered bytes
  (mirroring `WriteIssue`'s shape) so `domain/repositories` stays
  decoupled from `domain/entry`.
- `internal/infrastructure/persistence.NewDocumentWriter` implements it
  against the local filesystem, writing `issues/{repo}/{number}.md` per
  ADR-002. `evidenceWriter`'s `issuePath` method was lifted into a
  package-level function so both writers share the same path-building
  logic instead of duplicating it.
- `internal/application/services.ExportService` — the project's first
  `application`-layer package. `Export(ctx, ref)` fetches the issue
  resource, conditionally fetches the pull resource and review comments
  when `IsPullRequest` is true, persists every raw response via
  `EvidenceWriter`, classifies the timeline via `timeline.BuildEntries`,
  assembles a `Document` (`Body` first, then the classified entries), and
  persists the rendered Markdown via `DocumentWriter`. Returns
  `[]timeline.SkipNote` rather than discarding them, so a future caller
  (Slice 5's CLI, not yet designed) can decide how to surface them; any
  other failure aborts the whole export and returns a wrapped error.
- Tests: Detroit-school in-memory fakes for all three ports
  (`fakeEvidenceRepository`/`fakeEvidenceWriter`/`fakeDocumentWriter`),
  following `internal/infrastructure/github/retry_test.go`'s existing
  "fake requester" precedent — no mocking library, matching this
  project's boundary-mocking policy (these fakes stand in for the
  application layer's own external-boundary ports).
- No manual/CLI smoke test possible yet: Slice 5 (the CLI entrypoint) is
  still unimplemented, so `ExportService` has no wired-up caller in this
  branch, consistent with this project's one-slice-item-per-branch
  precedent.

C0: `internal/domain/timeline` 96.4% (up from 96.3%; new coverage for
`IsPullRequest`/`BuildBody` well above ADR-002's 90% floor),
`internal/infrastructure/persistence` 94.9% (qualitative target, no
numeric floor), `internal/application/services` 92.1% (no numeric floor
was pre-agreed for this new layer — the remaining gap is
`entry.NewDocument`/`Document.Render`'s own defensive error branches,
already an accepted, not-meaningfully-testable-without-a-fake-failing-
writer gap elsewhere in the project, not a new kind of one).

### Local review of the export service orchestration (2026-07-18)

A local review (high effort, verified) of the above found one correctness
bug and one simplification, both addressed on
`feat/application-export-service`; one efficiency finding was confirmed
but deferred (see this file's Design section); one additional claim was
checked and found not to reproduce.

- **A `BuildBody` failure left a partial evidence directory behind.**
  `Export` used to interleave fetches with writes (`WriteIssue`, then
  conditionally `WritePullRequest`/`WriteReviewComments`, *then*
  `BuildBody`, then `WriteTimeline`), so a `BuildBody` validation failure
  (e.g. an empty `html_url` failing `entry.NewAttribution`) could abort
  the export after raw JSON was already written but before
  `{number}.timeline.json` or `{number}.md` existed — a state a future
  consumer checking "does the evidence directory exist" for completion
  would misread as done. `Export` now runs every fetch and build/validate
  step (`FetchIssue` through `Document.Render`) to completion *before* any
  write is attempted, so a failure anywhere in that phase leaves nothing
  on disk at all.
- **`IsPullRequest` and `BuildBody` each independently unmarshaled the
  same `rawIssue`** into `issueResourceWire`, redoing the same parse
  twice and requiring both to stay in sync by hand.
  `IsPullRequest(json.RawMessage) bool` is replaced by
  `ParseIssueResource(rawIssue) (IssueResource, error)` plus an
  `IssueResource.IsPullRequest() bool` method; `BuildBody` now takes the
  already-parsed `IssueResource` instead of re-parsing `rawIssue` itself.
  As a side effect, a malformed `rawIssue` now surfaces as an explicit
  error from `ParseIssueResource` immediately, instead of being silently
  treated as "not a pull request" until `BuildBody` failed later.
- **Deferred**: `Export`'s independent fetches (`FetchPullRequest`+
  `FetchReviewComments` vs. `FetchTimeline`) still run sequentially even
  though neither depends on the other — recorded as a new open design
  question in this file's Design section rather than fixed here, since
  this codebase has no concurrency precedent yet and no production caller
  exists to observe the real latency impact (Slice 5 is unimplemented).
- **Checked and not reproduced**: a claim that `IsPullRequest` (now
  `IssueResource.IsPullRequest`) would misread an explicit
  `"pull_request": null` as "present" was verified against
  `json.RawMessage.UnmarshalJSON`'s actual behavior and found not to
  hold — `null` unmarshals to a nil/empty `json.RawMessage`, which
  `len(...) > 0` already correctly treats as absent.

C0 after this round: `internal/domain/timeline` 97.3%,
`internal/application/services` 92.9%.

### Local review of internal/domain/attachment (2026-07-18)

A local review found 2 correctness/efficiency issues, both addressed on
`feat/attachment-detection`:

- **`Resolution.ok()` inferred success from `reason == ""`**, so
  `FetchFailed("")` (an empty failure reason) or a zero-value `Resolution`
  was indistinguishable from a successful `Downloaded("")`. `Rewrite`
  would then silently substitute the empty local path, dropping the
  attachment reference entirely (e.g. `![alt]()`) instead of emitting
  ADR-002's "attachment unavailable: reason" placeholder. `Resolution` now
  carries an explicit `succeeded bool` field set by `Downloaded`/
  `FetchFailed`, rather than inferring success from any other field being
  empty.
- **`Rewrite` rescanned the full rendered Markdown once per attachment
  URL** (`bytes.ReplaceAll` in a loop over `resolutions`), an
  O(N×len(markdown)) cost for N attachments instead of one pass. Replaced
  with a single `strings.Replacer` built from all url/replacement pairs,
  rewriting the whole buffer in one pass; also short-circuits when there
  are no resolutions instead of doing a needless byte<->string round trip.

C0 after this round: `internal/domain/attachment` 96.8%.

### Domain layer package reorganization (2026-07-19)

The user flagged that `internal/domain`'s four packages mixed two
different splitting axes: `repositories` was split by type (abstract
ports), while `entry`/`timeline`/`attachment` were split by feature —
inconsistent granularity within the same layer. Discussed and confirmed
with the user on `refactor/domain-layer-layout`, before resuming Slice 4's
remaining implementation items:

- Adopted this user's own onion-architecture reference
  (`~/.agents/agent-docs/architecture/onion-architecture.md`), which
  specifies `domain/{entities, repositories, services}` as the canonical Go
  layout, over splitting the whole project by features — the latter is
  that same document's explicitly-scoped scale-out escape hatch for
  multiple independent feature domains with their own
  application/infrastructure/presentation slices, a precondition this
  single-feature (Issue/PR export) project does not meet yet.
- Used `valueobjects`, not `entities`: every type in `internal/domain/entry`
  plus `repositories.IssueRef` is a Value Object (no identity-based
  tracking); no Entity exists anywhere in the domain layer, so an
  `entities/` directory would name a concept this codebase doesn't have.
- `internal/domain/entry` renamed to `internal/domain/valueobjects`
  (package `entry` → `valueobjects`); `repositories.IssueRef` moved into it
  (a Value Object, not a port).
- `internal/domain/timeline` and `internal/domain/attachment` merged into
  `internal/domain/services` (package `timeline`/`attachment` →
  `services`) — both are stateless transformation logic spanning multiple
  Value Objects (timeline classification/joining; attachment
  detection/rewriting), matching the reference doc's `domain/services`
  bucket. No identifier collisions existed between the two merged
  packages, confirmed before merging.
- `internal/domain/repositories` now holds only the five ports
  (`EvidenceRepository`, `EvidenceWriter`, `DocumentWriter`,
  `AttachmentFetcher`, `AttachmentWriter`).
- Mechanical only — no logic changed. Verified with `go build ./...`,
  `go vet ./...`, `go test ./...` (all packages pass), and `gofmt -l .`
  (clean). C0 after the move: `internal/domain/valueobjects` 94.3%,
  `internal/domain/services` 97.2%, `internal/application/services` 92.9%,
  `internal/infrastructure/github` 89.0%,
  `internal/infrastructure/persistence` 94.9% — all at or above their
  prior figures, confirming no coverage regression from the package split.
- `.github/ISSUE_TEMPLATE/BUG_REPORT.md`'s affected-package checklist
  updated to the new package names (it still listed the pre-attachment-
  package `entry`/`timeline`/`repositories` set, already stale
  independent of this rename).

### EvidenceRepository renamed to EvidenceFetcher (2026-07-19)

Follow-up flagged by the user right after the reorg above:
`repositories.EvidenceRepository` was the one port left naming itself
`...Repository` while every sibling port uses a verb-derived noun
(`AttachmentFetcher`, `AttachmentWriter`, `EvidenceWriter`,
`DocumentWriter`) — the same inconsistent-granularity smell as the
package-layout issue, just one level down. Fixed on the same
`refactor/domain-layer-layout` branch:

- `internal/domain/repositories/repository.go` renamed to
  `evidence_fetcher.go`; `EvidenceRepository` → `EvidenceFetcher`.
- `internal/infrastructure/github/evidence_repository.go` renamed to
  `evidence_fetcher.go` (test file likewise); unexported
  `evidenceRepository` → `evidenceFetcher`, `NewEvidenceRepository` →
  `NewEvidenceFetcher`, and the test helper `newTestRepository` →
  `newTestFetcher`.
- `ExportService`'s `repo` field/constructor param renamed to `fetcher`
  to match its new type and its sibling fields (`writer`, `docs`) —
  production code only; test-file local variables named `repo` were left
  as-is (`internal/application/services/export_service_test.go` embeds
  `repo` inside GitHub URL test fixtures like
  `https://github.com/example/repo/issues/1`, so a blanket rename there
  risked corrupting test data rather than renaming an identifier).
- Mechanical only. Verified with `go build ./...`, `go vet ./...`,
  `go test ./... -cover`, and `gofmt -l .`: all pass, coverage unchanged
  from the prior entry's figures.

### Local review of the domain layer reorg (2026-07-19)

A local review of the two package-reorg commits above found one
correctness bug and one cleanup item, both addressed on
`refactor/domain-layer-layout`:

- **The `timeline` → `services` package rename's blanket `"timeline."` →
  `"services."` substitution matched inside the `"timeline.json"` string
  literal too**, in `internal/infrastructure/persistence/evidence_writer.go`.
  This silently changed the persisted evidence file from ADR-002's
  `issues/{repo}/{number}.timeline.json` to `.services.json` — a real
  on-disk output change the reorg's "mechanical only, no logic changes"
  framing did not intend. `evidence_writer_test.go`'s assertions had
  already been rewritten to match the broken filename (`42.services.json`),
  so the test suite could not catch this on its own. This is the same
  string-literal-collision risk already identified and guarded against
  during the reorg (see the `entry`/`timeline`/`attachment` string-literal
  checks above and the `repo` variable-name decision in the
  `EvidenceFetcher` rename entry) — but the check at the time only covered
  `internal/domain/services` and `internal/application/services`, not
  `internal/infrastructure/persistence`, so this instance was missed.
  Restored `"timeline.json"` in both the writer and its test assertions.
- Cleanup: `internal/domain/valueobjects/pull_request_review.go`'s doc
  comment still said "see the timeline package" after that logic moved
  into `services`; updated to match.
- No other instance of this class of collision was found in a repo-wide
  sweep after the fix. C0 unchanged from the prior two entries'
  figures; `go build ./...`, `go vet ./...`, `go test ./... -cover`, and
  `gofmt -l .` all pass.

### Attachment fetcher and writer (2026-07-19)

Slice 4's remaining three items are done, on `feat/attachment-handling`,
closing out Slice 4:

- `internal/infrastructure/github.AttachmentFetcher` wraps a plain
  `*http.Client` (`api.NewHTTPClient`), not `evidenceFetcher`'s
  `*api.RESTClient`: attachment URLs (`github.com/user-attachments/...`)
  are absolute and unrelated to the REST API host `*api.RESTClient` builds
  requests relative to, so the two fetchers need different go-gh client
  shapes even though both are go-gh-backed. **Confirmed with the user**:
  no rate-limit retry/backoff for attachment fetches, unlike
  `evidenceFetcher` — ADR-002 already treats a fetch failure as
  skip-and-continue rather than something worth retrying, and the
  attachment CDN is a separate rate-limit domain from the REST API
  `doWithRetry` was designed for.
- `internal/infrastructure/persistence.AttachmentWriter` reuses the
  existing `writeFile` helper; a new `issueDir(baseDir, ref)` builds the
  per-number directory (`issues/{repo}/{number}/`) both `WriteAsset`
  (`.../assets/{filename}`) and `WriteFetchErrorLog`
  (`.../fetch-errors.log`) write under, distinct from `issuePath`'s
  per-number *file* path since these two artifacts don't share
  `issuePath`'s one-file-per-suffix shape.
- `ExportService.Export` now runs attachment resolution
  (`services.Detect` → per-URL `AttachmentFetcher.Fetch` →
  `services.Filename`/`Downloaded`/`FetchFailed` → `services.Rewrite`)
  after `doc.Render(&buf)` and before any write, preserving the existing
  "every fetch/build step before any write" invariant from the export
  service orchestration review. A successful download's local path is
  written into the rewritten Markdown as `{number}/assets/{filename}`,
  relative to the rendered `.md` file's own directory. **Confirmed with
  the user**: `fetch-errors.log` is written only when at least one
  attachment fetch fails in that run, not unconditionally (an
  all-succeeded or no-attachments export leaves no
  `fetch-errors.log` behind).
- Tests: Detroit-school throughout, matching this project's existing
  precedent per layer — `attachment_fetcher_test.go` uses a real
  `httptest.Server` (via the existing `rewriteTransport` fixture,
  reused from `evidence_fetcher_test.go`) and asserts the configured
  auth token is forwarded, since that is attachment fetching's whole
  reason for needing an authenticated client (private-repo attachments);
  `attachment_writer_test.go` uses a real `t.TempDir()`;
  `export_service_test.go` adds `fakeAttachmentFetcher`/
  `fakeAttachmentWriter` alongside the three existing fakes.

C0: `internal/infrastructure/github` 88.1% (up from 89.0% before this
package gained a second file — the fetcher itself is fully covered;
the percentage move is `evidenceFetcher`'s existing untested branches
now being a smaller share of a larger package, not new gaps),
`internal/infrastructure/persistence` 95.7% (up from 94.9%),
`internal/application/services` 95.4% (up from 92.9%).

### Local review of the attachment fetch/write/wiring (2026-07-19)

A local review of the above found 5 issues, 4 addressed on
`feat/attachment-handling`; the fifth was discussed with the user and
accepted as a known limitation rather than fixed.

- **Context cancellation was misclassified as an ordinary attachment
  fetch failure.** `resolveAttachments` recorded every `Fetch` error —
  including `context.Canceled`/`context.DeadlineExceeded` — as a
  `FetchFailed` resolution, so a caller cancelling `Export` mid-run got a
  permanent "attachment unavailable" placeholder and a `fetch-errors.log`
  entry instead of `Export` aborting, unlike every other fetch step
  (`FetchIssue`, `FetchTimeline`, ...), which already propagates a context
  error as an aborting failure. `resolveAttachments` now checks
  `errors.Is` against both context error values first and returns them as
  an aborting error before any per-URL resolution logic runs.
- **Attachment fetches now run concurrently, bounded at
  `maxConcurrentAttachmentFetches` (4) in flight**, addressing both the
  sequential-fetch cleanup finding and, as a side effect, the
  latency-regression finding below: `Export`'s context budget now has to
  cover every attachment's download time in addition to the REST calls
  it already covered, and concurrency is the direct lever for keeping
  that additional cost from scaling linearly with attachment count. This
  is the codebase's first use of goroutines; `fakeAttachmentFetcher` in
  `export_service_test.go` was made safe for concurrent calls (a mutex
  guarding `fetchedURLs`), and `go test -race` was added to this
  package's verification going forward.
- **A stale `fetch-errors.log` from a prior failing run survived a rerun
  where every attachment succeeded**, since `WriteFetchErrorLog` was only
  called when the current run had a failure. `Export` now calls it
  unconditionally; `attachmentWriter.WriteFetchErrorLog` treats an empty
  log as "remove any existing file" rather than writing an empty one,
  self-healing a stale log on the next successful run instead of leaving
  it as a false signal.
- **`Export`'s docstring overclaimed write-phase atomicity.** "Either
  everything below succeeds and every file is written, or nothing is"
  was never actually true once the write phase itself starts — a failure
  between, say, `WriteIssue` and `WriteTimeline` already left a partial
  evidence directory before this branch existed; this branch's new
  `WriteAsset` loop only widened that pre-existing surface area. Fixed by
  narrowing the docstring to the guarantee this project actually built
  and tested: nothing is written until every fetch/build/validation step
  (attachment downloads included) has succeeded; the write phase itself
  has no rollback.
- **Discussed with the user, accepted as a known limitation, not
  fixed**: real write-phase atomicity (e.g. staging writes to temporary
  files and atomically renaming the whole evidence directory into place)
  was not implemented. `docs/todo.md`'s existing framing of the evidence
  directory as a "regenerable view" (ADR-002) means a partial write from
  an interrupted run is self-healing on the next successful `Export` call
  for the same `ref`, and Slice 5's CLI (the only prospective caller) is
  still unimplemented, so there is no observed real-world impact to size
  a fix against yet. Building a transactional/staged-write mechanism for
  local files is a disproportionate investment for this tool's scale
  without that evidence — the same "no production caller, defer" pattern
  already applied to the path-traversal and fetch-concurrency items
  above.

C0 after this round: `internal/application/services` 96.2% (up from
95.4%), `internal/infrastructure/persistence` 96.2% (up from 95.7%).
`go build ./...`, `go vet ./...`, `go test ./... -race -cover`, and
`gofmt -l .` all pass.

### CLI entrypoint (2026-07-19)

Slice 5's remaining implementation items (distribution scaffolding aside)
are done, on `feat/cli-entrypoint`, resolving this file's last open design
question:

- `internal/presentation/cli` — three small, independently testable pieces
  rather than one large `Run` function, matching this codebase's existing
  style of small composable functions:
  - `ParseArgs(args []string) (Args, error)`: parses the positional
    number-or-comma-list argument plus `--repo`/`-o`/`--output` flags via
    stdlib `flag` (`flag.NewFlagSet`, `ContinueOnError`). **Confirmed with
    the user**: no Cobra or other CLI framework — there is exactly one
    action (export) and no subcommands, and `go.mod` had no CLI-framework
    dependency to begin with, so stdlib `flag` avoids an unneeded new one.
    `--help` propagates `flag.ErrHelp` transparently (via `%w` wrapping) so
    `main` can exit `0` for it instead of printing an extra "error" line.
  - `ResolveRepo(flagRepo string, current func() (repository.Repository,
    error)) (repository.Repository, error)`: an explicit `--repo` is
    parsed via `go-gh`'s `repository.Parse`; otherwise delegates to
    `current` — an injected seam (same pattern as `retry.go`'s `sleeper` /
    `evidence_fetcher.go`'s `requester`) so tests don't depend on real git
    remotes. The production path passes `repository.Current` (also from
    `go-gh`, previously unused in this codebase despite being readily
    available).
  - `RunExports(ctx, exporter Exporter, owner, repo string, numbers []int,
    stdout, stderr io.Writer) int`: loops over `numbers`, reporting one
    line per ref to `stdout` on success or `stderr` on failure. **Confirmed
    with the user**: a failing ref does not stop the remaining ones in the
    same batch (this project's existing skip-and-continue precedent —
    attachment fetch failures, per-item `SkipNote`s — extended to the
    ref-batch level); returns `0` only if every ref succeeded. `Exporter`
    is a narrow interface defined in this package (not
    `*services.ExportService` directly) purely so tests can inject a fake;
    `*services.ExportService` satisfies it structurally on the production
    path, no adapter needed.
- `internal/registry.NewExportService(host, outputDir string)
  (*services.ExportService, error)` — the DI root wiring the two existing
  go-gh-backed fetchers (`github.NewEvidenceFetcher`/
  `NewAttachmentFetcher`, both scoped to `host`) together with the three
  existing filesystem-backed writers (`persistence.NewEvidenceWriter`/
  `NewDocumentWriter`/`NewAttachmentWriter`, both scoped to `outputDir`)
  into one `services.NewExportService(...)` call. No dedicated test file —
  same precedent already established for `internal/domain/repositories`'s
  port interfaces and this constructor's own upstream constructors: no
  branching logic of its own to exercise, just sequential composition of
  already-tested pieces.
- `cmd/gh-exhibit/main.go` — thin composition root: `ParseArgs` ->
  `ResolveRepo` -> `registry.NewExportService` -> `RunExports` ->
  `os.Exit`. No test file, standard Go convention.
- Tests: Detroit-school throughout, no mocking library — `fakeExporter` in
  `run_test.go` mirrors `export_service_test.go`'s existing fake-port
  style (canned results keyed by ref number). C0:
  `internal/presentation/cli` 94.2%.
- Manual smoke test (the first point in this project where an end-to-end
  run against the real GitHub API was even possible): built the binary and
  ran it against real `cli/cli` PRs (#13084, #13780) — with an explicit
  `--repo`, with the default repo-context resolution (a throwaway git repo
  with only a `cli/cli` remote configured), as a comma-separated list of
  two numbers, and with one nonexistent number mixed into a list (exit
  `1`, the valid number in the same batch still exported). Output matched
  ADR-002's layout (`issues/{repo}/{number}.json`/`.md`/etc.) in every
  case.
- Not yet done: `gh extension create` distribution scaffolding and a
  release workflow (this file's remaining unchecked Slice 5 item) — out of
  scope for this branch, per this project's one-slice-item-per-branch
  precedent.

`go build ./...`, `go vet ./...`, `go test ./... -race -cover`, and
`gofmt -l .` all pass.

### Local review of the CLI entrypoint (2026-07-19)

A local review found 3 correctness bugs, 1 test-coverage gap, and 1
simplification, all addressed on `feat/cli-entrypoint`:

- **`ParseArgs` broke when a flag followed the positional number.**
  `flag.FlagSet.Parse` stops scanning for flags at the first non-flag
  token, so `gh-exhibit 42 --repo owner/repo` misread `--repo` and its
  value as two extra positional arguments (`"expected exactly one ...,
  got 3"`) instead of a flag. `ParseArgs` now splits flag tokens from the
  positional argument itself (`splitFlagsAndPositional`) before handing
  the flag tokens to `flag.Parse`, so flags may appear before, after, or
  on both sides of the number/list.
- **A bare negative number was misread as an unrecognized flag.**
  `gh-exhibit -1` produced `"flag provided but not defined: -1"` instead
  of `parseNumbers`'s own `"must be positive"` validation error — the same
  root cause as the previous bug. `splitFlagsAndPositional` now routes a
  negative-number-shaped token (e.g. `-1`, `-1,2`) to the positional
  argument instead of forwarding it to `flag.Parse`.
- **`RunExports`'s success message ignored `--output`.** It always printed
  `issues/{repo}/{number}.md` regardless of `args.OutputDir`, so a
  non-default `--output` left the printed path pointing at the wrong
  location relative to where the evidence was actually written.
  `RunExports` now takes `outputDir` and builds the reported path with
  `filepath.Join(outputDir, "issues", repo, "{number}.md")`, matching how
  the persistence layer builds the real one.
- **`registry.NewExportService(host, outputDir string)` risked a silent
  argument swap**: both parameters are plain strings with no
  compiler-visible distinction, and neither the compiler nor a test could
  catch a transposed call (both constructions still "succeed" — the wrong
  host or directory only shows up at runtime). **Confirmed with the
  user**: fixed by taking a `Config{Host, OutputDir}` struct instead,
  naming the fields at the one existing call site, rather than adding a
  test (which couldn't meaningfully catch this class of bug here) or
  leaving it as an accepted single-call-site risk per this project's usual
  low-impact deferral pattern — the user preferred removing the risk at
  the type level.
- Cleanup: `RunExports`'s two near-identical success-message
  `fmt.Fprintf` branches (skip-count present/absent) are unified into one
  message built once, then printed once.
- All three correctness bugs were reproduced against the actual built
  binary before fixing (not just inferred from reading the code), and the
  fixes were re-verified the same way afterward, including a real
  `--output` smoke test against `cli/cli` PR #13084.

C0 after this round: `internal/presentation/cli` 92.1% (down slightly
from 94.2% — `splitFlagsAndPositional`'s unknown-flag-forwarding branch
and a couple of `looksLikeANegativeNumberList` edge cases are the new,
lightly-exercised surface; still well above this layer's qualitative
target). `go build ./...`, `go vet ./...`, `go test ./... -race -cover`,
and `gofmt -l .` all pass.

### IssueRef owner/repo validation (2026-07-19)

Revisits the path-traversal design question deferred during the
persistence-writer review, on `fix/issue-ref-owner-repo-validation`, now
that `internal/presentation/cli.RunExports` constructs `IssueRef` values
from user-supplied `--repo`/positional-number input (confirmed by tracing
`ref.Owner()`/`ref.Repo()`'s only production call sites:
`internal/infrastructure/github/evidence_fetcher.go`, which interpolates
both directly into the REST API request path, and
`internal/infrastructure/persistence`'s `issuePath`/`issueDir`, which
`filepath.Join` both into the on-disk evidence path — a `repo` of `".."`
reaches the filesystem path today, since `filepath.Join`'s `Clean`
resolves `..` lexically rather than rejecting it).

- **Scope confirmed with the user**: strict validation matching GitHub's
  own documented username/organization and repository name rules, not the
  minimal "reject a path separator or `..`" fix originally scoped by the
  deferred item — a deliberate scope increase the user chose explicitly.
- `NewIssueRef`'s `owner` argument must now match GitHub's username rule
  (`ownerPattern`: alphanumeric characters separated by single hyphens,
  never leading/trailing) and GitHub's 39-character username limit
  (`maxOwnerLength`).
- Its `repo` argument must match GitHub's repository-name rule
  (`repoPattern`: letters, digits, hyphens, underscores, periods), must not
  be exactly `"."` or `".."`, and must not exceed GitHub's 100-character
  repository-name limit (`maxRepoLength`).
- Both patterns exclude `/` and `\` from their character classes, so a
  path separator is rejected as an invalid character rather than needing a
  separate check; `.`/`..` needed their own explicit check since GitHub's
  repository-name character class otherwise allows periods.
- No existing production call site or test fixture broke: every
  already-committed `NewIssueRef` fixture across the test suite
  (`octocat`, `hello-world`, `connect0459`, `gh-exhibit`,
  `some-other-owner`) is alphanumeric-and-hyphen-only, well within both new
  patterns.
- Tests: Red/Green TDD, following `issue_ref_test.go`'s existing
  `TestNewIssueRef_Rejects...`/`Accepts...` naming convention — one test
  per rejected/accepted shape (slash, backslash, leading/trailing/
  consecutive hyphen, underscore, max-length overflow, `.`/`..`, invalid
  character), rather than one combined table-driven test, matching this
  file's existing per-case style.

C0 after this round: `internal/domain/valueobjects` 94.9% (up from 94.3%).
`go build ./...`, `go vet ./...`, `go test ./... -race -cover`, and
`gofmt -l .` all pass.

### Local review of the IssueRef validation (2026-07-19)

A local review (workflow effort=high, verified) found no correctness bugs
but one confirmed comment-policy violation, addressed on
`fix/issue-ref-owner-repo-validation`:

- **The `maxOwnerLength`/`maxRepoLength` const block and the
  `ownerPattern`/`repoPattern` package-level variables each carried a
  multi-line comment explaining the GitHub-limits reference and the
  path-traversal rationale.** Neither is a function/method Godoc, the one
  exception `CLAUDE.local.md` grants without per-instance permission; both
  were added without asking first. Flagged a real tension while resolving
  this: `docs/todo.md`'s own Fourth review round entry (2026-07-18) states
  `CLAUDE.local.md`'s comment policy was broadened to permit non-obvious
  WHY/WHY-NOT comments generally, but the file's current wording ("same as
  the global convention" for non-Godoc comments) reads as still requiring
  individual permission per instance, not a standing blanket approval.
  This discrepancy between what todo.md previously recorded and
  `CLAUDE.local.md`'s actual current text is unresolved as its own
  question — not adjudicated here, since it's a documentation-consistency
  question independent of this branch's change.
- **Resolved for this instance**: rather than seek individual permission,
  both comments were removed; the WHY they carried (GitHub's documented
  username/repository-name limits; the path-traversal rationale for
  excluding `/`/`\` from each pattern's character class) is already
  recorded in this PR's commit message and in this file's own entry above,
  per the reviewer's suggested alternative.
- No behavior change; `go build ./...`, `go vet ./...`,
  `go test ./... -race -cover`, and `gofmt -l .` all still pass.

### Concurrent pull-request-chain/timeline fetch (2026-07-19)

Revisits the `ExportService.Export` concurrency question deferred during
the export-service-orchestration review, on
`perf/concurrent-evidence-fetch`, now that Slice 5's CLI wiring is the
production caller that review's deferral was waiting on.

- **Scope confirmed with the user**: two-group concurrency, not full
  3-way — `FetchTimeline` runs in its own goroutine, concurrent with a
  second goroutine running the pull-request chain
  (`FetchPullRequest` then, only on success, `FetchReviewComments`). Full
  3-way parallelization (each of the three fetches independent) was
  considered and rejected: it would call `FetchReviewComments` even when
  `FetchPullRequest` fails, contradicting the existing, deliberately
  tested short-circuit
  (`TestExportService_Export_PropagatesAnErrorWhenFetchPullRequestFails`
  asserts `FetchReviewComments` is not called in that case). The two-group
  split matches this file's own previously-recorded deferred-item wording
  ("`FetchPullRequest`+`FetchReviewComments`" as one group, "`FetchTimeline`"
  as the other) and requires zero changes to any existing test.
- `ExportService.fetchPullRequestChainAndTimeline` joins both goroutines via
  `sync.WaitGroup` before checking either branch's error, following
  `resolveAttachments`' existing goroutine-join pattern. Error-check order
  is fixed (pull-request chain first, then timeline), so a failure in both
  branches at once is still deterministic to report and to test.
- Tests: Red/Green TDD. `TestExportService_Export_FetchesThePullRequestChainAndTheTimelineConcurrently`
  proves genuine concurrency (not just "both eventually get called") via a
  barrier fake (`barrierEvidenceFetcher`) — both `FetchPullRequest` and
  `FetchTimeline` block on a shared channel until the test observes both
  have started; a sequential implementation deadlocks waiting for the
  second one to start before ever unblocking the first, so the test fails
  (times out) against the pre-change code and passes against the fix.
  Every existing test in this file needed no changes; all still pass with
  `go test ./... -race -cover`, run 20× under `-race` to check for
  flakiness (none observed).

C0 after this change: `internal/application/services` 96.9% (up from
96.2%). `go build ./...`, `go vet ./...`, `go test ./... -race -cover`,
and `gofmt -l .` all pass.

### Local review of the concurrent fetch (2026-07-19)

A local review found one correctness bug (confirmed) and one cleanup item
(plausible), both addressed on `perf/concurrent-evidence-fetch`:

- **`wg.Wait()` blocked on both branches even after one had already
  failed.** A fast-failing `FetchPullRequest` (e.g. a nonexistent or
  inaccessible PR) still waited for `FetchTimeline` to finish before
  `Export` could return an error — and if `FetchTimeline` was mid rate-limit
  backoff, that wait could be up to an hour (`X-RateLimit-Reset`, per
  `internal/infrastructure/github/retry.go`), with `cmd/gh-exhibit/main.go`
  calling `Export` with a plain `context.Background()` and no way to cancel
  it. The prior sequential implementation never had this problem — a
  `FetchPullRequest` failure returned before `FetchTimeline` was ever
  called. Verified against `retry.go`'s `realSleep`, which already
  `select`s on `ctx.Done()`, so a cancelled context interrupts an
  in-progress backoff wait almost immediately rather than needing its own
  new interruption mechanism.
- **Fix chosen over the reviewer's minimal suggestion, confirmed with the
  user**: both branches now share a `context.WithCancel`-derived context;
  whichever branch fails first calls `cancel()`, unblocking the sibling's
  in-flight fetch instead of waiting it out. A fixed check-order priority
  (as before, pull-request chain checked first) was considered and
  rejected: once cancellation is in play, the branch that gets cancelled
  returns its own `context.Canceled`, and a fixed priority would report
  that collateral error over the sibling's real one whenever the sibling —
  not the pull-request chain — was the branch that actually failed. A
  mutex-guarded "first genuine failure wins" result (`fail`, called by
  either goroutine, only the first call takes effect) replaces the fixed
  priority, so the branch that failed first (not the branch cancellation
  happened to reach second) is what `Export` reports.
- **Folded in a related cleanup finding, confirmed with the user**:
  `fetchPullRequestChainAndTimeline`'s three positional
  `(json.RawMessage, []json.RawMessage, []json.RawMessage, error)` returns
  gave the compiler nothing to catch a transposed `reviewComments`/
  `timeline` assignment (both share the `[]json.RawMessage` type) at
  either call site. Replaced with a named
  `fetchedPullRequestAndTimeline{pullRequest, reviewComments, timeline}`
  struct.
- Tests: Red/Green TDD.
  `TestExportService_Export_ReturnsPromptlyWhenThePullRequestChainFailsWhileTimelineIsStillFetching`
  reproduces the reviewer's exact scenario without a real rate-limit wait —
  a `blockingTimelineFetcher` fake blocks `FetchTimeline` on `<-ctx.Done()`
  (standing in for an arbitrarily long backoff sleep) while
  `FetchPullRequest` fails immediately; confirmed failing (timing out
  against a 1s budget) on the pre-fix code, since nothing cancelled the
  context the fake was blocked on, then passing once the fix cancels it.
  No existing test needed changes. All tests re-run 30× under `-race`
  (in addition to the existing suite) to check for flakiness in either
  concurrency-sensitive test; none observed.

C0 after this round: `internal/application/services` 97.2% (up from
96.9%). `go build ./...`, `go vet ./...`, `go test ./... -race -cover`,
and `gofmt -l .` all pass.

### Test-naming and error-message convention resolution (2026-07-19)

Resolves the deferred "reconcile `CLAUDE.md`'s Evergreen Tests wording"
design item above, on `refactor/test-and-error-conventions`, following a
critical review of the project's actual test-naming and error-message
practice (not a fresh local-review workflow run — a direct discussion
with the user).

- **The resolution is sharper than either option this file's Design
  section originally posed** ("rename all existing tests" vs. "clarify
  `CLAUDE.md` to permit naming the unit under test generally"). Neither
  was quite right: a test's `<UnitName>` prefix must name an **exported**
  unit specifically, not any unit. An unexported function/method's
  correctness should be demonstrated through the exported entry point
  that calls it, not by naming the unexported unit directly — direct
  tests of an unexported unit are permitted only as an explicitly
  documented exception (a comment stating why the exported entry point
  cannot reach the behavior being verified), not silently.
- **Audit found 31 tests across 8 unexported units** naming an
  unexported function/method directly: `classify` (9), `nextPageURL`/
  `splitLinkHeader` (6), `doWithRetry` and its retry-decision helpers
  (5), `actorWire.resolvedLogin`/`reviewCommentWire.resolvedLine` (5),
  `writeMetaLine`/`newAttributionMeta` (3), and `equalPointers` (3).
  Each was individually classified rather than mechanically renamed:
  - `classify_test.go` deleted; all 9 scenarios reproduced as
    `TestBuildEntries_...` tests in `join_test.go` (`classify` is
    `BuildEntries`'s only caller, so every scenario translates directly:
    `rawTimeline` in, `entries`/`skipped` out).
  - `wire_test.go` deleted outright: all 5 scenarios (present login/line,
    ghost fallback, outdated-line fallback, file-level nil line) were
    already exercised transitively by `BuildEntries`-level tests (some
    pre-existing, one added by the `classify` migration above); coverage
    for both methods stayed at 100% with no replacement test needed.
  - `pagination_test.go` deleted; its 6 Link-header edge cases (absent
    header, multiple rels, last page, malformed header, an unescaped
    comma inside a next URL's query) reproduced as `httptest.Server`
    responses driven through `FetchTimeline` in
    `evidence_fetcher_test.go`. `nextPageURL`/`splitLinkHeader` stayed at
    100% coverage.
  - `retry_test.go` deleted; 3 of its 5 scenarios (403 rate-limited
    retry, 403 permission-denied no-retry, exhausted attempts) reproduced
    against a real `httptest.Server`. The remaining 2 (the exact wait
    duration parsed from `Retry-After`; a network-level error) still
    need a fake `requester` and sleep spy to avoid a real multi-second
    sleep or a broken connection — those fixtures moved into
    `evidence_fetcher_test.go` and drive `FetchIssue` via
    `&evidenceFetcher{...}` directly, the same construction-injection
    pattern this file already used for
    `TestFetchTimeline_StopsFollowingAnUnboundedLinkHeaderChain`.
  - `render_test.go`: `TestNewAttributionMeta_...` and
    `TestWriteMetaLine_WritesAnAnchoredMetaLineFollowedByTheBody` deleted
    as redundant with every Tier 1 type's own `Render()` test, which
    already asserts the identical `meta:{...}` line.
    `TestWriteMetaLine_WrapsAMetaMarshalFailure` kept as the one
    documented exception in the file: no Tier 1 type's `Render()` can
    ever pass `writeMetaLine` a value that fails to marshal, so this
    branch is unreachable from any exported entry point.
  - `pointer_test.go` deleted. Two of `equalPointers`'s three branches
    (one-nil-one-non-nil, both-non-nil-delegating-to-`eq`) were already
    covered by `Body`/`InlineContext`'s own `Equals` tests, but auditing
    found **neither type had a "both nil" `Equals` case** — a real,
    previously-unnoticed gap, not just a redundancy. Added
    `TestBody_Equals_TreatsTwoOpenBodiesAsEqual` to close it;
    `equalPointers` stayed at 100% coverage.
  - Every deletion was verified via `go tool cover -func` on the
    specific unexported function/method before removing its direct
    test, not assumed from reading the code.
- **Error messages, a related but distinct finding from the same
  discussion**: audited every `fmt.Errorf`/`errors.New` call site
  project-wide. Reworded 22 whose operation phrase was a near-literal,
  lowercased rendition of the specific exported function/method it
  wrapped (`"build body"` ~ `BuildBody`, `"write issue"` ~ `WriteIssue`,
  `"fetch timeline"` ~ `FetchTimeline`, `"resolve current repository"` ~
  `ResolveRepo`, and their siblings across `export_service.go`,
  `evidence_writer.go`, `registry.go`, `repo.go`) to describe the
  concrete operation/state instead, so a future rename of the function
  no longer obligates a matching error-string edit. Messages already
  describing a concrete operation not tied to one of this project's own
  identifiers (`"unmarshal issue resource"`, `"create directory for
  %s"`, messages naming a third-party `go-gh` constructor like
  `api.NewRESTClient`) were left as-is — verified against every existing
  test asserting on an error-message substring (`strings.Contains`) that
  none broke.
- `AGENTS.md`'s Evergreen Tests section rewritten to state both rules
  explicitly, replacing the ambiguous "must not reference implementation
  details" wording that admitted either reading.
- No coverage regression in any package: `internal/domain/services`
  97.2%, `internal/domain/valueobjects` 94.9%, `internal/infrastructure/
  github` 89.0% (up from 88.1%, since the `doWithRetry` migration
  happened to exercise a couple of previously-untested branches
  incidentally), `internal/infrastructure/persistence` 96.2%,
  `internal/application/services` 97.2%, `internal/presentation/cli`
  92.1% — all unchanged or improved from their pre-branch figures. `go
  build ./...`, `go vet ./...`, `go test ./... -race -cover`, and
  `gofmt -l .` all pass after every commit in this branch.

### Distribution scaffolding (2026-07-19)

Slice 5's last remaining item is done, on `feat/gh-extension-distribution`,
closing out Slice 5 entirely (only the design-section case-normalization
item remains open project-wide, itself deliberately deferred until a
concrete consumer exists).

- Verified what `gh extension create --precompiled=go` actually
  generates (run against a real, disposable directory rather than
  assumed from memory) before adapting it to this already-existing
  codebase: a `.github/workflows/release.yml` triggered on a `v*` tag
  push, running `cli/gh-extension-precompile@v2`, plus a `.gitignore`
  for the compiled binary.
- **Adapted for this project's layout, confirmed against
  `cli/gh-extension-precompile`'s own README**: the generated scaffold
  assumes `main.go` at the repository root, but this project's main
  package is `cmd/gh-exhibit/main.go` (Slice 5's own composition-root
  choice). Added `go_build_options: "./cmd/gh-exhibit"` to point the
  action's `go build` invocation at the right package instead of the
  repository root.
- `.gitignore` added (the project had none at all): ignores the
  locally built `/gh-exhibit` binary `go build ./cmd/gh-exhibit`
  produces at the repository root.
- **Confirmed with the user**: no release tag pushed this round to
  trigger a real end-to-end run of the workflow — deferred to whenever
  the user is ready to publish a real, user-facing release, rather than
  spending one on a throwaway verification tag.
- **README scope, confirmed with the user** after a round of
  discussion: the user initially reasoned that installation
  instructions were unnecessary because "distribution is something I
  do myself," while usage instructions were still needed. That
  conflated two different actors — the maintainer who pushes release
  tags, and a third-party reader trying the extension for the first
  time from a public repository (per `CLAUDE.md`'s "this project may be
  released publicly") — and usage instructions presuppose the reader
  already has the binary installed. Resolved as: both an
  `gh extension install connect0459/gh-exhibit` line and a minimal
  usage section (flags/positional argument, matching
  `internal/presentation/cli.ParseArgs`'s actual shape, plus two
  examples) were added to `README.md`, which previously had no content
  beyond its own title.
- No test file for any of this — none of it is Go logic; verified
  instead via `pre-commit run` (`check-yaml`, `markdownlint-cli2`,
  `end-of-file-fixer`, etc.), which passed on every changed file.

### Release workflow hardening (2026-07-19)

A follow-up to the distribution scaffolding above, on the same
`feat/gh-extension-distribution` branch, referencing
`connect0459/starlark-mbt`'s own `publish.yml` for this project's release
automation:

- `actions/checkout` and `cli/gh-extension-precompile` are both pinned to
  a full commit SHA (looked up against each action's own latest release
  tag), with the human-readable version kept in a trailing comment,
  rather than a floating major-version tag (`@v6`/`@v2`).
- `actions/checkout` now sets `persist-credentials: false` —
  `gh-extension-precompile` authenticates its own release/asset API calls
  via `github.token`, not checkout's persisted git credentials, so
  nothing in this workflow depends on them.
- Added `run-name: Release ${{ github.ref_name }}` so a workflow run is
  identifiable by its triggering tag from the Actions list, instead of
  every run sharing the same generic workflow name.
- No test file for any of this, same as the distribution scaffolding
  entry above; verified by inspecting the resulting `release.yml` diff
  directly (no `pre-commit` hook covers workflow-file semantics beyond
  `check-yaml`'s syntax check, which already passed).

### Case-insensitive identifier equality (2026-07-19)

Resolves the deferred case-normalization design item above, on
`fix/case-insensitive-identifier-equals`, before its trigger condition
(a concrete `Equals`-based consumer) actually existed — **confirmed with
the user** as an explicit, low-risk exception to that deferral, weighed
against YAGNI: the fix is two one-line changes, and revisiting it later
would otherwise mean re-deriving the same design discussion once a
consumer finally appears.

- `Attribution.Equals` and `IssueRef.Equals` now compare `author` and
  `owner`/`repo` respectively via `strings.EqualFold`, instead of `==`,
  so two differently-cased values naming the same GitHub login or
  repository are treated as equal. Applied to both types in the same
  pass, per this project's pattern-generalization rule.
- Stored values are untouched — no construction-time lowercasing.
  `Attribution.author` must stay verbatim for `entry/render.go`'s
  `meta:{...}` line (ADR-001's verbatim-evidence guarantee);
  `IssueRef.Owner()`/`Repo()` feed both the REST API request path and
  the on-disk evidence path, which must likewise preserve whatever case
  the caller provided. `url` and `number` are unaffected — the case
  question is specific to GitHub's login/repository-name matching rule,
  not to URLs or numeric identifiers.
- Tests: Red/Green TDD, one new case per type
  (`TestAttribution_Equals_TreatsDifferentlyCasedAuthorsAsEqual`,
  `TestIssueRef_Equals_TreatsDifferentlyCasedOwnerAndRepoAsEqual`),
  confirmed failing against the pre-fix `==` comparison before
  implementing.

C0 unchanged: `internal/domain/valueobjects` 94.9%. `go build ./...`,
`go vet ./...`, `go test ./... -race -cover`, and `gofmt -l .` all pass.

### Local review of the case-insensitive identifier equality fix (2026-07-19)

A local review (high effort, verified) of the above found one confirmed
correctness issue, addressed on the same
`fix/case-insensitive-identifier-equals` branch; one candidate finding
(`url`'s byte-exact comparison) was checked and rejected as working as
designed.

- **`strings.EqualFold`'s Unicode case folding is broader than
  ASCII-only case-insensitivity, and `Attribution.author` had no
  character-set constraint to guard against it** — unlike `IssueRef`'s
  `owner`/`repo`, already locked to GitHub's ASCII username/repo-name
  patterns by `validateOwner`/`validateRepoName`. A confusable non-ASCII
  character can fold to an ASCII letter under Go's simple case-folding
  table (verified directly: `strings.EqualFold("K", "k")` — U+212A
  KELVIN SIGN — returns `true`), so a crafted or coincidental author
  string could collide with a distinct real GitHub login in
  `Attribution.Equals`.
- **The reviewer's own suggested remedy (reuse `IssueRef.owner`'s
  strict GitHub-username pattern for `author`) was considered and
  rejected**: every production call site constructs `author` via
  `actorWire.resolvedLogin()`, which returns either GitHub's real
  `user.login` or the `"ghost"` sentinel — and GitHub bot accounts'
  logins (`dependabot[bot]`, `github-actions[bot]`, etc.) are real,
  common values in that same field, containing brackets that
  `IssueRef`'s pattern would reject. Applying `IssueRef`'s pattern
  as-is would have traded an unobserved theoretical collision for an
  observed, guaranteed loss of bot-authored evidence, contradicting
  ADR-001's audit-trail guarantee.
- **Fix**: `NewAttribution` now rejects any `author` containing a
  non-ASCII byte, a narrower constraint than `IssueRef`'s pattern —
  it blocks exactly the case-folding collision mechanism (which
  requires a non-ASCII rune) without rejecting bracket-bearing bot
  logins, which are all-ASCII. `IssueRef.owner`/`repo` are unchanged:
  a different domain (repository-addressing accounts, never bot
  identities) where the existing stricter pattern remains correct.
- **Checked and rejected as working as designed**: leaving `url` as a
  byte-exact comparison in `Attribution.Equals` — a URL differing only
  in letter case is not guaranteed to name the same resource, unlike a
  GitHub login or repository name, so no case-insensitive treatment
  applies there.
- Tests: Red/Green TDD —
  `TestNewAttribution_RejectsAuthorContainingANonASCIICharacter` and
  `TestNewAttribution_RejectsAuthorContainingAUnicodeConfusableCharacter`
  (the exact U+212A KELVIN SIGN scenario) confirmed failing before the
  fix; `TestNewAttribution_AcceptsAuthorContainingBotAccountBrackets`
  guards against a future regression that reintroduces `IssueRef`'s
  stricter pattern here instead.

C0 after this round: `internal/domain/valueobjects` 95.1% (up from
94.9%; a later `go test ./... -cover` run reports 95.4% for this
package, since subsequent unrelated test additions incidentally
exercised a few more branches — no entry below this one updates the
figure). `go build ./...`, `go vet ./...`, `go test ./... -race -cover`,
and `gofmt -l .` all pass.

### Repository migration and commit history reconstruction

2026-07-19T18:00:00+09:00

Implementation fixtures, `todo.md`, and similar files had inadvertently
exposed information related to `connect0459`, so the GitHub repository was
replaced with a new one. The affected parts were removed and the history
was recommitted from scratch, so the decisions recorded above are no
longer traceable at the commit-log level, but the final artifacts they
produced were carried over essentially unchanged.

### Cross-cutting local review after the migration (2026-07-19)

With every todo.md checklist item already marked done, a fresh review was
run across the whole repository (not scoped to one branch's diff, unlike
every prior review round above) — six independent lenses (correctness,
onion-architecture boundaries, test quality, comment/error-message
conventions, security, and doc-accuracy against this file's own claims),
each verified adversarially before being accepted. Correctness,
architecture, security, and doc-accuracy raised nothing; two rounds (one
per lens below) found real issues, all fixed on `test/review-followup-fixes`:

- **Stale package-name error-message prefixes, left behind by the
  2026-07-19 domain-layer reorg** (see "Domain layer package
  reorganization" above): the reorg's own follow-up review had already
  caught one instance of this class (the `"timeline."` → `"services."`
  substitution corrupting the `"timeline.json"` filename literal), but
  that check only covered the dotted-qualifier pattern, not the
  colon-prefixed error/skip-reason pattern (`"entry: ..."`,
  `"repositories: ..."`, `"timeline: ..."`), so 33 call sites across both
  reorganized packages kept naming a package that no longer exists.
  `internal/domain/valueobjects` (`attribution.go`, `document.go`,
  `inline_context.go`, `review_state.go`, `render.go` — 10 sites carrying
  `"entry:"`; `issue_ref.go` — 8 sites carrying `"repositories:"`, stale
  since before `IssueRef` moved here from `domain/repositories`) now use
  `"valueobjects:"`, matching every infrastructure-layer package's own
  convention (`github:`, `persistence:`, `cli:`). No test asserted on any
  of the old prefix strings, so this was a safe mechanical change;
  confirmed via `grep` before editing. `internal/domain/services`
  (`classify.go`, `body.go`, `join.go` — 15 sites carrying `"timeline:"`)
  is a separate case: see the follow-up review below, which found the
  first-pass `"timeline:"` → `"services:"` rename here introduced a new
  problem rather than fixing the old one.
- **Two test-coverage gaps**, both closed with Red/Green TDD: (1)
  `internal/infrastructure/persistence`'s `WritePullRequest`/
  `WriteReviewComments` had no test for the `ctx.Err()` cancellation guard,
  unlike their sibling `WriteIssue`/`WriteTimeline` (2026-07-18's "ctx was
  discarded by all four EvidenceWriter methods" fix, only tested for half
  of the four methods it was added to); (2) `internal/domain/services`'s
  `classifyCommentedEvent`, `classifyReviewedEvent`, and
  `buildReviewComment` each had no test for the branch where
  `valueobjects.NewAttribution` fails (e.g. an empty `html_url`), even
  though every sibling failure branch in the same three functions (bad
  state, missing path, malformed JSON) was already tested.

C0 after this round: `internal/infrastructure/persistence` 100% (up from
96.2%), `internal/domain/services` 99.3% (up from 97.2%). `go build ./...`,
`go vet ./...`, `go test ./... -race -cover`, and `gofmt -l .` all pass.

### Human review of the "services:" prefix rename (2026-07-19)

A human review (not the workflow above) of the branch this file's
previous entry describes found that the `"timeline:"` → `"services:"`
rename in `internal/domain/services` had itself introduced a new defect,
still on `test/review-followup-fixes`:

- **`"services:"` collides with `internal/application/services`, which
  already used that exact tag.** `internal/` has exactly one duplicated
  directory basename, `services` (`internal/domain/services` and
  `internal/application/services`); every other prefix (`valueobjects:`,
  `github:`, `persistence:`, `cli:`) names a directory unique across the
  repo. `export_service.go`'s `BuildBody`
  call site re-wraps the domain-layer error with its own `"services: ...`
  message, so an attribution failure produced a doubled
  `"services: could not derive a title and body from the issue/PR
  resource: services: issue resource attribution: ..."` string — the
  exact non-uniqueness the rename was meant to fix, now reintroduced
  between these two packages specifically.
- **Reconsidered the prefix scheme's actual audience, not just its
  collision.** `internal/presentation/cli/run.go`'s `RunExports` prints
  an `Export` failure straight to `stderr` via `%v` — this prefix is not
  an internal debugging aid a maintainer sees only with the source open;
  it is literally what `gh-exhibit`'s end user reads on a failed export.
  A Go package or onion-architecture-layer name means nothing to that
  reader (they have no notion of "domain" vs "application"), so a
  layer-qualifying fix (`"domain/services:"` / `"application/services:"`)
  was considered and rejected: it resolves the string collision but adds
  more meaningless jargon to a user-facing message rather than less.
- **Resolved by dropping `internal/domain/services`'s package tag
  entirely, not by qualifying it.** Every domain-layer error in this
  package is always re-wrapped by `internal/application/services` before
  it reaches the CLI's output, so the inner tag was pure redundancy on
  top of the operation-describing text it prefixed (which already reads
  as a complete, specific explanation without it — e.g. `"issue resource
  attribution: attribution author must not be empty"`). `SkipNote.Reason`
  values (the other half of the 15 sites) currently reach no output at
  all (`RunExports` only reports a skip *count*, not each reason), so the
  same argument applies there even more directly. `internal/application/
  services` keeps its own existing `"services:"` tag unchanged — it was
  never part of this collision and sits at the one place (closest to the
  CLI boundary) where a package tag might still carry marginal value.
- This does not reopen the doc-accuracy lens's earlier verdict (no
  findings): that lens checked specific behavioral claims (concurrency,
  retry, coverage figures), not this file's own just-written description
  of the rename, which this entry corrects.

No test asserted on the `"services:"` tag itself (only on operation words
like `"unmarshal"`/`"attribution"`, which are unchanged), so removing it
required no test changes. C0 unchanged: `internal/domain/services` 99.3%.
`go build ./...`, `go vet ./...`, `go test ./... -race -cover`, and
`gofmt -l .` all pass.

### Removal of every remaining package-name error-message tag (2026-07-19)

A follow-up question from the user ("doesn't `valueobjects:` have the
same problem?") after the entry above led to auditing every remaining
package-name prefix in the repo, not just the one this file already
fixed, still on `test/review-followup-fixes`:

- **`valueobjects:` has the identical defect** the prior entry just
  reasoned through for `internal/domain/services`: its errors are always
  either re-wrapped by a caller before reaching the CLI's output, or (for
  `NewIssueRef`, called directly from `internal/presentation/cli`)
  printed completely unwrapped — in neither case does a Go package name
  mean anything to the reader.
- **Traced every remaining prefix to its actual endpoint** rather than
  assuming: `cmd/gh-exhibit/main.go` prints `cli:`- and `registry:`-
  prefixed errors directly via `fmt.Fprintln(os.Stderr, err)`, and
  `internal/presentation/cli/run.go`'s `RunExports` prints everything
  `ExportService.Export` returns (including `application/services`'s own
  `"services:"` tag and anything it wraps — `github:`, `persistence:`)
  via `%v`. Every single package-name prefix in the repository is
  user-facing; none is an internal-only debugging aid.
- **Reconsidered and reversed this file's own prior claim** that
  `application/services`'s tag "sits at the one place where a package
  tag might still carry marginal value": that tag is applied uniformly
  to every `Export` failure regardless of which subsystem actually broke
  (GitHub fetch, local write, or domain validation alike), so it carries
  no discriminating information even for a maintainer — the claim did
  not hold up under the same scrutiny already applied to
  `domain/services`.
- **`github:` and `persistence:` were considered as possible exceptions**
  (they name real, external I/O boundaries — network vs. local disk —
  which are at least a distinction a user could plausibly act on,
  unlike an internal Go package name), but the user preferred full
  consistency: if a message can be made self-descriptive without the
  tag, drop the tag there too rather than carve out exceptions. Checked
  each of the 17 sites individually rather than blanket-stripping:
  `internal/infrastructure/persistence`'s 6 messages already interpolate
  a local filesystem path (unambiguous without a tag) and 2 of them
  (`joinRawArray`'s) were not I/O failures to begin with, so the tag was
  actively mischaracterizing them; `internal/infrastructure/github`'s
  `attachment_fetcher.go` messages interpolate a full attachment URL
  (already shows a `github.com` host), so those needed only the prefix
  dropped, but `evidence_fetcher.go`'s messages interpolate a bare REST
  API path with no host/scheme (e.g. `repos/owner/repo/issues/42`) and
  its two client-construction messages interpolate no identifier at all
  — both reworded to name "GitHub" directly in their own operation text
  instead of via a tag, so no information was lost by dropping the
  prefix.
- End state: zero package- or layer-name error-message prefixes remain
  anywhere in the repository. Every message is self-descriptive English
  text, verified by an exhaustive repo-wide grep for the `"word: "`
  pattern turning up nothing outside `_test.go` files.

No test asserted on any of the removed tags; only on operation words
(`"unmarshal"`, `"attribution"`, etc.), all unchanged, so no test needed
updating. C0 unchanged across every touched package. `go build ./...`,
`go vet ./...`, `go test ./... -race -cover`, and `gofmt -l .` all pass.

### Comment cleanup: Godoc coverage and thin-comment enhancements (2026-07-19)

The user raised two concerns about the codebase's current comment state:
few Godoc comments document parameters/return values, and a handful of
comments look like they might just restate obvious WHAT. Addressed on
`docs/comment-cleanup`, one package per commit (confirmed with the user
before starting):

- **A full survey preceded any edit** (workflow effort=high Explore
  agent, not a fresh local-review round): every non-test `.go` file under
  `internal/` and `cmd/` was inventoried for exported members lacking
  Godoc, existing Godoc quality, and any comment resembling pure
  WHAT-restatement. **Finding: this codebase's existing comment culture
  already skews WHY** — no egregious pure-WHAT violation was found
  anywhere; every WHY/WHY-NOT comment cited in prior review rounds
  (`isASCII`'s Unicode case-folding rationale, `joinRawArray`'s
  verbatim-bytes rationale, the concurrent-fetch cancellation-priority
  comments, etc.) was left untouched. The real gap was the first
  concern: most exported constructors/getters/`Equals`/`Render` methods
  in `internal/domain/valueobjects`, and the `Fetch*`/`Write*`
  implementation methods in `internal/infrastructure/{github,
  persistence}`, had no Godoc at all.
- **Scope confirmed with the user**: also add a package-level doc
  comment to the three packages that had none
  (`internal/domain/repositories`, `internal/infrastructure/persistence`,
  `internal/application/services`) — an extension beyond the user's
  literal ask, since consistency with every sibling package (which all
  already had one) was judged worth the small addition.
- `internal/domain/valueobjects`: every previously-undocumented exported
  constructor, getter, `Equals`, and `Render` across all nine files
  gained a one-line Godoc, documenting nil-handling and case-insensitive-
  comparison semantics where non-obvious (e.g. `Attribution.Equals`,
  `IssueRef.Equals`).
- `internal/domain/repositories`: added the package doc comment plus a
  Godoc on every interface method that had none (`Fetch`, `WriteAsset`,
  `WriteDocument`, all four `EvidenceFetcher`/`EvidenceWriter` methods).
  `WriteAsset`'s existing comment was enhanced to name its `data`
  parameter explicitly.
- `internal/infrastructure/github` /
  `internal/infrastructure/persistence`: each `Fetch*`/`Write*`
  implementation method gained a one-line Godoc pointing back to the
  `repositories` interface it implements, whose own doc comment already
  covers behavior in full — matching this codebase's existing pattern of
  not duplicating the same explanation at both the interface and its
  implementation.
- `internal/application/services`: added the package doc comment,
  including a note distinguishing it from the identically-named
  `internal/domain/services` package (two Go packages sharing a base
  name, at different import paths — already a source of confusion once
  before, during the error-message-prefix rounds above).
- `internal/registry`: `Config.Host`/`Config.OutputDir`'s field comments
  previously just restated the field name; both now name which of
  `NewExportService`'s constructed collaborators actually consumes them.
- **Considered and left as-is**: `internal/presentation/cli/args.go`'s
  `Args` type comment and `internal/domain/valueobjects/attribution.go`/
  `issue_comment.go`'s type comments were flagged by the survey as
  borderline-thin, but each already adds concrete, non-obvious context
  (the `meta:{...}` line format, the source timeline event, or — for
  `Args` — that the shape is validated, with the real substance carried
  by its own field-level comments) rather than merely restating the
  identifier, so rewriting them would have been enhancement for its own
  sake, not a fix for a real violation.
- No behavior change anywhere — comment-only across all seven commits.
  C0 unchanged in every package touched:
  `internal/domain/valueobjects` 95.4%, `internal/infrastructure/github`
  91.7%, `internal/infrastructure/persistence` 100%,
  `internal/application/services` 97.2%, `internal/presentation/cli`
  98.8%. `go build ./...`, `go vet ./...`, `go test ./... -race -cover`,
  and `gofmt -l .` all pass after every commit.

### Local review of the comment cleanup (2026-07-19)

A local review of the above found one confirmed correctness issue,
addressed on the same `docs/comment-cleanup` branch; every other Godoc
checked (`Attribution.Equals`'s case-insensitive comparison, `IssueRef`'s
validation rules, `registry.Config`'s field descriptions) was verified
accurate against its actual implementation.

- **`DocumentWriter.WriteDocument`'s Godoc was not a grammatical
  sentence.** It read "WriteDocument persists rendered, ref's fully
  rendered Markdown document." — a leftover mid-edit artifact (the
  `rendered` parameter name spliced in without finishing the sentence
  around it), unlike every other Godoc added in this pass. Reworded to
  "WriteDocument persists ref's fully rendered Markdown document.",
  matching the sibling `EvidenceWriter` methods' own "persists ref's ...
  resource" phrasing rather than naming the parameter explicitly.

No behavior change; `go build ./...`, `go vet ./...`,
`go test ./... -race -cover`, and `gofmt -l .` all pass.

### CI workflow (2026-07-19)

`.github/workflows/ci.yml` existed on disk but was untracked — a verbatim
copy of `starlark-mbt`'s MoonBit workflow (`moon fmt`/`moon check`/`moon
test`), never wired to this project's own Go toolchain. Replaced with a
Go-native workflow, on `ci/add-go-workflow` (a tooling-only change, not a
todo.md checklist item):

- One sequential job (`gofmt -l .` → `go vet ./...` → `golangci-lint` →
  `go build ./...` → `go test ./... -race -cover`), matching this
  project's own verification convention already recorded throughout this
  file. `golangci-lint-action` is pinned to `v2.12.2`, the same version
  `.pre-commit-config.yaml` already pins locally, so CI and the local
  pre-commit hook enforce identical lint behavior.
- `actions/checkout`/`actions/setup-go`/`golangci-lint-action` are pinned
  to a full commit SHA with a trailing version comment, matching
  `release.yml`'s existing pinning convention;
  `actions/checkout`'s SHA is the same one `release.yml` already uses.
- **Considered and rejected**: an initial draft split `lint`/`test` into
  two parallel jobs (mirroring `starlark-mbt`'s own two-job split,
  itself driven by MoonBit's four build targets — a condition Go's
  single-target build doesn't share). **Confirmed with the user**: two
  jobs duplicate the checkout/setup-go steps for a parallelism gain this
  project's small package count (7) doesn't meaningfully benefit from;
  consolidated into one job instead, matching this user's own
  `edit-pr-duration` Go project's existing CI structure.
- **Also considered**: GitHub Actions' newly-introduced step-level
  `background`/`wait`/`wait-all`/`parallel` keywords (announced
  2026-06-25), which could parallelize steps within a single job without
  duplicating checkout/setup-go. Rejected for now: the feature's exact
  syntax is not yet reflected in GitHub's own workflow-syntax reference
  documentation as of this writing, so adopting it here would mean
  committing to unconfirmed syntax for a three-week-old feature, not a
  verified one.
- `dorny/paths-filter`, pinned the same way, gates every check-running
  step behind whether the push/PR actually touched `**/*.go`, `go.mod`,
  `go.sum`, `.golangci.yml`, or the workflow file itself — matching
  `edit-pr-duration`'s own precedent for the same reason (a docs-only or
  `todo.md`-only change shouldn't pay for a Go build/test run).
  `actions/checkout` sets `fetch-depth: 0`, also matching that precedent,
  since `paths-filter` diffing a `push` event against its previous ref
  needs history a shallow (default) clone doesn't have.
- Every `run:` step sets `shell: bash` explicitly, rather than relying on
  the Linux runner's default shell.
- **Flagged by the user, fixed before opening the PR**: an initial draft
  gave every step an explicit `name:`, including `uses:` steps whose own
  `action.yml` already declares a self-descriptive default name
  (`actions/checkout`, `actions/setup-go`, `golangci-lint-action`, and
  `dorny/paths-filter` each surface their own — "Checkout", "Setup Go
  environment", and "Golangci-lint" confirmed directly from the first
  three actions' own `action.yml`) — restating that default at the call
  site is the workflow-YAML equivalent of a code comment restating
  obvious WHAT. Decided policy: a step-level `name:` is omitted wherever
  the action's own default name already covers it (every `uses:` step in
  this workflow), and given explicitly wherever this workflow supplies
  its own command (every `run:` step: "Check formatting", "Run go vet",
  "Build", "Test", "Skip CI checks (no relevant changes)") — a
  self-authored command has no built-in title to fall back on, unlike an
  action invocation.
- No test file for any of this — same as `release.yml`'s own precedent;
  verified via `pre-commit run --files .github/workflows/ci.yml`
  (`check-yaml` and the rest all pass).

### Community Standards and CHANGELOG (2026-07-19)

Ahead of the first tagged release, GitHub's Community Standards checklist
(`community_profile` API, `health_percentage` 28% beforehand) flagged four
missing files; a fifth (`CHANGELOG.md`) is not on that checklist itself but
is this author's own standing convention (present in both
`connect0459/starlark-mbt` and `connect0459/rustgression`). All five added
on `docs/community-health-files` (a tooling/docs change, not a `todo.md`
checklist item, same precedent as the CI workflow entry above):

- `LICENSE` — MIT, matching `connect0459/rustgression`'s choice (confirmed
  with the user over `connect0459/starlark-mbt`'s Apache-2.0).
- `CODE_OF_CONDUCT.md` — Contributor Covenant v2.1, reused verbatim from
  `connect0459/starlark-mbt`/`connect0459/rustgression` (identical between
  both), so contributor-facing policy is consistent across this author's
  public projects. Enforcement contact confirmed with the user as
  `connect0459@gmail.com`, matching those same two projects.
- `CONTRIBUTING.md` — written specific to this project's actual toolchain
  (`just`, `pre-commit`, `golangci-lint`) rather than adapted from either
  reference repo's language-specific instructions (MoonBit/`moon`,
  Rust/`cargo`+Python/`uv`). Documents the dev-workflow command sequence
  already fixed by `ci/add-go-workflow`'s CI job, the Red/Green TDD and
  per-layer coverage conventions already recorded throughout this file and
  `AGENTS.md`, and the commit/branch/PR conventions from `AGENTS.md`
  verbatim so a first-time external contributor doesn't need to
  reverse-engineer them from commit history.
- `SECURITY.md` — scoped to this project's actual attack surface rather
  than either reference repo's boilerplate scope: resource exhaustion
  (`fix/attachment-fetch-size-limit`'s risk), output-path traversal
  (mitigated by `IssueRef`'s `validateOwner`/`validateRepoName`, introduced
  in the initial domain-layer implementation commit `5466a48`, not a
  dedicated fix PR), and credential exposure via attachment-URL host
  spoofing. **Corrected**: an earlier draft of this entry attributed the
  credential-exposure mitigation to `fix/attachment-host-detection` — that
  PR's actual purpose, confirmed against its own commit `4d22181`, was
  GitHub Enterprise Server host support (`services.Detect` hardcoded
  `github.com`, silently skipping every attachment on a GHES repository);
  the host-scoped attachment-URL regex it's built on (`urlPattern`) has
  matched only the configured host, with the literal `.` already escaped,
  since the initial domain-layer implementation commit `5466a48` — no PR
  closed a credential-exposure hole, because the host-scoping was never
  open to begin with. Reporting channel: GitHub
  private vulnerability reporting or `connect0459@gmail.com`, both
  confirmed with the user.
- `CHANGELOG.md` — follows the `Keep a Changelog` format and Semantic
  Versioning, reusing both reference repos' maintenance-instructions
  header comment verbatim (repo name substituted). **Pre-populated for the
  `v0.1.0` first release, not left under `[Unreleased]`**, matching
  `connect0459/starlark-mbt`'s own release convention (e.g. its PR #390):
  the version section is written and merged to `main` before the
  corresponding tag exists, so `gh release create`
  has a matching entry the moment the tag is cut, rather than the
  CHANGELOG trailing the release via a separate follow-up PR. Every
  capability built across Slices 1-5 is recorded under
  `## [0.1.0] - 2026-07-19`, with `[Unreleased]` left empty above it; the
  footer links `[Unreleased]` as `compare/v0.1.0...HEAD` and `[0.1.0]` as
  `releases/tag/v0.1.0`.
- No test file for any of this — none of it is Go logic; verified via
  `pre-commit run --files LICENSE CODE_OF_CONDUCT.md CONTRIBUTING.md
  SECURITY.md CHANGELOG.md` (`markdownlint-cli2` and the rest all pass).

### --version flag and GoReleaser migration (2026-07-19)

The user asked whether a version-embedding step was missing, comparing
against `connect0459/starlark-mbt`'s own `moon.mod` version bump. `go.mod`
has no equivalent field — Go module versions are resolved from VCS tags,
not a manifest field, so there is no direct analog. But the underlying
question surfaced a real, separate gap: `gh-exhibit` had no `--version` at
all, even though `SECURITY.md` and `BUG_REPORT.md` already ask a reporter
for "the version of `gh-exhibit`" with no way to determine it. Addressed on
`chore/migrate-to-goreleaser` (Red/Green TDD for the CLI change; the
tooling change has no test file, same precedent as `release.yml` itself):

- `internal/presentation/cli.Args` gains a `Version bool` field;
  `ParseArgs` returns `Args{Version: true}` immediately after a successful
  `--version` parse, before the "exactly one positional argument" check —
  the same short-circuit precedent `--help`/`flag.ErrHelp` already
  established, confirmed by a new test asserting `--version` needs no
  positional number even combined with other flags.
- `cmd/gh-exhibit/main.go` gains package-level `version`/`commit`/`date`
  vars (default `"dev"`/`"none"`/`"unknown"`), printed and exited 0 when
  `args.Version` is set, before `ResolveRepo` runs.
- **Investigated before committing to an injection mechanism**: real-world
  precedent was surveyed rather than assumed. `vilmibm/gh-screensaver`
  (plain `cli/gh-extension-precompile`, no version string at all) and
  `dlvhdr/gh-dash`/`k1LoW/gh-grep` (both migrated to GoReleaser
  specifically for `ldflags`-injected version strings) were checked
  directly. The middle option (keep `cli/gh-extension-precompile`, inject
  `-ldflags` via its `go_build_options` input) was verified **not to
  work**: that action's `build_and_release.sh` passes `go_build_options`
  as a single quoted shell argument and already hardcodes its own
  `-ldflags="-s -w"`, so a second `-ldflags` value has no way in. Confirmed
  with the user: migrate to GoReleaser fully rather than hand-roll a
  `build_script_override`.
- **Migration verified end-to-end, not just configured**: `.goreleaser.yml`
  was validated with `goreleaser check` (caught and fixed one deprecated
  key, `archives.format` → `archives.formats: [binary]`) and exercised with
  `goreleaser release --snapshot --clean`, inspecting the actual output
  rather than assuming the config was correct:
  - **Release-asset naming was checked against `gh`'s own source**
    (`cli/cli`'s `pkg/cmd/extension/manager.go`: `strings.HasSuffix(a.Name,
    platform+ext)` where `platform` is `"{os}-{arch}"`) — confirmed
    `k1LoW/gh-grep`'s actual published assets use underscores
    (`gh-grep_v1.2.5_darwin_amd64`), which would **not** satisfy this
    match; `archives.name_template: "gh-exhibit-{{ .Os }}-{{ .Arch }}"`
    (hyphenated, `format: binary` so no archive extension is appended) was
    chosen specifically to avoid the same mistake. A snapshot release's
    `checksums.txt` was inspected directly and lists exactly
    `gh-exhibit-darwin-amd64`, `gh-exhibit-windows-amd64.exe`, etc.
  - `ldflags: -s -w -X main.version={{.Version}} -X main.commit={{.Commit}}
    -X main.date={{.Date}}` — a snapshot-built binary's `--version` output
    was run directly and printed the injected values, not assumed from the
    config alone.
  - `release.mode: keep-existing` (GoReleaser's own default, kept explicit
    with a comment) reproduces `build_and_release.sh`'s own existing
    behavior (upload to an already-existing release without touching its
    notes) — confirmed with the user: preserves the
    `gh release create vX.Y.Z --notes-file ... --target main` flow
    `CHANGELOG.md`'s header comment already documents, unchanged.
  - **A `before.hooks: [go mod tidy]` step was tried, based on
    `k1LoW/gh-grep`'s own `.goreleaser.yml`, and dropped**: running it
    locally mutated `go.sum`, pulling in `github.com/MakeNowJust/heredoc`
    — a transitive dependency unrelated to any import this project
    actually added. `ci.yml`'s own checks never call `go mod tidy`, so
    this project has no existing dependency on that step; reverted before
    committing (`git checkout -- go.sum`) rather than carrying an
    unexplained dependency-lock change.
  - 12 target platforms (`darwin`/`freebsd`/`linux`/`windows` ×
    `386`/`amd64`/`arm`/`arm64`, minus the same 4 combinations
    `cli/gh-extension-precompile` never supported) all built successfully
    in the snapshot run.
- `.gitignore` gains `/dist/` (GoReleaser's local build output directory;
  the project previously only ignored the single top-level binary
  `cli/gh-extension-precompile` produced).
- **Does not reopen ADR-002's Go-for-distribution-ease reasoning**
  (`gh extension create`'s scaffolding targets Go, which remains true) —
  only the specific precompile mechanism changed, so no ADR amendment was
  needed, matching how the CI workflow swap (MoonBit-copy → Go-native) was
  handled as a `docs/todo.md` entry rather than an ADR revision.

`go build ./...`, `go vet ./...`, `go test ./... -race -cover`, and
`gofmt -l .` all pass. No release tag has been pushed yet to exercise the
new workflow for real — deferred to whenever the user is ready to publish,
same as the original distribution-scaffolding entry's own deferral.

### `v0.1.0` release and a real-run-only `release.yml` bug (2026-07-19)

`v0.1.0` was tagged via `gh release create v0.1.0 --title "v0.1.0"
--notes-file .connect0459/gh-release-draft.md --target main` (#15 merged
first, closing #10), the first time `release.yml` ran against a real tag
push rather than a local `goreleaser --snapshot` dry run. GoReleaser
itself succeeded — all 12 platform binaries and `checksums.txt` were
built and attached to the GitHub Release correctly — but the workflow's
final step, `actions/attest-build-provenance`, failed:

- **`subject-path: dist/gh-exhibit-*` matched nothing on disk.** A
  `formats: [binary]` archive (per `.goreleaser.yml`) uploads its release
  asset under `archives.name_template` (`gh-exhibit-{os}-{arch}`), but
  that renaming happens only at upload time — the actual built file stays
  at its build-id directory the whole run, e.g.
  `dist/gh-exhibit_linux_amd64_v1/gh-exhibit` or
  `dist/gh-exhibit_windows_386_sse2/gh-exhibit.exe`, never at a
  `dist/gh-exhibit-*` top-level path. The `--version` flag entry's own
  "verified end-to-end" claim above covered `goreleaser release
  --snapshot --clean`'s asset naming and a built binary's `--version`
  output, but not this attestation step, which no snapshot run exercises
  (it only runs in `release.yml`, gated on a real tag push). Fixed on
  `ci/attest-build-provenance-subject-path`: `subject-path` now reads
  `dist/gh-exhibit_*/gh-exhibit*`, matching the real build-id-directory
  layout confirmed directly from this run's own artifact listing (not
  assumed).
- The GitHub Release itself needed no fix or re-run — its assets were
  already correct and complete before the failing step ran; only the
  build-provenance attestation was missing. This gap is deferred to the
  next tag push (no in-place way to re-run just the attestation step
  against `v0.1.0`'s existing assets without re-running the whole
  release).
- No test file for a workflow YAML change, same precedent as
  `release.yml`/`ci.yml`'s own prior entries; verified via `pre-commit
  run --files .github/workflows/release.yml` (`check-yaml` passes) and by
  reproducing the exact on-disk paths from the failed run's own artifact
  JSON, not by re-running GoReleaser locally.

### Anemic Domain Model fixes in attachment handling (2026-07-20)

With every `docs/todo.md` checklist item already closed and `v0.1.1`
released, a review of `internal/domain/services`'s attachment-download half
(ADR-002's mandatory-local-download policy) for Anemic Domain Model smells —
behavior that belongs on a domain object but is instead scattered across a
service — found three instances, all fixed on
`refactor/anemic-domain-model-in-attachment-services` (merged as PR #21;
this entry itself was not part of that PR, see below):

- **`Rewrite` reached into `Resolution`'s fields to decide what Markdown
  text should replace an attachment URL**, even though `Resolution` already
  carried every field that decision depends on (success/local path/failure
  reason). The decision is now `Resolution.Substitute`, a method on the
  type that owns that state; `Rewrite` is reduced to a thin loop calling it.
- **`Detect`, `Filename`, and `Resolution` were three independent fragments
  of the same "GitHub attachment" concept**, with `Filename` a free function
  taking a bare URL string re-passed at every call site. A new `Attachment`
  type now carries that URL; `Filename` is a method on it, so filename
  derivation travels with the URL it derives from.
- **The `{number}/assets/{filename}` relative path a rendered document uses
  to reference its own downloaded attachments was an inline `fmt.Sprintf` in
  the application layer**, duplicating a layout decision that belongs with
  `IssueRef`, which already owns the issue number it depends on. It is now
  `IssueRef.AssetPath`.
- No behavior change; no public API or on-disk output layout change — this
  only moves existing logic to where it already conceptually belonged.
  Tests were updated in lockstep with each signature change
  (`detect_test.go`, `filename_test.go`), and a direct test was added for
  each newly-introduced method before wiring it in
  (`resolution_test.go`'s `TestResolution_Substitute*` cases,
  `issue_ref_test.go`'s `TestIssueRef_AssetPath...`).
- **This `docs/todo.md` entry itself is a corrective, not a same-PR
  record**: PR #21's own Quality Checklist claimed `docs/todo.md` was
  updated, but no commit on that PR actually touched this file — the first
  gap of this kind found in the project's otherwise-consistent history of
  recording every PR's work here. Written after the fact, once noticed
  during a routine "what's next" check with no open issues/PRs and a fully
  green `go build`/`go vet`/`gofmt`/`go test -race -cover` baseline to work
  from.
- **Deferred item, now resolved**: PR #21's own "Deferred Items and TODOs"
  section flagged that `Resolution.Substitute` still took `url` as a
  parameter rather than `Resolution` storing its own URL, so the
  `map[string]Resolution` at the call site left the URL duplicated as both
  key and field — deliberately not pursued in that PR to keep its scope to
  the three findings above. Fixed here, on
  `refactor/resolution-owns-its-url`: `Downloaded`/`FetchFailed` now take
  `url` as their first argument and `Resolution` stores it; `Substitute`
  takes no argument; `Rewrite` takes `[]Resolution` instead of
  `map[string]Resolution`, since `Detect` already guarantees unique URLs
  (deduplicated, first-seen order) so a map's key-uniqueness was never
  load-bearing here. Red/Green TDD: `resolution_test.go`/`rewrite_test.go`
  were updated to the new signatures first and confirmed failing to compile
  against the pre-change code, then the implementation and
  `export_service.go`'s call site were updated to match.

C0 unchanged: `internal/domain/services` 99.3%, `internal/application/
services` 97.2%. `go build ./...`, `go vet ./...`, `go test ./... -race
-cover`, and `gofmt -l .` all pass.

### Attachment URL as a value object, not a primitive string (2026-07-20)

With every checklist item above (including the `Resolution`-owns-its-url
follow-up) merged, the user asked whether any remaining `url string` (or
similar primitive) crossing the domain/application/infrastructure layers
could become a Value Object for stronger type-level guarantees. A survey
of `internal/domain/{valueobjects,services,repositories}`,
`internal/application/services`, and `internal/infrastructure/{github,
persistence}` found several bare-`string` domain concepts (attachment URL
at the `AttachmentFetcher` port, `NewAttachment`/`Resolution`'s own
un-validated `url`, `Attribution.url`, `filename`, `contentType`,
`InlineContext.path`, `ExportService.host`), ranked by how concrete the
invariant gap actually was rather than by how many places a `string` could
theoretically become a type. Three were confirmed with the user and fixed
on `refactor/attachment-url-invariants`; the rest were judged either
already covered or not worth their own type given no invariant a new type
would actually enforce:

- **`repositories.AttachmentFetcher.Fetch` took a bare `url string` even
  though `resolveAttachments` already held a validated `services.Attachment`
  for every URL it fetched** — the call site unwrapped it via `a.URL()`
  just to cross the port boundary, then the infra implementation never used
  it as anything but a string again. `Fetch` now takes `services.Attachment`
  directly, removing that round trip; `internal/infrastructure/github`'s
  implementation extracts `.URL()` internally instead.
- **`NewAttachment(url string)` had no validation**, unlike
  `NewAttribution`/`NewIssueRef`/`NewInlineContext`, which all reject empty
  input at construction. Now returns `(Attachment, error)`. `Detect`'s own
  regex match can never actually be empty, so the new error branch is
  unreachable through that path specifically — kept anyway as the same
  defensive skip-and-continue this project already applies to other
  value-object invariants (e.g. the sixth review round's `subject_type:
  "file"` fix), rather than leaving a public constructor with no invariant
  for some future caller that doesn't go through `Detect`.
- **`Resolution`'s `Downloaded`/`FetchFailed` accepted an empty `url`**
  with no check at all, the one gap left after the `Resolution`-owns-its-
  url follow-up above. Both now return `(Resolution, error)`;
  `export_service.go`'s `resolveAttachments` propagates a construction
  failure like its other defensive fetch-loop errors (unreachable in
  practice for the same reason as `NewAttachment`'s: every caller already
  passes an `Attachment`-derived, non-empty URL). **Superseded by the next
  entry below**: a later round in this same PR replaced this non-empty
  check with `valueobjects.Url` itself, and `Downloaded`/`FetchFailed` no
  longer return an error at all — this bullet describes this round's own
  point-in-time design, not the PR's final state.
- **Checked and found already covered**: `Attribution.url` was initially
  flagged by the same survey, but `NewAttribution` already rejects an empty
  `url` (present since the domain layer's original implementation) — no
  change needed there; the survey's premise was stale on this one point.
  **Reversed by the next entry below**: a later round in this same PR
  found this conclusion itself premature and changed `Attribution.url` to
  `valueobjects.Url` after all — this bullet describes this round's own
  point-in-time conclusion, not the PR's final state.
- **Considered and not pursued**: `filename`/`contentType` staying as bare
  strings through `Attachment.Filename`/`AttachmentFetcher`'s return value;
  `InlineContext.path`'s shape (only used for rendering, never filesystem
  access); `ExportService.host`. None of these has a concrete invariant a
  new type would enforce beyond what already holds today, unlike the three
  fixed above, each of which either removed an actual round-trip or closed
  a real (if currently unreached) validation gap shared with this
  package's other value objects.
- Red/Green TDD per concern, one commit each: test files were updated to
  the new signatures first (confirmed failing to compile), then each
  production change followed. `newTestAttachment`/`mustDownloaded`/
  `mustFetchFailed` test helpers replace direct constructor calls now that
  each can fail, mirroring `newAttribution`'s own precedent.

C0 after this round: `internal/domain/services` 98.7% (down slightly from
99.3% — the new defensive, structurally-unreachable error branches in
`Detect`/`resolveAttachments` are the only new gaps, the same
"defensive, not meaningfully testable" shape as this project's other
accepted gaps, not a new kind of one). `internal/application/services`
95.7% (down from 97.2%, same reason). No behavior change and no on-disk
output layout change. `go build ./...`, `go vet ./...`, `go test ./...
-race -cover`, and `gofmt -l .` all pass.

### valueobjects.Url introduced, closing the previous round's own admitted gap (2026-07-20)

A local review of the round above pushed back on two points, both
addressed on the same `refactor/attachment-url-invariants` branch:

- **Object-selection critique from an even earlier round no longer
  applies.** That critique held that a `Url` type driven only by "reject
  an empty string" was arbitrarily scoped, since `title`/`author`/`path`/
  `owner`/`repo` have the same duplication. This round's actual proposal
  shares "is this string a parseable URL" — a semantics closed to
  url-shaped fields only — so the objection's premise no longer holds.
- **"Parse responsibility" was still hollow as stated.** `net/url.Parse`
  is permissive enough that `url.Parse("not-a-url")` and
  `url.Parse("just some text")` both return a nil error — verified
  directly, not assumed. A constructor that only checks `Parse`'s error
  rejects almost nothing beyond what a bare non-empty check already
  rejects; asserting "this is a URL" requires the constructor to also
  check `IsAbs()`, an allowed scheme (`http`/`https`), and a non-empty
  host.
- **The reviewer's sharpest point**: introducing `Url` mainly to
  strengthen `Attachment`'s own validation is weak, since `Detect`'s
  `urlPattern` regex already enforces scheme/host/path shape more tightly
  than a generic `Url` type would — wrapping an already-regex-matched
  string in `Url` is a redundant second check there. The real payoff is
  `Resolution`: it received `r.attachment.URL()` — an already-validated
  URL — as a bare string, so the previous round's non-empty check on
  `Downloaded`/`FetchFailed` was structurally unreachable and had no
  better option available as long as `url` stayed a plain `string`. Once
  `Attachment.URL()` returns `Url`, `Resolution` can carry that same
  already-checked value forward and drop its own validation and error
  return entirely (Parse, don't validate — validate once, carry proof in
  the type, never re-check).
- Confirmed with the user to apply `Url` uniformly to `Attribution.url`
  too, not only `Attachment`/`Resolution`: `Attribution.URL()` returning a
  bare `string` while `Attachment.URL()` returns `Url` would leave the
  same "already validated, then thrown away" gap at a different boundary
  (rendering, not domain-to-domain handoff) — any future caller wanting
  the structured value would have to reparse a string this package had
  already validated once. Verified this carries no behavioral
  incompatibility first: every `NewAttribution` call site in production
  and tests already passes an absolute `https://github.com/...` URL, so
  the stronger check rejects nothing that previously succeeded.
- `internal/domain/valueobjects.Url` (new): `NewUrl(raw string) (Url,
  error)` validates via `url.Parse` plus the `IsAbs`/scheme/host checks
  above. `String()`/`Scheme()`/`Host()`/`Path()`/`Equals()`.
  `MarshalText()` renders the original raw string, so a `Url`-typed
  struct field marshals to JSON identically to a plain `string` field —
  verified directly against `json.Marshal` on an equivalent string, not
  assumed, since ADR-001's `meta:{...}` line requires byte-exact output.
- `Attachment.url`/`Attribution.url`/`Resolution.url` all changed from
  `string` to `Url`. `Attachment.Filename` now derives its id from the
  `Url`'s own `Path()` rather than treating the whole raw string as a
  path (same result, more precise about which component is meant).
  `Downloaded`/`FetchFailed` no longer return an error at all — an
  invalid `Resolution.url` is no longer representable. The four Tier 1
  types' meta structs (`Body`/`IssueComment`/`InlineReviewComment`/
  `PullRequestReview`) change their `URL` field type from `string` to
  `Url`; every existing byte-exact rendering test passed unchanged,
  serving as this change's own regression check.
- Red/Green TDD, one commit per type (`Url` itself, then `Attachment`,
  then `Resolution`, then `Attribution`), each left the tree building and
  green before the next commit — `Attachment`'s commit temporarily added
  an explicit `.String()` at its downstream call sites as a compatibility
  bridge, removed once `Resolution`'s own commit accepted `Url` directly,
  since Go's whole-module compilation left no smaller commit boundary
  available for a change this coupled.

C0 after this round: `internal/domain/services` 98.7% (`Detect`'s
regex-guaranteed-unreachable skip branch is the only remaining gap;
`Resolution`'s previously-added unreachable branches are gone along with
its error return, so this package's dip from the prior round partially
reverses). `internal/application/services` recovers fully to 97.2% (its
own unreachable branches are gone with them). `internal/domain/
valueobjects` 95.9% (up slightly — `Attribution`'s stronger check adds
covered branches). No behavior change and no on-disk output layout
change. `go build ./...`, `go vet ./...`, `go test ./... -race -cover`,
and `gofmt -l .` all pass.

### ADRs replaced by docs/specs/README.md (2026-07-20)

With every implementation slice long closed, the user raised whether the
two ADRs (`docs/adrs/adr-001-initial-plan.md`,
`docs/adrs/adr-002-language-and-domain-design.md`) and the ~40 Go comments
citing them across ~20 files should be discontinued. Both were already
Accepted, never Deprecated, and both had been revised in place with dated
addenda (e.g. "Timeline API correction", "Naming addendum") rather than
superseded by a new ADR — the template (`docs/adrs/adr-000-template.md`)
itself invited this via its own "Revision" section, unlike the classical
Nygard ADR convention (immutable once Accepted; changed only by a new,
superseding ADR). Discussed with the user and resolved on
`docs/deprecate-adrs-in-favor-of-spec`:

- **The two premises behind full removal were checked, not just accepted.**
  (1) "`docs/todo.md` already records everything" does not fully hold: this
  file's own Design section only summarizes each ADR's conclusion in 1-3
  sentences and explicitly deferred to the ADR body for detail: several
  investigation findings existed only there (the `gh-dossier`/
  `gh-paper-trail`/`gh-attest` naming-candidate rejections; the GraphQL
  timeline-union member-count findings; `gh api rate_limit --include`'s
  actual output shape and the confirmation that neither `gh api` nor
  `go-gh` retries rate limits; the MoonBit-disqualification reasoning; the
  Entity-vs-Value-Object argument tied to this project's git-diff-based
  YAGNI stance). (2) "Decision records don't matter at this project's
  scale" was accepted as scoped to this project specifically (a
  single-maintainer OSS tool, no regulatory or financial stakes), not
  generalized.
  - This is resolved by the same reasoning this project already applies to
    other historical information: nothing is destroyed by deleting a
    committed file — `git log --follow -- docs/adrs/` and
    `git show <commit>:docs/adrs/adr-002-language-and-domain-design.md`
    (on a commit prior to their removal) still surface it. Discoverability
    moves from "always-visible doc" to "deliberately searched git
    history," which matches this project's own "commit messages = why,
    tests = what, code = how" convention more closely than a permanently
    live, separately-maintained decision-record document does.
- **A grep-verified survey preceded any comment deletion**, not an
  assumption: every one of the ~40 `ADR-00X`-citing Go comments across ~20
  files was inspected before removal. All were restatements of *current*
  behavior ("per ADR-002's on-disk layout", "ADR-001's Markdown dialect")
  rather than references to rejected-alternative reasoning — i.e., already
  a violation of this project's own "code = How, comments only for
  non-obvious WHY" policy, independent of the ADR-removal question. None
  needed migrating; each was either deleted outright or reworded to point
  at `docs/specs/README.md` where a live pointer was still useful (on-disk layout,
  Markdown dialect, coverage targets).
- **`docs/specs/README.md` (new)**: a single, always-in-place-edited specification
  of gh-exhibit's *current* behavior — distribution/stack, CLI shape,
  domain model, timeline classification, on-disk layout, Markdown dialect,
  attachment policy, rate-limit/retry policy, concurrency, package layout,
  coverage targets, and error-message conventions. Written from the current
  code and this file's own history, not transcribed from the ADRs' text as
  originally accepted — several details had already drifted since
  acceptance (e.g. `entry`/`timeline`/`attachment` renamed to
  `valueobjects`/`services`; `EvidenceRepository` renamed to
  `EvidenceFetcher`; distribution migrated from
  `cli/gh-extension-precompile` to GoReleaser). Explicitly states its own
  maintenance convention up front (edited in place to match current
  behavior; no dated addenda; rationale belongs in commit messages and
  this file, not here) so it does not silently drift into the same
  live-history/current-truth conflation this entry is resolving.
- `docs/adrs/` deleted in full (all three files, including the template).
  `.github/ISSUE_TEMPLATE/BUG_REPORT.md`, `FEATURE_REQUEST.md`,
  `PULL_REQUEST_TEMPLATE.md`, and `CONTRIBUTING.md`'s ADR references
  repointed to `docs/specs/README.md`.
- No test file for any of this — none of it is Go logic (only comments and
  Markdown changed); verified via `go build ./...`, `go vet ./...`,
  `go test ./... -race -cover`, and `gofmt -l .` (no behavior change, so no
  coverage figure moved), plus `pre-commit run --all-files`.
