# Queue Tasks

## What it is

The Agents SDK provides a per-agent built-in FIFO queue. Tasks are persisted in SQLite and processed automatically. This is useful for background processing inside an agent instance.

## Use queues for

- background tasks that do not need immediate response
- ordered processing
- batch operations
- deferring expensive work after a user message
- serializing work inside one agent
- tasks that should survive restarts

## Basic pattern

```ts
class MyAgent extends Agent {
  async onMessage(message: string) {
    const taskId = await this.queue("processMessage", {
      message,
      receivedAt: Date.now(),
    });
    this.setState({ ...this.state, lastQueuedTaskId: taskId });
  }

  async processMessage(
    payload: { message: string; receivedAt: number },
    queueItem: QueueItem<typeof payload>,
  ) {
    await this.analyzeMessage(payload.message);
  }
}
```

## Retry options

```ts
await this.queue(
  "sendEmail",
  { to: "user@example.com", subject: "Welcome" },
  {
    retry: {
      maxAttempts: 5,
      baseDelayMs: 500,
      maxDelayMs: 5000,
    },
  },
);
```

## Queue processing behavior

- queue validates the callback exists
- tasks are stored in the agent SQLite table
- tasks are processed FIFO by creation time
- successful callbacks are dequeued automatically
- callback errors can be retried if retry options are set
- if callback method no longer exists, the task is skipped/logged

## When not to use Agent queue

Do not use the per-agent queue for global cross-agent fanout, high-throughput multi-tenant queues, or workloads that need independent infrastructure-level queue semantics. Use Cloudflare Queues or Workflows for those.

## Sources

- https://developers.cloudflare.com/agents/api-reference/queue-tasks/
