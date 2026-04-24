# Retries

## What it is

The Agents SDK provides retry support with exponential backoff and jitter. Retries are available through `this.retry()` and as retry options on scheduled and queued tasks.

## Use retries for

- flaky external APIs
- transient network errors
- temporary rate limits
- background callbacks that may fail briefly
- model/provider calls where retry is safe

## Basic pattern

```ts
const data = await this.retry(async (attempt) => {
  const res = await fetch("https://api.example.com/data");
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
});
```

## Custom options

```ts
const data = await this.retry(
  async (attempt) => {
    const res = await fetch("https://slow-api.example.com/data");
    if (!res.ok) throw new Error(`HTTP ${res.status}`);
    return res.json();
  },
  {
    maxAttempts: 5,
    baseDelayMs: 500,
    maxDelayMs: 10_000,
    shouldRetry: (err, nextAttempt) => {
      // Return false for permanent errors.
      return nextAttempt <= 5;
    },
  },
);
```

## Queue/schedule retry pattern

```ts
await this.schedule(60, "process", { id }, {
  retry: { maxAttempts: 3, baseDelayMs: 500, maxDelayMs: 5000 },
});

await this.queue("sendEmail", { to }, {
  retry: { maxAttempts: 5 },
});
```

## Important behavior

- Default `this.retry()` retries up to three times.
- Retry options are validated eagerly when calling `this.retry()`, `queue()`, `schedule()`, or `scheduleEvery()`.
- If queued/scheduled callback retries are exhausted, the error is logged and the task is dequeued.
- To disable retries for a specific task, set `maxAttempts: 1`.

## Coding-agent rules

- Do not retry non-idempotent operations blindly.
- Add idempotency keys for payments, emails, writes, and external side effects.
- Do not retry validation errors or permanent authorization errors.
- Consider rate-limit response headers when designing `shouldRetry`.

## Sources

- https://developers.cloudflare.com/agents/api-reference/retries/
