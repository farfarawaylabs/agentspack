---
name: cloudflare-workflows-with-agents
description: Orchestrate long, multi-step durable processes with Cloudflare Workflows while keeping the Agent as the stateful, UI-facing surface. Use when the user needs a process that lasts minutes/hours/days/weeks, has explicit steps with per-step retries, needs human-approval gates or sleeps, must continue without a connected client, or mentions Workflows v2, `WorkflowEntrypoint`, `step.do`, `step.sleep`, or `step.waitForEvent`. Produces an Agent-starts-Workflow → Workflow-reports-back-to-Agent pattern with progress updates streaming through `this.state`.
---

# Cloudflare Workflows with Agents

Use this skill whenever durable orchestration belongs outside the agent's immediate activation — long processes, per-step retries, human approvals, or work that must continue when no client is connected. The **Agent stays the UI surface**; the **Workflow stays the durable orchestration engine**.

## When to use

- Process lasts minutes, hours, days, or weeks.
- Explicit multi-step pipeline with distinct boundaries where retries belong.
- Needs `step.sleep(...)` / `step.waitForEvent(...)` / human-approval gate.
- Coordinates multiple services, agents, or external integrations.
- Must run to completion even if no browser/WebSocket is connected.
- You want per-step visibility/auditability in Cloudflare's Workflows UI.

## When NOT to use (use fibers instead)

| Situation                                                | Use                                     |
| -------------------------------------------------------- | --------------------------------------- |
| Work is agent-internal only; no external visibility needed | `this.runFiber(...)` — see fibers skill |
| You mainly need eviction survival for one longish task   | `this.keepAliveWhile(...)` or a fiber   |
| One slow API call that just needs retries                | `this.retry(...)`                       |
| FIFO internal background queue                           | `this.queue(...)`                       |

Rule: **fibers = inside the agent. Workflows = independent durable process.**

## The canonical pattern

1. Agent receives a user/client action (via `@callable` or HTTP).
2. Agent flips `this.state.status = "running"` so the UI sees immediate feedback.
3. Agent starts the Workflow, passing `{ agentId, agentClass, taskId, input }` as params.
4. Workflow runs durable steps.
5. At step boundaries, Workflow calls back into the Agent via `getAgentByName(...)` and invokes a progress method.
6. Agent updates `this.state.progress` / `this.state.currentStep` (auto-synced to UI).
7. Workflow stores final artifacts; calls a final completion method on the Agent.

## Template — Agent side

```ts
import { Agent, callable } from "agents";
import { z } from "zod";

export interface Env {
  REPORT_AGENT: DurableObjectNamespace<ReportAgent>;
  REPORT_WORKFLOW: Workflow<ReportWorkflowParams>;
}

export type ReportWorkflowParams = {
  agentId: string;
  agentClass: "REPORT_AGENT";
  taskId: string;
  topic: string;
};

type State = {
  status: "idle" | "running" | "done" | "error";
  progress: number;
  currentStep?: string;
  workflowId?: string;
  reportUrl?: string;
  lastError?: string;
};

export class ReportAgent extends Agent<Env, State> {
  initialState: State = { status: "idle", progress: 0 };

  @callable()
  async startReport(raw: unknown) {
    const { topic } = z.object({ topic: z.string().min(1) }).parse(raw);

    const taskId = crypto.randomUUID();
    this.setState({
      ...this.state,
      status: "running",
      progress: 0,
      currentStep: "queued",
    });

    const instance = await this.env.REPORT_WORKFLOW.create({
      id: taskId,
      params: {
        agentId: this.name,
        agentClass: "REPORT_AGENT",
        taskId,
        topic,
      } satisfies ReportWorkflowParams,
    });

    this.setState({ ...this.state, workflowId: instance.id });
    return { taskId };
  }

  async reportProgress(step: string, progress: number) {
    this.setState({
      ...this.state,
      status: progress >= 100 ? "done" : "running",
      progress,
      currentStep: step,
    });
  }

  async reportCompletion(reportUrl: string) {
    this.setState({
      ...this.state,
      status: "done",
      progress: 100,
      currentStep: "completed",
      reportUrl,
    });
  }

  async reportFailure(message: string) {
    this.setState({ ...this.state, status: "error", lastError: message });
  }
}
```

