package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/doctor"
	"github.com/ongyoo/gokub/internal/generator"
	"github.com/ongyoo/gokub/internal/manifest"
	"github.com/ongyoo/gokub/internal/projectgraph"
	"github.com/ongyoo/gokub/internal/projectstatus"
)

func TestNewParsesFlagsAfterProjectName(t *testing.T) {
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

	var out bytes.Buffer
	err = Run([]string{
		"new",
		"payment-api",
		"--recipe", "event-driven",
		"--module", "github.com/acme/payment-api",
		"--go-version", "1.25",
	}, bytes.NewBuffer(nil), &out, &out)
	if err != nil {
		t.Fatal(err)
	}

	goMod, err := os.ReadFile(filepath.Join(temp, "payment-api", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if string(goMod) != "module github.com/acme/payment-api\n\ngo 1.25\n" {
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
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "cmd", "payment-api-service", "main.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "example", "service.go")); err != nil {
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
	workflow, err := os.ReadFile(filepath.Join(temp, "payment-api", ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(workflow), `go-version: "1.25.x"`) {
		t.Fatalf("classic template CI does not match go.mod:\n%s", workflow)
	}
}

func TestNoArgsRemainsNonInteractiveForPipesAndCI(t *testing.T) {
	var out bytes.Buffer
	if err := Run(nil, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Commands") || !strings.Contains(out.String(), "start the step-by-step project wizard") {
		t.Fatalf("non-interactive usage missing: %s", out.String())
	}
}

func TestHomebrewExecutableDetection(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Cellar", "gokub", "0.2.0", "bin", "gokub")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "bin", "gokub")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if !homebrewExecutable(link) {
		t.Fatal("Homebrew symlink was not detected")
	}
	if homebrewExecutable(filepath.Join(root, "go", "bin", "gokub")) {
		t.Fatal("regular Go installation was detected as Homebrew")
	}
}

func TestCommandCenterActionsAreProjectAware(t *testing.T) {
	outside := strings.Join(commandCenterActions(false), ",")
	inside := strings.Join(commandCenterActions(true), ",")
	if strings.Contains(outside, "Doctor") || !strings.Contains(outside, "New project") {
		t.Fatalf("unexpected outside-project actions: %s", outside)
	}
	for _, action := range []string{"Add feature", "Generate model from JSON", "Doctor", "Project score", "Upgrade project"} {
		if !strings.Contains(inside, action) {
			t.Fatalf("project action %q missing from %s", action, inside)
		}
	}
}

func TestNewScaffoldsSelectedProviders(t *testing.T) {
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
	err = Run([]string{"new", "events", "--framework", "fiber", "--database", "mongodb", "--messaging", "nats"}, bytes.NewBuffer(nil), &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"cmd/events-service/main.go",
		"internal/example/service.go",
		"pkg/httpserver/fiber/http.go",
		"pkg/middleware/fiber/middleware.go",
	} {
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

	if _, err := os.Stat(filepath.Join(temp, "payment-api", "cmd", "payment-api-service", "main.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "example", "router.go")); err != nil {
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
	if !strings.Contains(string(goMod), "go 1.26\n") {
		t.Fatalf("default project does not use the recommended Go version:\n%s", goMod)
	}
}

func TestEnableAndSwitchCapabilityProvider(t *testing.T) {
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
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "app", "events", "bus_kafka.go")); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{"switch", "messaging", "rabbitmq"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "app", "events", "bus_rabbitmq.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(temp, "payment-api", "internal", "app", "events", "bus_kafka.go")); err == nil {
		t.Fatal("switch did not remove the previous kafka bus file")
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
	if err := Run([]string{"new", "service", "--go-version", "1.26.1"}, bytes.NewBuffer(nil), &out, &out); err == nil {
		t.Fatal("expected patch-level Go version to fail")
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
	m.GoVersion = "1.25"
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

func TestScoreJSONDoesNotWriteTerminalLogo(t *testing.T) {
	var out bytes.Buffer
	if err := Run([]string{"score", "--json"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	var report struct {
		Max int `json:"max"`
	}
	if !strings.HasPrefix(out.String(), "{") || json.Unmarshal(out.Bytes(), &report) != nil || report.Max != 100 {
		t.Fatalf("score output is not a clean JSON report: %q", out.String())
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

func TestAddModelGeneratesFromJSON(t *testing.T) {
	temp := t.TempDir()
	project := filepath.Join(temp, "service")
	if err := os.MkdirAll(filepath.Join(project, "internal", "domain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Write(filepath.Join(project, manifest.FileName), manifest.New("service", "example.com/service")); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(project, "user.json")
	if err := os.WriteFile(input, []byte(`{"id":1,"name":"Ada"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	if err := Run([]string{"add", "model", "user", "--from", "user.json"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(filepath.Join(project, "internal", "domain", "user", "model_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(generated), "type User struct") || !strings.Contains(out.String(), "generated") {
		t.Fatalf("model command produced unexpected output:\n%s\n%s", generated, out.String())
	}
}

func TestAddModelDiscoversJSONAndSuggestsName(t *testing.T) {
	temp := t.TempDir()
	project := filepath.Join(temp, "service")
	if err := os.MkdirAll(filepath.Join(project, "internal", "domain"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := manifest.Write(filepath.Join(project, manifest.FileName), manifest.New("service", "example.com/service")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "profile.schema.json"), []byte(`{"type":"object","properties":{"id":{"type":"string"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "other.json"), []byte(`{"ignored":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	if err := Run([]string{"add", "model"}, bytes.NewBufferString("1\n\n"), &out, &out); err != nil {
		t.Fatal(err)
	}
	generated := filepath.Join(project, "internal", "domain", "profile", "model_gen.go")
	content, err := os.ReadFile(generated)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "type Profile struct") || !strings.Contains(out.String(), "profile.schema.json") {
		t.Fatalf("interactive model flow failed:\n%s\n%s", content, out.String())
	}
}

func TestUpgradeJSONPlanAndApply(t *testing.T) {
	project := t.TempDir()
	m := manifest.New("service", "example.com/service")
	m.SchemaVersion = 0
	m.GeneratorVersion = ""
	if err := manifest.Write(filepath.Join(project, manifest.FileName), m); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	if err := Run([]string{"upgrade", "--json"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "{") || !strings.Contains(out.String(), `"needs_upgrade":true`) {
		t.Fatalf("upgrade JSON is not clean or actionable: %s", out.String())
	}
	out.Reset()
	if err := Run([]string{"upgrade", "--yes"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	upgraded, err := manifest.Read(filepath.Join(project, manifest.FileName))
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.SchemaVersion != manifest.CurrentSchemaVersion || upgraded.GeneratorVersion == "" {
		t.Fatalf("manifest not upgraded: %+v", upgraded)
	}
	if _, err := os.Stat(filepath.Join(project, manifest.FileName+".bak")); err != nil {
		t.Fatal(err)
	}
}

func TestCompletionScriptHasNoLogo(t *testing.T) {
	var out bytes.Buffer
	if err := Run([]string{"completion", "zsh"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out.String(), "#compdef gokub") {
		t.Fatalf("completion output was polluted:\n%s", out.String())
	}
}

func TestCompletionInstallUsesSelectedShell(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var out bytes.Buffer
	if err := Run([]string{"completion", "install", "fish"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "fish", "completions", "gokub.fish")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "installed shell completion") {
		t.Fatalf("missing install confirmation: %s", out.String())
	}
}

func TestDoctorJSONIsCleanAndSummarized(t *testing.T) {
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
	err = Run([]string{"doctor", "--json"}, bytes.NewBuffer(nil), &out, &out)
	if err == nil {
		t.Fatal("expected empty project to fail doctor")
	}
	var report doctor.Report
	if decodeErr := json.Unmarshal(out.Bytes(), &report); decodeErr != nil {
		t.Fatalf("doctor JSON was polluted: %v\n%s", decodeErr, out.String())
	}
	if report.OK || report.Total == 0 || report.Failed == 0 {
		t.Fatalf("unexpected doctor report: %+v", report)
	}
}

func TestScoreFailUnderEnforcesThresholdAndKeepsJSONClean(t *testing.T) {
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
	err = Run([]string{"score", "--json", "--fail-under", "80"}, bytes.NewBuffer(nil), &out, &out)
	if err == nil || !strings.Contains(err.Error(), "below required threshold 80") {
		t.Fatalf("expected threshold failure, got %v", err)
	}
	var report doctor.ScoreReport
	if decodeErr := json.Unmarshal(out.Bytes(), &report); decodeErr != nil {
		t.Fatalf("score JSON was polluted: %v\n%s", decodeErr, out.String())
	}
	if report.Score >= 80 {
		t.Fatalf("fixture unexpectedly passed threshold: %+v", report)
	}
	if err := scoreThresholdError(80, 80); err != nil {
		t.Fatalf("equal threshold should pass: %v", err)
	}
	if err := scoreThresholdError(79, 101); err == nil {
		t.Fatal("direct threshold helper should fail below threshold")
	}
}

func TestScoreRejectsInvalidThreshold(t *testing.T) {
	for _, value := range []string{"-1", "101"} {
		var out bytes.Buffer
		err := Run([]string{"score", "--fail-under", value}, bytes.NewBuffer(nil), &out, &out)
		if err == nil || !strings.Contains(err.Error(), "between 0 and 100") {
			t.Fatalf("threshold %s returned %v", value, err)
		}
	}
}

func TestVersionJSONIsCleanAndComplete(t *testing.T) {
	var out bytes.Buffer
	if err := Run([]string{"version", "--json"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	var info versionInfo
	if err := json.Unmarshal(out.Bytes(), &info); err != nil {
		t.Fatalf("version JSON was polluted: %v\n%s", err, out.String())
	}
	if info.CLI != "gokub" || info.Version == "" || info.Repository == "" || info.GoVersion == "" || info.OS == "" || info.Arch == "" {
		t.Fatalf("incomplete version metadata: %+v", info)
	}
}

func TestStatusJSONIsCleanAndReportsCapabilities(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("service", "example.com/service")
	m.Features = append(m.Features, "kafka")
	if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
		t.Fatal(err)
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	if err := Run([]string{"status", "--json"}, bytes.NewBuffer(nil), &out, &out); err != nil {
		t.Fatal(err)
	}
	var report projectstatus.Report
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("status JSON was polluted: %v\n%s", err, out.String())
	}
	if report.Project.Name != "service" || len(report.Capabilities) == 0 || !strings.HasPrefix(out.String(), "{") {
		t.Fatalf("incomplete status report: %+v", report)
	}
}

func TestGraphCheckJSONFailsOnBoundaryViolation(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("service", "example.com/service")
	if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"internal/domain/order/service.go":    "package order\nimport _ \"example.com/service/internal/platform/postgres\"\n",
		"internal/platform/postgres/store.go": "package postgres\n",
	}
	for path, content := range files {
		fullPath := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(previous) })

	var out bytes.Buffer
	err = Run([]string{"graph", "--check", "--format", "json"}, bytes.NewBuffer(nil), &out, &out)
	if err == nil || !strings.Contains(err.Error(), "architecture violation") {
		t.Fatalf("expected architecture gate failure, got %v", err)
	}
	var analysis projectgraph.Analysis
	if decodeErr := json.Unmarshal(out.Bytes(), &analysis); decodeErr != nil {
		t.Fatalf("graph check JSON was polluted: %v\n%s", decodeErr, out.String())
	}
	if analysis.OK || len(analysis.Violations) != 1 || analysis.Violations[0].Type != "boundary" {
		t.Fatalf("unexpected graph analysis: %+v", analysis)
	}
}
