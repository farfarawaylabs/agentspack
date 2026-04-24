---
name: cloudflare-queued-tasks
description: Use per-agent FIFO background processing on Cloudflare Agents with `this.queue()`. Use when the user asks to queue background work inside an agent, defer expensive work after a user message, batch/serialize operations, process tasks in order, add retries to background callbacks, or wonders whether to use `this.queue()` vs Cloudflare Queues vs `this.schedule()`. Produces a correct `queue()` call with a matching callback signature, retry options, and clear guidance on when to reach for global Cloudflare Queues instead.
---

# Cloudflare Agents — Queued Tasks

Use this skill whenever an agent needs FIFO background processing inside **one** agent instance. `this.queue()` persists tasks in SQLite and processes them in order, surviving restarts.

## When to use

- Deferring expensive work (e.g. analysis, summarization) after handling a user message.
- Serializing work inside a single agent so operations don't race.
- Batch operations that must happen in a deterministic order.
- Background tasks with retry-on-failure semantics scoped to one agent.

## When NOT to use

| Situation                                                   | Use instead                            |
| ----------------------------------------------------------- | -------------------------------------- |
| Cross-agent fanout / multi-tenant queue / global throughput | Cloudflare Queues (platform primitive) |
| Time-triggered callbacks (cron, delay, recurring)           | `cloudflare-scheduled-tasks`           |
| Long multi-step durable process with external visibility    | `cloudflare-workflows-with-agents`     |
| Transient API call that just needs backoff                  | `cloudflare-retries-with-idempotency`  |

Rule: if the work is **one agent's FIFO list**, use `this.queue()`. If it's a pipe between services, use Cloudflare Queues.

## Basic pattern

```ts
import { Agent, type QueueItem } from "agents";

type ProcessPayload = {
  message: string;
  receivedAt: number;
};

export class ChatAgent extends Agent<Env> {
  async onMessage(message: string) {
    const taskId = await this.queue<ProcessPayload>("processMessage", {
      message,
      receivedAt: Date.now(),
    });
    this.setState({ ...this.state, lastQueuedTaskId: taskId });
  }

  async processMessage(
    payload: ProcessPayload,
    _item: QueueItem<ProcessPayload>,
  ) {
    const analysis = await this.analyze(payload.message);
    await this.sql`
      INSERT INTO analyses (message, result, created_at)
      VALUES (${payload.message}, ${JSON.stringify(analysis)}, ${payload.receivedAt})
    `;
  }

  private async analyze(_: string) {
    return { summary: "…" };
  }
}
```

Signature rules:
- Callback must be a method on `this` — `queue()` validates it exists at enqueue time.
- First parameter is the payload; optional second parameter is the `QueueItem<T>` wrapper (attempt count, id, timestamps).
- Callback must be JSON-serializable in and out — same rules as callable methods.

## Retry options

```ts
await this.queue(
  "sendEmail",
  { to: "user@example.com", subject: "Welcome" },
  {
    retry: {
      maxAttempts: 5,
      baseDelayMs: 500,
      maxDelayMs: 5_000,
    },
  },
);
```

Without retry options, failures are raised once and the task is dropped. With them, the callback retries with backoff until exhausted, then logs and dequeues. For idempotency (emails, payments, writes) see `cloudflare-retries-with-idempotency`.

## Processing behavior (worth remembering)

- FIFO by creation time. No priority queue; if you need priority, split into separate callback names and route work in.
- Successful runs dequeue automatically.
- Failed runs with no retry options are logged and dequeued.
- If the callback method was renamed/removed, the task is skipped/logged — refactors must sweep queued method names.
- Tasks persist across DO eviction and code redeploys.

## Compose with other primitives

- Schedule → queue: `schedule()` callback enqueues expensive follow-up work so it respects FIFO ordering.
- Queue → schedule: a queue callback can `schedule()` a future retry window for a permanent-error-after-cooldown case.
- Queue → fiber: for a single queue item that must survive eviction with checkpoints, run the work inside `runFiber` — see `cloudflare-durable-execution-fibers`.

## Common mistakes this skill prevents

- Using `this.queue()` for cross-agent broadcasts — that belongs in Cloudflare Queues.
- Passing large payloads — they sit in SQLite per row. Store blobs in `this.sql` and enqueue the id.
- Forgetting method renames — the task silently skips when the callback no longer exists.
- Mixing order-critical + order-insensitive work on one queue — split into two callback names.
- Setting `maxAttempts: 100` on non-idempotent ops — multiply-sent emails will ship. Pair retries with idempotency keys.

## See also

- `.agentspack/docs/cloudflare-agent-stack/06_QUEUE_TASKS.md` — deeper reference.
- `cloudflare-retries-with-idempotency` — safe retry design for side-effectful work.
- `cloudflare-scheduled-tasks` — time-based alternative.
- Official: https://developers.cloudflare.com/agents/api-reference/queue-tasks/
