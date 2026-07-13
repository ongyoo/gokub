# Develop GOKUB

## Requirements

- Go 1.22 or newer for the GOKUB CLI.
- Node.js 20 and npm only when packaging the VS Code extension.

CI verifies the CLI with both the minimum Go 1.22 toolchain and the Go 1.25
release toolchain. Generated projects default to Go 1.26 independently; see the
[Go version policy](go-versions.md).

## Build And Test

```bash
make fmt
make test
make build
make extension
```

Run the same preflight used by CI and releases with:

```bash
make verify
```

Artifacts are written to:

```text
dist/gokub
dist/gokub-vscode.vsix
```

Install the current source build locally with:

```bash
make install
```

## Release

GoReleaser builds macOS and Linux archives for Intel and ARM, injects the version
source commit, build date, and `ongyoo/gokub` repository metadata, and writes
`checksums.txt`. The installer
and standalone self-updater use those checksums before replacing the local binary.
The release workflow verifies all four archive checksums and the embedded Linux
binary provenance before rendering package-manager metadata.

Push a SemVer tag such as `v0.2.0` to start the release workflow. Before publishing,
the workflow checks formatting, module-file cleanliness, race-enabled tests, vet,
installer security, and packaging. GoReleaser does not modify module files during
packaging.
Linux CI also opens the interactive Command Center through a real pseudo-terminal,
sends arrow-key input, and verifies that the selected workflow exits normally.
After GoReleaser publishes the archives, the workflow renders `gokub.rb` from the
release checksums and attaches it to the same release for the Homebrew tap.
The workflow also sets the VS Code extension version from the release tag, packages
the VSIX, and attaches it to the release. The tracked extension version is only a
development baseline and does not need a manual bump for CLI releases.
When `HOMEBREW_TAP_TOKEN` is configured, it also updates
`ongyoo/homebrew-tap/Formula/gokub.rb`. Use a fine-grained token scoped to that tap
with Contents read/write permission; the default workflow token cannot write to a
different repository.

Homebrew packaging instructions are in
[`packaging/homebrew/README.md`](../packaging/homebrew/README.md).

## Project References

- [Template architecture](template-architecture.md)
- [Reference audit](reference-audit.md)
- [Custom-template guide](custom-templates.md)
- [Agent-skills guide](agent-skills.md)
