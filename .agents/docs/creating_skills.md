# Creating Skills Across Cursor, Claude Code, and Codex

This guide compares how **Cursor**, **Claude Code**, and **Codex** implement skills, with the goal of helping you write skills that are both portable and platform-optimized.

## TL;DR

- All three use the **Agent Skills** model: a folder containing a `SKILL.md`.
- `SKILL.md` uses **YAML frontmatter + markdown instructions**.
- The most portable baseline is:
  - `name`
  - `description`
  - clear, imperative instructions
  - optional `scripts/`, `references/`, `assets/`
- Biggest differences are in:
  - where skills are discovered
  - invocation control fields
  - advanced extensions (subagents, dynamic context, tool policy metadata)

## Core Structure Shared by All Three

At minimum, a skill is:

```text
<skill-name>/
└── SKILL.md
```

Common extended structure:

```text
<skill-name>/
├── SKILL.md
├── scripts/
├── references/
└── assets/
```

Common `SKILL.md` pattern:

```markdown
---
name: my-skill
description: Explain what this skill does and when it should be used.
---

# My Skill

## When to Use
- ...

## Instructions
1. ...
2. ...
```

## Platform-by-Platform Format

### Cursor

**Discovery paths**
- `.agents/skills/` (project)
- `.cursor/skills/` (project)
- `~/.cursor/skills/` (user)
- Compatibility paths also supported: `.claude/skills/`, `.codex/skills/`, `~/.claude/skills/`, `~/.codex/skills/`

**Frontmatter (documented in Cursor docs)**
- `name` (**required**, and must match the parent folder name)
- `description` (**required**)
- `license` (optional)
- `compatibility` (optional)
- `metadata` (optional)
- `disable-model-invocation` (optional)

**Invocation behavior**
- Implicit invocation based on `description` is supported.
- Set `disable-model-invocation: true` to make it explicit-only (slash invocation).

### Claude Code

**Discovery paths**
- Enterprise: managed settings (organization-wide)
- `~/.claude/skills/<skill>/SKILL.md` (personal)
- `.claude/skills/<skill>/SKILL.md` (project)
- Plugin skill paths and nested monorepo discovery are supported.
- Existing `.claude/commands/` still work; skills supersede same-name commands.

**Frontmatter (Claude extension of Agent Skills)**
- `name` (optional; defaults to folder name)
- `description` (recommended; used for matching)
- `argument-hint` (optional)
- `disable-model-invocation` (optional)
- `user-invocable` (optional)
- `allowed-tools` (optional)
- `model` (optional)
- `context` (optional, e.g. `fork`)
- `agent` (optional, used with `context: fork`)
- `hooks` (optional)

**Special capabilities**
- Argument substitutions: `$ARGUMENTS`, `$ARGUMENTS[N]`, `$N`, `${CLAUDE_SESSION_ID}`
- If arguments are passed but `$ARGUMENTS` is omitted, Claude appends `ARGUMENTS: <user input>` to the end of the skill content.
- Can run skills in forked subagent contexts (`context: fork`)
- Dynamic context injection via shell command substitution (prefix with `!` and wrap the command in backticks, for example ``!`gh pr diff` ``).

### Codex

**Discovery paths**
- Repo: `.agents/skills` (scanned in every directory from CWD up to repo root)
- User: `$HOME/.agents/skills`
- Admin: `/etc/codex/skills`
- System: built-in skills

**Frontmatter**
- `name` (**required**)
- `description` (**required**)

**Codex-specific metadata**
- Optional `agents/openai.yaml` supports:
  - `interface` metadata (display name, icons, color, default prompt)
  - `policy.allow_implicit_invocation`
  - `dependencies.tools` (including MCP dependencies)

**Management / operations**
- Built-in creators/installers (`$skill-creator`, `$skill-installer`)
- Skill enable/disable via `~/.codex/config.toml` (Unix) or `%USERPROFILE%\.codex\config.toml` (Windows) using `[[skills.config]]`

## Similarities

- **Shared conceptual model**: skills are reusable instruction packages.
- **Progressive loading**: metadata first, full content when invoked/needed.
- **Description-driven matching**: `description` quality strongly influences implicit invocation.
- **Script support**: all can orchestrate scripts through instructions (with platform permission/tooling behavior).
- **Portable baseline**: a simple Agent Skills-compliant `SKILL.md` works across platforms with little/no change.

## Key Differences

| Area | Cursor | Claude Code | Codex |
| --- | --- | --- | --- |
| Required frontmatter | `name`, `description` | none strictly required; `description` recommended | `name`, `description` |
| Invocation control | `disable-model-invocation` | `disable-model-invocation`, `user-invocable`, permissions interplay | `allow_implicit_invocation` via `agents/openai.yaml` |
| Skill locations | `.agents/skills`, `.cursor/skills`, plus compat dirs | `.claude/skills` (+ nested/add-dir/plugin behavior) | `.agents/skills` across repo hierarchy + user/admin/system |
| Advanced execution | Standard skill behavior | strong subagent/fork model and hooks | optional metadata and dependency declarations in `agents/openai.yaml` |
| Argument templating | not emphasized in doc page | rich built-ins (`$ARGUMENTS`, `$N`, etc.) | not documented as a first-class SKILL.md feature in the referenced page |

## Practical Authoring Strategy (Write Once, Adapt Lightly)

1. Start with a **portable core**:
   - required-safe fields: `name`, `description`
   - explicit instructions with clear input/output expectations
2. Keep `SKILL.md` concise and move heavy docs into `references/`.
3. Add platform extensions only when needed:
   - Claude: `allowed-tools`, `context: fork`, `agent`, placeholders
   - Codex: `agents/openai.yaml` for policy/UI/dependencies
   - Cursor: leverage supported directories and explicit invocation switch
4. Test both invocation paths:
   - implicit (natural prompt matching)
   - explicit (Cursor/Claude: `/skill-name`; Codex: `$skill-name` and `/skills`)

## agentspack Source-of-Truth Pattern

In this repository, cross-platform baseline skills should be authored in:

- `system/base-skills/<skill-name>.md`

Then generated into provider-native outputs:

- Cursor: `.cursor/skills/<skill-name>/SKILL.md`
- Claude Code: `.claude/skills/<skill-name>/SKILL.md`
- Codex: `.agents/skills/<skill-name>/SKILL.md`

Authoring expectations:

- Include `name` and `description` in frontmatter.
- Keep instructions portable first; add platform-specific behavior only when needed.
- Prefer one canonical source skill over duplicated provider-specific edits.

## Recommended Portable Template

```markdown
---
name: your-skill-name
description: One sentence on what this skill does, when to use it, and when not to use it.
---

# Purpose
Brief statement of outcome.

## Inputs
- Expected arguments or context the user should provide.

## Steps
1. Step one in imperative form.
2. Step two in imperative form.
3. Validate results and report outcome.

## Output
- Exact deliverable format this skill should return.

## Optional Resources
- `references/REFERENCE.md` for deeper docs
- `scripts/do_work.sh` for deterministic automation
```

## Gotchas to Avoid

- Vague `description` text causes accidental or missed auto-invocation.
- Overloading one skill with many unrelated jobs hurts trigger quality.
- Putting long reference docs in `SKILL.md` wastes context budget.
- Using platform-specific fields without fallbacks reduces portability.

## Sources

- [Cursor Agent Skills docs](https://cursor.com/docs/context/skills)
- [Claude Code skills docs](https://code.claude.com/docs/en/skills)
- [OpenAI Codex skills docs](https://developers.openai.com/codex/skills/)
