package templates

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeRepository(t *testing.T) {
	repository, name, err := normalizeRepository("ongyoo/gokub-template", false)
	if err != nil || repository != "https://github.com/ongyoo/gokub-template.git" || name != "gokub-template" {
		t.Fatalf("unexpected shorthand result: %q %q %v", repository, name, err)
	}
	for _, unsafe := range []string{"http://github.com/a/b", "https://user:token@github.com/a/b", "https://example.com/a/b", "../repo"} {
		if _, _, err := normalizeRepository(unsafe, false); err == nil {
			t.Fatalf("unsafe repository accepted: %s", unsafe)
		}
	}
}

func TestInstallFromRepositorySubdirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	home := t.TempDir()
	t.Setenv("GOKUB_HOME", home)
	repository := filepath.Join(t.TempDir(), "community-template")
	if err := os.MkdirAll(filepath.Join(repository, "templates", "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "templates", "api", "go.mod"), []byte("module {{module}}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repository, "templates", "api", ".env"), []byte("SECRET=value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"init"}, {"add", "."}, {"-c", "user.name=GOKUB", "-c", "user.email=test@example.com", "commit", "-m", "template"}} {
		command := exec.Command("git", args...)
		command.Dir = repository
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	repositoryURL := (&url.URL{Scheme: "file", Path: repository}).String()
	name, err := Install(InstallOptions{Repository: repositoryURL, Subdir: "templates/api", Name: "community-api", allowFile: true})
	if err != nil {
		t.Fatal(err)
	}
	if name != "community-api" {
		t.Fatalf("unexpected name %q", name)
	}
	destination, _ := storedPath(name)
	metadata, err := os.ReadFile(filepath.Join(destination, metadataFile))
	if err != nil || !strings.Contains(string(metadata), repositoryURL) {
		t.Fatalf("metadata missing: %v %s", err, metadata)
	}
	if _, err := os.Stat(filepath.Join(destination, ".git")); !os.IsNotExist(err) {
		t.Fatal("repository metadata was copied")
	}
	if _, err := os.Stat(filepath.Join(destination, ".env")); !os.IsNotExist(err) {
		t.Fatal("secret environment file was copied")
	}
}
