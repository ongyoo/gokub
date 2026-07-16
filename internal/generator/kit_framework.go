package generator

import "fmt"

// kitMainFile returns the cmd/<service>/main.go entrypoint wired for the chosen
// HTTP framework.
func kitMainFile(module, framework, domain string) string {
	switch framework {
	case "fiber":
		return kitFiberMain(module, domain)
	case "echo":
		return kitEchoMain(module, domain)
	default:
		return kitGinMain(module, domain)
	}
}

func kitHTTPServerFile(framework string) string {
	switch framework {
	case "fiber":
		return kitFiberServer()
	case "echo":
		return kitEchoServer()
	default:
		return kitGinServer()
	}
}

func kitMiddlewareFile(framework string) string {
	switch framework {
	case "fiber":
		return kitFiberMiddleware()
	case "echo":
		return kitEchoMiddleware()
	default:
		return kitGinMiddleware()
	}
}

func kitHandlerFile(module, framework, domain, typeName string) string {
	switch framework {
	case "fiber":
		return kitFiberHandler(module, domain, typeName)
	case "echo":
		return kitEchoHandler(module, domain, typeName)
	default:
		return kitGinHandler(module, domain, typeName)
	}
}

func kitRouterFile(framework, domain string) string {
	switch framework {
	case "fiber":
		return kitFiberRouter(domain)
	case "echo":
		return kitEchoRouter(domain)
	default:
		return kitGinRouter(domain)
	}
}

// ---------------------------------------------------------------------------
// main.go
// ---------------------------------------------------------------------------

func kitFiberMain(module, domain string) string {
	return fmt.Sprintf(`package main

import (
	"context"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"

	"%[1]s/config"
	"%[1]s/internal/app/events"
	"%[1]s/internal/%[2]s"
	db "%[1]s/pkg/database/postgresql"
	httpserver "%[1]s/pkg/httpserver/fiber"
	middleware "%[1]s/pkg/middleware/fiber"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		logrus.Fatalf("connect database: %%v", err)
	}

	repo := %[2]s.NewRepository(database)
	service := %[2]s.NewService(repo, events.NewPublisherFromEnvOrNoop())
	if err := service.AutoMigrate(context.Background()); err != nil {
		logrus.Fatalf("migrate: %%v", err)
	}
	handler := %[2]s.NewHandler(service)

	server := httpserver.NewServer(cfg.Port)
	server.App.Use(middleware.Recover(), middleware.SecureHeaders(), middleware.RequestLogger())
	server.App.Get("/health/live", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	server.App.Get("/health/ready", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	api := server.App.Group("/api")
	%[2]s.SetRoutes(api.Group("/%[2]ss"), handler)

	if err := server.Run(); err != nil {
		logrus.Fatalf("server: %%v", err)
	}
}
`, module, domain)
}

func kitGinMain(module, domain string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"%[1]s/config"
	"%[1]s/internal/app/events"
	"%[1]s/internal/%[2]s"
	db "%[1]s/pkg/database/postgresql"
	httpserver "%[1]s/pkg/httpserver/gin"
	middleware "%[1]s/pkg/middleware/gin"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		logrus.Fatalf("connect database: %%v", err)
	}

	repo := %[2]s.NewRepository(database)
	service := %[2]s.NewService(repo, events.NewPublisherFromEnvOrNoop())
	if err := service.AutoMigrate(context.Background()); err != nil {
		logrus.Fatalf("migrate: %%v", err)
	}
	handler := %[2]s.NewHandler(service)

	server := httpserver.NewServer(cfg.Port)
	server.Router.Use(middleware.Recover(), middleware.SecureHeaders(), middleware.RequestLogger())
	server.Router.GET("/health/live", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	server.Router.GET("/health/ready", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })

	api := server.Router.Group("/api")
	%[2]s.SetRoutes(api.Group("/%[2]ss"), handler)

	if err := server.Run(); err != nil {
		logrus.Fatalf("server: %%v", err)
	}
}
`, module, domain)
}

func kitEchoMain(module, domain string) string {
	return fmt.Sprintf(`package main

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"

	"%[1]s/config"
	"%[1]s/internal/app/events"
	"%[1]s/internal/%[2]s"
	db "%[1]s/pkg/database/postgresql"
	httpserver "%[1]s/pkg/httpserver/echo"
	middleware "%[1]s/pkg/middleware/echo"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		logrus.Fatalf("connect database: %%v", err)
	}

	repo := %[2]s.NewRepository(database)
	service := %[2]s.NewService(repo, events.NewPublisherFromEnvOrNoop())
	if err := service.AutoMigrate(context.Background()); err != nil {
		logrus.Fatalf("migrate: %%v", err)
	}
	handler := %[2]s.NewHandler(service)

	server := httpserver.NewServer(cfg.Port)
	server.Echo.Use(middleware.Recover(), middleware.SecureHeaders(), middleware.RequestLogger())
	server.Echo.GET("/health/live", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok"}) })
	server.Echo.GET("/health/ready", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"status": "ok"}) })

	api := server.Echo.Group("/api")
	%[2]s.SetRoutes(api.Group("/%[2]ss"), handler)

	if err := server.Run(); err != nil {
		logrus.Fatalf("server: %%v", err)
	}
}
`, module, domain)
}

// ---------------------------------------------------------------------------
// pkg/httpserver
// ---------------------------------------------------------------------------

func kitFiberServer() string {
	return `package httpserver

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

