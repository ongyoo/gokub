<p align="center">
  <img src="gokub_logo.png" alt="GOKUB - Go Project Kit" width="440">
</p>

<h1 align="center">GOKUB</h1>

<p align="center">
  <strong>Build a solid Go service without rebuilding the foundation.</strong><br>
  Guided setup, domain-focused architecture, production defaults, and AI-ready workflows.
</p>

<p align="center">
  <a href="https://github.com/ongyoo/gokub/actions/workflows/ci.yml"><img src="https://github.com/ongyoo/gokub/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/ongyoo/gokub/releases/latest"><img src="https://img.shields.io/github/v/release/ongyoo/gokub?display_name=tag" alt="Latest release"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-MIT-00bfe8.svg" alt="MIT license"></a>
</p>

GOKUB creates Go applications through a friendly terminal wizard. Pick an HTTP
framework (Gin, Fiber, or Echo), database, messaging provider, and Go version;
GOKUB generates ordinary Go code that your team fully owns.

Every service follows the same production layout:

```text
cmd/<name>-service/   entrypoint and wiring
config/               environment configuration (envconfig)
internal/<domain>/    model, repository, service, handler, router
internal/app/         composition and event contracts
pkg/                  api, database (gorm), error, httpserver, middleware, utils
```

```text
You choose                         GOKUB prepares
--------------------------------  -------------------------------------
Domain-focused service layout      cmd/<name>-service, internal/<domain>, pkg/*
Gin, Fiber, or Echo                Secure HTTP lifecycle and health checks
gorm + PostgreSQL                  Repository, service, handler, and router
RabbitMQ, Kafka, or NATS           Real, swappable event publisher
Go 1.26, 1.25, or custom          Matching go.mod, Docker, and CI versions
Developer and AI workflows        IDE debug, agent skills, and MCP config
```

## Install

The installer supports macOS and Linux on Intel and ARM:

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ongyoo/gokub/main/install.sh)"
```

Or use your preferred package manager:

```bash
# Homebrew
brew install ongyoo/tap/gokub

# Go
go install github.com/ongyoo/gokub/cmd/gokub@latest
```

Verify the installation:

```bash
gokub version
```

Generated projects require a Go toolchain matching the version selected in the
wizard. New projects default to Go 1.26, with Go 1.25 available as the conservative
baseline.

## Create Your First Project

Start the wizard:

```bash
gokub new
```

Use **Up/Down** to move, **Enter** to select, and press **Enter** on a text field
to accept the recommended value. No long command is required.

```text
Project name      example-api
Go module         github.com/example/example-api
Go version        1.26 (recommended)
Project style     monolith
Framework         gin | fiber | echo
Database          postgres
Messaging         none
Recipe            none
```

Run the generated service:

```bash
cd example-api
go test ./...
go run ./cmd/example-api-service
```

The project already includes Docker, GitHub Actions, VS Code and JetBrains debug
configuration, health checks, tests, agent guidance, and local environment examples.

## Everyday Workflows

Run `gokub` inside a generated project for a context-aware command menu, or use
commands directly:

| Goal | Command |
|---|---|
| Add a CRUD domain | `gokub add crud product` |
| Generate models from JSON | `gokub add model user --from user.json` |
| Add authentication | `gokub add auth` |
| Enable Kafka | `gokub enable messaging kafka` |
| Switch to RabbitMQ | `gokub switch messaging rabbitmq` |
| Apply an event-driven stack | `gokub recipe add event-driven` |
| Inspect project capabilities | `gokub status` |
| Check project health | `gokub doctor` |
| Run the quality gate | `gokub score --fail-under 80` |
| Check architecture boundaries | `gokub graph --check` |

Discover every command without leaving the terminal:

```bash
gokub help
gokub help new
gokub help add
```

## Templates That Fit Your Team

Choose the HTTP framework with `--framework gin|fiber|echo`, or import any local
project folder as a reusable team template:

```bash
gokub template add team-api ./path/to/example-project
gokub template list
gokub new
```

Custom templates support placeholders such as `{{project_name}}`, `{{module}}`,
and `{{go_version}}`. GOKUB excludes Git metadata, secrets, caches, dependencies,
and build output when importing a folder.

## Built For Developers And AI

Every generated project includes shared context for humans and coding agents:

- VS Code and GoLand/IntelliJ run and debug configurations
- `AGENTS.md`, `CLAUDE.md`, Gemini, and GitHub Copilot instructions
- Portable skills for Codex, Claude, Copilot, Gemini, and compatible agents
- `.codex/config.toml` and `.mcp.json` for MCP clients
- Machine-readable `status`, `doctor`, `score`, and graph output

Install or refresh agent skills in any GOKUB project:

```bash
gokub skill install
gokub agent init
gokub mcp serve
```

MCP mode uses clean JSON-RPC output, so the terminal logo never pollutes the
protocol stream.

Install the VS Code extension from the latest release:

```bash
curl -fL https://github.com/ongyoo/gokub/releases/latest/download/gokub-vscode.vsix \
  -o gokub-vscode.vsix
code --install-extension gokub-vscode.vsix
```

## Automation

Interactive use is the default, but every project choice is also scriptable:

```bash
gokub new payments \
  --module github.com/example/payments \
  --go-version 1.26 \
  --style monolith \
  --framework gin \
  --database postgres \
  --recipe api
```

JSON output is available for CI and agents:

```bash
gokub status --json
gokub doctor --json
gokub score --json
gokub graph --format json
```

## Update And Uninstall

```bash
gokub update --check
gokub update
gokub uninstall
```

For Homebrew installations, use `brew upgrade gokub` and `brew uninstall gokub`
so Homebrew keeps its package state consistent.

## Documentation

| Guide | What you will find |
|---|---|
| [Getting started](docs/getting-started.md) | Installation, project creation, and first run |
| [CLI reference](docs/cli-reference.md) | Commands, features, capabilities, and recipes |
| [Project templates](docs/project-templates.md) | Monolith, microservices, stack, and defaults |
| [Go version policy](docs/go-versions.md) | Recommended, conservative, and custom versions |
| [Custom templates](docs/custom-templates.md) | Turn a local folder into a reusable template |
| [JSON model generator](docs/json-model-generator.md) | Generate typed Go models from JSON or JSON Schema |
| [AI and IDE integrations](docs/integrations.md) | Codex, Claude, Copilot, MCP, VS Code, and JetBrains |
| [Full documentation](docs/README.md) | Upgrades, plugins, skills, development, and releases |

## Platform Support

| Platform | Intel | ARM |
|---|:---:|:---:|
| macOS | Yes | Yes |
| Linux | Yes | Yes |

GOKUB is available under the [MIT License](LICENSE). Contributions, templates,
recipes, providers, plugins, and skill packs are welcome.

---

<p align="center">
  Powered by <a href="https://www.roomkub.com"><strong>Roomkub</strong></a><br>
  <a href="mailto:roomkub.thailand@gmail.com">roomkub.thailand@gmail.com</a>
</p>
