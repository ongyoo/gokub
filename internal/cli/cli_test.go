package cli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokub/gokub/internal/generator"
	"github.com/gokub/gokub/internal/manifest"
)

func TestNewParsesFlagsAfterProjectName(t *testing.T) {
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})

	var out bytes.Buffer
	err = Run([]string{
		"new",
		"payment-api",
		"--recipe", "event-driven",
		"--module", "github.com/acme/payment-api",
	}, bytes.NewBuffer(nil), &out, &out)
	if err != nil {
		t.Fatal(err)
	}

	goMod, err := os.ReadFile(filepath.Join(temp, "payment-api", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(goMod) != "module github.com/acme/payment-api\n\ngo 1.24\n" {
		t.Fatalf("unexpected go.mod:\n%s", goMod)
	}

	if _, err := os.Stat(filepath.Join(temp, "payment-api", "docs", "gokub_logo.png")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "CLAUDE.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "postgres", "postgres.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", ".vscode", "launch.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", ".run", "GOKUB.run.xml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", ".codex", "config.toml")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", ".mcp.json")); err != nil {
		t.Fatal(err)
	}
}

func TestNewScaffoldsSelectedProviders(t *testing.T) {
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	err = Run([]string{"new", "events", "--database", "mongodb", "--messaging", "nats"}, bytes.NewBuffer(nil), &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"internal/mongodb/mongodb.go", "internal/nats/nats.go"} {
		if _, err := os.Stat(filepath.Join(temp, "events", path)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestNewWizardCreatesProjectFromStepAnswers(t *testing.T) {
	t.Setenv("GOKUB_SKIP_INSTALL", "1")
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})

	input := bytes.NewBufferString(strings.Join([]string{
		"payment-api",
		"github.com/ongyoo/payment-api",
		"1",
		"1",
		"1",
		"1",
		"1",
		"2",
		"3",
		"",
	}, "\n"))
	var out bytes.Buffer
	err = Run([]string{"new"}, input, &out, &out)
	if err != nil {
		t.Fatal(err)
	}

	goMod, err := os.ReadFile(filepath.Join(temp, "payment-api", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "module github.com/ongyoo/payment-api\n") {
		t.Fatalf("unexpected go.mod:\n%s", goMod)
	}

	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "platform", "messaging", "kafka", "kafka.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "outbox", "outbox.go")); err != nil {
		t.Fatal(err)
	}
}

func TestNewWizardUsesExampleAPIDefault(t *testing.T) {
	t.Setenv("GOKUB_SKIP_INSTALL", "1")
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	input := bytes.NewBufferString(strings.Repeat("\n", 8))
	if err := Run([]string{"new"}, input, &out, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "example-api", ".gokub.yaml")); err != nil {
		t.Fatal(err)
	}
	goMod, err := os.ReadFile(filepath.Join(temp, "example-api", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "module github.com/example/example-api\n") {
		t.Fatalf("unexpected default go.mod:\n%s", goMod)
	}
}

func TestEnableAndSwitchCapabilityProvider(t *testing.T) {
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previous)
	})

	var out bytes.Buffer
	if err := Run([]string{"new", "payment-api", "--module", "github.com/acme/payment-api"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(filepath.Join(temp, "payment-api")); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"enable", "messaging", "kafka"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "kafka", "kafka.go")); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"switch", "messaging", "rabbitmq"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "rabbitmq", "rabbitmq.go")); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(temp, "payment-api", ".gokub.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "messaging: rabbitmq") {
		t.Fatalf("manifest did not switch messaging:\n%s", text)
	}
	if strings.Contains(text, "  - kafka\n") {
		t.Fatalf("manifest still contains kafka:\n%s", text)
	}
	if !strings.Contains(text, "  - rabbitmq\n") {
		t.Fatalf("manifest missing rabbitmq:\n%s", text)
	}
	out.Reset()
	if err := Run([]string{"status"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "messaging") || !strings.Contains(out.String(), "rabbitmq") {
		t.Fatalf("status did not show enabled messaging provider:\n%s", out.String())
	}
	if err := Run([]string{"disable", "messaging"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(filepath.Join(temp, "payment-api", ".gokub.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	text = string(content)
	if !strings.Contains(text, "messaging: none") {
		t.Fatalf("manifest did not disable messaging:\n%s", text)
	}
	if strings.Contains(text, "  - rabbitmq\n") {
		t.Fatalf("manifest still contains rabbitmq:\n%s", text)
	}
}

func TestNewRejectsInvalidOptionsAndExistingTarget(t *testing.T) {
	temp := t.TempDir()
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	if err := Run([]string{"new", "../escape"}, bytes.NewBuffer(nil), &out, &out); err == nil {
		t.Fatal("expected invalid project name to fail")
	}
	if err := Run([]string{"new", "service", "--framework", "unknown"}, bytes.NewBuffer(nil), &out, &out); err == nil {
		t.Fatal("expected invalid framework to fail")
	}
	if err := Run([]string{"new", "service"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"new", "service"}, bytes.NewBuffer(nil), &out, &out); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing target error, got %v", err)
	}
}

func TestGeneratedProjectTestsPass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping generated project build in short mode")
	}
	temp := t.TempDir()
	m := manifest.New("service", "example.com/service")
	if err := generator.NewProject(temp, m); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = filepath.Join(temp, "service")
	command.Env = append(os.Environ(), "GOCACHE="+filepath.Join(temp, "go-cache"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated project tests failed: %v\n%s", err, output)
	}
}

func TestMCPServeDoesNotWriteTerminalLogo(t *testing.T) {
	input := bytes.NewBufferString(`{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n")
	var out bytes.Buffer
	if err := Run([]string{"mcp", "serve"}, input, &out, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "GOKUB") || !strings.HasPrefix(out.String(), "{") {
		t.Fatalf("MCP stdout contains terminal output: %q", out.String())
	}
}

func TestRemoveExecutableSafety(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gokub")
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := removeExecutable(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("executable was not removed")
	}
	unexpected := filepath.Join(t.TempDir(), "application")
	if err := os.WriteFile(unexpected, []byte("keep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := removeExecutable(unexpected); err == nil {
		t.Fatal("unexpected executable name was accepted")
	}
}

func TestTemplateCommandInstallsAndGenerates(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("GOKUB_HOME", filepath.Join(temp, "gokub-home"))
	source := filepath.Join(temp, "team-api")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "go.mod"), []byte("module {{module}}\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(temp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	if err := Run([]string{"template", "add", source}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"new", "custom-api", "--module", "github.com/example/custom-api", "--template", "team-api"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(temp, "custom-api", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "module github.com/example/custom-api") {
		t.Fatalf("custom template was not rendered: %s", content)
	}
}
