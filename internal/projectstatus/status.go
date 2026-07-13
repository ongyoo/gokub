package projectstatus

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ongyoo/gokub/internal/catalog"
	"github.com/ongyoo/gokub/internal/goversion"
	"github.com/ongyoo/gokub/internal/manifest"
)

type Project struct {
	Name         string `json:"name"`
	Module       string `json:"module"`
	Template     string `json:"template"`
	Style        string `json:"style"`
	Framework    string `json:"framework"`
	Database     string `json:"database"`
	Architecture string `json:"architecture"`
	Messaging    string `json:"messaging"`
	GoVersion    string `json:"go_version"`
	GoSupport    string `json:"go_support"`
	GoGuidance   string `json:"go_guidance"`
}

type Capability struct {
	Name               string   `json:"name"`
	Enabled            bool     `json:"enabled"`
	Providers          []string `json:"providers"`
	AvailableProviders []string `json:"available_providers"`
}

type Report struct {
	SchemaVersion    int          `json:"schema_version"`
	GeneratorVersion string       `json:"generator_version"`
	Project          Project      `json:"project"`
	Capabilities     []Capability `json:"capabilities"`
	Features         []string     `json:"features"`
	Recipes          []string     `json:"recipes"`
}

func Build(root string) (Report, error) {
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return Report{}, fmt.Errorf("read project manifest: %w", err)
	}
	goVersion := m.GoVersion
	if goVersion == "" {
		if content, readErr := os.ReadFile(filepath.Join(root, "go.mod")); readErr == nil {
			goVersion = goversion.ParseGoMod(string(content))
		}
	}
	goSupport := "unknown"
	goGuidance := "run gokub upgrade to record the project Go version"
	if goVersion != "" {
		goSupport = string(goversion.Classify(goVersion))
		goGuidance = goversion.Description(goVersion)
	}
	report := Report{
		SchemaVersion: m.SchemaVersion, GeneratorVersion: m.GeneratorVersion,
		Project: Project{Name: m.Name, Module: m.Module, Template: m.Template, Style: m.Style,
			Framework: m.Framework, Database: m.Database, Architecture: m.Architecture, Messaging: m.Messaging,
			GoVersion: goVersion, GoSupport: goSupport, GoGuidance: goGuidance},
		Features: append([]string(nil), m.Features...), Recipes: append([]string(nil), m.Recipes...),
	}
	if report.GeneratorVersion == "" {
		report.GeneratorVersion = "unversioned"
	}
	for _, name := range catalog.CapabilityNames() {
		definition := catalog.Capabilities[name]
		providers := enabledProviders(m, name, definition.Providers)
		report.Capabilities = append(report.Capabilities, Capability{
			Name: name, Enabled: len(providers) > 0, Providers: providers,
			AvailableProviders: append([]string(nil), definition.Providers...),
		})
	}
	return report, nil
}

func enabledProviders(m manifest.Manifest, capability string, providers []string) []string {
	enabled := []string{}
	for _, provider := range providers {
		if contains(m.Features, provider) {
			enabled = append(enabled, provider)
		}
	}
	selected := ""
	switch capability {
	case "database":
		selected = m.Database
	case "messaging":
		selected = m.Messaging
	}
	if selected != "" && selected != "none" && contains(providers, selected) && !contains(enabled, selected) {
		enabled = append(enabled, selected)
	}
	return enabled
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
