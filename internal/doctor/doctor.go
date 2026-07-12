package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gokub/gokub/internal/catalog"
	"github.com/gokub/gokub/internal/manifest"
)

type Result struct {
	Name string
	OK   bool
	Info string
}

func Check(root string) []Result {
	checks := []Result{}
	requiredDirs := []string{"cmd", "internal", "configs", "deployments", "docs", "scripts", "tests", "migrations"}
	for _, dir := range requiredDirs {
		checks = append(checks, exists(filepath.Join(root, dir), "directory "+dir))
	}
	checks = append(checks, exists(filepath.Join(root, manifest.FileName), "manifest"))
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		checks = append(checks, Result{Name: "manifest readable", OK: false, Info: err.Error()})
	} else {
		checks = append(checks, Result{Name: "manifest readable", OK: true, Info: "ok"})
		if err := manifest.Validate(m); err != nil {
			checks = append(checks, Result{Name: "manifest valid", OK: false, Info: err.Error()})
		} else {
			checks = append(checks, Result{Name: "manifest valid", OK: true, Info: "ok"})
		}
		checks = append(checks, fileContains(filepath.Join(root, "go.mod"), "module "+m.Module, "module path"))
		checks = append(checks, exists(filepath.Join(root, "cmd", m.Name, "main.go"), "service entrypoint"))
		checks = append(checks, exists(filepath.Join(root, ".env.example"), "environment example"))
		checks = append(checks, exists(filepath.Join(root, ".codex", "config.toml"), "Codex MCP config"))
		checks = append(checks, exists(filepath.Join(root, ".mcp.json"), "MCP client config"))
		for _, feature := range m.Features {
			base := strings.SplitN(feature, ":", 2)[0]
			if !catalog.HasFeature(base) {
				checks = append(checks, Result{Name: "feature " + feature, OK: false, Info: "unknown feature in manifest"})
			}
		}
	}
	return checks
}

func fileContains(path, expected, name string) Result {
	content, err := os.ReadFile(path)
	if err != nil {
		return Result{Name: name, OK: false, Info: err.Error()}
	}
	if !strings.Contains(string(content), expected) {
		return Result{Name: name, OK: false, Info: fmt.Sprintf("%s does not contain %q", path, expected)}
	}
	return Result{Name: name, OK: true, Info: "ok"}
}

func exists(path, name string) Result {
	if _, err := os.Stat(path); err != nil {
		return Result{Name: name, OK: false, Info: fmt.Sprintf("missing %s", path)}
	}
	return Result{Name: name, OK: true, Info: "ok"}
}
