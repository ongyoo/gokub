---
name: gokub-add-domain
description: Add or extend a business domain in a GOKUB Go service with model, repository port, service behavior, transport mapping, dependency wiring, and tests. Use when creating CRUD resources, API modules, use cases, repository methods, handlers, routes, events, or a new service-owned domain.
---

# Add A GOKUB Domain

Build the domain from its business behavior outward. Do not start with framework or
database code.

## Process

1. Read `.gokub.yaml`, `AGENTS.md`, and the nearest existing domain package.
2. Use `gokub add crud <name>` when its scaffold matches the request; otherwise
   create the package within the established domain location.
3. Define domain models and repository interfaces without HTTP, SQL, queue, or SDK
   types in their signatures. When representative JSON or JSON Schema exists, use
   `gokub add model <name> --from <file.json>` as a typed starting point and review
   its optional fields before adding behavior.
4. Implement service behavior with constructor-injected interfaces and propagated
   `context.Context`.
5. Validate inputs at the service or transport boundary and wrap errors with the
   failed operation.
6. Implement infrastructure adapters outside core business logic.
7. Map requests and responses in handlers. Do not expose persistence models merely
   because their fields currently match the API.
8. Wire constructors and routes in the relevant `cmd` composition root.
9. Add service tests using a focused repository stub and transport tests for status,
   validation, and error mapping.

## Microservices

- Keep domain state owned by one service.
- Put stable cross-service payloads in `pkg/contracts`, not shared business services.
- Add timeout, cancellation, retry, and idempotency decisions for remote operations.
- Use an outbox when state changes and event publication must be atomic.

## Completion

Run `gofmt`, focused tests, `go test -race ./...`, `go vet ./...`, and
`gokub doctor`. Confirm `gokub status` reflects any capability added through the CLI
or MCP tools.
