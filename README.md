<p align="center">
  <img src="gokub_logo.png" alt="GOKUB" width="520">
</p>

<h1 align="center">GOKUB</h1>

<p align="center">
  <strong>Start a production-ready Go service without spending day one on boilerplate.</strong>
</p>

<p align="center">
  Monolith or microservices. Core libraries included. Docker, CI, tests, IDE debug,
  Codex, Claude, and MCP ready from the first commit.
</p>

## Why GOKUB

Most Go projects need the same foundation: configuration, graceful shutdown,
health checks, secure HTTP defaults, structured logs, database connections,
messaging, tests, containers, and CI. GOKUB turns those decisions into a short,
repeatable wizard while keeping the generated code ordinary Go that your team owns.

The default structures were distilled from working production patterns for both
single-deployment applications and independently deployable services. The design
decisions were preserved in a documented audit before the local source copies were
removed.

See [`docs/reference-audit.md`](docs/reference-audit.md) for the complete adoption
and exclusion map.

Business names, secrets, private modules, caches, and application-specific code are
not copied into generated projects.

## Install

The final repository URL is not selected yet. Replace `<owner>/<repo>` when publishing.

### Homebrew

```bash
brew install <owner>/tap/gokub
```

### One-line installer

```bash
GOKUB_REPOSITORY=<owner>/<repo> /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/<owner>/<repo>/main/install.sh)"
```

The installer supports macOS and Linux on Intel and ARM. It verifies the release
SHA-256 before installing into `/usr/local/bin` or `~/.local/bin` without `sudo`.

### Go install

```bash
go install github.com/<owner>/<repo>/cmd/gokub@latest
```

For local development, run `make install`.

## Create A Project

```bash
gokub new
```

Move with Up/Down and confirm with Enter. Press Enter on text fields to accept the
recommended value.

```text
Project name      example-api
Go module         github.com/example/example-api
Project style     monolith | microservices
Template          monolith | microservices | gin-clean | fiber-clean | custom
Framework         gin | fiber | grpc | none
Database          postgres | mongodb | none
Architecture      clean | hexagonal | layered
Messaging         none | kafka | rabbitmq | nats
Recipe            none | api | event-driven
```

For scripts and CI:

```bash
gokub new payments --module github.com/example/payments --style monolith
gokub new platform --module github.com/example/platform --style microservices
```

GOKUB downloads and pins the selected template dependencies, writes `go.sum`, and
leaves the project ready for `go test ./...`.

## Monolith Or Microservices

| Choose | Best fit | Generated entrypoints | Deployment shape |
|---|---|---|---|
| `monolith` | New products, small teams, shared transactions | `cmd/<project>` | One image and one service |
| `microservices` | Independent ownership, scaling, or release cycles | `cmd/gateway`, `cmd/example-service` | Independently runnable services |

Start with monolith unless you already have a concrete reason to operate multiple
services. The domain and platform boundaries are intentionally similar, so code can
be extracted later without redesigning everything.

### Monolith structure

```text
cmd/<project>/                 application composition and lifecycle
internal/domain/example/      model, repository port, service, tests
internal/http/                router, health routes, secure middleware
internal/platform/            database, cache, auth, messaging, telemetry
pkg/contracts/                shared event contracts
migrations/                   database migrations
deployments/                  deployment assets
```

### Microservices structure

```text
cmd/gateway/                  public gateway entrypoint
cmd/example-service/          example independently runnable service
internal/domain/example/      service-owned business rules
internal/platform/            reusable infrastructure adapters
pkg/contracts/                cross-service event contracts
docker-compose.yml            local multi-service environment
```

## Included Go Stack

The `monolith` and `microservices` templates use Go 1.25 and pin a practical core set:

| Area | Libraries |
|---|---|
| HTTP | Gin, Fiber, and hardened `net/http` lifecycle |
| Config and validation | caarlos0/env, validator |
| Data | pgx, MongoDB driver, go-redis |
| Messaging | franz-go Kafka, RabbitMQ, NATS |
| Security | golang-jwt, UUID, Go crypto |
| Observability | OpenTelemetry, Prometheus, structured `log/slog` |
| Delivery | golang-migrate, Docker, GitHub Actions |
| Testing | Go testing, race detector, Testify |

