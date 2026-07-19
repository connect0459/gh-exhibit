# ADR-001: Separate acquisition from rendering; store full-fidelity REST data

## Status

- [ ] Proposed
- [x] Accepted
- [ ] Deprecated

Accepted on 2026-07-18.

## Context

This tool exports GitHub Issues/PRs as local files to serve as reference
material and audit evidence for technical assessment READMEs. The target
format is modeled on an existing hand-maintained export directory (part of
a private assessment repository, not included here): one Markdown file per
Issue/PR, where the title is an H1 heading, each entry (body or comment)
starts with a line-anchored `meta:{...}` JSON line carrying
author/created/url, and entries are separated by a `------` line. Analyzing
that directory surfaced these facts:

- The `meta:{...}` line-anchored JSON and the `------` separator are
  deliberate non-standard tokens chosen to avoid collision with legitimate
  Markdown content (code blocks, YAML samples, `---` rules). This design
  holds as long as parsing is anchored to the start of line.
- Inline PR review comments lose their file/line context in the current
  format; attachments mix hotlinks and local `assets/` references.
- Replacing `meta:{...}` with YAML frontmatter was considered and
  rejected: repeating frontmatter-like blocks per comment falls outside
  the standard frontmatter usage (a custom parser would still be needed),
  and `---` appears in real Issue/PR bodies (YAML samples, horizontal
  rules) more often than a 6-hyphen line does, so the collision risk
  would worsen rather than improve.

For the intermediate representation, mirroring the GitHub GraphQL timeline
unions was considered and rejected. Verified via introspection on 2026-07-18:

- `IssueTimelineItems` has 51 member types; `PullRequestTimelineItems` has 78.
  Modeling all of them is overinvestment for this purpose.
- Both `Issue.timelineItems` and `PullRequest.timelineItems` accept an
  `itemTypes` filter argument, so server-side filtering is possible.
- There is no common interface across timeline members that exposes
  `actor`/`createdAt` generically (e.g. `ClosedEvent` implements only `Node`
  and `UniformResourceLocatable`; `MentionedEvent` only `Node`). GraphQL
  returns only the fields requested per concrete type, so an "open
  receptacle" that passes through raw JSON for unmodeled types is
  **impossible over GraphQL** — unknown types degrade to `__typename` only.

The project README frames the tool as an audit trail ("the file itself is
independent evidence"), not a mere archive. Server-side filtering
(`itemTypes`) would contradict that framing by selectively collecting
evidence. The tension dissolves by separating two policies that an earlier
draft conflated:

- **Acquisition policy** — what the stored intermediate data retains.
- **Rendering policy** — what the generated Markdown shows.

### Repository naming (added 2026-07-18)

The working name `gh-audit-export` failed on identifiability. Surveying the
ecosystem via `gh search repos` and `gh ext search` showed the "audit"
namespace among gh extensions is occupied by security/configuration audit
tools (`gh-ghas-audit`, `gh-branch-auditor`) and GitHub Audit Log tooling
(`ghec-audit-log-cli`), so the name reads as "a tool that exports audit
logs". The intended reading of "audit" was the noun (audit trail), but in
this namespace it is parsed as the verb (to audit). The name also lacks its
object (Issues/PRs). The `gh-` prefix itself is a packaging constraint:
`gh extension install` resolves repositories named `gh-*`, so keeping the
prefix is required to distribute the tool as a gh CLI extension.

Candidates evaluated:

- **`gh-exhibit`** — courtroom term for a piece of evidence; maps
  one-to-one onto the README's framing that "the file itself is
  independent evidence". No collision with existing gh extensions or
  repositories (verified). **Adopted.**
- `gh-dossier` — fits the "bundle of documents" form but connotes a mere
  collection; the audit-trail implication is weak.
- `gh-paper-trail` — idiomatically ideal but collides with Papertrail
  (SolarWinds SaaS) and paper_trail (Ruby gem). Rejected.
- `gh-attest` / `gh-provenance` — a `gh attestation` subcommand already
  exists, and "provenance" is occupied by the SLSA/supply-chain context;
  either would reproduce the same collision failure as "audit". Rejected.

Two objections were considered and dismissed:

- **Dropping the `gh-` prefix for forge portability** conflates product
  scope (supporting other forges) with the name's promise (behaving as a
  gh extension). Portability already lives in the architecture — the
  raw-JSON pass-through layer is forge-agnostic where it needs to be —
  and encoding it in the name is speculative generality. GitHub redirects
  renamed repositories, so the cost of correcting an overly specific name
  later is low.
- **Language suffixes** (`-go`, `-rust`) belong to library/spec
  implementations where the host language is part of the usage contract
  (`starlark-go`, `starlark-rust`). A CLI's contract is its process
  interface; the implementation language is a mutable detail and must not
  be encoded in a stable identifier.

