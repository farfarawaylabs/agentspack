---
name: cloudflare-agent-generated-miniapp-pattern
description: Blueprint for a supervisor agent that generates a small, isolated, stateful mini-app (UI + behavior) on demand, sandboxes the generated code, stores versioned artifacts, and exposes the app via a Dynamic Worker / Durable Object Facet. Use when the user asks for "AI that builds small apps", "an agent that generates tools", "per-user custom mini-apps", "agent-authored UI components", "on-demand micro-services", "facet-based mini-apps", or any recipe that combines code generation + sandboxed execution + per-resource state + dynamic routing. Produces a SupervisorAgent that generates code in a Sandbox, persists versioned artifacts, enforces allowed bindings, and spins up a Dynamic Worker/Facet to serve the generated mini-app per user/project.
---

# Recipe: Agent-generated mini-app

Use this recipe when an agent must **produce a new, small, runnable app** (UI + small backend + tiny state) on behalf of a user, then **host that app** so the user can interact with it — and do so safely, per-user, with versioned code and isolated state.

## When to use

- User says "build me a little tool that…" and you want to serve them a real thing, not just text.
- Each generated mini-app has its own state/data and should not share memory with other users' apps.
- Generated code needs to run (tests, compile, limited I/O) — not just be pasted back as a string.
- You want policy control: allowed bindings, allowed fetch hosts, CPU caps, etc.

## When NOT to use

- "Generate a static file" (HTML/PDF) → just generate and return; no runtime needed.
- "Run this script once, show the output" → `cloudflare-sandboxes` directly, no miniapp hosting.
- Multi-tenant SaaS where all users share the same app → normal Worker with a user-scoped agent, not dynamic per-app hosting.

## Primitives used

| Primitive                                                | Role                                                                |
| -------------------------------------------------------- | ------------------------------------------------------------------- |
| `cloudflare-agents-sdk-core`                             | SupervisorAgent (per user or per session).                          |
| `cloudflare-sandboxes`                                   | Compile/test the generated code, run quick smoke checks.            |
| `cloudflare-durable-objects-facets-dynamic-workers`      | Host the generated mini-app per resource, with isolated state.      |
| `cloudflare-agent-state-and-sql`                         | Supervisor stores versioned code + metadata per mini-app.           |
| `cloudflare-agent-security`                              | Enforce allowed bindings, fetch egress policy, caps.                |
| `cloudflare-workflows-with-agents` (optional)             | Orchestrate the generate → test → deploy pipeline.                  |

## Conceptual layout

```
 ┌──────────────────┐
 │ SupervisorAgent   │  per user/session
 │  - state: catalog │
 │  - sql:  versions │
 │  - callables:     │
 │     generateApp() │
 │     listApps()    │
 │     destroyApp()  │
 └──────┬───────────┘
        │ generates + tests code
        ▼
 ┌──────────────────┐      ┌───────────────────────────┐
 │ Sandbox           │──▶  │ Dynamic Worker / Facet     │
 │ (compile + test)  │      │  one per generated app     │
 └──────────────────┘      │  isolated state + routing  │
                           └───────────────────────────┘
```

## Blueprint — supervisor

