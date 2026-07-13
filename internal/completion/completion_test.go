package completion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScriptsContainGOKUB(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish"} {
		script, err := Script(shell)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(script, "gokub") || !strings.Contains(script, "completion") {
			t.Fatalf("%s completion is incomplete", shell)
		}
	}
	if _, err := Script("powershell"); err == nil {
		t.Fatal("expected unsupported shell error")
	}
}

func TestInstallZshIsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	for i := 0; i < 2; i++ {
		path, err := Install("")
		if err != nil {
			t.Fatal(err)
		}
		if path != filepath.Join(home, ".gokub", "completions", "_gokub") {
			t.Fatalf("unexpected path %s", path)
		}
	}
	rc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(rc), "# GOKUB completion") != 1 {
		t.Fatalf("completion block was duplicated:\n%s", rc)
	}
}

func TestInstallFishUsesNativeDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := Install("fish")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(home, ".config", "fish", "completions", "gokub.fish") {
		t.Fatalf("unexpected path %s", path)
	}
	if _, err := os.Stat(filepath.Join(home, ".fishrc")); !os.IsNotExist(err) {
		t.Fatal("fish install should not create an rc file")
	}
}
