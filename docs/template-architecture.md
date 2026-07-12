# GOKUB Template Architecture

The default templates distill reusable production patterns from audited monolith
and microservices sources. Their local copies were removed after the review because
they contained application-specific code, nested repositories, local caches,
reports, and environment files that must not become part of the GOKUB distribution.
The adoption map is preserved in `docs/reference-audit.md`.

## Adopted Conventions

- One entrypoint for monolith projects; gateway and service entrypoints for
  microservices projects.
- Domain code under `internal/domain/<name>` with model, repository port, service,
  and focused tests.
- Infrastructure adapters under `internal/platform`.
- Environment configuration loaded once and validated during startup.
- Structured JSON logging with request ID, access log, panic recovery, and secure
  HTTP headers.
- Explicit HTTP timeouts, bounded request headers, signal-aware graceful shutdown,
  liveness, readiness, and container health checks.
- Multi-stage, non-root distroless container images.
- CI formatting, vet, race detection, coverage, tests, and build checks.
- Shared Run/Debug configuration for VS Code and JetBrains IDEs.
- Durable guidance for Codex and Claude plus a local stdio MCP server.

## Deliberate Changes

The templates do not copy business domains, secrets, dynamic SQL ordering, startup
`panic` calls, or private dependencies from a reference application. The monolith
and microservices templates intentionally install a pinned core library set and
generate compiling adapters. Smaller `gin-clean`, `fiber-clean`, worker, and custom
templates can remain dependency-light.

## Agent Contract

Generated projects expose these MCP tools through `gokub mcp serve`:

- `gokub_project_status`
- `gokub_doctor`
- `gokub_catalog`
- `gokub_add_feature`
- `gokub_apply_recipe`

Write tools validate names and supported catalog values before changing generated
files or `.gokub.yaml`.