Generated adapters live under `internal/platform`; no external service connection is
opened until you wire and call it from the composition root.

## Production Defaults

- JSON structured logging and configurable log levels.
- Request IDs, access logs, panic recovery, and secure response headers.
- Read-header, read, write, idle, and graceful-shutdown timeouts.
- Liveness at `/health/live` and readiness at `/health/ready`.
- Non-root distroless container image with a built-in healthcheck.
- `.env.example` with no committed secrets.
- CI checks for formatting, vet, race conditions, coverage, tests, and build.
- VS Code and GoLand/IntelliJ Run/Debug configurations.

## Work With A Project

```bash
gokub status
gokub doctor
gokub add crud product
gokub add auth
gokub enable messaging kafka
gokub switch messaging rabbitmq
gokub recipe add event-driven
```

Use `gokub help` or `gokub help <command>` for command-specific examples.

## Recipes

- `api`: authentication, PostgreSQL, Redis, Docker, and CI.
- `event-driven`: PostgreSQL, Kafka, outbox, OpenTelemetry, Docker, and CI.

Recipes update `.gokub.yaml` and generate only missing feature scaffolds.

## Custom Templates

Any folder can become a reusable wizard option:

```bash
gokub template add ./my-team-template
gokub template add team-api ./another-template
gokub template list
gokub new
```

Use a folder once without installing it:

```bash
gokub new example-api --template ./my-template
```

Template paths and text files support:

```text
{{project_name}}  {{module}}       {{template}}      {{style}}
{{framework}}     {{database}}     {{architecture}}  {{messaging}}
```

Imported templates are copied into `~/.gokub/templates`. GOKUB excludes `.git`,
`.env`, caches, dependencies, build output, and symlinks to avoid packaging local
state or secrets.

## AI Collaboration And MCP

Every generated project includes:

- `AGENTS.md` for Codex and compatible coding agents.
- `CLAUDE.md` for Claude Code project guidance.
- `.codex/config.toml` for Codex project MCP configuration.
- `.mcp.json` for Claude and MCP-compatible clients.
- Portable project skills for Codex, Claude, Copilot, Gemini, and other compatible agents.

Install or refresh the complete skill pack in any project:

```bash
gokub skill install
gokub skill list
```

The pack includes `gokub-project`, `gokub-add-domain`, and
`gokub-verify-change`. See [`docs/agent-skills.md`](docs/agent-skills.md) for agent
paths, targeted installation, updates, and removal.

The stdio server starts with:

```bash
gokub mcp serve
```

It exposes typed tools for project status, doctor checks, catalog discovery, feature
generation, and recipe application. MCP mode never prints the terminal logo to
stdout, keeping the JSON-RPC stream valid.

## VS Code Extension

Install the packaged extension:

```bash
code --install-extension dist/gokub-vscode.vsix
```

The Command Palette provides New Project, Add Custom Template, Add Feature, Status,
Doctor, Open MCP Configuration, and Uninstall CLI. Interactive workflows run in the
integrated terminal, so arrow-key menus behave exactly like the standalone CLI.

Set `gokub.executablePath` when the binary is not available on `PATH`.

## Update And Uninstall

```bash
gokub update --check
gokub uninstall
gokub uninstall --yes
gokub uninstall --purge
```

Normal uninstall removes only the active CLI binary. `--purge` also removes custom
templates and local GOKUB data under `~/.gokub`.

## Develop GOKUB

```bash
make fmt
make test
make build
make extension
```

- CLI binary: `dist/gokub`
- VS Code extension: `dist/gokub-vscode.vsix`
- Product specification: [`spec.md`](spec.md)
- Template decisions: [`docs/template-architecture.md`](docs/template-architecture.md)
- Reference audit: [`docs/reference-audit.md`](docs/reference-audit.md)
- Custom-template guide: [`docs/custom-templates.md`](docs/custom-templates.md)
- Agent-skills guide: [`docs/agent-skills.md`](docs/agent-skills.md)
- Homebrew setup: [`packaging/homebrew/README.md`](packaging/homebrew/README.md)

---

Powered by [Roomkub](https://www.roomkub.com)  
Contact: [roomkub.thailand@gmail.com](mailto:roomkub.thailand@gmail.com)