// Server wraps a Fiber application with graceful shutdown.
type Server struct {
	App  *fiber.App
	Addr string
}

// NewServer creates a Fiber server bound to the given port.
func NewServer(port string) *Server {
	app := fiber.New(fiber.Config{
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	})
	return &Server{App: app, Addr: ":" + port}
}

// Run starts the server and blocks until SIGINT or SIGTERM, then shuts down.
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logrus.Infof("http server listening on %s", s.Addr)
		if err := s.App.Listen(s.Addr); err != nil {
			logrus.Errorf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	logrus.Info("shutting down http server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.App.ShutdownWithContext(shutdownCtx)
}
`
}

func kitGinServer() string {
	return `package httpserver

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server wraps a Gin engine and http.Server with graceful shutdown.
type Server struct {
	Router *gin.Engine
	server *http.Server
}

// NewServer creates a Gin server bound to the given port.
func NewServer(port string) *Server {
	router := gin.New()
	return &Server{Router: router, server: &http.Server{
		Addr:              ":" + port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}}
}

// Run starts the server and blocks until SIGINT or SIGTERM, then shuts down.
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logrus.Infof("http server listening on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	logrus.Info("shutting down http server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.server.Shutdown(shutdownCtx)
}
`
}

func kitEchoServer() string {
	return `package httpserver

import (
	"context"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// Server wraps an Echo instance with graceful shutdown.
type Server struct {
	Echo *echo.Echo
	Addr string
}

// NewServer creates an Echo server bound to the given port.
func NewServer(port string) *Server {
	e := echo.New()
	e.HideBanner = true
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 15 * time.Second
	e.Server.WriteTimeout = 30 * time.Second
	e.Server.IdleTimeout = 60 * time.Second
	return &Server{Echo: e, Addr: ":" + port}
}

// Run starts the server and blocks until SIGINT or SIGTERM, then shuts down.
func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logrus.Infof("http server listening on %s", s.Addr)
		if err := s.Echo.Start(s.Addr); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("listen: %v", err)
		}
	}()

	<-ctx.Done()
	logrus.Info("shutting down http server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return s.Echo.Shutdown(shutdownCtx)
}
`
}

// ---------------------------------------------------------------------------
// pkg/middleware
// ---------------------------------------------------------------------------

func kitFiberMiddleware() string {
	return `package middleware

import (
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

// RequestLogger logs one structured line per request.
func RequestLogger() fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		logrus.WithFields(logrus.Fields{
			"method":      c.Method(),
			"path":        c.Path(),
			"status":      c.Response().StatusCode(),
			"duration_ms": time.Since(start).Milliseconds(),
		}).Info("http request")
		return err
	}
}

// Recover converts panics into 500 responses.
func Recover() fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("panic recovered: %v", r)
				err = c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false, "message": "internal server error"})
			}
		}()
		return c.Next()
	}
}

