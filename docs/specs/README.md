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
gh exhibit export <number>[,<number>...] [--repo <owner>/<repo>] [-o|--output <dir>]
gh exhibit --version
```

- Root level: only `--version` and the automatic `-h`/`--help` are
  recognized. A missing or unrecognized subcommand is an error (`export` is
  currently the only one defined). This grammar separates verbs — operations
  on data, expressed as bare subcommands — from meta-queries about the tool
  itself, following the convention `git`/`docker`/`kubectl`/`gh` already use,
  ahead of a second subcommand being added later.
- `export`'s positional argument: a single issue/PR number, or a
  comma-separated list of them (deduplicated, first-seen order). No range or
  `--all` syntax. Flags may appear before, after, or interleaved around it.
- `export --repo owner/repo`: target repository; defaults to the current
  directory's repository context (`go-gh`'s `repository.Current`) when
  omitted.
- `export -o`, `export --output`: output directory the evidence is written
  under; defaults to `.`.
- `--version`: prints `gh-exhibit {version} (commit {commit}, built {date})`
  and exits, without requiring a subcommand.
- A failing ref in a list does not stop the remaining ones in the same
  invocation. Each ref's success or failure is reported on its own line to
  stdout/stderr respectively. Process exit code is `0` only if every ref
  succeeded, `1` otherwise.

## Domain model

### Tier 1 entries

The rendered Markdown's content is drawn from fourteen entry types, all
Value Objects (no identity-based tracking — re-fetch diffing is git's job
in the separate repository the evidence is copied into, not gh-exhibit's):

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
- `ClosureEvent` — the issue/PR being closed or reopened, sourced from the
  timeline's `closed`/`reopened` events, carrying a `ClosureAction`
  (`closed` / `reopened`) plus an optional reason (GitHub's `state_reason`,
  populated only for a `closed` action).
- `RenameEvent` — the issue/PR's title being changed, sourced from the
  timeline's `renamed` event, carrying the title's previous (`from`) and
  new (`to`) values.
- `MilestoneEvent` — a milestone added to or removed from the issue/PR,
  sourced from the timeline's `milestoned`/`demilestoned` events, carrying
  a `MilestoneAction` (`milestoned` / `demilestoned`) plus the affected
  milestone's title.
- `AssignmentEvent` — an assignee added to or removed from the issue/PR,
  sourced from the timeline's `assigned`/`unassigned` events, carrying an
  `AssignmentAction` (`assigned` / `unassigned`) plus the affected
  assignee's login.
- `PullRequestDiff` — a pull request's changed files (PRs only), sourced
  from `GET /pulls/{number}/files`, carrying a list of `ChangedFile`
  (filename, optional previous filename for a rename, `FileStatus`,
  additions, deletions, and an optional unified diff patch) plus the pull
  request's total additions/deletions and a `truncated` flag. See
  "Pull request diff and commit list rendering" below.
- `PullRequestCommits` — a pull request's commit list (PRs only), sourced
  from `GET /pulls/{number}/commits`, carrying a list of `Commit` (sha, git
  author name and authored time, git committer name and committed time, and
  full message). See "Pull request diff and commit list rendering" below.
- `ParentIssue` — the issue this issue is a sub-issue of (plain issues
  only, and only when one exists), sourced from the issue resource's own
  `parent_issue_url` refetched via `FetchIssue`, carrying a single
  `IssueSummary`. See "Parent issue and sub-issue rendering" below.
- `SubIssues` — this issue's list of sub-issues (plain issues only, and
  only when at least one exists), sourced from `GET
  /issues/{number}/sub_issues`, carrying a list of `IssueSummary`. See
  "Parent issue and sub-issue rendering" below.
- `PullRequestChecks` — the check runs associated with a pull request's
  head commit (PRs only, and only when at least one check run exists),
  sourced from `GET /repos/{owner}/{repo}/commits/{sha}/check-runs`,
  carrying a list of `CheckRun` (name, `CheckOutcome`, url) plus the head
  commit sha and a `capturedAt` timestamp. See "Pull request check-run
  rendering" below.

All fourteen implement the sealed `valueobjects.Entry` interface
(`Render(io.Writer) error` plus an unexported marker method) — the closest
Go analogue to a closed sum type. Supporting Value Objects: `Attribution`
(author, created, url — the common `<!-- {"meta":...} -->` fields), `Url`
(an absolute http/https URL, parsed and validated once at construction),
`IssueRef` (owner, repo, number — validated against GitHub's own username/
repository-name character-set and length rules; `repo` additionally
rejects an all-dots segment such as `.`, `..`, or `...`, optionally
followed by trailing spaces, via the same check `AssetFilename` uses —
see "Attachment policy" below), `IssueSummary` (number, title, `IssueState`,
url — a lightweight reference to a related issue, distinct from `IssueRef`
in that it is display data already resolved from a fetched resource rather
than an address used to fetch one), `CheckRun` (name, `CheckOutcome`, url
— one check run), `CheckOutcome` (an enum unifying a check run's `status`
before it completes with its `conclusion` once it does — see "Pull request
check-run rendering" below), `Provenance` (tool,
version, commit — which gh-exhibit build produced an export, persisted to
`evidence/provenance.json` by `ProvenanceWriter` rather than rendered into
`Document`), `AssetFilename` (a downloaded attachment's on-disk filename,
guaranteed by its constructor to be a single path-safe segment — see
"Attachment policy" below).

### Timeline classification

Seven of the fourteen Tier 1 types (`IssueComment`, `PullRequestReview`,
`LabelEvent`, `ClosureEvent`, `RenameEvent`, `MilestoneEvent`, and
`AssignmentEvent`) are classified from `GET .../issues/{number}/timeline`'s
heterogeneous array via a two-pass unmarshal (discriminator peek, then
dispatch), checked for exhaustiveness by the `exhaustive` golangci-lint
rule against `eventKind`. The timeline array also carries other event
kinds with no corresponding Tier 1 type (e.g. `review_requested`,
`cross-referenced`), which are left unclassified rather than causing an
error.
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

### Label and other history-event rendering

A label added to or removed from the issue/PR is rendered as a `LabelEvent`
interleaved chronologically with the rest of the timeline (a `labeled` or
`unlabeled` event, classified the same way as `commented`/`reviewed`),
rather than as a static snapshot of the current label set placed near the
document's top. This was a deliberate choice over the static-list
alternative: it stays consistent with how every other entry in the document
already presents content in the order it happened, at the cost of needing to
recognize the corresponding timeline event kind rather than only reading the
issue/PR resource's own `labels` field. `ClosureEvent`, `RenameEvent`,
`MilestoneEvent`, and `AssignmentEvent` follow the identical
interleaved-by-timeline-position approach, for the same reason.

Unlike `commented`/`reviewed` events, GitHub's `labeled`/`unlabeled`,
`closed`/`reopened`, `renamed`, `milestoned`/`demilestoned`, and
`assigned`/`unassigned` timeline payloads carry no per-event `html_url` —
only an API URL (`.../issues/events/{id}`) that doesn't resolve to a
human-readable page. Each of these five Tier 1 types' attribution
therefore falls back to the issue/PR's own `html_url` (already available
from the issue/PR resource fetched alongside the timeline) rather than a
link to the specific historical event.

### Pull request diff and commit list rendering

Unlike the nine timeline-classified/`Body` Tier 1 types, `PullRequestDiff`
and `PullRequestCommits` have no timeline event or per-event actor of their
own — each is a snapshot of the pull request's current state (its changed
files, or its commit list), not something that happened at a point in
time (`ParentIssue`/`SubIssues`, described further below, share this same
snapshot property for a plain issue's parent/children). Both therefore
reuse the pull request's own `Attribution` (author,
created, url) — the same `Attribution` `Body` was built from — rather than
a per-event one, the same "fall back to the resource's own attribution"
precedent `LabelEvent`/`ClosureEvent`/`RenameEvent`/`MilestoneEvent`/
`AssignmentEvent` already established for a different reason (their event
payload's missing `html_url`). Because they are snapshots rather than
chronological events, both are placed once, immediately after `Body` —
`PullRequestDiff` first, then `PullRequestCommits` — rather than
interleaved by timeline position; both are present only when the exported
ref is a pull request. This shared placement and attribution-reuse
decision is deliberately made once and applied to both entries, rather
than each choosing its own convention.

The pull request resource's own `additions`/`deletions` fields (already
fetched as part of building `Body`, no extra request needed) decide
whether each changed file's patch is rendered: once their total exceeds
`maxDiffTotalLines` (1000 changed lines), every file's patch is suppressed
while the file list itself (filename, status, additions, deletions) is
still rendered in full — the "list of changed files" fallback. A file's
own patch can also be empty below that threshold, when GitHub itself
omits `patch` for an individually oversized file; `PullRequestDiff`'s
render logic does not distinguish the two cases (it renders a diff block
only when a file's `Patch()` is non-empty), but its `truncated` flag,
carried in the meta line, records only the threshold case.

`PullRequestCommits` carries one `Commit` per element of `GET
/pulls/{number}/commits`: its git-level author name and authored timestamp
(`commit.author`), its git-level committer name and committed timestamp
(`commit.committer`), and its full message. Author and committer are kept
as two distinct identities because they can genuinely differ — for
example, GitHub's web UI or a rebase/squash operation re-commits an
existing author's work under a different committer. Unlike
`PullRequestDiff`'s patch, no size threshold suppresses a commit's message:
every commit's full message is always rendered in full under a
"**Commit `sha`**" label, the same per-item expansion shape
`PullRequestDiff` uses for a file's diff hunk.

### Parent issue and sub-issue rendering

GitHub's sub-issue (parent/child issue) relationship is a plain-issue-only
feature: a pull request's own resource never carries a `parent_issue_url`,
and its `GET /issues/{number}/sub_issues` always returns an empty list
(verified directly against a merged pull request). `ParentIssue` and
`SubIssues` are therefore only ever fetched, built, and rendered when the
exported ref is a plain issue — the opposite gating condition from
`PullRequestDiff`/`PullRequestCommits`, which are PR-only.

Like `PullRequestDiff`/`PullRequestCommits`, both are snapshots with no
timeline event or per-event actor of their own, so both reuse the issue's
own `Attribution` — the same one `Body` was built from — rather than a
per-event one. Unlike the two PR-only entries, each is added to the
document only when it actually has content: `ParentIssue` is present only
when the issue resource's own `parent_issue_url` is populated; `SubIssues`
is present only when `GET /issues/{number}/sub_issues` returns at least one
item. This is a deliberate difference from `PullRequestDiff`/
`PullRequestCommits`, whose presence is gated purely by ref kind (a PR
always gets both, even one with an empty commit list) — here, presence is
additionally gated by whether the issue actually has a parent or children,
since most issues have neither and a permanently-empty entry on every
plain issue export would be pure noise.

`ParentIssue`'s parent is not the issue resource's own `parent_issue_url`
value — that field only names the parent's API URL — but a second
`FetchIssue` call against the `IssueRef` parsed out of it, giving
`ParentIssue` the parent's title, state, and URL the same way `Body`
itself is built. Parsing that URL only checks the path's trailing
`repos/{owner}/{repo}/issues/{number}` segments rather than the path's
whole shape, since a GitHub Enterprise Server host serves its REST API
under an additional `/api/v3/` prefix (matching `go-gh`'s own outgoing
request routing), giving a GHES-origin `parent_issue_url` two more leading
path segments than a `github.com`-origin one. `SubIssues`' children need
no such second fetch: `GET
/issues/{number}/sub_issues` already returns each child's full resource
(number, title, state, `html_url`), the same shape `ParentIssue`'s second
fetch produces. Both are modeled as `IssueSummary` (number, title,
`IssueState`, url) — a lightweight reference distinct from `IssueRef`,
which addresses an issue for fetching rather than displaying it. A
sub-issue's own "completion status" is simply its own `IssueSummary.State`
(open/closed), read directly off the fetched child rather than from the
issue resource's separate `sub_issues_summary` total/completed counter,
so the rendered list can never disagree with itself about which children
are done.

`ParentIssue` renders as a meta-line-only entry (no separate body content),
the same shape `LabelEvent` uses for a single self-contained fact — its
meta line already carries the parent's number, title, state, and url in
full. `SubIssues` renders a `sub_issues` count in its meta line plus a
bullet list of every child, the same count-in-meta-plus-per-item-list
shape `PullRequestCommits` uses for its own list of commits. Each bullet
(`issueSummaryLine`) reuses the title-first, backtick-wrapped,
linked-number shape "Issue/PR reference linking" below established for a
bare issue/PR reference — `` `{title}` [#{number}](url) ({state}) ``,
e.g. `` `Include issue/PR labels` [#65](https://github.com/example/repo/issues/65) (closed) ``
— rather than its own earlier `` `#{number}` {title} ({state}) `` shape,
which left title as unlinked plain prose. The backtick-fencing technique
(`titleCodeSpan`/`longestBacktickRun`) is duplicated in `valueobjects`
rather than shared with `services`' own copy: `services` already depends
on `valueobjects`, so the reverse dependency sharing would need is
unavailable, and this project prefers duplication over a premature
cross-package abstraction for a handful of similar lines.

