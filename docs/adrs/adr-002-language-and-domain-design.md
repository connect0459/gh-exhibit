# ADR-002: Implementation language, domain type design, coverage targets, and on-disk layout

## Status

- [ ] Proposed
- [x] Accepted
- [ ] Deprecated

Accepted on 2026-07-18.

## Context

ADR-001 settled the acquisition/rendering split and the Markdown dialect but
left the implementation language, the acquisition strategy, the Go-level
type design for Tier 1 entries, coverage targets, and the on-disk layout for
raw JSON and attachments as open items (tracked in `docs/todo.md`). This ADR
records the resolution of all four, reached in a single design discussion.

### Implementation language and acquisition strategy

Three languages were on the table: Go, Rust, and MoonBit. MoonBit was
disqualified from parallel consideration: `CLAUDE.local.md` already
conditioned it on a spike confirming process-invocation and JSON-parsing
practicality, which was never run, so treating it as a peer candidate would
have ignored a verification requirement the project had already imposed on
itself.

The acquisition strategy (`gh api` shellout vs. native HTTP) and the
language choice are not independent variables. Three acquisition tiers were
identified:

1. **Full shellout** — delegate authentication, request execution,
   pagination, and error formatting entirely to the `gh` process.
2. **Hybrid** — delegate only credential resolution to `gh auth token`;
   implement HTTP request execution directly.
3. **Full native** — no process invocation at all. `go-gh` provides gh's own
   auth/config resolution logic as a library, but this is Go-exclusive;
   choosing it commits the language choice to Go as a side effect.

Two empirical findings, verified rather than assumed, corrected earlier
speculation and narrowed the comparison:

- `gh api rate_limit --include` produces a standard HTTP-style block
  (status line + `Key: Value` headers + blank line + JSON body), which is
  trivially line-parseable — an earlier claim that header/body mixing made
  this awkward was inaccurate and is retracted here.
- Reading `cli/cli`'s `pkg/cmd/api/api.go` and `cli/go-gh`'s
  `pkg/api/http_client.go` confirmed that **neither implements automatic
  retry or backoff for rate limiting** (403/429,
  `X-RateLimit-Remaining`, `Retry-After`). Retry policy is the caller's
  responsibility regardless of which acquisition tier is chosen, so this
  axis does not meaningfully differentiate the three tiers. `GET
  /rate_limit` was confirmed (via GitHub's documentation) not to consume
  quota, and is usable for proactive checks in any tier.

With the rate-limit axis roughly neutralized, the deciding factor was
maintainability: `gh api`'s CLI behavioral contract (flag semantics, error
text, exit codes) is less stable than a versioned library dependency.
`go-gh` is officially maintained by GitHub specifically for gh extensions,
making it the most stable dependency contract among the three tiers, for Go.
Distribution ease was the second factor: `gh extension create`'s official
scaffolding targets Go exclusively, which is consistent with ADR-001's
decision to distribute as a gh CLI extension (the `gh-` prefix requirement).

### Tier 1 Go type design

ADR-001's Tier 1 render set (issue/PR body, issue comments, PR reviews with
state, inline review comments with `path`/`line`/`diff_hunk`) is sourced
from `GET .../issues/{number}/timeline`, a heterogeneous JSON array — not
per-type endpoints. Classifying which raw element maps to which concrete
type is therefore a one-time parsing concern, not something the rendering
logic should repeat.

Go has no true sum type (no equivalent to a Rust `enum`). An earlier
comparison favoring Rust's `enum` + `match` for this reason is corrected
here: that comparison implicitly assumed rendering logic centralized in one
match expression. Under the project's Rich Domain Objects principle
(behavior belongs on the type, not scattered in a switch), Go's idiomatic
alternative — each concrete type owning its own `Render()` method, invoked
polymorphically — arguably fits the stated philosophy better than a
match-centralized design, which tends toward an Anemic Domain Model.

The chosen pattern:

- A sealed `Entry` interface (`Render(w io.Writer) error` plus an
  unexported marker method) restricts implementers to this package — the
  closest Go analogue to a closed sum type.
- Classification of timeline elements into concrete `Entry` types still
  requires an explicit dispatch point (a two-pass unmarshal: a
  discriminator-only peek, then full unmarshal into the resolved concrete
  type). Go's compiler does not enforce exhaustiveness here, unlike Rust's
  `match`; the `exhaustive` golangci-lint check is the practical
  substitute at this one seam.

