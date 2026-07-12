---
name: gokub-project
description: Build, modify, debug, review, and verify Go services generated or managed by GOKUB. Use when working in a repository with .gokub.yaml, using GOKUB features or recipes, changing domain/service/repository/transport/platform code, diagnosing project health, or preparing a GOKUB project for delivery.
---

# GOKUB Project

Work with the project as ordinary maintainable Go while preserving its declared
structure and capability state.

## Workflow

1. Read `AGENTS.md` and `.gokub.yaml` before changing code.
2. Run `gokub status`, or call `gokub_project_status` through MCP when available.
3. Inspect the affected domain, transport, and platform packages before choosing
   where the change belongs.
4. Use `gokub add`, `gokub enable`, or `gokub recipe add` for supported scaffolding.
5. Implement business behavior behind domain-owned interfaces. Keep infrastructure
   dependencies in `internal/platform`.
6. Add focused tests at the changed boundary.
7. Run formatting, tests, vet, and `gokub doctor` before reporting completion.

## Architecture Rules

- Keep domain logic independent of HTTP frameworks, databases, queues, and SDKs.
- Pass `context.Context` through service, repository, and outbound operations.
- Return errors with operation context; do not log and return the same error at
  every layer.
- Load configuration once at startup. Never commit `.env` or print secrets.
- Preserve graceful shutdown, health endpoints, secure middleware, and timeouts.
- In microservices projects, avoid importing another service's `internal` package;
  share stable contracts through `pkg/contracts` or an explicitly versioned module.

Read [references/architecture.md](references/architecture.md) when adding a domain,
adapter, entrypoint, or cross-service contract.

## GOKUB Tools

Prefer MCP tools when exposed by `gokub mcp serve`:

- `gokub_project_status` for manifest state.
- `gokub_doctor` for structural checks.
- `gokub_catalog` before selecting a feature or recipe.
- `gokub_add_feature` and `gokub_apply_recipe` for repeatable generation.

Do not hand-edit `.gokub.yaml` when an equivalent CLI or MCP operation exists.
Review generated files before adapting them to domain requirements.

## Verification

Read [references/verification.md](references/verification.md) and run the smallest
relevant checks during development, followed by the complete required checks before
completion. Report commands that could not run and why.
