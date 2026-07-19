# Contributing

## Prerequisites

- [Go](https://go.dev/dl/) — version pinned in `go.mod`
- [gh CLI](https://cli.github.com/) — this project is a `gh` extension
- [just](https://just.systems/) — task runner
- [pre-commit](https://pre-commit.com/) — hook runner

## Setup

```sh
git clone https://github.com/connect0459/gh-exhibit
cd gh-exhibit
just setup
```

`just setup` installs the pre-commit hooks (`pre-commit install`).

To run all hooks manually:

```sh
pre-commit run --all-files
```

To try the extension locally against your own checkout:

```sh
go build ./cmd/gh-exhibit
gh extension install .
```

## Development workflow

This mirrors the CI pipeline (`.github/workflows/ci.yml`), in order:

| Command | Purpose |
| :--- | :--- |
| `gofmt -l .` | Check formatting (empty output = clean) |
| `go vet ./...` | Static analysis |
| `golangci-lint run` | Lint (same config CI and pre-commit use) |
| `go build ./...` | Build |
| `go test ./... -race -cover` | Run all tests with the race detector |

The pre-commit hooks enforce formatting and lint checks on every commit;
`go test` also runs on every commit that touches a `.go` file.

## Testing guidelines

This project follows **Red → Green → Refactor** (Detroit-school TDD):

- Write a failing test first, then implement.
- Use real objects; mocks are only permitted at external boundaries (file
  system, network/HTTP).
- A test names the **exported** unit it exercises as its prefix
  (`Test<ExportedUnit>_<BehaviorDescription>`) — an unexported unit's
  correctness is demonstrated through the exported entry point that calls
  it, not by testing the unexported unit directly, unless a comment
  documents why the exported entry point cannot reach that behavior.
- Error messages describe a concrete operation or state, not the name of
  the function/method that produced them.

Coverage targets (see `docs/adrs/adr-002-language-and-domain-design.md`):
domain layer (`internal/domain/...`) — C0 90% floor; boundary layer
(`internal/infrastructure/...`, HTTP/file I/O) — qualitative branch coverage
via real fixtures (`httptest.Server`, `t.TempDir()`), no fixed numeric floor.

## Commit format

```text
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `tidy`, `test`, `chore`, `ci`, `perf`

**Scope**: package name when the change targets a specific package (e.g.
`cli`, `github`, `persistence`); omit for project-wide changes. Use a
domain-axis type (`docs`, `style`, `test`, `chore`, `ci`, `tidy`) as `type`
when the change is fully contained within that domain, rather than as
`scope` on an impact-axis type.

**Subject**: imperative mood ("add", "fix", "remove"), 72 characters max, no
trailing period.

**Body** (optional): wrap at 72 characters; explain **why**, not what — the
diff already shows what changed.

**Footer** (optional): `BREAKING CHANGE: <description>`, or `Closes #123` /
`Fixes #456` to link issues.

## Pull request process

1. Fork the repository and create a branch: `feat/xxx`, `fix/xxx`, `docs/xxx`.
2. Follow the Red → Green → Refactor cycle; keep one concern per commit.
3. Run `go test ./... -race -cover` and `pre-commit run --all-files`, and
   ensure both pass.
4. Open a pull request using the repository's PR template — CI
   (`gofmt`, `go vet`, `golangci-lint`, `go build`, `go test -race -cover`)
   runs automatically.

## Code style

- Onion architecture: `internal/domain` defines abstract types (ports,
  value objects); `internal/infrastructure` implements them;
  `internal/application` orchestrates across layers; `internal/presentation`
  is the CLI entrypoint. No layer depends directly on a concrete type from
  a layer it doesn't own.
- Rich domain objects: pair data and logic in the same type; prefer
  immutability.
- Avoid code comments unless the **why** is genuinely non-obvious — let
  tests document behavior. Exported functions/methods carry a one-line
  Godoc.
- All identifiers, test names, error messages, and documentation must be in
  **English** (see `AGENTS.md`).
