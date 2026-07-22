package projectinit

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gokub "github.com/ongyoo/gokub"
	"github.com/ongyoo/gokub/internal/generator"
	"github.com/ongyoo/gokub/internal/goversion"
	"github.com/ongyoo/gokub/internal/manifest"
	"github.com/ongyoo/gokub/internal/projectmeta"
)

type Options struct {
	Provider     string
	Name         string
	Framework    string
	Database     string
	Messaging    string
	Architecture string
	Style        string
	Force        bool
}

type Result struct {
	Root            string
	Manifest        manifest.Manifest
	CreatedManifest bool
	Written         []string
}

// Initialize adopts an existing Go module without changing application source.
// Existing manifests and agent instruction files are preserved unless Force is
// explicitly enabled for agent files.
func Initialize(root string, options Options) (Result, error) {
	if !supportedProvider(options.Provider) {
		return Result{}, fmt.Errorf("unknown agent provider %q (choose: all, codex, claude, copilot, gemini, portable)", options.Provider)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return Result{}, err
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return Result{}, fmt.Errorf("open project: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("project path %s is not a directory", absRoot)
	}

	manifestPath := filepath.Join(absRoot, manifest.FileName)
	m, readErr := manifest.Read(manifestPath)
	created := false
	if readErr != nil && !os.IsNotExist(readErr) {
		return Result{}, fmt.Errorf("read project manifest: %w", readErr)
	}
	if os.IsNotExist(readErr) {
		m, err = inspect(absRoot, options)
		if err != nil {
			return Result{}, err
		}
		m.GeneratorVersion = gokub.Version
		if err := manifest.Write(manifestPath, m); err != nil {
			return Result{}, fmt.Errorf("write project manifest: %w", err)
		}
		created = true
	} else if applyOverrides(&m, options) {
		if err := manifest.Validate(m); err != nil {
			return Result{}, err
		}
		if err := manifest.Write(manifestPath, m); err != nil {
			return Result{}, fmt.Errorf("update project manifest: %w", err)
		}
	}

	provider := options.Provider
	if provider == "" {
		provider = "all"
	}
	written, err := generator.InitializeAgentFiles(absRoot, provider, options.Force)
	if err != nil {
		return Result{}, fmt.Errorf("install agent context: %w", err)
	}
	if err := projectmeta.WriteMarker(absRoot, gokub.Version, m); err != nil {
		return Result{}, fmt.Errorf("write %s: %w", projectmeta.MarkerFile, err)
	}
	written = append(written, projectmeta.MarkerFile)
	if created {
		written = append([]string{manifest.FileName}, written...)
	}
	return Result{Root: absRoot, Manifest: m, CreatedManifest: created, Written: written}, nil
}

func inspect(root string, options Options) (manifest.Manifest, error) {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return manifest.Manifest{}, fmt.Errorf("gokub init requires a Go module: %w", err)
	}
	module := modulePath(string(content))
	if module == "" {
		return manifest.Manifest{}, fmt.Errorf("go.mod does not contain a module directive")
	}
	goVersion := goversion.ParseGoMod(string(content))
	if goVersion == "" {
		return manifest.Manifest{}, fmt.Errorf("go.mod does not contain a Go version")
	}

	name := options.Name
	if name == "" {
		name = projectName(filepath.Base(root))
	}
	m := manifest.New(name, module)
	m.GoVersion = goVersion
	m.Template = "existing"
	m.Style = first(options.Style, detectStyle(root))
	m.Framework = first(options.Framework, detectFramework(string(content)))
	m.Database = first(options.Database, detectDatabase(string(content)))
	m.Architecture = first(options.Architecture, detectArchitecture(root))
	m.Messaging = first(options.Messaging, detectMessaging(string(content)))
	m.Agents = first(options.Provider, "all")
	m.Features = detectFeatures(root, m)
	m.Recipes = nil
	if err := manifest.Validate(m); err != nil {
		return manifest.Manifest{}, err
	}
	return m, nil
}

