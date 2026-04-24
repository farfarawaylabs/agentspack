# Sandboxes

## Purpose

Use Cloudflare Sandboxes when an agent needs a persistent isolated environment with filesystem, shell, terminal, code execution, previews, snapshots, or a computer-like workspace.

## Use Sandboxes for

- coding agents
- generated app previews
- test execution
- terminal/shell commands
- file manipulation
- isolated tool execution
- long-running development environments
- credentials injected into isolated environments

## Do not use Sandboxes for

- simple deterministic computations that fit in Workers
- pure HTTP routing
- small stateless transformations

## Security rules

- Treat sandbox contents and code as untrusted.
- Use scoped credentials.
- Prefer temporary credentials.
- Snapshot and audit important changes.
- Do not leak secrets into logs or generated files.

## Source docs

- https://blog.cloudflare.com/sandbox-ga/
