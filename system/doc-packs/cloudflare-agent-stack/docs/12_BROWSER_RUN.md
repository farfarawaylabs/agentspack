# Browser Run

## Purpose

Use Browser Run when an agent needs a real browser instead of fetch-based scraping.

## Use Browser Run for

- JS-heavy websites
- login flows
- form filling
- screenshots
- page interactions
- live view / human-in-the-loop takeover
- CDP-based automation
- WebMCP/browser-agent integrations
- recordings/debugging

## Do not use Browser Run for

- simple HTTP APIs
- static pages that can be fetched directly
- high-volume scraping where browser cost is unnecessary

## Pattern

1. Agent decides browser is necessary.
2. Browser Run session starts.
3. Agent navigates/interacts.
4. Human can take over if needed.
5. Agent extracts structured result.
6. Store artifacts/screenshots/logs as references, not giant state blobs.

## Source docs

- https://blog.cloudflare.com/browser-run-for-ai-agents/
- https://developers.cloudflare.com/agents/api-reference/browse-the-web/
