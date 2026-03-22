---
name: refresh-repo-docs
description: Refresh repository documentation so it matches the current codebase, features, and architecture. Use when the user asks to fix docs drift, improve documentation hygiene, update a PRD or README, audit onboarding docs, reconcile documentation against code, restore a single source of truth, reorganize `.agentspack` versus root `Docs`, or compress a large todo file.
---

# Repo Docs Refresh

## Goal

Bring project documentation back in sync with the real codebase without turning the docs into a bloated museum of every historical detail.

## When to Use

- The codebase changed and docs likely drifted
- The user asks to update `PRD.md`, product docs, technical docs, or onboarding docs
- The repo has both root `Docs/` content and `.agentspack` docs that may be overlapping
- `todos.md` has grown too large and needs to be compressed into a useful handoff artifact

## Required Workflow

1. Start with a plan. Create a short task list for the audit, update, relocation, and verification work.
2. Locate the canonical documentation paths before editing:
   - the project PRD, such as `PRD.md`, `.agents/prd.md`, `.agents/PRD.md`, or the repo's established equivalent
   - the main todo file, such as `todos.md`, `.agents/todos.md`, or the repo's established equivalent
   - root `Docs/` or the repo's existing root docs folder convention
   - `.agentspack/Docs/` and `.agentspack/Product/` if present
   - `README.md` and any other clearly project-owned documentation folders the repo already uses
3. Read each relevant document, then validate it against the actual code, structure, commands, config, and architecture. Treat the codebase as the source of truth unless the user says otherwise.
4. If product docs intentionally describe future or target behavior that is not yet implemented, do not silently rewrite that intent away. Mark the gap clearly as planned, pending, or not yet implemented.
5. Update or relocate docs as needed, keeping the documentation set small, accurate, and easy to navigate.
6. After moving or merging docs, update internal links, cross-references, onboarding pointers, and any references from `README.md`, agent instructions, or automation that mention the old paths.
7. Verify the final docs still cover what a new contributor needs to understand and work effectively in the repository.

## PRD Rules

- Update the project's canonical PRD so it reflects the current product, features, architecture, workflows, and important constraints.
- Keep the PRD lean. Assume it may be loaded into context frequently, so trim stale detail, duplicate explanations, changelog-like history, and low-value prose.
- Preserve high-signal content:
  - current purpose and problem statement
  - current architecture and major system boundaries
  - core workflows
  - important constraints and conventions
  - near-term roadmap or open questions only if still relevant
- Do not let the PRD become a dump of implementation minutiae that belong in deeper docs.

## Documentation Placement Rules

- Keep `.agentspack/Docs/` or `.agentspack/Product/` limited to the minimum documents required for coding effectively in the project.
- Examples of content that can stay under `.agentspack`:
  - technical requirements
  - folder structure
  - concise product context required during implementation
  - small workflow docs that agents need regularly
- Put broader or less frequently needed documentation under root `Docs/`, or the repo's existing root docs folder if that convention already exists.
- Examples of content that should usually live in root `Docs/`:
  - feature deep-dives
  - migration notes
  - architecture explainers
  - integration guides
  - historical decisions
  - extended operational or troubleshooting docs
- Do not split similar content across both locations without a clear reason.
- Prefer one minimal `.agentspack` documentation area, following the repo's existing convention if one already exists.
- Do not rename documentation folders just for cosmetics on case-sensitive filesystems unless the user explicitly asks for that cleanup.

## Audit Process

For each document you inspect:

1. Identify the document's purpose and intended audience.
2. Compare its claims to the current codebase, folder structure, commands, config, and architecture.
3. Use lightweight verification where relevant:
   - compare documented commands to actual scripts, make targets, or CLI entrypoints
   - check documented folders and files against the current repository structure
   - verify architecture claims against the current modules and boundaries in code
   - run tests or build commands if they are directly relevant and inexpensive
4. Mark content as one of:
   - still accurate
   - stale and needs correction
   - duplicated elsewhere
   - useful but in the wrong folder
   - obsolete and safe to remove or merge
5. Update the document so it describes the current state, not the intended past state. If the document intentionally describes future or target behavior, keep that intent and label it clearly as planned, pending, or not yet implemented.
6. If a document is mostly obsolete, compress it into a shorter summary or merge the important parts into a better canonical document instead of preserving bloat.

## `todos.md` Compression Rules

- Treat `todos.md` as a handoff artifact, not a permanent event log.
- Compress large completed sections into concise summaries that preserve useful context for a new contributor.
- Keep:
  - active work
  - pending decisions
  - unresolved risks
  - notable architectural changes worth knowing
  - short summaries of major completed milestones
- Remove or condense:
  - repetitive completed checklist items
  - low-value execution trivia
  - step-by-step history that no longer affects future work
- When compressing, preserve enough context that someone new can understand what was done and what still matters.
- Prefer a compact structure such as:

```markdown
## Active
- Current work in progress

## Pending Decisions
- Open questions or blockers

## Recent Milestones
- Short summaries of major completed work

## Risks / Follow-ups
- Important unresolved items
```

## Quality Bar

- Prefer accuracy over completeness theater.
- Prefer one canonical explanation over repeated variants.
- Prefer concise summaries over long historical timelines.
- If docs and code disagree, update the docs or explicitly call out the uncertainty.
- Keep terminology consistent across the PRD, `.agentspack` docs, root docs, `README.md`, and the canonical todo file.

## Output

When finishing, report:

- which docs were updated, moved, merged, or removed
- any assumptions made where code and docs were ambiguous
- any remaining gaps that still need user or team confirmation
