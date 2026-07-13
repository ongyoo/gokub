# Project Templates

## Choose A Project Style

| Style | Best fit | Entrypoints | Deployment |
|---|---|---|---|
| `monolith` | New products, small teams, shared transactions | `cmd/<project>` | One image and service |
| `microservices` | Independent ownership, scaling, or release cycles | `cmd/gateway`, `cmd/example-service` | Independently runnable services |

Start with monolith unless there is a concrete operational reason to own and deploy
multiple services. Both styles use similar domain and platform boundaries so a
domain can be extracted later.

## Monolith Layout

```text
cmd/<project>/                 application composition and lifecycle
internal/domain/example/      model, repository port, service, tests
internal/http/                router, health routes, secure middleware
internal/platform/            database, cache, auth, messaging, telemetry
pkg/contracts/                shared event contracts
migrations/                   database migrations
deployments/                  deployment assets
```

## Microservices Layout

```text
cmd/gateway/                  public gateway entrypoint
cmd/example-service/          independently runnable service
internal/domain/example/      service-owned business rules
internal/platform/            reusable infrastructure adapters
pkg/contracts/                cross-service event contracts
docker-compose.yml            local multi-service environment
```

## Included Go Stack

New projects default to Go 1.26; Go 1.25 is the supported conservative baseline.
The wizard also accepts a custom team version. See the
[Go version policy](go-versions.md) before selecting an older release.

| Area | Libraries |
|---|---|
| HTTP | Gin, Fiber, hardened `net/http` lifecycle |
| Config and validation | caarlos0/env, validator |
| Data | pgx, MongoDB driver, go-redis |
| Messaging | franz-go Kafka, RabbitMQ, NATS |
| Security | golang-jwt, UUID, Go crypto |
| Observability | OpenTelemetry, Prometheus, `log/slog` |
| Delivery | golang-migrate, Docker, GitHub Actions |
| Testing | Go testing, race detector, Testify |

Generated adapters live under `internal/platform`. No external service connection
is opened until it is wired and called from the composition root.

## Production Defaults

- JSON structured logging and configurable log levels.
- Request IDs, access logs, panic recovery, and secure response headers.
- HTTP timeouts and graceful shutdown.
- Liveness at `/health/live` and readiness at `/health/ready`.
- Non-root distroless container image with a healthcheck.
- `.env.example` without committed secrets.
- CI for formatting, vet, race detection, coverage, tests, and build.
- VS Code and GoLand/IntelliJ run/debug configurations.

For implementation decisions, see [Template architecture](template-architecture.md)
and the [Reference audit](reference-audit.md).
