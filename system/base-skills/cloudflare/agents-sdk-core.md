---
name: cloudflare-agents-sdk-core
description: Scaffold a new stateful agent on Cloudflare using the Agents SDK. Use when the user asks to build a Cloudflare agent, AI agent, chat agent, session-based agent, stateful agent, Durable Object-backed agent, or mentions Agents SDK, `Agent` class, `getAgentByName`, routing requests to an agent, choosing per-user vs per-thread vs per-task identity, or wiring the agent binding in `wrangler.jsonc`. Produces a minimal-but-correct `Agent<Env, State>` class, a Worker routing entry point, and the matching wrangler config with the SDK's conventions already applied.
---

# Cloudflare Agents SDK — Core Scaffold

Use this skill when starting a new Cloudflare-agent project, or when adding the first agent class to an existing Worker. It gets the skeleton right on the first try so you don't fight the SDK conventions later.

## When to use

- Creating a new `Agent` class (no existing class in the project).
- Adding a second agent type to an existing project that already has one.
- Wiring a Worker entry point that needs to route HTTP/WebSocket traffic to agent instances.
- Deciding agent identity (per user / workspace / task / thread / external resource).
- Configuring the Durable Object binding in `wrangler.jsonc` for an `Agent` subclass.

## When NOT to use (pick the right sibling skill instead)

- Designing `this.state` shape or deciding state-vs-SQL → `cloudflare-agent-state-and-sql`.
- Exposing methods to browser/mobile clients via `@callable` → `cloudflare-callable-methods`.
- Future/cron callbacks, polling loops → `cloudflare-scheduled-tasks`.
- Long-running agent-internal work that must survive eviction → `cloudflare-durable-execution-fibers`.
- Multi-step durable orchestration with external visibility → `cloudflare-workflows-with-agents`.

## Decision: how should the agent be identified?

Agent instances are named. Pick the name strategy deliberately — it determines contention, state size, scheduling scope, and routing complexity.

| Identity             | Use when                                                                               | Tradeoff                                            |
| -------------------- | -------------------------------------------------------------------------------------- | --------------------------------------------------- |
| per user             | Personal assistants, chat copilots, inboxes.                                           | State grows with user lifetime; single writer.      |
| per workspace / team | Multi-tenant tools where collaborators share state.                                    | More contention; need ACL in callable methods.     |
| per task / job       | One-shot background tasks, research runs, long-running transactions.                   | Short-lived; cheap; easy to reason about.          |
| per conversation     | Threaded chat; each thread is independent.                                             | Many instances; simple state.                       |
| per external resource | Document, room, project, calendar, issue — one agent mirrors the external entity.     | Natural webhook target; lifecycle tied to resource. |

Rule of thumb: start narrower than you think. It's easier to promote to a broader identity later than to split one overloaded agent.

## Step-by-step

1. Install the SDK:
   ```bash
   npm install agents
   ```
2. Add the Durable Object binding and migration to `wrangler.jsonc` (template below).
3. Define an `Env` interface that includes the binding.
4. Create the `Agent<Env, State>` subclass with an `initialState` (or `undefined` state if stateless).
5. Create a Worker entry point that uses `getAgentByName` for RPC or `routeAgentRequest` for HTTP.
6. Regenerate bindings: `wrangler types`.
7. Run locally: `wrangler dev`. Deploy: `wrangler deploy`.

## Template — `wrangler.jsonc`

```jsonc
{
  "name": "my-agent-worker",
  "main": "src/worker.ts",
  "compatibility_date": "2026-04-01",
  "compatibility_flags": ["nodejs_compat"],

  "durable_objects": {
    "bindings": [
      {
        "name": "MY_AGENT",
        "class_name": "MyAgent"
      }
    ]
  },

  "migrations": [
    {
      "tag": "v1",
      "new_sqlite_classes": ["MyAgent"]
    }
  ]
}
```

Notes:
- `new_sqlite_classes` (not `new_classes`) — agents use the SQLite-backed storage needed by `this.sql` and state persistence.
- Keep the binding name uppercase-snake-case; it becomes the env key.

## Template — minimal agent + worker

```ts
// src/agent.ts
import { Agent } from "agents";

export interface Env {
  MY_AGENT: DurableObjectNamespace<MyAgent>;
}

type State = {
  status: "idle" | "working" | "done";
  createdAt: number;
};

export class MyAgent extends Agent<Env, State> {
  initialState: State = {
    status: "idle",
    createdAt: Date.now(),
  };

  async ping(): Promise<string> {
    return `agent ${this.name} is ${this.state.status}`;
  }
}
```

```ts
// src/worker.ts
import { getAgentByName, routeAgentRequest } from "agents";
import { MyAgent } from "./agent";

export { MyAgent };

export default {
  async fetch(request: Request, env: Env, ctx: ExecutionContext) {
    const url = new URL(request.url);

    if (url.pathname.startsWith("/agent/")) {
      const response = await routeAgentRequest(request, env);
      if (response) return response;
    }

    if (url.pathname === "/ping") {
      const userId = url.searchParams.get("user") ?? "anonymous";
      const agent = await getAgentByName<Env, MyAgent>(env.MY_AGENT, userId);
      const message = await agent.ping();
      return new Response(message);
    }

    return new Response("not found", { status: 404 });
  },
} satisfies ExportedHandler<Env>;
```

## Why this shape

- **`getAgentByName`** is the canonical way for server-side code (Workers, other agents) to talk to an agent via Durable Object RPC. Use this for Worker → agent calls.
- **`routeAgentRequest`** is the canonical way to forward HTTP/WebSocket traffic from a public entry point to the right named agent (path-based routing like `/agent/:class/:name`). Use this for client-facing transport.
- **`export { MyAgent }`** is required so Cloudflare can resolve the Durable Object class at deploy time.
- **`initialState`** must exist if any method reads `this.state` before the first `setState()` — otherwise you'll get `undefined` on a fresh instance.

## Common mistakes this skill prevents

- Forgetting `new_sqlite_classes` in the migration — storage won't be SQLite-backed and `this.sql` will fail.
- Using `fetch(env.MY_AGENT.idFromName(...))` directly — works, but bypasses Agents SDK helpers; prefer `getAgentByName` so you get typed RPC.
- Threading `env` through every function — import from `cloudflare:workers` inside helpers instead.
- Storing large blobs in `this.state` (it syncs to every connected client) — see `cloudflare-agent-state-and-sql`.
- Picking "per user" identity by default when the real unit of work is a task — see the identity table above.

## See also

- `.agentspack/docs/cloudflare-agent-stack/02_AGENTS_SDK_CORE.md` — deeper reference.
- `.agentspack/docs/cloudflare-agent-stack/00_START_HERE.md` — mental model.
- `.agentspack/docs/cloudflare-agent-stack/01_DECISION_GUIDE.md` — full primitive decision matrix.
- Official: https://developers.cloudflare.com/agents/
