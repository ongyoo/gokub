# CLI Reference

Use `gokub help <command>` for the complete flags and examples shipped with your
installed version.

Build and platform metadata is available for diagnostics and automation:

```bash
gokub version
gokub version --json
```

Release builds include the source commit and build date. JSON mode also reports the
repository, Go toolchain, operating system, and architecture without printing the
logo.

Running `gokub` without arguments opens a context-aware arrow-key Command Center in
an interactive terminal. In pipes and CI it prints help and never waits for input.

## Shell Completion

Install completion for the shell in `$SHELL`:

```bash
gokub completion install
```

An explicit shell can be selected with `gokub completion install bash`, `zsh`, or
`fish`. Bash and Zsh installation adds one marked, idempotent block to the matching
shell rc file. Fish uses its native user completion directory. Open a new terminal
after installation.

To inspect or source a generated script without changing configuration:

```bash
gokub completion zsh
gokub completion bash
gokub completion fish
```

## Projects

```bash
gokub new
gokub new example-api --module github.com/example/example-api --go-version 1.26
gokub status
gokub status --json
gokub doctor
gokub doctor --json
gokub score
gokub graph
gokub upgrade --check
```

`new` creates a project through the interactive wizard. Use `--go-version`
with a `major.minor` value for non-interactive creation; see the
[Go version policy](go-versions.md). `status` reads the project
manifest; JSON mode returns project metadata, features, recipes, and capability
provider state without reading environment files. `doctor` checks manifest readability, expected structure, configuration,
and generated project health. JSON output includes a stable summary and individual
checks, prints no logo, and exits unsuccessfully when any check fails.

`score` measures static project signals across Architecture, Security, Testing, and
Operations. It returns recommendations for every missed check:

```bash
gokub score
gokub score --json
gokub score --fail-under 80
```

The JSON mode does not print the terminal logo, so it can be consumed safely by CI
and automation. Scoring never executes project code. `--fail-under 80` exits with
an error when the score is lower than 80; it can be combined with `--json` without
changing the report schema.
Architecture scoring uses the same static cycle and boundary analysis as
`gokub graph --check`; `doctor` reports those violations as a failed check as well.

Generated projects expose the same gate through `make score`. The default minimum
is 80 and can be overridden without editing the Makefile:

```bash
make score SCORE_MIN=90
```

VS Code users can run **GOKUB: Quality Gate** and enter the minimum score.

`graph` parses imports between packages inside the project's Go module:

```bash
gokub graph
gokub graph --format mermaid
gokub graph --format json
gokub graph --include-tests
gokub graph --check
gokub graph --check --format json
```

Mermaid and JSON output do not print the logo. Test imports are excluded by default,
and the scanner never compiles or executes project code. `--check` fails on package
cycles and outward clean-architecture dependencies such as domain importing
platform or transport. Combined with JSON it returns the graph and typed violations.
Generated projects expose the gate through `make graph-check` and the VS Code task
**GOKUB: Architecture Check**.

## Features

```bash
gokub add auth
gokub add crud product
gokub add grpc
gokub add model user --from user.json
gokub remove auth
```

Available feature scaffolds include authentication, CRUD, PostgreSQL, MongoDB,
Redis, Kafka, RabbitMQ, NATS, gRPC, cron, email, WebSocket, OpenTelemetry, Docker,
GitHub Actions, and transactional outbox.

The model generator accepts sample JSON and JSON Schema. See the
[JSON model generator guide](json-model-generator.md) for type mapping, paths, and
overwrite controls.

## Capabilities

Capabilities describe what the project needs while providers describe how it is
implemented.

```bash
gokub enable messaging
gokub enable messaging kafka
gokub switch messaging rabbitmq
gokub disable messaging
```

| Capability | Providers |
|---|---|
| Authentication | auth |
| Cache | Redis |
| Database | PostgreSQL, MongoDB |
| Messaging | Kafka, RabbitMQ, NATS |
| Observability | OpenTelemetry |
| Infrastructure | Docker, GitHub Actions |

When a provider is omitted in an interactive terminal, GOKUB asks you to choose
with Up/Down and Enter.

## Recipes

```bash
gokub recipe list
gokub recipe add api
gokub recipe add event-driven
```

| Recipe | Adds |
|---|---|
| `api` | Auth, PostgreSQL, Redis, Docker, and GitHub Actions |
| `event-driven` | PostgreSQL, Kafka, outbox, OpenTelemetry, Docker, and GitHub Actions |

Recipes update `.gokub.yaml` and generate only missing feature scaffolds.

## Templates And Agents

```bash
gokub template add ./my-template
gokub template search api --install
gokub template install owner/go-service-template --ref v1.0.0
gokub template list
gokub skill install
gokub agent init
gokub mcp serve
gokub plugin list
```

See [Custom templates](custom-templates.md) and [Integrations](integrations.md) for
the full workflows. See [Plugins](plugins.md) for the executable plugin contract and
security model.

## Maintenance

```bash
gokub update --check
gokub update
gokub upgrade
gokub uninstall
gokub uninstall --yes
gokub uninstall --purge
```

Normal uninstall removes the active CLI binary. `--purge` also removes custom
templates and local GOKUB data. Homebrew installations must be removed with
`brew uninstall gokub` so Homebrew's package state remains consistent.

`update` checks the latest GitHub release, selects the current platform archive,
verifies it against `checksums.txt`, asks for confirmation, and atomically replaces
a standalone macOS/Linux binary. Use `--check` for read-only checks, `--yes` for
controlled automation, and `--json` for a machine-readable plan. Homebrew-managed
installations must use `brew upgrade gokub`.

`upgrade` versions and migrates `.gokub.yaml` after showing a plan and creating a
backup. See [Project upgrades](project-upgrades.md) for interactive, CI, JSON, and
rollback behavior.

## Roadmap Commands

Signed remote plugin publishing is not exposed by the current release. It requires
artifact signing, provenance, compatibility metadata, revocation, and a trusted
registry policy before remote installation can be enabled safely.
