package mcpserver

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/generator"
	"github.com/ongyoo/gokub/internal/manifest"
)

func TestServeListsToolsAndReadsProject(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("example-api", "github.com/example/example-api")
	if err := generator.NewProject(root, m); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, m.Name)
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"gokub_project_status","arguments":{}}}`,
	}, "\n") + "\n"

	var output bytes.Buffer
	if err := Serve(project, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected three responses, got %d: %s", len(lines), output.String())
	}
	if !strings.Contains(lines[1], "gokub_add_feature") || !strings.Contains(lines[1], "gokub_generate_model") || !strings.Contains(lines[1], "gokub_install_template") || !strings.Contains(lines[1], "gokub_search_templates") || !strings.Contains(lines[1], "gokub_plugins") || !strings.Contains(lines[1], "gokub_project_score") || !strings.Contains(lines[1], "gokub_dependency_graph") || !strings.Contains(lines[1], "gokub_project_upgrade") || !strings.Contains(lines[2], "example-api") {
		t.Fatalf("unexpected MCP output: %s", output.String())
	}
	for _, line := range lines {
		var value any
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			t.Fatalf("invalid JSON-RPC response: %v", err)
		}
	}
}

func TestServeAddsCRUDFeature(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("example-api", "github.com/example/example-api")
	if err := generator.NewProject(root, m); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, m.Name)
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"gokub_add_feature","arguments":{"feature":"crud","name":"product"}}}` + "\n"

	var output bytes.Buffer
	if err := Serve(project, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"isError":false`) {
		t.Fatalf("tool failed: %s", output.String())
	}
	if _, err := os.Stat(filepath.Join(project, "internal", "product", "service.go")); err != nil {
		t.Fatal(err)
	}
}

func TestServeRejectsUnsafeCRUDName(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("example-api", "github.com/example/example-api")
	if err := generator.NewProject(root, m); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, m.Name)
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"gokub_add_feature","arguments":{"feature":"crud","name":"../escape"}}}` + "\n"

	var output bytes.Buffer
	if err := Serve(project, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"isError":true`) {
		t.Fatalf("unsafe name was accepted: %s", output.String())
	}
}

func TestServeGeneratesModelFromProjectJSON(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("example-api", "github.com/example/example-api")
	if err := generator.NewProject(root, m); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(root, m.Name)
	if err := os.WriteFile(filepath.Join(project, "user.json"), []byte(`{"id":1,"name":"Ada"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"gokub_generate_model","arguments":{"name":"user","input":"user.json"}}}` + "\n"

	var output bytes.Buffer
	if err := Serve(project, strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"isError":false`) {
		t.Fatalf("model tool failed: %s", output.String())
	}
	if _, err := os.Stat(filepath.Join(project, "internal", "domain", "user", "model_gen.go")); err != nil {
		t.Fatal(err)
	}
}
