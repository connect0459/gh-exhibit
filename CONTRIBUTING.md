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

To try the extension locally from your checkout:

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
| `pre-commit run golangci-lint --all-files` | Lint (same config CI uses; no separate golangci-lint install needed) |
| `go build ./...` | Build |
| `go test ./... -race -cover` | Run all tests with the race detector |

The pre-commit hooks enforce formatting and lint checks on every commit, and `go test` also runs on every commit that touches a `.go` file.

## Testing guidelines

This project follows **Red → Green → Refactor** (Detroit-school TDD):

- Write a failing test first, then implement.
- Use real objects; mocks are permitted only at external boundaries (file system, network/HTTP).
- Name each test with the **exported** unit it exercises as a prefix (`Test<ExportedUnit>_<BehaviorDescription>`). Test an unexported unit through the exported entry point that calls it, not directly, unless a comment explains why the exported entry point cannot reach that behavior.
- Error messages describe a concrete operation or state, not the name of the function or method that produced them.

Coverage targets (see `docs/specs/README.md`): domain layer (`internal/domain/...`) has a C0 90% floor; boundary layer (`internal/infrastructure/...`, HTTP/file I/O) is judged by qualitative branch coverage using real fixtures (`httptest.Server`, `t.TempDir()`), with no fixed numeric floor.

## Commit format

```text
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `tidy`, `test`, `chore`, `ci`, `perf`

**Scope**: the package name when the change targets a specific package (e.g. `cli`, `github`, `persistence`); omit it for project-wide changes. When a change falls entirely within a domain (`docs`, `style`, `test`, `chore`, `ci`, `tidy`), use the matching domain-axis type as the `type` rather than as the `scope` of an impact-axis type.

**Subject**: imperative mood ("add", "fix", "remove"), 72 characters max, no trailing period.

**Body** (optional): wrap at 72 characters; explain **why**, not what, since the diff already shows what changed.

**Footer** (optional): `BREAKING CHANGE: <description>`, or `Closes <https://github.com/owner/repo/issues/123>` to link issues.

## Pull request process

1. Fork the repository and create a branch: `feat/xxx`, `fix/xxx`, `docs/xxx`.
2. Follow the Red → Green → Refactor cycle; keep one concern per commit.
3. Run `go test ./... -race -cover` and `pre-commit run --all-files`, and make sure both pass.
4. Open a pull request using the repository's PR template — CI (`gofmt`, `go vet`, `golangci-lint`, `go build`, `go test -race -cover`) runs automatically.

Do not edit `CHANGELOG.md` in regular pull requests. Changelog entries are written together in the Release PR.

## Code style

- Onion architecture: `internal/domain` defines abstract types (ports, value objects); `internal/infrastructure` implements them; `internal/application` orchestrates across layers; `internal/presentation` is the CLI entrypoint. No layer depends directly on a concrete type from a layer it doesn't own.
- Rich domain objects: pair data and logic in the same type; prefer immutability.
- Avoid code comments unless the **why** is genuinely non-obvious; let tests document behavior. Exported functions and methods carry a one-line Godoc.
- Don't cite `docs/specs/` paths in code comments. `docs/specs/README.md` is the single, always-current reference for behavior (linked from the Documentation section of `README.md`). A comment repeating that pointer is redundant, breaks when a path inside `docs/specs/` moves, and carries no WHY a reader couldn't get from the doc itself.
- All identifiers, test names, error messages, and documentation must be in **English** (see `AGENTS.md`).
