---
name: cloudflare-mcp-tools
description: Build and consume MCP (Model Context Protocol) tools on Cloudflare Agents — exposing remote MCP servers, designing narrow tool schemas, validating inputs, keeping secrets server-side, and connecting agents to external MCP servers. Use when the user asks to build an MCP server on Cloudflare, add an MCP tool to an agent, expose agent capabilities as tools, connect to a remote MCP server, design a tool schema, or mentions MCP handler/agent/client APIs. Produces a remote MCP server scaffold, a tool with Zod-validated inputs and structured outputs, and allowlist/least-privilege patterns.
---

# Cloudflare Agents — MCP Tools and External APIs

Use this skill when exposing capabilities to LLM agents (yours or external) via MCP, or when an agent consumes an external MCP server. MCP is the right interface for tool discovery and invocation; explicit adapters are the right interface for everything else. In both cases: **narrow schemas, validated inputs, structured outputs, least privilege**.

## When to use

- Building a remote MCP server on Cloudflare that exposes tools to external agents.
- Adding a tool to an agent that LLMs will choose and invoke.
- Connecting an agent to an external MCP server.
- Standardizing tool discovery/invocation across multiple model backends.

## When NOT to use

| Situation                                                   | Use instead                     |
| ----------------------------------------------------------- | ------------------------------- |
| One-off HTTP call inside an agent method                    | Plain `fetch` + `this.retry`    |
| UI command triggered by a browser                           | `@callable()` (not MCP)         |
| Agent-to-agent call you control                             | Durable Object RPC              |
| Pure data retrieval for RAG                                 | `cloudflare-ai-search-rag`      |

Rule: MCP is a **public tool contract**. Use it when third-party models/agents will call in. For internal wiring, use typed RPC.

## Core tool design rules

- **Narrow schemas** — one capability per tool. No `runTool(name, args)` meta-tools.
- **Validate inputs with Zod** — the model can and will send malformed args.
- **Structured outputs** — objects with named fields, not blobs of text.
- **Citations / source refs** when the tool returns retrieved content.
- **Log call metadata** — tool name, args hash, latency, caller id. You'll want this the first time something goes wrong.
- **Secrets stay server-side** — never include credentials in the tool description or pass them through args.
- **User-scoped tokens** over global tokens — every tool call carries the caller's authority, not the server's.

## Template — remote MCP server on Cloudflare

```ts
import { McpServer } from "@cloudflare/mcp";
import { z } from "zod";

export interface Env {
  DB: D1Database;
  SEARCH: AISearchBinding;
  ORDERS_API_BASE: string;
}

const SearchOrdersInput = z.object({
  workspaceId: z.string().uuid(),
  query: z.string().min(1).max(200),
  limit: z.number().int().min(1).max(50).default(10),
});

const GetOrderInput = z.object({
  workspaceId: z.string().uuid(),
  orderId: z.string().uuid(),
});

const server = new McpServer({
  name: "orders-mcp",
  version: "1.0.0",
});

server.tool({
  name: "search_orders",
  description: "Search orders within a workspace by keyword.",
  input: SearchOrdersInput,
  handler: async (input, ctx) => {
    await assertAccess(ctx, input.workspaceId);
    const hits = await ctx.env.SEARCH.query({
      query: input.query,
      topK: input.limit,
      filter: { workspaceId: input.workspaceId, type: "order" },
    });
    return {
      orders: hits.map((h) => ({
        id: h.id,
        title: h.metadata.title,
        url: h.metadata.url,
        score: h.score,
      })),
    };
  },
});

server.tool({
  name: "get_order",
  description: "Fetch a single order's details by id.",
  input: GetOrderInput,
  handler: async (input, ctx) => {
    await assertAccess(ctx, input.workspaceId);
    const row = await ctx.env.DB.prepare(
      "SELECT id, status, total_cents, currency FROM orders WHERE id = ?1 AND workspace_id = ?2",
    )
      .bind(input.orderId, input.workspaceId)
      .first();
    if (!row) return { error: "not_found" as const };
    return { order: row };
  },
});

async function assertAccess(ctx: unknown, _workspaceId: string) {
  // Inspect the MCP connection auth (OAuth/JWT/etc.). Throw on failure.
}

export default server;
```

Notes:
- **One narrow tool per capability.** Do not add a `run_query` tool that takes SQL.
- **Input schemas are part of the API** — breaking changes = a new tool name.
- **Return structured data**, never a pre-formatted natural-language string. Let the caller's model write prose.
- **`assertAccess` on every handler** — the MCP connection auth determines what the caller can see.

## Template — consuming an external MCP server from an agent

```ts
import { Agent } from "agents";
import { McpClient } from "@cloudflare/mcp/client";

export class ResearchAgent extends Agent<Env> {
  private mcp?: McpClient;

  async onStart() {
    this.mcp = await McpClient.connect({
      url: this.env.EXTERNAL_MCP_URL,
      auth: { token: await this.mintScopedToken() },
    });
  }

  async gatherContext(topic: string) {
    const toolCall = await this.mcp!.call("search_orders", {
      workspaceId: this.state.workspaceId,
      query: topic,
      limit: 8,
    });
    return toolCall.result;
  }

  private async mintScopedToken() {
    // Return a per-user / per-workspace scoped token — never a global secret.
    return "…";
  }
}
```

Rules:
- Mint **scoped tokens** per agent / per user. Global tokens are an exfiltration risk the moment a prompt-injection lands.
- Cap tool outputs fed to the model — structure, then excerpt; don't dump.
- Don't expose arbitrary-URL `fetch` as an MCP tool. "Fetch this URL for me" is SSRF-as-a-service.

## Allowlists and quotas

- **Allowlist tools** the agent is permitted to call, not the full MCP catalog the server advertises.
- **Rate-limit** expensive tools per agent — store counters in `this.sql`.
- **Timeouts** on every tool call — a hung tool shouldn't freeze the agent activation.
- **Redact secrets** in any log line that includes tool args.

## Common mistakes this skill prevents

- Writing an `exec_sql(sql)` or `fetch_url(url)` tool — the model will eventually send something you didn't want.
- Returning raw provider JSON directly — leaks fields, inflates model context. Shape it.
- Passing a platform API token into a tool's args — use server-side bindings, never arg-level secrets.
- No per-tenant filter in tool handlers — the model has no concept of "the right workspace".
- Forgetting timeouts / rate limits — one bad tool call ties up the agent forever.
- Treating MCP as internal RPC — use DO RPC for that.

## See also

- `.agentspack/docs/cloudflare-agent-stack/15_MCP_TOOLS_EXTERNAL_APIS.md` — deeper reference.
- `.agentspack/docs/cloudflare-agent-stack/16_SECURITY_AUTH_PERMISSIONS.md` — auth patterns that apply to tool calls.
- `cloudflare-agent-security` — tool allowlists, scoped tokens, redaction.
- `cloudflare-ai-search-rag` — common tool back-end for retrieval.
- Official: https://developers.cloudflare.com/agents/api-reference/mcp-handler-api/ and https://developers.cloudflare.com/agents/api-reference/mcp-client-api/
