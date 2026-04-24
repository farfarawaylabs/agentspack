---
name: cloudflare-ui-controlled-agent-pattern
description: Blueprint for an agent whose state is rendered live in a browser/mobile UI and whose actions are triggered by UI buttons calling agent methods over WebSocket. Use when the user asks for "a live UI for my agent", "a React app that talks to an agent", "let the UI trigger agent actions", "queue background work from a button click", "stream progress to the browser", "agents with real-time state sync", or any recipe that combines client UI + live state + user-triggered actions + expensive background work. Produces a correctly-shaped Agent with @callable actions, Zod-validated inputs, a per-agent task queue for expensive work, progress written back to state, and sample client code that subscribes to state + invokes the stub.
---

# Recipe: UI-controlled agent

Use this recipe when a human operator drives an agent from a UI and needs to see state change in real time. The agent must accept user actions (validated, authorized), kick off potentially-expensive work, and surface progress back to every connected client — without the client having to poll.

## When to use

- You're building a live "control panel" UI for one agent (e.g. per-project assistant, per-chat copilot, per-doc workspace).
- The agent has multiple discrete actions the user triggers (start, pause, retry, submit, approve…).
- User actions sometimes kick off slow work (AI generation, long tool calls, file processing).
- Multiple clients may connect simultaneously and should see the same state.

## When NOT to use

- External service → agent via HTTP webhook → `cloudflare-agents-sdk-core` + plain `fetch` handler.
- Server-side Worker → agent RPC → Durable Object stub (`getAgentByName`), not `@callable`.
- Long-running external wait (email / human-in-the-loop reply) → `cloudflare-long-running-assistant-pattern`.
- Many independent tasks with orchestration visibility → `cloudflare-workflows-with-agents`.

## Primitives used

| Primitive                                  | Role in this recipe                                           |
| ------------------------------------------ | ------------------------------------------------------------- |
| `cloudflare-agents-sdk-core`               | Scaffold + identity (per project/workspace/thread).           |
| `cloudflare-agent-state-and-sql`           | `state` = what the UI renders; `this.sql` = large history.    |
| `cloudflare-callable-methods`              | Every UI action = one `@callable` with Zod validation.        |
| `cloudflare-queued-tasks`                  | Expensive work queued behind the callable so UI stays snappy. |
| `cloudflare-retries-with-idempotency`      | Safe retry of side effects triggered from the UI.             |
| Agents React client (`useAgent`, `useAgentState`) | Subscribe to state + call the stub.                     |

## State-shape rule

State is **what the UI renders**. Keep it:

- small (ships over WebSocket on every change),
- serializable (plain JSON),
- denormalized so the UI can render without extra fetches.

Anything large, historical, or queryable → `this.sql`.

## Blueprint — server