// SecureHeaders sets conservative security response headers.
func SecureHeaders() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("X-Frame-Options", "DENY")
		c.Set("Referrer-Policy", "no-referrer")
		return c.Next()
	}
}
`
}

func kitGinMiddleware() string {
	return `package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLogger logs one structured line per request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		logrus.WithFields(logrus.Fields{
			"method":      c.Request.Method,
			"path":        c.Request.URL.Path,
			"status":      c.Writer.Status(),
			"duration_ms": time.Since(start).Milliseconds(),
		}).Info("http request")
	}
}

// Recover converts panics into 500 responses.
func Recover() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logrus.Errorf("panic recovered: %v", r)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"success": false, "message": "internal server error"})
			}
		}()
		c.Next()
	}
}

// SecureHeaders sets conservative security response headers.
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}
`
}

func kitEchoMiddleware() string {
	return `package middleware

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// RequestLogger logs one structured line per request.
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			logrus.WithFields(logrus.Fields{
				"method":      c.Request().Method,
				"path":        c.Request().URL.Path,
				"status":      c.Response().Status,
				"duration_ms": time.Since(start).Milliseconds(),
			}).Info("http request")
			return err
		}
	}
}

// Recover converts panics into 500 responses.
func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logrus.Errorf("panic recovered: %v", r)
					err = c.JSON(http.StatusInternalServerError, map[string]any{"success": false, "message": "internal server error"})
				}
			}()
			return next(c)
		}
	}
}

// SecureHeaders sets conservative security response headers.
func SecureHeaders() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			h := c.Response().Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "no-referrer")
			return next(c)
		}
	}
}
`
}

// ---------------------------------------------------------------------------
// internal/<domain>/handler.go and router.go
// ---------------------------------------------------------------------------

func kitFiberHandler(module, domain, typeName string) string {
	return fmt.Sprintf(`package %[2]s

import (
	"net/http"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"%[1]s/pkg/api"
	"%[1]s/pkg/utils"
	"%[1]s/pkg/validator"
)

// Handler exposes the %[3]s HTTP endpoints.
type Handler struct {
	service Service
}

// NewHandler builds a Handler over the %[3]s service.
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// List godoc
// @Summary  List %[3]s records
// @Tags     %[2]s
// @Produce  json
// @Param    page      query  int     false  "page number"
// @Param    pageSize  query  int     false  "page size"
// @Param    search    query  string  false  "search term"
// @Success  200  {object}  api.PaginatedContent[[]%[3]s]
// @Router   /%[2]ss [get]
func (h Handler) List(c fiber.Ctx) error {
	page := utils.Atoi(c.Query("page"), 1)
	pageSize := utils.Atoi(c.Query("pageSize"), 20)
	items, total, err := h.service.List(c.Context(), Query{Page: page, PageSize: pageSize, Search: c.Query("search")})
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(api.PaginatedContent[[]%[3]s]{
		APIResponse: api.APIResponse[[]%[3]s]{Success: true, Result: items},
		Total:       total,
		Page:        int64(page),
		PerPage:     int64(pageSize),
		TotalPage:   totalPages(total, pageSize),
	})
}

// Create godoc
// @Summary  Create a %[3]s
// @Tags     %[2]s
// @Accept   json
// @Produce  json
// @Param    body  body  %[4]s  true  "payload"
// @Success  201  {object}  api.APIResponse[%[3]s]
// @Failure  400  {object}  api.APIError
// @Router   /%[2]ss [post]
func (h Handler) Create(c fiber.Ctx) error {
	var req %[4]s
	if err := c.Bind().Body(&req); err != nil {
		return fail(c, http.StatusBadRequest, err.Error())
	}
	if err := validator.Struct(req); err != nil {
		return fail(c, http.StatusBadRequest, err.Error())
	}
	item := %[3]s{Name: req.Name, Price: req.Price}
	if err := h.service.Create(c.Context(), &item); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.Status(http.StatusCreated).JSON(api.APIResponse[%[3]s]{Success: true, Result: item})
}

