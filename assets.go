package gokub

import "embed"

// Assets embeds the product spec and logo for CLI packaging.
//
//go:embed spec.md gokub_logo.png logo.txt skill-packs
var Assets embed.FS

// Version and Repository are replaced by release builds.
var (
	Version    = "0.1.0"
	Repository = "gokub/gokub"
)
