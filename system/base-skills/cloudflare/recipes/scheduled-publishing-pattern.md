---
name: cloudflare-scheduled-publishing-pattern
description: Blueprint for a per-channel agent that publishes items on a schedule (cron or dynamic cadence), safely retries each publish, and auto-generates more content when the queue is empty. Use when the user asks for "a scheduler", "an agent that posts on a schedule", "drip publishing", "auto-post to social", "send a daily digest", "cron-driven content", or any recipe that combines per-channel state + a publishing queue + timed execution + retries + optional generation. Produces a ChannelAgent with a SQL-backed publishing queue, cron or dynamic rescheduling, idempotent outbound publish, and optional Workflow-triggered generation when the queue runs dry.
---

# Recipe: Scheduled publishing

Use this recipe when you want **one agent per publishable channel** (social account, newsletter, blog, queue, etc.) that drips items out on its own cadence, retries transient failures, and requests more content when the queue is empty.

## When to use

- "Publish channel X on cadence Y" is the model.
- Each channel is an independent writer — different cron, different backlog, different generation policy.
- Publishes are side-effecting and need idempotency (no double posts).
- The cadence can be fixed cron (e.g. daily 08:00 UTC) **or** dynamic (e.g. "next item in 37 minutes because engagement peaked").

## When NOT to use

- One-time "send this at 8pm" reminder → plain `this.schedule({ at })`, no queue/generation machinery.
- Massive fan-out to millions of channels → consider Cloudflare Queues + Workers, not one DO per channel.
- Generation is the hard part and publishing is trivial → build a generation-first Workflow and call the publish inline.

## Primitives used

| Primitive                               | Role                                                              |
| --------------------------------------- | ----------------------------------------------------------------- |
| `cloudflare-agents-sdk-core`             | Identity = channel ID.                                            |
| `cloudflare-agent-state-and-sql`         | `state` = cadence + last publish; `this.sql` = queued items.      |
| `cloudflare-scheduled-tasks`             | Cron callback (`"0 8 * * *"`) or dynamic `{ in }` reschedule.     |
| `cloudflare-retries-with-idempotency`    | Safe outbound publish.                                            |
| `cloudflare-workflows-with-agents`       | (Optional) Triggered when the queue runs dry, to generate N items. |

## State shape

```ts
type State = {
  channelId: string;
  cadence: { kind: "cron"; expr: string } | { kind: "dynamic" };
  lastPublishedAt?: number;
  lastPublishedItemId?: string;
  paused: boolean;
  generation: { kind: "idle" } | { kind: "running"; workflowId: string };
};
```

SQL tables:

```sql
queued_items (id PK, payload TEXT, position INTEGER)
published_log (id PK, item_id, published_at, external_id)
```

## Blueprint

