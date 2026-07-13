package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	maxArchiveSize  = 200 << 20
	maxChecksumSize = 5 << 20
	maxBinarySize   = 100 << 20
)

type Options struct {
	Repository     string
	CurrentVersion string
	Executable     string
	GOOS           string
	GOARCH         string
	APIBase        string
	Client         *http.Client
	allowHTTP      bool
}

type Plan struct {
	Repository   string `json:"repository"`
	Current      string `json:"current"`
	Latest       string `json:"latest"`
	Update       bool   `json:"update_available"`
	ArchiveName  string `json:"archive_name,omitempty"`
	ArchiveURL   string `json:"-"`
	ChecksumsURL string `json:"-"`
	Executable   string `json:"-"`
	allowHTTP    bool
}

type Result struct {
	Previous   string `json:"previous"`
	Installed  string `json:"installed"`
	Executable string `json:"executable"`
	SHA256     string `json:"sha256"`
}

func Check(options Options) (Plan, error) {
	options = defaults(options)
	if options.Repository == "" || strings.ContainsAny(options.Repository, "\r\n ") {
		return Plan{}, fmt.Errorf("invalid update repository %q", options.Repository)
	}
	archiveName, err := archiveName(options.GOOS, options.GOARCH, "VERSION")
	if err != nil {
		return Plan{}, err
	}
	endpoint := strings.TrimRight(options.APIBase, "/") + "/repos/" + options.Repository + "/releases/latest"
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, endpoint, nil)
	if err != nil {
		return Plan{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "gokub-cli")
	response, err := options.Client.Do(request)
	if err != nil {
		return Plan{}, fmt.Errorf("check latest GOKUB release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return Plan{}, fmt.Errorf("GitHub release check failed: %s", response.Status)
	}
	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxChecksumSize)).Decode(&release); err != nil {
		return Plan{}, fmt.Errorf("decode latest release: %w", err)
	}
	latest := strings.TrimPrefix(release.TagName, "v")
	if latest == "" {
		return Plan{}, fmt.Errorf("latest release did not include a version tag")
	}
	archiveName = strings.Replace(archiveName, "VERSION", latest, 1)
	plan := Plan{Repository: options.Repository, Current: options.CurrentVersion, Latest: latest, ArchiveName: archiveName, Executable: options.Executable, allowHTTP: options.allowHTTP}
	plan.Update = compareVersions(options.CurrentVersion, latest) < 0
	if !plan.Update {
		return plan, nil
	}
	for _, asset := range release.Assets {
		switch asset.Name {
		case archiveName:
			plan.ArchiveURL = asset.URL
		case "checksums.txt":
			plan.ChecksumsURL = asset.URL
		}
	}
	if plan.ArchiveURL == "" || plan.ChecksumsURL == "" {
		return Plan{}, fmt.Errorf("release %s is missing %s or checksums.txt", release.TagName, archiveName)
	}
	if err := validateDownloadURL(plan.ArchiveURL, options.allowHTTP); err != nil {
		return Plan{}, err
	}
	if err := validateDownloadURL(plan.ChecksumsURL, options.allowHTTP); err != nil {
		return Plan{}, err
	}
	return plan, nil
}

