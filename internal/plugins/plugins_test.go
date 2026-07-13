package plugins

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCreateScaffoldsVersionedPlugin(t *testing.T) {
	path, err := Create(t.TempDir(), "audit", "example.com/audit-plugin")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := readManifest(filepath.Join(path, ManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "audit" || manifest.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
	if _, err := os.Stat(filepath.Join(path, "cmd", "plugin", "main.go")); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("go", "test", "./...")
	command.Dir = path
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("generated plugin does not compile: %v\n%s", err, output)
	}
}

func TestInstallListExecuteAndRemove(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses a POSIX executable")
	}
	t.Setenv("GOKUB_HOME", t.TempDir())
	source := t.TempDir()
	manifest := `{"schema_version":1,"name":"hello","version":"1.0.0","description":"test","entrypoint":"bin/hello","commands":[{"name":"greet","description":"greet"}]}`
	writePluginFile(t, filepath.Join(source, ManifestFile), manifest, 0o644)
	writePluginFile(t, filepath.Join(source, "bin", "hello"), "#!/bin/sh\nprintf 'plugin:%s:%s' \"$GOKUB_PLUGIN_NAME\" \"$1\"\n", 0o755)
	writePluginFile(t, filepath.Join(source, ".env"), "SECRET=value\n", 0o644)

	installed, err := Install(source)
	if err != nil {
		t.Fatal(err)
	}
	if installed.Name != "hello" {
		t.Fatalf("unexpected plugin: %+v", installed)
	}
	items, err := List()
	if err != nil || len(items) != 1 || items[0].Name != "hello" {
		t.Fatalf("unexpected list: %+v %v", items, err)
	}
	var output bytes.Buffer
	if err := Execute(t.TempDir(), "hello", []string{"greet"}, strings.NewReader(""), &output, &output); err != nil {
		t.Fatal(err)
	}
	if output.String() != "plugin:hello:greet" {
		t.Fatalf("unexpected output %q", output.String())
	}
	path, _ := installedPath("hello")
	if _, err := os.Stat(filepath.Join(path, ".env")); !os.IsNotExist(err) {
		t.Fatal("secret file was installed")
	}
	if err := Remove("hello"); err != nil {
		t.Fatal(err)
	}
}

func TestInstallRejectsEscapingEntrypoint(t *testing.T) {
	t.Setenv("GOKUB_HOME", t.TempDir())
	source := t.TempDir()
	manifest := `{"schema_version":1,"name":"unsafe","version":"1.0.0","entrypoint":"../run","commands":[{"name":"run"}]}`
	writePluginFile(t, filepath.Join(source, ManifestFile), manifest, 0o644)
	if _, err := Install(source); err == nil || !strings.Contains(err.Error(), "inside") {
		t.Fatalf("unsafe entrypoint accepted: %v", err)
	}
}

func writePluginFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
