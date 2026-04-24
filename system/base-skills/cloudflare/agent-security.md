---
name: cloudflare-agent-security
description: Security, authentication, authorization, and permission-boundary checklist for Cloudflare Agents — secrets, callable-method auth, scoped OAuth tokens, idempotency for side effects, log redaction, browser/sandbox/dynamic-worker least privilege, tool allowlists. Use when the user asks about securing an agent, auth between the client and agent, keeping secrets out of state, storing OAuth tokens, redacting logs, rate-limiting destructive ops, or review/hardening any agent code path before shipping. Produces an audit-style walkthrough with concrete patterns for each boundary.
---

# Cloudflare Agents — Security, Auth, and Permissions

Use this skill as a review pass on any agent code you're about to ship, and as a design input whenever you add a new capability — callable method, tool, email inbox, browser session, sandbox, dynamic worker. Every agent capability is a **permission boundary**.

## Core rule

Treat **tools, email, browser, sandbox, dynamic code, and MCP** as privileged operations. Each has a matching auth question: *who is allowed to trigger this, what can it reach, and what gets logged?*

## Checklist (run this on every PR)

1. **Authenticate the user** before opening an Agent connection. Never trust "the agent name is a UUID" as auth — it isn't.
2. **Authorize access** to the specific Agent instance name/id. Map `userId → allowedAgentIds`; reject everything else.
3. **Validate every `@callable` input** with Zod/Valibot. TypeScript types don't exist at runtime.
4. **Keep secrets out of `this.state`**. State syncs to every connected client. Use `this.sql`, secret bindings, or external secret stores.
5. **Scope OAuth tokens** per user / workspace / provider. Mint short-TTL tokens; don't share across tenants.
6. **Redact logs** — strip access tokens, emails, PII from anything you `console.log` or offload to R2.
7. **Idempotency keys on side effects** — emails, payments, writes. See `cloudflare-retries-with-idempotency`.
8. **Restrict network access** in browser sessions, sandboxes, and dynamic workers. Use allowlists, not blocklists.
9. **Allowlist tools** the agent can call and URLs it can fetch. "Fetch any URL" is SSRF-as-a-service.
10. **Rate-limit destructive operations** per agent — store counters in `this.sql`.

## Pattern — authenticate + authorize agent access

```ts
import { getAgentByName, routeAgentRequest } from "agents";

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext) {
    const session = await verifySession(request, env);
    if (!session) return new Response("unauthorized", { status: 401 });

    const url = new URL(request.url);
    if (url.pathname.startsWith("/agent/")) {
      const agentName = url.pathname.split("/")[3] ?? "";
      if (!(await canAccessAgent(session.userId, agentName, env))) {
        return new Response("forbidden", { status: 403 });
      }
      const response = await routeAgentRequest(request, env, {
        headers: { "X-User-Id": session.userId },
      });
      if (response) return response;
    }

    return new Response("not found", { status: 404 });
  },
} satisfies ExportedHandler<Env>;

async function canAccessAgent(userId: string, agentName: string, env: Env) {
  const row = await env.DB.prepare(
    "SELECT 1 FROM agent_access WHERE user_id = ?1 AND agent_name = ?2",
  ).bind(userId, agentName).first();
  return !!row;
}
```

The Worker performs auth once, at the edge. The agent receives a trusted `X-User-Id` header and can use it to scope state and queries.

## Pattern — callable-method authorization

```ts
import { Agent, callable } from "agents";

export class ProjectAgent extends Agent<Env, { ownerId: string; members: string[] }> {
  private async assertCanEdit(connection: AgentConnection) {
    const userId = connection.headers.get("X-User-Id");
    if (!userId) throw new Error("unauthenticated");
    if (userId !== this.state.ownerId && !this.state.members.includes(userId)) {
      throw new Error("forbidden");
    }
    return userId;
  }

  @callable()
  async renameProject(raw: unknown, connection: AgentConnection) {
    const userId = await this.assertCanEdit(connection);
    const { name } = RenameProjectInput.parse(raw);
    this.setState({ ...this.state, name });
    await this.audit(userId, "rename", { name });
  }
}
```

Every destructive callable should:
- authorize the caller,
- validate input,
- record an audit entry (SQL row with `who`, `what`, `when`).

## Pattern — secret storage

```ts
// DO: bindings / secret store.
const apiKey = this.env.STRIPE_KEY;

// DO: per-user scoped tokens in SQL (not state).
await this.sql`
  INSERT OR REPLACE INTO oauth_tokens (user_id, provider, token, expires_at)
  VALUES (${userId}, 'github', ${token}, ${expiresAt})
`;

// DON'T: put tokens in this.state — it syncs to every connected client.
// this.setState({ ...this.state, githubToken: token });   // nope
```

## Pattern — log redaction

```ts
function redact<T extends Record<string, unknown>>(obj: T): T {
  const clone: Record<string, unknown> = { ...obj };
  for (const key of Object.keys(clone)) {
    if (/token|secret|password|authorization|cookie/i.test(key)) {
      clone[key] = "[REDACTED]";
    }
  }
  return clone as T;
}

console.log("tool_call", redact({ tool, args, userId }));
```

Do the redact pass before `console.log`, before persisting to `this.sql`, and before offloading to R2.

## Generated-code boundary (Sandboxes, Dynamic Workers, Facets)

Treat all AI-generated code as untrusted. Specifically:

- **Minimum bindings** — only the bindings the generated code needs for its declared purpose.
- **Network allowlist** — explicit list of domains the generated code can reach.
- **No platform secrets** in the generated environment (no `CF_API_TOKEN`, no org-wide keys).
- **Per-task scoped credentials** with short TTL.
- **Supervisor object** outside the sandbox owns lifecycle, policy, and the secret mint flow.
- **Version generated code** with a hash / version id so you can correlate behavior to a known snapshot.

## Rate-limiting destructive ops

```ts
async beforeDestructiveOp(userId: string, op: string) {
  const recent = this.sql<{ n: number }>`
    SELECT COUNT(*) as n FROM audit
    WHERE user_id = ${userId} AND op = ${op} AND ts > ${Date.now() - 60_000}
  `;
  if (recent[0].n > 10) throw new Error("rate_limited");
}
```

Do it per agent + per user, not globally — one noisy user shouldn't block everyone.

## Common mistakes this skill prevents

- Storing OAuth tokens in `this.state` — broadcast to every connected client.
- Assuming `@callable()` is private because it's not in a public Worker route — any connected client can invoke it.
- Logging tool args verbatim — tokens leak into observability.
- Giving a Sandbox / Dynamic Worker access to global secrets or all network — prompt injection turns into data exfiltration.
- Retrying non-idempotent ops with no idempotency key — duplicate payments/emails.
- Skipping audit rows on destructive ops — when something goes wrong, you have no timeline.

## See also

- `.agentspack/docs/cloudflare-agent-stack/16_SECURITY_AUTH_PERMISSIONS.md` — deeper reference.
- `.agentspack/docs/cloudflare-agent-stack/18_ANTI_PATTERNS.md` — common security-adjacent mistakes.
- `cloudflare-callable-methods` — per-method authz, input validation.
- `cloudflare-retries-with-idempotency` — idempotency pattern for writes.
- `cloudflare-mcp-tools` — tool allowlists and scoped tokens.
- `cloudflare-sandboxes` / `cloudflare-browser-run` / `cloudflare-durable-objects-facets-dynamic-workers` — untrusted-code boundaries.