func Apply(plan Plan, client *http.Client) (Result, error) {
	if !plan.Update {
		return Result{Previous: plan.Current, Installed: plan.Current, Executable: plan.Executable}, nil
	}
	if runtime.GOOS == "windows" {
		return Result{}, fmt.Errorf("self-update is not supported on Windows; install the new release manually")
	}
	if strings.Contains(filepath.ToSlash(plan.Executable), "/Cellar/") {
		return Result{}, fmt.Errorf("Homebrew installation detected; use brew upgrade gokub")
	}
	if err := validateDownloadURL(plan.ArchiveURL, plan.allowHTTP); err != nil {
		return Result{}, err
	}
	if err := validateDownloadURL(plan.ChecksumsURL, plan.allowHTTP); err != nil {
		return Result{}, err
	}
	info, err := os.Lstat(plan.Executable)
	if err != nil {
		return Result{}, fmt.Errorf("inspect current executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return Result{}, fmt.Errorf("refusing to replace a non-regular or symlinked executable")
	}
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	checksums, err := download(client, plan.ChecksumsURL, maxChecksumSize, plan.allowHTTP)
	if err != nil {
		return Result{}, fmt.Errorf("download checksums: %w", err)
	}
	expected, err := expectedChecksum(checksums, plan.ArchiveName)
	if err != nil {
		return Result{}, err
	}
	archive, err := download(client, plan.ArchiveURL, maxArchiveSize, plan.allowHTTP)
	if err != nil {
		return Result{}, fmt.Errorf("download release archive: %w", err)
	}
	digest := sha256.Sum256(archive)
	if subtle.ConstantTimeCompare(expected, digest[:]) != 1 {
		return Result{}, fmt.Errorf("release archive checksum verification failed")
	}
	binary, mode, err := extractBinary(archive)
	if err != nil {
		return Result{}, err
	}
	directory := filepath.Dir(plan.Executable)
	temporary, err := os.CreateTemp(directory, ".gokub-update-")
	if err != nil {
		return Result{}, fmt.Errorf("create update beside executable: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if _, err := temporary.Write(binary); err != nil {
		_ = temporary.Close()
		return Result{}, err
	}
	if err := temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return Result{}, err
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return Result{}, err
	}
	if err := temporary.Close(); err != nil {
		return Result{}, err
	}
	if err := os.Rename(temporaryPath, plan.Executable); err != nil {
		return Result{}, fmt.Errorf("replace executable atomically: %w", err)
	}
	return Result{Previous: plan.Current, Installed: plan.Latest, Executable: plan.Executable, SHA256: hex.EncodeToString(digest[:])}, nil
}

func defaults(options Options) Options {
	if options.GOOS == "" {
		options.GOOS = runtime.GOOS
	}
	if options.GOARCH == "" {
		options.GOARCH = runtime.GOARCH
	}
	if options.APIBase == "" {
		options.APIBase = "https://api.github.com"
	}
	if options.Client == nil {
		options.Client = &http.Client{Timeout: 15 * time.Second}
	}
	return options
}

func archiveName(goos, goarch, version string) (string, error) {
	osName := map[string]string{"darwin": "Darwin", "linux": "Linux"}[goos]
	archName := map[string]string{"amd64": "x86_64", "arm64": "arm64"}[goarch]
	if osName == "" || archName == "" {
		return "", fmt.Errorf("self-update does not support %s/%s", goos, goarch)
	}
	return fmt.Sprintf("gokub_%s_%s_%s.tar.gz", version, osName, archName), nil
}

func validateDownloadURL(value string, allowHTTP bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid release asset URL")
	}
	if parsed.Scheme != "https" && !(allowHTTP && parsed.Scheme == "http") {
		return fmt.Errorf("release asset URL must use HTTPS")
	}
	if !allowHTTP && !trustedGitHubDownloadHost(parsed.Hostname()) {
		return fmt.Errorf("release asset URL must use a trusted GitHub download host")
	}
	return nil
}

func trustedGitHubDownloadHost(host string) bool {
	switch strings.ToLower(host) {
	case "github.com", "release-assets.githubusercontent.com", "objects.githubusercontent.com":
		return true
	default:
		return false
	}
}

func download(client *http.Client, location string, limit int64, allowHTTP bool) ([]byte, error) {
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, location, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "gokub-cli")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.Request != nil {
		if err := validateDownloadURL(response.Request.URL.String(), allowHTTP); err != nil {
			return nil, fmt.Errorf("unsafe download redirect: %w", err)
		}
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", response.Status)
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("download exceeds %d bytes", limit)
	}
	return content, nil
}

func expectedChecksum(content []byte, filename string) ([]byte, error) {
	for _, line := range strings.Split(string(content), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == filename {
			digest, err := hex.DecodeString(fields[0])
			if err == nil && len(digest) == sha256.Size {
				return digest, nil
			}
		}
	}
	return nil, fmt.Errorf("checksum for %s was not found", filename)
}

func extractBinary(archive []byte) ([]byte, os.FileMode, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, 0, fmt.Errorf("open release archive: %w", err)
	}
	defer gzipReader.Close()
	reader := tar.NewReader(gzipReader)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read release archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != "gokub" || header.Size < 1 || header.Size > maxBinarySize {
			continue
		}
		content, err := io.ReadAll(io.LimitReader(reader, maxBinarySize+1))
		if err != nil || int64(len(content)) != header.Size {
			return nil, 0, fmt.Errorf("read gokub binary from release archive")
		}
		mode := os.FileMode(header.Mode).Perm()
		if mode&0o111 == 0 {
			mode = 0o755
		}
		return content, mode, nil
	}
	return nil, 0, fmt.Errorf("release archive does not contain the gokub executable")
}

func compareVersions(current, latest string) int {
	current = strings.TrimPrefix(current, "v")
	latest = strings.TrimPrefix(latest, "v")
	currentParts := strings.SplitN(current, "-", 2)
	latestParts := strings.SplitN(latest, "-", 2)
	for index := 0; index < 3; index++ {
		currentValue := numericPart(currentParts[0], index)
		latestValue := numericPart(latestParts[0], index)
		if currentValue < latestValue {
			return -1
		}
		if currentValue > latestValue {
			return 1
		}
	}
	if len(currentParts) == 2 && len(latestParts) == 1 {
		return -1
	}
	if len(currentParts) == 1 && len(latestParts) == 2 {
		return 1
	}
	return strings.Compare(current, latest)
}

func numericPart(version string, index int) int {
	parts := strings.Split(version, ".")
	if index >= len(parts) {
		return 0
	}
	value, err := strconv.Atoi(parts[index])
	if err != nil {
		return -1
	}
	return value
}
