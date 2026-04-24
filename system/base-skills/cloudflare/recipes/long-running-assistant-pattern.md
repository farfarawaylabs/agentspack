---
name: cloudflare-long-running-assistant-pattern
description: Blueprint for an agent that accepts a user request, performs a step that requires waiting on an external human (email reply, approval, callback), and then resumes automatically to reply. Use when the user asks for "an assistant that emails someone and waits", "a task that can take hours or days", "handle slow human responses", "wait for a reply and continue", "agent that sends an email and follows up", or any recipe that combines chat/email input + external wait + eventual reply. Produces a UserTaskAgent that wires together SQL task records, a status state machine, outbound email, a scheduled nudge, inbound email correlation, and either a fiber or Workflow for the resume path — with the right primitive picked at each step.
---

# Recipe: Long-running assistant task with external email replies

Use this recipe when a single user request triggers work that cannot finish in one Worker invocation because it is blocked on an **external human**. The agent must:

- take the request,
- kick off the external interaction,
- go quiet,
- wake up when the external side responds (or a timeout fires),
- and reply to the user.

## When to use

- User request can take minutes, hours, or days (not seconds).
- The wait is blocked on a third party (vendor, teammate, lead, customer) responding via email / webhook / scheduled check-in.
- Each task has a stable correlation ID so inbound replies can be routed back to the same agent.
- You want guaranteed resumption even across Worker restarts / code deploys.

## When NOT to use

- Pure chat with no external waits → `cloudflare-agents-sdk-core` + `cloudflare-callable-methods`.
- Many parallel tasks per user that need independent retries and external visibility → `cloudflare-workflows-with-agents`.
- Fire-and-forget background work with no correlation → `cloudflare-queued-tasks`.
- Periodic polling with no correlated reply → `cloudflare-scheduled-tasks` alone.

## Primitives used (and why)

| Primitive                                          | Role in this recipe                                              |
| -------------------------------------------------- | ---------------------------------------------------------------- |
| `cloudflare-agents-sdk-core`                       | Identity = per (user × task). Stable name = correlation key.     |
| `cloudflare-agent-state-and-sql`                   | `state` holds the status FSM. `this.sql` holds the task record + message log. |
| `cloudflare-callable-methods`                      | Entry point from UI / API; also used to cancel or re-poke a task. |
| `cloudflare-email-agents`                          | Outbound email → third party; inbound email → this same agent.   |
| `cloudflare-scheduled-tasks`                       | Timeout / nudge if the reply never arrives.                      |
| `cloudflare-durable-execution-fibers`              | Resume path when reply arrives — so follow-up work survives eviction. |
| `cloudflare-retries-with-idempotency`              | Guard every outbound effect (send email, reply to user).         |

## Status state machine

```
idle ─▶ running ─▶ waiting_for_email ─▶ resuming ─▶ done
                              │                  │
                              └──▶ timed_out ────┘
                                         │
                                         └──▶ failed
```

- Drive transitions only from callables, inbound-email handlers, or scheduled callbacks — never mid-fiber.
- Persist `status` in `this.state` so the UI can observe progress live via state sync.

## Blueprint