### Pull request check-run rendering

Unlike every other Tier 1 entry, the content `PullRequestChecks` captures
can keep changing after the export is taken — a check can be re-run, or a
pending one can later resolve — so it is a snapshot of something that may
still be moving, not a fixed fact the way a posted comment or review is.
This is addressed by recording an explicit `captured_at` timestamp (the
wall-clock time the check-run fetch happened, resolved once per `Export`
call via a `repositories.Clock` port so it stays substitutable in a test)
in the meta line, alongside — not instead of — `PullRequestDiff`/
`PullRequestCommits`/`ParentIssue`/`SubIssues`' existing `created`
(the pull request's own creation time, reused via `Attribution` as usual).
A reader is therefore not left to assume the rendered outcome is still
current: `captured_at` names when it was observed, distinct from `created`.
Whether this is worth capturing for a pull request that is already merged
or closed (where the check state is less likely to change further) is
treated as a secondary question that does not gate this design: every pull
request's check runs are fetched and persisted the same way regardless of
its own merged/closed state.

Check runs are sourced from `GET
/repos/{owner}/{repo}/commits/{sha}/check-runs` against the pull request's
own head commit sha (`pull.head.sha`, resolved from the already-fetched
pull request resource) — the Checks API, not the older combined-status
API (`GET /commits/{sha}/status`), since the Checks API is what GitHub
Actions and most modern third-party CI integrations report through. Like
`PullRequestDiff`/`PullRequestCommits`, `PullRequestChecks` has no
timeline event or per-event actor of its own, so it reuses the pull
request's own `Attribution` (author, created, url). Unlike the two
PR-only snapshot entries, whose presence is gated purely by ref kind, it
is added to the document only when at least one check run exists — the
same content-gated presence `ParentIssue`/`SubIssues` use, since a pull
request with no CI configured would otherwise carry a permanently-empty
entry.

Each `CheckRun`'s displayable outcome unifies GitHub's own two-field
`status`/`conclusion` shape (`conclusion` is populated only once `status`
reaches `completed`) into a single `CheckOutcome` enum: a run not yet
completed reports its status (`queued`, `in_progress`); a completed one
reports its conclusion (`success`, `failure`, `neutral`, `cancelled`,
`skipped`, `timed_out`, `action_required`, `stale`). `PullRequestChecks`
renders a `checks` count, the head sha, and the captured-at timestamp in
its meta line, plus a bullet list of every check run's name and outcome.
A check run's name is arbitrary, attacker-influenceable text (a CI job
name, or a third-party Checks app's own naming), so — unlike this entry's
own meta line, whose fields are safe by construction (`encoding/json`
escaping; see "Markdown dialect" above) — it is rendered as plain
backtick-wrapped text rather than as a `[name](url)` markdown link: a name
containing `]` or `(` embedded in link syntax could otherwise close the
link early and splice in an attacker-chosen URL, the same untrusted-string
handling `changedFileLine`/`commitLine`/`issueSummaryLine` already apply
to a filename/commit-identity/issue-title of their own. Each run's own
`html_url` is therefore not rendered inline — it is still available
verbatim in `evidence/check-runs.json`.

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
    ├── pull-files.json               changed files, from GET /pulls/{number}/files (PRs only)
    ├── pull-commits.json             commit list, from GET /pulls/{number}/commits (PRs only)
    ├── sub-issues.json               sub-issue list, from GET /issues/{number}/sub_issues (plain issues only)
    ├── parent-issue.json             parent issue resource, refetched via FetchIssue (plain issues with a parent only)
    ├── check-runs.json               check runs, from GET /commits/{sha}/check-runs (PRs only)
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
Unlike its eight siblings, `provenance.json`'s content is not a verbatim
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
itself (the raw JSON is). `parent-issue.json` is the one file this
overwrite-every-time rule doesn't literally cover, since its presence can
change between runs of the same ref (an issue's parent can be added or
removed): a rerun that finds no parent removes any file an earlier run
left behind, the same self-healing property applied to a file whose
absence, not just its content, needs to be regenerated.

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

## Issue/PR reference linking

A bare (not already formatted as a link) `#123` or `owner/repo#123`
reference appearing anywhere in the rendered document — an issue/PR body,
a comment, a review, an inline review comment — is rewritten into a link
carrying its target's own title, once that title is resolved. This runs as
a post-render pass over the whole rendered `Document`
(`services.DetectIssueReferences`/`services.RewriteIssueReferences`), the
same shape the attachment policy above already uses
(`services.Detect`/`services.Rewrite`), so no Tier 1 entry needs a
content-mutation path of its own for this either.

Detection excludes: a reference already formatted as a markdown link; one
inside an HTML comment (a Tier 1 entry's own `<!-- {"meta":...} -->`
line); one inside a fenced code block (a diff patch's or commit message's
own verbatim content, which this rewrite must not alter); and one inside
an inline code span (this project's own backtick-wrapped, deliberately
non-linked untrusted-text convention — see `issueSummaryLine`/
`changedFileLine`/`commitLine`/`checkRunLine` above). A reference whose
owner/repo/number fails `valueobjects.IssueRef`'s own validation (e.g.
`#0`, a non-positive number) is silently skipped, the same
skip-and-continue handling this project already applies elsewhere to a
single malformed item.

