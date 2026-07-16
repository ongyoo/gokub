package generator

// featureScaffold returns a starter implementation for a capability feature.
// Every scaffold ships working code plus a guideline comment describing what to
// configure and implement next, so no generated file is left empty.
func featureScaffold(feature string) string {
	switch feature {
	case "auth":
		return featureAuth()
	case "redis":
		return featureRedis()
	case "postgres":
		return featurePostgres()
	case "mongodb":
		return featureMongo()
	case "otel":
		return featureOtel()
	case "cron":
		return featureCron()
	case "email":
		return featureEmail()
	case "websocket":
		return featureWebsocket()
	case "grpc":
		return featureGRPC()
	case "outbox":
		return featureOutbox()
	default:
		return featureGuidelineOnly(feature)
	}
}

func featureAuth() string {
	return `package auth

// Package auth issues and verifies JWT access tokens.
//
// Guideline:
//   - Set AUTH_JWT_SECRET (>= 32 bytes) in the environment.
//   - Call Sign after authenticating a user and return the token to the client.
//   - Call Parse in middleware to authorize requests; store only the subject
//     (user id) in the token and load the rest from the database.

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrMissingSecret is returned when AUTH_JWT_SECRET is unset or too short.
var ErrMissingSecret = errors.New("AUTH_JWT_SECRET must be set to at least 32 bytes")

func secret() ([]byte, error) {
	value := os.Getenv("AUTH_JWT_SECRET")
	if len(value) < 32 {
		return nil, ErrMissingSecret
	}
	return []byte(value), nil
}

// Sign returns a signed HS256 JWT for subject valid for ttl.
func Sign(subject string, ttl time.Duration) (string, error) {
	key, err := secret()
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(key)
}

// Parse validates token and returns its subject.
func Parse(token string) (string, error) {
	key, err := secret()
	if err != nil {
		return "", err
	}
	parsed, err := jwt.ParseWithClaims(token, &jwt.RegisteredClaims{}, func(*jwt.Token) (any, error) {
		return key, nil
	})
	if err != nil {
		return "", err
	}
	claims, ok := parsed.Claims.(*jwt.RegisteredClaims)
	if !ok || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	return claims.Subject, nil
}
`
}

func featureRedis() string {
	return `package redis

// Package redis provides a shared Redis client.
//
// Guideline:
//   - Set REDIS_ADDR (host:port); defaults to localhost:6379.
//   - Call Open once at startup and inject the client into caches and
//     rate limiters.

import (
	"context"
	"os"

	goredis "github.com/redis/go-redis/v9"
)

// Open connects to Redis and verifies connectivity.
func Open(ctx context.Context) (*goredis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	client := goredis.NewClient(&goredis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
`
}

func featurePostgres() string {
	return `package postgres

// Package postgres documents the PostgreSQL integration.
//
// This project already ships a gorm PostgreSQL adapter at
// pkg/database/postgresql. Prefer that package for new code.
//
// Guideline:
//   - Set DATABASE_URL in the environment.
//   - Open the connection in main and inject *gorm.DB into repositories.
//   - Run schema changes through a domain AutoMigrate call or a migration tool.
`
}

func featureMongo() string {
	return `package mongodb

// Package mongodb provides a shared MongoDB client.
//
// Guideline:
//   - Set MONGODB_URI; defaults to mongodb://localhost:27017.
//   - Call Open once at startup and inject the database handle into
//     repositories.

import (
	"context"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Open connects to MongoDB and verifies connectivity.
func Open(ctx context.Context) (*mongo.Client, error) {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, err
	}
	return client, nil
}
`
}

func featureOtel() string {
	return `package otel

// Package otel wires OpenTelemetry tracing.
//
// Guideline:
//   - Configure an OTLP exporter and set OTEL_EXPORTER_OTLP_ENDPOINT.
//   - Call Tracer(service) to obtain a tracer and start spans around handlers,
//     service methods, and outbound calls.
//   - Add otelhttp/otelgin middleware to propagate trace context.

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// Tracer returns a named tracer for the service.
func Tracer(service string) trace.Tracer {
	return otel.Tracer(service)
}
`
}

