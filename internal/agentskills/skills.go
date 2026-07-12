package agentskills

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	gokub "github.com/gokub/gokub"
)

const (
	sourceBase = "skill-packs"
)

var skillNames = []string{"gokub-project", "gokub-add-domain", "gokub-verify-change"}

var destinations = map[string]string{
	"portable": filepath.Join(".agents", "skills"),
	"codex":    filepath.Join(".agents", "skills"),
	"claude":   filepath.Join(".claude", "skills"),
	"copilot":  filepath.Join(".github", "skills"),
	"gemini":   filepath.Join(".agents", "skills"),
}

func Install(root, agent string, force bool) ([]string, error) {
	targets, err := targets(agent)
	if err != nil {
		return nil, err
	}
	written := []string{}
	for _, target := range targets {
		for _, skillName := range skillNames {
			files, err := copySkill(root, target, skillName, force)
			if err != nil {
				return nil, err
			}
			written = append(written, files...)
		}
	}
	for path, content := range instructionFiles(agent) {
		fullPath := filepath.Join(root, path)
		if !force {
			if _, err := os.Stat(fullPath); err == nil {
				continue
			}
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return nil, err
		}
		written = append(written, path)
	}
	sort.Strings(written)
	return written, nil
}

func Remove(root, agent string) ([]string, error) {
	targets, err := targets(agent)
	if err != nil {
		return nil, err
	}
	removed := []string{}
	for _, target := range targets {
		for _, skillName := range skillNames {
			relative := filepath.Join(target, skillName)
			path := filepath.Join(root, relative)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				continue
			} else if err != nil {
				return nil, err
			}
			if err := os.RemoveAll(path); err != nil {
				return nil, err
			}
			removed = append(removed, relative)
		}
	}
	sort.Strings(removed)
	return removed, nil
}

func Status(root string) map[string]bool {
	status := map[string]bool{}
	for agent, target := range destinations {
		installed := true
		for _, skillName := range skillNames {
			if _, err := os.Stat(filepath.Join(root, target, skillName, "SKILL.md")); err != nil {
				installed = false
				break
			}
		}
		status[agent] = installed
	}
	return status
}

func copySkill(root, target, skillName string, force bool) ([]string, error) {
	written := []string{}
	sourceRoot := path.Join(sourceBase, skillName)
	err := fs.WalkDir(gokub.Assets, sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative := strings.TrimPrefix(sourcePath, sourceRoot+"/")
		destination := filepath.Join(root, target, skillName, relative)
		if !force {
			if _, err := os.Stat(destination); err == nil {
				return nil
			}
		}
		content, err := gokub.Assets.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, content, 0o644); err != nil {
			return err
		}
		written = append(written, filepath.Join(target, skillName, relative))
		return nil
	})
	return written, err
}

func targets(agent string) ([]string, error) {
	switch agent {
	case "", "all":
		return []string{destinations["portable"], destinations["claude"], destinations["copilot"]}, nil
	case "codex", "portable", "claude", "copilot", "gemini":
		return []string{destinations[agent]}, nil
	default:
		return nil, fmt.Errorf("unknown agent %q (choose: all, codex, claude, copilot, gemini, portable)", agent)
	}
}

func instructionFiles(agent string) map[string]string {
	files := map[string]string{}
	if agent == "" || agent == "all" || agent == "copilot" {
		files[filepath.Join(".github", "copilot-instructions.md")] = `# GOKUB Project

Read AGENTS.md and .gokub.yaml before changing code. Use the gokub-project skill
for project workflows. Preserve domain boundaries, add tests, and run go test,
go vet, and gokub doctor before completion.
`
	}
	if agent == "" || agent == "all" || agent == "gemini" {
		files["GEMINI.md"] = `# GOKUB Project

Read AGENTS.md, .gokub.yaml, and .agents/skills/gokub-project/SKILL.md before
working. Use GOKUB commands or MCP tools for generated capabilities and verify
changes with tests, vet, build, and gokub doctor.
`
	}
	return files
}
