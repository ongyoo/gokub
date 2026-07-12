package catalog

import "sort"

type Feature struct {
	Name        string
	Capability  string
	Description string
}

type Recipe struct {
	Name        string
	Description string
	Features    []string
}

type Capability struct {
	Name        string
	Description string
	Providers   []string
}

var Features = map[string]Feature{
	"auth":      {Name: "auth", Capability: "authentication", Description: "JWT authentication with secure password hashing"},
	"crud":      {Name: "crud", Capability: "api", Description: "CRUD module scaffold"},
	"redis":     {Name: "redis", Capability: "cache", Description: "Redis client and config"},
	"kafka":     {Name: "kafka", Capability: "messaging", Description: "Kafka producer and consumer wiring"},
	"rabbitmq":  {Name: "rabbitmq", Capability: "messaging", Description: "RabbitMQ publisher and consumer wiring"},
	"grpc":      {Name: "grpc", Capability: "api", Description: "gRPC server scaffold"},
	"cron":      {Name: "cron", Capability: "jobs", Description: "Scheduled job runner"},
	"email":     {Name: "email", Capability: "notification", Description: "Email delivery abstraction"},
	"websocket": {Name: "websocket", Capability: "realtime", Description: "WebSocket endpoint scaffold"},
	"postgres":  {Name: "postgres", Capability: "database", Description: "PostgreSQL database wiring"},
	"mongodb":   {Name: "mongodb", Capability: "database", Description: "MongoDB database wiring"},
	"nats":      {Name: "nats", Capability: "messaging", Description: "NATS publisher and subscriber wiring"},
	"otel":      {Name: "otel", Capability: "observability", Description: "OpenTelemetry tracing setup"},
	"docker":    {Name: "docker", Capability: "infrastructure", Description: "Docker and compose files"},
	"github-actions": {Name: "github-actions", Capability: "ci",
		Description: "GitHub Actions build and test workflow"},
	"outbox": {Name: "outbox", Capability: "messaging", Description: "Transactional outbox scaffold"},
}

var Capabilities = map[string]Capability{
	"authentication": {
		Name:        "authentication",
		Description: "Login, identity, and access control",
		Providers:   []string{"auth"},
	},
	"cache": {
		Name:        "cache",
		Description: "Fast key-value caching",
		Providers:   []string{"redis"},
	},
	"database": {
		Name:        "database",
		Description: "Primary persistence provider",
		Providers:   []string{"postgres", "mongodb"},
	},
	"infrastructure": {
		Name:        "infrastructure",
		Description: "Deployment and CI foundation",
		Providers:   []string{"docker", "github-actions"},
	},
	"messaging": {
		Name:        "messaging",
		Description: "Async events and background communication",
		Providers:   []string{"kafka", "rabbitmq", "nats"},
	},
	"observability": {
		Name:        "observability",
		Description: "Tracing and production visibility",
		Providers:   []string{"otel"},
	},
}

var Recipes = map[string]Recipe{
	"event-driven": {
		Name:        "event-driven",
		Description: "PostgreSQL, Kafka, outbox, OpenTelemetry, Docker, and GitHub Actions",
		Features:    []string{"postgres", "kafka", "outbox", "otel", "docker", "github-actions"},
	},
	"api": {
		Name:        "api",
		Description: "REST API with auth, PostgreSQL, Redis, Docker, and CI",
		Features:    []string{"auth", "postgres", "redis", "docker", "github-actions"},
	},
}

func HasFeature(name string) bool {
	_, ok := Features[name]
	return ok
}

func FeatureNames() []string {
	names := make([]string, 0, len(Features))
	for name := range Features {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func CapabilityNames() []string {
	names := make([]string, 0, len(Capabilities))
	for name := range Capabilities {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func RecipeNames() []string {
	names := make([]string, 0, len(Recipes))
	for name := range Recipes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func HasCapability(name string) bool {
	_, ok := Capabilities[name]
	return ok
}

func ProviderForCapability(capability, provider string) bool {
	cap, ok := Capabilities[capability]
	if !ok {
		return false
	}
	for _, candidate := range cap.Providers {
		if candidate == provider {
			return true
		}
	}
	return false
}