**Correction (spike finding, 2026-07-18):** the two-pass unmarshal above
classifies only three of the four Tier 1 types. Verified empirically
against real `cli/cli` timeline responses: the `reviewed` timeline event
never carries per-line comments, and no `line-commented` event exists in
the current REST timeline at all. `InlineReviewComment` is not classified
out of the timeline array; it is fetched from a separate, already-
homogeneous endpoint (`GET /pulls/{number}/comments`) and joined to its
parent `PullRequestReview` via `pull_request_review_id`, which matches the
`reviewed` timeline event's `id`. The discriminator peek + `exhaustive`
lint applies to the timeline array's three remaining member types
(issue/PR body is not part of the timeline array either — it comes from
the issue/pull resource itself); `InlineReviewComment` construction is a
direct deserialization plus a join, not a classification decision. See
ADR-001's corresponding correction for the acquisition-side detail.

**Entity vs. Value Object.** Treating Tier 1 entries as entities merely
because they carry a GitHub-assigned ID would be premature: this project's
entity/value distinction hinges on whether an identifier's persistence
across attribute changes is actually tracked by some behavior. gh-exhibit
does not track re-fetch diffs — exported evidence is copied into a separate
GitHub repository, where git history already provides this via textual
diffing; adding identity-based diff tracking inside gh-exhibit would
duplicate a capability git already provides at a different layer (YAGNI).
**Decision: all four Tier 1 types are Value Objects.** No entity base type,
identity-based equality, or lifecycle tracking is needed.

Supporting value objects: `Attribution` (author, created, url — the common
`meta:{...}` fields), `ReviewState` (Approved / ChangesRequested /
Commented), `InlineContext` (path, line, diff_hunk, which always co-occur).

Implementation note: `time.Time`'s monotonic clock reading means naive `==`
comparison can fail even for values that represent the same instant (as
documented by the Go standard library). Any equality check needed on these
value objects must use `time.Time.Equal()` rather than `==`.

### Coverage targets

Go's built-in coverage tooling (`go test -cover`) reports only a statement-
coverage rollup; it has no separate, single metric for C1 (decision/branch)
coverage. Getting a distinct C1 number requires either a third-party
toolchain addition or manual inspection of `go tool cover -html`'s
per-block coloring. Given the project's proportionality precedent (favoring
lightweight tooling elsewhere), the decision is to avoid adding coverage
tooling solely to auto-gate C1.

Coverage risk and testability also differ meaningfully by layer, following
the onion-architecture separation already adopted: domain logic (Value
Objects, `Render()`, classification) is pure and cheaply testable in
memory; boundary code (`go-gh` HTTP calls, file I/O) is the Detroit-school
mock boundary, where covering every error branch (network failure, rate
limiting, malformed responses) costs more test-writing effort per branch. A
single blended target across both layers risks incentivizing tests written
only to hit a number rather than to document behavior (in tension with the
Evergreen Tests principle).

Decision:

- **C0/C1 gating**: CI auto-gates C0 only. C1 is reviewed manually via
  `go tool cover -html` during code review, not auto-gated.
- **Target granularity**: per layer, not a single project-wide number.
  Domain layer: C0 90% / C1 75-80% as a floor. Boundary layer: qualitative
  branch coverage via mocks, not a numeric target.
- **Backing off if infeasible**: no pre-agreed lower floor. Coverage
  breakdown is presented per package after implementation, and the floor
  is negotiated at that point rather than fixed in advance.

### On-disk layout and attachment policy

**Raw JSON layout** is split by source endpoint rather than consolidated
into a self-authored wrapper, preserving the "stores complete raw REST API
responses... verbatim" claim most literally (a consolidated wrapper would
itself be a schema this project invented, which is not verbatim):

- `issues/{repo}/{number}.json` — issue or pull request resource.
- `issues/{repo}/{number}.timeline.json` — timeline; multiple paginated
  responses are concatenated into a single JSON array before being
  persisted (not stored as separate per-page files).
- `issues/{repo}/{number}.pull.json` — pull request resource, additionally,
  for PRs.
- `issues/{repo}/{number}.review-comments.json` — inline review comments
  (`GET /pulls/{number}/comments`), additionally, for PRs. Added by the
  2026-07-18 timeline-API correction: this data does not appear in the
  timeline (see the Tier 1 Go type design section above), so it needs its
  own raw-JSON file alongside `{number}.timeline.json` rather than being
  folded into it.

Rendered Markdown is unaffected and stays `issues/{repo}/{number}.md`, per
the existing dialect (ADR-001). Whether these files are subsequently
committed to a repository, and which one, is downstream of this tool's
responsibility (the exported directory is copied by the user into a
separate evidence repository) and is out of scope for gh-exhibit's design.

**Attachment policy.** Hotlinking to `user-attachments` was rejected: it is
an external dependency on GitHub's CDN, vulnerable to link rot, access
changes, and future URL-format changes, which is in direct tension with the
README's stated principle that offline verifiability is the essence of this
tool. **Decision: local download is mandatory.** Attachments are fetched via
an authenticated request (required for private-repository attachments),
with the file extension derived from the response's `Content-Type` header
(GitHub's `user-attachments` URLs do not reliably encode an extension in
the path), and stored under `issues/{repo}/{number}/assets/{filename}`.

