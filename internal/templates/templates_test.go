package templates_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/generator"
	"github.com/ongyoo/gokub/internal/manifest"
	"github.com/ongyoo/gokub/internal/templates"
)

func TestAddAndGenerate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOKUB_HOME", home)
	source := filepath.Join(t.TempDir(), "team-api")
	mustWrite(t, filepath.Join(source, "go.mod"), "module {{module}}\n\ngo {{go_version}}\n")
	mustWrite(t, filepath.Join(source, "cmd", "{{project_name}}", "main.go"), "package main\n")
	mustWrite(t, filepath.Join(source, "README.md"), "# {{project_name}}\n\nFramework: {{framework}}\n")
	mustWrite(t, filepath.Join(source, ".env"), "SECRET=do-not-copy\n")
	mustWrite(t, filepath.Join(source, ".env.example"), "PORT=8080\n")
	mustWrite(t, filepath.Join(source, ".git", "config"), "ignored\n")

	name, err := templates.Add("", source)
	if err != nil {
		t.Fatal(err)
	}
	if name != "team-api" {
		t.Fatalf("unexpected template name %q", name)
	}
	names, err := templates.Names()
	if err != nil || len(names) != 1 || names[0] != "team-api" {
		t.Fatalf("unexpected templates: %v, %v", names, err)
	}

	m := manifest.New("orders-api", "github.com/example/orders-api")
	m.Template = "team-api"
	output := t.TempDir()
	if err := generator.NewProject(output, m); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(output, m.Name)
	content, err := os.ReadFile(filepath.Join(project, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "module github.com/example/orders-api") || !strings.Contains(string(content), "go 1.26") {
		t.Fatalf("placeholder was not rendered: %s", content)
	}
	if _, err := os.Stat(filepath.Join(project, "cmd", "orders-api", "main.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".env")); !os.IsNotExist(err) {
		t.Fatal(".env must not be copied")
	}
	if _, err := os.Stat(filepath.Join(project, ".git")); !os.IsNotExist(err) {
		t.Fatal(".git must not be copied")
	}
	if _, err := os.Stat(filepath.Join(project, ".env.example")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project, ".codex", "config.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestRemove(t *testing.T) {
	t.Setenv("GOKUB_HOME", t.TempDir())
	source := filepath.Join(t.TempDir(), "custom")
	mustWrite(t, filepath.Join(source, "README.md"), "custom")
	if _, err := templates.Add("custom", source); err != nil {
		t.Fatal(err)
	}
	if err := templates.Remove("custom"); err != nil {
		t.Fatal(err)
	}
	if names, err := templates.Names(); err != nil || len(names) != 0 {
		t.Fatalf("template was not removed: %v, %v", names, err)
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
