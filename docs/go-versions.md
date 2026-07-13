# Go Version Policy

GOKUB keeps the CLI toolchain and generated-project toolchain separate.

| Use | Version | Guidance |
|---|---:|---|
| New projects | Go 1.26 | Recommended default and latest supported release line |
| Existing teams | Go 1.25 | Supported conservative baseline |
| GOKUB CLI source | Go 1.22+ | Minimum version used to build the CLI itself |
| Go 1.24 and older | Team-defined | Upstream support has ended; use temporarily while planning an upgrade |

For teams carrying older services, Go 1.24, 1.23, and 1.22 are all legacy
baselines under the current upstream policy. GOKUB accepts them as custom values
and reports upgrade guidance instead of blocking generation. Versions older than
the GOKUB CLI's Go 1.22 source baseline should be treated as migration-only choices.

The Go project supports each major release until two newer major releases are
available. GOKUB's built-in policy is a release-time snapshot of the official
[Go release history and support policy](https://go.dev/doc/devel/release), not a
network lookup performed whenever the CLI runs.

## Select A Version

The interactive wizard offers Go 1.26, Go 1.25, and a custom value. For automation:

```bash
gokub new example-api --go-version 1.26
gokub new legacy-api --go-version 1.24
```

Use `major.minor` format. Do not include `go`, a patch version, or a toolchain
suffix. GOKUB writes the selected value to all version-sensitive generated files:

- `.gokub.yaml`
- `go.mod`
- `Dockerfile`
- `.github/workflows/ci.yml`

Run `gokub status` to see the selected version and lifecycle guidance. Run
`gokub doctor` to detect drift between `.gokub.yaml` and `go.mod`.

## Choosing A Baseline

Choose Go 1.26 for a new service unless deployment infrastructure or an approved
organization toolchain requires otherwise. Choose Go 1.25 when the team needs a
supported baseline with more adoption time. A custom older version is useful for
migration work, but verify that every pinned dependency still supports that
toolchain.

`gokub upgrade` preserves the version found in an existing `go.mod`; it does not
silently move an older project to the recommended release. Change the version
explicitly when the project and CI are ready.
