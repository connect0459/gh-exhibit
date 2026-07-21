# gh-exhibit Specification

This document describes gh-exhibit's current behavior and design. It is a
living document: when behavior changes, this file is edited in place to
match, not appended to. Rationale for *why* a given decision was made lives
in commit messages and `docs/todo.md`'s dated history, not here — this file
states what is currently true, not why it became true or what alternatives
were rejected along the way.

## Purpose

gh-exhibit is a `gh` CLI extension that exports a GitHub issue or pull
request's full discussion (body, comments, reviews, inline review comments,
attachments) as offline-verifiable Markdown alongside the raw JSON evidence
it was rendered from. The exported directory is intended as audit-trail
evidence: the file itself, not a live link back to GitHub, is the record.

## Distribution and stack

- Language: Go. Acquisition is full-native via `go-gh` (`github.com/cli/go-gh`)
  — no `gh api` subprocess shellout, no hybrid shellout-for-token-only
  approach. `go-gh` supplies `gh`'s own auth/config resolution as a library.
- Distributed as a `gh` extension (`gh extension install
  connect0459/gh-exhibit`); the `gh-` repository name prefix is required for
  `gh extension install` to resolve it.
- Released via GoReleaser (`.goreleaser.yml`), triggered by a `v*` tag push
  (`.github/workflows/release.yml`). Release assets are named
  `gh-exhibit-{os}-{arch}` (hyphenated, no archive extension —
  `archives.formats: [binary]`), matching the pattern `gh`'s extension
  manager matches release assets against. `-ldflags` injects `main.version`/
  `main.commit`/`main.date` at build time.
- CI (`.github/workflows/ci.yml`): `gofmt -l .` → `go vet ./...` →
  `golangci-lint` → `go build ./...` → `go test ./... -race -cover`, gated
  behind whether the push/PR touched a Go-relevant path.

## CLI interface

`cmd/gh-exhibit/main.go` is the composition root:
`cli.ParseArgs` → `cli.ResolveRepo` → `registry.NewExportService` →
`cli.RunExports`.

```sh
gh exhibit <number>[,<number>...] [--repo <owner>/<repo>] [-o|--output <dir>]
gh exhibit --version
```

- Positional argument: a single issue/PR number, or a comma-separated list
  of them (deduplicated, first-seen order). No range or `--all` syntax.
  Flags may appear before, after, or interleaved around it.
- `--repo owner/repo`: target repository; defaults to the current
  directory's repository context (`go-gh`'s `repository.Current`) when
  omitted.
- `-o`, `--output`: output directory the evidence is written under; defaults
  to `.`.
- `--version`: prints `gh-exhibit {version} (commit {commit}, built {date})`
  and exits, without requiring a positional number.
- A failing ref in a list does not stop the remaining ones in the same
  invocation. Each ref's success or failure is reported on its own line to
  stdout/stderr respectively. Process exit code is `0` only if every ref
  succeeded, `1` otherwise.

## Domain model

### Tier 1 entries

