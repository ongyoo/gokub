package generator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	gokub "github.com/ongyoo/gokub"
	"github.com/ongyoo/gokub/internal/agentskills"
	"github.com/ongyoo/gokub/internal/manifest"
)

func newStandardProject(root string, m manifest.Manifest) error {
	target := filepath.Join(root, m.Name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("target %s already exists", target)
	} else if !os.IsNotExist(err) {
		return err
	}
	commands := []string{m.Name}
	if m.Template == "microservices" {
		m.Style = "microservices"
		commands = []string{"gateway", "example-service"}
	} else {
		m.Style = "monolith"
	}
	dirs := []string{
		"internal/config", "internal/health", "internal/http", "internal/domain/example",
		"internal/platform/postgres", "internal/platform/redis", "internal/platform/validation",
		"internal/platform/observability", "internal/platform/messaging", "internal/platform/auth",
		"internal/platform/mongodb", "internal/platform/messaging/rabbitmq",
		"internal/platform/messaging/nats", "internal/platform/messaging/kafka",
		"internal/platform/httpserver/gin", "internal/platform/httpserver/fiber",
		"pkg/contracts", "configs", "deployments", "docs", "scripts", "tests", "migrations",
		".github/workflows", ".vscode", ".run", ".codex", "tools",
	}
	for _, command := range commands {
		dirs = append(dirs, filepath.Join("cmd", command))
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(target, dir), 0o755); err != nil {
			return err
		}
	}
	files := map[string]string{
		"go.mod":                                           standardModuleFile(m.Module, m.GoVersion),
		"internal/config/config.go":                        configFile(),
		"internal/health/health.go":                        healthFile(),
		"internal/http/server.go":                          serverFile(m),
		"internal/http/middleware.go":                      middlewareFile(),
		"internal/http/server_test.go":                     serverTestFile(m),
		"internal/domain/example/model.go":                 standardModelFile(),
		"internal/domain/example/repository.go":            standardRepositoryFile(),
		"internal/domain/example/service.go":               standardServiceFile(),
		"internal/domain/example/service_test.go":          standardServiceTestFile(),
		"internal/platform/postgres/postgres.go":           postgresAdapterFile(),
		"internal/platform/redis/redis.go":                 redisAdapterFile(),
		"internal/platform/mongodb/mongodb.go":             mongodbAdapterFile(),
		"internal/platform/validation/validation.go":       validationAdapterFile(),
		"internal/platform/observability/tracing.go":       tracingAdapterFile(),
		"internal/platform/messaging/messaging.go":         messagingAdapterFile(),
		"internal/platform/messaging/rabbitmq/rabbitmq.go": rabbitMQAdapterFile(),
		"internal/platform/messaging/nats/nats.go":         natsAdapterFile(),
		"internal/platform/messaging/kafka/kafka.go":       kafkaAdapterFile(),
		"internal/platform/auth/token.go":                  authAdapterFile(),
		"internal/platform/httpserver/gin/gin.go":          ginAdapterFile(),
		"internal/platform/httpserver/fiber/fiber.go":      fiberAdapterFile(),
		"pkg/contracts/event.go":                           contractsFile(),
		"tools/dependencies.go":                            dependencyPinsFile(),
		"README.md":                                        standardReadmeFile(m, commands),
		"Makefile":                                         standardMakefile(commands[0]),
		"Dockerfile":                                       standardDockerfile(commands[0], m.GoVersion),
		"docker-compose.yml":                               standardComposeFile(m, commands),
		".gitignore":                                       gitignore(),
		".dockerignore":                                    dockerignore(),
		".env.example":                                     standardEnvFile(m),
		"AGENTS.md":                                        agentsFile(m),
		"CLAUDE.md":                                        claudeFile(m),
		".github/workflows/ci.yml":                         ciFile(ciGoVersion(m)),
		".vscode/launch.json":                              standardVSCodeFile(commands),
		".vscode/tasks.json":                               vscodeTasksFile(),
		".run/GOKUB.run.xml":                               standardJetBrainsFile(m, commands[0]),
		".codex/config.toml":                               codexConfigFile(),
		".mcp.json":                                        mcpConfigFile(),
	}
	for _, command := range commands {
		files[filepath.Join("cmd", command, "main.go")] = mainFile(m)
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
	if _, err := agentskills.Install(target, "all", false); err != nil {
		return err
	}
	if err := manifest.Write(filepath.Join(target, manifest.FileName), m); err != nil {
		return err
	}
	if os.Getenv("GOKUB_SKIP_INSTALL") != "1" {
		command := exec.Command("go", "mod", "tidy")
		command.Dir = target
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("install core dependencies: %w", err)
		}
	}
	return nil
}

