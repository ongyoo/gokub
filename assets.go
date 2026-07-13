package gokub

import "embed"

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
