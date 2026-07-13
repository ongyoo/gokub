package projectstatus

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ongyoo/gokub/internal/manifest"
)

func TestBuildReportsCapabilitiesWithoutSecrets(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("service", "example.com/service")
	m.Features = append(m.Features, "kafka")
	if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".env"), []byte("SECRET=do-not-read\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Project.Name != "service" || report.Project.GoVersion != "1.26" || report.Project.GoSupport != "latest" || len(report.Capabilities) == 0 {
		t.Fatalf("incomplete report: %+v", report)
	}
	found := false
	databaseFound := false
	for _, capability := range report.Capabilities {
		if capability.Name == "messaging" && capability.Enabled && len(capability.Providers) == 1 && capability.Providers[0] == "kafka" {
			found = true
		}
		if capability.Name == "database" && capability.Enabled && contains(capability.Providers, "postgres") {
			databaseFound = true
		}
	}
	if !found || !databaseFound {
		t.Fatalf("enabled messaging was not reported: %+v", report.Capabilities)
	}
}

func TestBuildInfersLegacyGoVersion(t *testing.T) {
	root := t.TempDir()
	legacy := "project:\n  name: service\n  module: example.com/service\n  template: monolith\n  style: monolith\n  framework: gin\n  architecture: clean\ndatabase: none\nmessaging: none\n"
	if err := os.WriteFile(filepath.Join(root, manifest.FileName), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/service\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if report.Project.GoVersion != "1.24" || report.Project.GoSupport != "unsupported" {
		t.Fatalf("legacy Go version was not inferred: %+v", report.Project)
	}
}
