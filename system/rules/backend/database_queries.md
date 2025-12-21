## Database query best practices

- **Prevent Injection**: Use parameterized queries / prepared statements / ORM query builders; never interpolate untrusted input into queries.
- **Validate Query Inputs**: Constrain sort fields, filter keys, and page sizes to an allowlist; reject unknown operators/fields.
- **Avoid N+1 Queries**: Batch, join, or eager-load related data where appropriate; profile and fix hot paths.
- **Select Only Needed Data**: Fetch only required columns/fields; avoid `SELECT *` and over-fetching large blobs.
- **Use the Right Pagination**: Prefer cursor/keyset pagination for large datasets; avoid deep offset pagination in hot paths.
- **Index to Match Access Patterns**: Add indexes that support real WHERE/JOIN/ORDER BY patterns; periodically review and remove unused indexes.
- **Keep Queries Predictable**: Avoid query shapes that change drastically with input (e.g., unbounded `IN` lists); cap complexity and sizes.
- **Be Careful with Text Search**: Use purpose-built search (DB indexes/FTS/external search) rather than `%LIKE%` scans on large tables.
- **Transactions for Related Writes**: Wrap related changes in transactions when atomicity is required; keep transactions short.
- **Concurrency & Locking Awareness**: Avoid long-running transactions/locks; choose appropriate isolation/locking strategies for the use case.
- **Set Query Timeouts**: Enforce timeouts and sensible limits (rows scanned/returned) to prevent runaway queries.
- **Handle Partial Failures**: Make retries safe (idempotent writes, dedupe where needed); avoid retrying non-transient errors.
- **Cache Where It Helps**: Cache expensive or high-frequency reads when appropriate; define TTLs and invalidation strategies.
- **Measure and Observe**: Log query timing and key metadata (not sensitive data); use EXPLAIN/profilers to optimize before guessing.
- **Avoid Doing Heavy Work in the DB by Accident**: Push large computations/formatting to the app layer unless the DB is the right tool (aggregations, filtering, sorting).
- **NoSQL-specific**: Design queries around indexes/partitions; avoid full scans; understand consistency guarantees and use the appropriate read/write consistency level.
