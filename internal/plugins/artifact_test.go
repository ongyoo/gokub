package plugins

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPackIsReproducibleAndVerifiable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test fixture uses POSIX executable permissions")
	}
	source := t.TempDir()
	manifest := `{"schema_version":1,"name":"audit","version":"1.2.3","description":"test","entrypoint":"bin/audit","commands":[{"name":"run","description":"run"}]}`
	writePluginFile(t, filepath.Join(source, ManifestFile), manifest, 0o644)
	writePluginFile(t, filepath.Join(source, "bin", "audit"), "binary", 0o755)
	writePluginFile(t, filepath.Join(source, ".env.local"), "SECRET=value", 0o644)

	first, err := Pack(source, filepath.Join(t.TempDir(), "one"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Pack(source, filepath.Join(t.TempDir(), "two"))
	if err != nil {
		t.Fatal(err)
	}
	if first.SHA256 != second.SHA256 {
		t.Fatalf("artifacts are not reproducible: %s != %s", first.SHA256, second.SHA256)
	}
	assertSafeArchive(t, first.Archive)
	if _, err := Verify(first.Archive, first.ChecksumFile); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first.Archive, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(first.Archive, first.ChecksumFile); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("tampered artifact passed verification: %v", err)
	}
}

func assertSafeArchive(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	foundExecutable := false
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(filepath.Base(header.Name), ".env") {
			t.Fatalf("secret file included in artifact: %s", header.Name)
		}
		if header.Name == "bin/audit" && header.Mode&0o111 != 0 {
			foundExecutable = true
		}
	}
	if !foundExecutable {
		t.Fatal("plugin entrypoint or executable permission missing from artifact")
	}
}
