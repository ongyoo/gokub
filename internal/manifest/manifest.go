package manifest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ongyoo/gokub/internal/goversion"
)

const (
	FileName             = ".gokub.yaml"
	CurrentSchemaVersion = 2
)

var projectNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

type Manifest struct {
	SchemaVersion    int
	GeneratorVersion string
	Name             string
	Module           string
	GoVersion        string
	Template         string
	Style            string
	Framework        string
	Database         string
	Architecture     string
	Messaging        string
	Agents           string
	Recipes          []string
	Features         []string
}

func New(name, module string) Manifest {
	if module == "" {
		module = name
	}
	return Manifest{
		SchemaVersion: CurrentSchemaVersion,
		Name:          name,
		Module:        module,
		GoVersion:     goversion.Recommended,
		Template:      "gin-clean",
		Style:         "monolith",
		Framework:     "gin",
		Database:      "postgres",
		Architecture:  "clean",
		Messaging:     "none",
		Agents:        "all",
		Features:      []string{"docker", "github-actions"},
		Recipes:       []string{},
	}
}

func Validate(m Manifest) error {
	if m.SchemaVersion < 0 || m.SchemaVersion > CurrentSchemaVersion {
		return fmt.Errorf("unsupported manifest schema version %d (CLI supports up to %d)", m.SchemaVersion, CurrentSchemaVersion)
	}
	if strings.ContainsAny(m.GeneratorVersion, "\r\n:") {
		return fmt.Errorf("invalid generator version %q", m.GeneratorVersion)
	}
	if !projectNamePattern.MatchString(m.Name) || m.Name == "." || m.Name == ".." {
		return fmt.Errorf("project name %q must contain only letters, numbers, hyphens, or underscores", m.Name)
	}
	if strings.TrimSpace(m.Module) == "" || strings.ContainsAny(m.Module, " \t\r\n") {
		return fmt.Errorf("module %q must be a non-empty Go module path without spaces", m.Module)
	}
	if m.GoVersion == "" {
		if m.SchemaVersion >= 2 {
			return fmt.Errorf("go_version is required for manifest schema %d", m.SchemaVersion)
		}
	} else if err := goversion.Validate(m.GoVersion); err != nil {
		return err
	}
	for field, value := range map[string]string{
		"template": m.Template, "style": m.Style, "framework": m.Framework, "database": m.Database,
		"architecture": m.Architecture, "messaging": m.Messaging,
	} {
		if value == "" || strings.ContainsAny(value, "\r\n:") {
			return fmt.Errorf("invalid %s %q", field, value)
		}
	}
	return nil
}

func Read(path string) (Manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return Manifest{}, err
	}
	defer f.Close()

	m := Manifest{}
	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") && strings.HasSuffix(trimmed, ":") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			value := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			switch section {
			case "features":
				m.Features = append(m.Features, value)
			case "recipes":
				m.Recipes = append(m.Recipes, value)
			}
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), `"`)
		switch key {
		case "schema_version":
			if parsed, err := strconv.Atoi(value); err == nil {
				m.SchemaVersion = parsed
			}
		case "generator_version":
			m.GeneratorVersion = value
		case "name":
			m.Name = value
		case "module":
			m.Module = value
		case "go_version":
			m.GoVersion = value
		case "template":
			m.Template = value
		case "style":
			m.Style = value
		case "framework":
			m.Framework = value
		case "database":
			m.Database = value
		case "architecture":
			m.Architecture = value
		case "messaging":
			m.Messaging = value
		case "agents":
			m.Agents = value
		}
	}
	if m.Style == "" {
		m.Style = "monolith"
	}
	if m.Agents == "" {
		m.Agents = "all"
	}
	return m, scanner.Err()
}

func Write(path string, m Manifest) error {
	if err := Validate(m); err != nil {
		return err
	}
	m.Features = uniqueSorted(m.Features)
	m.Recipes = uniqueSorted(m.Recipes)
	content := &strings.Builder{}
	if m.SchemaVersion > 0 {
		fmt.Fprintf(content, "schema_version: %d\n", m.SchemaVersion)
	}
	if m.GeneratorVersion != "" {
		fmt.Fprintf(content, "generator_version: %s\n", m.GeneratorVersion)
	}
	fmt.Fprintln(content, "project:")
	fmt.Fprintf(content, "  name: %s\n", m.Name)
	fmt.Fprintf(content, "  module: %s\n", m.Module)
	if m.GoVersion != "" {
		fmt.Fprintf(content, "  go_version: %s\n", m.GoVersion)
	}
	fmt.Fprintf(content, "  template: %s\n", m.Template)
	fmt.Fprintf(content, "  style: %s\n", m.Style)
	fmt.Fprintf(content, "  framework: %s\n", m.Framework)
	fmt.Fprintf(content, "  architecture: %s\n", m.Architecture)
	if m.Agents != "" {
		fmt.Fprintf(content, "  agents: %s\n", m.Agents)
	}
	fmt.Fprintf(content, "database: %s\n", m.Database)
	fmt.Fprintf(content, "messaging: %s\n", m.Messaging)
	fmt.Fprintln(content, "security: asvs-l2")
	fmt.Fprintln(content, "features:")
	for _, feature := range m.Features {
		fmt.Fprintf(content, "  - %s\n", feature)
	}
	fmt.Fprintln(content, "recipes:")
	for _, recipe := range m.Recipes {
		fmt.Fprintf(content, "  - %s\n", recipe)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content.String()), 0o644)
}

func AddFeature(m *Manifest, name string) {
	m.Features = append(m.Features, name)
	m.Features = uniqueSorted(m.Features)
}

func RemoveFeature(m *Manifest, name string) {
	next := m.Features[:0]
	for _, feature := range m.Features {
		if feature != name {
			next = append(next, feature)
		}
	}
	m.Features = next
}

func RemoveFeatures(m *Manifest, names []string) {
	remove := map[string]bool{}
	for _, name := range names {
		remove[name] = true
	}
	next := m.Features[:0]
	for _, feature := range m.Features {
		if !remove[feature] {
			next = append(next, feature)
		}
	}
	m.Features = next
}

func AddRecipe(m *Manifest, name string) {
	m.Recipes = append(m.Recipes, name)
	m.Recipes = uniqueSorted(m.Recipes)
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
