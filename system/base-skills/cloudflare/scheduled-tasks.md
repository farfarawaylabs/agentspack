---
name: cloudflare-scheduled-tasks
description: Schedule future, cron, or interval callbacks from a Cloudflare Agent using `this.schedule`, `this.scheduleEvery`, `getSchedules`, and `cancelSchedule`. Use when the user asks for a delayed callback, a reminder, a cron job, a recurring poll/loop, "run in X minutes", "every day at 9am", dynamic polling intervals, or needs to cancel/list scheduled work. Avoids `setTimeout`-based polling that dies on eviction. Produces delay/timestamp/cron/interval templates, a self-rescheduling dynamic-polling pattern, and cancellation helpers.
---

# Cloudflare Agents — Scheduled Tasks

Use this skill whenever an agent needs to run a callback in the future (once, on a cron, or on an interval). Scheduled tasks are stored in SQLite and survive restarts — `setTimeout` does not.

## When to use

- Reminders, delayed follow-ups, scheduled publishing.
- Cron jobs scoped to a single agent (per user / per workspace).
- Polling loops where the agent should go to sleep between iterations.
- Cleanup / self-destruct callbacks.
- Retry with explicit timing (not exponential backoff — use `this.retry` for that).

## When NOT to use (pick the right sibling skill instead)

- FIFO background work where order matters more than timing → `cloudflare-queued-tasks`.
- Transient API failures with backoff → `cloudflare-retries-with-idempotency`.
- Multi-step durable orchestration that needs step-level retries and external visibility → `cloudflare-workflows-with-agents`.
- Long agent-internal loops that must survive eviction with checkpoints → `cloudflare-durable-execution-fibers`.

## Four scheduling modes

```ts
import { Agent } from "agents";

export class ReminderAgent extends Agent<Env> {
  async remindInSeconds(message: string) {
    return this.schedule(60, "sendReminder", { message });
  }

  async remindAt(iso: string, message: string) {
    return this.schedule(new Date(iso), "sendReminder", { message });
  }

  async dailyDigestFor(userId: string) {
    return this.schedule("0 8 * * *", "dailyDigest", { userId });
  }

  async startPollingUpdates() {
    return this.scheduleEvery(30, "poll", { source: "updates" });
  }

  async sendReminder(payload: { message: string }) {
    await this.sendNotification(payload.message);
  }

  async dailyDigest(payload: { userId: string }) {
    // ... assemble and send digest ...
  }

  async poll(payload: { source: string }) {
    // ... do work ...
  }

  private async sendNotification(_msg: string) {
    // push / email / state flag / etc.
  }
}
```

Notes:
- First arg shapes the timing: `number` = seconds from now, `Date` = absolute, `string` = cron expression.
- Second arg is the **method name on this agent** — it must exist, or the task is skipped when it fires.
- Third arg is the JSON-serializable payload passed into the callback.

## Dynamic polling (preferred over `setTimeout` loops)

When the interval itself is dynamic, schedule the next run from inside the callback:

```ts
export class PollingAgent extends Agent<Env, { pollEvery: number }> {
  initialState = { pollEvery: 60 };

  async startPolling(intervalSeconds: number) {
    this.setState({ ...this.state, pollEvery: intervalSeconds });
    await this.schedule(intervalSeconds, "poll", { intervalSeconds });
  }

  async poll(payload: { intervalSeconds: number }) {
    try {
      const res = await fetch("https://api.example.com/updates");
      await this.processUpdates(await res.json());
    } finally {
      await this.schedule(payload.intervalSeconds, "poll", payload);
    }
  }

  async stopPolling() {
    const schedules = this.getSchedules({ type: "delayed" });
    for (const s of schedules) {
      if (s.callback === "poll") await this.cancelSchedule(s.id);
    }
  }

  private async processUpdates(_data: unknown) {}
}
```

Why the `finally` matters: if the poll throws, the next one still gets scheduled so the loop doesn't die silently. Pair with `this.retry(...)` inside the poll body if you want the individual iteration to survive transient failures before giving up for this tick.

## Listing and cancelling schedules

```ts
const upcoming = this.getSchedules({ type: "delayed" });
for (const s of upcoming) {
  if (s.callback === "poll") await this.cancelSchedule(s.id);
}
```

You can filter by type (`delayed` / `cron` / `scheduled` etc.), callback name, or just iterate and match. Always cancel specific tasks; don't wipe the whole schedule table unless that's actually what you want.

## Schedules vs queues vs Workflows

| Need                                             | Use                                      |
| ------------------------------------------------ | ---------------------------------------- |
| Fire at a specific time / after a delay          | `this.schedule(delay \| date \| cron)`   |
| FIFO background work, order-sensitive            | `this.queue(...)` — see queued-tasks skill |
| Recurring long-running multi-step process        | Workflow — see workflows-with-agents skill |

A schedule callback can enqueue work; a queue callback can create a follow-up schedule. Compose them.

## Common mistakes this skill prevents

- Using `setTimeout`/`setInterval` for agent loops — dies on eviction, never recovers.
- Calling `this.schedule(..., "missingMethod", ...)` — the task silently skips when it fires. Keep method name in sync with refactors.
- Stashing large blobs in the payload — kept in SQLite per schedule row. Store blobs in `this.sql` and pass an id.
- Forgetting to `cancelSchedule` when the user stops polling — you keep polling forever.
- Using cron for "one-off at 9am tomorrow" — use a `Date`, not a cron.

## See also

- `.agentspack/docs/cloudflare-agent-stack/05_SCHEDULE_TASKS.md` — deeper reference.
- `cloudflare-queued-tasks` — ordered background work inside an agent.
- `cloudflare-retries-with-idempotency` — backoff for transient failures.
- Official: https://developers.cloudflare.com/agents/api-reference/schedule-tasks/
