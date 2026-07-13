package plugins

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

const (
	ManifestFile         = "gokub-plugin.json"
	CurrentSchemaVersion = 1
)

var safeName = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var safeVersion = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

type Command struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	Name          string    `json:"name"`
	Version       string    `json:"version"`
	Description   string    `json:"description"`
	Entrypoint    string    `json:"entrypoint"`
	Commands      []Command `json:"commands"`
}

func Create(root, name, module string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	if module == "" {
		module = "example.com/gokub-plugin-" + name
	}
	target := filepath.Join(root, "gokub-plugin-"+name)
	if _, err := os.Stat(target); err == nil {
		return "", fmt.Errorf("target %s already exists", target)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	manifest := Manifest{
		SchemaVersion: CurrentSchemaVersion,
		Name:          name, Version: "0.1.0", Description: name + " plugin for GOKUB",
		Entrypoint: filepath.ToSlash(filepath.Join("bin", "gokub-plugin-"+name+executableSuffix())),
		Commands:   []Command{{Name: "hello", Description: "Run the example plugin command"}},
	}
	manifestContent, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}
	manifestContent = append(manifestContent, '\n')
	files := map[string][]byte{
		ManifestFile: manifestContent,
		"go.mod":     []byte("module " + module + "\n\ngo 1.25\n"),
		filepath.Join("cmd", "plugin", "main.go"): []byte(pluginMain(name)),
		"Makefile":   []byte(pluginMakefile(name)),
		"README.md":  []byte(pluginReadme(name)),
		".gitignore": []byte("bin/\n"),
	}
	for relative, content := range files {
		path := filepath.Join(target, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return "", err
		}
		if err := os.WriteFile(path, content, 0o644); err != nil {
			return "", err
		}
	}
	return target, nil
}

func Install(source string) (Manifest, error) {
	manifest, err := readManifest(filepath.Join(source, ManifestFile))
	if err != nil {
		return Manifest{}, err
	}
	entrypoint, err := safeEntrypoint(source, manifest.Entrypoint)
	if err != nil {
		return Manifest{}, err
	}
	if err := executableFile(entrypoint); err != nil {
		return Manifest{}, err
	}
	destination, err := installedPath(manifest.Name)
	if err != nil {
		return Manifest{}, err
	}
	if _, err := os.Stat(destination); err == nil {
		return Manifest{}, fmt.Errorf("plugin %q is already installed", manifest.Name)
	} else if !os.IsNotExist(err) {
		return Manifest{}, err
	}
	if err := copyPlugin(source, destination); err != nil {
		_ = os.RemoveAll(destination)
		return Manifest{}, err
	}
	return manifest, nil
}

