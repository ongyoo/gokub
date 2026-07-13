package gokub

import (
	"embed"
	"regexp"
	"runtime/debug"
	"strings"
)

var pseudoVersionPattern = regexp.MustCompile(`\.\d{14}-[0-9a-f]{12,}(?:\+.*)?$`)

// Assets embeds files required by generated projects and agent integrations.
//
//go:embed gokub_logo.png skill-packs
var Assets embed.FS

// Version and Repository are replaced by release builds.
var (
	Version    = "0.1.0"
	Repository = "ongyoo/gokub"
	Commit     = "dev"
	BuildDate  = "unknown"
)

func init() {
	if Version != "0.1.0" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	Version = releasedModuleVersion(Version, info.Main.Version)
}

func releasedModuleVersion(fallback, moduleVersion string) string {
	if moduleVersion == "" || moduleVersion == "(devel)" || pseudoVersionPattern.MatchString(moduleVersion) {
		return fallback
	}
	return strings.TrimPrefix(moduleVersion, "v")
}
