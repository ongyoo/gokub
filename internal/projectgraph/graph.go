package projectgraph

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/ongyoo/gokub/internal/manifest"
)

type Node struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Layer string `json:"layer"`
}

type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type Graph struct {
	Module string `json:"module"`
	Nodes  []Node `json:"nodes"`
	Edges  []Edge `json:"edges"`
}

func Build(root string, includeTests bool) (Graph, error) {
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return Graph{}, fmt.Errorf("read project manifest: %w", err)
	}
	if err := manifest.Validate(m); err != nil {
		return Graph{}, fmt.Errorf("validate project manifest: %w", err)
	}

	nodes := map[string]Node{}
	edges := map[Edge]bool{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules", "dist":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || (!includeTests && strings.HasSuffix(path, "_test.go")) {
			return nil
		}
		relativeDir, err := filepath.Rel(root, filepath.Dir(path))
		if err != nil {
			return err
		}
		packagePath := m.Module
		if relativeDir != "." {
			packagePath += "/" + filepath.ToSlash(relativeDir)
		}
		nodes[packagePath] = Node{ID: packagePath, Path: filepath.ToSlash(relativeDir), Layer: layer(relativeDir)}

		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		for _, spec := range parsed.Imports {
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil || (importPath != m.Module && !strings.HasPrefix(importPath, m.Module+"/")) {
				continue
			}
			edges[Edge{From: packagePath, To: importPath}] = true
		}
		return nil
	})
	if err != nil {
		return Graph{}, err
	}

	graph := Graph{Module: m.Module, Nodes: make([]Node, 0, len(nodes)), Edges: make([]Edge, 0, len(edges))}
	for _, node := range nodes {
		graph.Nodes = append(graph.Nodes, node)
	}
	for edge := range edges {
		graph.Edges = append(graph.Edges, edge)
	}
	sort.Slice(graph.Nodes, func(i, j int) bool { return graph.Nodes[i].ID < graph.Nodes[j].ID })
	sort.Slice(graph.Edges, func(i, j int) bool {
		if graph.Edges[i].From == graph.Edges[j].From {
			return graph.Edges[i].To < graph.Edges[j].To
		}
		return graph.Edges[i].From < graph.Edges[j].From
	})
	return graph, nil
}

func Mermaid(graph Graph) string {
	var output strings.Builder
	output.WriteString("graph TD\n")
	ids := map[string]string{}
	for index, node := range graph.Nodes {
		id := fmt.Sprintf("n%d", index)
		ids[node.ID] = id
		label := node.Path
		if label == "." {
			label = filepath.Base(graph.Module)
		}
		fmt.Fprintf(&output, "  %s[\"%s\\n%s\"]\n", id, escape(label), escape(node.Layer))
	}
	for _, edge := range graph.Edges {
		from, fromOK := ids[edge.From]
		to, toOK := ids[edge.To]
		if fromOK && toOK {
			fmt.Fprintf(&output, "  %s --> %s\n", from, to)
		}
	}
	return output.String()
}

func layer(relativeDir string) string {
	path := filepath.ToSlash(relativeDir)
	switch {
	case path == "cmd" || strings.HasPrefix(path, "cmd/"):
		return "service"
	case path == "internal/domain" || strings.HasPrefix(path, "internal/domain/"):
		return "domain"
	case path == "internal/platform" || strings.HasPrefix(path, "internal/platform/"):
		return "platform"
	case path == "internal/http" || strings.HasPrefix(path, "internal/http/"):
		return "transport"
	case path == "pkg" || strings.HasPrefix(path, "pkg/"):
		return "shared"
	default:
		return "application"
	}
}

func escape(value string) string {
	return strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(value)
}
