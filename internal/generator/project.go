package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	gokub "github.com/gokub/gokub"
	"github.com/gokub/gokub/internal/agentskills"
	"github.com/gokub/gokub/internal/manifest"
	customtemplates "github.com/gokub/gokub/internal/templates"
)

var resourceNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func NewProject(root string, m manifest.Manifest) error {
	if err := manifest.Validate(m); err != nil {
		return err
	}
	if source, custom, err := customtemplates.Resolve(m.Template); err != nil {
		return err
	} else if custom {
		if err := customtemplates.Generate(source, root, m); err != nil {
			return err
		}
		return addProjectTooling(filepath.Join(root, m.Name), m)
	}
	if m.Template == "monolith" || m.Template == "microservices" {
		return newStandardProject(root, m)
	}
	target := filepath.Join(root, m.Name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("target %s already exists", target)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		return err
	}
	dirs := []string{
		"cmd/" + m.Name,
		"internal/config",
		"internal/http",
		"internal/health",
		"internal/domain",
		"internal/platform",
		"pkg",
		"configs",
		"deployments",
		"docs",
		"scripts",
		"tests",
		"migrations",
		".github/workflows",
		".vscode",
		".run",
		".codex",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(target, dir), 0o755); err != nil {
			return err
		}
	}
	files := map[string]string{
		"go.mod":                       moduleFile(m.Module),
		"cmd/" + m.Name + "/main.go":   mainFile(m),
		"internal/config/config.go":    configFile(),
		"internal/health/health.go":    healthFile(),
		"internal/http/server.go":      serverFile(m),
		"internal/http/middleware.go":  middlewareFile(),
		"internal/http/server_test.go": serverTestFile(m),
		"README.md":                    readmeFile(m),
		"Makefile":                     makefile(m),
		"Dockerfile":                   dockerfile(m),
		"docker-compose.yml":           composeFile(m),
		".gitignore":                   gitignore(),
		".dockerignore":                dockerignore(),
		".env.example":                 envExample(m),
		"AGENTS.md":                    agentsFile(m),
		"CLAUDE.md":                    claudeFile(m),
		".github/workflows/ci.yml":     ciFile(),
		".vscode/launch.json":          vscodeLaunchFile(m),
		".vscode/tasks.json":           vscodeTasksFile(),
		".run/GOKUB.run.xml":           jetbrainsRunFile(m),
		".codex/config.toml":           codexConfigFile(),
		".mcp.json":                    mcpConfigFile(),
	}
	for name, content := range files {
		if err := writeNew(filepath.Join(target, name), content); err != nil {
			return err
		}
	}
	if err := addProjectTooling(target, m); err != nil {
		return err
	}
	return manifest.Write(filepath.Join(target, manifest.FileName), m)
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
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return nil, err
		}
		written = append(written, name)
	}
	installed, err := agentskills.Install(root, provider, false)
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
	case "auth":
		return writeNew(filepath.Join(root, "internal", "auth", "auth.go"), packageFile("auth", "Service owns authentication workflows."))
	case "kafka", "rabbitmq", "nats", "redis", "postgres", "mongodb", "grpc", "cron", "email", "websocket", "otel", "outbox":
		return writeNew(filepath.Join(root, "internal", feature, feature+".go"), packageFile(featureName(feature), fmt.Sprintf("%s capability scaffold.", feature)))
	case "docker":
		return writeNew(filepath.Join(root, "deployments", "docker.md"), "# Docker\n\nDocker support is enabled for this project.\n")
	case "github-actions":
		return writeNew(filepath.Join(root, ".github", "workflows", "ci.yml"), ciFile())
	default:
		return fmt.Errorf("unknown feature %q", feature)
	}
}

