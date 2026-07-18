package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	gokub "github.com/ongyoo/gokub"
	"github.com/ongyoo/gokub/internal/agentskills"
	"github.com/ongyoo/gokub/internal/goversion"
	"github.com/ongyoo/gokub/internal/manifest"
	"github.com/ongyoo/gokub/internal/projectmeta"
	customtemplates "github.com/ongyoo/gokub/internal/templates"
)

var resourceNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func NewProject(root string, m manifest.Manifest) error {
	if m.SchemaVersion == 0 {
		m.SchemaVersion = manifest.CurrentSchemaVersion
	}
	if m.GeneratorVersion == "" {
		m.GeneratorVersion = gokub.Version
	}
	if m.GoVersion == "" {
		m.GoVersion = goversion.Recommended
	}
	if err := manifest.Validate(m); err != nil {
		return err
	}
	if source, custom, err := customtemplates.Resolve(m.Template); err != nil {
		return err
	} else if custom {
		if err := customtemplates.Generate(source, root, m); err != nil {
			return err
		}
		target := filepath.Join(root, m.Name)
		if err := addProjectTooling(target, m); err != nil {
			return err
		}
		if err := manifest.Write(filepath.Join(target, manifest.FileName), m); err != nil {
			return err
		}
		return projectmeta.WriteMarker(target, gokub.Version, m)
	}
	return newKitProject(root, m)
}

func addProjectTooling(target string, m manifest.Manifest) error {
	files := map[string]string{
		"AGENTS.md":           agentsFile(m),
		"CLAUDE.md":           claudeFile(m),
		".codex/config.toml":  codexConfigFile(),
		".mcp.json":           mcpConfigFile(),
		".vscode/launch.json": vscodeLaunchFile(m),
		".vscode/tasks.json":  vscodeTasksFile(),
		".run/GOKUB.run.xml":  jetbrainsRunFile(m),
	}
	for name, content := range files {
		if err := writeNew(filepath.Join(target, name), content); err != nil {
			return err
		}
	}
	if logo, err := gokub.Assets.ReadFile("gokub_logo.png"); err == nil {
		if err := writeNewBytes(filepath.Join(target, "docs", "gokub_logo.png"), logo); err != nil {
			return err
		}
	}
	_, err := agentskills.Install(target, "all", false)
	return err
}

func WriteAgentFiles(root, provider string) ([]string, error) {
	return writeAgentFiles(root, provider, true, false)
}

// InitializeAgentFiles installs agent context into an existing project. By
// default it preserves instruction/config files already owned by the project.
func InitializeAgentFiles(root, provider string, force bool) ([]string, error) {
	return writeAgentFiles(root, provider, force, force)
}

func writeAgentFiles(root, provider string, replaceFiles, forceSkills bool) ([]string, error) {
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return nil, err
	}
	files := map[string]string{}
	switch provider {
	case "codex":
		files["AGENTS.md"] = agentsFile(m)
		files[filepath.Join(".codex", "config.toml")] = codexConfigFile()
	case "claude":
		files["CLAUDE.md"] = claudeFile(m)
		files[".mcp.json"] = mcpConfigFile()
	case "all", "":
		files["AGENTS.md"] = agentsFile(m)
		files["CLAUDE.md"] = claudeFile(m)
		files[filepath.Join(".codex", "config.toml")] = codexConfigFile()
		files[".mcp.json"] = mcpConfigFile()
	case "copilot", "gemini", "portable":
		// Agent-specific skill and instruction files are installed below.
	default:
		return nil, fmt.Errorf("unknown agent provider %q", provider)
	}
	written := make([]string, 0, len(files))
	for name, content := range files {
		path := filepath.Join(root, name)
		if !replaceFiles {
			if _, err := os.Stat(path); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, err
		}
		written = append(written, name)
	}
	installed, err := agentskills.Install(root, provider, forceSkills)
	if err != nil {
		return nil, err
	}
	written = append(written, installed...)
	return written, nil
}

func AddFeature(root, feature, name string) error {
	if name == "" {
		name = feature
	}
	switch feature {
	case "crud":
		return addCRUD(root, name)
	case "kafka", "rabbitmq", "nats":
		return wireMessaging(root, feature)
	case "auth", "redis", "postgres", "mongodb", "grpc", "cron", "email", "websocket", "otel", "outbox":
		return writeNew(filepath.Join(root, "internal", feature, feature+".go"), featureScaffold(feature))
	case "docker":
		return writeNew(filepath.Join(root, "deployments", "docker.md"), "# Docker\n\nDocker support is enabled for this project.\n")
	case "github-actions":
		m, err := manifest.Read(filepath.Join(root, manifest.FileName))
		if err != nil {
			return fmt.Errorf("read project manifest: %w", err)
		}
		return writeNew(filepath.Join(root, ".github", "workflows", "ci.yml"), ciFile(ciGoVersion(m)))
	default:
		return fmt.Errorf("unknown feature %q", feature)
	}
}

