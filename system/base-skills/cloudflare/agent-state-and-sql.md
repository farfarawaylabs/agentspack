---
name: cloudflare-agent-state-and-sql
description: Design agent state and local SQL on Cloudflare Agents correctly. Use when the user asks about `this.state`, `this.setState`, `initialState`, `onStateChanged`, `this.sql`, agent SQLite storage, persisting conversation history, storing tool results, tracking task progress, sending progress updates to the UI, or when they hit infinite setState loops, oversized state payloads, or "do I put this in state or SQL?". Produces typed state definitions, safe setState patterns, SQL schema + query examples, and a clear decision rubric so client-visible data stays small and heavy/private data lives in SQL.
---

# Cloudflare Agents — State and SQL

Use this skill whenever persisting data inside an `Agent` subclass. Getting the split right between `this.state` (small, synced to clients) and `this.sql` (larger, local, private) is the single most impactful structural decision in an agent.

## When to use

- Designing the shape of state for a new agent.
- Refactoring an agent whose state has grown too large or whose UI is sluggish.
- Persisting tool results, conversation transcripts, audit logs, or embeddings inside an agent.
- Reacting to state changes without causing infinite `setState` loops.
- Mirroring Workflow progress back into the agent so the UI can follow along.

## When NOT to use (pick the right sibling skill instead)

- Creating the agent class itself → `cloudflare-agents-sdk-core`.
- Exposing methods to clients (stateful reads/writes from the browser) → `cloudflare-callable-methods`.
- Persisting embeddings for retrieval across many documents/tenants → `cloudflare-ai-search-rag`.

## The decision in one table

| Data                                          | Use         | Why                                                              |
| --------------------------------------------- | ----------- | ---------------------------------------------------------------- |
| Task status, progress %, current step         | `this.state` | Small, UI-visible, auto-synced to every connected WebSocket client. |
| Current UI mode, feature flags, session prefs | `this.state` | Client needs to render instantly from the last known snapshot.   |
| Latest result summary (a sentence or number)  | `this.state` | Small, display-facing.                                           |
| Conversation transcripts, long message lists  | `this.sql`   | Grows unbounded; don't ship to every client on every change.     |
| Tool call results, large JSON artifacts       | `this.sql`   | Often per-row; needs indexing/filtering.                         |
| Audit logs, event history                     | `this.sql`   | Append-only; queried by time range.                              |
| Embeddings / raw documents                    | `this.sql` or AI Search | State sync would be catastrophic.                     |
| Secrets (tokens, API keys)                    | Bindings/secrets | State is visible to any connected client.                   |

**Rule of thumb:** if it would be weird to beam this to every connected browser every time any field changes, it does not belong in `this.state`.

## Template — typed state

```ts
import { Agent } from "agents";

export interface Env {
  MY_AGENT: DurableObjectNamespace<TaskAgent>;
}

type TaskState = {
  status: "idle" | "running" | "waiting" | "done" | "error";
  progress: number;
  currentStep?: string;
  lastError?: string;
  updatedAt: number;
};

export class TaskAgent extends Agent<Env, TaskState> {
  initialState: TaskState = {
    status: "idle",
    progress: 0,
    updatedAt: Date.now(),
  };

  async startTask(step: string) {
    this.setState({
      ...this.state,
      status: "running",
      progress: 0,
      currentStep: step,
      updatedAt: Date.now(),
    });
  }

  async recordProgress(progress: number, step?: string) {
    this.setState({
      ...this.state,
      progress,
      currentStep: step ?? this.state.currentStep,
      updatedAt: Date.now(),
    });
  }

  async completeTask() {
    this.setState({
      ...this.state,
      status: "done",
      progress: 100,
      updatedAt: Date.now(),
    });
  }
}
```

## Template — SQL for heavy / private data

`this.sql` is a template-literal API over the per-agent SQLite database. Tables are automatically created; migrations are your responsibility.

