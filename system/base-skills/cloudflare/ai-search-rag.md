---
name: cloudflare-ai-search-rag
description: Add managed retrieval / RAG to a Cloudflare Agent using AI Search — ingest documents, index with metadata, query hybrid semantic+keyword, return citations, isolate per-tenant data. Use when the user asks about RAG, retrieval-augmented generation, vector search, hybrid search, knowledge base search, agent memory over a corpus, per-user/workspace/tenant isolation, or giving an agent a "search tool". Produces ingestion + query code, a callable agent tool with a strict schema, tenant-scoped metadata filters, and citation handling.
---

# Cloudflare AI Search + RAG for Agents

Use this skill whenever an agent needs to search over documents, knowledge bases, or memory-like corpora. AI Search is the managed primitive — hybrid semantic + keyword, metadata filters, dynamic instances. Reach for it before building bespoke vector infra.

## When to use

- Retrieval-augmented generation over user/workspace documents.
- Knowledge-base search embedded in an agent.
- A "search" tool the model can invoke with structured arguments.
- Per-tenant isolation across many users or workspaces.
- Answers that must carry citations back to source material.

## When NOT to use

- Answering purely from agent state / SQL — just query the agent's own data directly.
- A handful of static FAQ entries — inline them or read from KV.
- Real browser scraping (JS-heavy sites, login flows) — use Browser Run.

## Four-step pattern

1. **Ingest** documents as rows or chunks, attaching metadata (tenant, source, section, doc id).
2. **Index** via AI Search. The platform handles embeddings, keyword indices, and hybrid scoring.
3. **Query** with strict input schema; pass tenant metadata as a **filter**, not as part of the query text.
4. **Feed relevant excerpts only** into the model; persist and return citations.

## Template — ingest

```ts
import type { Env } from "./worker-env";

export async function ingestDoc(env: Env, args: {
  workspaceId: string;
  docId: string;
  title: string;
  chunks: { section: string; text: string }[];
}) {
  await env.AI_SEARCH.upsert(
    args.chunks.map((chunk, idx) => ({
      id: `${args.docId}:${idx}`,
      text: chunk.text,
      metadata: {
        workspaceId: args.workspaceId,
        docId: args.docId,
        title: args.title,
        section: chunk.section,
      },
    })),
  );
}
```

Notes:
- Use stable, deterministic ids so re-ingestion is an upsert, not a duplicate.
- Attach **every field you might later filter on** in metadata. You cannot filter by data that isn't indexed.

## Template — query from an agent with strict schema

```ts
import { Agent, callable } from "agents";
import { z } from "zod";

const SearchInput = z.object({
  query: z.string().min(1).max(400),
  k: z.number().int().min(1).max(20).default(6),
});

type Hit = {
  id: string;
  text: string;
  score: number;
  metadata: {
    workspaceId: string;
    docId: string;
    title: string;
    section: string;
  };
};

export class SearchAgent extends Agent<Env, { workspaceId: string }> {
  initialState = { workspaceId: "" };

  @callable()
  async search(raw: unknown) {
    const { query, k } = SearchInput.parse(raw);
    return this.queryIndex(query, k);
  }

  async queryIndex(query: string, k: number) {
    const hits = (await this.env.AI_SEARCH.query({
      query,
      topK: k,
      filter: { workspaceId: this.state.workspaceId },
    })) as Hit[];

    return hits.map((h) => ({
      text: h.text,
      score: h.score,
      citation: {
        docId: h.metadata.docId,
        title: h.metadata.title,
        section: h.metadata.section,
      },
    }));
  }
}
```

Key rules:
- **Zod** the input. The query string comes from users/models — length-cap it.
- **Filter by tenant** (`workspaceId`, `orgId`, `userId`) at the query layer. Never rely on the model to self-restrict.
- **Return citations** alongside text so downstream code can render sourced answers.

## Feeding the model (don't dump the corpus)

```ts
async answer(question: string) {
  const hits = await this.queryIndex(question, 6);
  const context = hits
    .map((h, i) => `[${i + 1}] (${h.citation.title} — ${h.citation.section})\n${h.text}`)
    .join("\n\n");

  const completion = await this.modelCall({
    system: "Answer using the context. Cite sources as [n].",
    user: `Question: ${question}\n\nContext:\n${context}`,
  });

  return { answer: completion, sources: hits.map((h) => h.citation) };
}
```

Why this shape:
- The model gets **bounded, top-k excerpts** — never the full corpus.
- Citations are **data**, not an afterthought — UIs and audits need structured sources.
- System prompt names the `[n]` citation format so the model cites consistently.

## Per-tenant isolation (required)

Always set the tenant filter server-side in a variable the client cannot influence. Common shape:

```ts
const filter = {
  workspaceId: this.state.workspaceId,
  ...(accessibleDocIds.length > 0 ? { docId: { $in: accessibleDocIds } } : {}),
};
```

Do not build the filter from the raw client request. Do not concatenate tenant ids into the query string.

## Common mistakes this skill prevents

- Passing the full corpus into the model — blows cost and latency, dilutes the answer.
- Filtering by tenant in post-processing instead of at the index — a slow query + leak risk.
- Missing citations — users and compliance both need them.
- Re-ingesting with random ids — the index grows forever with duplicates.
- Using AI Search for data that already lives in `this.sql` — just query SQL.

## See also

- `.agentspack/docs/cloudflare-agent-stack/11_AI_SEARCH_AND_RAG.md` — deeper reference.
- `cloudflare-agent-state-and-sql` — where to persist per-agent facts vs shared corpus.
- `cloudflare-mcp-tools` — exposing the search as a tool to other agents/models.
- Official: https://developers.cloudflare.com/agents/api-reference/rag/ and https://blog.cloudflare.com/ai-search-agent-primitive/
