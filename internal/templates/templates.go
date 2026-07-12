package templates

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gokub/gokub/internal/manifest"
)

var excluded = map[string]bool{
	".git": true, ".gocache": true, ".idea": true, "node_modules": true,
	"dist": true, "tmp": true, ".env": true, ".DS_Store": true,
}

func Add(name, source string) (string, error) {
	info, err := os.Stat(source)
	if err != nil {
		return "", fmt.Errorf("read template source: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("template source must be a directory")
	}
	if name == "" {
		name = filepath.Base(filepath.Clean(source))
	}
	if err := validateName(name); err != nil {
		return "", err
	}
	destination, err := storedPath(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(destination); err == nil {
		return "", fmt.Errorf("template %q already exists; remove it before replacing", name)
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := copyTree(source, destination, nil); err != nil {
		_ = os.RemoveAll(destination)
		return "", err
	}
	return name, nil
}

func Remove(name string) error {
	if err := validateName(name); err != nil {
		return err
	}
	path, err := storedPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("template %q is not installed", name)
	}
	return os.RemoveAll(path)
}

func Names() ([]string, error) {
	root, err := rootDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	names := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func Resolve(nameOrPath string) (string, bool, error) {
	if info, err := os.Stat(nameOrPath); err == nil && info.IsDir() {
		absolute, err := filepath.Abs(nameOrPath)
		return absolute, true, err
	}
	if err := validateName(nameOrPath); err != nil {
		return "", false, nil
	}
	path, err := storedPath(nameOrPath)
	if err != nil {
		return "", false, err
	}
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return path, info.IsDir(), nil
}

func Generate(source, root string, m manifest.Manifest) error {
	target := filepath.Join(root, m.Name)
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("target %s already exists", target)
	} else if !os.IsNotExist(err) {
		return err
	}
	replacements := map[string]string{
		"{{project_name}}": m.Name,
		"{{module}}":       m.Module,
		"{{template}}":     m.Template,
		"{{style}}":        m.Style,
		"{{framework}}":    m.Framework,
		"{{database}}":     m.Database,
		"{{architecture}}": m.Architecture,
		"{{messaging}}":    m.Messaging,
	}
	if err := copyTree(source, target, replacements); err != nil {
		_ = os.RemoveAll(target)
		return err
	}
	return manifest.Write(filepath.Join(target, manifest.FileName), m)
}

func copyTree(source, destination string, replacements map[string]string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return os.MkdirAll(destination, 0o755)
		}
		if excluded[entry.Name()] {
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
		relative = replace(relative, replacements)
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if replacements != nil && isText(content) {
			content = []byte(replace(string(content), replacements))
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, content, info.Mode().Perm())
	})
}

func replace(value string, replacements map[string]string) string {
	for placeholder, replacement := range replacements {
		value = strings.ReplaceAll(value, placeholder, replacement)
	}
	return value
}

func isText(content []byte) bool {
	for _, value := range content {
		if value == 0 {
			return false
		}
	}
	return true
}

func rootDir() (string, error) {
	data, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(data, "templates"), nil
}

func Purge() error {
	data, err := dataDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(data)
}

func dataDir() (string, error) {
	if value := os.Getenv("GOKUB_HOME"); value != "" {
		return value, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gokub"), nil
}

func storedPath(name string) (string, error) {
	root, err := rootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, name), nil
}

func validateName(name string) error {
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("template name %q must be a single directory-safe name", name)
	}
	return nil
}
