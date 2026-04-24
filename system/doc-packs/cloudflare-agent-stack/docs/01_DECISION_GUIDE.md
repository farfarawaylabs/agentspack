# Decision Guide

Use this file to choose Cloudflare primitives before coding.

## Quick selection table


| Need                                                | Use                                           | Avoid                                            |
| --------------------------------------------------- | --------------------------------------------- | ------------------------------------------------ |
| Public HTTP/webhook/API route                       | Worker route + Agents routing helpers         | A giant stateful Worker                          |
| Stateful user/session/task/thread                   | Agents SDK / Durable Object-backed Agent      | Stateless Worker with in-memory maps             |
| Small UI-visible state synced to clients            | `this.state` + `this.setState()`              | Custom WebSocket broadcast layer                 |
| Larger per-agent internal data                      | `this.sql`                                    | Putting large arrays in state                    |
| Browser/mobile/client calls into agent methods      | `@callable()` over WebSocket                  | Raw ad-hoc message protocols                     |
| Worker-to-Agent call or Client to agent call        | Durable Object RPC / agent stub               | `@callable()`                                    |
| Agent-to-Agent call                                 | Durable Object RPC / `getAgentByName()`       | WebSocket callable methods                       |
| Run callback in future                              | `this.schedule(delay/date/cron)`              | `setTimeout` loops                               |
| Recurring dynamic loop                              | reschedule from callback or `scheduleEvery()` | long-running polling loop                        |
| FIFO background work inside one agent               | `this.queue()`                                | external queue unless cross-agent/global needed  |
| Transient network/API failure                       | `this.retry()` or retry options               | manual fragile retry loops                       |
| Agent-internal long work surviving DO eviction      | `runFiber()` with `ctx.stash()`               | in-memory loop only                              |
| Multi-step durable workflow with retries/visibility | Workflows                                     | Fibers for everything                            |
| Managed search/RAG                                  | AI Search                                     | bespoke vector+keyword infra by default          |
| Real browser interaction                            | Browser Run                                   | fetch-only scraping for JS-heavy flows           |
| Code execution / filesystem / terminal              | Sandboxes                                     | trying to emulate a shell in Workers             |
| Generated stateful app surfaces                     | Durable Object Facets + Dynamic Workers       | redeploying static Workers for each generated UI |
| Email-native agent                                  | Email Routing + Email Sending + `onEmail`     | polling an inbox manually                        |


## State vs SQL

Use Agent state for current status, progress, small config, and synchronized UI state. Use Agent SQL for tables, history, logs, queue metadata, conversation transcripts, tool call records, and data that should not be broadcast to every client.

## Schedules vs queues vs retries vs fibers vs workflows

- **Schedule**: “run this callback later/at a time/on a cron.”
- **Queue**: “run these tasks FIFO when the agent can process them.”
- **Retry**: “retry this failing thing with backoff and jitter.”
- **Fiber**: “this agent is doing work now, but eviction should not lose progress.”
- **Workflow**: “this process is a durable multi-step orchestration that should be externally visible and recoverable.”

## The most common architecture for your apps

For long-running personal assistant / operating-agent tasks:

1. Ingress Worker receives user message, webhook, email, or HTTP request.
2. Routing resolves the agent identity: user, workspace, thread, task, or resource.
3. Agent instance owns local state and live connections.
4. Agent updates `this.state` for visible progress.
5. Agent persists detailed records in `this.sql`.
6. Agent uses `@callable()` for UI commands.
7. Agent uses `this.schedule()` for timed follow-ups.
8. Agent uses `this.queue()` for internal background tasks.
9. Agent wraps fragile external APIs with `this.retry()`.
10. Agent uses `runFiber()` for long internal loops that need checkpoint recovery.
11. Agent starts Workflows for independent multi-step processes.
12. Agent uses AI Search, Browser Run, Sandboxes, Email, and MCP as tools.

## Strong default rules

- Never store task truth only inside the LLM context.
- Never rely on in-memory state for work that may outlive a single request.
- Never use `setTimeout` as the core scheduler for agent workflows.
- Do not expose all agent methods as callable. Callable methods are an API surface.
- Do not use `@callable()` for internal RPC between Workers/Agents.
- Do not use Agent state as a database.
- Do not use Workflows when a simple delayed callback is enough.
- Do not use fibers as a substitute for Workflows when the task needs independent orchestration, external visibility, or per-step retries.

