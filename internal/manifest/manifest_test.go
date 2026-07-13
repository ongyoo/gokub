package manifest

import (
	"path/filepath"
	"testing"
)

func TestManifestPersistsGoVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), FileName)
	m := New("service", "example.com/service")
	if m.GoVersion != "1.26" {
		t.Fatalf("unexpected recommended Go version %q", m.GoVersion)
	}
	m.GoVersion = "1.25"
	if err := Write(path, m); err != nil {
		t.Fatal(err)
	}
	got, err := Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.GoVersion != "1.25" || got.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("unexpected manifest: %+v", got)
	}
}

func TestManifestRejectsInvalidGoVersion(t *testing.T) {
	m := New("service", "example.com/service")
	m.GoVersion = "1.26.1"
	if err := Validate(m); err == nil {
		t.Fatal("patch-level Go version was accepted")
	}
}
