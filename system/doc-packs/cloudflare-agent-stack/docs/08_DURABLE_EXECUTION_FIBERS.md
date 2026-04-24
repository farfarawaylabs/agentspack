# Durable Execution and Fibers

## What it is

`runFiber()` lets an Agent run work that survives Durable Object eviction. It registers the work in SQLite, keeps the agent alive during execution, supports checkpointing through `stash()`, and calls `onFiberRecovered()` on the next activation if the Agent was evicted mid-task.

## Why it matters

Durable Objects can be evicted during inactivity, code updates, runtime restarts, resource limits, or alarm timeouts. When that happens, in-memory state and upstream connections are lost. `keepAlive()` reduces eviction risk; `runFiber()` makes some work recoverable.

## keepAlive vs runFiber

Use `keepAlive()` / `keepAliveWhile()` when the work is short and cheap to restart, such as one slow API call.

Use `runFiber()` when:

- work has multiple steps
- progress is expensive to lose
- the task may run for 10+ minutes
- you can define meaningful checkpoints
- fire-and-forget recovery is acceptable

Use Workflows when:

- the process is independent from current agent activation
- you need per-step retries
- you need visibility/auditability
- the process may last days/weeks
- the process coordinates many services or agents

## API shape

```ts
class Agent {
  runFiber<T>(name: string, fn: (ctx: FiberContext) => Promise<T>): Promise<T>;
  stash(data: unknown): void;
  onFiberRecovered(ctx: FiberRecoveryContext): Promise<void>;
}

type FiberContext = {
  id: string;
  stash(data: unknown): void;
  snapshot: unknown | null;
};

type FiberRecoveryContext = {
  id: string;
  name: string;
  snapshot: unknown | null;
};
```

## Basic pattern

```ts
import { Agent } from "agents";
import type { FiberRecoveryContext } from "agents";

class ResearchAgent extends Agent {
  async startResearch(topic: string) {
    void this.runFiber("research", async (ctx) => {
      const searchResults = await this.search(topic);
      ctx.stash({ topic, stage: "searched", searchResults });

      const analysis = await this.analyze(searchResults);
      ctx.stash({ topic, stage: "analyzed", searchResults, analysis });

      const report = await this.writeReport(analysis);
      this.setState({ ...this.state, status: "done", report });
    });
  }

  async onFiberRecovered(ctx: FiberRecoveryContext) {
    if (ctx.name !== "research") return;

    const snapshot = ctx.snapshot as
      | { topic: string; stage: "searched"; searchResults: unknown }
      | { topic: string; stage: "analyzed"; searchResults: unknown; analysis: unknown }
      | null;

    if (!snapshot) return;

    if (snapshot.stage === "searched") {
      void this.runFiber("research", async (nextCtx) => {
        const analysis = await this.analyze(snapshot.searchResults);
        nextCtx.stash({ ...snapshot, stage: "analyzed", analysis });
        const report = await this.writeReport(analysis);
        this.setState({ ...this.state, status: "done", report });
      });
    }

    if (snapshot.stage === "analyzed") {
      void this.runFiber("research", async () => {
        const report = await this.writeReport(snapshot.analysis);
        this.setState({ ...this.state, status: "done", report });
      });
    }
  }
}
```

## Inline vs fire-and-forget

```ts
// Inline: caller waits.
const result = await this.runFiber("work", async (ctx) => {
  return computeExpensiveThing();
});

// Fire-and-forget: safer for long work.
void this.runFiber("background", async (ctx) => {
  await longRunningProcess();
});
```

If eviction happens during an inline `await`, the original caller is gone. Recovery cannot return a result to that caller. For long-running work, prefer fire-and-forget with state updates and recovery logic.

## Checkpoint rules

- `ctx.stash(data)` writes the snapshot to SQLite.
- Each stash replaces the previous snapshot; it is not a merge.
- Stash the full recovery state needed to resume.
- Use discriminated union snapshots: `{ stage: "searched", ... }`.
- Do not stash giant raw documents; store those in R2/SQL and stash references.
- `this.stash()` works inside a fiber; it throws outside a `runFiber` callback.

## Important limitation

`runFiber()` does not automatically retry thrown errors. If the fiber callback throws, the row is deleted and the error propagates/logs. Use `this.retry()` inside the fiber, or encode recovery in `onFiberRecovered()`.

## Sources

- https://developers.cloudflare.com/agents/api-reference/durable-execution/