func addCRUD(root, name string) error {
	if !resourceNamePattern.MatchString(name) {
		return fmt.Errorf("resource name %q must contain only letters, numbers, hyphens, or underscores", name)
	}
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return fmt.Errorf("read project manifest: %w", err)
	}
	framework := m.Framework
	if !containsString(supportedFrameworks, framework) {
		return fmt.Errorf("CRUD generation requires gin, fiber, or echo; run `gokub init --framework <name>` to set the existing project's framework")
	}
	database := normalizeDatabase(m.Database)
	pkg := featureName(name)
	typeName := exported(name)
	dir := filepath.Join(root, "internal", pkg)
	files := map[string]string{
		filepath.Join(dir, "model.go"):        kitModelFile(pkg, typeName, database),
		filepath.Join(dir, "repository.go"):   kitRepositoryFile(pkg, typeName, database),
		filepath.Join(dir, "service.go"):      kitServiceFile(m.Module, pkg, typeName),
		filepath.Join(dir, "service_test.go"): kitServiceTestFile(pkg, typeName),
		filepath.Join(dir, "handler.go"):      kitHandlerFile(m.Module, framework, pkg, typeName),
		filepath.Join(dir, "router.go"):       kitRouterFile(framework, pkg),
	}
	for path, content := range files {
		if err := writeNew(path, content); err != nil {
			return err
		}
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func writeNew(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeNewBytes(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return os.WriteFile(path, content, 0o644)
}

func moduleFile(module, goVersion string) string {
	return "module " + module + "\n\ngo " + goVersion + "\n"
}

func vscodeLaunchFile(m manifest.Manifest) string {
	return fmt.Sprintf(`{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "GOKUB: Run service",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/%s",
      "cwd": "${workspaceFolder}",
      "env": {
        "APP_ENV": "local",
        "PORT": "8080"
      }
    },
    {
      "name": "GOKUB: Debug current test",
      "type": "go",
      "request": "launch",
      "mode": "test",
      "program": "${fileDirname}"
    }
  ]
}
`, m.Name)
}

func vscodeTasksFile() string {
	return `{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "GOKUB: Test",
      "type": "shell",
      "command": "go test ./...",
      "group": { "kind": "test", "isDefault": true },
      "problemMatcher": "$go"
    },
    {
      "label": "GOKUB: Build",
      "type": "shell",
      "command": "go build ./...",
      "group": { "kind": "build", "isDefault": true },
      "problemMatcher": "$go"
    },
    {
      "label": "GOKUB: Quality Gate",
      "type": "shell",
      "command": "gokub score --fail-under ${input:gokubScoreMin}",
      "problemMatcher": []
    },
    {
      "label": "GOKUB: Architecture Check",
      "type": "shell",
      "command": "gokub graph --check",
      "problemMatcher": []
    }
  ],
  "inputs": [
    {
      "id": "gokubScoreMin",
      "type": "promptString",
      "description": "Minimum GOKUB project score",
      "default": "80"
    }
  ]
}
`
}

func jetbrainsRunFile(m manifest.Manifest) string {
	return fmt.Sprintf(`<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="GOKUB: Run service" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="%s" />
    <working_directory value="$PROJECT_DIR$" />
    <envs>
      <env name="APP_ENV" value="local" />
      <env name="PORT" value="8080" />
    </envs>
    <kind value="PACKAGE" />
    <package value="%s/cmd/%s" />
    <directory value="$PROJECT_DIR$" />
    <filePath value="$PROJECT_DIR$" />
    <method v="2" />
  </configuration>
</component>
`, m.Name, m.Module, m.Name)
}

func codexConfigFile() string {
	return `[mcp_servers.gokub]
command = "gokub"
args = ["mcp", "serve"]
`
}

func mcpConfigFile() string {
	return `{
  "mcpServers": {
    "gokub": {
      "command": "gokub",
      "args": ["mcp", "serve"]
    }
  }
}
`
}

func agentsFile(m manifest.Manifest) string {
	if m.Template == "existing" {
		return fmt.Sprintf(`# AGENTS.md

This existing Go project is initialized for GOKUB-assisted development.

## Project

- Name: %[1]s
- Module: %[2]s
- HTTP framework: %[3]s
- Database: %[4]s
- Messaging: %[5]s

## Workflow

- Read `+"`gokub.init`"+`, `+"`.gokub.yaml`"+`, and the relevant skill under `+"`.agents/skills`"+` before changing code.
- Preserve the repository's existing architecture and conventions.
- Use `+"`gokub add <feature>`"+` or GOKUB MCP tools for supported scaffolds, then integrate generated code with existing application wiring.
- Keep `+"`.gokub.yaml`"+` in sync with enabled capabilities.
- Run the repository's tests, `+"`go vet ./...`"+`, and `+"`gokub doctor`"+` before delivery.
- Never edit secrets or commit `+"`.env`"+` files.

`, m.Name, m.Module, m.Framework, m.Database, m.Messaging)
	}
	return fmt.Sprintf(`# AGENTS.md

You are working in a GOKUB-generated Go service.

## Project

- Name: %[1]s
- Module: %[2]s
- HTTP framework: %[3]s
- Database: %[4]s (gorm)
- Messaging: %[5]s

## Layout

- `+"`cmd/%[1]s-service/`"+` service entrypoint and wiring
- `+"`config/`"+` environment configuration (envconfig)
- `+"`internal/<domain>/`"+` model, repository, service, handler, router
- `+"`internal/app/`"+` composition and event contracts
- `+"`pkg/`"+` shared api, database, error, httpserver, middleware, utils

## Commands

`+"```bash"+`
go test ./...
go run ./cmd/%[1]s-service
gokub doctor
gokub score
gokub graph
`+"```"+`

## Workflow Rules

- Prefer GOKUB commands for generated structure changes.
- Use `+"`gokub add <feature>`"+` for capabilities such as auth, redis, kafka, rabbitmq, grpc, cron, email, websocket, or crud.
- Use `+"`gokub recipe add <name>`"+` for multi-capability installs.
- Keep `+"`.gokub.yaml`"+` in sync when generated capabilities change.
- Do not hand-edit generated wiring if a GOKUB command can perform the change.
- Run `+"`gokub doctor`"+` after structural changes.
- Run `+"`gokub score`"+` before delivery and review its recommendations.
- Use `+"`gokub graph`"+` to inspect package dependencies before changing boundaries.
- Prefer GOKUB MCP tools for project status, health checks, scoring, dependency graphs, features, and recipes when available.
- Use the relevant skill under `+"`.agents/skills`"+` for project, domain, and verification workflows.
- Never edit secrets or commit `+"`.env`"+` files.
- Never run an unreviewed GOKUB plugin; plugin execution is an explicit trust decision.

`, m.Name, m.Module, m.Framework, m.Database, m.Messaging)
}

func claudeFile(m manifest.Manifest) string {
	if m.Template == "existing" {
		return fmt.Sprintf(`# CLAUDE.md

This existing Go project is initialized for GOKUB-assisted development.

- Read `+"`gokub.init`"+`, `+"`.gokub.yaml`"+`, and the matching skill under `+"`.claude/skills`"+` before working.
- Preserve the existing architecture and use GOKUB commands or MCP tools for supported project changes.
- Integrate generated feature code with the repository's existing application wiring.
- Verify changes with the repository's tests, `+"`go vet ./...`"+`, and `+"`gokub doctor`"+`.

Project: %s
`, m.Name)
	}
	return fmt.Sprintf(`# CLAUDE.md

This repository is a GOKUB-generated Go service.

## Development

`+"```bash"+`
go test ./...
go run ./cmd/%s-service
gokub doctor
gokub score
gokub graph
`+"```"+`

## GOKUB Context

- Read `+"`.gokub.yaml`"+` before adding or removing capabilities.
- Use `+"`gokub help`"+` and `+"`gokub help <command>`"+` for CLI behavior.
- Use GOKUB for repeatable project changes instead of manual boilerplate edits.
- Keep each domain self-contained under `+"`internal/<domain>/`"+` (model, repository, service, handler, router) and shared adapters under `+"`pkg/`"+`.
- Use the GOKUB MCP tools exposed by `+"`gokub mcp serve`"+` for repeatable project changes.
- Load the matching GOKUB skill under `+"`.claude/skills`"+` for detailed workflows.

`, m.Name)
}

func ciFile(goVersion string) string {
	return fmt.Sprintf(`name: ci

on:
  push:
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "%s.x"
          cache: true
      - run: gofmt -l . | tee /tmp/gofmt.out && test ! -s /tmp/gofmt.out
      - run: go vet ./...
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go build ./...
`, goVersion)
}

func ciGoVersion(m manifest.Manifest) string {
	if m.GoVersion == "" {
		return goversion.Recommended
	}
	return m.GoVersion
}

func gitignore() string {
	return `.env
.env.*
!.env.example
dist/
*.log
tmp/
coverage.out
.idea/
.DS_Store
.gokub.yaml*.bak
`
}

func dockerignore() string {
	return `.git
.github
.idea
.vscode
.run
.env*
coverage.out
dist
tmp
*.log
`
}

func featureName(name string) string {
	name = strings.ReplaceAll(name, "-", "")
	name = strings.ReplaceAll(name, "_", "")
	return strings.ToLower(name)
}

func exported(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || r == ' ' })
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	if len(parts) == 0 {
		return "Resource"
	}
	return strings.Join(parts, "")
}
