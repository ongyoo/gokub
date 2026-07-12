# Reference Project Audit

This document preserves the design review performed before removing the local
the local reference directory. The audit covered 211 Go files: 135 from the
monolith source and 76 from the microservices source, plus their module, Docker,
CI, IDE, environment-example, API-specification, and test configuration files.

## Reference Roles

### Monolith Source

Used as the monolith reference. It demonstrated one application containing many
business modules, separate public/internal/consumer entrypoints, repository and
service interfaces, pipeline orchestration, MongoDB, Redis locks, transactions,
JWT middleware, typed errors, health routes, tracing, external clients, mocks,
and tests.

### Microservices Source

Used as the microservices reference. It demonstrated multiple service entrypoints,
domain packages with model/repository/service/handler/router files, shared
PostgreSQL and MongoDB adapters, Gin/Fiber servers, RabbitMQ events, cache-aside,
JWT middleware, Docker Compose, CI, IDE launch configuration, and isolated tests.

## Adopted In GOKUB

| Reference pattern | GOKUB implementation |
|---|---|
| Multiple `cmd` entrypoints | `monolith` creates one; `microservices` creates gateway and service entrypoints |
| Domain boundaries | `internal/domain/<name>` with model, repository port, service, and tests |
| Shared infrastructure | `internal/platform` adapters |
| Repository/service dependency direction | Domain owns interfaces; adapters remain outside domain logic |
| Context propagation | Repository, service, database, messaging, and shutdown APIs accept context |
| Graceful shutdown | Signal-aware lifecycle with bounded shutdown timeout |
| Health endpoints | `/health`, `/health/live`, and `/health/ready` |
| HTTP safety | Request IDs, access logs, panic recovery, secure headers, and server timeouts |
| Database clients | Compiling pgx and MongoDB v2 adapters with startup ping and cleanup |
| Cache | Compiling go-redis adapter with startup ping |
| Messaging | Provider-neutral contracts plus Kafka, RabbitMQ, and NATS publishers |
| Authentication | JWT signing adapter without embedded secrets |
| Validation | Validator-backed service input validation |
| Observability | `log/slog`, OpenTelemetry tracer, and pinned Prometheus client |
| Event contracts | Generic typed event envelope under `pkg/contracts` |
| Testing | Domain and HTTP tests, race detector, coverage, vet, and formatting CI |
| Local delivery | `.env.example`, Docker, Compose, migrations, and non-root images |
| Developer tools | VS Code, JetBrains, Codex, Claude, and MCP configuration |

## Improved Instead Of Copied

- Startup functions return useful errors instead of calling `panic` deep in adapters.
- Configuration never prints private keys or secrets.
- Current stable Go modules replace old, beta, duplicate, and private dependencies.
- `log/slog` replaces application-wide dependence on a third-party logger.
- Database adapters validate connectivity and close partially initialized clients.
- HTTP servers set bounded timeouts and header limits.
- Middleware emits generic internal errors and does not leak panic details to clients.
- Dynamic database ordering and unvalidated update maps are not included as defaults.
- The misspelled `radis` package and duplicated error packages were not preserved.
- Framework-specific lifecycle code is separated from domain contracts.

## Deliberately Not Included

- KBank, OMS, Nclavis, VOS, LINE private modules, and vendor-specific payloads.
- Merchant, onboarding, room, billing, meter, ticket, and tenant business rules.
- Business workflow pipelines and state transitions that cannot be generic defaults.
- Committed `.env` files, private module credentials, nested Git repositories,
  build caches, coverage reports, generated mocks, and OS metadata.
- Generated Swagger files tied to old business APIs.
- Redis locking and idempotency policy as global defaults; lock TTL, ownership,
  retries, and failure semantics must be chosen per workflow.
- Automatic database migrations at every service startup; deployment ownership
  differs between teams and should remain explicit.

## Available As Follow-up Features

The audit identified useful capabilities that should be explicit GOKUB features,
not silently enabled defaults: OpenAPI generation, transactional helpers, distributed
locks, idempotency middleware, resilient outbound HTTP clients, consumer workers,
outbox processing, and generated test doubles.

## Removal Decision

The reusable architecture, dependency choices, and exclusions are represented in
the generator, tests, this audit, and `docs/template-architecture.md`. No generated
file reads from `example-project`, so the 643 MB reference directory can be removed
without changing CLI output or template generation.