func standardModuleFile(module, goVersion string) string {
	return fmt.Sprintf(`module %s

go %s

require (
	github.com/caarlos0/env/v11 v11.4.1
	github.com/gin-gonic/gin v1.12.0
	github.com/go-playground/validator/v10 v10.30.3
	github.com/gofiber/fiber/v3 v3.4.0
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.10.0
	github.com/nats-io/nats.go v1.52.0
	github.com/prometheus/client_golang v1.23.2
	github.com/rabbitmq/amqp091-go v1.12.0
	github.com/redis/go-redis/v9 v9.21.0
	github.com/stretchr/testify v1.11.1
	github.com/twmb/franz-go v1.21.5
	go.mongodb.org/mongo-driver/v2 v2.8.0
	go.opentelemetry.io/otel v1.44.0
)
`, module, goVersion)
}

func standardModelFile() string {
	return `package example

import (
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID        uuid.UUID ` + "`json:\"id\"`" + `
	Name      string    ` + "`json:\"name\" validate:\"required,min=2,max=120\"`" + `
	CreatedAt time.Time ` + "`json:\"createdAt\"`" + `
	UpdatedAt time.Time ` + "`json:\"updatedAt\"`" + `
}
`
}

func standardRepositoryFile() string {
	return `package example

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	List(context.Context) ([]Item, error)
	Get(context.Context, uuid.UUID) (Item, error)
	Create(context.Context, Item) error
	Update(context.Context, Item) error
	Delete(context.Context, uuid.UUID) error
}
`
}

func standardServiceFile() string {
	return `package example

import (
	"context"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Service struct {
	repository Repository
	validate   *validator.Validate
}

func NewService(repository Repository, validate *validator.Validate) Service {
	return Service{repository: repository, validate: validate}
}

func (s Service) Create(ctx context.Context, item Item) (Item, error) {
	if err := s.validate.Struct(item); err != nil {
		return Item{}, fmt.Errorf("validate item: %w", err)
	}
	item.ID = uuid.New()
	item.CreatedAt = time.Now().UTC()
	item.UpdatedAt = item.CreatedAt
	if err := s.repository.Create(ctx, item); err != nil {
		return Item{}, fmt.Errorf("create item: %w", err)
	}
	return item, nil
}
`
}

func standardServiceTestFile() string {
	return `package example

import (
	"context"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type repositoryStub struct{ created Item }
func (r *repositoryStub) List(context.Context) ([]Item, error) { return nil, nil }
func (r *repositoryStub) Get(context.Context, uuid.UUID) (Item, error) { return Item{}, nil }
func (r *repositoryStub) Create(_ context.Context, item Item) error { r.created = item; return nil }
func (r *repositoryStub) Update(context.Context, Item) error { return nil }
func (r *repositoryStub) Delete(context.Context, uuid.UUID) error { return nil }

func TestServiceCreate(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository, validator.New())
	created, err := service.Create(context.Background(), Item{Name: "example"})
	if err != nil { t.Fatal(err) }
	if created.ID == uuid.Nil || repository.created.ID != created.ID { t.Fatal("item was not persisted") }
}
`
}

func postgresAdapterFile() string {
	return `package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Open(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil { return nil, fmt.Errorf("parse postgres config: %w", err) }
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil { return nil, fmt.Errorf("open postgres: %w", err) }
	if err := pool.Ping(ctx); err != nil { pool.Close(); return nil, fmt.Errorf("ping postgres: %w", err) }
	return pool, nil
}
`
}

func redisAdapterFile() string {
	return `package redis

import (
	"context"
	"fmt"

	goredis "github.com/redis/go-redis/v9"
)

func Open(ctx context.Context, address string) (*goredis.Client, error) {
	client := goredis.NewClient(&goredis.Options{Addr: address})
	if err := client.Ping(ctx).Err(); err != nil { _ = client.Close(); return nil, fmt.Errorf("ping redis: %w", err) }
	return client, nil
}
`
}

func mongodbAdapterFile() string {
	return `package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Open(ctx context.Context, uri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil { return nil, fmt.Errorf("open mongodb: %w", err) }
	if err := client.Ping(ctx, nil); err != nil { _ = client.Disconnect(ctx); return nil, fmt.Errorf("ping mongodb: %w", err) }
	return client, nil
}
`
}

func validationAdapterFile() string {
	return `package validation

import "github.com/go-playground/validator/v10"

func New() *validator.Validate { return validator.New(validator.WithRequiredStructEnabled()) }
`
}

func tracingAdapterFile() string {
	return `package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func Tracer(service string) trace.Tracer { return otel.Tracer(service) }
`
}

