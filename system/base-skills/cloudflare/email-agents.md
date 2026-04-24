---
name: cloudflare-email-agents
description: Build Cloudflare Agents that receive and send email as a native channel — using Email Routing, Email Sending, and the Agent `onEmail` hook. Use when the user asks about email-driven agents, inbound email as an agent wakeup, routing email to an agent, thread/reply correlation, reply flows, automated follow-ups when no reply arrives, or mentions `onEmail`, MailChannels, Email Routing, Email Sending. Produces an email-routed-to-agent pattern with `onEmail` parsing, thread correlation, reply sending, and a schedule-based follow-up.
---

# Cloudflare Agents — Email as an Agent Channel

Use this skill when email is a first-class channel for the agent — not just a notification sink. Email Routing delivers inbound mail to an agent, the Agent's `onEmail` hook reacts, and Email Sending/MailChannels replies. Schedule-based follow-ups close the loop when no human replies.

## Mental model

Email for agents is **an asynchronous wakeup event**, not a notification bus. An inbound email can:
- resume a waiting task,
- answer a question the agent previously asked,
- approve/reject a pending step,
- kick off a fresh task.

The agent is the source of truth for thread state; email is just the transport.

## When to use

- Personal assistant tasks that span days (vendors, scheduling, approvals).
- Customer support agents that take tickets by email.
- Vendor / partner follow-ups that need to wait for replies.
- Async workflows where a human responds on their own time.
- User-authenticated flows (reply address + DKIM-verified sender as identity signal).

## When NOT to use

- Real-time chat UIs — use `@callable` over WebSocket.
- One-shot transactional email (password reset) — just send via Email Sending, no agent needed.
- High-volume broadcast — use a dedicated marketing pipeline, not an agent.

## Pattern

1. **Email Route** captures inbound mail on your domain.
2. Route maps sender/thread/recipient → a specific `Agent` instance.
3. `onEmail` parses the message + attachments, correlates to an existing task (or starts one).
4. Agent updates state + SQL to record the interaction.
5. Agent replies via Email Sending when appropriate.
6. Agent `this.schedule(...)`s a follow-up if no response arrives.

## Template — routing inbound email to an agent

```ts
import { getAgentByName } from "agents";
import type { ForwardableEmailMessage } from "@cloudflare/workers-types";

export interface Env {
  SUPPORT_AGENT: DurableObjectNamespace<SupportAgent>;
}

export default {
  async email(message: ForwardableEmailMessage, env: Env) {
    const agentId = resolveAgentId(message);
    const agent = await getAgentByName<Env, SupportAgent>(env.SUPPORT_AGENT, agentId);

    await agent.onEmail({
      from: message.from,
      to: message.to,
      subject: message.headers.get("subject") ?? "",
      messageId: message.headers.get("message-id") ?? crypto.randomUUID(),
      inReplyTo: message.headers.get("in-reply-to") ?? null,
      references: message.headers.get("references")?.split(/\s+/) ?? [],
      raw: await readAll(message.raw),
    });
  },
} satisfies ExportedHandler<Env>;

function resolveAgentId(msg: ForwardableEmailMessage): string {
  // Example: support+<ticket_id>@yourdomain — use the ticket id.
  const local = msg.to.split("@")[0];
  return local.split("+")[1] ?? `from:${msg.from}`;
}

async function readAll(stream: ReadableStream): Promise<string> {
  const reader = stream.getReader();
  const chunks: Uint8Array[] = [];
  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    chunks.push(value);
  }
  const total = chunks.reduce((n, c) => n + c.length, 0);
  const merged = new Uint8Array(total);
  let off = 0;
  for (const c of chunks) { merged.set(c, off); off += c.length; }
  return new TextDecoder().decode(merged);
}
```

## Template — the agent side

