package projectinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/manifest"
)

func TestInitializeExistingProject(t *testing.T) {
	root := filepath.Join(t.TempDir(), "orders-api")
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	goMod := `module github.com/example/orders

go 1.24

require (
	github.com/gofiber/fiber/v3 v3.0.0
	go.mongodb.org/mongo-driver v1.17.0
	github.com/nats-io/nats.go v1.40.0
)
`
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("# Team rules\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Initialize(root, Options{Provider: "all"})
	if err != nil {
		t.Fatal(err)
	}
	if !result.CreatedManifest {
		t.Fatal("expected a new manifest")
	}
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if m.Template != "existing" || m.Framework != "fiber" || m.Database != "mongodb" || m.Messaging != "nats" {
		t.Fatalf("unexpected detection: %+v", m)
	}
	for _, path := range []string{
		"gokub.init", ".codex/config.toml", ".mcp.json",
		".agents/skills/gokub-project/SKILL.md",
		".claude/skills/gokub-add-domain/SKILL.md",
		".github/skills/gokub-verify-change/SKILL.md",
	} {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatalf("missing %s: %v", path, err)
		}
	}
	agents, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil || string(agents) != "# Team rules\n" {
		t.Fatalf("existing AGENTS.md was replaced: %q, %v", agents, err)
	}
	marker, err := os.ReadFile(filepath.Join(root, "gokub.init"))
	if err != nil || !strings.Contains(string(marker), "module: github.com/example/orders") {
		t.Fatalf("unexpected marker: %s, %v", marker, err)
	}
}

func TestInitializeIsIdempotentAndAppliesOverrides(t *testing.T) {
	root := filepath.Join(t.TempDir(), "api")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/api\n\ngo 1.23\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Initialize(root, Options{}); err != nil {
		t.Fatal(err)
	}
	result, err := Initialize(root, Options{Framework: "gin", Database: "postgres"})
	if err != nil {
		t.Fatal(err)
	}
	if result.CreatedManifest || result.Manifest.Framework != "gin" || result.Manifest.Database != "postgres" {
		t.Fatalf("unexpected reinitialize result: %+v", result)
	}
}

func TestInitializeRejectsNonGoProject(t *testing.T) {
	if _, err := Initialize(t.TempDir(), Options{}); err == nil || !strings.Contains(err.Error(), "requires a Go module") {
		t.Fatalf("expected Go module error, got %v", err)
	}
}

func TestProjectNameNormalizesExistingFolder(t *testing.T) {
	if got := projectName("orders.api v2"); got != "orders-api-v2" {
		t.Fatalf("unexpected normalized name %q", got)
	}
}
