package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const metadataFile = ".gokub-template.json"

var githubShorthand = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
var githubComponent = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

type InstallOptions struct {
	Repository string
	Ref        string
	Subdir     string
	Name       string
	allowFile  bool
}

type Metadata struct {
	Repository  string `json:"repository"`
	Ref         string `json:"ref,omitempty"`
	Subdir      string `json:"subdir,omitempty"`
	InstalledAt string `json:"installed_at"`
}

func Install(options InstallOptions) (string, error) {
	repository, defaultName, err := normalizeRepository(options.Repository, options.allowFile)
	if err != nil {
		return "", err
	}
	if options.Name == "" {
		options.Name = defaultName
	}
	if err := validateName(options.Name); err != nil {
		return "", err
	}
	if err := validateRef(options.Ref); err != nil {
		return "", err
	}
	subdir, err := validateSubdir(options.Subdir)
	if err != nil {
		return "", err
	}

	temp, err := os.MkdirTemp("", "gokub-template-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(temp)
	checkout := filepath.Join(temp, "repository")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	args := []string{"clone", "--depth", "1"}
	if options.Ref != "" {
		args = append(args, "--branch", options.Ref, "--single-branch")
	}
	args = append(args, "--", repository, checkout)
	command := exec.CommandContext(ctx, "git", args...)
	if output, err := command.CombinedOutput(); err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("clone template repository: %w", ctx.Err())
		}
		return "", fmt.Errorf("clone template repository: %w: %s", err, strings.TrimSpace(string(output)))
	}
	source := checkout
	if subdir != "" {
		source = filepath.Join(checkout, subdir)
	}
	info, err := os.Stat(source)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("template subdirectory %q does not exist", options.Subdir)
	}
	installed, err := Add(options.Name, source)
	if err != nil {
		return "", err
	}
	destination, err := storedPath(installed)
	if err != nil {
		return "", err
	}
	metadata := Metadata{Repository: repository, Ref: options.Ref, Subdir: filepath.ToSlash(subdir), InstalledAt: time.Now().UTC().Format(time.RFC3339)}
	content, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", err
	}
	content = append(content, '\n')
	if err := os.WriteFile(filepath.Join(destination, metadataFile), content, 0o644); err != nil {
		_ = os.RemoveAll(destination)
		return "", err
	}
	return installed, nil
}

func normalizeRepository(value string, allowFile bool) (string, string, error) {
	value = strings.TrimSpace(value)
	if githubShorthand.MatchString(value) {
		parts := strings.Split(value, "/")
		repositoryName := strings.TrimSuffix(parts[1], ".git")
		if !validGitHubComponent(parts[0]) || !validGitHubComponent(repositoryName) {
			return "", "", fmt.Errorf("invalid GitHub owner or repository name")
		}
		return "https://github.com/" + parts[0] + "/" + repositoryName + ".git", repositoryName, nil
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" && parsed.Scheme != "file" {
		return "", "", fmt.Errorf("repository must be owner/repo or an HTTPS GitHub URL")
	}
	if parsed.Scheme == "file" && allowFile {
		name := strings.TrimSuffix(filepath.Base(parsed.Path), ".git")
		return parsed.String(), name, nil
	}
	if parsed.Scheme != "https" || !strings.EqualFold(parsed.Hostname(), "github.com") || parsed.User != nil {
		return "", "", fmt.Errorf("repository must be an HTTPS github.com URL without embedded credentials")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("GitHub repository URL must contain owner/repository")
	}
	name := strings.TrimSuffix(parts[1], ".git")
	if !validGitHubComponent(parts[0]) || !validGitHubComponent(name) {
		return "", "", fmt.Errorf("invalid GitHub owner or repository name")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = "/" + parts[0] + "/" + name + ".git"
	return parsed.String(), name, nil
}

func validGitHubComponent(value string) bool {
	return value != "." && value != ".." && githubComponent.MatchString(value)
}

func validateRef(ref string) error {
	if strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\x00\r\n\t ") {
		return fmt.Errorf("invalid Git reference %q", ref)
	}
	return nil
}

func validateSubdir(value string) (string, error) {
	if value == "" || value == "." {
		return "", nil
	}
	clean := filepath.Clean(value)
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("template subdirectory must stay inside the repository")
	}
	return clean, nil
}
