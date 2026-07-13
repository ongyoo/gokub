package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return fn(request)
}

func TestCheckAndApply(t *testing.T) {
	archive := releaseArchive(t, []byte("new-binary"))
	digest := sha256.Sum256(archive)
	archiveName := "gokub_1.2.0_Darwin_arm64.tar.gz"
	client := updateClient(t, archiveName, archive, hex.EncodeToString(digest[:]))
	executable := filepath.Join(t.TempDir(), "gokub")
	if err := os.WriteFile(executable, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := Check(Options{
		Repository: "ongyoo/gokub", CurrentVersion: "1.1.0", Executable: executable,
		GOOS: "darwin", GOARCH: "arm64", APIBase: "https://api.test", Client: client, allowHTTP: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Update || plan.Latest != "1.2.0" || plan.ArchiveName != archiveName {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	result, err := Apply(plan, client)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new-binary" || result.Installed != "1.2.0" {
		t.Fatalf("update was not applied: %+v %q", result, content)
	}
}

func TestApplyRejectsChecksumMismatchWithoutReplacing(t *testing.T) {
	archive := releaseArchive(t, []byte("new-binary"))
	archiveName := "gokub_1.2.0_Linux_x86_64.tar.gz"
	client := updateClient(t, archiveName, archive, strings.Repeat("0", 64))
	executable := filepath.Join(t.TempDir(), "gokub")
	if err := os.WriteFile(executable, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	plan, err := Check(Options{
		Repository: "ongyoo/gokub", CurrentVersion: "1.1.0", Executable: executable,
		GOOS: "linux", GOARCH: "amd64", APIBase: "https://api.test", Client: client, allowHTTP: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(plan, client); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("checksum mismatch accepted: %v", err)
	}
	content, _ := os.ReadFile(executable)
	if string(content) != "old-binary" {
		t.Fatalf("executable changed after failed update: %q", content)
	}
}

func TestCheckCurrentVersionDoesNotRequireAssets(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return httpResponse(http.StatusOK, `{"tag_name":"v1.2.0","assets":[]}`), nil
	})}
	plan, err := Check(Options{Repository: "ongyoo/gokub", CurrentVersion: "1.2.0", GOOS: "darwin", GOARCH: "arm64", Client: client})
	if err != nil || plan.Update {
		t.Fatalf("current release reported incorrectly: %+v %v", plan, err)
	}
}

func TestValidateDownloadURLAllowsGitHubReleaseRedirects(t *testing.T) {
	for _, location := range []string{
		"https://github.com/ongyoo/gokub/releases/download/v1.2.0/gokub.tar.gz",
		"https://release-assets.githubusercontent.com/github-production-release-asset/file",
		"https://objects.githubusercontent.com/github-production-release-asset/file",
	} {
		if err := validateDownloadURL(location, false); err != nil {
			t.Fatalf("trusted GitHub URL %s was rejected: %v", location, err)
		}
	}
	for _, location := range []string{
		"http://github.com/ongyoo/gokub/releases/file",
		"https://github.com.evil.example/file",
		"https://raw.githubusercontent.com/ongyoo/gokub/main/file",
	} {
		if err := validateDownloadURL(location, false); err == nil {
			t.Fatalf("untrusted URL %s was accepted", location)
		}
	}
}

func TestDownloadValidatesFinalRedirectHost(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		finalURL, err := url.Parse("https://release-assets.githubusercontent.com/github-production-release-asset/file")
		if err != nil {
			t.Fatal(err)
		}
		response := httpResponse(http.StatusOK, "artifact")
		response.Request = &http.Request{URL: finalURL}
		return response, nil
	})}
	content, err := download(client, "https://github.com/ongyoo/gokub/releases/download/v1.2.0/file", 100, false)
	if err != nil || string(content) != "artifact" {
		t.Fatalf("trusted final redirect failed: %q %v", content, err)
	}

	client.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		finalURL, err := url.Parse("https://downloads.evil.example/file")
		if err != nil {
			t.Fatal(err)
		}
		response := httpResponse(http.StatusOK, "artifact")
		response.Request = &http.Request{URL: finalURL}
		return response, nil
	})
	if _, err := download(client, "https://github.com/ongyoo/gokub/releases/download/v1.2.0/file", 100, false); err == nil {
		t.Fatal("untrusted final redirect was accepted")
	}
}

func updateClient(t *testing.T, archiveName string, archive []byte, checksum string) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		switch request.URL.String() {
		case "https://api.test/repos/ongyoo/gokub/releases/latest":
			body := fmt.Sprintf(`{"tag_name":"v1.2.0","assets":[{"name":%q,"browser_download_url":"http://assets.test/archive"},{"name":"checksums.txt","browser_download_url":"http://assets.test/checksums"}]}`, archiveName)
			return httpResponse(http.StatusOK, body), nil
		case "http://assets.test/archive":
			return httpResponseBytes(http.StatusOK, archive), nil
		case "http://assets.test/checksums":
			return httpResponse(http.StatusOK, checksum+"  "+archiveName+"\n"), nil
		default:
			t.Fatalf("unexpected request: %s", request.URL.String())
			return nil, nil
		}
	})}
}

func releaseArchive(t *testing.T, binary []byte) []byte {
	t.Helper()
	var output bytes.Buffer
	gzipWriter := gzip.NewWriter(&output)
	gzipWriter.Header.ModTime = time.Unix(0, 0)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{Name: "gokub", Mode: 0o755, Size: int64(len(binary)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tarWriter.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func httpResponse(status int, body string) *http.Response {
	return httpResponseBytes(status, []byte(body))
}

func httpResponseBytes(status int, body []byte) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: make(http.Header), Body: io.NopCloser(bytes.NewReader(body))}
}