```ts
// src/project-agent.ts
import { Agent, callable } from "agents";
import { z } from "zod";

export interface Env {
  PROJECT_AGENT: DurableObjectNamespace<ProjectAgent>;
}

type Progress = { kind: "idle" } | { kind: "running"; percent: number; label: string } | { kind: "error"; message: string };

type State = {
  projectId: string;
  title: string;
  items: Array<{ id: string; text: string; done: boolean }>;
  progress: Progress;
  updatedAt: number;
};

const AddItemInput = z.object({ text: z.string().min(1).max(500) });
const ToggleItemInput = z.object({ id: z.string().min(1) });
const GenerateInput = z.object({ prompt: z.string().min(1).max(2000) });

export class ProjectAgent extends Agent<Env, State> {
  initialState: State = {
    projectId: "",
    title: "Untitled",
    items: [],
    progress: { kind: "idle" },
    updatedAt: 0,
  };

  @callable()
  async addItem(input: unknown): Promise<void> {
    const { text } = AddItemInput.parse(input);
    this.setState({
      ...this.state,
      items: [...this.state.items, { id: crypto.randomUUID(), text, done: false }],
      updatedAt: Date.now(),
    });
  }

  @callable()
  async toggleItem(input: unknown): Promise<void> {
    const { id } = ToggleItemInput.parse(input);
    this.setState({
      ...this.state,
      items: this.state.items.map((it) => (it.id === id ? { ...it, done: !it.done } : it)),
      updatedAt: Date.now(),
    });
  }

  @callable()
  async generate(input: unknown): Promise<void> {
    const { prompt } = GenerateInput.parse(input);

    this.setState({
      ...this.state,
      progress: { kind: "running", percent: 0, label: "queued" },
      updatedAt: Date.now(),
    });

    await this.queue("runGeneration", { prompt });
  }

  async runGeneration({ prompt }: { prompt: string }): Promise<void> {
    try {
      this.setState({ ...this.state, progress: { kind: "running", percent: 10, label: "thinking" }, updatedAt: Date.now() });

      const drafted = await this.retry(() => draftItemsFromPrompt(prompt), { key: `draft-${this.name}-${prompt.slice(0, 24)}` });

      this.setState({ ...this.state, progress: { kind: "running", percent: 70, label: "saving" }, updatedAt: Date.now() });

      this.setState({
        ...this.state,
        items: [...this.state.items, ...drafted.map((text) => ({ id: crypto.randomUUID(), text, done: false }))],
        progress: { kind: "idle" },
        updatedAt: Date.now(),
      });
    } catch (err) {
      this.setState({
        ...this.state,
        progress: { kind: "error", message: err instanceof Error ? err.message : "unknown error" },
        updatedAt: Date.now(),
      });
    }
  }
}

async function draftItemsFromPrompt(_prompt: string): Promise<string[]> {
  // Replace with Workers AI / Anthropic / OpenAI / whatever.
  return ["Draft item A", "Draft item B", "Draft item C"];
}
```

## Blueprint — client (React)

```tsx
// web/src/ProjectPanel.tsx
import { useAgent, useAgentState } from "agents/react";
import type { ProjectAgent } from "../../src/project-agent";

export function ProjectPanel({ projectId }: { projectId: string }) {
  const agent = useAgent<ProjectAgent>({ name: projectId, agent: "project-agent" });
  const state = useAgentState<ProjectAgent>({ agent });

  if (!state) return <div>connecting…</div>;

  return (
    <div>
      <h1>{state.title}</h1>

      <ProgressBar progress={state.progress} />

      <ul>
        {state.items.map((item) => (
          <li key={item.id}>
            <input type="checkbox" checked={item.done} onChange={() => agent.stub.toggleItem({ id: item.id })} />
            {item.text}
          </li>
        ))}
      </ul>

      <AddItemInput onAdd={(text) => agent.stub.addItem({ text })} />

      <GenerateButton
        disabled={state.progress.kind === "running"}
        onGenerate={(prompt) => agent.stub.generate({ prompt })}
      />
    </div>
  );
}
```

## Common mistakes this recipe prevents

- Doing expensive work **inside** the callable — UI hangs until it finishes.
- Returning computed results from a callable that should instead be observed via state — clients not watching miss the result.
- Putting full history / logs / message lists into `state` — blows up every state-sync payload.
- Skipping Zod parsing — a hostile client can put anything on the wire.
- Performing side effects directly in callables without `this.retry()` + idempotency keys — UI-triggered double-clicks become duplicate API calls.
- Forgetting per-method authorization — any connected client can call any callable.

## See also

- `cloudflare-callable-methods` — @callable details, validation, and auth.
- `cloudflare-queued-tasks` — per-agent FIFO queue for background work.
- `cloudflare-agent-state-and-sql` — keeping state small, SQL large.
- `cloudflare-long-running-assistant-pattern` — when the work waits on humans, not just compute.
- `.agentspack/docs/cloudflare-agent-stack/17_PATTERNS_AND_RECIPES.md` — prose version.