## Template — Workflow side

```ts
import { WorkflowEntrypoint, WorkflowStep, WorkflowEvent } from "cloudflare:workers";
import { getAgentByName } from "agents";
import type { Env, ReportAgent, ReportWorkflowParams } from "./agent";

export class ReportWorkflow extends WorkflowEntrypoint<Env, ReportWorkflowParams> {
  async run(event: WorkflowEvent<ReportWorkflowParams>, step: WorkflowStep) {
    const { agentId, topic } = event.payload;
    const agent = await getAgentByName<Env, ReportAgent>(this.env.REPORT_AGENT, agentId);

    const sources = await step.do("gather-sources", { retries: { limit: 3 } }, async () => {
      await agent.reportProgress("gathering sources", 10);
      return this.gatherSources(topic);
    });

    const draft = await step.do("draft-report", { retries: { limit: 3 } }, async () => {
      await agent.reportProgress("drafting", 40);
      return this.draftReport(topic, sources);
    });

    // Optional: wait for a human approval event before publishing.
    const approval = await step.waitForEvent("approval", {
      type: "approve-report",
      timeout: "7 days",
    });

    if (!approval) {
      await agent.reportFailure("approval timed out");
      return;
    }

    const reportUrl = await step.do("publish", { retries: { limit: 5 } }, async () => {
      await agent.reportProgress("publishing", 90);
      return this.publishReport(draft);
    });

    await agent.reportCompletion(reportUrl);
  }

  private async gatherSources(_topic: string) { return []; }
  private async draftReport(_t: string, _s: unknown) { return ""; }
  private async publishReport(_d: string) { return "https://example.com/report/..."; }
}
```

## `wrangler.jsonc` snippet

```jsonc
{
  "workflows": [
    {
      "name": "report-workflow",
      "binding": "REPORT_WORKFLOW",
      "class_name": "ReportWorkflow"
    }
  ],
  "durable_objects": {
    "bindings": [
      { "name": "REPORT_AGENT", "class_name": "ReportAgent" }
    ]
  },
  "migrations": [
    { "tag": "v1", "new_sqlite_classes": ["ReportAgent"] }
  ]
}
```

## Rules of engagement

- **Put retries in `step.do`** options, not ad-hoc inside step bodies. The platform tracks them.
- **Agent writes are small** — state is synced to clients. Store raw artifacts in R2/`this.sql`.
- **Workflow calls into Agent, not the other way around** for progress — the Agent doesn't poll the Workflow.
- **Idempotency**: every `step.do` should be safe to re-execute. If a step writes externally, pair with an idempotency key (see retries skill).
- **Human approval**: use `step.waitForEvent` with a sensible timeout; handle the timeout path explicitly.

## Common mistakes this skill prevents

- Starting a Workflow and then awaiting its completion inside a callable — the caller is long gone before the Workflow finishes. Return `{ taskId }` and let the UI watch agent state.
- Writing progress to SQL and polling from the client — the Agent's `this.state` is already the sync channel; use it.
- Putting business logic in `step.do` that can't be safely re-run — retry will duplicate work. Add idempotency.
- Using Workflows for sub-minute in-agent work that fibers could handle — unnecessary complexity and tooling surface.
- Forgetting the `new_sqlite_classes` migration for the Agent — callbacks from the Workflow will fail when trying to persist.

## See also

- `.agentspack/docs/cloudflare-agent-stack/09_WORKFLOWS_AND_RUN_WORKFLOWS.md` — deeper reference.
- `cloudflare-durable-execution-fibers` — when to prefer fibers over Workflows.
- `cloudflare-agent-state-and-sql` — the progress-through-state pattern.
- `cloudflare-retries-with-idempotency` — safe `step.do` retries.
- Official: https://developers.cloudflare.com/agents/api-reference/run-workflows/ and https://blog.cloudflare.com/workflows-v2/
