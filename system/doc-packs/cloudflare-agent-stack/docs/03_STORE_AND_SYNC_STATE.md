# Store and Sync State

## What it is

Agents provide built-in persistent state that is automatically synchronized to connected WebSocket clients. State is saved to SQLite inside each Agent instance and can be accessed through `this.state` and updated with `this.setState()`.

## Use state for

- current task status
- progress percentage
- small user/session settings
- connection-visible flags
- current UI mode
- summary of latest result
- data that should be broadcast to clients

## Do not use state for

- long transcripts
- large tool results
- raw documents
- long audit logs
- vector/RAG data
- secrets
- data that should not be sent to clients

Use `this.sql` for those.

## Key API shape

```ts
import { Agent } from "agents";

type TaskState = {
  status: "idle" | "running" | "waiting" | "done" | "error";
  progress: number;
  currentStep?: string;
  lastError?: string;
};

export class TaskAgent extends Agent<Env, TaskState> {
  initialState: TaskState = {
    status: "idle",
    progress: 0,
  };

  async startTask() {
    this.setState({
      ...this.state,
      status: "running",
      progress: 0,
      currentStep: "starting",
    });
  }

  onStateChanged(state: TaskState, source: unknown) {
    // React to state changes carefully.
    // Avoid setState loops.
  }
}
```

## Coding-agent instructions

- Always define a TypeScript state type for non-trivial agents.
- Always define `initialState` if methods assume `this.state` exists.
- Treat state updates as full replacements or immutable updates.
- Keep state small and serializable.
- Use SQL for relational/history data.
- Be careful in `onStateChanged`; do not call `setState` unconditionally or you may create infinite loops.

## State + Workflows pattern

If a Workflow performs durable work, let it write progress back to the owning Agent state through an explicit agent method. The Agent state becomes the live UI/progress surface; the Workflow remains the durable orchestration engine.

## Sources

- https://developers.cloudflare.com/agents/api-reference/store-and-sync-state/