Each distinct target is fetched at most once via the same
`EvidenceFetcher.FetchIssue` port `ParentIssue`'s own second fetch already
uses, even when the same reference occurs multiple times in the rendered
document — a repeated reference reuses the first fetch's result rather
than refetching. A target that cannot be fetched (deleted, made private, a
transient error) or whose resource fails to parse is left exactly as
originally written: unlike a failed attachment fetch, no placeholder is
substituted, since the reference was already valid, readable text before
this rewrite ran — only the readability improvement is forgone, nothing
about which issue/PR was referenced is lost. A context
cancellation/deadline during this resolution is not treated as an
ordinary per-reference failure — it aborts the whole export, the same as
any other fetch step.

The substitution places the resolved title *before* the link rather than
inside its `[...]` text — `` `{title}` [{original text}](url) ``, e.g.
`` `Fix the thing` [#42](https://github.com/owner/repo/issues/42) `` —
rather than `[{title}](url)`. An issue/PR title is arbitrary,
attacker-influenceable text (anyone can title their own issue), so it is
never embedded inside this rewrite's own constructed link syntax: the
same "untrusted text is never placed inside a `[text](url)` span"
precedent `changedFileLine`/`commitLine`/`issueSummaryLine`/`checkRunLine`
already establish for a filename/commit identity/issue title/check-run
name of their own (see "Pull request check-run rendering" above), applied
here to a link this rewrite constructs itself rather than avoiding a link
altogether. The original matched text (`#123` or `owner/repo#123`,
verbatim as the author wrote it) is reused as the link's own label,
rather than a normalized form.

The title is additionally backtick-wrapped, rather than left as bare
prose: without a delimiter, a title that itself starts with a bracketed
tag (e.g. an issue titled "[Feature] ...", a common convention this
project's own issues use) reads as ambiguous with this rewrite's own
inserted text, with no visual boundary between the two. This matches the
same backtick-wrapped-untrusted-text convention
`issueSummaryLine`/`changedFileLine`/`commitLine`/`checkRunLine` already
use elsewhere, and is safe for the same structural reason placing title
outside the link already is: a backtick inside title only ends its own
code span early, unlike `]`/`(`, it cannot affect this rewrite's own link
destination, which is built entirely from url, never from title. The
fence itself uses a backtick run one character longer than the longest
run already inside title — the same longest-run-plus-one technique
`diffFence` uses for a fenced diff hunk, adapted for an inline span — with
a single padding space added when title starts or ends with a backtick,
so the fence's own delimiter does not merge with title's; CommonMark
strips exactly one leading and trailing space from a code span's content
when it has both, so this padding never appears in the rendered result.

This pass runs *after* the attachment policy's own detect/resolve/rewrite
pass, not before: a referenced issue/PR's own title is text controlled by
whoever titled that other issue — not a participant in the exported
issue/PR's own discussion, and, for a cross-repository reference, not
even someone with any relationship to the exported repository at all.
Were the attachment policy's `Detect` to run over a buffer that already
had such a title spliced in, it would treat any `user-attachments`-shaped
URL embedded in that title as a genuine attachment referenced by the
exported discussion, fetching and downloading content a third party never
actually attached to it. Running attachment resolution first closes this
off structurally rather than by sanitizing the substituted title text:
`Detect` never sees title text, because it does not exist in the buffer
yet at the point it runs.

## Rate limiting and retry

REST API calls (issue/PR resource, timeline, pull request resource, review
comments, pull request files, pull request commits, check runs,
sub-issues, parent issue resource, and a referenced issue/PR resource
resolved from a bare reference detected in the rendered document) retry
on a 429, or a
403 whose headers
identify it as rate limiting (`Retry-After` present, or
`X-RateLimit-Remaining: 0` — a 403 without either is a permission error and
is not retried). Wait duration
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

`ExportService.Export` runs `FetchTimeline` concurrently with a second
branch that depends on whether ref is a pull request or a plain issue: for
a pull request, the chain (`FetchPullRequest`, then, only on success,
`FetchReviewComments`, then `FetchPullRequestFiles`, then
`FetchPullRequestCommits`, then, after resolving the pull request's own
head commit sha, `FetchCheckRuns`); for a plain issue, `FetchSubIssues`, then, only
on success, a second `FetchIssue` for the parent when
`IssueResource.ParentIssueRef` resolves one. Either branch's own internal
short-circuit is unaffected by running concurrently with `FetchTimeline`.
All branches share a cancellable context: whichever fails first cancels
it, so the other branch's in-flight fetch (possibly blocked in a
rate-limit wait up to an hour long) is interrupted rather than waited out.
The first genuine failure to occur is what `Export` reports, not whichever
branch happens to observe cancellation first.

Attachment fetches run concurrently, bounded at 4 in flight
(`maxConcurrentAttachmentFetches`).

## Package layout

Onion architecture, following this project's own reference layout:

- `internal/domain/valueobjects` — the fourteen Tier 1 entry types and their
  supporting Value Objects (`Attribution`, `Url`, `ReviewState`,
  `InlineContext`, `IssueRef`, `IssueSummary`, `IssueState`, `CheckRun`,
  `CheckOutcome`, `Document`, `Provenance`, `AssetFilename`).
  No I/O.
- `internal/domain/services` — stateless domain transformations: timeline
  classification/joining (`classify.go`, `join.go`, `body.go`), attachment
  detection/rewriting (`attachment.go`, `resolution.go`, `filename.go`,
  `rewrite.go`), and issue/PR reference detection/rewriting
  (`issue_reference.go`, `issue_reference_protected.go`,
  `issue_reference_resolution.go`, `issue_reference_rewrite.go`). No I/O.
- `internal/domain/repositories` — abstract ports the application layer
  depends on: `EvidenceFetcher`, `EvidenceWriter`, `ProvenanceWriter`,
  `DocumentWriter`, `AttachmentFetcher`, `AttachmentWriter`, `Clock`.
- `internal/infrastructure/github` — `go-gh`-backed implementations of
  `EvidenceFetcher`/`AttachmentFetcher`, plus retry/pagination.
- `internal/infrastructure/persistence` — local-filesystem implementations
  of `EvidenceWriter`/`ProvenanceWriter`/`DocumentWriter`/`AttachmentWriter`.
- `internal/infrastructure/clock` — a `Clock` implementation backed by the
  operating system's own wall clock.
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
