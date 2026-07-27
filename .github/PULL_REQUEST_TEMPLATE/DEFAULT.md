<!-- # PULL_REQUEST_TEMPLATE -->

<!-- Remove unnecessary sections to keep the review focused -->

## Related Links

- Issues
  - <!-- <https://github.com/connect0459/gh-exhibit/issues/xxx> -->
- PRs
  - <!-- <https://github.com/connect0459/gh-exhibit/pull/xxx> -->

## [Required] Overview

- Describe the problem being solved, its background, and what changes when this PR is merged.
- Links to `docs/specs/README.md`, `docs/todo.md` entries, or other references are welcome.

```txt
It is difficult to review without knowing the specifications and background.
```

## Scope of Change

- [ ] `internal/domain` package(s)
- [ ] `internal/infrastructure` package(s)
- [ ] `internal/application` package(s)
- [ ] `internal/presentation` package(s)
- [ ] `cmd/gh-exhibit`
- [ ] Tooling / CI
- [ ] Documentation (`docs/todo.md`, `docs/specs/README.md`, README)

## Breaking Changes

- [ ] No breaking changes
- [ ] Breaking changes (describe below)

<!--
If this changes a public API or the on-disk output layout (docs/specs/README.md), describe what breaks and why the breakage is justified, and update docs/specs/README.md to match.
-->

## Deferred Items and TODOs

- Items intentionally deferred and the reasons why.

```txt
If you deferred something due to time constraints, document it here.
Reviewers cannot tell whether something was intentionally skipped or overlooked
without this information.
```

## Test Items

- Describe the tests added, following Red/Green TDD (which test was written first, and what it confirmed failed before the implementation existed).
- Note per-package C0 coverage if it changed meaningfully (see `docs/specs/README.md` for this project's coverage targets).
- Confirm `go build ./...`, `go vet ./...`, and `go test ./...` all pass with no regressions.

## [Required] Quality Checklist

**Please check all items before merging.**

- [ ] **CI Workflow Execution**: All checks passed on the [CI workflow](../actions/workflows/ci.yml) for this PR.
- [ ] **Code Comments**: Limited to Godoc and non-obvious WHY/WHY-NOT explanations, per this project's comment policy.
- [ ] **Reference Docs**: `docs/todo.md` updated to check off completed items and record any design decisions made along the way.

> **Important**: This checklist ensures quality. Please verify all items before requesting review.