func modulePath(goMod string) string {
	scanner := bufio.NewScanner(strings.NewReader(goMod))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 2 && fields[0] == "module" {
			return strings.Trim(fields[1], `"`)
		}
	}
	return ""
}

func projectName(directory string) string {
	var name strings.Builder
	previousSeparator := false
	for _, char := range directory {
		valid := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '_' || char == '-'
		if valid {
			name.WriteRune(char)
			previousSeparator = false
			continue
		}
		if name.Len() > 0 && !previousSeparator {
			name.WriteByte('-')
			previousSeparator = true
		}
	}
	return strings.Trim(name.String(), "-_")
}

func detectFramework(goMod string) string {
	switch {
	case strings.Contains(goMod, "github.com/gin-gonic/gin"):
		return "gin"
	case strings.Contains(goMod, "github.com/gofiber/fiber/v2"):
		return "fiber-v2"
	case strings.Contains(goMod, "github.com/gofiber/fiber"):
		return "fiber"
	case strings.Contains(goMod, "github.com/labstack/echo"):
		return "echo"
	default:
		return "custom"
	}
}

func detectDatabase(goMod string) string {
	switch {
	case strings.Contains(goMod, "gorm.io/"):
		return "postgres"
	case strings.Contains(goMod, "github.com/jackc/pgx"), strings.Contains(goMod, "github.com/lib/pq"):
		return "pgx"
	case strings.Contains(goMod, "go.mongodb.org/mongo-driver"):
		return "mongodb"
	default:
		return "none"
	}
}

func detectMessaging(goMod string) string {
	return match(goMod, "none", map[string]string{
		"github.com/segmentio/kafka-go":  "kafka",
		"github.com/IBM/sarama":          "kafka",
		"github.com/rabbitmq/amqp091-go": "rabbitmq",
		"github.com/nats-io/nats.go":     "nats",
	})
}

func match(content, fallback string, candidates map[string]string) string {
	for dependency, value := range candidates {
		if strings.Contains(content, dependency) {
			return value
		}
	}
	return fallback
}

func detectStyle(root string) string {
	if directoryExists(filepath.Join(root, "services")) {
		return "microservices"
	}
	entries, err := os.ReadDir(filepath.Join(root, "cmd"))
	if err == nil {
		directories := 0
		for _, entry := range entries {
			if entry.IsDir() {
				directories++
			}
		}
		if directories > 1 {
			return "microservices"
		}
	}
	return "monolith"
}

func detectArchitecture(root string) string {
	if directoryExists(filepath.Join(root, "internal", "ports")) || directoryExists(filepath.Join(root, "internal", "adapters")) {
		return "hexagonal"
	}
	if directoryExists(filepath.Join(root, "internal", "domain")) || directoryExists(filepath.Join(root, "internal", "application")) {
		return "clean"
	}
	return "layered"
}

func detectFeatures(root string, m manifest.Manifest) []string {
	features := []string{}
	for _, value := range []string{m.Database, m.Messaging} {
		if value != "" && value != "none" {
			features = append(features, value)
		}
	}
	if fileExists(filepath.Join(root, "Dockerfile")) || fileExists(filepath.Join(root, "docker-compose.yml")) || fileExists(filepath.Join(root, "compose.yaml")) {
		features = append(features, "docker")
	}
	if directoryExists(filepath.Join(root, ".github", "workflows")) {
		features = append(features, "github-actions")
	}
	return features
}

func first(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func supportedProvider(provider string) bool {
	switch provider {
	case "", "all", "codex", "claude", "copilot", "gemini", "portable":
		return true
	default:
		return false
	}
}

func applyOverrides(m *manifest.Manifest, options Options) bool {
	changed := false
	overrides := []struct {
		value  string
		target *string
	}{
		{options.Name, &m.Name},
		{options.Framework, &m.Framework},
		{options.Database, &m.Database},
		{options.Messaging, &m.Messaging},
		{options.Architecture, &m.Architecture},
		{options.Style, &m.Style},
	}
	for _, override := range overrides {
		if override.value != "" && *override.target != override.value {
			*override.target = override.value
			changed = true
		}
	}
	return changed
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
