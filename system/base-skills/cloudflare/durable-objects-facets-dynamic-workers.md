---
name: cloudflare-durable-objects-facets-dynamic-workers
description: Build generated stateful mini-apps on Cloudflare using Durable Object Facets and Dynamic Workers under an agent supervisor. Use when the user asks to let an agent generate a custom per-customer app, spin up a stateful mini-app, dynamically load behavior into a Durable Object, deploy generated code as a Worker, or mentions Durable Object Facets, Dynamic Workers, supervisor-managed policy, generated-code guardrails. Produces a supervisor agent + Facet attach flow, a dynamic-worker deploy wrapper, and the guardrail checklist for untrusted generated code.
---

# Cloudflare Agents — Durable Object Facets and Dynamic Workers

Use this skill for the advanced pattern where an agent **generates a stateful surface** — a mini-app, a per-customer UI, a tool-with-its-own-UI — rather than handling everything inside one pre-deployed agent. Durable Object Facets attach dynamic behavior to a DO while preserving stateful coordination; Dynamic Workers run generated code inside Worker-like isolates.

## When to use

- Agent generates a per-customer mini-app that needs its own state.
- Agent creates a tool that comes with its own UI surface.
- Supervisor object needs to load/unload dynamic behavior without redeploying.
- Generated app must persist data independently (not just in the parent agent).

## When NOT to use

| Situation                                                   | Use instead                     |
| ----------------------------------------------------------- | ------------------------------- |
| Run generated code once, inspect the output                 | `cloudflare-sandboxes`          |
| Stateless generated HTTP handler, no storage needed         | A regular Worker route          |
| Per-user stateful agent with fixed behavior                 | `cloudflare-agents-sdk-core`    |
| One-shot code execution in an isolated environment          | Sandbox `exec`                  |

Rule: reach for Facets + Dynamic Workers when **the generated thing needs to keep state and serve traffic**. Otherwise Sandboxes or a normal agent are cheaper.

## Architecture

```
┌─────────────────────────┐
│ Supervisor Agent        │   stable, owns lifecycle + policy
│  - generates code       │
│  - deploys DynamicWorker│
│  - attaches Facet       │
│  - owns credentials     │
└───────────┬─────────────┘
            │  supervised by
            ▼
┌─────────────────────────┐
│ Durable Object + Facet  │   stateful coordination + dynamic behavior
│  - SQLite storage       │
│  - loaded Facet module  │
└───────────┬─────────────┘
            │  serves
            ▼
┌─────────────────────────┐
│ Dynamic Worker          │   generated HTTP surface for the mini-app
└─────────────────────────┘
```

Rules:
- Supervisor is **stable, deployed, trusted**. It stays the source of policy.
- Facets / Dynamic Workers contain **untrusted generated code**. They get minimum bindings.
- State boundaries are enforced by the supervisor — the generated code never sees cross-tenant data.

## Supervisor agent template

```ts
import { Agent, callable } from "agents";
import { z } from "zod";

const GenerateAppInput = z.object({
  workspaceId: z.string().uuid(),
  spec: z.string().min(1).max(10_000),
});

type State = {
  apps: Record<string, { version: string; deployedAt: number }>;
};

export class SupervisorAgent extends Agent<Env, State> {
  initialState: State = { apps: {} };

  @callable()
  async generateAndDeploy(raw: unknown) {
    const { workspaceId, spec } = GenerateAppInput.parse(raw);
    await this.assertOwnership(workspaceId);

    const generated = await this.generateCode(spec);
    await this.validateGenerated(generated);

    const version = await this.hash(generated.code);
    const appId = `app-${workspaceId}`;

    const dynamicWorker = await this.env.DYNAMIC_WORKERS.deploy({
      name: `${appId}@${version}`,
      code: generated.code,
      bindings: this.minimalBindings(workspaceId),
      network: { allowlist: ["api.example.com"] },
    });

    await this.env.FACETS.attach({
      durableObjectName: appId,
      moduleUrl: dynamicWorker.moduleUrl,
      version,
    });

    this.setState({
      ...this.state,
      apps: {
        ...this.state.apps,
        [appId]: { version, deployedAt: Date.now() },
      },
    });

    return { appId, previewUrl: dynamicWorker.previewUrl };
  }

  private async assertOwnership(_workspaceId: string) {}
  private async generateCode(_spec: string) { return { code: "" }; }
  private async validateGenerated(_g: { code: string }) {}
  private async hash(code: string) {
    const buf = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(code));
    return [...new Uint8Array(buf)].map((b) => b.toString(16).padStart(2, "0")).join("").slice(0, 16);
  }
  private minimalBindings(_workspaceId: string) {
    return {};
  }
}
```

Notes:
- **Version every deploy** — name includes the code hash, state records the version, rollback = redeploy the previous hash.
- **Minimum bindings** (`minimalBindings`) — only what the spec requires. No org-wide tokens.
- **Network allowlist** — explicit; generated code doesn't get open internet access.
- **Validation pass** before deploy — at minimum, reject banned imports, known-dangerous APIs, or references to secrets that shouldn't be there.

## Facet-loaded Durable Object

```ts
// The stable DO class. Behavior comes from the loaded Facet module.
import { DurableObject } from "cloudflare:workers";

export class MiniAppDO extends DurableObject<Env> {
  async fetch(request: Request) {
    const facet = await this.env.FACETS.load(this.ctx.id);
    if (!facet) return new Response("no facet", { status: 404 });
    return facet.fetch(request, this.ctx);
  }
}
```

The DO keeps the SQLite-backed storage and coordination; the Facet module provides the request handler. The supervisor can swap the module (new version) without touching state.

## Guardrails for generated code

- **No platform secrets** in generated bindings. No `CF_API_TOKEN`, no org-scoped keys.
- **No broad `fetch`** — network allowlist only.
- **No access to other tenants' storage.** The DO id is tenant-scoped; bindings are tenant-scoped.
- **Quotas per mini-app** — cap CPU, egress, and storage growth; revoke on breach.
- **Audit every deploy** — who generated what spec, which version got deployed, which user can access it.
- **Kill switch** — the supervisor can unload a Facet without redeploying the DO.

## Common mistakes this skill prevents

- Giving the generated worker the parent agent's secrets — prompt injection turns into data theft.
- Letting generated code reach arbitrary URLs — SSRF / exfiltration.
- Skipping versioning — you can't roll back a bad generation.
- Putting supervisor trust inside the generated code itself — policy must live in the stable supervisor.
- Using a Dynamic Worker for one-shot code execution — use a Sandbox (cheaper, disposable).
- Sharing a DO id across tenants — state leakage waiting to happen.

## See also

- `.agentspack/docs/cloudflare-agent-stack/10_DURABLE_OBJECTS_FACETS_DYNAMIC_WORKERS.md` — deeper reference.
- `cloudflare-agent-security` — generated-code boundary checklist.
- `cloudflare-sandboxes` — the simpler primitive when you just need isolated execution.
- `cloudflare-agents-sdk-core` — the stable supervisor agent pattern.
- Official: https://blog.cloudflare.com/durable-object-facets-dynamic-workers/
