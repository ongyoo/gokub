# Getting Started

## Install

### Installer

```bash
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ongyoo/gokub/main/install.sh)"
```

The installer supports macOS and Linux on Intel and ARM. It verifies the release
SHA-256 and installs into `/usr/local/bin` or `~/.local/bin` without requiring
`sudo` when a user-local installation is needed. Downloads and redirects are HTTPS
only; release tags, checksums, and the extracted executable are validated before
installation.

The interactive wizard and arrow-key menus are supported on both macOS and Linux.

### Go

```bash
go install github.com/ongyoo/gokub/cmd/gokub@latest
```

### Homebrew

```bash
brew install ongyoo/tap/gokub
```

## Create A Project

Start the interactive Command Center:

```bash
gokub
```

Select **New project** with Up/Down and Enter. To skip the Command Center and open
project creation directly:

```bash
gokub new
```

Use Up/Down to move, Enter to select, and Enter on a text field to accept its
recommended value. The wizard configures:

```text
Project name      example-api
Go module         github.com/example/example-api
Go version        1.26 recommended | 1.25 conservative | custom
Project style     monolith | microservices
Template          monolith | microservices | gin-clean | fiber-clean | custom
Framework         gin | fiber | grpc | none
Database          postgres | mongodb | none
Architecture      clean | hexagonal | layered
Messaging         none | kafka | rabbitmq | nats
Recipe            none | api | event-driven
```

For scripts and CI, pass options directly:

```bash
gokub new payments --module github.com/example/payments --go-version 1.26 --style monolith
gokub new platform --module github.com/example/platform --style microservices
```

GOKUB downloads pinned dependencies, writes `go.sum`, and leaves the generated
project ready to test.

## Run The Project

Enter the generated directory and use its generated development configuration:

```bash
go test ./...
go run ./cmd/<project-name>
```

For microservices, run an entrypoint such as `./cmd/gateway` or
`./cmd/example-service`. VS Code and JetBrains run/debug configurations are already
included.

## Next Steps

```bash
gokub status
gokub doctor
gokub help
```

See the [CLI reference](cli-reference.md) to add capabilities or apply a recipe.
See the [Go version policy](go-versions.md) before selecting an older team baseline.
