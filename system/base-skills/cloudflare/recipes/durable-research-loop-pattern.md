---
name: cloudflare-durable-research-loop-pattern
description: Blueprint for a long, checkpointed research/analysis loop that runs inside one agent and resumes from the last checkpoint if evicted. Use when the user asks for "a research agent", "deep research flow", "multi-step analysis that can take 10+ minutes", "an agent that searches, crawls, and summarizes", "a checkpointable background task", or any recipe that combines search + fetch + analyze + synthesize as an internal agent loop. Also covers when to reach for Workflows instead. Produces a ResearchAgent with fiber-based execution, stash points between phases, external artifact storage, and progress visible via state.
---

# Recipe: Durable research loop

Use this recipe when a single agent needs to run a **long, multi-phase** process internally and must resume from the last completed phase if the Durable Object is evicted or the Worker is redeployed. Classic shape: search → crawl → analyze → synthesize.

## When to use

- The whole process conceptually belongs to **one** agent (per user, per query, per project).
- You have clear, stashable checkpoints between phases (results of each phase are small-ish JSON).
- Total duration can exceed a single request window (minutes to tens of minutes).
- You don't need external orchestration, dashboards, or per-step retry policies visible outside the agent.

## When NOT to use

- Many independent steps with retries and external visibility → `cloudflare-workflows-with-agents`.
- Needs multiple agents to collaborate → orchestrate with a supervisor agent + agent-to-agent RPC.
- External-party wait (email reply, approval) → `cloudflare-long-running-assistant-pattern`.
- Short, in-one-request work → just a plain callable or queued task.

## Primitives used

| Primitive                               | Role                                                    |
| --------------------------------------- | ------------------------------------------------------- |
| `cloudflare-durable-execution-fibers`   | Fiber that runs the loop, stashing after each phase.    |
| `cloudflare-agent-state-and-sql`        | `state` = progress summary; `sql` = per-phase results.  |
| `cloudflare-retries-with-idempotency`   | Idempotent external calls (search API, fetches).        |
| `cloudflare-callable-methods`           | `start()`, `status()`, `cancel()` UI hooks.             |
| R2 / Workers KV / D1 (external)         | Store large artifacts (raw HTML, embeddings) outside fiber snapshots. |

## Phase structure

```
search       → stash searchResults
crawl        → stash crawledPages (IDs only, bodies in R2)
analyze      → stash analysisSummaries
synthesize   → set state.status = "done"
```

Checkpoint rule: stash only **small structured data**. Raw HTML, large embeddings, binary blobs → write to R2 / D1 and stash the ID/URL.

## Blueprint

