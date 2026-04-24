# Schedule Tasks

## What it is

The Agents SDK can persist callbacks that run later. Schedules survive agent restarts and are stored in SQLite. Under the hood, scheduled tasks use Durable Object alarms to wake the agent.

## Use schedules for

- reminders
- delayed follow-ups
- retries with explicit timing
- scheduled publishing
- polling loops with a safe sleep/wake pattern
- recurring cron jobs scoped to one agent
- self-destruct or cleanup callbacks

## Scheduling modes

```ts
// Delayed: run after N seconds
await this.schedule(60, "sendReminder", { message: "Check status" });

// Specific timestamp
await this.schedule(new Date("2026-05-01T09:00:00Z"), "sendReminder", {
  message: "Monthly report due",
});

// Cron: recurring schedule
await this.schedule("0 8 * * *", "dailyDigest", { userId });

// Interval
await this.scheduleEvery(30, "poll", { source: "updates" });
```

## Callback pattern

```ts
class ReminderAgent extends Agent {
  async sendReminder(payload: { message: string }) {
    await this.sendNotification(payload.message);
  }
}
```

## Dynamic recurring pattern

For dynamic recurring schedules, schedule the next run from inside the callback:

```ts
class PollingAgent extends Agent {
  async startPolling(intervalSeconds: number) {
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
    for (const schedule of schedules) {
      if (schedule.callback === "poll") await this.cancelSchedule(schedule.id);
    }
  }
}
```

## Schedules vs queues

Use schedules when time matters. Use queues when order/background processing matters. A schedule can enqueue work; a queue callback can create a schedule.

## Schedules vs Workflows

Use schedules for local future callbacks. Use Workflows for durable multi-step orchestration, external visibility, independent retries, and complex process state.

## Sources

- https://developers.cloudflare.com/agents/api-reference/schedule-tasks/
