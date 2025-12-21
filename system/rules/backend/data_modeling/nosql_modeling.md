### Document / NoSQL modeling guidelines (Firestore, DynamoDB, Mongo, etc.)

- **Design from Access Patterns**: Start with the queries you must support. Model to avoid expensive scans and cross-collection joins.
- **Denormalize Intentionally**: Duplicate data to serve reads when needed, but define the source of truth and how duplicates stay consistent.
- **Keep Documents Bounded**: Avoid “unbounded arrays” and ever-growing documents. Use subcollections/partitioning for high-cardinality data.
- **Sharding & Hotspots**: Avoid write hotspots (single doc updated by many clients). Partition counters/feeds/timelines.
- **Atomicity Limits**: Respect transaction/batch limits; design workflows so that partial failure is recoverable and idempotent.
- **Consistent Timestamps & Ordering**: Use server-side timestamps where possible; design for pagination (cursor-like fields).
- **Security Rules Compatibility**: Model documents so that access control can be enforced cleanly (per-tenant/per-user paths, explicit owner fields).