```ts
// src/supervisor-agent.ts
import { Agent, callable } from "agents";
import { z } from "zod";

export interface Env {
  SUPERVISOR_AGENT: DurableObjectNamespace<SupervisorAgent>;
  SANDBOX: { run(spec: { code: string; tests: string; allowedBindings: string[] }): Promise<SandboxResult> };
  MINIAPP_DEPLOYER: { deploy(spec: MiniAppSpec): Promise<{ url: string; appId: string }> };
  CODE_STORE: R2Bucket;
}

type SandboxResult = { ok: boolean; testsPassed: boolean; logs: string };

type MiniAppSpec = {
  appId: string;
  version: number;
  entrypointKey: string; // R2 key to the bundled code.
  allowedBindings: string[];
  cpuLimitMs: number;
};

type CatalogEntry = {
  appId: string;
  title: string;
  currentVersion: number;
  url: string;
  createdAt: number;
};

type State = {
  userId: string;
  catalog: CatalogEntry[]; // keep small; heavier history lives in SQL.
};

const GenerateInput = z.object({
  title: z.string().min(1).max(120),
  brief: z.string().min(1).max(4000),
});

const DestroyInput = z.object({ appId: z.string().min(1) });

export class SupervisorAgent extends Agent<Env, State> {
  initialState: State = { userId: "", catalog: [] };

  async onStart() {
    this.sql`
      CREATE TABLE IF NOT EXISTS app_versions (
        app_id TEXT NOT NULL,
        version INTEGER NOT NULL,
        entrypoint_key TEXT NOT NULL,
        created_at INTEGER NOT NULL,
        logs TEXT,
        PRIMARY KEY (app_id, version)
      );
    `;
  }

  @callable()
  async generateApp(input: unknown): Promise<CatalogEntry> {
    const { title, brief } = GenerateInput.parse(input);

    const appId = this.findAppIdByTitle(title) ?? crypto.randomUUID();
    const nextVersion = (this.currentVersion(appId) ?? 0) + 1;

    const { code, tests } = await draftCodeAndTests(brief);

    const allowedBindings = policyForBrief(brief); // e.g. ["KV_USER_NOTES"]
    const result = await this.env.SANDBOX.run({ code, tests, allowedBindings });
    if (!result.ok || !result.testsPassed) {
      throw new Error("generated app failed sandbox tests");
    }

    const entrypointKey = `${this.name}/${appId}/v${nextVersion}.js`;
    await this.env.CODE_STORE.put(entrypointKey, code);

    const { url } = await this.env.MINIAPP_DEPLOYER.deploy({
      appId,
      version: nextVersion,
      entrypointKey,
      allowedBindings,
      cpuLimitMs: 50,
    });

    this.sql`
      INSERT INTO app_versions (app_id, version, entrypoint_key, created_at, logs)
      VALUES (${appId}, ${nextVersion}, ${entrypointKey}, ${Date.now()}, ${result.logs})
    `;

    const entry: CatalogEntry = {
      appId,
      title,
      currentVersion: nextVersion,
      url,
      createdAt: Date.now(),
    };

    this.setState({
      ...this.state,
      catalog: upsertCatalog(this.state.catalog, entry),
    });

    return entry;
  }

  @callable()
  listApps(): CatalogEntry[] {
    return this.state.catalog;
  }

  @callable()
  async destroyApp(input: unknown): Promise<void> {
    const { appId } = DestroyInput.parse(input);
    this.setState({
      ...this.state,
      catalog: this.state.catalog.filter((e) => e.appId !== appId),
    });
    this.sql`DELETE FROM app_versions WHERE app_id = ${appId}`;
    // Deployer should also be told to tear the Facet/Dynamic Worker down.
  }

  private currentVersion(appId: string): number | undefined {
    const rows = this.sql`SELECT MAX(version) AS v FROM app_versions WHERE app_id = ${appId}`;
    return rows[0]?.v ?? undefined;
  }

  private findAppIdByTitle(title: string): string | undefined {
    return this.state.catalog.find((e) => e.title === title)?.appId;
  }
}

function upsertCatalog(list: CatalogEntry[], entry: CatalogEntry): CatalogEntry[] {
  const filtered = list.filter((e) => e.appId !== entry.appId);
  return [...filtered, entry];
}

async function draftCodeAndTests(_brief: string): Promise<{ code: string; tests: string }> {
  return { code: "", tests: "" }; // Replace with your code-gen model.
}

function policyForBrief(_brief: string): string[] {
  return []; // Map capabilities the brief implies to binding allowlist.
}
```

## Hosting layer

The mini-app itself is hosted as:

- A **Dynamic Worker** if each mini-app is a separate logical service.
- A **Durable Object Facet** if each mini-app is a stateful object whose state belongs to one resource (per document, per user, per project).

See `cloudflare-durable-objects-facets-dynamic-workers` for the exact wiring of `MINIAPP_DEPLOYER` and how to route `/app/:appId/*` to the generated code.

## Policy enforcement checklist

- Generated code only sees the bindings listed in `allowedBindings`.
- Fetch egress is restricted to an explicit allowlist (apply via Worker outbound bindings).
- CPU / memory / wall-clock limits per request.
- Every deploy is versioned in R2; rollbacks revert the Facet/Worker to a previous entrypointKey.
- Sandbox **must** be on the critical path — never deploy code that didn't pass the sandbox.

## Common mistakes this recipe prevents

- Giving the generated mini-app access to the supervisor's full `env` — policy escape.
- Storing generated source in `state` — blows up state sync and costs DO storage.
- Treating each new generation as a new appId — loses version history; can't rollback.
- Skipping the sandbox on "trivial" apps — all generations must pass.
- Sharing a single DO namespace across all users' mini-apps — cross-user state contamination. Keep per-resource Facet IDs.
- Hot-patching the deployed Worker from the supervisor instead of re-deploying a versioned entrypoint.

## See also

- `cloudflare-sandboxes` — compile + test isolation.
- `cloudflare-durable-objects-facets-dynamic-workers` — hosting model for the generated apps.
- `cloudflare-agent-security` — binding allowlist + egress policy.
- `cloudflare-workflows-with-agents` — orchestrate generate → test → deploy as a visible pipeline.
- `cloudflare-agent-state-and-sql` — why catalog lives in state and versions live in SQL.
- `.agentspack/docs/cloudflare-agent-stack/17_PATTERNS_AND_RECIPES.md` — prose version.