```ts
// src/user-task-agent.ts
import { Agent, callable, unstable_getSchedulePayload as getSchedule } from "agents";
import { z } from "zod";

export interface Env {
  USER_TASK_AGENT: DurableObjectNamespace<UserTaskAgent>;
  EMAIL: { send(to: string, subject: string, body: string, headers?: Record<string, string>): Promise<void> };
}

type Status =
  | "idle"
  | "running"
  | "waiting_for_email"
  | "resuming"
  | "timed_out"
  | "done"
  | "failed";

type State = {
  status: Status;
  userId: string;
  taskId: string;
  subject: string;
  createdAt: number;
  lastUpdateAt: number;
  lastError?: string;
};

const StartInput = z.object({
  userId: z.string().min(1),
  subject: z.string().min(1).max(200),
  vendorEmail: z.string().email(),
  ask: z.string().min(1).max(4000),
});

// Name strategy: "<userId>:<taskId>" — stable correlation ID used as the
// In-Reply-To / Message-Id tag when mailing out.
export class UserTaskAgent extends Agent<Env, State> {
  initialState: State = {
    status: "idle",
    userId: "",
    taskId: "",
    subject: "",
    createdAt: 0,
    lastUpdateAt: 0,
  };

  async onStart() {
    this.sql`
      CREATE TABLE IF NOT EXISTS task_record (
        id TEXT PRIMARY KEY,
        vendor_email TEXT NOT NULL,
        ask TEXT NOT NULL,
        reply TEXT,
        reply_received_at INTEGER
      );
      CREATE TABLE IF NOT EXISTS message_log (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        ts INTEGER NOT NULL,
        direction TEXT NOT NULL,
        body TEXT NOT NULL
      );
    `;
  }

  @callable()
  async start(input: z.infer<typeof StartInput>): Promise<{ taskId: string }> {
    const { userId, subject, vendorEmail, ask } = StartInput.parse(input);
    const taskId = this.name;

    if (this.state.status !== "idle") {
      throw new Error(`task ${taskId} already started (status=${this.state.status})`);
    }

    this.sql`INSERT INTO task_record (id, vendor_email, ask) VALUES (${taskId}, ${vendorEmail}, ${ask})`;
    this.sql`INSERT INTO message_log (ts, direction, body) VALUES (${Date.now()}, 'user_in', ${ask})`;

    this.setState({
      ...this.state,
      userId,
      taskId,
      subject,
      status: "running",
      createdAt: Date.now(),
      lastUpdateAt: Date.now(),
    });

    await this.retry(
      async () => {
        await this.env.EMAIL.send(vendorEmail, subject, ask, {
          "Message-ID": `<task-${taskId}@agentspack.local>`,
          "X-Task-Id": taskId,
        });
      },
      { key: `send-initial-${taskId}` },
    );

    this.sql`INSERT INTO message_log (ts, direction, body) VALUES (${Date.now()}, 'vendor_out', ${ask})`;

    this.setState({
      ...this.state,
      status: "waiting_for_email",
      lastUpdateAt: Date.now(),
    });

    await this.schedule({ in: "72h" }, "checkTimeout", { taskId });

    return { taskId };
  }

  // Inbound email handler — wire your email binding to call this.
  async onInboundEmail(email: { from: string; body: string; headers: Record<string, string> }) {
    const taskId = email.headers["x-task-id"] ?? parseTaskIdFromInReplyTo(email.headers["in-reply-to"]);
    if (!taskId || taskId !== this.name) return;
    if (this.state.status !== "waiting_for_email") return;

    this.sql`UPDATE task_record SET reply = ${email.body}, reply_received_at = ${Date.now()} WHERE id = ${taskId}`;
    this.sql`INSERT INTO message_log (ts, direction, body) VALUES (${Date.now()}, 'vendor_in', ${email.body})`;

    this.setState({
      ...this.state,
      status: "resuming",
      lastUpdateAt: Date.now(),
    });

    await this.runFiber("resumeWithReply", { reply: email.body });
  }

  async resumeWithReply({ reply }: { reply: string }): Promise<void> {
    const summary = await summarizeForUser(reply);
    await this.stash("summary", summary);

    await this.retry(
      async () => this.notifyUser(this.state.userId, this.state.subject, summary),
      { key: `notify-user-${this.name}` },
    );

    this.sql`INSERT INTO message_log (ts, direction, body) VALUES (${Date.now()}, 'user_out', ${summary})`;

    this.setState({
      ...this.state,
      status: "done",
      lastUpdateAt: Date.now(),
    });
  }

  async checkTimeout({ taskId }: { taskId: string }): Promise<void> {
    if (this.state.status !== "waiting_for_email") return;

    this.setState({
      ...this.state,
      status: "timed_out",
      lastUpdateAt: Date.now(),
      lastError: "vendor did not reply within 72h",
    });

    await this.retry(
      async () =>
        this.notifyUser(
          this.state.userId,
          this.state.subject,
          `We didn't hear back on "${this.state.subject}" within 72h. I'll keep the thread open — reply to this message to nudge them again.`,
        ),
      { key: `notify-timeout-${taskId}` },
    );
  }

  @callable()
  status(): State {
    return this.state;
  }

  private async notifyUser(userId: string, subject: string, body: string): Promise<void> {
    // Replace with your user-channel binding (push, email, chat, in-app, etc.).
    await this.env.EMAIL.send(`${userId}@your-domain.example`, `Re: ${subject}`, body);
  }
}

function parseTaskIdFromInReplyTo(header: string | undefined): string | undefined {
  if (!header) return undefined;
  const match = /<task-([^@]+)@/.exec(header);
  return match?.[1];
}

async function summarizeForUser(reply: string): Promise<string> {
  // Replace with your preferred AI summarization (Workers AI, Anthropic, etc.).
  return reply.length > 500 ? `${reply.slice(0, 500)}…` : reply;
}
```

## Variations

- Replace the single `resumeWithReply` with `cloudflare-workflows-with-agents` if the resume path is multi-step, has many retries, or needs external visibility.
- Drop `runFiber` for a simple path if resumption fits in one synchronous handler.
- Swap email for Slack / webhooks / SMS — the shape (correlation ID + state FSM + scheduled timeout + resumption) does not change.

## Common mistakes this recipe prevents

- Correlating inbound replies by `from` address instead of a task-owned ID — breaks when the vendor replies from a teammate.
- Storing the message transcript in `state` — blows up state sync; keep transcripts in `this.sql`.
- Forgetting the timeout schedule — stuck tasks live forever.
- Re-sending emails on every retry without an idempotency key — vendor sees 5 identical pings.
- Kicking off `runFiber` from inside another fiber without stashing first — state loss on restart.
- Using Workflows for a single agent-local resume — adds external orchestration weight for no benefit.

## See also

- `cloudflare-agents-sdk-core` — identity & scaffold.
- `cloudflare-agent-state-and-sql` — state vs SQL split.
- `cloudflare-email-agents` — inbound/outbound wiring.
- `cloudflare-durable-execution-fibers` — fiber stash/resume.
- `cloudflare-retries-with-idempotency` — safe external effects.
- `cloudflare-workflows-with-agents` — when to upgrade beyond a single fiber.
- `.agentspack/docs/cloudflare-agent-stack/17_PATTERNS_AND_RECIPES.md` — prose version.