// Get godoc
// @Summary  Get a %[3]s by id
// @Tags     %[2]s
// @Produce  json
// @Param    id   path  string  true  "identifier"
// @Success  200  {object}  api.APIResponse[%[3]s]
// @Failure  404  {object}  api.APIError
// @Router   /%[2]ss/{id} [get]
func (h Handler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fail(c, http.StatusBadRequest, "invalid id")
	}
	item, err := h.service.Get(c.Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, err.Error())
	}
	return c.JSON(api.APIResponse[%[3]s]{Success: true, Result: *item})
}

// Update godoc
// @Summary  Update a %[3]s
// @Tags     %[2]s
// @Accept   json
// @Produce  json
// @Param    id    path  string          true  "identifier"
// @Param    body  body  map[string]any  true  "fields to update"
// @Success  200  {object}  api.APIResponse[%[3]s]
// @Failure  400  {object}  api.APIError
// @Router   /%[2]ss/{id} [patch]
func (h Handler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fail(c, http.StatusBadRequest, "invalid id")
	}
	var updates map[string]any
	if err := c.Bind().Body(&updates); err != nil {
		return fail(c, http.StatusBadRequest, err.Error())
	}
	item, err := h.service.Update(c.Context(), id, updates)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(api.APIResponse[%[3]s]{Success: true, Result: *item})
}

// Delete godoc
// @Summary  Delete a %[3]s
// @Tags     %[2]s
// @Produce  json
// @Param    id   path  string  true  "identifier"
// @Success  200  {object}  api.APIMessage
// @Router   /%[2]ss/{id} [delete]
func (h Handler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fail(c, http.StatusBadRequest, "invalid id")
	}
	if err := h.service.Delete(c.Context(), id); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(api.APIMessage{Success: true, Message: "deleted"})
}

func fail(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(api.APIError{ErrorCode: http.StatusText(status), Message: message})
}

func totalPages(total int64, pageSize int) int64 {
	if pageSize <= 0 {
		return 0
	}
	return (total + int64(pageSize) - 1) / int64(pageSize)
}
`, module, domain, typeName, requestTypeName(typeName))
}

func kitFiberRouter(domain string) string {
	return fmt.Sprintf(`package %s

import "github.com/gofiber/fiber/v3"

// SetRoutes attaches the resource endpoints to the router group, applying any
// group-scoped middleware passed by the caller.
func SetRoutes(router fiber.Router, h Handler, middlewares ...fiber.Handler) {
	for _, m := range middlewares {
		router.Use(m)
	}
	router.Get("/", h.List)
	router.Post("/", h.Create)
	router.Get("/:id", h.Get)
	router.Patch("/:id", h.Update)
	router.Delete("/:id", h.Delete)
}
`, domain)
}

func kitGinHandler(module, domain, typeName string) string {
	return fmt.Sprintf(`package %[2]s

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"%[1]s/pkg/api"
	"%[1]s/pkg/utils"
	"%[1]s/pkg/validator"
)

// Handler exposes the %[3]s HTTP endpoints.
type Handler struct {
	service Service
}

// NewHandler builds a Handler over the %[3]s service.
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// List godoc
// @Summary  List %[3]s records
// @Tags     %[2]s
// @Produce  json
// @Param    page      query  int     false  "page number"
// @Param    pageSize  query  int     false  "page size"
// @Param    search    query  string  false  "search term"
// @Success  200  {object}  api.PaginatedContent[[]%[3]s]
// @Router   /%[2]ss [get]
func (h Handler) List(c *gin.Context) {
	page := utils.Atoi(c.Query("page"), 1)
	pageSize := utils.Atoi(c.Query("pageSize"), 20)
	items, total, err := h.service.List(c.Request.Context(), Query{Page: page, PageSize: pageSize, Search: c.Query("search")})
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, api.PaginatedContent[[]%[3]s]{
		APIResponse: api.APIResponse[[]%[3]s]{Success: true, Result: items},
		Total:       total,
		Page:        int64(page),
		PerPage:     int64(pageSize),
		TotalPage:   totalPages(total, pageSize),
	})
}

