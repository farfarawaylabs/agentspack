---
name: cloudflare-platform-development
description: Use when building, designing, or reviewing backend applications on the Cloudflare platform (Workers, Pages, D1, KV, R2, Durable Objects, Queues). Covers Hono framework usage, three-tier architecture, database patterns for D1 and Supabase, edge performance optimization, Vercel AI SDK integration, testing with Jest, and wrangler configuration.
---

# Cloudflare Platform Development

Guidance for building production-grade backend applications on Cloudflare's edge platform using Workers, Hono, and the broader Cloudflare ecosystem.

## When to Use

- Creating or extending a Cloudflare Workers project
- Designing APIs with Hono on Workers
- Integrating D1 (edge SQLite) or Supabase (managed PostgreSQL)
- Adding LLM capabilities via Vercel AI SDK V6+
- Optimizing for edge performance, caching, or global distribution
- Reviewing or refactoring existing Workers code for best practices

## Core Technology Stack

- **Runtime:** Cloudflare Workers (edge, serverless)
- **API Framework:** Hono — always prefer Hono for Workers API development
- **Databases:** D1 (SQLite at the edge) or Supabase (PostgreSQL with managed services)
- **LLM Integration:** Vercel AI SDK V6+ for model-agnostic LLM interactions with streaming support
- **Testing:** Jest with comprehensive unit tests for all functions and endpoints
- **Ecosystem:** KV, R2, Durable Objects, Queues, Analytics Engine, Cache API

## Architecture: Three-Tier Separation

Enforce strict layering in every Workers project:

1. **API Layer (`src/api`)** — Hono routes, middleware, authentication/authorization, input validation, response formatting. This layer must remain thin.
2. **Business Logic Layer (`src/bl`)** — Pure TypeScript functions, domain rules, orchestration, Vercel AI SDK LLM calls. No direct database access.
3. **Data Layer (`src/dl`)** — All database operations (D1 or Supabase), external data-source integrations. Only this layer touches the database.

**Call direction:** API → BL → DL. The API layer must never call the DL layer directly.

## Project Structure

When no existing structure is present, use this layout:

```
src/
├── api/          # One file per resource (customers.ts, orders.ts)
├── bl/           # Subfolder per resource, one file per operation
│   ├── customers/
│   │   ├── addCustomer.ts
│   │   └── deleteCustomer.ts
│   └── orders/
│       ├── createOrder.ts
│       └── cancelOrder.ts
├── dl/           # Mirrors bl/ structure
│   ├── customers/
│   │   ├── addCustomer.ts
│   │   └── deleteCustomer.ts
│   └── orders/
│       ├── createOrder.ts
│       └── cancelOrder.ts
├── models/       # TypeScript interfaces, types, enums
├── services/     # Third-party API integrations
└── utils/        # Reusable helpers (auth, string manipulation, etc.)
test/             # Jest tests mirroring src/ structure
wrangler.toml     # Workers bindings and configuration
```

If a project already has an established structure, follow its conventions instead.

## Development Steps

1. **Identify the database** — Determine whether the project uses D1 or Supabase and follow appropriate patterns.
2. **Consult latest docs** — Use Context7 MCP (or equivalent) for up-to-date Hono, Workers, D1/Supabase, and Vercel AI SDK documentation.
3. **Analyze requirements** — Determine optimal Hono routing, middleware, and Cloudflare bindings.
4. **Design the schema** — For D1: SQLite schema with indexes tuned for edge performance. For Supabase: PostgreSQL schema with RLS policies and efficient query patterns.
5. **Implement the API** — Build Hono routes with proper middleware, validation, and error handling.
6. **Integrate LLMs (if needed)** — Use Vercel AI SDK V6+ for any LLM interactions, ensuring model flexibility and streaming.
7. **Add TypeScript types** — Leverage D1 or Supabase generated types, Hono type definitions, and Vercel AI SDK interfaces.
8. **Write Jest tests** — Cover all functions, methods, and API endpoints with proper mocking for Workers bindings and database access.
9. **Optimize for edge** — Apply caching strategies (KV, Cache API), minimize database round trips, and target sub-50ms response times.
10. **Secure the application** — Implement authentication (JWT, Cloudflare Access, or Supabase Auth), input validation, and LLM prompt-injection protection where applicable.

## Key Rules

- **Environment variables via `cloudflare:workers`** — Access env vars and bindings from anywhere using the `cloudflare:workers` module. This eliminates the need to thread the `env` parameter through every function call:

```typescript
import { env } from "cloudflare:workers";
const apiKey = env.API_KEY;
const db = env.DB; // D1 binding
```

- **Database access isolation** — Only code in the DL tier may access the database directly.
- **Thin API layer** — The API tier handles requests, auth, validation, and responses. All logic lives in BL.
- **Edge constraints** — Design for 128MB memory limit, CPU time budgets, and payload size limits. Write stateless, idempotent functions that scale to zero.
- **Cold start targets** — Aim for <5ms cold starts; minimize top-level imports and initialization.
- **Wrangler configuration** — Keep bindings (D1, KV, R2, Durable Objects) properly declared in `wrangler.toml`.
- **Database-appropriate patterns** — Use D1 for edge-optimized, latency-sensitive workloads; Supabase for complex relational features, auth, and real-time subscriptions.
- **Error handling** — Return proper HTTP status codes, structured error responses, and never leak internal details.
- **Check existing code first** — Before writing new code, review the codebase for established authentication patterns, database access methods, environment bindings, and conventions.
