# Anti-Patterns

## Anti-pattern: using `setTimeout` for durable tasks

Do not build long-running agent logic around in-memory timers. Use schedules, queues, fibers, or Workflows.

## Anti-pattern: putting everything in Agent state

Agent state syncs to clients and should be small. Use SQL/R2/AI Search for larger data.

## Anti-pattern: exposing all methods as `@callable()`

Callable methods are external API surface. Expose only deliberate client-safe methods.

## Anti-pattern: using `@callable()` for internal RPC

Use Durable Object RPC / agent stubs for Worker-to-Agent and Agent-to-Agent calls.

## Anti-pattern: using fibers instead of Workflows for business processes

Fibers are for agent-internal durable execution. Workflows are better for explicit durable multi-step orchestration.

## Anti-pattern: no idempotency around retries

Retries can duplicate side effects. Use idempotency keys for emails, payments, posts, writes, and external API actions.

## Anti-pattern: stashing huge data in fibers

`stash()` replaces a snapshot in SQLite. Store large artifacts externally and stash references.

## Anti-pattern: browser automation for everything

Use Browser Run only when a browser is needed. Prefer APIs/fetch for simple flows.

## Anti-pattern: generated code with broad secrets

Dynamic Workers, Facets, and Sandboxes must receive minimal scoped permissions.
