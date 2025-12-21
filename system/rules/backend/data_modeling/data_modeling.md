## Data modeling best practices

### General (applies to relational + NoSQL)

- **Model the Domain, Not the UI**: Start from real domain concepts and workflows; don’t shape the schema around a single screen or endpoint.
- **Stable Identifiers**: Use stable primary identifiers for entities; avoid identifiers derived from mutable attributes.
- **Explicit Ownership & Boundaries**: Define which entity “owns” which data; avoid unclear shared/mirrored ownership.
- **Consistency Rules as Invariants**: Write down invariants (e.g., “an order total equals sum of line items”) and enforce them in one place (DB constraints, transactions, or application layer).
- **Schema Evolution**: Assume the schema will change. Prefer additive changes; avoid breaking existing readers/writers; plan migrations/rollouts.
- **Avoid Duplication Unless Intentional**: Duplicate only for performance/read reasons, and document the source of truth + sync strategy.
- **Indexing Strategy**: Add indexes to support critical queries; remove unused indexes; keep indexes aligned with actual access patterns.
- **Keep Writes Simple & Safe**: Prefer simple, atomic writes; make multi-step writes transactional or compensating where possible.
- **Auditability**: Track `created_at`, `updated_at`, and (when needed) `created_by`/`updated_by`. Use soft delete only when required and define retention rules.
- **Security & Data Access**: Model authorization needs early (tenant_id/org_id ownership, row/doc access rules). Don’t rely on obscurity.
- **PII/Secrets Hygiene**: Minimize stored sensitive data; encrypt where required; avoid storing derived secrets; apply least-privilege access.
- **Validation at Boundaries**: Validate data before persisting; treat DB as the last line of defense, not the first.
- **Plan for Scale Drivers**: Identify growth axes (users, events, files) and ensure the model won’t hit obvious limits.
