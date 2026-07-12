# Custom Templates

GOKUB can turn any local folder into a reusable project template. Installed
templates are copied into `~/.gokub/templates`, so the original folder can be moved
or deleted afterward.

## Quick Start

Given this folder:

```text
my-service/
├── cmd/
│   └── {{project_name}}/
│       └── main.go
├── internal/
├── .env.example
├── go.mod
└── README.md
```

Install it using the folder name:

```bash
gokub template add ./my-service
```

Or choose the name shown in the wizard:

```bash
gokub template add team-service ./my-service
```

Run `gokub new` and select the installed template with Up/Down and Enter.

## Commands

```bash
gokub template list
gokub template add <path>
gokub template add <name> <path>
gokub template remove <name>
gokub new example-api --template ./one-time-template
```

To replace an installed template explicitly:

```bash
gokub template remove team-service
gokub template add team-service ./my-service
```

## Placeholders

Placeholders work in text-file contents, file names, and directory names.

| Placeholder | Example value |
|---|---|
| `{{project_name}}` | `payments-api` |
| `{{module}}` | `github.com/example/payments-api` |
| `{{template}}` | `team-service` |
| `{{style}}` | `monolith` or `microservices` |
| `{{framework}}` | `gin`, `fiber`, `grpc`, or `none` |
| `{{database}}` | `postgres`, `mongodb`, or `none` |
| `{{architecture}}` | `clean`, `hexagonal`, or `layered` |
| `{{messaging}}` | `none`, `kafka`, `rabbitmq`, or `nats` |

Example `go.mod`:

```go
module {{module}}

go 1.25
```

Example source path:

```text
cmd/{{project_name}}/main.go
```

## Files GOKUB Adds

After rendering the custom folder, GOKUB adds missing collaboration and IDE files
without overwriting versions supplied by the template:

- `AGENTS.md`
- `CLAUDE.md`
- `.codex/config.toml`
- `.mcp.json`
- `.vscode/launch.json`
- `.vscode/tasks.json`
- `.run/GOKUB.run.xml`
- `docs/gokub_logo.png`
- `.gokub.yaml`

## Import Safety

GOKUB skips these local or unsafe entries while importing:

- `.git`, `.gocache`, `.idea`, `.DS_Store`
- `.env` while preserving `.env.example`
- `node_modules`, `dist`, and `tmp`
- symbolic links

Binary files are copied without placeholder replacement. Text files retain their
permissions, including executable scripts.

Always review a third-party folder before installing it. Custom templates contain
code that will be copied into future projects.

## CI And Shared Templates

Set `GOKUB_HOME` to isolate template storage in CI:

```bash
export GOKUB_HOME="$PWD/.gokub-ci"
gokub template add team-service ./templates/team-service
gokub new smoke-test --template team-service --module example.com/smoke-test
```

For a team, keep template source in its own reviewed repository and install it from
a checked-out path. Pin the repository revision in CI before calling
`gokub template add`.

## Debugging

```bash
gokub template list
gokub help template
gokub doctor
```

If a template name already exists, remove it first. If a generated project fails,
inspect its rendered `go.mod`, `.gokub.yaml`, and placeholder values before adding
more features.
