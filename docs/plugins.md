# Plugins

GOKUB plugins are explicit local executables with a versioned
`gokub-plugin.json` manifest. They extend workflows without changing generated
project ownership or loading code into the GOKUB process.

## Create

```bash
gokub plugin create api-audit
cd gokub-plugin-api-audit
make build
```

The scaffold contains a Go module, example command, Makefile, README, and plugin
manifest. Customize the declared commands and implementation, then build the
entrypoint configured by the manifest.

## Install And Run

```bash
gokub plugin install .
gokub plugin list
gokub plugin run api-audit hello
gokub plugin remove api-audit
```

When only one command is declared, `gokub plugin run api-audit` selects it
automatically.

## Manifest

```json
{
  "schema_version": 1,
  "name": "api-audit",
  "version": "0.1.0",
  "description": "Audit an API project",
  "entrypoint": "bin/gokub-plugin-api-audit",
  "commands": [
    { "name": "audit", "description": "Run API checks" }
  ]
}
```

The entrypoint receives its selected command as the first argument and inherits
stdin, stdout, and stderr. GOKUB provides:

```text
GOKUB_PROJECT_ROOT
GOKUB_PLUGIN_NAME
GOKUB_PLUGIN_VERSION
```

## Package And Verify

Create a reproducible platform-specific archive and SHA-256 checksum:

```bash
gokub plugin pack .
gokub plugin verify dist/gokub-plugin-api-audit_0.1.0_darwin_arm64.tar.gz
```

Use `--output <directory>` to choose another artifact directory. Packaging sorts
paths, normalizes archive timestamps and ownership, excludes secrets and local
state, and preserves executable permissions. The adjacent `.sha256` file can be
published with the archive in a GitHub Release.

SHA-256 proves artifact integrity only. It does not prove who published the plugin.
Remote installation remains disabled until GOKUB has signing, provenance,
compatibility, and revocation policies.

## Security Model

- Installation validates the manifest, command names, and entrypoint.
- Entrypoints must be regular executable files inside the plugin directory.
- Symlinks, `.git`, `.env`, dependencies, and build output are not copied.
- Installing a plugin never executes it.
- Execution requires the explicit `gokub plugin run` command.
- MCP can list plugin metadata but cannot execute plugins.

Plugins are executable code. Review and build third-party source yourself before
installing it. Remote publishing, signatures, and a searchable registry remain
future work; the current release intentionally installs only local built folders.
