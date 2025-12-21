## API endpoint standards and conventions

- **Resource-Oriented Design**: Model endpoints around resources and relationships; avoid RPC-style endpoints unless there’s a clear reason.
- **RESTful Methods**: Use HTTP methods correctly (GET, POST, PUT, PATCH, DELETE) and keep semantics consistent.
- **Consistent Naming**: Use consistent naming for paths (lowercase, hyphenated or underscored) and stick to it across the API.
- **Plural Nouns**: Use plural nouns for collections (e.g., `/users`, `/products`) and stable identifiers for items (e.g., `/users/{id}`).
- **Versioning**: Use an explicit versioning strategy (path or header) for breaking changes; avoid breaking existing clients unintentionally.
- **Limit Nested Resources**: Keep nesting shallow (typically ≤ 2 levels). Prefer linking/filters over deeply nested URLs.
- **Query Parameters for Retrieval**: Use query params for filtering, sorting, search, and pagination rather than new endpoints.
- **Consistent Pagination**: Standardize pagination inputs/outputs (limit/offset or cursor-based). Return pagination metadata consistently.
- **Idempotency Where Needed**: Ensure PUT/DELETE are idempotent; support idempotency keys for retryable POST operations that create resources.
- **Consistent Status Codes**: Return accurate status codes (200/201/204, 400/401/403/404/409/422, 429, 5xx) and use them consistently.
- **Standard Error Envelope**: Return a consistent error shape across endpoints (stable error code, human message, optional details, request/trace id).
- **Validation & Sanitization**: Validate request body, query, and path params at the boundary; reject invalid inputs with clear, consistent errors.
- **Authentication & Authorization**: Enforce authn/authz consistently; never rely on the client; return 401 vs 403 correctly.
- **Content Negotiation**: Be explicit about content types (e.g., JSON). Avoid surprising behavior based on implicit defaults.
- **Caching**: For cacheable GETs, define caching behavior (ETag/If-None-Match, Cache-Control) where it helps.
- **Rate Limiting**: Apply rate limiting as needed and include rate limit information in headers (and 429 with retry guidance).
- **Observability**: Include correlation/request ids; log key request metadata and timing (without sensitive data).
- **Documentation by Contract**: Keep an API spec up to date (OpenAPI/JSON schema equivalent), including examples and error cases.
