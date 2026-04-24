---
name: cloudflare-retries-with-idempotency
description: Retry transient failures inside a Cloudflare Agent with `this.retry()`, `shouldRetry`, and queue/schedule retry options — always paired with idempotency keys so payments/emails/writes don't duplicate. Use when the user asks about retrying a flaky API, exponential backoff, jitter, rate-limit retry logic, distinguishing permanent vs transient errors, retrying a queued or scheduled callback, adding idempotency keys to external side effects, or diagnosing duplicate emails/payments. Produces `retry()` usage with `shouldRetry`, retry options on `schedule` / `queue`, and an idempotency-key pattern for writes.
---

# Cloudflare Agents — Retries and Idempotency

Use this skill whenever an agent method calls something that can fail transiently — external APIs, third-party webhooks, model providers, rate-limited services — or whenever background work might retry and duplicate a side effect.

## When to use

- Flaky external APIs or transient network errors.
- Model provider calls that occasionally time out.
- Temporarily rate-limited endpoints (HTTP 429, 503).
- Any queued/scheduled callback that can fail but should recover.
- Paired with **idempotency keys** for emails, payments, writes, webhooks — anything side-effectful.

## When NOT to use

- Validation errors (400), permanent auth errors (401/403 without renewal path), "not found" (404) — these are not transient. Do not retry.
- Long-running multi-step business processes — use a Workflow with per-step retries (`cloudflare-workflows-with-agents`).
- Agent-internal loops that need to survive eviction between steps — use fibers (`cloudflare-durable-execution-fibers`).

## Default `this.retry()`

```ts
const data = await this.retry(async (_attempt) => {
  const res = await fetch("https://api.example.com/data");
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
});
```

Default: up to 3 attempts, exponential backoff with jitter. Good for read-only GETs where "try again soon" is the right answer.

## Custom backoff + `shouldRetry`

Only retry transient errors. Everything else should fail fast.

```ts
const data = await this.retry(
  async (_attempt) => {
    const res = await fetch("https://api.example.com/data");
    if (res.status === 429 || res.status >= 500) {
      throw new TransientError(`HTTP ${res.status}`);
    }
    if (!res.ok) {
      throw new PermanentError(`HTTP ${res.status}`);
    }
    return res.json();
  },
  {
    maxAttempts: 5,
    baseDelayMs: 500,
    maxDelayMs: 10_000,
    shouldRetry: (err, nextAttempt) => {
      if (err instanceof PermanentError) return false;
      return nextAttempt <= 5;
    },
  },
);

class TransientError extends Error {}
class PermanentError extends Error {}
```

Tips:
- Respect `Retry-After` response headers when present — read the header, resolve a promise that sleeps that long, then return to `retry` on next attempt.
- Keep `maxDelayMs` bounded. Infinite retry on an always-broken API is just a slower infinite loop.

## Retry options on `queue()` / `schedule()`

```ts
await this.schedule(60, "process", { id }, {
  retry: { maxAttempts: 3, baseDelayMs: 500, maxDelayMs: 5_000 },
});

await this.queue("sendEmail", { to }, {
  retry: { maxAttempts: 5 },
});
```

If the scheduled or queued callback throws, the agent retries with the configured backoff. When retries are exhausted, the error is logged and the task is dequeued — it does not requeue automatically. Set `maxAttempts: 1` to disable retries for a specific task.

## Idempotency key pattern (required for side effects)

Retries + non-idempotent operations = duplicated emails, doubled payments, ghost DB rows. Always pair them.

```ts
import { z } from "zod";

const SendEmailInput = z.object({
  to: z.string().email(),
  subject: z.string().min(1),
  body: z.string().min(1),
  idempotencyKey: z.string().min(8),
});

export class MailAgent extends Agent<Env> {
  async onStart() {
    this.sql`
      CREATE TABLE IF NOT EXISTS sent_emails (
        idem_key TEXT PRIMARY KEY,
        to_addr  TEXT NOT NULL,
        sent_at  INTEGER NOT NULL
      )
    `;
  }

  async sendEmail(raw: unknown) {
    const input = SendEmailInput.parse(raw);

    const existing = this.sql<{ sent_at: number }>`
      SELECT sent_at FROM sent_emails WHERE idem_key = ${input.idempotencyKey}
    `;
    if (existing.length > 0) {
      return { ok: true, deduped: true, sentAt: existing[0].sent_at };
    }

    await this.retry(async (attempt) => {
      await this.callEmailProvider(input, input.idempotencyKey, attempt);
    }, { maxAttempts: 5 });

    this.sql`
      INSERT OR IGNORE INTO sent_emails (idem_key, to_addr, sent_at)
      VALUES (${input.idempotencyKey}, ${input.to}, ${Date.now()})
    `;

    return { ok: true, deduped: false, sentAt: Date.now() };
  }

  private async callEmailProvider(
    input: z.infer<typeof SendEmailInput>,
    key: string,
    _attempt: number,
  ) {
    // Most providers accept an Idempotency-Key header; include it so
    // even cross-retry duplicates on the provider side collapse.
  }
}
```

Three layers make this safe:
1. **Pre-check**: look up the key before calling the provider.
2. **Provider-level**: include the idempotency key in the provider's header — Stripe, Postmark, SES, etc. all honor this.
3. **Post-write**: insert the key so a later retry is a no-op.

## Error-classification cheat sheet

| Category                 | Retry?                        | Example                                           |
| ------------------------ | ----------------------------- | ------------------------------------------------- |
| Network timeout          | Yes (transient)               | `fetch` rejects, DNS timeout                      |
| HTTP 429 / 503           | Yes, honor `Retry-After`      | Rate limit, temporary unavailability              |
| HTTP 500 / 502 / 504     | Yes (transient)               | Upstream blip                                     |
| HTTP 400                 | No                            | Your payload is wrong; fix before retrying        |
| HTTP 401 / 403           | Usually no; refresh auth once | Expired token → refresh & retry once              |
| HTTP 404                 | No                            | Permanent miss                                    |
| Validation (Zod) failure | No                            | Caller error; return a structured error instead   |

## Common mistakes this skill prevents

- Retrying blindly on 4xx — you just hammer the API while the fix is in your code.
- Retrying non-idempotent writes without a key — duplicate charges, duplicate emails.
- `maxAttempts` without `maxDelayMs` — exponential backoff that eventually sleeps for hours.
- Using `try { … } catch { sleep; retry; }` hand-rolled loops inside the agent — you lose jitter, observability, and the schedule/queue integration.
- Storing the idempotency key in `this.state` — state syncs to clients. Keep it in `this.sql`.

## See also

- `.agentspack/docs/cloudflare-agent-stack/07_RETRIES.md` — deeper reference.
- `cloudflare-queued-tasks` and `cloudflare-scheduled-tasks` — background callbacks where retries live.
- `cloudflare-workflows-with-agents` — for per-step retries across a long process.
- Official: https://developers.cloudflare.com/agents/api-reference/retries/