func addCRUD(root, name string) error {
	if !resourceNamePattern.MatchString(name) {
		return fmt.Errorf("resource name %q must contain only letters, numbers, hyphens, or underscores", name)
	}
	pkg := featureName(name)
	dir := filepath.Join(root, "internal", pkg)
	files := map[string]string{
		filepath.Join(dir, "model.go"):      fmt.Sprintf("package %s\n\ntype %s struct {\n\tID string `json:\"id\"`\n}\n", pkg, exported(name)),
		filepath.Join(dir, "repository.go"): fmt.Sprintf("package %s\n\ntype Repository interface {\n\tFindAll() ([]%s, error)\n}\n", pkg, exported(name)),
		filepath.Join(dir, "service.go"):    fmt.Sprintf("package %s\n\ntype Service struct {\n\trepo Repository\n}\n\nfunc NewService(repo Repository) Service {\n\treturn Service{repo: repo}\n}\n", pkg),
		filepath.Join(dir, "handler.go"):    fmt.Sprintf("package %s\n\n// Handler exposes %s HTTP endpoints.\ntype Handler struct {\n\tservice Service\n}\n", pkg, name),
	}
	for path, content := range files {
		if err := writeNew(path, content); err != nil {
			return err
		}
	}
	return nil
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

func moduleFile(module string) string {
	return "module " + module + "\n\ngo 1.24\n"
}

func mainFile(m manifest.Manifest) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"%s/internal/config"
	apphttp "%s/internal/http"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("invalid configuration", "error", err)
		return 1
	}
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		return healthcheck(cfg.Port)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel()}))
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           apphttp.NewRouterWithLogger(logger),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "address", server.Addr, "environment", cfg.Environment)
		serverErrors <- server.ListenAndServe()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			return 1
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		return 1
	}
	logger.Info("http server stopped")
	return 0
}