func messagingAdapterFile() string {
	return `package messaging

import "context"

type Publisher interface { Publish(context.Context, string, []byte) error }
type Consumer interface { Subscribe(context.Context, string, func(context.Context, []byte) error) error }
`
}

func rabbitMQAdapterFile() string {
	return `package rabbitmq

import (
	"context"
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct { connection *amqp.Connection; channel *amqp.Channel }

func Open(url string) (*Publisher, error) {
	connection, err := amqp.Dial(url)
	if err != nil { return nil, fmt.Errorf("open rabbitmq: %w", err) }
	channel, err := connection.Channel()
	if err != nil { _ = connection.Close(); return nil, fmt.Errorf("open rabbitmq channel: %w", err) }
	return &Publisher{connection: connection, channel: channel}, nil
}

func (p *Publisher) Publish(ctx context.Context, topic string, body []byte) error {
	return p.channel.PublishWithContext(ctx, "", topic, false, false, amqp.Publishing{ContentType: "application/json", Body: body})
}

func (p *Publisher) Close() error { _ = p.channel.Close(); return p.connection.Close() }
`
}

func natsAdapterFile() string {
	return `package nats

import (
	"context"
	"fmt"

	gonats "github.com/nats-io/nats.go"
)

type Publisher struct { connection *gonats.Conn }

func Open(url string) (*Publisher, error) {
	connection, err := gonats.Connect(url)
	if err != nil { return nil, fmt.Errorf("open nats: %w", err) }
	return &Publisher{connection: connection}, nil
}

func (p *Publisher) Publish(_ context.Context, topic string, body []byte) error { return p.connection.Publish(topic, body) }
func (p *Publisher) Close() { p.connection.Close() }
`
}

func kafkaAdapterFile() string {
	return `package kafka

import (
	"context"
	"fmt"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Publisher struct { client *kgo.Client }

func Open(ctx context.Context, brokers ...string) (*Publisher, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil { return nil, fmt.Errorf("open kafka: %w", err) }
	if err := client.Ping(ctx); err != nil { client.Close(); return nil, fmt.Errorf("ping kafka: %w", err) }
	return &Publisher{client: client}, nil
}

func (p *Publisher) Publish(ctx context.Context, topic string, body []byte) error {
	return p.client.ProduceSync(ctx, &kgo.Record{Topic: topic, Value: body}).FirstErr()
}

func (p *Publisher) Close() { p.client.Close() }
`
}

func authAdapterFile() string {
	return `package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func Sign(subject string, secret []byte, ttl time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{Subject: subject, ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), IssuedAt: jwt.NewNumericDate(time.Now())}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}
`
}

func contractsFile() string {
	return `package contracts

import "time"

type Event[T any] struct {
	ID         string    ` + "`json:\"id\"`" + `
	Type       string    ` + "`json:\"type\"`" + `
	OccurredAt time.Time ` + "`json:\"occurredAt\"`" + `
	Data       T         ` + "`json:\"data\"`" + `
}
`
}

func dependencyPinsFile() string {
	return `//go:build tools

package tools

import (
	_ "github.com/caarlos0/env/v11"
	_ "github.com/gin-gonic/gin"
	_ "github.com/gofiber/fiber/v3"
	_ "github.com/golang-migrate/migrate/v4"
	_ "github.com/nats-io/nats.go"
	_ "github.com/prometheus/client_golang/prometheus"
	_ "github.com/rabbitmq/amqp091-go"
	_ "github.com/stretchr/testify/assert"
	_ "github.com/twmb/franz-go/pkg/kgo"
	_ "go.mongodb.org/mongo-driver/v2/mongo"
)
`
}

func ginAdapterFile() string {
	return `package gin

import "github.com/gin-gonic/gin"

func New(mode string) *gin.Engine {
	gin.SetMode(mode)
	router := gin.New()
	router.Use(gin.Recovery())
	return router
}
`
}

func fiberAdapterFile() string {
	return `package fiber

import "github.com/gofiber/fiber/v3"

func New(service string) *fiber.App {
	return fiber.New(fiber.Config{AppName: service})
}
`
}

