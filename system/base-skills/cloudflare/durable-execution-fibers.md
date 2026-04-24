---
name: cloudflare-durable-execution-fibers
description: Run long-running agent-internal work that survives Durable Object eviction using Cloudflare Agents fibers (`this.runFiber`, `ctx.stash`, `onFiberRecovered`, `keepAlive` / `keepAliveWhile`). Use when the user asks about surviving agent eviction, checkpointing long tasks, resuming work after a restart, fire-and-forget background jobs, or mentions `runFiber`, `stash`, `keepAlive`. Produces a discriminated-union snapshot pattern, an `onFiberRecovered` switch that resumes at the right step, and guidance on fiber vs keepAlive vs Workflow.
---

# Cloudflare Agents — Durable Execution (Fibers)

Use this skill when an agent needs to run multi-step internal work that must survive Durable Object eviction. Fibers checkpoint to SQLite, keep the agent alive during execution, and call `onFiberRecovered` on the next activation if the agent was evicted mid-task.

## When to use

- Work has multiple logical steps with meaningful checkpoints.
- Progress is expensive to re-do if lost (API costs, compute time, human time).
- Task may run for 10+ minutes.
- Fire-and-forget recovery is acceptable — the original caller may be gone by the time recovery runs.

## When NOT to use (pick the right sibling skill instead)

| Situation                                                             | Use instead                           |
| --------------------------------------------------------------------- | ------------------------------------- |
| Short, cheap-to-restart work (a single slow API call)                 | `this.keepAlive()` / `this.keepAliveWhile()` |
| Independent process with external visibility, per-step retries, days/weeks lifetime, multi-service coordination | Workflow — see `cloudflare-workflows-with-agents` |
| Just retrying a flaky external call                                   | `this.retry()` — see retries skill    |
| Ordered background work within one agent                              | `this.queue()` — see queued-tasks skill |

Rule: fibers are for **agent-internal** durability. Workflows are for **externally visible** orchestration.

## `keepAlive` vs `runFiber`

```ts
// keepAlive: short, restartable work — keeps the DO awake for one activation.
await this.keepAliveWhile(async () => {
  return this.slowApiCall();
});

// runFiber: multi-step, checkpointed, recoverable after eviction.
void this.runFiber("research", async (ctx) => {
  /* steps with ctx.stash(...) between them */
});
```

Use `keepAliveWhile` for one short async operation you don't want interrupted. Use `runFiber` when you want **recovery** — the agent can be evicted and pick up where it left off.

## Full pattern — discriminated-union snapshots + recovery switch

```ts
import { Agent } from "agents";
import type { FiberRecoveryContext } from "agents";

type Snapshot =
  | { stage: "searched"; topic: string; searchResults: unknown }
  | { stage: "analyzed"; topic: string; searchResults: unknown; analysis: unknown };

export class ResearchAgent extends Agent<Env, { status: string }> {
  initialState = { status: "idle" };

  async startResearch(topic: string) {
    this.setState({ ...this.state, status: "running" });

    void this.runFiber("research", async (ctx) => {
      const searchResults = await this.search(topic);
      ctx.stash<Snapshot>({ stage: "searched", topic, searchResults });

      const analysis = await this.analyze(searchResults);
      ctx.stash<Snapshot>({ stage: "analyzed", topic, searchResults, analysis });

      const report = await this.writeReport(analysis);
      this.setState({ ...this.state, status: "done" });
      await this.storeReport(report);
    });
  }

  async onFiberRecovered(ctx: FiberRecoveryContext) {
    if (ctx.name !== "research") return;
    const snap = ctx.snapshot as Snapshot | null;
    if (!snap) return;

    if (snap.stage === "searched") {
      void this.runFiber("research", async (next) => {
        const analysis = await this.analyze(snap.searchResults);
        next.stash<Snapshot>({ ...snap, stage: "analyzed", analysis });
        const report = await this.writeReport(analysis);
        this.setState({ ...this.state, status: "done" });
        await this.storeReport(report);
      });
      return;
    }

    if (snap.stage === "analyzed") {
      void this.runFiber("research", async () => {
        const report = await this.writeReport(snap.analysis);
        this.setState({ ...this.state, status: "done" });
        await this.storeReport(report);
      });
    }
  }

  private async search(_topic: string) {
    return {};
  }
  private async analyze(_r: unknown) {
    return {};
  }
  private async writeReport(_a: unknown) {
    return "";
  }
  private async storeReport(_r: string) {}
}
```

Why this shape:
- **Fire-and-forget** (`void this.runFiber(...)`) because the original HTTP/WebSocket caller may already be disconnected when the fiber finishes.
- **Discriminated union snapshots** (`stage: "searched" | "analyzed"`) make recovery a compile-time-exhaustive switch.
- **Each `stash` replaces** — it's not a merge. Always stash the full state needed to resume.
- **`onFiberRecovered` re-launches a fiber for the remaining work** — that keeps every subsequent step also recoverable.

## Inline vs fire-and-forget

```ts
// Inline: caller awaits. If eviction happens mid-run, the caller is gone.
const result = await this.runFiber("quickWork", async (ctx) => {
  return expensiveCompute();
});

// Fire-and-forget: safer for long work; progress lives in state + snapshots.
void this.runFiber("longWork", async (ctx) => {
  await longRunningProcess();
});
```

For anything over ~30 seconds, prefer fire-and-forget.

## Checkpoint discipline

- Stash **after** each expensive step, not before.
- Don't stash giant documents — store them in R2 or `this.sql`, stash a reference id.
- Never stash secrets — snapshots persist in SQLite.
- Stash a version/schema field so a future code change can still recognize and resume old snapshots.

## Errors and retries inside fibers

`runFiber()` does not automatically retry thrown errors. If the callback throws:
- The fiber row is deleted.
- The error propagates (inline) or is logged (fire-and-forget).
- `onFiberRecovered` is **not** called for errors; it's called for eviction/restart.

Pair with `this.retry()` around fragile sub-steps so transient blips don't torch the fiber:

```ts
void this.runFiber("publish", async (ctx) => {
  const result = await this.retry(() => callExternal(), { maxAttempts: 5 });
  ctx.stash({ stage: "published", result });
});
```

## Common mistakes this skill prevents

- Awaiting a long `runFiber` inline — eviction loses the return value; caller is confused.
- Merging new data into previous stash in your head — `stash` always replaces; always include full state.
- Forgetting `onFiberRecovered` — eviction leaves half-done work forever.
- Non-discriminated snapshots (`{ ... }` without a `stage` tag) — recovery becomes guesswork.
- Using fibers for cross-service orchestration — use Workflows for visibility and per-step retries.
- Swallowing errors silently in fire-and-forget fibers — log to `this.sql` or update `this.state.status` to `"error"` so the UI knows.

## See also

- `.agentspack/docs/cloudflare-agent-stack/08_DURABLE_EXECUTION_FIBERS.md` — deeper reference.
- `cloudflare-workflows-with-agents` — when fibers aren't enough.
- `cloudflare-retries-with-idempotency` — pair with fibers for fragile sub-steps.
- `cloudflare-agent-state-and-sql` — update state from inside fibers.
- Official: https://developers.cloudflare.com/agents/api-reference/durable-execution/