The rendered Markdown's content is drawn from five entry types, all Value
Objects (no identity-based tracking — re-fetch diffing is git's job in the
separate repository the evidence is copied into, not gh-exhibit's):

- `Body` — the issue/PR's own body.
- `IssueComment` — a top-level comment.
- `PullRequestReview` — a review, carrying a `ReviewState`
  (`approved` / `changes_requested` / `commented`).
- `InlineReviewComment` — a comment anchored to a diff position, carrying an
  `InlineContext` (`path`, optional `line`, optional `start_line`,
  `diff_hunk`, `outdated`).
- `LabelEvent` — a label added to or removed from the issue/PR, sourced from
  the timeline's `labeled`/`unlabeled` events, carrying a `LabelAction`
  (`labeled` / `unlabeled`) plus the affected label's name and color.

All five implement the sealed `valueobjects.Entry` interface
(`Render(io.Writer) error` plus an unexported marker method) — the closest
Go analogue to a closed sum type. Supporting Value Objects: `Attribution`
(author, created, url — the common `<!-- {"meta":...} -->` fields), `Url`
(an absolute http/https URL, parsed and validated once at construction),
`IssueRef` (owner, repo, number — validated against GitHub's own username/
repository-name character-set and length rules; `repo` additionally
rejects an all-dots segment such as `.`, `..`, or `...`, optionally
followed by trailing spaces, via the same check `AssetFilename` uses —
see "Attachment policy" below), `Provenance` (tool,
version, commit — which gh-exhibit build produced an export, persisted to
`evidence/provenance.json` by `ProvenanceWriter` rather than rendered into
`Document`), `AssetFilename` (a downloaded attachment's on-disk filename,
guaranteed by its constructor to be a single path-safe segment — see
"Attachment policy" below).

### Timeline classification

Three of the five Tier 1 types (`IssueComment`, `PullRequestReview`, and
`LabelEvent`) are classified from `GET .../issues/{number}/timeline`'s
heterogeneous array via a two-pass unmarshal (discriminator peek, then
dispatch), checked for exhaustiveness by the `exhaustive` golangci-lint
rule against `eventKind`. The timeline array also carries other event
kinds with no corresponding Tier 1 type (e.g. `review_requested`), which
are left unclassified rather than causing an error.
`InlineReviewComment` is not part of the timeline array at all — it is
fetched from `GET /pulls/{number}/comments` and joined to its parent
`PullRequestReview` via `pull_request_review_id`, matching the `reviewed`
timeline event's own `id`.

A timeline item or review comment that fails to parse or violates a Value
Object's invariant does not abort the whole export: it is recorded as a
`services.SkipNote` (reason plus the raw JSON) and processing continues.
Deleted-account authors (`user: null`) are attributed to `"ghost"`. An
outdated inline comment (`line: null`) falls back to `original_line` and
renders an `outdated` flag; a file-level comment (`subject_type: "file"`,
both `line` and `original_line` null) has no line at all; a comment
anchored to a range of lines carries a `start_line` (or, once outdated,
`original_start_line`) alongside its `line`, rendering both so the export
preserves the comment's full span rather than only its last line.
Duplicate timeline items/review comments sharing the same id (e.g.
overlapping pagination) are deduplicated, not re-rendered.

`Attribution.author`/`IssueRef.owner`/`IssueRef.repo` compare
case-insensitively via `Equals` (GitHub's own case-insensitive uniqueness
rule for logins and repository names), but are stored and rendered
verbatim — no construction-time case normalization. `Attribution.author`
additionally rejects any non-ASCII byte, since Unicode case folding can
conflate a non-ASCII character with an ASCII one (e.g. U+212A KELVIN SIGN
folds to `"k"`); this does not apply to `IssueRef.owner`/`repo`, which are
already constrained to GitHub's ASCII username/repository-name pattern.

### Label rendering

A label added to or removed from the issue/PR is rendered as a `LabelEvent`
interleaved chronologically with the rest of the timeline (a `labeled` or
`unlabeled` event, classified the same way as `commented`/`reviewed`),
rather than as a static snapshot of the current label set placed near the
document's top. This was a deliberate choice over the static-list
alternative: it stays consistent with how every other entry in the document
already presents content in the order it happened, at the cost of needing to
recognize the corresponding timeline event kind rather than only reading the
issue/PR resource's own `labels` field.

Unlike `commented`/`reviewed` events, GitHub's `labeled`/`unlabeled`
timeline payload carries no per-event `html_url` — only an API URL
(`.../issues/events/{id}`) that doesn't resolve to a human-readable page.
`LabelEvent`'s attribution therefore falls back to the issue/PR's own
`html_url` (already available from the issue/PR resource fetched
alongside the timeline) rather than a link to the specific historical
event.

## On-disk layout

Every artifact for a given issue/PR number lives under one self-contained
directory (a page-bundle layout, the same shape as a Hugo/Zola leaf
bundle), rather than a rendered file sitting as a same-stem sibling of a
directory holding its own assets:

```text
{output}/{repo}/{number}/
├── index.md                          rendered Markdown
├── assets/{filename}                 downloaded attachments
└── evidence/
    ├── issue.json                    issue or pull request resource
    ├── timeline.json                 timeline (paginated responses concatenated into one array)
    ├── pull.json                     pull request resource (PRs only)
    ├── review-comments.json          inline review comments (PRs only)
    ├── provenance.json               which gh-exhibit tool/version/commit produced this export
    └── fetch-errors.log              this run's attachment fetch failures, if any
```

`{repo}` only — the owner is deliberately not part of the path. Raw JSON is
split by source endpoint rather than consolidated into a self-authored
wrapper, to keep each file's content a literal, verbatim REST response; the
enclosing `evidence/` directory (rather than a `{number}` filename prefix)
disambiguates it from the rendered document, since the number is already
encoded by the parent directory. Multi-page timeline/review-comment
responses are spliced into one JSON array by concatenating each page's raw
bytes directly (not `json.Marshal`-ing a `[]json.RawMessage` slice, which
would compact each element's whitespace and break the verbatim guarantee).

`fetch-errors.log` and `provenance.json` live under `evidence/` alongside
the raw JSON rather than at the `{number}/` top level: the operative
grouping is "final rendered exhibit" (`index.md` + `assets/`) vs.
"everything else supporting it," so gh-exhibit-generated artifacts about
the export itself — a failed fetch's log, or which tool/version/commit
produced the export — belong with the other non-presentation artifacts.
Unlike its four siblings, `provenance.json`'s content is not a verbatim
external REST response — it is gh-exhibit's own self-reported
`{"tool":...,"version":...,"commit":...}`, written by a distinct
`ProvenanceWriter` port rather than `EvidenceWriter`, which is scoped to
raw GitHub-origin data only.

`Export` runs every fetch, classify, and render step to completion before
any file is written, so a failure during that phase leaves nothing on disk.
The write phase itself has no rollback or staging — a failure partway
through it (e.g. the timeline file succeeds but the rendered document fails)
can leave a partial evidence directory behind. This is accepted rather than
built around: a rerun of the same ref overwrites every file, so the
directory is a self-healing, regenerable view, not the record of truth
itself (the raw JSON is).

## Markdown dialect

One Markdown file per issue/PR: an H1 title line, then each entry's
rendered output, separated by a `------` (6-hyphen) line. Each entry starts
with a `<!-- {"meta":{...}} -->` line anchored to the start of a line — an
HTML comment, hidden from a rendered Markdown preview but still greppable
as raw text, wrapping a standalone-parseable JSON object (`meta` nested
under its own key: `author`, `created` in RFC 3339 UTC, `url`, plus
type-specific fields — `PullRequestReview` includes `state`), optionally
followed by a blank line and the entry's body content. `InlineReviewComment`
renders its diff hunk under an explicit `**Diff:**` label in a fenced code
block, using a fence one backtick longer than the longest backtick run
inside the hunk itself (minimum 3), so a hunk containing its own
triple-backtick run cannot prematurely close the fence.

`index.md` deliberately mirrors GitHub's own content as closely as
possible; which tool/version/commit produced the export is not GitHub
content, so it is recorded separately as `evidence/provenance.json`
(see "On-disk layout" above) rather than rendered into the Markdown
itself.

`<!-- {"meta":...} -->` and `------` are deliberately non-standard tokens
chosen to avoid collision with legitimate Markdown content (code blocks,
YAML samples, `---` rules), on the condition that parsing stays anchored
to the start of a line. An HTML comment's own terminator is the literal
3-character sequence `-->`; this is never produced from a field's own
content, because the meta line is built through plain
`encoding/json.Marshal`, whose documented default behavior replaces every
`<`, `>`, and `&` byte with its 6-character numeric escape instead of
emitting it raw (verified directly: marshaling a `>`-containing string
never yields a literal `>` in the output). This holds regardless of what a
field's own value contains — notably, `InlineReviewComment`'s `path` has
no character-set constraint at all (`NewInlineContext` only rejects an
empty one, since a git path may contain almost any byte), so it is this
escaping behavior, not any field's content being inherently `>`-free, that
keeps the comment from closing early. This does depend on `writeMetaLine`
never switching to a `json.Encoder` with HTML escaping disabled, or to
building the line by hand instead of through `encoding/json`.

## Attachment policy

Hotlinking to GitHub's `user-attachments` CDN is not used — local download
is mandatory, to keep the exported directory offline-verifiable. After a
`Document` is fully rendered:

1. `services.Detect` finds every `user-attachments` asset URL referencing
   the target repository's own host (`github.com`, or a GitHub Enterprise
   Server host) in the rendered Markdown, deduplicated in first-seen order.
2. Each is fetched via an authenticated request (required for
   private-repository attachments), up to 4 concurrently, capped at 100 MiB
   per attachment.
3. A successful fetch is saved under `{number}/assets/{filename}`, where
   `filename` is a `valueobjects.AssetFilename` built from the UUID GitHub
   assigns in the URL path plus an extension resolved from the response's
   `Content-Type` header (via an explicit, hermetic lookup table, not the
   host's own mime database) — the URL path itself does not reliably
   encode one. An unrecognized content type yields no extension.
   `AssetFilename`'s constructor guarantees the result is always a single,
   path-safe segment (rejecting empty, any all-dots value such as `.`,
   `..`, or `...` — optionally followed by trailing spaces — a path
   separator, or an absolute path), so `AttachmentWriter.WriteAsset` can
   trust any value of this type without re-validating it itself.
4. A failed fetch (broken link, access denied, network error) does not fail
   the export: the reference is rewritten to an inline placeholder noting
   the original URL and failure reason, and the run's failures are
   persisted to `{number}/evidence/fetch-errors.log`. That log is written
   unconditionally (an all-succeeded or no-attachments run removes any
   stale log left by a prior failing run).
5. A context cancellation/deadline during attachment fetching is not
   treated as a per-attachment failure — it aborts the whole export, the
   same as any other fetch step.

## Rate limiting and retry

REST API calls (issue/PR resource, timeline, pull request resource, review
comments) retry on a 429, or a 403 whose headers identify it as rate
limiting (`Retry-After` present, or `X-RateLimit-Remaining: 0` — a 403
without either is a permission error and is not retried). Wait duration
honors `Retry-After` (seconds) or `X-RateLimit-Reset` (epoch seconds) when
present and parseable as a non-negative value that does not overflow
`time.Duration`/`time.Unix`'s arithmetic; an absent, negative, or
implausibly large value for either header falls back to fixed exponential
backoff instead of erroring. Retry ceiling:
3 attempts total, then the error surfaces to the caller. There is no
proactive `GET /rate_limit` check before a run.

Attachment fetches (a separate rate-limit domain, the CDN rather than the
REST API) have no retry/backoff — a fetch failure there goes straight to
the attachment-fetch-failure path described above.

Paginated timeline/review-comment fetches follow the `Link` response
header's `rel="next"` relation, but only when its origin (scheme and host
together, compared case-insensitively — both are themselves
case-insensitive) matches the origin the current page was actually fetched
from; a `next` URL naming a different host, or the same host under a
different scheme (an `https`-to-`http` downgrade), is refused with an
error instead of followed. This guards against a compromised,
misconfigured, or proxy-broken host (including a GitHub Enterprise Server
host) redirecting gh-exhibit's next request somewhere else, or downgrading
it to an
unencrypted connection.

Independently of that check, every REST API request also refuses to follow
an HTTP redirect (a `3xx` response) whose target origin differs from the
origin the redirected request was itself sent to. This closes a gap the
pagination-origin check above does not cover on its own: that check trusts
the origin a response's own request actually reached, but an in-flight
redirect on the very first page (or on a non-paginated fetch, neither of
which the pagination check applies to at all) would otherwise let a
compromised or misconfigured host redirect gh-exhibit to an
attacker-controlled origin before any origin has been recorded to check
against. The redirect guard is enforced one layer below the HTTP client's
request/response handling (as the client's own transport), so a
cross-origin redirect is refused before the redirected request is ever
sent, not merely detected afterward.

Attachment fetches are deliberately exempt from this redirect-origin
guard: a real attachment URL (e.g. `github.com/user-attachments/assets/`)
legitimately redirects cross-origin to serve its bytes (e.g. to a signed,
time-limited S3 URL), so pinning the origin there would reject every such
fetch rather than only a malicious one. This stays safe without the guard
because `net/http` itself strips the `Authorization`/`Cookie` headers on a
redirect whose host differs from the original request's, so the
credential gh-exhibit's client attaches never reaches the redirect target;
the existing response-size cap still bounds how much of whatever that
target returns is read into memory.

Both checks share a fail-closed default: an unknown or indeterminate
expected origin is treated as a mismatch rather than trusted.

## Concurrency

`ExportService.Export` runs `FetchTimeline` concurrently with the
pull-request chain (`FetchPullRequest`, then, only on success,
`FetchReviewComments`) — the chain's own internal short-circuit is
unaffected. Both share a cancellable context: whichever branch fails first
cancels it, so the other branch's in-flight fetch (possibly blocked in a
rate-limit wait up to an hour long) is interrupted rather than waited out.
The first genuine failure to occur is what `Export` reports, not whichever
branch happens to observe cancellation first.

Attachment fetches run concurrently, bounded at 4 in flight
(`maxConcurrentAttachmentFetches`).

## Package layout

Onion architecture, following this project's own reference layout:

- `internal/domain/valueobjects` — the four Tier 1 entry types and their
  supporting Value Objects (`Attribution`, `Url`, `ReviewState`,
  `InlineContext`, `IssueRef`, `Document`, `Provenance`, `AssetFilename`).
  No I/O.
- `internal/domain/services` — stateless domain transformations: timeline
  classification/joining (`classify.go`, `join.go`, `body.go`), and
  attachment detection/rewriting (`attachment.go`, `resolution.go`,
  `filename.go`, `rewrite.go`). No I/O.
- `internal/domain/repositories` — abstract ports the application layer
  depends on: `EvidenceFetcher`, `EvidenceWriter`, `ProvenanceWriter`,
  `DocumentWriter`, `AttachmentFetcher`, `AttachmentWriter`.
- `internal/infrastructure/github` — `go-gh`-backed implementations of
  `EvidenceFetcher`/`AttachmentFetcher`, plus retry/pagination.
- `internal/infrastructure/persistence` — local-filesystem implementations
  of `EvidenceWriter`/`ProvenanceWriter`/`DocumentWriter`/`AttachmentWriter`.
- `internal/application/services` — `ExportService`, orchestrating the
  ports above into one `Export` call. Distinct from
  `internal/domain/services` despite the shared base package name.
- `internal/presentation/cli` — argument parsing, repository resolution,
  and the export-loop driver (`ParseArgs`, `ResolveRepo`, `RunExports`).
- `internal/registry` — the dependency-injection root wiring
  infrastructure-layer constructors into `services.NewExportService`.
- `cmd/gh-exhibit` — the thin `main` composition root.

Every infrastructure-layer concrete type is unexported; its constructor
returns the `domain/repositories` interface type, so no caller can depend on
the concrete type directly.

## Coverage targets

- Domain layer (`valueobjects`, `services` under `domain/`): C0 90% / C1
  75-80% floor. C0 is CI-gated; C1 is reviewed manually via
  `go tool cover -html`, since Go's toolchain has no single-metric C1
  report.
- Boundary layer (`infrastructure/github`, `infrastructure/persistence`):
  qualitative branch coverage via Detroit-school tests (real `t.TempDir()`,
  real `httptest.Server`), no numeric floor.
- Mocks are used only at external boundaries (network, filesystem); domain
  objects collaborate via real in-memory values in every other test.

## Error messages and logging

No error message or `SkipNote.Reason` carries a Go package or
onion-architecture-layer name as a prefix — every prefix that once existed
(`entry:`, `timeline:`, `services:`, etc.) was removed once it was
confirmed that every one of them eventually reaches either an end user's
terminal (`RunExports`'s stderr output) or a maintainer with no notion of
which internal package produced it; a self-descriptive operation phrase
(e.g. `"issue resource attribution: attribution author must not be empty"`)
carries the same information without the jargon.