// Create godoc
// @Summary  Create a %[3]s
// @Tags     %[2]s
// @Accept   json
// @Produce  json
// @Param    body  body  %[4]s  true  "payload"
// @Success  201  {object}  api.APIResponse[%[3]s]
// @Failure  400  {object}  api.APIError
// @Router   /%[2]ss [post]
func (h Handler) Create(c *gin.Context) {
	var req %[4]s
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	if err := validator.Struct(req); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item := %[3]s{Name: req.Name, Price: req.Price}
	if err := h.service.Create(c.Request.Context(), &item); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusCreated, api.APIResponse[%[3]s]{Success: true, Result: item})
}

// Get godoc
// @Summary  Get a %[3]s by id
// @Tags     %[2]s
// @Produce  json
// @Param    id   path  string  true  "identifier"
// @Success  200  {object}  api.APIResponse[%[3]s]
// @Failure  404  {object}  api.APIError
// @Router   /%[2]ss/{id} [get]
func (h Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		fail(c, http.StatusNotFound, err.Error())
		return
	}
	c.JSON(http.StatusOK, api.APIResponse[%[3]s]{Success: true, Result: *item})
}

// Update godoc
// @Summary  Update a %[3]s
// @Tags     %[2]s
// @Accept   json
// @Produce  json
// @Param    id    path  string          true  "identifier"
// @Param    body  body  map[string]any  true  "fields to update"
// @Success  200  {object}  api.APIResponse[%[3]s]
// @Failure  400  {object}  api.APIError
// @Router   /%[2]ss/{id} [patch]
func (h Handler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	var updates map[string]any
	if err := c.ShouldBindJSON(&updates); err != nil {
		fail(c, http.StatusBadRequest, err.Error())
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, updates)
	if err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, api.APIResponse[%[3]s]{Success: true, Result: *item})
}

// Delete godoc
// @Summary  Delete a %[3]s
// @Tags     %[2]s
// @Produce  json
// @Param    id   path  string  true  "identifier"
// @Success  200  {object}  api.APIMessage
// @Router   /%[2]ss/{id} [delete]
func (h Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		fail(c, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		fail(c, http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, api.APIMessage{Success: true, Message: "deleted"})
}

func fail(c *gin.Context, status int, message string) {
	c.JSON(status, api.APIError{ErrorCode: http.StatusText(status), Message: message})
}

func totalPages(total int64, pageSize int) int64 {
	if pageSize <= 0 {
		return 0
	}
	return (total + int64(pageSize) - 1) / int64(pageSize)
}
`, module, domain, typeName, requestTypeName(typeName))
}

func kitGinRouter(domain string) string {
	return fmt.Sprintf(`package %s

import "github.com/gin-gonic/gin"

// SetRoutes attaches the resource endpoints to the router group, applying any
// group-scoped middleware passed by the caller.
func SetRoutes(group *gin.RouterGroup, h Handler, middlewares ...gin.HandlerFunc) {
	group.Use(middlewares...)
	group.GET("", h.List)
	group.POST("", h.Create)
	group.GET("/:id", h.Get)
	group.PATCH("/:id", h.Update)
	group.DELETE("/:id", h.Delete)
}
`, domain)
}

func kitEchoHandler(module, domain, typeName string) string {
	return fmt.Sprintf(`package %[2]s

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"%[1]s/pkg/api"
	"%[1]s/pkg/utils"
	"%[1]s/pkg/validator"
)

// Handler exposes the %[3]s HTTP endpoints.
type Handler struct {
	service Service
}

// NewHandler builds a Handler over the %[3]s service.
func NewHandler(service Service) Handler {
	return Handler{service: service}
}

// List godoc
// @Summary  List %[3]s records
// @Tags     %[2]s
// @Produce  json
// @Param    page      query  int     false  "page number"
// @Param    pageSize  query  int     false  "page size"
// @Param    search    query  string  false  "search term"
// @Success  200  {object}  api.PaginatedContent[[]%[3]s]
// @Router   /%[2]ss [get]
func (h Handler) List(c echo.Context) error {
	page := utils.Atoi(c.QueryParam("page"), 1)
	pageSize := utils.Atoi(c.QueryParam("pageSize"), 20)
	items, total, err := h.service.List(c.Request().Context(), Query{Page: page, PageSize: pageSize, Search: c.QueryParam("search")})
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, api.PaginatedContent[[]%[3]s]{
		APIResponse: api.APIResponse[[]%[3]s]{Success: true, Result: items},
		Total:       total,
		Page:        int64(page),
		PerPage:     int64(pageSize),
		TotalPage:   totalPages(total, pageSize),
	})
}

