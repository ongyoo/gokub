# Architecture Reference

## Monolith

- `cmd/<project>` owns composition and process lifecycle.
- `internal/domain/<name>` owns models, repository ports, and business services.
- `internal/http` owns routing, request/response mapping, and middleware.
- `internal/platform` owns database, cache, messaging, auth, and telemetry adapters.

Keep domain-to-domain calls explicit through service interfaces. Prefer one database
transaction boundary per use case rather than exposing transactions to handlers.

## Microservices

- Each `cmd/<service>` is independently runnable and deployable.
- A service owns its domain data and internal implementation.
- `pkg/contracts` contains stable transport/event contracts, not shared business logic.
- Cross-service communication must have timeouts, cancellation, and observable errors.

Do not create distributed services only to mirror folders. Split when ownership,
scaling, reliability, or release cadence requires an independent process.

## Adding A Domain

1. Define the model and repository interface in the domain package.
2. Implement business behavior in a service using constructor injection.
3. Implement storage or external clients under `internal/platform` or a domain
   adapter file that depends on the domain interface.
4. Map transport input/output at the handler boundary.
5. Wire dependencies only in the relevant `cmd` composition root.
6. Test the service with a small repository stub and test transport behavior separately.

Use `gokub graph` before and after moving code across architecture boundaries. Use
`gokub graph --format mermaid` when a review needs a portable diagram.

## Events

- Use the typed envelope in `pkg/contracts`.
- Treat event type and payload shape as versioned contracts.
- Publish after durable state changes; use an outbox when atomic delivery matters.
- Make consumers idempotent and define retry/dead-letter behavior explicitly.