```ts
// src/research-agent.ts
import { Agent, callable } from "agents";
import { z } from "zod";

export interface Env {
  RESEARCH_AGENT: DurableObjectNamespace<ResearchAgent>;
  ARTIFACTS: R2Bucket;
  SEARCH: { query(q: string): Promise<Array<{ url: string; title: string }>> };
  FETCH: { get(url: string): Promise<string> };
}

type Phase = "idle" | "searching" | "crawling" | "analyzing" | "synthesizing" | "done" | "failed";

type State = {
  query: string;
  phase: Phase;
  progress: { completed: number; total: number };
  error?: string;
  resultId?: string;
};

const StartInput = z.object({ query: z.string().min(3).max(500) });

export class ResearchAgent extends Agent<Env, State> {
  initialState: State = {
    query: "",
    phase: "idle",
    progress: { completed: 0, total: 0 },
  };

  async onStart() {
    this.sql`
      CREATE TABLE IF NOT EXISTS search_results (
        idx INTEGER PRIMARY KEY,
        url TEXT NOT NULL,
        title TEXT NOT NULL
      );
      CREATE TABLE IF NOT EXISTS analyses (
        idx INTEGER PRIMARY KEY,
        url TEXT NOT NULL,
        summary TEXT NOT NULL
      );
    `;
  }

  @callable()
  async start(input: unknown): Promise<void> {
    const { query } = StartInput.parse(input);
    if (this.state.phase !== "idle" && this.state.phase !== "done" && this.state.phase !== "failed") {
      throw new Error(`research already in progress (phase=${this.state.phase})`);
    }
    this.setState({ ...this.initialState, query, phase: "searching" });
    await this.runFiber("research");
  }

  @callable()
  status(): State {
    return this.state;
  }

  async research(): Promise<void> {
    try {
      // 1. SEARCH
      const results = await this.retry(() => this.env.SEARCH.query(this.state.query), { key: `search-${this.name}` });
      this.sql`DELETE FROM search_results`;
      for (const [i, r] of results.entries()) {
        this.sql`INSERT INTO search_results (idx, url, title) VALUES (${i}, ${r.url}, ${r.title})`;
      }
      await this.stash("search", { count: results.length });
      this.setState({ ...this.state, phase: "crawling", progress: { completed: 0, total: results.length } });

      // 2. CRAWL (resumable: only fetch URLs we haven't analyzed yet)
      for (const [i, r] of results.entries()) {
        const already = this.sql`SELECT 1 FROM analyses WHERE idx = ${i}`;
        if (already.length > 0) {
          this.setState({ ...this.state, progress: { completed: i + 1, total: results.length } });
          continue;
        }

        const body = await this.retry(() => this.env.FETCH.get(r.url), { key: `fetch-${this.name}-${r.url}` });
        const artifactId = `${this.name}/${i}.html`;
        await this.env.ARTIFACTS.put(artifactId, body);

        const summary = await this.retry(() => summarize(r.url, body), { key: `summary-${this.name}-${r.url}` });
        this.sql`INSERT INTO analyses (idx, url, summary) VALUES (${i}, ${r.url}, ${summary})`;

        await this.stash(`crawl-${i}`, { artifactId });
        this.setState({ ...this.state, progress: { completed: i + 1, total: results.length } });
      }

      this.setState({ ...this.state, phase: "synthesizing" });

      // 3. SYNTHESIZE
      const analyses = this.sql`SELECT url, summary FROM analyses ORDER BY idx ASC`;
      const final = await this.retry(() => synthesize(this.state.query, analyses), { key: `synth-${this.name}` });
      const resultId = `${this.name}/final.md`;
      await this.env.ARTIFACTS.put(resultId, final);

      this.setState({ ...this.state, phase: "done", resultId });
    } catch (err) {
      this.setState({
        ...this.state,
        phase: "failed",
        error: err instanceof Error ? err.message : "unknown",
      });
    }
  }
}

async function summarize(_url: string, _body: string): Promise<string> {
  return "…"; // Replace with Workers AI / Anthropic.
}

async function synthesize(_query: string, _analyses: Array<{ url: string; summary: string }>): Promise<string> {
  return "…"; // Replace with Workers AI / Anthropic.
}
```

## Why this shape survives eviction

- The agent is a Durable Object, so `this.name` and `this.sql` persist.
- Each phase writes its results to SQL or R2 **before** advancing the FSM, so a cold restart reads the same state.
- The crawl loop checks `this.sql` for already-analyzed URLs, so it resumes at the first gap instead of restarting from zero.
- Fiber `stash()` calls checkpoint the fiber's own locals — but the authoritative progress store is SQL.

## When to upgrade to Workflows

Upgrade to `cloudflare-workflows-with-agents` when you need:

- Per-step retry policies + dead-letter behavior visible outside the agent.
- A dashboard/observability view of which step each run is on.
- Fanned-out parallelism across many independent items (Workflows' step parallelism is more ergonomic than orchestrating fibers yourself).
- A process that spans **multiple** agents (Workflows are the natural orchestrator).

Keep it as a fiber if the loop is agent-local and single-purpose.

## Common mistakes this recipe prevents

- Stashing the raw fetched HTML — blows up the fiber snapshot.
- Forgetting to write partial results to SQL between phases — eviction forces a full restart.
- Restarting from phase 1 on resume instead of scanning for the first gap.
- Treating the search API or fetch as safe to call without idempotency keys — duplicate work on retry.
- Using Workflows here out of habit — for one agent's internal loop, fibers are cheaper and simpler.

## See also

- `cloudflare-durable-execution-fibers` — fiber, stash, runFiber semantics.
- `cloudflare-agent-state-and-sql` — what belongs in each.
- `cloudflare-workflows-with-agents` — the "upgrade" option.
- `cloudflare-retries-with-idempotency` — mandatory for any external call in this loop.
- `.agentspack/docs/cloudflare-agent-stack/17_PATTERNS_AND_RECIPES.md` — prose version.
