# AI Search and RAG

## Purpose

Use AI Search as the managed retrieval primitive for agents that need to search knowledge bases, documents, indexed content, or memory-like corpora.

## Use AI Search for

- document retrieval
- knowledge base search
- hybrid semantic + keyword search
- RAG over user/workspace data
- search-backed tools for agents
- metadata-filtered retrieval
- dynamic search instances

## Pattern

1. Ingest documents or records.
2. Index content with metadata.
3. Give agent a search tool with strict input schema.
4. Retrieve top results.
5. Feed only relevant excerpts to the model.
6. Persist citations/source references.

## Do not

- dump whole corpora into the prompt
- build bespoke search infra before checking AI Search
- omit source references for generated answers
- mix tenants without explicit metadata isolation

## Source docs

- https://blog.cloudflare.com/ai-search-agent-primitive/
- https://developers.cloudflare.com/agents/api-reference/rag/
