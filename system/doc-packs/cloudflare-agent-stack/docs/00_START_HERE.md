# Cloudflare Agent Knowledge Pack

Last updated: 2026-04-24

This folder is designed for coding agents such as Codex, Claude Code, Cursor, and similar tools. It is not a marketing summary. It is an operating manual for building AI-agent systems on Cloudflare using the current Agents platform.

## What this pack covers

- Agents SDK core architecture
- routing, sessions, WebSockets, HTTP/SSE, protocol messages
- store and sync state
- callable methods
- scheduled tasks
- queued tasks
- retries
- durable execution with `keepAlive`, `keepAliveWhile`, and `runFiber`
- Workflows integration
- Durable Objects, Facets, and Dynamic Workers
- AI Search and RAG
- Browser Run
- Sandboxes
- Email agents
- MCP and external tools
- security and permission boundaries
- recipes and anti-patterns

## Mental model

Cloudflare agent apps usually combine several primitives:

- **Worker**: public HTTP/webhook/API edge entry point.
- **Agent instance**: a named stateful Durable Object-backed agent/session/thread.
- **Agent state**: small synchronized persistent JSON state.
- **Agent SQL**: per-agent durable SQLite tables for larger local data.
- **Callable methods**: WebSocket RPC exposed to external clients.
- **Schedules**: future, cron, or interval callbacks stored in the agent.
- **Queues**: FIFO per-agent background callbacks.
- **Retries**: exponential backoff with jitter for transient failures.
- **Fibers**: agent-internal durable execution that survives Durable Object eviction.
- **Workflows**: durable multi-step orchestration outside the immediate agent activation.
- **AI Search**: managed search/RAG primitive.
- **Browser Run**: managed browser control for agents.
- **Sandboxes**: persistent isolated computer-like environments for code execution and agent workspaces.

## How to use this folder

Start with `01_DECISION_GUIDE.md`. It tells you which primitive to choose. Then read only the topic docs relevant to the implementation. Do not load every file unless the context window is large enough.

## Source docs

Each file includes source links. When implementation details matter, verify against the official docs before writing final code.
