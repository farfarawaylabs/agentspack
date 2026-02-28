---
name: create-new-skill
description: Create or update a skill by delegating to the create-cross-platform-skill skill.
---

# Create New Skill

Use this command when the user asks to create, update, revise, or migrate a skill.

## Input

- Use `$ARGUMENTS` as the skill request.
- If `$ARGUMENTS` is empty, ask the user what skill they want to create or update before proceeding.

## Execution

1. Delegate execution to the `create-cross-platform-skill` skill.
2. Pass the user's request and constraints through to that skill.
3. Follow that skill's process completely instead of duplicating its logic here.

## Guardrails

- Default behavior is cross-platform unless the user explicitly asks for a single-platform exception.
- Ensure resulting skill files are created or updated in relevant provider folders:
  - `.cursor/skills/`
  - `.claude/skills/`
  - `.agents/skills/`

## Completion Output

Report:

- The skill name that was created or updated
- Which provider folders were modified
- Any explicit user constraints that were applied

Skill request:

$ARGUMENTS
