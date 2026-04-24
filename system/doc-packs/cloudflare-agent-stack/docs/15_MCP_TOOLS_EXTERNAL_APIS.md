# MCP, Tools, and External APIs

## Purpose

Use MCP and explicit tool adapters to let agents interact with external systems while preserving authorization, validation, observability, and least privilege.

## Use MCP for

- exposing tools to agent clients
- connecting to external MCP servers
- building remote MCP servers on Cloudflare
- standardizing tool discovery and invocation

## Tool design rules

- Define narrow tool schemas.
- Validate inputs.
- Return structured outputs.
- Include source/citation references where possible.
- Log tool call metadata.
- Keep secrets server-side.
- Prefer user-scoped tokens over global tokens.

## Do not

- expose raw arbitrary HTTP fetch as a universal tool
- let the model choose arbitrary URLs for privileged requests
- return huge payloads directly into the model
- mix tenant credentials

## Source docs

- https://developers.cloudflare.com/agents/api-reference/mcp-handler-api/
- https://developers.cloudflare.com/agents/api-reference/mcp-agent-api/
- https://developers.cloudflare.com/agents/api-reference/mcp-client-api/