func Remove(name string) error {
	path, err := installedPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("plugin %q is not installed", name)
	} else if err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func List() ([]Manifest, error) {
	root, err := rootDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []Manifest{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []Manifest{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest, err := readManifest(filepath.Join(root, entry.Name(), ManifestFile))
		if err != nil {
			return nil, fmt.Errorf("read installed plugin %s: %w", entry.Name(), err)
		}
		items = append(items, manifest)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func Execute(projectRoot, name string, args []string, in io.Reader, out, errOut io.Writer) error {
	path, err := installedPath(name)
	if err != nil {
		return err
	}
	manifest, err := readManifest(filepath.Join(path, ManifestFile))
	if err != nil {
		return fmt.Errorf("plugin %q is not installed or invalid: %w", name, err)
	}
	if len(args) == 0 && len(manifest.Commands) == 1 {
		args = []string{manifest.Commands[0].Name}
	}
	if len(args) == 0 || !hasCommand(manifest, args[0]) {
		return fmt.Errorf("plugin %q command must be one of: %s", name, commandNames(manifest))
	}
	entrypoint, err := safeEntrypoint(path, manifest.Entrypoint)
	if err != nil {
		return err
	}
	if err := executableFile(entrypoint); err != nil {
		return err
	}
	absoluteRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return err
	}
	command := exec.Command(entrypoint, args...)
	command.Dir = absoluteRoot
	command.Env = append(os.Environ(), "GOKUB_PROJECT_ROOT="+absoluteRoot, "GOKUB_PLUGIN_NAME="+manifest.Name, "GOKUB_PLUGIN_VERSION="+manifest.Version)
	command.Stdin = in
	command.Stdout = out
	command.Stderr = errOut
	if err := command.Run(); err != nil {
		return fmt.Errorf("plugin %s failed: %w", name, err)
	}
	return nil
}

func readManifest(path string) (Manifest, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := validateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported plugin schema version %d", manifest.SchemaVersion)
	}
	if err := validateName(manifest.Name); err != nil {
		return err
	}
	if !safeVersion.MatchString(manifest.Version) {
		return fmt.Errorf("plugin version %q contains unsafe characters", manifest.Version)
	}
	if len(manifest.Commands) == 0 {
		return fmt.Errorf("plugin must declare at least one command")
	}
	seen := map[string]bool{}
	for _, command := range manifest.Commands {
		if err := validateName(command.Name); err != nil {
			return fmt.Errorf("invalid plugin command: %w", err)
		}
		if seen[command.Name] {
			return fmt.Errorf("duplicate plugin command %q", command.Name)
		}
		seen[command.Name] = true
	}
	return nil
}

func validateName(name string) error {
	if !safeName.MatchString(name) {
		return fmt.Errorf("plugin name %q must use lowercase letters, numbers, and hyphens", name)
	}
	return nil
}

func safeEntrypoint(root, relative string) (string, error) {
	if relative == "" || filepath.IsAbs(relative) {
		return "", fmt.Errorf("plugin entrypoint must be a relative path")
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("plugin entrypoint must stay inside the plugin directory")
	}
	return filepath.Join(root, clean), nil
}

func executableFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("read plugin entrypoint: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("plugin entrypoint must be a regular file")
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("plugin entrypoint %s is not executable", path)
	}
	return nil
}

func copyPlugin(source, destination string) error {
	excluded := map[string]bool{".git": true, ".env": true, "node_modules": true, "dist": true, ".DS_Store": true}
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return os.MkdirAll(destination, 0o755)
		}
		if excluded[entry.Name()] || strings.HasPrefix(entry.Name(), ".env") {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	})
}

func installedPath(name string) (string, error) {
	if err := validateName(name); err != nil {
		return "", err
	}
	root, err := rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

func rootDir() (string, error) {
	home := os.Getenv("GOKUB_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(home, ".gokub")
	}
	return filepath.Join(home, "plugins"), nil
}

func hasCommand(manifest Manifest, name string) bool {
	for _, command := range manifest.Commands {
		if command.Name == name {
			return true
		}
	}
	return false
}

func commandNames(manifest Manifest) string {
	names := make([]string, 0, len(manifest.Commands))
	for _, command := range manifest.Commands {
		names = append(names, command.Name)
	}
	return strings.Join(names, ", ")
}

func executableSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func pluginMain(name string) string {
	return fmt.Sprintf(`package main

import (
	"fmt"
	"os"
)

func main() {
	command := "hello"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	switch command {
	case "hello":
		fmt.Printf("Hello from the %s plugin in %%s\n", os.Getenv("GOKUB_PROJECT_ROOT"))
	default:
		fmt.Fprintf(os.Stderr, "unknown command %%q\n", command)
		os.Exit(2)
	}
}
`, name)
}

func pluginMakefile(name string) string {
	return fmt.Sprintf(".PHONY: build test\n\nbuild:\n\tgo build -o bin/gokub-plugin-%s%s ./cmd/plugin\n\ntest:\n\tgo test ./...\n", name, executableSuffix())
}

func pluginReadme(name string) string {
	return fmt.Sprintf("# GOKUB Plugin: %s\n\nBuild and install locally:\n\n```bash\nmake build\ngokub plugin install .\ngokub plugin run %s hello\n```\n", name, name)
}