```ts
// src/channel-agent.ts
import { Agent, callable } from "agents";
import { z } from "zod";

export interface Env {
  CHANNEL_AGENT: DurableObjectNamespace<ChannelAgent>;
  PUBLISHER: { publish(item: QueuedItem): Promise<{ externalId: string }> };
  GEN_WORKFLOW: Workflow<{ channelId: string; count: number }>;
}

type QueuedItem = { id: string; payload: string };

type Cadence = { kind: "cron"; expr: string } | { kind: "dynamic" };
type State = {
  channelId: string;
  cadence: Cadence;
  lastPublishedAt?: number;
  paused: boolean;
  generation: { kind: "idle" } | { kind: "running"; workflowId: string };
};

const ConfigureInput = z.object({
  cadence: z.discriminatedUnion("kind", [
    z.object({ kind: z.literal("cron"), expr: z.string().min(5) }),
    z.object({ kind: z.literal("dynamic") }),
  ]),
});

const EnqueueInput = z.object({ payload: z.string().min(1).max(5000) });

export class ChannelAgent extends Agent<Env, State> {
  initialState: State = {
    channelId: "",
    cadence: { kind: "cron", expr: "0 8 * * *" },
    paused: false,
    generation: { kind: "idle" },
  };

  async onStart() {
    this.sql`
      CREATE TABLE IF NOT EXISTS queued_items (
        id TEXT PRIMARY KEY,
        payload TEXT NOT NULL,
        position INTEGER NOT NULL
      );
      CREATE TABLE IF NOT EXISTS published_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        item_id TEXT NOT NULL,
        external_id TEXT,
        published_at INTEGER NOT NULL
      );
    `;

    if (this.state.cadence.kind === "cron") {
      await this.ensureCron(this.state.cadence.expr);
    }
  }

  @callable()
  async configure(input: unknown): Promise<void> {
    const { cadence } = ConfigureInput.parse(input);
    this.setState({ ...this.state, cadence });
    if (cadence.kind === "cron") {
      await this.ensureCron(cadence.expr);
    }
  }

  @callable()
  async enqueue(input: unknown): Promise<{ id: string }> {
    const { payload } = EnqueueInput.parse(input);
    const id = crypto.randomUUID();
    const [{ max }] = this.sql`SELECT COALESCE(MAX(position), 0) + 1 AS max FROM queued_items`;
    this.sql`INSERT INTO queued_items (id, payload, position) VALUES (${id}, ${payload}, ${max})`;

    if (this.state.cadence.kind === "dynamic") {
      await this.schedule({ in: pickNextDelayMs(this.state) }, "publishNext");
    }
    return { id };
  }

  @callable()
  async pause(): Promise<void> {
    this.setState({ ...this.state, paused: true });
  }

  @callable()
  async resume(): Promise<void> {
    this.setState({ ...this.state, paused: false });
    if (this.state.cadence.kind === "cron") {
      await this.ensureCron(this.state.cadence.expr);
    } else {
      await this.schedule({ in: 60_000 }, "publishNext");
    }
  }

  async publishNext(): Promise<void> {
    if (this.state.paused) return;

    const rows = this.sql`SELECT id, payload FROM queued_items ORDER BY position ASC LIMIT 1`;
    if (rows.length === 0) {
      await this.requestMoreContent();
      return;
    }

    const item: QueuedItem = { id: rows[0].id, payload: rows[0].payload };

    const { externalId } = await this.retry(
      () => this.env.PUBLISHER.publish(item),
      { key: `publish-${this.name}-${item.id}` },
    );

    this.sql`DELETE FROM queued_items WHERE id = ${item.id}`;
    this.sql`INSERT INTO published_log (item_id, external_id, published_at) VALUES (${item.id}, ${externalId}, ${Date.now()})`;
    this.setState({ ...this.state, lastPublishedAt: Date.now() });

    if (this.state.cadence.kind === "dynamic") {
      await this.schedule({ in: pickNextDelayMs(this.state) }, "publishNext");
    }
  }

  private async ensureCron(expr: string): Promise<void> {
    // Cancel any previous schedules registered under this key, then reinstate.
    await this.cancelSchedules("publishNext");
    await this.schedule({ cron: expr }, "publishNext");
  }

  private async requestMoreContent(): Promise<void> {
    if (this.state.generation.kind === "running") return;

    const workflowId = `${this.name}-gen-${Date.now()}`;
    await this.env.GEN_WORKFLOW.create({
      id: workflowId,
      params: { channelId: this.name, count: 5 },
    });

    this.setState({ ...this.state, generation: { kind: "running", workflowId } });
  }

  async onGenerationComplete(items: Array<{ payload: string }>): Promise<void> {
    for (const { payload } of items) {
      await this.enqueue({ payload });
    }
    this.setState({ ...this.state, generation: { kind: "idle" } });
  }
}

function pickNextDelayMs(_state: State): number {
  return 30 * 60 * 1000; // Replace with engagement-aware logic.
}
```

## Common mistakes this recipe prevents

- Scheduling publishes **outside** the agent (e.g. a global cron trigger) — loses per-channel isolation and makes retries/backlogs ugly.
- Publishing without an idempotency key — at-least-once schedules cause duplicate posts on retry.
- Letting the publish handler do generation inline — one slow generation blocks the channel.
- Re-arming cron from every `publishNext` invocation — accumulates stacked schedules.
- Storing the backlog in `state` — UI-sync traffic balloons with every enqueue.
- Running a generation Workflow twice because the "queue empty" branch didn't check `state.generation`.

## See also

- `cloudflare-scheduled-tasks` — cron vs one-shot vs dynamic semantics.
- `cloudflare-retries-with-idempotency` — patterns for safe publish.
- `cloudflare-workflows-with-agents` — the generation Workflow you call when empty.
- `cloudflare-agent-state-and-sql` — state vs SQL split.
- `.agentspack/docs/cloudflare-agent-stack/17_PATTERNS_AND_RECIPES.md` — prose version.
