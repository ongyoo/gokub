package modelgen

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestGenerateFromSampleJSON(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "user.json")
	content := `{"id":42,"display_name":"Ada","created_at":"2026-07-13T10:30:00Z","active":true,"profile":{"avatar_url":"https://example.com/a.png"},"roles":[{"id":"admin"}]}`
	if err := os.WriteFile(input, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := Generate(Options{Root: root, Name: "user", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	for _, expected := range []string{`import "time"`, `type User struct`, `ID\s+int64`, `CreatedAt\s+time\.Time`, `DisplayName\s+string`, `Profile\s+UserProfile`, `Roles\s+\[\]UserRole`, `json:"display_name"`} {
		if !regexp.MustCompile(expected).MatchString(text) {
			t.Fatalf("generated model missing %q:\n%s", expected, text)
		}
	}
}

func TestGenerateFromInlineContent(t *testing.T) {
	root := t.TempDir()
	path, err := Generate(Options{Root: root, Name: "account", Content: []byte(`{"id":1,"balance":9.9}`)})
	if err != nil {
		t.Fatal(err)
	}
	generated, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(generated)
	if !strings.Contains(text, "type Account struct") || !strings.Contains(text, `json:"balance"`) {
		t.Fatalf("inline content model missing fields:\n%s", text)
	}
}

func TestGenerateRejectsOutputOutsideProject(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "model.json")
	if err := os.WriteFile(input, []byte(`{"id":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Generate(Options{Root: root, Name: "model", Input: input, Output: filepath.Join(root, "..", "model.go")})
	if err == nil || !strings.Contains(err.Error(), "inside the project") {
		t.Fatalf("expected output path rejection, got %v", err)
	}
}

func TestGenerateFromJSONSchema(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "domain"), 0o755); err != nil {
		t.Fatal(err)
	}
	input := filepath.Join(root, "event.schema.json")
	content := `{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object","required":["id","created_at"],"properties":{"id":{"type":"string"},"created_at":{"type":"string","format":"date-time"},"note":{"type":["string","null"]},"items":{"type":"array","items":{"type":"integer"}}}}`
	if err := os.WriteFile(input, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	path, err := Generate(Options{Root: root, Name: "event", Input: input})
	if err != nil {
		t.Fatal(err)
	}
	generated, _ := os.ReadFile(path)
	text := string(generated)
	for _, expected := range []string{`import "time"`, `CreatedAt\s+time\.Time`, `Note\s+\*string`, `json:"note,omitempty"`, `Items\s+\[\]int64`} {
		if !regexp.MustCompile(expected).MatchString(text) {
			t.Fatalf("generated schema model missing %q:\n%s", expected, text)
		}
	}
	if _, err := Generate(Options{Root: root, Name: "event", Input: input}); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite protection, got %v", err)
	}
}
