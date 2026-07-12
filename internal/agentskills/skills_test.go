package agentskills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallPreserveForceAndRemove(t *testing.T) {
	root := t.TempDir()
	written, err := Install(root, "all", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(written) == 0 {
		t.Fatal("expected installed skill files")
	}
	paths := []string{
		".agents/skills/gokub-project/SKILL.md",
		".agents/skills/gokub-add-domain/SKILL.md",
		".agents/skills/gokub-verify-change/SKILL.md",
		".claude/skills/gokub-project/SKILL.md",
		".github/skills/gokub-project/SKILL.md",
		".github/copilot-instructions.md",
		"GEMINI.md",
	}
	for _, path := range paths {
		if _, err := os.Stat(filepath.Join(root, path)); err != nil {
			t.Fatal(err)
		}
	}

	canonical := filepath.Join(root, ".agents", "skills", "gokub-project", "SKILL.md")
	if err := os.WriteFile(canonical, []byte("custom"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(root, "codex", false); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "custom" {
		t.Fatal("install without force overwrote a customized skill")
	}
	if _, err := Install(root, "codex", true); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "name: gokub-project") {
		t.Fatal("force did not refresh the skill")
	}
	removed, err := Remove(root, "all")
	if err != nil {
		t.Fatal(err)
	}
	if len(removed) != 9 {
		t.Fatalf("expected nine removed skill directories, got %v", removed)
	}
}

func TestRejectsUnknownAgent(t *testing.T) {
	if _, err := Install(t.TempDir(), "unknown", false); err == nil {
		t.Fatal("expected unknown agent error")
	}
}
