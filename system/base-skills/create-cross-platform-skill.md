---
name: create-cross-platform-skill
description: Create, add, update, revise, refactor, migrate, or remove agent skills from one canonical source and ensure they are generated for Cursor, Claude Code, and Codex. Use for any skill-related user request even if the user mentions only one platform or does not mention platforms at all.
---

# Create Cross-Platform Skill

## Goal

Keep skill behavior aligned across provider folders so users get the same capability in Cursor, Claude Code, and Codex without needing to ask per-platform.

## Default Invocation Policy

- Treat every skill-related request as cross-platform by default.
- Do not require the user to mention all platforms.
- If the user mentions one platform, still create/update matching skills for all relevant provider folders.
- Only scope to a single platform when the user explicitly requests that limitation.

## Required Workflow

1. Clarify the skill's purpose, trigger scenarios, expected output format, and whether the user explicitly wants a single-platform exception.
2. Choose the skill name and prepare a portable base `SKILL.md` content using frontmatter:
   - `name`
   - `description`
3. Keep the body platform-neutral by default:
   - Use clear, imperative steps.
   - Prefer portable instructions before platform-specific details.
4. Create or update provider skill files for every relevant platform path:
   - `.cursor/skills/<skill-name>/SKILL.md`
   - `.claude/skills/<skill-name>/SKILL.md`
   - `.agents/skills/<skill-name>/SKILL.md`
5. Add platform-specific behavior only when necessary:
   - Invocation controls (`disable-model-invocation`, `user-invocable`) should be minimal and intentional.
   - Avoid platform-only fields unless there is a clear need.
6. Verify quality before finishing:
   - Check frontmatter validity and non-empty `description`.
   - Confirm naming consistency between folder and `name` in each provider.
   - Confirm all provider copies are semantically aligned.

## Rules

- Never maintain duplicate source-of-truth skill content in provider output folders.
- Assume a user-requested skill should exist on every supported platform unless they explicitly opt out.
- In generated projects, write skill updates directly into provider folders (`.cursor/skills`, `.claude/skills`, `.agents/skills`).
- Prefer additive edits to existing skills instead of creating near-duplicate variants.

## Completion Checklist

- [ ] Skill is created/updated in each relevant provider skills folder
- [ ] Skill name and description are explicit and trigger-friendly
- [ ] Provider copies are aligned in behavior (except intentional platform-specific metadata)
