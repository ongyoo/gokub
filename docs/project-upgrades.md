# Project Upgrades

GOKUB versions each generated project through `schema_version` and
`generator_version` in `.gokub.yaml`. Check whether an older project needs a
migration:

```bash
gokub upgrade --check
gokub upgrade --json
```

Run the interactive migration:

```bash
gokub upgrade
```

Use Up/Down and Enter to confirm. For CI or a controlled automation step:

```bash
gokub upgrade --yes
gokub upgrade --yes --json
```

## Safety Contract

- GOKUB shows the complete migration plan before confirmation.
- A `0600` backup is written beside the manifest before any change.
- A migration is idempotent and does nothing when the project is current.
- A CLI refuses manifests with a newer schema to prevent unsafe downgrades.
- The initial schema migration updates only `.gokub.yaml`; it does not overwrite
  business code, generated adapters, configuration, or dependencies.

Review and remove `.gokub.yaml.bak` after confirming the migration. If a previous
backup exists, GOKUB creates a timestamped backup instead of replacing it.

AI agents can inspect or apply the same plan through the `gokub_project_upgrade`
MCP tool. Applying through MCP follows the same backup contract.
