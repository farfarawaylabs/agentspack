# Durable Objects, Facets, and Dynamic Workers

## Durable Objects in the agent stack

Agents are Durable-Object-backed. Durable Objects are the stateful coordination primitive for one logical object: a user, room, document, task, agent, session, or generated app.

## Durable Object Facets

Durable Object Facets let you attach dynamically loaded/generated behavior to a Durable Object while preserving stateful coordination and SQLite-backed storage. This is useful for AI-generated mini-apps, custom UIs, or dynamically generated stateful components.

## Dynamic Workers

Dynamic Workers support generated code execution in Worker-like isolates. They are useful when an agent generates code that should run as a small Worker/service rather than simply be interpreted by the agent.

## Use Facets + Dynamic Workers for

- generated mini-apps
- custom per-customer agent surfaces
- agent-created tools with their own UI
- safe dynamic behavior loaded by a supervisor object
- generated apps that need state

## Guardrails

- Treat generated code as untrusted.
- Restrict network access.
- Restrict bindings.
- Version generated code.
- Keep a supervisor object responsible for policy, lifecycle, and state boundaries.
- Never give generated code blanket access to user secrets.

## Source docs

- https://blog.cloudflare.com/durable-object-facets-dynamic-workers/
