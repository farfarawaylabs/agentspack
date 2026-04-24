# Workflows and Run Workflows

## Purpose

Use Workflows when you need durable multi-step execution outside the immediate agent activation.

## Use Workflows for

- processes that last minutes, hours, days, or weeks
- orchestration with explicit steps
- human approval gates
- sleeps/waits
- retries at step boundaries
- long-running integrations
- reliable progress tracking
- work that should continue even if no client is connected

## Use Agent fibers instead when

- the work is tightly part of this agent instance
- you mainly need eviction survival
- the work is not an externally visible process
- you can checkpoint manually

## Pattern: Agent starts Workflow, Workflow reports back

1. Agent receives user action.
2. Agent writes visible state: `status: "running"`.
3. Agent starts Workflow with task ID and agent identity.
4. Workflow runs durable steps.
5. Workflow calls back into Agent to update progress.
6. Agent broadcasts progress through state sync.
7. Workflow stores final artifacts and signals completion.

## Rule

Do not put a whole weeks-long business process in a single Agent method or fiber if it has natural durable steps. Use Workflows.

## Source docs

- https://developers.cloudflare.com/agents/api-reference/run-workflows/
- https://developers.cloudflare.com/agents/api-reference/durable-execution/
- https://blog.cloudflare.com/workflows-v2/
