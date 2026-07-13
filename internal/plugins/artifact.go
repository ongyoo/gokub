package plugins

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Artifact struct {
	Archive      string `json:"archive"`
	ChecksumFile string `json:"checksum_file"`
	SHA256       string `json:"sha256"`
}

func Pack(source, outputDir string) (Artifact, error) {
	manifest, err := readManifest(filepath.Join(source, ManifestFile))
	if err != nil {
		return Artifact{}, err
	}
	entrypoint, err := safeEntrypoint(source, manifest.Entrypoint)
	if err != nil {
		return Artifact{}, err
	}
	if err := executableFile(entrypoint); err != nil {
		return Artifact{}, err
	}
	if outputDir == "" {
		outputDir = filepath.Join(source, "dist")
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Artifact{}, err
	}
	base := fmt.Sprintf("gokub-plugin-%s_%s_%s_%s.tar.gz", manifest.Name, manifest.Version, runtime.GOOS, runtime.GOARCH)
	archivePath := filepath.Join(outputDir, base)
	files, err := packageFiles(source)
	if err != nil {
		return Artifact{}, err
	}
	if err := writeArchive(source, archivePath, files); err != nil {
		_ = os.Remove(archivePath)
		return Artifact{}, err
	}
	digest, err := fileSHA256(archivePath)
	if err != nil {
		return Artifact{}, err
	}
	checksumPath := archivePath + ".sha256"
	if err := os.WriteFile(checksumPath, []byte(digest+"  "+base+"\n"), 0o644); err != nil {
		return Artifact{}, err
	}
	return Artifact{Archive: archivePath, ChecksumFile: checksumPath, SHA256: digest}, nil
}

func Verify(archivePath, checksumPath string) (Artifact, error) {
	if checksumPath == "" {
		checksumPath = archivePath + ".sha256"
	}
	content, err := os.ReadFile(checksumPath)
	if err != nil {
		return Artifact{}, fmt.Errorf("read checksum: %w", err)
	}
	fields := strings.Fields(string(content))
	if len(fields) != 2 || fields[1] != filepath.Base(archivePath) {
		return Artifact{}, fmt.Errorf("invalid checksum file format or artifact name")
	}
	expected, err := hex.DecodeString(fields[0])
	if err != nil || len(expected) != sha256.Size {
		return Artifact{}, fmt.Errorf("invalid SHA-256 checksum")
	}
	digest, err := fileSHA256(archivePath)
	if err != nil {
		return Artifact{}, err
	}
	actual, _ := hex.DecodeString(digest)
	if subtle.ConstantTimeCompare(expected, actual) != 1 {
		return Artifact{}, fmt.Errorf("plugin artifact checksum mismatch")
	}
	return Artifact{Archive: archivePath, ChecksumFile: checksumPath, SHA256: digest}, nil
}

func packageFiles(source string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == source {
			return nil
		}
		name := entry.Name()
		excluded := name == ".git" || name == "node_modules" || name == "dist" || name == ".DS_Store" || strings.HasPrefix(name, ".env")
		if excluded {
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
		if info.Mode().IsRegular() {
			relative, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			files = append(files, relative)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func writeArchive(source, destination string, files []string) error {
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	gzipWriter := gzip.NewWriter(file)
	gzipWriter.Header.ModTime = time.Unix(0, 0).UTC()
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)
	closeAll := func() error {
		if err := tarWriter.Close(); err != nil {
			return err
		}
		if err := gzipWriter.Close(); err != nil {
			return err
		}
		return file.Close()
	}
	for _, relative := range files {
		path := filepath.Join(source, relative)
		info, err := os.Stat(path)
		if err != nil {
			_ = file.Close()
			return err
		}
		header := &tar.Header{
			Name: filepath.ToSlash(relative), Mode: int64(info.Mode().Perm()), Size: info.Size(),
			ModTime: time.Unix(0, 0).UTC(), AccessTime: time.Unix(0, 0).UTC(), ChangeTime: time.Unix(0, 0).UTC(),
			Typeflag: tar.TypeReg,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			_ = file.Close()
			return err
		}
		input, err := os.Open(path)
		if err != nil {
			_ = file.Close()
			return err
		}
		_, copyErr := io.Copy(tarWriter, input)
		closeErr := input.Close()
		if copyErr != nil {
			_ = file.Close()
			return copyErr
		}
		if closeErr != nil {
			_ = file.Close()
			return closeErr
		}
	}
	return closeAll()
}

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
