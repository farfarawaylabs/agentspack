---
name: cloudflare-callable-methods
description: Expose selected Cloudflare Agent methods to browser/mobile/external clients over WebSocket RPC with `@callable()`. Use when the user asks to add a UI action, expose an agent method to the client, add "stub" RPC, stream AI output or progress to the browser, validate client payloads into an agent, decide between `@callable()` and Durable Object RPC, or harden an existing agent method that's now being called by untrusted clients. Produces correctly-shaped `@callable()` methods (sync + async + streaming), client-side stub usage, Zod validation, authorization gating, and serialization-safe signatures.
---

# Cloudflare Agents — Callable Methods

Use this skill whenever adding, reviewing, or securing an agent method that a browser/mobile/external client will invoke. `@callable()` is an **API surface**, not an internal RPC shortcut — treat every decorated method as though anyone with a connection can call it.

## When to use

- A browser or mobile client needs to trigger an action on a live agent (send message, start task, confirm step, vote, etc.).
- Streaming AI output, progress updates, or partial results down to a connected client.
- Building a typed client SDK via `agent.stub.<method>()`.
- Exposing agent-controlled writes/reads to the UI (with validation + authz).

## When NOT to use (use something else instead)

| Caller                                       | Use instead of `@callable()`                          |
| -------------------------------------------- | ----------------------------------------------------- |
| A Worker in the same codebase → the agent    | `getAgentByName<Env, MyAgent>(env.MY_AGENT, name)` then call the method directly (typed DO RPC). |
| Another agent → this agent                   | Same: Durable Object stub / `getAgentByName`.         |
| Internal helpers on the same agent instance  | Plain private methods. No decorator.                  |
| Admin / destructive operations               | Only callable if gated behind explicit authorization. Prefer a protected Worker route or a separate admin agent. |

Rule: if the caller is trusted (your own server code), use DO RPC. If the caller is a client you don't control, use `@callable()` **with validation and authorization**.

## Basic pattern

```ts
import { Agent, callable } from "agents";
import { z } from "zod";

export interface Env {
  COUNTER_AGENT: DurableObjectNamespace<CounterAgent>;
}

type State = { count: number };

const AddItemInput = z.object({
  value: z.string().min(1).max(200),
});

export class CounterAgent extends Agent<Env, State> {
  initialState: State = { count: 0 };

  @callable()
  increment(): number {
    this.setState({ ...this.state, count: this.state.count + 1 });
    return this.state.count;
  }

  @callable()
  async addItem(raw: unknown): Promise<{ ok: true; id: number }> {
    const { value } = AddItemInput.parse(raw);
    await this.assertCaller();
    const row = this.sql<{ id: number }>`
      INSERT INTO items (value, created_at)
      VALUES (${value}, ${Date.now()})
      RETURNING id
    `;
    return { ok: true, id: row[0].id };
  }

  private async assertCaller() {
    // Example: check a token set during connection auth, or verify a header
    // stored on the connection. Throw on failure so the client gets a clean error.
  }
}
```

## Client usage

```ts
import { AgentClient } from "agents/client";

const agent = new AgentClient({
  agent: "counter-agent",
  name: "user_42",
});

const next = await agent.stub.increment();
const { id } = await agent.stub.addItem({ value: "hello" });
```

`agent.stub.<method>()` is a typed proxy over a WebSocket — each call becomes one RPC frame. The return value is awaited on the next reply frame.

## Streaming pattern (AI output, progress, partial results)

```ts
import { Agent, callable, type StreamingResponse } from "agents";
import { z } from "zod";

const GenerateInput = z.object({
  prompt: z.string().min(1).max(4_000),
});

export class AIAgent extends Agent<Env> {
  @callable({ streaming: true })
  async generateText(stream: StreamingResponse, raw: unknown) {
    const { prompt } = GenerateInput.parse(raw);

    try {
      for await (const chunk of this.modelStream(prompt)) {
        stream.send(chunk);
      }
      stream.end();
    } catch (err) {
      stream.send({ error: "generation_failed" });
      stream.end();
    }
  }

  private async *modelStream(prompt: string): AsyncIterable<string> {
    // e.g. Workers AI / Vercel AI SDK / custom — yield text chunks.
  }
}
```

Client side:

```ts
for await (const chunk of agent.stub.generateText({ prompt })) {
  ui.append(chunk);
}
```

Notes:
- The first parameter of a streaming callable is always the `StreamingResponse`; your payload starts at parameter 2.
- Always `stream.end()` — on both happy path and error — or the client hangs.
- Do not `throw` after `stream.send(...)`; send a structured error chunk and end.

## Serialization rules (this bites a lot of people)

Arguments and return values must be **JSON-serializable**. In particular, avoid:

| Don't return / accept | Why                          | Use instead                       |
| --------------------- | ---------------------------- | --------------------------------- |
| `Date`                | Becomes a string, inconsistently. | `number` (epoch ms) in/out.       |
| `Map` / `Set`         | Not JSON; silently becomes `{}` / `[]`. | Array of entries or plain object. |
| Functions             | Not transferable.            | Don't.                            |
| Class instances       | Prototype chain is lost; only own enumerable props survive. | Plain object / DTO.         |
| Errors                | `Error.message` drops; stack leaks on some runtimes. | `{ code, message }` object. |
| Streams / `Request`   | Not serializable.            | Use a streaming callable instead. |

## Security checklist (apply to every `@callable()`)

- **Validate input** with Zod/Valibot/etc. Never trust the argument shape.
- **Authorize** inside the method (or during connection setup) — the agent's name alone is not auth.
- **Don't expose broad tool-runners** like `runTool(name, args)` — use strict allowlists per method.
- **Rate-limit destructive ops** with `this.state` counters, `this.sql`, or by caching idempotency keys.
- **Redact** before returning — strip secrets, internal IDs, or server paths from responses.
- **Keep methods focused** — one method per user-intent action. Avoid god-methods.

## Common mistakes this skill prevents

- Using `@callable()` for Worker-to-agent calls (use DO RPC; you get typed methods and no WebSocket detour).
- Treating `@callable()` as private because it "isn't in a public Worker route" — a connected client can still call it.
- Returning a `Date` and getting `"2026-04-24T..."` on the client instead of a number.
- Throwing after streaming started — the client sees a clean close and never learns it was an error. Emit a structured error chunk, then `stream.end()`.
- Skipping Zod because "TypeScript already types the argument" — TypeScript types don't exist at runtime; the client can send anything.
- Exposing one giant callable that takes `{ action, payload }` — this defeats every static safety benefit; keep methods specific.

## See also

- `.agentspack/docs/cloudflare-agent-stack/04_CALLABLE_METHODS.md` — deeper reference.
- `.agentspack/docs/cloudflare-agent-stack/16_SECURITY_AUTH_PERMISSIONS.md` — auth patterns.
- `cloudflare-agents-sdk-core` — getting the base agent class right.
- `cloudflare-agent-state-and-sql` — what to read/write safely from a callable.
- Official: https://developers.cloudflare.com/agents/api-reference/callable-methods/
