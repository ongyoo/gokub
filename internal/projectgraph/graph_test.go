package projectgraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/manifest"
)

func TestBuildFindsInternalDependenciesAndLayers(t *testing.T) {
	root := t.TempDir()
	m := manifest.New("service", "example.com/service")
	if err := manifest.Write(filepath.Join(root, manifest.FileName), m); err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, "cmd/service/main.go", `package main
import "example.com/service/internal/domain/order"
`)
	writeFile(t, root, "internal/domain/order/service.go", `package order
import "example.com/service/internal/platform/postgres"
`)
	writeFile(t, root, "internal/domain/order/service_test.go", `package order
import "example.com/service/internal/testing"
`)
	writeFile(t, root, "internal/platform/postgres/store.go", "package postgres\n")

	graph, err := Build(root, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(graph.Nodes) != 3 || len(graph.Edges) != 2 {
		t.Fatalf("unexpected graph: %+v", graph)
	}
	if graph.Nodes[0].Layer != "service" || !strings.Contains(Mermaid(graph), "n0 -->") {
		t.Fatalf("layers or Mermaid output are invalid: %+v\n%s", graph, Mermaid(graph))
	}

	withTests, err := Build(root, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(withTests.Edges) != 3 {
		t.Fatalf("test dependency not included: %+v", withTests.Edges)
	}
}

func writeFile(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