func featureCron() string {
	return `package cron

// Package cron runs periodic background jobs.
//
// Guideline:
//   - Register jobs at startup and run Start in a goroutine.
//   - Cancel the context on shutdown for a clean stop.
//   - For calendar (cron-syntax) schedules, plug in a cron library.

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// Job is a unit of periodic work.
type Job func(context.Context) error

// Scheduler runs a Job on a fixed interval until the context is cancelled.
type Scheduler struct {
	interval time.Duration
	job      Job
}

// NewScheduler builds a Scheduler for the given interval and job.
func NewScheduler(interval time.Duration, job Job) *Scheduler {
	return &Scheduler{interval: interval, job: job}
}

// Start blocks running the job every interval until ctx is cancelled.
func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.job(ctx); err != nil {
				logrus.Errorf("cron job failed: %v", err)
			}
		}
	}
}
`
}

func featureEmail() string {
	return `package email

// Package email sends transactional email over SMTP.
//
// Guideline:
//   - Set SMTP_ADDR (host:port), SMTP_USER, SMTP_PASSWORD, and EMAIL_FROM.
//   - Call Send for simple messages; switch to a provider SDK for templates,
//     attachments, and deliverability tracking.

import (
	"fmt"
	"net/smtp"
	"os"
	"strings"
)

// Send delivers a plain-text email to recipients.
func Send(to []string, subject, body string) error {
	addr := os.Getenv("SMTP_ADDR")
	from := os.Getenv("EMAIL_FROM")
	if addr == "" || from == "" {
		return fmt.Errorf("SMTP_ADDR and EMAIL_FROM must be set")
	}
	host := addr
	if i := strings.IndexByte(addr, ':'); i >= 0 {
		host = addr[:i]
	}
	auth := smtp.PlainAuth("", os.Getenv("SMTP_USER"), os.Getenv("SMTP_PASSWORD"), host)
	msg := []byte("From: " + from + "\r\n" +
		"To: " + strings.Join(to, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n\r\n" + body + "\r\n")
	return smtp.SendMail(addr, auth, from, to, msg)
}
`
}

func featureWebsocket() string {
	return `package websocket

// Package websocket documents the real-time transport.
//
// Guideline:
//   - Add a WebSocket library (for example github.com/coder/websocket) and run
//     ` + "`go mod tidy`" + `.
//   - Upgrade the connection in an HTTP handler, then read and write JSON frames.
//   - Track connected clients in a hub and broadcast domain events published by
//     internal/app/events.
//
// This file is intentionally dependency-free; wire the library above when you
// start implementing.
`
}

func featureGRPC() string {
	return `package grpc

// Package grpc bootstraps a gRPC server.
//
// Guideline:
//   - Define services in .proto files and generate code with protoc or buf.
//   - Register generated servers on Server() before calling Serve.
//   - Set GRPC_ADDR (host:port) and reuse the HTTP graceful-shutdown pattern.

import "google.golang.org/grpc"

// Server returns a new gRPC server. Register generated services on it, then
// call srv.Serve(listener).
func Server() *grpc.Server {
	return grpc.NewServer()
}
`
}

func featureOutbox() string {
	return `package outbox

// Package outbox implements the transactional outbox pattern.
//
// Guideline:
//   - Persist domain changes and an outbox Message in the same DB transaction.
//   - A relay polls unpublished messages, publishes them via internal/app/events,
//     and stamps PublishedAt on success.
//   - This gives at-least-once delivery without dual-write races.

import (
	"time"

	"github.com/google/uuid"
)

// Message is a pending event awaiting publication.
type Message struct {
	ID          uuid.UUID  ` + "`json:\"id\" gorm:\"type:uuid;primaryKey\"`" + `
	Topic       string     ` + "`json:\"topic\"`" + `
	Payload     []byte     ` + "`json:\"payload\"`" + `
	CreatedAt   time.Time  ` + "`json:\"createdAt\"`" + `
	PublishedAt *time.Time ` + "`json:\"publishedAt\"`" + `
}
`
}

func featureGuidelineOnly(feature string) string {
	return "package " + feature + "\n\n" +
		"// Package " + feature + " is a capability scaffold generated by GOKUB.\n" +
		"//\n" +
		"// Guideline: implement the " + feature + " integration here, add any\n" +
		"// required dependency, and run `go mod tidy`.\n"
}
