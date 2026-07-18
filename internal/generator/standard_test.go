package generator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ongyoo/gokub/internal/manifest"
)

func TestKitProjectGeneratesLayout(t *testing.T) {
	t.Setenv("GOKUB_SKIP_INSTALL", "1")
	for _, framework := range []string{"gin", "fiber", "echo"} {
		t.Run(framework, func(t *testing.T) {
			m := manifest.New("svc", "github.com/example/svc")
			m.Framework = framework
			root := t.TempDir()
			if err := NewProject(root, m); err != nil {
				t.Fatal(err)
			}
			project := filepath.Join(root, "svc")
			for _, path := range []string{
				"cmd/svc-service/main.go",
				"config/config.go",
				"internal/app/app.go",
				"internal/app/events/publisher.go",
				"internal/example/model.go",
				"internal/example/repository.go",
				"internal/example/service.go",
				"internal/example/service_test.go",
				"internal/example/handler.go",
				"internal/example/router.go",
				"pkg/api/response.go",
				"pkg/crypto/crypto.go",
				"pkg/error/error.go",
				"pkg/database/postgresql/postgresql.go",
				"pkg/utils/utils.go",
				"pkg/validator/validator.go",
				".env",
				"pkg/httpserver/" + framework + "/http.go",
				"pkg/middleware/" + framework + "/middleware.go",
				"Dockerfile",
				"Makefile",
				"docker-compose.yml",
				".env.example",
				".github/workflows/ci.yml",
				".agents/skills/gokub-project/SKILL.md",
			} {
				if _, err := os.Stat(filepath.Join(project, filepath.FromSlash(path))); err != nil {
					t.Fatalf("%s: %v", path, err)
				}
			}

			goMod, err := os.ReadFile(filepath.Join(project, "go.mod"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(goMod), "module github.com/example/svc\n") {
				t.Fatalf("unexpected go.mod:\n%s", goMod)
			}

			manifestBytes, err := os.ReadFile(filepath.Join(project, manifest.FileName))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(manifestBytes), "framework: "+framework) {
				t.Fatalf("manifest missing framework %q:\n%s", framework, manifestBytes)
			}
		})
	}
}
