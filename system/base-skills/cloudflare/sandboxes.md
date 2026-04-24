---
name: cloudflare-sandboxes
description: Use Cloudflare Sandboxes to give a Cloudflare Agent a persistent, isolated computer-like environment — filesystem, shell/terminal, code execution, preview URLs, snapshots. Use when the user is building a coding agent, needs to run user/generated code safely, preview a generated app, execute tests, drive a shell, or mentions Sandbox SDK, sandbox terminal/filesystem/previews, or scoped credentials for isolated execution. Produces a sandbox lifecycle template, command/file/preview primitives, credential-scoping rules, and log-redaction guidance.
---

# Cloudflare Agents — Sandboxes

Use this skill when an agent needs a **persistent isolated environment** to run code, drive a shell, manipulate files, or host generated app previews. Sandboxes are the right primitive for coding agents, generated-app demos, test runs, and anything requiring a real computer-like workspace.

## When to use

- Coding agents that write + execute code.
- Running user- or AI-generated code safely.
- Preview URLs for generated apps / mini-projects.
- Running test suites or CI-like steps inside an agent's flow.
- Shell / terminal access the agent drives.
- Credentials injected into the sandbox scoped to one task.

## When NOT to use

| Situation                                   | Use instead                   |
| ------------------------------------------- | ----------------------------- |
| Simple deterministic compute                | Plain Worker / agent method   |
| HTTP/API routing                            | Worker route                  |
| Single-shot prompt → text compute           | Model call                    |
| One-off deterministic transform (CSV, JSON) | Agent method + `this.sql`     |

Rule: sandboxes cost more than a Worker call. Reach for them when you need state (a filesystem) and interactivity (a shell / multi-step code run), not for pure functions.

## Session lifecycle

```ts
import { Agent, callable } from "agents";
import { openSandbox } from "@cloudflare/sandbox";
import { z } from "zod";

const RunTestsInput = z.object({
  files: z.record(z.string(), z.string()),
  command: z.string().min(1).max(200),
});

export class CodingAgent extends Agent<Env, { sandboxId?: string; lastExit?: number }> {
  initialState = {};

  @callable()
  async runTests(raw: unknown) {
    const { files, command } = RunTestsInput.parse(raw);

    const sandbox = await openSandbox(this.env.SANDBOX, {
      id: this.name,
      env: {
        NODE_ENV: "test",
        CI: "1",
      },
    });

    try {
      for (const [path, contents] of Object.entries(files)) {
        await sandbox.writeFile(path, contents);
      }

      const result = await sandbox.exec(command, {
        timeoutMs: 120_000,
        captureOutput: true,
      });

      this.setState({
        ...this.state,
        sandboxId: sandbox.id,
        lastExit: result.exitCode,
      });

      return {
        exitCode: result.exitCode,
        stdoutKey: await this.offload(result.stdout, "stdout"),
        stderrKey: await this.offload(result.stderr, "stderr"),
      };
    } finally {
      if (!this.shouldKeepAlive()) {
        await sandbox.close();
      }
    }
  }

  private async offload(content: string, kind: string) {
    if (content.length < 8_000) return null;
    const key = `sandbox/${this.name}/${Date.now()}-${kind}.log`;
    await this.env.R2_LOGS.put(key, content);
    return key;
  }

  private shouldKeepAlive() {
    return false;
  }
}
```

## Primitives you'll actually use

- **`sandbox.writeFile(path, contents)`** / **`readFile(path)`** — materialize and inspect the workspace.
- **`sandbox.exec(command, opts)`** — run shell commands with a timeout. Always set one.
- **`sandbox.openTerminal()`** — interactive terminal the agent drives turn-by-turn.
- **`sandbox.getPreviewUrl(port)`** — public preview for a running dev server, good for generated apps.
- **`sandbox.snapshot()`** / **`restore(snapshotId)`** — checkpoint and fork the workspace (useful for branching attempts).
- **`sandbox.close()`** — release resources when done.

## Credential scoping (critical)

Never hand a sandbox your global secrets. Pattern:

```ts
const tempToken = await mintScopedToken({
  userId: this.state.userId,
  scope: ["repo:read", "deploy:staging"],
  ttlSeconds: 15 * 60,
});

const sandbox = await openSandbox(this.env.SANDBOX, {
  id: this.name,
  env: {
    REPO_TOKEN: tempToken,
  },
});
```

Rules:
- Use **short-TTL, scoped** credentials. Mint at the start of the task; don't persist in the sandbox.
- Do not pass platform-level secrets (Cloudflare API tokens, org-wide keys) into a sandbox the user/model controls.
- Redact secrets from stdout/stderr before offloading to R2 — run a mask pass if the command might echo env vars.

## Generated-app previews

```ts
await sandbox.writeFile("package.json", pkgJson);
await sandbox.writeFile("src/index.ts", generatedCode);
await sandbox.exec("npm install", { timeoutMs: 60_000 });

const { pid } = await sandbox.execDetached("npm run dev", {
  env: { PORT: "3000" },
});

const previewUrl = await sandbox.getPreviewUrl(3000);
this.setState({ ...this.state, previewUrl, devPid: pid });
```

Preview URLs are public by default — wrap with your own auth (signed URL, workspace-scoped routing) if the app shouldn't be world-readable.

## When to keep the sandbox alive vs close

- **Close** after each task if the user workflow is one-shot (e.g. run tests and report).
- **Keep alive** when the user is actively iterating (coding agent with a live chat) — store `sandboxId` in state so the next `@callable` can reopen the same workspace.

## Common mistakes this skill prevents

- Passing long-lived Cloudflare or org credentials into a sandbox — the model can exfiltrate them.
- No `timeoutMs` on `exec` — runaway commands tie up the sandbox forever.
- Storing huge stdout/stderr in `this.state` — floods clients. Offload to R2, store the key.
- Forgetting `sandbox.close()` — silent cost leakage.
- Using a sandbox to do 5 lines of deterministic computation — Workers does that at 1/100th the cost.
- Treating generated code as trusted — it isn't. Always scope env + network.

## See also

- `.agentspack/docs/cloudflare-agent-stack/13_SANDBOXES.md` — deeper reference.
- `cloudflare-durable-objects-facets-dynamic-workers` — when generated code should be a deployed worker rather than sandbox-run.
- `cloudflare-agent-security` — credential scoping and log redaction.
- Official: https://blog.cloudflare.com/sandbox-ga/
