### Relational modeling guidelines (SQL / relational ORMs)

- **Normalize by Default**: Keep a normalized core model; denormalize only for proven performance needs.
- **Use Constraints**: Prefer DB constraints where possible (NOT NULL, UNIQUE, CHECK, FOREIGN KEY) to enforce invariants.
- **Relationships Explicitness**: Model relations explicitly (1:1, 1:N, N:M via join tables). Prefer join tables over arrays for N:M.
- **Transactions for Consistency**: Use transactions for multi-row invariants and multi-step writes.
- **Clear Cascades**: Be explicit about cascade behavior (delete/update). Avoid accidental cascading deletes.
- **Migration Discipline**: Migrations must be reversible when possible and safe for production (additive first; backfill; then enforce constraints).
