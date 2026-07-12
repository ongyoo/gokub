# Agent Skills

GOKUB ships a portable skill pack for Codex, Claude, GitHub Copilot, Gemini, and
other agents that implement the Agent Skills standard. The skills are embedded in
the CLI binary, so Homebrew and one-line installations do not need this source
repository.

## Install

New GOKUB projects receive all skills automatically. For an existing project:

```bash
gokub skill install
```

Install for one agent:

```bash
gokub skill install --agent codex
gokub skill install --agent claude
gokub skill install --agent copilot
gokub skill install --agent gemini
gokub skill install --agent portable
```

Check or remove installations:

```bash
gokub skill list
gokub skill remove --agent all
```

Existing files are preserved. Refresh GOKUB-managed skill and instruction files
explicitly with:

```bash
gokub skill install --force
```

Review customized files before using `--force` because it intentionally replaces
files at the selected agent target.

## Included Skills

### gokub-project

General workflow for reading `.gokub.yaml`, respecting monolith/microservices
boundaries, using CLI or MCP tools, implementing changes, and running verification.

### gokub-add-domain

Focused workflow for adding models, repository ports, services, handlers, routes,
events, infrastructure adapters, and tests without leaking framework dependencies
into business logic.

### gokub-verify-change

Review and delivery workflow covering regressions, context, errors, cleanup,
timeouts, concurrency, secrets, authorization, dependency changes, race tests, vet,
build, and doctor checks.

## Installation Paths

| Agent | Project path |
|---|---|
| Codex and portable agents | `.agents/skills/<skill>/SKILL.md` |
| Claude | `.claude/skills/<skill>/SKILL.md` |
| GitHub Copilot | `.github/skills/<skill>/SKILL.md` |
| Gemini and other compatible agents | `.agents/skills/<skill>/SKILL.md` |

GOKUB also creates `.github/copilot-instructions.md` and `GEMINI.md`. Existing
`AGENTS.md`, `CLAUDE.md`, `.codex/config.toml`, and `.mcp.json` continue to provide
project-wide guidance and MCP configuration; skills contain task-specific workflows
loaded only when relevant.

## MCP Installation

An agent connected to `gokub mcp serve` can call `gokub_install_skills` with an
optional `agent` and `force` argument. This allows an agent to bootstrap the skill
pack itself after the user grants it project write access.

## Maintaining The Pack

Canonical sources live under `skill-packs/`. Edit those files, run the official
skill validator, then rebuild GOKUB. Do not edit generated copies as the source of
truth.
