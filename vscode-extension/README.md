# GOKUB for VS Code

Release builds take their extension version from the matching GOKUB Git tag and are
attached to the GitHub release as `gokub-vscode.vsix`.

Run GOKUB workflows from the Command Palette while keeping the interactive CLI
wizard in VS Code's integrated terminal.

## Commands

- `GOKUB: Initialize Existing Project`
- `GOKUB: New Project`
- `GOKUB: Add Custom Template`
- `GOKUB: Install Community Template`
- `GOKUB: Search Community Templates`
- `GOKUB: List Custom Templates`
- `GOKUB: Add Feature`
- `GOKUB: Generate Model from JSON`
- `GOKUB: Project Status`
- `GOKUB: Run Doctor`
- `GOKUB: Project Score`
- `GOKUB: Dependency Graph`
- `GOKUB: Upgrade Project`
- `GOKUB: Update CLI`
- `GOKUB: Open MCP Configuration`
- `GOKUB: Install Agent Skills`
- `GOKUB: Create Plugin`
- `GOKUB: Install Plugin`
- `GOKUB: Package Plugin`
- `GOKUB: List Plugins`
- `GOKUB: Uninstall CLI`

Install the GOKUB CLI first with `make install` or configure
`gokub.executablePath` with the path to the binary.

Open an existing Go workspace and run `GOKUB: Initialize Existing Project` to
detect its module settings, create `gokub.init`, and install agent skills and MCP
configuration without replacing application source or existing instructions.

Build the extension from this directory with `npm install && npm run package`.
