---
name: cloudflare-browser-run
description: Drive a real browser from a Cloudflare Agent using Browser Run — for login flows, JS-heavy sites, form filling, screenshots, and human-in-the-loop takeover. Use when the user asks to automate a website, scrape a JS-rendered page, log into a site, fill a form, take a screenshot, hand over to a human, or mentions Browser Run, Puppeteer, Playwright, CDP, WebMCP. Produces a session lifecycle template with navigation, interaction, structured extraction, artifact offloading, and the rules for when to use `fetch` instead.
---

# Cloudflare Agents — Browser Run

Use this skill when an agent needs a **real browser** rather than `fetch`-based scraping. Browser Run is the managed primitive for JS-heavy sites, login-gated flows, form filling, screenshots, and live human takeover.

## When to use

- JavaScript-rendered pages where `fetch` sees an empty shell.
- Login / OAuth / SSO flows that only work interactively.
- Form filling and multi-step wizards.
- Screenshots or PDF-style captures for auditing.
- Human-in-the-loop: agent drives up to a decision, a human takes over, agent resumes.
- CDP-based automation or WebMCP integrations.

## When NOT to use

| Situation                                          | Use instead                         |
| -------------------------------------------------- | ----------------------------------- |
| Plain HTTP JSON API                                | `fetch` / typed client              |
| Static page that renders the data server-side      | `fetch` + HTML parser               |
| High-volume scraping where browser cost is wasteful | `fetch` or a dedicated scraper      |
| Simple deterministic compute                       | Workers / agent methods             |

Rule: reach for a browser only when a browser is actually required. Every session has startup cost.

## Session lifecycle

```ts
import { Agent, callable } from "agents";
import { launch } from "@cloudflare/browser-run";
import { z } from "zod";

const ScrapeInput = z.object({
  url: z.string().url(),
  waitFor: z.string().min(1).max(120).optional(),
});

export class ScraperAgent extends Agent<Env> {
  @callable()
  async extractPrice(raw: unknown) {
    const { url, waitFor = "[data-price]" } = ScrapeInput.parse(raw);

    const session = await launch(this.env.BROWSER);
    try {
      const page = await session.newPage();
      await page.goto(url, { waitUntil: "networkidle" });
      await page.waitForSelector(waitFor, { timeout: 15_000 });

      const price = await page.$eval(waitFor, (el) => el.textContent?.trim() ?? null);
      const screenshotKey = await this.captureAudit(page, url);

      return { price, screenshotKey };
    } finally {
      await session.close();
    }
  }

  private async captureAudit(page: import("@cloudflare/browser-run").Page, url: string) {
    const buffer = await page.screenshot({ type: "png", fullPage: false });
    const key = `audit/${Date.now()}-${encodeURIComponent(url)}.png`;
    await this.env.R2_AUDIT.put(key, buffer);
    return key;
  }
}
```

Key rules:
- **Always `try / finally`** around session use — leaked sessions are expensive.
- **Bound `waitForSelector` timeouts** — page never loads → hung agent.
- **Never put raw page HTML or screenshots in `this.state`** — persist to R2, store only the key.
- **Never let the client pick arbitrary URLs** without an allowlist or user-scoped authorization check. "Fetch this URL for me" is an SSRF vector; a real browser makes it worse.

## Login + human-in-the-loop handoff

```ts
@callable()
async startLogin(raw: unknown) {
  const { site } = z.object({ site: z.string().url() }).parse(raw);

  const session = await launch(this.env.BROWSER);
  const page = await session.newPage();
  await page.goto(site);

  const liveViewUrl = await session.getLiveViewUrl();
  this.setState({
    ...this.state,
    status: "awaiting-human",
    liveViewUrl,
    sessionId: session.id,
  });

  void this.runFiber("await-login", async (ctx) => {
    try {
      await page.waitForSelector("[data-logged-in]", { timeout: 15 * 60_000 });
      const cookies = await session.cookies();
      await this.persistCookies(cookies);
      this.setState({ ...this.state, status: "logged-in", liveViewUrl: undefined });
    } finally {
      await session.close();
    }
  });
}
```

Pattern:
1. Agent opens the site in a live-viewable session.
2. Agent writes `status: "awaiting-human"` + `liveViewUrl` to state so the UI can embed it.
3. Agent waits (inside a **fiber** so it survives eviction) for a "logged-in" signal.
4. Agent persists cookies / tokens to `this.sql`; closes the session.

See `cloudflare-durable-execution-fibers` for why the wait lives in a fiber.

## Structured extraction — prefer `page.evaluate` over regex

```ts
const data = await page.evaluate(() => {
  const rows = Array.from(document.querySelectorAll("table tr"));
  return rows.slice(1).map((r) => {
    const cells = Array.from(r.querySelectorAll("td")).map((c) => c.textContent?.trim() ?? "");
    return { name: cells[0], amount: Number(cells[1]), date: cells[2] };
  });
});
```

Cast to a typed Zod schema after — the page is untrusted input.

## Artifact handling

- **Screenshots / PDFs / recordings** → R2 + key in state or SQL.
- **Page DOM snapshot** → R2 if you need an audit trail, or skip.
- **Cookies / tokens** → `this.sql` with scoped access; never `this.state`.

Rule: the **reference** lives in agent storage, the **blob** lives in R2.

## Common mistakes this skill prevents

- No `try/finally` around the session — cost + leaks.
- Storing screenshots in `this.state` — floods the client and blows the sync budget.
- Accepting any URL from the user/model — pair with an allowlist or user-scoped auth.
- Endless `waitForSelector` with no timeout — agent hangs; the user gives up.
- Putting the human-wait loop directly in a callable — if eviction happens, the wait dies. Use a fiber.
- Using Browser Run for JSON APIs — dramatically more expensive than `fetch`.

## See also

- `.agentspack/docs/cloudflare-agent-stack/12_BROWSER_RUN.md` — deeper reference.
- `cloudflare-durable-execution-fibers` — for long waits (human approval / login completion).
- `cloudflare-agent-security` — URL allowlists, credential handling.
- Official: https://developers.cloudflare.com/agents/api-reference/browse-the-web/ and https://blog.cloudflare.com/browser-run-for-ai-agents/
