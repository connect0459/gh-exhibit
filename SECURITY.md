# Security Policy

## Supported Versions

Only the latest release receives security fixes. Because this project is pre-1.0, older releases have no guaranteed support window.

## Reporting a Vulnerability

**Please do not open a public GitHub issue for security vulnerabilities.**

Use GitHub's [private vulnerability reporting][private-report] feature, or email <connect0459@gmail.com>, to report issues confidentially. You can expect an acknowledgment within 7 days and a status update within 30 days.

**What to include:**

- A minimal reproducible example (a sample issue/PR body, timeline payload, or CLI invocation) that triggers the issue.
- The `gh-exhibit` version and your OS and architecture.
- A description of the impact.

[private-report]: https://github.com/connect0459/gh-exhibit/security/advisories/new

## Scope

`gh-exhibit` fetches GitHub issue/PR content (using the authenticated `gh` credential) and writes it to the local filesystem. The following classes of issues are in scope:

- **Credential exposure via attachment fetching**: any input (a crafted issue/PR body, comment, or review) that causes `gh-exhibit` to send its authenticated GitHub request to a host other than the target repository's own attachment host, or that otherwise leaks the resolved `gh` token to an unintended destination.
- **Path traversal in output paths**: any `--repo`, issue/PR number, or fetched-content value that causes `gh-exhibit` to read or write outside the intended `{repo}/{number}` layout under the configured output directory.
- **Resource exhaustion**: an attacker-controlled response (timeline page, attachment body) that causes unbounded memory growth, an unbounded pagination loop, or unbounded disk usage.
- **Panics or crashes on malformed input**: any malformed or adversarial GitHub API response that causes a panic instead of a handled error or a per-item skip.

The following are **out of scope**:

- Vulnerabilities in the `gh` CLI itself, `go-gh`, or GitHub's own API — report those to their respective maintainers.
- Content-based risks when viewing the rendered Markdown. It intentionally reproduces GitHub content verbatim (the tool's audit-trail purpose), so how a *reader* views it is the responsibility of the reader's own tooling, not `gh-exhibit`.
- Issues that require the local operator to already have write access to the output directory beyond what `gh-exhibit` itself writes.
- **Content substitution via an attachment fetch's legitimate cross-origin redirect.** A real GitHub attachment URL (`.../user-attachments/assets/...`) legitimately redirects cross-origin to serve its bytes (e.g. to a signed, time-limited S3 URL). Attachment fetches therefore deliberately follow a redirect to any external hostname or public IP address, once the address the hostname actually resolves to is confirmed not to be a loopback, link-local, or private-network address (including a cloud-metadata endpoint like `169.254.169.254`). That check happens at the exact moment of connecting, not in an earlier, separate lookup, which closes the DNS-rebinding TOCTOU gap a check-then-dial design would leave open. See `internal/infrastructure/github/attachment_redirect_guard.go`. A compromised or misconfigured host can therefore still substitute an attachment's content by redirecting to a different *external* host it controls. This is an accepted, currently unmitigated trade-off, not a gap this project intends to close with an allowlist of known hosts, which would break attachment delivery from a self-hosted GitHub Enterprise Server instance's arbitrary storage host. No credential is exposed to that host either way: `net/http` strips the `Authorization` and `Cookie` headers on any cross-host redirect, regardless of the target's address.

## Disclosure Policy

Once a fix is released, a GitHub Security Advisory will be published with full details. The typical timeline from report to public disclosure is 30 days, which may be extended by mutual agreement when a fix requires significant changes.