// Create godoc
// @Summary  Create a %[3]s
// @Tags     %[2]s
// @Accept   json
// @Produce  json
// @Param    body  body  %[4]s  true  "payload"
// @Success  201  {object}  api.APIResponse[%[3]s]
// @Failure  400  {object}  api.APIError
// @Router   /%[2]ss [post]
func (h Handler) Create(c echo.Context) error {
	var req %[4]s
	if err := c.Bind(&req); err != nil {
		return fail(c, http.StatusBadRequest, err.Error())
	}
	if err := validator.Struct(req); err != nil {
		return fail(c, http.StatusBadRequest, err.Error())
	}
	item := %[3]s{Name: req.Name, Price: req.Price}
	if err := h.service.Create(c.Request().Context(), &item); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusCreated, api.APIResponse[%[3]s]{Success: true, Result: item})
}

// Get godoc
// @Summary  Get a %[3]s by id
// @Tags     %[2]s
// @Produce  json
// @Param    id   path  string  true  "identifier"
// @Success  200  {object}  api.APIResponse[%[3]s]
// @Failure  404  {object}  api.APIError
// @Router   /%[2]ss/{id} [get]
func (h Handler) Get(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return fail(c, http.StatusBadRequest, "invalid id")
	}
	item, err := h.service.Get(c.Request().Context(), id)
	if err != nil {
		return fail(c, http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, api.APIResponse[%[3]s]{Success: true, Result: *item})
}

// Update godoc
// @Summary  Update a %[3]s
// @Tags     %[2]s
// @Accept   json
// @Produce  json
// @Param    id    path  string          true  "identifier"
// @Param    body  body  map[string]any  true  "fields to update"
// @Success  200  {object}  api.APIResponse[%[3]s]
// @Failure  400  {object}  api.APIError
// @Router   /%[2]ss/{id} [patch]
func (h Handler) Update(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return fail(c, http.StatusBadRequest, "invalid id")
	}
	var updates map[string]any
	if err := c.Bind(&updates); err != nil {
		return fail(c, http.StatusBadRequest, err.Error())
	}
	item, err := h.service.Update(c.Request().Context(), id, updates)
	if err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, api.APIResponse[%[3]s]{Success: true, Result: *item})
}

// Delete godoc
// @Summary  Delete a %[3]s
// @Tags     %[2]s
// @Produce  json
// @Param    id   path  string  true  "identifier"
// @Success  200  {object}  api.APIMessage
// @Router   /%[2]ss/{id} [delete]
func (h Handler) Delete(c echo.Context) error {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return fail(c, http.StatusBadRequest, "invalid id")
	}
	if err := h.service.Delete(c.Request().Context(), id); err != nil {
		return fail(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, api.APIMessage{Success: true, Message: "deleted"})
}

func fail(c echo.Context, status int, message string) error {
	return c.JSON(status, api.APIError{ErrorCode: http.StatusText(status), Message: message})
}

func totalPages(total int64, pageSize int) int64 {
	if pageSize <= 0 {
		return 0
	}
	return (total + int64(pageSize) - 1) / int64(pageSize)
}
`, module, domain, typeName, requestTypeName(typeName))
}

func kitEchoRouter(domain string) string {
	return fmt.Sprintf(`package %s

import "github.com/labstack/echo/v4"

// SetRoutes attaches the resource endpoints to the router group, applying any
// group-scoped middleware passed by the caller.
func SetRoutes(group *echo.Group, h Handler, middlewares ...echo.MiddlewareFunc) {
	group.Use(middlewares...)
	group.GET("", h.List)
	group.POST("", h.Create)
	group.GET("/:id", h.Get)
	group.PATCH("/:id", h.Update)
	group.DELETE("/:id", h.Delete)
}
`, domain)
}
