package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gokub/gokub/internal/manifest"
)

func TestStandardProjectStyles(t *testing.T) {
	t.Setenv("GOKUB_SKIP_INSTALL", "1")
	tests := []struct {
		name     string
		template string
		style    string
		commands []string
	}{
		{name: "app", template: "monolith", style: "monolith", commands: []string{"app"}},
		{name: "platform", template: "microservices", style: "microservices", commands: []string{"gateway", "example-service"}},
	}
	for _, test := range tests {
		t.Run(test.template, func(t *testing.T) {
			m := manifest.New(test.name, "github.com/example/"+test.name)
			m.Template = test.template
			m.Style = test.style
			root := t.TempDir()
			if err := NewProject(root, m); err != nil {
				t.Fatal(err)
			}
			project := filepath.Join(root, test.name)
			for _, command := range test.commands {
				if _, err := os.Stat(filepath.Join(project, "cmd", command, "main.go")); err != nil {
					t.Fatal(err)
				}
			}
			for _, path := range []string{
				"internal/platform/postgres/postgres.go",
				"internal/platform/mongodb/mongodb.go",
				"internal/platform/messaging/kafka/kafka.go",
				"internal/domain/example/service_test.go",
				".agents/skills/gokub-project/SKILL.md",
				".claude/skills/gokub-project/SKILL.md",
				".github/skills/gokub-project/SKILL.md",
				".github/copilot-instructions.md",
			} {
				if _, err := os.Stat(filepath.Join(project, path)); err != nil {
					t.Fatal(err)
				}
			}
			content, err := os.ReadFile(filepath.Join(project, "go.mod"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(content), "github.com/twmb/franz-go v1.21.5") {
				t.Fatal("core dependencies are missing")
			}
		})
	}
}