### Timeline API correction (spike finding, 2026-07-18)

The premise stated earlier in this document — that Tier 1's render set is
sourced entirely from `.../issues/{number}/timeline` — does not hold for
inline review comments. This was verified empirically (the spike deferred
in `docs/todo.md`) against two `cli/cli` pull requests with real review
activity (#13780, 24 review comments; #13084):

- The `reviewed` timeline event carries only the review's own top-level
  `body` and `state`; it never embeds per-line comments.
- No `line-commented` event was observed in either PR's timeline, despite
  #13780 having 24 inline review comments across 5 reviews. This event type
  does not appear in the current REST timeline response, contradicting the
  assumption an earlier draft of this ADR and the `docs/todo.md` spike item
  had carried forward.
- Inline review comments (`path`, `line`, `diff_hunk`) exist only via a
  separate endpoint: `GET /repos/{owner}/{repo}/pulls/{number}/comments`, a
  homogeneous array where each element references its parent review via
  `pull_request_review_id` (confirmed on #13780: a comment with
  `path: "docs/release-process-deep-dive.md"`, `line: 195`,
  `pull_request_review_id: 4618365681` matching a `reviewed` event's `id`).

This corrects the acquisition source, not the Tier 1 render set itself:
inline review comments are still Tier 1, but sourced from a second REST
call joined by `pull_request_review_id`, not classified out of the
timeline array alongside the other three Tier 1 types.

## Decision

1. **Acquisition: full-fidelity REST storage.** Fetch via the REST API
   (`GET /repos/{owner}/{repo}/issues/{number}` plus
   `GET /repos/{owner}/{repo}/issues/{number}/timeline`; PRs additionally
   `GET /repos/{owner}/{repo}/pulls/{number}` and
   `GET /repos/{owner}/{repo}/pulls/{number}/comments` for inline review
   comments, joined to their parent review via `pull_request_review_id`)
   and persist the raw JSON responses verbatim. REST timeline events are
   self-describing JSON objects, so unknown or future event types are
   stored without any schema to break — the receptacle is genuinely open.
2. **Rendering: Tier 1 only, generated from the stored raw data.**
   Tier 1 = issue/PR body, issue comments, PR reviews (with state:
   approved / changes requested / commented), and inline review comments
   including their `path`/`line`/`diff_hunk` context (restoring what the
   current format loses).
3. **Closed/merged timestamps come from the resource itself**
   (`closed_at` / `merged_at`), not from timeline events. Tier 2
   (one-line summaries of state-change events interleaved in the
   timeline) is deferred until a concrete need arises; the raw data
   already contains everything needed to add it without re-fetching.
4. **Keep the existing Markdown dialect**: `meta:{...}` anchored at start
   of line, `------` (6 hyphens) as the entry separator, one file per
   Issue/PR.
5. **Name the tool `gh-exhibit`** (renamed from the working name
   `gh-audit-export`), keeping the `gh-` prefix for distribution as a
   gh CLI extension. All in-repo references are updated in one pass.

## Consequences

- Unknown event types cannot crash acquisition or lose data; rendering can
  be extended (Tier 2+) later by re-rendering from stored raw JSON without
  re-fetching.
- Inline review comments require a second, homogeneous-array REST call
  (`pulls/{number}/comments`) in addition to the timeline, joined by
  `pull_request_review_id`. Classification of Tier 1 types is therefore
  not a single two-pass unmarshal over one array; it is a discriminator
  peek over the timeline array for three types plus a direct join for the
  fourth (see ADR-002's corresponding correction).
- Stored payloads are larger than a filtered export; this is the accepted
  cost of the audit-trail framing.
- Rendering is a pure function of local raw data, which keeps the
  renderer testable without network access (mocks only at the REST
  boundary, per the Detroit-school policy).

## References

- <https://docs.github.com/en/rest/issues/timeline> — REST timeline API
- <https://docs.github.com/en/graphql/reference/unions#issuetimelineitems>
- <https://docs.github.com/en/graphql/reference/unions#pullrequesttimelineitems>

## Related file paths

### Initial decision (2026-07-18)

- docs/adrs/adr-001-initial-plan.md (new)
- README.md (rewritten in English; framing aligned with this decision)
- docs/todo.md (follow-up items)

### Naming addendum (2026-07-18)

- docs/adrs/adr-001-initial-plan.md (naming context and decision added)
- README.md (references renamed to gh-exhibit)

### Timeline API correction (2026-07-18)

- docs/adrs/adr-001-initial-plan.md (acquisition source corrected for
  inline review comments; `pulls/{number}/comments` added)
- docs/adrs/adr-002-language-and-domain-design.md (classification design
  and on-disk layout corrected to match)
- docs/todo.md (spike item checked off with finding)
