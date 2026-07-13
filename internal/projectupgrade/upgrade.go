package projectupgrade

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ongyoo/gokub/internal/goversion"
	"github.com/ongyoo/gokub/internal/manifest"
)

type Plan struct {
	CurrentSchema    int      `json:"current_schema"`
	TargetSchema     int      `json:"target_schema"`
	CurrentGenerator string   `json:"current_generator"`
	TargetGenerator  string   `json:"target_generator"`
	NeedsUpgrade     bool     `json:"needs_upgrade"`
	Changes          []string `json:"changes"`
}

type Result struct {
	Plan       Plan   `json:"plan"`
	BackupPath string `json:"backup_path"`
}

func Check(root, targetVersion string) (Plan, error) {
	m, err := manifest.Read(filepath.Join(root, manifest.FileName))
	if err != nil {
		return Plan{}, fmt.Errorf("read project manifest: %w", err)
	}
	if err := manifest.Validate(m); err != nil {
		return Plan{}, err
	}
	plan := Plan{
		CurrentSchema: m.SchemaVersion, TargetSchema: manifest.CurrentSchemaVersion,
		CurrentGenerator: m.GeneratorVersion, TargetGenerator: targetVersion,
		Changes: []string{},
	}
	if m.SchemaVersion < manifest.CurrentSchemaVersion {
		plan.Changes = append(plan.Changes, fmt.Sprintf("manifest schema %d -> %d", m.SchemaVersion, manifest.CurrentSchemaVersion))
	}
	if m.GoVersion == "" {
		version := projectGoVersion(root)
		plan.Changes = append(plan.Changes, fmt.Sprintf("Go version metadata -> %s", version))
	}
	if m.GeneratorVersion != targetVersion {
		current := m.GeneratorVersion
		if current == "" {
			current = "unversioned"
		}
		plan.Changes = append(plan.Changes, fmt.Sprintf("generator metadata %s -> %s", current, targetVersion))
	}
	plan.NeedsUpgrade = len(plan.Changes) > 0
	return plan, nil
}

func Apply(root, targetVersion string) (Result, error) {
	plan, err := Check(root, targetVersion)
	if err != nil {
		return Result{}, err
	}
	result := Result{Plan: plan}
	if !plan.NeedsUpgrade {
		return result, nil
	}
	path := filepath.Join(root, manifest.FileName)
	content, err := os.ReadFile(path)
	if err != nil {
		return Result{}, err
	}
	backup := path + ".bak"
	if _, err := os.Stat(backup); err == nil {
		backup = fmt.Sprintf("%s.%s.bak", path, time.Now().UTC().Format("20060102T150405.000000000Z"))
	} else if !os.IsNotExist(err) {
		return Result{}, err
	}
	if err := os.WriteFile(backup, content, 0o600); err != nil {
		return Result{}, fmt.Errorf("backup manifest: %w", err)
	}
	m, err := manifest.Read(path)
	if err != nil {
		return Result{}, err
	}
	m.SchemaVersion = manifest.CurrentSchemaVersion
	m.GeneratorVersion = targetVersion
	if m.GoVersion == "" {
		m.GoVersion = projectGoVersion(root)
	}
	if err := manifest.Write(path, m); err != nil {
		return Result{}, fmt.Errorf("write upgraded manifest (backup: %s): %w", backup, err)
	}
	result.BackupPath = backup
	return result, nil
}

func projectGoVersion(root string) string {
	content, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err == nil {
		if version := goversion.ParseGoMod(string(content)); version != "" {
			return version
		}
	}
	return goversion.Conservative
}