```ts
import { Agent } from "agents";

type EmailPayload = {
  from: string;
  to: string;
  subject: string;
  messageId: string;
  inReplyTo: string | null;
  references: string[];
  raw: string;
};

type State = {
  threadStatus: "idle" | "waiting-on-human" | "waiting-on-agent" | "closed";
  lastActivityAt: number;
};

export class SupportAgent extends Agent<Env, State> {
  initialState: State = { threadStatus: "idle", lastActivityAt: 0 };

  async onStart() {
    this.sql`
      CREATE TABLE IF NOT EXISTS messages (
        id           INTEGER PRIMARY KEY AUTOINCREMENT,
        message_id   TEXT UNIQUE NOT NULL,
        in_reply_to  TEXT,
        direction    TEXT NOT NULL,
        from_addr    TEXT NOT NULL,
        subject      TEXT NOT NULL,
        body         TEXT NOT NULL,
        created_at   INTEGER NOT NULL
      )
    `;
  }

  async onEmail(e: EmailPayload) {
    const existing = this.sql<{ id: number }>`
      SELECT id FROM messages WHERE message_id = ${e.messageId}
    `;
    if (existing.length > 0) return;

    this.sql`
      INSERT INTO messages
        (message_id, in_reply_to, direction, from_addr, subject, body, created_at)
      VALUES
        (${e.messageId}, ${e.inReplyTo}, 'inbound', ${e.from}, ${e.subject}, ${e.raw}, ${Date.now()})
    `;

    this.setState({
      ...this.state,
      threadStatus: "waiting-on-agent",
      lastActivityAt: Date.now(),
    });

    const reply = await this.composeReply(e);
    await this.sendReply(reply, e);
    await this.scheduleFollowUp();
  }

  private async composeReply(_e: EmailPayload) {
    return { subject: "Re: …", body: "…" };
  }

  private async sendReply(reply: { subject: string; body: string }, e: EmailPayload) {
    const replyMessageId = `<${crypto.randomUUID()}@yourdomain>`;
    await this.env.EMAIL_SEND.send({
      from: `support+${this.name}@yourdomain`,
      to: e.from,
      subject: reply.subject,
      headers: {
        "Message-ID": replyMessageId,
        "In-Reply-To": e.messageId,
        References: [...e.references, e.messageId].join(" "),
      },
      body: reply.body,
    });

    this.sql`
      INSERT INTO messages
        (message_id, in_reply_to, direction, from_addr, subject, body, created_at)
      VALUES
        (${replyMessageId}, ${e.messageId}, 'outbound',
         ${"support@yourdomain"}, ${reply.subject}, ${reply.body}, ${Date.now()})
    `;

    this.setState({
      ...this.state,
      threadStatus: "waiting-on-human",
      lastActivityAt: Date.now(),
    });
  }

  private async scheduleFollowUp() {
    await this.schedule(3 * 24 * 60 * 60, "followUpIfQuiet", { since: Date.now() });
  }

  async followUpIfQuiet(payload: { since: number }) {
    if (this.state.lastActivityAt > payload.since) return;
    if (this.state.threadStatus !== "waiting-on-human") return;
    await this.env.EMAIL_SEND.send({
      from: `support+${this.name}@yourdomain`,
      to: "…",
      subject: "Checking in",
      body: "Just following up — did you get a chance to look at this?",
    });
  }
}
```

## Thread correlation rules

- Use `Message-ID` / `In-Reply-To` / `References` headers, **not subject matching** — "Re:" games break badly.
- Persist every observed `Message-ID` so replays/loops don't duplicate work.
- The **local-part plus tag** (`support+<id>@domain`) pattern is the easiest routing key — survives forwards and multi-party threads.
- Store direction (`inbound` / `outbound`) so the full thread is reconstructable from `this.sql`.

## Follow-ups and give-ups

- After sending a reply, schedule a follow-up a few days out.
- When the follow-up fires, check `lastActivityAt` — if a human has replied since, cancel silently.
- Cap follow-ups (2–3 max). Don't badger people.

## Common mistakes this skill prevents

- Correlating by subject — broken the moment someone changes "Re:" to "Fwd:" or a mail client rewrites it.
- Storing raw email bodies only in `this.state` — they're big and private. Keep in `this.sql`.
- Ignoring the `Message-ID` dedup check — Email Routing can deliver the same message twice.
- Replying from a bare `support@` address — use `support+<id>@` so replies come back to the right agent instance.
- Infinite follow-up chains — always cap and check `lastActivityAt`.
- Treating email as "just notifications" — you miss the agent-resumption pattern that makes multi-day tasks work.

## See also

- `.agentspack/docs/cloudflare-agent-stack/14_EMAIL_AGENTS.md` — deeper reference.
- `cloudflare-scheduled-tasks` — follow-up scheduling.
- `cloudflare-agent-state-and-sql` — thread history storage patterns.
- Official: https://developers.cloudflare.com/agents/api-reference/email/ and https://blog.cloudflare.com/email-for-agents/
