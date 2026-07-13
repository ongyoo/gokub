package projectupgrade

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/manifest"
)

func TestCheckAndApplyLegacyManifest(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, manifest.FileName)
	legacy := `project:
  name: service
  module: example.com/service
  template: monolith
  style: monolith
  framework: gin
  architecture: clean
database: postgres
messaging: none
security: asvs-l2
features:
  - docker
recipes:
`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/service\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := Check(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if !plan.NeedsUpgrade || plan.CurrentSchema != 0 || len(plan.Changes) != 3 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	result, err := Apply(root, "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if result.BackupPath == "" {
		t.Fatal("upgrade did not create a backup")
	}
	backup, _ := os.ReadFile(result.BackupPath)
	if string(backup) != legacy {
		t.Fatal("backup does not preserve the original manifest")
	}
	upgraded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(upgraded), "schema_version: 2") || !strings.Contains(string(upgraded), "generator_version: 1.2.3") || !strings.Contains(string(upgraded), "go_version: 1.24") {
		t.Fatalf("manifest metadata not upgraded:\n%s", upgraded)
	}
	current, err := Check(root, "1.2.3")
	if err != nil || current.NeedsUpgrade {
		t.Fatalf("upgrade is not idempotent: %+v %v", current, err)
	}
}

func TestCheckRejectsFutureSchema(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, manifest.FileName)
	content := "schema_version: 3\nproject:\n  name: service\n  module: example.com/service\n  go_version: 1.26\n  template: monolith\n  style: monolith\n  framework: gin\n  architecture: clean\ndatabase: none\nmessaging: none\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(root, "1.2.3"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Fatalf("future schema was accepted: %v", err)
	}
}