On fetch failure (broken link, access denied, network error): skip the
attachment and continue processing (do not fail the run). The Markdown
reference is replaced with a placeholder noting the original URL and
failure reason, rather than silently dropping the reference (which would
lose evidence that the attachment existed) or leaving a hotlink that
implies a working reference. A failure summary is persisted as a file in
the run's output directory (e.g.
`issues/{repo}/{number}/fetch-errors.log`), consistent with the audit-trail
framing, rather than only printed to stderr and lost after the run.

## Decision

1. **Language: Go.** **Acquisition: full-native via `go-gh`** (no `gh api`
   subprocess shellout, no hybrid shellout-for-token-only approach).
2. **Tier 1 domain model**: a sealed `Entry` interface with a polymorphic
   `Render()` method per concrete type. Three of the four Tier 1 types
   (IssueComment, PullRequestReview, plus the timeline's other member
   types) are classified from the timeline array via a two-pass unmarshal,
   checked for exhaustiveness via the `exhaustive` lint rule.
   InlineReviewComment is not classified from the timeline (verified
   empirically to be absent from it); it is fetched from
   `GET /pulls/{number}/comments` and joined to its parent
   PullRequestReview via `pull_request_review_id`. All four Tier 1 types
   are modeled as Value Objects composed from `Attribution`, `ReviewState`,
   and `InlineContext` value objects.
3. **Coverage**: C0 auto-gated in CI at a per-layer target (domain 90%,
   boundary judged qualitatively); C1 (75-80% domain floor) reviewed
   manually via `go tool cover -html`, not auto-gated; no pre-agreed lower
   floor for backing off — renegotiated per package after implementation.
4. **On-disk layout**: raw JSON split by source endpoint
   (`{number}.json`, `{number}.timeline.json`, `{number}.pull.json`,
   `{number}.review-comments.json`); rendered Markdown unchanged
   (`{number}.md`).
5. **Attachments**: mandatory local download with authenticated fetch,
   extension from `Content-Type`, stored under `{number}/assets/`; on
   failure, skip, warn, continue, emit a Markdown placeholder, and persist
   a failure summary file in the output directory.

## Consequences

- The language and acquisition-strategy decisions are coupled by
  construction: `go-gh` has no equivalent in other languages, so this
  decision forecloses Rust and MoonBit for this project without a separate
  re-litigation of language.
- Go's lack of true sum types is compensated by a sealed-interface +
  polymorphic-method pattern for rendering, and by an `exhaustive` lint
  check at the one remaining classification-time type switch. This is an
  accepted, partial substitute for compiler-enforced exhaustiveness, not an
  equivalent guarantee.
- Rate-limit retry/backoff logic must be implemented by gh-exhibit itself
  in all cases; neither `gh api` nor `go-gh` provides it.
- C1 coverage is not machine-enforced; its accuracy depends on manual
  review discipline rather than a CI gate.
- Raw JSON files are the evidentiary source of truth; Markdown remains a
  regenerable view per ADR-001. Committing either to a repository is
  entirely the user's downstream workflow, not something gh-exhibit
  enforces or assumes.
- Attachment fetch failures are visible both inline (Markdown placeholder)
  and in a persisted per-run log, so missing evidence is traceable rather
  than silent.
- The MoonBit spike previously flagged in `CLAUDE.local.md` is now moot
  given the language decision, unless revisited explicitly.
- Acquisition for PRs now requires four REST calls, not three
  (`issues/{number}`, `issues/{number}/timeline`, `pulls/{number}`,
  `pulls/{number}/comments`), each persisted to its own raw-JSON file. The
  `Entry` classification seam narrows to the timeline array's own member
  types; joining `InlineReviewComment` to `PullRequestReview` by
  `pull_request_review_id` is a separate, non-exhaustiveness-checked
  concern and needs its own test coverage (e.g., an orphaned comment whose
  `pull_request_review_id` matches no fetched review).

## References

- <https://github.com/cli/cli/blob/trunk/pkg/cmd/api/api.go>
- <https://github.com/cli/go-gh/blob/trunk/pkg/api/http_client.go>
- <https://docs.github.com/en/rest/rate-limit/rate-limit>
- <https://docs.github.com/en/rest/issues/timeline>
- <https://docs.github.com/en/rest/pulls/comments>
- docs/adrs/adr-001-initial-plan.md

## Related File Paths

### Initial decision (2026-07-18)

- docs/adrs/adr-002-language-and-domain-design.md (new)
- docs/todo.md (Design items checked off)

### Timeline API correction (2026-07-18)

- docs/adrs/adr-002-language-and-domain-design.md (classification design
  and on-disk layout corrected: `InlineReviewComment` sourced from
  `pulls/{number}/comments`, not the timeline)
- docs/adrs/adr-001-initial-plan.md (acquisition source corrected)
- docs/todo.md (spike item checked off with finding)
