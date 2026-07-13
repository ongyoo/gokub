# AI And IDE Integrations

## AI Project Files

Generated projects include guidance and configuration for common coding agents:

- `AGENTS.md` for Codex and compatible agents.
- `CLAUDE.md` for Claude Code.
- `GEMINI.md` for Gemini.
- `.github/copilot-instructions.md` for GitHub Copilot.
- `.codex/config.toml` and `.mcp.json` for MCP clients.

Create or refresh these files in an existing project:

```bash
gokub agent init
gokub agent init --provider codex
gokub agent init --provider claude
```

## Agent Skills

```bash
gokub skill install
gokub skill install --agent codex
gokub skill install --agent claude
gokub skill install --agent copilot
gokub skill list
```

The pack includes `gokub-project`, `gokub-add-domain`, and
`gokub-verify-change`. See the [Agent skills guide](agent-skills.md) for paths,
updates, targeted installation, and removal.

## MCP

Start the JSON-RPC stdio server with:

```bash
gokub mcp serve
```

The server exposes project status, doctor checks, health scoring, dependency graphs,
catalog discovery, feature and JSON model generation, project upgrade planning,
community-template search and installation, recipe application, and skill
installation. MCP mode never writes the terminal logo to stdout, keeping the
protocol stream valid.

MCP can list installed plugin manifests through `gokub_plugins`, but it cannot run
plugin executables. Plugin execution remains an explicit terminal approval boundary.

## VS Code

Install the packaged extension during local development:

```bash
code --install-extension dist/gokub-vscode.vsix
```

The Command Palette provides project creation, custom templates, feature addition,
status, doctor, MCP configuration, skill installation, and CLI uninstall. Configure
`gokub.executablePath` if `gokub` is not on `PATH`.

## JetBrains IDEs

Generated projects contain run/debug configurations compatible with GoLand and
IntelliJ with the Go plugin. Open the generated project root, select the desired
entrypoint, and run or debug it from the IDE toolbar.