func healthcheck(port string) int {
	client := http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://127.0.0.1:" + port + "/health/ready")
	if err != nil {
		return 1
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
`, m.Module, m.Module)
}

func configFile() string {
	return `package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Environment       string
	Port              string
	LogLevelName      string
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	ShutdownTimeout   time.Duration
}

func Load() (Config, error) {
	cfg := Config{
		Environment:       getenv("APP_ENV", "local"),
		Port:              getenv("PORT", "8080"),
		LogLevelName:      getenv("LOG_LEVEL", "info"),
		ReadHeaderTimeout: duration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:       duration("HTTP_READ_TIMEOUT", 15*time.Second),
		WriteTimeout:      duration("HTTP_WRITE_TIMEOUT", 30*time.Second),
		IdleTimeout:       duration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		ShutdownTimeout:   duration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),
	}
	port, err := strconv.Atoi(cfg.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be between 1 and 65535")
	}
	if _, ok := parseLogLevel(cfg.LogLevelName); !ok {
		return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
	return cfg, nil
}

func (c Config) LogLevel() slog.Level {
	level, _ := parseLogLevel(c.LogLevelName)
	return level
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func duration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func parseLogLevel(value string) (slog.Level, bool) {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug, true
	case "info":
		return slog.LevelInfo, true
	case "warn":
		return slog.LevelWarn, true
	case "error":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}
`
}

func healthFile() string {
	return `package health

type Status struct {
	Status string ` + "`json:\"status\"`" + `
}

func Check() Status {
	return Status{Status: "ok"}
}
`
}

func serverFile(m manifest.Manifest) string {
	return fmt.Sprintf(`package http

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"%s/internal/health"
)

func NewRouter() http.Handler {
	return NewRouterWithLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func NewRouterWithLogger(logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	healthHandler := func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, health.Check())
	}
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /health/live", healthHandler)
	mux.HandleFunc("GET /health/ready", healthHandler)

	return middleware(logger, mux)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("encode response", "error", err)
	}
}
`, m.Module)
}

func middlewareFile() string {
	return `package http

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (w *responseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *responseWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func middleware(logger *slog.Logger, next http.Handler) http.Handler {
	return accessLog(logger, recoverPanic(logger, requestID(secureHeaders(next))))
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			var value [12]byte
			if _, err := rand.Read(value[:]); err == nil {
				id = hex.EncodeToString(value[:])
			}
		}
		if id != "" {
			w.Header().Set("X-Request-ID", id)
		}
		next.ServeHTTP(w, r)
	})
}

func accessLog(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		logger.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.status,
			"duration_ms", time.Since(started).Milliseconds(),
			"request_id", wrapped.Header().Get("X-Request-ID"),
		)
	})
}

func recoverPanic(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("panic recovered", "panic", recovered, "stack", string(debug.Stack()))
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}
`
}

func serverTestFile(_ manifest.Manifest) string {
	return `package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	NewRouter().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", response.Code)
	}
	if got := response.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Fatalf("expected secure headers, got X-Frame-Options=%q", got)
	}
	if got := response.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected request ID header")
	}
}
`
}

func readmeFile(m manifest.Manifest) string {
	return fmt.Sprintf(`# %s

![GOKUB](docs/gokub_logo.png)

Generated by GOKUB.

## Architecture

`+"```text"+`
cmd/%s/             application entrypoint and lifecycle
internal/config/    validated environment configuration
internal/http/      transport, middleware, and route composition
internal/health/    liveness and readiness contracts
internal/<domain>/  domain model, repository, service, and handler
internal/platform/  infrastructure adapters
`+"```"+`

## Development

`+"```bash"+`
make test
make run
`+"```"+`

Copy `+"`.env.example`"+` to `+"`.env`"+` when your local runner supports env files.

## Run and Debug

- VS Code: open Run and Debug, then select `+"`GOKUB: Run service`"+`.
- GoLand or IntelliJ IDEA: select the shared `+"`GOKUB: Run service`"+` configuration.
- Other IDEs: run package `+"`./cmd/%s`"+` with `+"`APP_ENV=local`"+` and `+"`PORT=8080`"+`.

## Run

`+"```bash"+`
go run ./cmd/%s
`+"```"+`

## Health

`+"```bash"+`
curl http://localhost:8080/health
`+"```"+`

## AI Agents and MCP

Codex reads `+"`.codex/config.toml`"+`; Claude and compatible clients can read
`+"`.mcp.json`"+`. Both launch `+"`gokub mcp serve`"+` and expose typed project tools.
`, m.Name, m.Name, m.Name, m.Name)
}

func makefile(m manifest.Manifest) string {
	return fmt.Sprintf(`.PHONY: run test build fmt doctor

run:
	go run ./cmd/%s

test:
	go test ./...

build:
	go build ./...

fmt:
	gofmt -w $$(find cmd internal pkg -name '*.go')

doctor:
	gokub doctor
`, m.Name)
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
	return fmt.Sprintf(`# AGENTS.md

You are working in a GOKUB-generated Go service.

## Project

- Name: %s
- Module: %s
- Template: %s
- Style: %s
- Architecture: %s

## Commands

`+"```bash"+`
go test ./...
go run ./cmd/%s
gokub doctor
`+"```"+`

## Workflow Rules

- Prefer GOKUB commands for generated structure changes.
- Use `+"`gokub add <feature>`"+` for capabilities such as auth, redis, kafka, rabbitmq, grpc, cron, email, websocket, or crud.
- Use `+"`gokub recipe add <name>`"+` for multi-capability installs.
- Keep `+"`.gokub.yaml`"+` in sync when generated capabilities change.
- Do not hand-edit generated wiring if a GOKUB command can perform the change.
- Run `+"`gokub doctor`"+` after structural changes.
- Prefer GOKUB MCP tools for project status, health checks, features, and recipes when available.
- Use the relevant skill under `+"`.agents/skills`"+` for project, domain, and verification workflows.
- Never edit secrets or commit `+"`.env`"+` files.

`, m.Name, m.Module, m.Template, m.Style, m.Architecture, m.Name)
}

func claudeFile(m manifest.Manifest) string {
	return fmt.Sprintf(`# CLAUDE.md

This repository is a GOKUB-generated Go service.

## Development

`+"```bash"+`
go test ./...
go run ./cmd/%s
gokub doctor
`+"```"+`

## GOKUB Context

- Read `+"`.gokub.yaml`"+` before adding or removing capabilities.
- Use `+"`gokub help`"+` and `+"`gokub help <command>`"+` for CLI behavior.
- Use GOKUB for repeatable project changes instead of manual boilerplate edits.
- Preserve clean architecture boundaries under `+"`internal/`"+`.
- Use the GOKUB MCP tools exposed by `+"`gokub mcp serve`"+` for repeatable project changes.
- Load the matching GOKUB skill under `+"`.claude/skills`"+` for detailed workflows.

`, m.Name)
}

func dockerfile(m manifest.Manifest) string {
	return fmt.Sprintf(`FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/%s

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD ["/app", "healthcheck"]
ENTRYPOINT ["/app"]
`, m.Name)
}

func composeFile(m manifest.Manifest) string {
	return fmt.Sprintf(`services:
  %s:
    build: .
    ports:
      - "8080:8080"
    environment:
      APP_ENV: local
      LOG_LEVEL: debug
    healthcheck:
      test: ["CMD", "/app", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
`, m.Name)
}

func ciFile() string {
	return `name: ci

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
          go-version: "1.24.x"
          cache: true
      - run: gofmt -l . | tee /tmp/gofmt.out && test ! -s /tmp/gofmt.out
      - run: go vet ./...
      - run: go test -race -coverprofile=coverage.out ./...
      - run: go build ./...
`
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

func envExample(m manifest.Manifest) string {
	return fmt.Sprintf(`# %s
APP_ENV=local
PORT=8080
LOG_LEVEL=debug

HTTP_READ_HEADER_TIMEOUT=5s
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=30s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=15s
`, m.Name)
}

func packageFile(pkg, comment string) string {
	return fmt.Sprintf("package %s\n\n// %s\ntype Config struct{}\n", pkg, comment)
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
