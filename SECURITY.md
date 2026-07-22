# Security Policy

## Supported Versions

Only the latest release is supported with security fixes. As this project is pre-1.0, no long-term support window is guaranteed for older releases.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Use GitHub's [private vulnerability reporting][private-report] feature, or email <connect0459@gmail.com>, to disclose issues confidentially. You can expect an acknowledgement within 7 days and a status update within 30 days.

**What to include:**

- A minimal, reproducible example (a sample issue/PR body, timeline payload, or CLI invocation) that triggers the issue.
- The version of `gh-exhibit` and OS/architecture.
- A description of the impact.

[private-report]: https://github.com/connect0459/gh-exhibit/security/advisories/new

## Scope

`gh-exhibit` fetches GitHub issue/PR content (via the authenticated `gh` credential) and writes it to the local filesystem. The following classes of issues are in scope:

- **Credential exposure via attachment fetching** — any input (a crafted issue/PR body, comment, or review) that causes `gh-exhibit` to send its authenticated GitHub request to a host other than the target repository's own attachment host, or that otherwise leaks the resolved `gh` token to an unintended destination.
- **Path traversal in output paths** — any `--repo`, issue/PR number, or fetched-content value that causes `gh-exhibit` to read or write outside the intended `{repo}/{number}` layout under the configured output directory.
- **Resource exhaustion** — an attacker-controlled response (timeline page, attachment body) that causes unbounded memory growth, an unbounded pagination loop, or unbounded disk usage.
- **Panics/crashes on malformed input** — any malformed or adversarial GitHub API response that causes a panic instead of a handled error or a per-item skip.

The following are **out of scope**:

- Vulnerabilities in the `gh` CLI itself, `go-gh`, or GitHub's own API — report those to their respective maintainers.
- The rendered Markdown intentionally reproduces GitHub content verbatim (this tool's audit-trail purpose); content-based risks in how a *reader* chooses to view that Markdown are the reader's own tooling's responsibility, not `gh-exhibit`'s.
- Issues that require the local operator to already have write access to the output directory beyond what `gh-exhibit` itself would write.

## Disclosure Policy

Once a fix is released, a GitHub Security Advisory will be published with full details. The typical timeline from report to public disclosure is 30 days, though this may be extended by mutual agreement when a fix requires significant changes.
