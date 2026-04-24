# Patterns and Recipes

## Recipe: Long-running assistant task with email replies

Use when a user asks an agent to handle something that may require third-party email and waiting.

1. User request enters via chat/email/API.
2. Worker routes to `UserTaskAgent` by user ID + task ID.
3. Agent writes `state.status = "running"`.
4. Agent stores task record in SQL.
5. Agent sends email to third party.
6. Agent sets `state.status = "waiting_for_email"`.
7. Agent schedules a follow-up check with `this.schedule()`.
8. Inbound email routes back to same Agent.
9. Agent correlates thread/message to task.
10. Agent resumes with `runFiber()` or starts a Workflow if the process is complex.
11. Agent replies to user and marks done.

## Recipe: UI-controlled agent

1. Browser connects via Agents client SDK.
2. Agent state syncs to UI.
3. UI invokes `@callable()` methods for user actions.
4. Callable method validates input.
5. Agent queues expensive work with `this.queue()`.
6. Queue callback updates state as it progresses.
7. UI updates automatically through state sync.

## Recipe: Durable research loop

Use `runFiber()` when the research loop is internal to one agent and checkpointable.

- stash after search
- stash after crawl
- stash after analysis
- stash before synthesis
- store large artifacts outside state/fiber snapshot

Use Workflows if the research process has many independent steps, retries, or needs external visibility.

## Recipe: Scheduled publishing

1. Channel agent stores publishing config in SQL/state.
2. `this.schedule("0 8 * * *", "publishNext", { channelId })` sets the cadence.
3. `publishNext` checks queue of prepared items.
4. If item exists, publish with retry and idempotency key.
5. If queue empty, trigger generation Workflow.
6. Reschedule dynamically if cadence is not a fixed cron.

## Recipe: Agent-generated mini-app

1. Supervisor Agent receives request.
2. Agent uses Sandbox to generate/test code.
3. Agent stores versioned code/artifacts.
4. Dynamic Worker/Facet serves generated UI or behavior.
5. Supervisor enforces policy and allowed bindings.
6. State is isolated per generated app/resource.
