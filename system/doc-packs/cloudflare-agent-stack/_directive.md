# Cloudflare Agent Stack

This project may build on Cloudflare's modern agent platform. Cloudflare has shipped many recent primitives for AI agents that are **not in older Workers-only knowledge**: Agents SDK, Workflows v2, Durable Object Facets, Dynamic Workers, Sandboxes, Browser Run, AI Search, Email agents, and fibers/durable execution.

**Do not assume pre-Agents-SDK patterns are still the best answer.** Before designing, coding, or reviewing anything that could use these primitives, use the decision guide below and drill into the detailed docs in `.agentspack/docs/cloudflare-agent-stack/` before making a choice.

## Decision matrix (read first)

| Need                                                | Prefer                                        | Avoid                                       |
| --------------------------------------------------- | --------------------------------------------- | ------------------------------------------- |
| Public HTTP / webhook / API ingress                 | Worker route + Agents routing helpers         | Stateful monolith Workers                   |
| Stateful user / session / task / thread             | Agents SDK (Durable-Object-backed `Agent`)    | Stateless Worker with in-memory maps        |
| Small UI-visible state synced to clients            | `this.state` + `this.setState()`              | Custom WebSocket broadcast                  |
| Larger per-agent internal data                      | `this.sql`                                    | Large arrays in state                       |
| Browser/mobile client → agent RPC                   | `@callable()` over WebSocket                  | Ad-hoc message protocols                    |
| Worker → agent or agent → agent RPC                 | Durable Object RPC / `getAgentByName()`       | `@callable()` (that's for external clients) |
| Run callback in future / cron                       | `this.schedule()` / `scheduleEvery()`         | `setTimeout`, polling loops                 |
| FIFO background work inside one agent               | `this.queue()`                                | External queues for single-agent work       |
| Transient network / API failure                     | `this.retry()` or retry options               | Manual retry loops                          |
| Agent-internal long work surviving DO eviction      | `runFiber()` with `ctx.stash()` checkpoints   | In-memory loops only                        |
| Multi-step durable workflow (retries, visibility)   | Workflows                                     | Fibers for cross-service orchestration      |
| Managed search / RAG                                | AI Search                                     | Bespoke vector+keyword infra by default     |
| Real browser interaction (login, JS-heavy pages)    | Browser Run                                   | Fetch-only scraping for JS flows            |
| Code execution / filesystem / terminal / previews   | Sandboxes                                     | Emulating a shell in Workers                |
| Generated stateful app surfaces                     | Durable Object Facets + Dynamic Workers       | Redeploying static Workers per generated UI |
| Email-native agent                                  | Email Routing/Sending + `onEmail`             | Manually polling an inbox                   |

## Core rules (always apply)

- **Never store task truth only in LLM context.** Persist it in Agent state or `this.sql`.
- **Never rely on in-memory state** for work that may outlive a single request — agents get evicted.
- **Never use `setTimeout`** as the core scheduler for agent work.
- **State vs SQL**: `this.state` is small, synchronized, client-visible. `this.sql` is larger, local, private.
- **`@callable()` is an API surface**, not internal RPC. Validate inputs, check authorization, don't expose every method.
- **Retries need idempotency keys** for emails, payments, writes, and external side effects.
- **Fibers vs Workflows**: fibers for agent-internal recovery; Workflows for independent multi-step processes with external visibility, per-step retries, and long durations.
- **Generated code (Dynamic Workers, Facets, Sandboxes) is untrusted** — minimal bindings, no broad secrets, restricted network access.

## Detailed docs (`.agentspack/docs/cloudflare-agent-stack/`)

Fetch these only when relevant to the current task. The decision guide (`01_DECISION_GUIDE.md`) is the richer version of the matrix above.

| File                                            | Read when                                                                                      |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `00_START_HERE.md`                              | You want the mental model for how Cloudflare agent primitives compose.                         |
| `01_DECISION_GUIDE.md`                          | Choosing between state/SQL/schedule/queue/retry/fiber/Workflow for a specific task.            |
| `02_AGENTS_SDK_CORE.md`                         | Designing agent identity, naming, and which SDK primitives to use.                             |
| `03_STORE_AND_SYNC_STATE.md`                    | Implementing `this.state`, `initialState`, `setState`, `onStateChanged`.                       |
| `04_CALLABLE_METHODS.md`                        | Exposing agent methods to browser/mobile clients, streaming responses, serialization, security. |
| `05_SCHEDULE_TASKS.md`                          | Delayed callbacks, cron, `scheduleEvery`, dynamic polling, cancellation.                       |
| `06_QUEUE_TASKS.md`                             | Per-agent FIFO background work with `this.queue()` and retry options.                          |
| `07_RETRIES.md`                                 | `this.retry()` with backoff/jitter, `shouldRetry`, idempotency, permanent vs transient errors. |
| `08_DURABLE_EXECUTION_FIBERS.md`                | `runFiber`, `stash`, `onFiberRecovered`, `keepAlive` / `keepAliveWhile`, checkpoint patterns.  |
| `09_WORKFLOWS_AND_RUN_WORKFLOWS.md`             | Long-running multi-step durable orchestration + Agent ↔ Workflow patterns.                     |
| `10_DURABLE_OBJECTS_FACETS_DYNAMIC_WORKERS.md`  | Dynamically generated stateful surfaces, mini-apps, generated-code guardrails.                 |
| `11_AI_SEARCH_AND_RAG.md`                       | Document retrieval, hybrid search, per-tenant metadata isolation, citations.                   |
| `12_BROWSER_RUN.md`                             | Real-browser automation, login flows, human-takeover, screenshots.                             |
| `13_SANDBOXES.md`                               | Coding agents, test runs, terminal/filesystem access, generated-app previews.                  |
| `14_EMAIL_AGENTS.md`                            | Inbound/outbound email as an agent channel, `onEmail` correlation, schedule-based follow-ups.  |
| `15_MCP_TOOLS_EXTERNAL_APIS.md`                 | Remote MCP servers, tool schemas, input validation, least privilege for tools.                 |
| `16_SECURITY_AUTH_PERMISSIONS.md`               | AuthN/Z per agent instance, callable-method security, secrets in state, scoped OAuth tokens.   |
| `17_PATTERNS_AND_RECIPES.md`                    | Long-running task + email, UI-controlled agent, research loop, scheduled publishing, mini-apps. |
| `18_ANTI_PATTERNS.md`                           | Common mistakes to avoid before shipping agent code.                                            |
| `19_CHANGELOG.md`                               | What's new in this knowledge pack vs older Cloudflare guidance.                                |

## When writing code on this stack

1. **Pick primitives from the decision matrix first.** If unsure, read `01_DECISION_GUIDE.md`.
2. **Read the relevant topic doc(s)** before writing substantive code — APIs and idioms have changed recently.
3. **Explain tradeoffs in a short comment** if you deliberately chose an older pattern over a newer one.
4. **Verify against official Cloudflare docs** (linked at the bottom of each topic file) before finalizing non-trivial implementations.
