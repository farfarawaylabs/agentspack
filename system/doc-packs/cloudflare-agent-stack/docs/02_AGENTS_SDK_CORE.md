# Agents SDK Core

## Purpose

Use the Agents SDK when you need a stateful agent instance that can own state, WebSocket connections, schedules, queues, retries, SQL, and lifecycle hooks.

## Important capabilities

- server-side `Agent` class
- named agent instances
- routing helpers
- WebSockets and HTTP/SSE
- client SDK
- callable methods
- state and SQL
- scheduled tasks
- queued tasks
- retries
- durable execution / fibers
- Workflows integration
- MCP integration
- AI model calls
- RAG and browser tools
- observability

## Implementation rule

Before writing any custom infrastructure, check whether the Agents SDK already provides the primitive:

- state sync: `this.state` / `this.setState()`
- method calls: `@callable()`
- future work: `this.schedule()`
- FIFO background work: `this.queue()`
- transient failures: `this.retry()`
- eviction survival: `this.runFiber()`
- model calls: Agents SDK model APIs / AI Gateway / Workers AI

## Naming strategy

Choose agent identity carefully. Common patterns:

- one agent per user
- one agent per workspace
- one agent per task
- one agent per conversation/thread
- one agent per external resource, such as a document, room, or project

The identity determines contention, state size, scheduling scope, and routing complexity.