func standardReadmeFile(m manifest.Manifest, commands []string) string {
	return fmt.Sprintf(`# %s

![GOKUB](docs/gokub_logo.png)

Production-ready %s project generated by GOKUB from audited Go application patterns.

## Start

`+"```bash"+`
cp .env.example .env
go mod download
make test
make run
`+"```"+`

## Entrypoints

%s

## Structure

`+"```text"+`
cmd/                         deployable entrypoints
internal/domain/             business rules and repository ports
internal/http/               HTTP routing and middleware
internal/platform/           database, cache, auth, messaging, and telemetry adapters
pkg/contracts/               versionable cross-service event contracts
migrations/                  schema migrations
deployments/                 deployment configuration
`+"```"+`

## Core Libraries

Gin, Fiber, pgx, MongoDB, Redis, Kafka, RabbitMQ, NATS, JWT, validator,
OpenTelemetry, Prometheus, golang-migrate, UUID, env, and Testify are pinned in
`+"`go.mod`"+`. The generated adapters compile and are ready to wire in the
composition root.

## Health

`+"```bash"+`
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
`+"```"+`

## AI Collaboration

Codex uses `+"`.codex/config.toml`"+`, Claude-compatible clients use
`+"`.mcp.json`"+`, and both can call the typed tools from `+"`gokub mcp serve`"+`.
`, m.Name, m.Style, commandList(commands))
}

func commandList(commands []string) string {
	result := ""
	for _, command := range commands {
		result += fmt.Sprintf("- `go run ./cmd/%s`\n", command)
	}
	return result
}

func standardMakefile(command string) string {
	return fmt.Sprintf(`SCORE_MIN ?= 80

.PHONY: run test build fmt vet tidy doctor score graph graph-check upgrade

run:
	go run ./cmd/%s

test:
	go test -race -cover ./...

build:
	go build ./...

fmt:
	gofmt -w $$(find cmd internal pkg -name '*.go')

vet:
	go vet ./...

tidy:
	go mod tidy

doctor:
	gokub doctor

score:
	gokub score --fail-under $(SCORE_MIN)

graph:
	gokub graph

graph-check:
	gokub graph --check

upgrade:
	gokub upgrade
`, command)
}

func standardDockerfile(command, goVersion string) string {
	return fmt.Sprintf(`FROM golang:%s-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG COMMAND=%s
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/${COMMAND}

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD ["/app", "healthcheck"]
ENTRYPOINT ["/app"]
`, goVersion, command)
}

func standardComposeFile(m manifest.Manifest, commands []string) string {
	if len(commands) == 1 {
		return fmt.Sprintf(`services:
  %s:
    build: .
    env_file: .env
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app"]
      interval: 5s
      timeout: 3s
      retries: 10
  redis:
    image: redis:8-alpine
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 10
`, m.Name)
	}
	return `services:
  gateway:
    build:
      context: .
      args:
        COMMAND: gateway
    env_file: .env
    environment:
      PORT: "8080"
    ports:
      - "8080:8080"
  example-service:
    build:
      context: .
      args:
        COMMAND: example-service
    env_file: .env
    environment:
      PORT: "8081"
    ports:
      - "8081:8081"
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
  redis:
    image: redis:8-alpine
`
}

func standardEnvFile(m manifest.Manifest) string {
	return fmt.Sprintf(`# %s
APP_ENV=local
PORT=8080
LOG_LEVEL=debug
HTTP_READ_HEADER_TIMEOUT=5s
HTTP_READ_TIMEOUT=15s
HTTP_WRITE_TIMEOUT=30s
HTTP_IDLE_TIMEOUT=60s
HTTP_SHUTDOWN_TIMEOUT=15s

POSTGRES_URL=postgres://app:app@localhost:5432/app?sslmode=disable
MONGODB_URL=mongodb://localhost:27017
REDIS_ADDR=localhost:6379
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
NATS_URL=nats://localhost:4222
JWT_SECRET=replace-with-at-least-32-random-bytes
`, m.Name)
}

func standardVSCodeFile(commands []string) string {
	configurations := ""
	for index, command := range commands {
		if index > 0 {
			configurations += ",\n"
		}
		configurations += fmt.Sprintf(`    {
      "name": "GOKUB: Debug %s",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/%s",
      "cwd": "${workspaceFolder}",
      "envFile": "${workspaceFolder}/.env"
    }`, command, command)
	}
	return fmt.Sprintf(`{
  "version": "0.2.0",
  "configurations": [
%s
  ]
}
`, configurations)
}

func standardJetBrainsFile(m manifest.Manifest, command string) string {
	return fmt.Sprintf(`<component name="ProjectRunConfigurationManager">
  <configuration default="false" name="GOKUB: Run %s" type="GoApplicationRunConfiguration" factoryName="Go Application">
    <module name="%s" />
    <working_directory value="$PROJECT_DIR$" />
    <envs>
      <env name="APP_ENV" value="local" />
      <env name="PORT" value="8080" />
    </envs>
    <kind value="PACKAGE" />
    <package value="%s/cmd/%s" />
    <method v="2" />
  </configuration>
</component>
`, command, m.Name, m.Module, command)
}
