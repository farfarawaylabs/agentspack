# Security, Auth, and Permissions

## Core rule

Every agent capability is a permission boundary. Treat tools, email, browser, sandbox, dynamic code, and MCP as privileged operations.

## Checklist

- Authenticate the user before connecting to an Agent.
- Authorize access to the specific Agent instance name/ID.
- Validate all callable method inputs.
- Keep secrets out of Agent state if state syncs to clients.
- Use SQL/R2/secret bindings for sensitive data.
- Scope OAuth tokens per user/workspace/provider.
- Redact logs.
- Add idempotency keys for side effects.
- Restrict browser/sandbox/dynamic worker network access where possible.
- Use explicit allowlists for tools and external APIs.

## Callable method security

Callable methods are externally reachable over the client connection. Do not expose methods unless the client should call them. Check authorization and validate arguments.

## Generated code security

Dynamic Workers, Facets, and Sandboxes are powerful. Generated code should receive minimum bindings, minimum network access, and no broad secrets.
