## Error handling best practices

- **User-Friendly Messages**: Provide clear, actionable messages to users; avoid exposing stack traces, internals, or sensitive data.
- **Fail Fast and Explicitly**: Validate inputs and preconditions early; prevent invalid state from propagating.
- **Use Specific Error Types**: Throw/return specific error types (or error codes) rather than generic ones to enable targeted handling.
- **Preserve Context**: When rethrowing/wrapping, keep the original error/cause and add useful context (operation, identifiers, parameters).
- **Don’t Catch What You Can’t Handle**: Avoid broad catch-all handlers except at explicit boundaries; never swallow errors silently.
- **Centralize at Boundaries**: Convert internal errors into user/API-safe responses at boundaries (API/controller/CLI entrypoints), not deep inside core logic.
- **Log with Structure (and Care)**: Log actionable context (request id, component, operation) while redacting secrets/PII; avoid noisy logs.
- **Define Error Contracts**: Document which errors a module/API can emit and how callers should handle them (retryable vs permanent, user-facing vs internal).
- **Retry Only Transient Failures**: Use exponential backoff + jitter for retryable failures; cap retries/time; avoid retrying on validation/auth/permission errors.
- **Timeouts and Cancellation**: Use explicit timeouts for I/O; propagate cancellation (where supported) to avoid hung requests and resource leaks.
- **Idempotency and Safe Retries**: Ensure retried operations are idempotent or protected (idempotency keys, dedupe) to prevent duplicates.
- **Resource Cleanup**: Always release resources (files, locks, sockets, transactions) via finally/defer/using/context managers.
- **Avoid Exceptions for Normal Control Flow**: Prefer explicit return values/results for expected “not found/empty” paths when idiomatic in the codebase.
