---
name: remember
description: Persist a user-provided memory in both AGENTS and CLAUDE project instruction files.
---

# Remember

Save the user's memory so it survives future sessions.

## Input

- Use `$ARGUMENTS` as the memory text.
- If `$ARGUMENTS` is empty, ask the user what they want you to remember before editing files.

## File Targets

Resolve and update both targets:

1. AGENTS target:
   - Prefer `AGENTS.md`
   - Else use `.AGENTS` if it exists
   - Else create `AGENTS.md`
2. CLAUDE target:
   - Prefer `CLAUDE.md`
   - Else use `.CLAUDE` if it exists
   - Else create `CLAUDE.md`

## Update Rules

For each target file:

1. Ensure a section exists named `## Things to remember`.
   - If an equivalent heading already exists with different casing (for example `## Things to Remember`), reuse it.
   - If no equivalent section exists, append it at the end of the file.
2. Under that section, add a bullet: `- <memory>`.
3. Do not add duplicates if the same memory already exists.
4. Preserve unrelated content.

## Completion Output

Report:

- Which AGENTS file was updated or created
- Which CLAUDE file was updated or created
- Whether the memory was newly added or already present

Memory to save:

$ARGUMENTS
