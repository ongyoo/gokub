package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/manifest"
)

func TestScoreCompleteProject(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("service", "example.com/service")
	writeScoreFixture(t, root, manifest.FileName, "")
	if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod":                                "module example.com/service\n",
		"cmd/service/main.go":                   "package main\n// signal.NotifyContext log/slog\n",
		"internal/http/server.go":               "package http\n// /health/live X-Content-Type-Options\n",
		"internal/domain/example/model.go":      "package example\n",
		"internal/platform/postgres/store.go":   "package postgres\n",
		"internal/domain/example/model_test.go": "package example\n",
		".env.example":                          "PORT=8080\n",
		".gitignore":                            ".env\n",
		"Dockerfile":                            "FROM scratch\nUSER 65532\n",
		".github/workflows/ci.yml":              "go vet ./...\ngo test -race ./...\n",
	}
	for path, content := range files {
		writeScoreFixture(t, root, path, content)
	}
	for _, dir := range []string{"tests", "deployments"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	report := Score(root)
	if report.Score != 100 || report.Grade != "Excellent" {
		t.Fatalf("unexpected score: %+v", report)
	}
}

func TestScoreEmptyDirectoryReturnsActionableReport(t *testing.T) {
	report := Score(t.TempDir())
	if report.Score != 0 || report.Max != 100 || report.Grade != "At risk" {
		t.Fatalf("unexpected empty score: %+v", report)
	}
	if len(report.Categories) != 4 || report.Categories[0].Checks[0].Recommendation == "" {
		t.Fatalf("report is not actionable: %+v", report)
	}
}

func TestScorePenalizesArchitectureViolation(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("service", "example.com/service")
	if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
		t.Fatal(err)
	}
	writeScoreFixture(t, root, "internal/domain/order/service.go", "package order\nimport _ \"example.com/service/internal/platform/postgres\"\n")
	writeScoreFixture(t, root, "internal/platform/postgres/store.go", "package postgres\n")
	report := Score(root)
	found := false
	for _, check := range report.Categories[0].Checks {
		if check.Name == "Clean dependencies" {
			found = true
			if check.OK || check.Points != 0 || !strings.Contains(check.Recommendation, "graph --check") {
				t.Fatalf("architecture violation was not actionable: %+v", check)
			}
		}
	}
	if !found || report.Max != 100 {
		t.Fatalf("architecture score contract changed unexpectedly: %+v", report)
	}
}

func writeScoreFixture(t *testing.T, root, path, content string) {
	t.Helper()
	fullPath := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
