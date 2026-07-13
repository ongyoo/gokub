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

## Install From GitHub

Search public repositories carrying the `gokub-template` GitHub topic:

```bash
gokub template search
gokub template search api
gokub template search api --install
```

`--install` opens an arrow-key selector and then uses the same validated installer.
Set `GITHUB_TOKEN` when using search heavily or from CI to increase the GitHub API
rate limit. Search reads repository metadata only and does not clone code.

Install a public community template without cloning it manually:

```bash
gokub template install owner/go-service-template
gokub template install https://github.com/owner/go-service-template
```

Pin a reviewed branch or tag and select a template from a monorepo:

```bash
gokub template install owner/go-templates \
  --ref v1.2.0 \
  --subdir templates/api \
  --name team-api
```

Remote installation accepts HTTPS GitHub repositories without embedded credentials.
GOKUB performs a shallow clone with a timeout, copies through the same secret and
symlink filters as local templates, removes Git metadata, and records source/ref
metadata in the private template store. Review third-party source before generating
a project from it.

## Commands

```bash
gokub template list
gokub template search [query]
gokub template install <owner/repo>
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
| `{{go_version}}` | `1.26`, `1.25`, or the selected custom version |
| `{{template}}` | `team-service` |
| `{{style}}` | `monolith` or `microservices` |
| `{{framework}}` | `gin`, `fiber`, `grpc`, or `none` |
| `{{database}}` | `postgres`, `mongodb`, or `none` |
| `{{architecture}}` | `clean`, `hexagonal`, or `layered` |
| `{{messaging}}` | `none`, `kafka`, `rabbitmq`, or `nats` |

Example `go.mod`:

```go
module {{module}}

go {{go_version}}
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

For a team, keep template source in its own reviewed repository and pin a release
tag with `gokub template install --ref`. A checked-out path and `template add`
remain available for private repositories authenticated by your existing Git tools.

## Debugging

```bash
gokub template list
gokub help template
gokub doctor
```

If a template name already exists, remove it first. If a generated project fails,
inspect its rendered `go.mod`, `.gokub.yaml`, and placeholder values before adding
more features.