```ts
export class TaskAgent extends Agent<Env, TaskState> {
  initialState: TaskState = {
    status: "idle",
    progress: 0,
    updatedAt: Date.now(),
  };

  async onStart() {
    this.sql`
      CREATE TABLE IF NOT EXISTS messages (
        id         INTEGER PRIMARY KEY AUTOINCREMENT,
        role       TEXT NOT NULL,
        content    TEXT NOT NULL,
        created_at INTEGER NOT NULL
      )
    `;
    this.sql`
      CREATE INDEX IF NOT EXISTS idx_messages_created_at
      ON messages(created_at)
    `;
  }

  async appendMessage(role: "user" | "assistant" | "tool", content: string) {
    this.sql`
      INSERT INTO messages (role, content, created_at)
      VALUES (${role}, ${content}, ${Date.now()})
    `;
  }

  async recentMessages(limit = 50) {
    return this.sql<{
      id: number;
      role: string;
      content: string;
      created_at: number;
    }>`
      SELECT id, role, content, created_at
      FROM messages
      ORDER BY created_at DESC
      LIMIT ${limit}
    `;
  }
}
```

Notes:
- The tagged template automatically parameterizes values — never interpolate untrusted input with `${}` inside the SQL text itself.
- Use `CREATE TABLE IF NOT EXISTS` and explicit indexes inside `onStart()` so the agent self-heals on cold start.
- Return types can be generics: `this.sql<Row>\`...\`` gives a typed row array back.

## Safe `onStateChanged` (avoid infinite loops)

The #1 mistake: calling `setState` unconditionally inside `onStateChanged`. Always compare before writing, and only write fields that are derived from external signals, not from `state` itself.

```ts
export class TaskAgent extends Agent<Env, TaskState> {
  // ... initialState, methods ...

  onStateChanged(next: TaskState, _source: unknown) {
    // OK: react to terminal states by doing side-effects (no setState).
    if (next.status === "done" || next.status === "error") {
      this.notifyCompletion(next).catch(console.error);
    }

    // BAD — do NOT do this:
    // this.setState({ ...next, updatedAt: Date.now() });   // infinite loop

    // If you must derive a state field, guard on value change:
    // if (next.progress === 100 && next.status !== "done") {
    //   this.setState({ ...next, status: "done" });
    // }
  }

  private async notifyCompletion(state: TaskState) {
    // e.g. post webhook, append row to SQL, etc.
  }
}
```

## Pattern — Workflow progress → agent state

When a Workflow does the durable work, let it call back into the agent via a callable method and write small progress updates into `this.state`. The UI watches agent state; the Workflow stays the source of truth for step-level retries.

```ts
// Inside the agent
async reportWorkflowProgress(step: string, progress: number) {
  this.setState({
    ...this.state,
    status: progress >= 100 ? "done" : "running",
    progress,
    currentStep: step,
    updatedAt: Date.now(),
  });
}
```

The Workflow calls `await agentStub.reportWorkflowProgress(...)` at each step boundary — see `cloudflare-workflows-with-agents` for the orchestration side.

## Common mistakes this skill prevents

- **Packing transcripts into `this.state`** — every message replay re-syncs the entire blob to every client. Use `this.sql`.
- **Stashing secrets in state** — state is readable by any connected client. Use bindings / secret store / scoped tokens.
- **Unbounded `setState` from inside `onStateChanged`** — classic infinite loop. Guard or move to a side-effect.
- **Skipping `initialState`** — any method that reads `this.state.x` on a fresh instance will see `undefined` and crash.
- **Schema drift** — always define tables with `CREATE TABLE IF NOT EXISTS` inside `onStart()`; don't assume previous code created them.
- **String-interpolating user input into SQL text** — use the `${value}` parameterization; never build the SQL string yourself.

## See also

- `.agentspack/docs/cloudflare-agent-stack/03_STORE_AND_SYNC_STATE.md` — deeper reference with sources.
- `.agentspack/docs/cloudflare-agent-stack/18_ANTI_PATTERNS.md` — state-related pitfalls.
- `cloudflare-agents-sdk-core` — class/wrangler scaffolding.
- `cloudflare-callable-methods` — exposing state reads/writes to clients.
- Official: https://developers.cloudflare.com/agents/api-reference/store-and-sync-state/
