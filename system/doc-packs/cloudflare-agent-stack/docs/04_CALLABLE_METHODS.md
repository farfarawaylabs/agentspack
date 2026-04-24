# Callable Methods

## What they are

Callable methods expose selected Agent methods to external clients over WebSocket RPC. Mark a method with `@callable()` when a browser, mobile app, or external client should invoke it through the Agents client connection.

## Use callable methods for

- UI actions
- user-triggered commands
- client-to-agent RPC
- browser/mobile calls into a live agent
- streaming responses to the client

## Do not use callable methods for

- Worker-to-Agent calls in the same codebase
- Agent-to-Agent calls
- internal Durable Object RPC
- private methods
- sensitive admin operations unless explicitly authorized

For Worker-to-Agent and Agent-to-Agent calls, use Durable Object RPC / agent stubs directly.

## Basic pattern

```ts
import { Agent, callable } from "agents";

export class CounterAgent extends Agent<Env, { count: number }> {
  initialState = { count: 0 };

  @callable()
  increment(): number {
    this.setState({ ...this.state, count: this.state.count + 1 });
    return this.state.count;
  }

  @callable()
  async addItem(item: string): Promise<{ ok: true }> {
    await this.sql`INSERT INTO items (value) VALUES (${item})`;
    return { ok: true };
  }
}
```

Client-side:

```ts
const count = await agent.stub.increment();
const result = await agent.stub.addItem("new item");
```

## Streaming callable methods

Use streaming methods when the caller needs incremental output, such as AI generation, progress, or partial search results.

```ts
import { Agent, callable, type StreamingResponse } from "agents";

export class AIAgent extends Agent {
  @callable({ streaming: true })
  async generateText(stream: StreamingResponse, prompt: string) {
    for await (const chunk of this.generate(prompt)) {
      stream.send(chunk);
    }
    stream.end();
  }
}
```

## Serialization rules

Arguments and return values must be JSON-serializable. Avoid returning `Date`, `Map`, `Set`, functions, class instances, streams, request objects, or raw errors.

## Security rules

- Treat callable methods as public API methods available to connected clients.
- Validate payloads with Zod or equivalent.
- Check authorization inside the method or at connection setup.
- Do not expose broad methods like `runTool(toolName, args)` without strict allowlists.

## Sources

- https://developers.cloudflare.com/agents/api-reference/callable-methods/
