## Coding style best practices

- **Follow Existing Patterns First**: Match the project’s established architecture, naming, error-handling, logging, and folder conventions before introducing new patterns.
- **Consistent Naming Conventions**: Use consistent, descriptive names for variables, functions, types/classes, and files.
- **Meaningful Names**: Prefer names that reveal intent; avoid abbreviations unless they’re widely standard in the domain.
- **Self-Documenting Code**: Write code that explains itself through clear structure and naming.
- **Automated Formatting**: Use auto-formatters/linters to enforce indentation, spacing, imports, and line breaks.
- **Small, Focused Units**: Keep functions/modules focused on a single responsibility; build systems from composable pieces.
- **Readable Over Clever**: Prefer straightforward solutions over “smart” ones; optimize for clarity and maintainability.
- **DRY, But Don’t Over-Abstract**: Reduce real duplication, but avoid premature abstractions that make code harder to follow.
- **Remove Dead Code**: Delete unused code, commented-out blocks, and unused imports rather than leaving clutter.
- **Explicit Inputs/Outputs**: Keep function boundaries clear; minimize hidden side effects and shared mutable state.
- **Clear Error Handling**: Handle errors at the right layer; don’t swallow failures; use actionable error messages.
- **Validate at Boundaries**: Treat inputs from users/files/network/DB as untrusted; validate/sanitize at the edges.
- **Security & Secrets Hygiene**: Never hardcode secrets; avoid logging sensitive data; prefer least-privilege access patterns.
- **Logging & Observability**: Add logs/metrics where they help debugging; keep them consistent; avoid noisy logs.
- **Document the “Why”**: Comment on intent, constraints, and tradeoffs; document non-obvious behavior and invariants.
- **Consistent Project Structure**: Organize files and directories in a predictable, logical structure that team members can navigate easily. Always read current project structure to learn about its conventions, and follow them.

## Constraints / assumptions

- **Backward compatibility only when required**: Unless specifically instructed otherwise, assume you do not need to add compatibility shims or legacy support.
