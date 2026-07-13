package doctor

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ongyoo/gokub/internal/manifest"
	"github.com/ongyoo/gokub/internal/projectgraph"
)

const pointsPerCheck = 5

type ScoreCheck struct {
	Name           string `json:"name"`
	OK             bool   `json:"ok"`
	Points         int    `json:"points"`
	Recommendation string `json:"recommendation,omitempty"`
}

type ScoreCategory struct {
	Name   string       `json:"name"`
	Score  int          `json:"score"`
	Max    int          `json:"max"`
	Checks []ScoreCheck `json:"checks"`
}

type ScoreReport struct {
	Score      int             `json:"score"`
	Max        int             `json:"max"`
	Grade      string          `json:"grade"`
	Categories []ScoreCategory `json:"categories"`
}

// Score evaluates static project signals without executing project code.
func Score(root string) ScoreReport {
	m, manifestOK := validManifest(root)
	categories := []ScoreCategory{
		category("Architecture",
			scoreCheck("Valid GOKUB manifest", manifestOK, "Run gokub doctor and repair .gokub.yaml."),
			scoreCheck("Matching Go module", manifestOK && fileHas(filepath.Join(root, "go.mod"), "module "+m.Module), "Set the go.mod module path to match .gokub.yaml."),
			scoreCheck("Application entrypoint", treeHas(root, "cmd", func(path string) bool { return filepath.Base(path) == "main.go" }), "Add a cmd/<service>/main.go composition root."),
			scoreCheck("Architecture boundaries", isDir(filepath.Join(root, "internal", "domain")) && isDir(filepath.Join(root, "internal", "platform")), "Keep business rules under internal/domain and adapters under internal/platform."),
			scoreCheck("Clean dependencies", cleanDependencies(root), "Run gokub graph --check and move outward dependencies behind domain interfaces."),
		),
		category("Security",
			scoreCheck("Environment example", isFile(filepath.Join(root, ".env.example")), "Document required variables in .env.example without secrets."),
			scoreCheck("Local secrets ignored", fileHas(filepath.Join(root, ".gitignore"), ".env"), "Add .env files to .gitignore."),
			scoreCheck("Non-root container", fileHas(filepath.Join(root, "Dockerfile"), "USER "), "Run the production container as a non-root user."),
			scoreCheck("Secure HTTP headers", goTreeHas(root, "X-Content-Type-Options"), "Add secure response headers such as X-Content-Type-Options."),
			scoreCheck("Security profile", fileHas(filepath.Join(root, manifest.FileName), "security:"), "Declare the project security profile in .gokub.yaml."),
		),
		category("Testing",
			scoreCheck("Go tests", treeHas(root, "", func(path string) bool { return strings.HasSuffix(path, "_test.go") }), "Add focused unit or transport tests."),
			scoreCheck("Test workspace", isDir(filepath.Join(root, "tests")), "Add a tests directory for integration and acceptance tests."),
			scoreCheck("Continuous integration", isDir(filepath.Join(root, ".github", "workflows")), "Add a CI workflow under .github/workflows."),
			scoreCheck("Race detector", treeContentHas(root, ".github/workflows", "go test -race"), "Run go test -race ./... in CI."),
			scoreCheck("Static analysis", treeContentHas(root, ".github/workflows", "go vet"), "Run go vet ./... in CI."),
		),
		category("Operations",
			scoreCheck("Container build", isFile(filepath.Join(root, "Dockerfile")), "Add a reproducible multi-stage Dockerfile."),
			scoreCheck("Deployment assets", isDir(filepath.Join(root, "deployments")) || isFile(filepath.Join(root, "docker-compose.yml")), "Add deployment or local orchestration assets."),
			scoreCheck("Health endpoint", goTreeHas(root, "/health/live"), "Expose a liveness endpoint at /health/live."),
			scoreCheck("Graceful shutdown", goTreeHas(root, "signal.NotifyContext"), "Handle SIGINT and SIGTERM with graceful shutdown."),
			scoreCheck("Structured logging", goTreeHas(root, "log/slog"), "Use structured logging for operational events."),
		),
	}
	report := ScoreReport{Max: len(categories) * 5 * pointsPerCheck, Categories: categories}
	for _, item := range categories {
		report.Score += item.Score
	}
	report.Grade = grade(report.Score)
	return report
}

func cleanDependencies(root string) bool {
	graph, err := projectgraph.Build(root, false)
	return err == nil && projectgraph.Analyze(graph).OK
}

func category(name string, checks ...ScoreCheck) ScoreCategory {
	result := ScoreCategory{Name: name, Max: len(checks) * pointsPerCheck, Checks: checks}
	for _, check := range checks {
		result.Score += check.Points
	}
	return result
}

func scoreCheck(name string, ok bool, recommendation string) ScoreCheck {
	points := 0
	if ok {
		points = pointsPerCheck
		recommendation = ""
	}
	return ScoreCheck{Name: name, OK: ok, Points: points, Recommendation: recommendation}
}

func grade(score int) string {
	switch {
	case score >= 90:
		return "Excellent"
	case score >= 75:
		return "Healthy"
	case score >= 50:
		return "Needs work"
	default:
		return "At risk"
	}
}

func validManifest(root string) (manifest.Manifest, bool) {
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	return m, err == nil && manifest.Validate(m) == nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func fileHas(path, text string) bool {
	content, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(content), text)
}

func goTreeHas(root, text string) bool {
	return treeContentHas(root, "", text, ".go")
}

func treeContentHas(root, relative, text string, extensions ...string) bool {
	return treeHas(root, relative, func(path string) bool {
		if len(extensions) > 0 {
			matched := false
			for _, extension := range extensions {
				matched = matched || strings.HasSuffix(path, extension)
			}
			if !matched {
				return false
			}
		}
		return fileHas(path, text)
	})
}

func treeHas(root, relative string, match func(string) bool) bool {
	base := filepath.Join(root, relative)
	found := false
	_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "vendor", "node_modules", "dist":
				if path != base {
					return filepath.SkipDir
				}
			}
			return nil
		}
		found = match(path)
		return nil
	})
	return found
}
