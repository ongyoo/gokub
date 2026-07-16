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

// tick returns a raw backtick for embedding Go struct tags in template strings.
const tick = "`"

// supportedFrameworks lists the HTTP frameworks the kit generator can emit.
var supportedFrameworks = []string{"gin", "fiber", "echo"}

func normalizeFramework(framework string) string {
	for _, candidate := range supportedFrameworks {
		if framework == candidate {
			return framework
		}
	}
	return "gin"
}

// serviceName returns the cmd entrypoint directory for the project, matching the
// roomkub-api-v2 convention of cmd/<name>-service.
func serviceName(name string) string { return name + "-service" }

// newKitProject generates a production-ready service using the roomkub-api-v2
// layout: a flat internal/<domain> module (handler, service, repository, router,
// model), shared pkg/* packages, an internal/app composition layer, and a single
// cmd/<name>-service entrypoint wired to the selected HTTP framework.
func newKitProject(root string, m manifest.Manifest) error {
	m.Framework = normalizeFramework(m.Framework)
	m.Style = "monolith"
	domain := "example"
	typeName := "Example"
	service := serviceName(m.Name)

	target := filepath.Join(root, m.Name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("target %s already exists", target)
	} else if !os.IsNotExist(err) {
		return err
	}

	dirs := []string{
		"cmd/" + service,
		"config",
		"internal/app/events",
		"internal/" + domain,
		"pkg/api",
		"pkg/database/postgresql",
		"pkg/error",
		"pkg/httpserver/" + m.Framework,
		"pkg/middleware/" + m.Framework,
		"pkg/utils",
		"tests",
		"docs",
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
		"go.mod":                                           moduleFile(m.Module, m.GoVersion),
		"config/config.go":                                 kitConfigFile(),
		"internal/app/app.go":                              kitAppFile(service),
		"internal/app/events/publisher.go":                 kitEventsPublisherFile(),
		"internal/app/events/contracts.go":                 kitEventsContractsFile(),
		"internal/" + domain + "/model.go":                 kitModelFile(domain, typeName),
		"internal/" + domain + "/repository.go":            kitRepositoryFile(domain, typeName),
		"internal/" + domain + "/service.go":               kitServiceFile(m.Module, domain, typeName),
		"internal/" + domain + "/service_test.go":          kitServiceTestFile(domain, typeName),
		"internal/" + domain + "/handler.go":               kitHandlerFile(m.Module, m.Framework, domain, typeName),
		"internal/" + domain + "/router.go":                kitRouterFile(m.Framework, domain),
		"pkg/api/response.go":                              kitAPIResponseFile(),
		"pkg/error/error.go":                               kitErrorFile(),
		"pkg/database/postgresql/postgres.go":              kitPostgresFile(),
		"pkg/utils/utils.go":                               kitUtilsFile(),
		"pkg/httpserver/" + m.Framework + "/http.go":       kitHTTPServerFile(m.Framework),
		"pkg/middleware/" + m.Framework + "/middleware.go": kitMiddlewareFile(m.Framework),
		"cmd/" + service + "/main.go":                      kitMainFile(m.Module, m.Framework, domain),
		"tests/README.md":                                  "# Tests\n\nIntegration and acceptance tests live here. Unit tests sit next to the code they cover under `internal/`.\n",
		"README.md":                                        kitReadmeFile(m, service, domain),
		"Makefile":                                         kitMakefile(service),
		"Dockerfile":                                       kitDockerfile(service, m.GoVersion),
		"docker-compose.yml":                               kitComposeFile(m.Name),
		".env.example":                                     kitEnvFile(m.Name, m.Messaging),
		".env":                                             kitEnvFile(m.Name, m.Messaging),
		".gitignore":                                       gitignore(),
		".dockerignore":                                    dockerignore(),
		".github/workflows/ci.yml":                         ciFile(ciGoVersion(m)),
		".vscode/launch.json":                              kitVSCodeFile(service),
		".vscode/tasks.json":                               vscodeTasksFile(),
		".run/GOKUB.run.xml":                               kitJetBrainsFile(m, service),
	}
	for name, content := range agentFilesFor(m) {
		files[name] = content
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
	if provider := agentProvider(m); provider != "none" {
		if _, err := agentskills.Install(target, provider, false); err != nil {
			return err
		}
	}
	if err := manifest.Write(filepath.Join(target, manifest.FileName), m); err != nil {
		return err
	}
	if isMessagingProvider(m.Messaging) {
		if err := wireMessaging(target, m.Messaging); err != nil {
			return err
		}
	}
	if err := TidyModule(target); err != nil {
		return fmt.Errorf("resolve dependencies: %w", err)
	}
	return nil
}

// TidyModule runs `go mod tidy` in root unless GOKUB_SKIP_INSTALL=1.
func TidyModule(root string) error {
	if os.Getenv("GOKUB_SKIP_INSTALL") == "1" {
		return nil
	}
	command := exec.Command("go", "mod", "tidy")
	command.Dir = root
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

// agentProvider returns the AI-agent provider to install, defaulting to all.
func agentProvider(m manifest.Manifest) string {
	if m.Agents == "" {
		return "all"
	}
	return m.Agents
}

// agentFilesFor returns the top-level AI guidance files to write for the chosen
// provider. Skill files are installed separately via agentskills.
func agentFilesFor(m manifest.Manifest) map[string]string {
	files := map[string]string{}
	switch agentProvider(m) {
	case "none":
		// no AI guidance files
	case "codex":
		files["AGENTS.md"] = agentsFile(m)
		files[".codex/config.toml"] = codexConfigFile()
	case "claude":
		files["CLAUDE.md"] = claudeFile(m)
		files[".mcp.json"] = mcpConfigFile()
	case "copilot", "gemini":
		// instruction and skill files are installed via agentskills
	default: // all
		files["AGENTS.md"] = agentsFile(m)
		files["CLAUDE.md"] = claudeFile(m)
		files[".codex/config.toml"] = codexConfigFile()
		files[".mcp.json"] = mcpConfigFile()
	}
	return files
}

func kitConfigFile() string {
	return `package config

import (
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
)

// Config holds runtime configuration loaded from the environment.
type Config struct {
	AppEnv      string ` + tick + `envconfig:"APP_ENV" default:"local"` + tick + `
	Port        string ` + tick + `envconfig:"PORT" default:"8080"` + tick + `
	LogLevel    string ` + tick + `envconfig:"LOG_LEVEL" default:"debug"` + tick + `
	DatabaseURL string ` + tick + `envconfig:"DATABASE_URL" default:"postgres://app:app@localhost:5432/app?sslmode=disable"` + tick + `
}

// Load reads configuration from a local .env file (when present) and the
// environment, then configures logging.
func Load() Config {
	// Best-effort: load .env for local development. Real environment variables
	// always take precedence and a missing file is not an error.
	_ = godotenv.Load()
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		logrus.Fatalf("load config: %v", err)
	}
	if level, err := logrus.ParseLevel(cfg.LogLevel); err == nil {
		logrus.SetLevel(level)
	}
	return cfg
}
`
}

func kitAppFile(service string) string {
	return fmt.Sprintf(`package app

// Name identifies the service in logs, metrics, and traces.
const Name = %q
`, service)
}

func kitEventsPublisherFile() string {
	return `package events

import "context"

// Publisher publishes domain events to a message bus. The default implementation
// is a no-op so services run without a broker during local development.
type Publisher interface {
	Publish(ctx context.Context, topic string, payload any) error
}

type noopPublisher struct{}

// NewNoopPublisher returns a Publisher that discards every event.
func NewNoopPublisher() Publisher { return noopPublisher{} }

func (noopPublisher) Publish(context.Context, string, any) error { return nil }

// fromEnv is replaced by a messaging provider file (bus_<provider>.go) when a
// provider is enabled with ` + "`gokub enable messaging <provider>`" + `. It returns a
// live publisher and true only when the broker is configured in the environment.
var fromEnv = func() (Publisher, bool) { return nil, false }

// NewPublisherFromEnvOrNoop returns a broker-backed publisher when a messaging
// provider is enabled and configured, otherwise a no-op publisher.
func NewPublisherFromEnvOrNoop() Publisher {
	if p, ok := fromEnv(); ok {
		return p
	}
	return NewNoopPublisher()
}
`
}

// kitRabbitBusFile returns a RabbitMQ-backed publisher that registers itself as
// the environment factory. Named bus_rabbitmq.go so it can be swapped by other
// providers.
func kitRabbitBusFile() string {
	return `package events

import (
	"context"
	"encoding/json"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
)

func init() {
	fromEnv = func() (Publisher, bool) {
		url := os.Getenv("RABBITMQ_URL")
		if url == "" {
			return nil, false
		}
		publisher, err := newRabbitPublisher(url, os.Getenv("RABBITMQ_EXCHANGE"))
		if err != nil {
			logrus.Warnf("rabbitmq unavailable, using no-op publisher: %v", err)
			return nil, false
		}
		return publisher, true
	}
}

type rabbitPublisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

func newRabbitPublisher(url, exchange string) (*rabbitPublisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, err
	}
	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if exchange != "" {
		if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
			_ = channel.Close()
			_ = conn.Close()
			return nil, err
		}
	}
	return &rabbitPublisher{conn: conn, channel: channel, exchange: exchange}, nil
}

func (p *rabbitPublisher) Publish(ctx context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.channel.PublishWithContext(ctx, p.exchange, topic, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

// Close releases the channel and connection.
func (p *rabbitPublisher) Close() error {
	_ = p.channel.Close()
	return p.conn.Close()
}
`
}

// kitKafkaBusFile returns a Kafka-backed publisher (franz-go) that registers
// itself as the environment factory when KAFKA_BROKERS is set.
func kitKafkaBusFile() string {
	return `package events

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
)

func init() {
	fromEnv = func() (Publisher, bool) {
		brokers := os.Getenv("KAFKA_BROKERS")
		if brokers == "" {
			return nil, false
		}
		publisher, err := newKafkaPublisher(strings.Split(brokers, ","))
		if err != nil {
			logrus.Warnf("kafka unavailable, using no-op publisher: %v", err)
			return nil, false
		}
		return publisher, true
	}
}

type kafkaPublisher struct {
	client *kgo.Client
}

func newKafkaPublisher(brokers []string) (*kafkaPublisher, error) {
	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, err
	}
	return &kafkaPublisher{client: client}, nil
}

func (p *kafkaPublisher) Publish(ctx context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.client.ProduceSync(ctx, &kgo.Record{Topic: topic, Value: body}).FirstErr()
}

// Close releases the Kafka client.
func (p *kafkaPublisher) Close() { p.client.Close() }
`
}

// kitNatsBusFile returns a NATS-backed publisher that registers itself as the
// environment factory when NATS_URL is set.
func kitNatsBusFile() string {
	return `package events

import (
	"context"
	"encoding/json"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

func init() {
	fromEnv = func() (Publisher, bool) {
		url := os.Getenv("NATS_URL")
		if url == "" {
			return nil, false
		}
		publisher, err := newNatsPublisher(url)
		if err != nil {
			logrus.Warnf("nats unavailable, using no-op publisher: %v", err)
			return nil, false
		}
		return publisher, true
	}
}

type natsPublisher struct {
	conn *nats.Conn
}

func newNatsPublisher(url string) (*natsPublisher, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &natsPublisher{conn: conn}, nil
}

func (p *natsPublisher) Publish(_ context.Context, topic string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.conn.Publish(topic, body)
}

// Close drains and closes the NATS connection.
func (p *natsPublisher) Close() { p.conn.Close() }
`
}

// messagingProviders lists providers that wire the internal/app/events bus.
var messagingProviders = []string{"rabbitmq", "kafka", "nats"}

func isMessagingProvider(name string) bool {
	for _, provider := range messagingProviders {
		if provider == name {
			return true
		}
	}
	return false
}

func messagingBusContent(provider string) (string, error) {
	switch provider {
	case "rabbitmq":
		return kitRabbitBusFile(), nil
	case "kafka":
		return kitKafkaBusFile(), nil
	case "nats":
		return kitNatsBusFile(), nil
	default:
		return "", fmt.Errorf("unknown messaging provider %q", provider)
	}
}

// wireMessaging writes the events bus file for provider, removing any other
// provider's bus file so exactly one environment factory is active.
func wireMessaging(root, provider string) error {
	eventsDir := filepath.Join(root, "internal", "app", "events")
	if _, err := os.Stat(eventsDir); err != nil {
		return fmt.Errorf("messaging requires a GOKUB kit project with internal/app/events: %w", err)
	}
	content, err := messagingBusContent(provider)
	if err != nil {
		return err
	}
	UnwireMessaging(root)
	if err := os.MkdirAll(eventsDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(eventsDir, "bus_"+provider+".go"), []byte(content), 0o644)
}

// UnwireMessaging removes any generated messaging bus file.
func UnwireMessaging(root string) {
	eventsDir := filepath.Join(root, "internal", "app", "events")
	for _, provider := range messagingProviders {
		_ = os.Remove(filepath.Join(eventsDir, "bus_"+provider+".go"))
	}
}

func kitEventsContractsFile() string {
	return `package events

// Domain event topics published by the example module.
const (
	TopicExampleCreated = "example.created"
	TopicExampleUpdated = "example.updated"
	TopicExampleDeleted = "example.deleted"
)
`
}

func kitModelFile(pkg, typeName string) string {
	return fmt.Sprintf(`package %[1]s

import (
	"time"

	"github.com/google/uuid"
)

// %[2]s is the domain entity persisted by this module.
type %[2]s struct {
	ID        uuid.UUID `+tick+`json:"id" gorm:"type:uuid;primaryKey"`+tick+`
	Name      string    `+tick+`json:"name" gorm:"not null"`+tick+`
	Price     float64   `+tick+`json:"price"`+tick+`
	CreatedAt time.Time `+tick+`json:"createdAt"`+tick+`
	UpdatedAt time.Time `+tick+`json:"updatedAt"`+tick+`
}

// Query captures listing, filtering, and pagination options.
type Query struct {
	Page     int
	PageSize int
	Search   string
}
`, pkg, typeName)
}

func kitRepositoryFile(pkg, typeName string) string {
	return fmt.Sprintf(`package %[1]s

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Repository is the persistence port for %[2]s entities.
type Repository interface {
	AutoMigrate(ctx context.Context) error
	List(ctx context.Context, q Query) ([]%[2]s, int64, error)
	Create(ctx context.Context, item *%[2]s) error
	Get(ctx context.Context, id uuid.UUID) (*%[2]s, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]any) (*%[2]s, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository returns a gorm-backed Repository.
func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) AutoMigrate(ctx context.Context) error {
	return r.db.WithContext(ctx).AutoMigrate(&%[2]s{})
}

func (r *repository) List(ctx context.Context, q Query) ([]%[2]s, int64, error) {
	tx := r.db.WithContext(ctx).Model(&%[2]s{})
	if q.Search != "" {
		tx = tx.Where("name ILIKE ?", "%%"+q.Search+"%%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []%[2]s
	if err := tx.Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *repository) Create(ctx context.Context, item *%[2]s) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *repository) Get(ctx context.Context, id uuid.UUID) (*%[2]s, error) {
	var item %[2]s
	if err := r.db.WithContext(ctx).First(&item, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *repository) Update(ctx context.Context, id uuid.UUID, updates map[string]any) (*%[2]s, error) {
	if err := r.db.WithContext(ctx).Model(&%[2]s{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&%[2]s{}, "id = ?", id).Error
}
`, pkg, typeName)
}

func kitServiceFile(module, pkg, typeName string) string {
	return fmt.Sprintf(`package %[1]s

import (
	"context"
	"time"

	"github.com/google/uuid"

	"%[2]s/internal/app/events"
)

//go:generate mockgen -destination=./mocks/service_mock.go -package=mocks -source=service.go Service

// Service is the business API for the %[3]s module.
type Service interface {
	AutoMigrate(ctx context.Context) error
	List(ctx context.Context, q Query) ([]%[3]s, int64, error)
	Create(ctx context.Context, item *%[3]s) error
	Get(ctx context.Context, id uuid.UUID) (*%[3]s, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]any) (*%[3]s, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct {
	repo      Repository
	publisher events.Publisher
}

// NewService wires a Service to its repository and event publisher.
func NewService(repo Repository, publisher events.Publisher) Service {
	if publisher == nil {
		publisher = events.NewNoopPublisher()
	}
	return &service{repo: repo, publisher: publisher}
}

func (s *service) AutoMigrate(ctx context.Context) error { return s.repo.AutoMigrate(ctx) }

func (s *service) List(ctx context.Context, q Query) ([]%[3]s, int64, error) {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 || q.PageSize > 100 {
		q.PageSize = 20
	}
	return s.repo.List(ctx, q)
}

func (s *service) Create(ctx context.Context, item *%[3]s) error {
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	now := time.Now().UTC()
	item.CreatedAt, item.UpdatedAt = now, now
	if err := s.repo.Create(ctx, item); err != nil {
		return err
	}
	_ = s.publisher.Publish(ctx, "%[1]s.created", item)
	return nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (*%[3]s, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) Update(ctx context.Context, id uuid.UUID, updates map[string]any) (*%[3]s, error) {
	item, err := s.repo.Update(ctx, id, updates)
	if err == nil {
		_ = s.publisher.Publish(ctx, "%[1]s.updated", item)
	}
	return item, err
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	_ = s.publisher.Publish(ctx, "%[1]s.deleted", map[string]string{"id": id.String()})
	return nil
}
`, pkg, module, typeName)
}

func kitServiceTestFile(pkg, typeName string) string {
	return fmt.Sprintf(`package %[1]s

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type memRepository struct {
	items map[uuid.UUID]*%[2]s
}

func newMemRepository() *memRepository {
	return &memRepository{items: map[uuid.UUID]*%[2]s{}}
}

func (r *memRepository) AutoMigrate(context.Context) error { return nil }

func (r *memRepository) List(_ context.Context, _ Query) ([]%[2]s, int64, error) {
	out := make([]%[2]s, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, *item)
	}
	return out, int64(len(out)), nil
}

func (r *memRepository) Create(_ context.Context, item *%[2]s) error {
	r.items[item.ID] = item
	return nil
}

func (r *memRepository) Get(_ context.Context, id uuid.UUID) (*%[2]s, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return item, nil
}

func (r *memRepository) Update(_ context.Context, id uuid.UUID, _ map[string]any) (*%[2]s, error) {
	item, ok := r.items[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return item, nil
}

func (r *memRepository) Delete(_ context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}

func TestServiceCreateAssignsIdentity(t *testing.T) {
	svc := NewService(newMemRepository(), nil)
	item := &%[2]s{Name: "widget", Price: 9.99}
	if err := svc.Create(context.Background(), item); err != nil {
		t.Fatalf("create: %%v", err)
	}
	if item.ID == uuid.Nil {
		t.Fatal("expected generated identifier")
	}
	got, err := svc.Get(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("get: %%v", err)
	}
	if got.Name != "widget" {
		t.Fatalf("expected persisted item, got %%q", got.Name)
	}
}
`, pkg, typeName)
}

func kitAPIResponseFile() string {
	return `package api

// APIResponse is the standard success envelope.
type APIResponse[T any] struct {
	Success bool   ` + tick + `json:"success" example:"true"` + tick + `
	Message string ` + tick + `json:"message,omitempty"` + tick + `
	Result  T      ` + tick + `json:"result"` + tick + `
}

// APIMessage is a bodyless success or informational response.
type APIMessage struct {
	Success bool   ` + tick + `json:"success" example:"true"` + tick + `
	Message string ` + tick + `json:"message"` + tick + `
}

// APIError is the standard error envelope.
type APIError struct {
	Success   bool   ` + tick + `json:"success" example:"false"` + tick + `
	ErrorCode string ` + tick + `json:"errorCode" example:"NOT_FOUND"` + tick + `
	Message   string ` + tick + `json:"message" example:"not found"` + tick + `
}

// PaginatedContent wraps a page of results with pagination metadata.
type PaginatedContent[T any] struct {
	APIResponse[T]
	Total     int64 ` + tick + `json:"total"` + tick + `
	Page      int64 ` + tick + `json:"page"` + tick + `
	PerPage   int64 ` + tick + `json:"perPage"` + tick + `
	TotalPage int64 ` + tick + `json:"totalPage"` + tick + `
}
`
}

func kitErrorFile() string {
	return `package apperror

import "net/http"

// Error is a typed domain error carrying an HTTP status and stable code.
type Error struct {
	Status  int
	Code    string
	Message string
}

func (e Error) Error() string { return e.Message }

// NotFound reports a missing resource.
func NotFound(message string) Error {
	return Error{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: message}
}

// BadRequest reports invalid input.
func BadRequest(message string) Error {
	return Error{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: message}
}

// Internal reports an unexpected failure.
func Internal(message string) Error {
	return Error{Status: http.StatusInternalServerError, Code: "INTERNAL", Message: message}
}
`
}

func kitPostgresFile() string {
	return `package postgresql

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Open connects to PostgreSQL using gorm.
func Open(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
`
}

func kitUtilsFile() string {
	return `package utils

import "strconv"

// Atoi parses s, returning fallback when s is empty or invalid.
func Atoi(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return fallback
}
`
}

func kitMakefile(service string) string {
	return fmt.Sprintf(`SCORE_MIN ?= 80

.PHONY: run test build fmt vet tidy doctor score graph graph-check

run:
	go run ./cmd/%s

test:
	go test -race -cover ./...

build:
	go build ./...

fmt:
	gofmt -w $$(find cmd internal pkg config -name '*.go')

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
`, service)
}

func kitDockerfile(service, goVersion string) string {
	return fmt.Sprintf(`FROM golang:%s-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./cmd/%s

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/app /app
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app"]
`, goVersion, service)
}

func kitComposeFile(name string) string {
	return fmt.Sprintf(`services:
  %s:
    build: .
    env_file: .env
    ports:
      - "8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
  postgres:
    image: postgres:17-alpine
    environment:
      POSTGRES_USER: app
      POSTGRES_PASSWORD: app
      POSTGRES_DB: app
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app"]
      interval: 5s
      timeout: 3s
      retries: 10
`, name)
}

func kitEnvFile(name, messaging string) string {
	base := fmt.Sprintf(`# %s
APP_ENV=local
PORT=8080
LOG_LEVEL=debug
DATABASE_URL=postgres://app:app@localhost:5432/app?sslmode=disable
`, name)
	switch messaging {
	case "rabbitmq":
		base += "RABBITMQ_URL=amqp://guest:guest@localhost:5672/\nRABBITMQ_EXCHANGE=events\n"
	case "kafka":
		base += "KAFKA_BROKERS=localhost:9092\n"
	case "nats":
		base += "NATS_URL=nats://localhost:4222\n"
	}
	return base
}

func kitVSCodeFile(service string) string {
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
      "envFile": "${workspaceFolder}/.env"
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
`, service)
}

func kitJetBrainsFile(m manifest.Manifest, service string) string {
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
    <method v="2" />
  </configuration>
</component>
`, m.Name, m.Module, service)
}

func kitReadmeFile(m manifest.Manifest, service, domain string) string {
	return fmt.Sprintf(`# %s

![GOKUB](docs/gokub_logo.png)

Production-ready %s service generated by GOKUB.

## Stack

- HTTP framework: **%s**
- Persistence: **gorm + PostgreSQL**
- Config: **envconfig**
- Logging: **logrus**

## Start

A ready-to-run `+"`.env`"+` is generated (and git-ignored). Adjust it as needed.

`+"```bash"+`
docker compose up -d postgres
go mod download
make test
make run
`+"```"+`

## Structure

`+"```text"+`
cmd/%s/            service entrypoint and wiring
config/            environment configuration
internal/%s/       domain: model, repository, service, handler, router
internal/app/      composition and event contracts
pkg/api/           response envelopes
pkg/database/      gorm database adapters
pkg/error/         typed domain errors
pkg/httpserver/    framework server bootstrap and graceful shutdown
pkg/middleware/    logging, recovery, and secure headers
pkg/utils/         small shared helpers
`+"```"+`

## Health

`+"```bash"+`
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready
`+"```"+`

## AI Agents and MCP

Codex reads `+"`.codex/config.toml`"+`; Claude-compatible clients read `+"`.mcp.json`"+`.
Both launch `+"`gokub mcp serve`"+` and expose typed project tools.
`, m.Name, m.Style, m.Framework, service, domain)
}
