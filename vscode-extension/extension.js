const vscode = require("vscode");

function executable() {
  return vscode.workspace.getConfiguration("gokub").get("executablePath", "gokub");
}

function shellQuote(value) {
  return `'${String(value).replaceAll("'", `'"'"'`)}'`;
}

function workspacePath() {
  return vscode.workspace.workspaceFolders?.[0]?.uri.fsPath;
}

function terminal(name = "GOKUB") {
  const existing = vscode.window.terminals.find((item) => item.name === name);
  return existing || vscode.window.createTerminal({ name, cwd: workspacePath() });
}

function runInteractive(args, name) {
  const instance = terminal(name);
  instance.show();
  instance.sendText([shellQuote(executable()), ...args.map(shellQuote)].join(" "));
}

async function addTemplate() {
  const selected = await vscode.window.showOpenDialog({
    canSelectFiles: false,
    canSelectFolders: true,
    canSelectMany: false,
    openLabel: "Add GOKUB Template"
  });
  if (!selected?.length) return;
  const suggested = selected[0].path.split("/").filter(Boolean).at(-1);
  const name = await vscode.window.showInputBox({
    title: "Template name",
    value: suggested,
    prompt: "This name will appear in the GOKUB project wizard."
  });
  if (!name) return;
  runInteractive(["template", "add", name, selected[0].fsPath], "GOKUB Templates");
}

async function addFeature() {
  const feature = await vscode.window.showQuickPick(
    ["auth", "crud", "redis", "postgres", "mongodb", "kafka", "rabbitmq", "nats", "grpc", "cron", "email", "websocket", "otel", "docker", "github-actions"],
    { title: "Add GOKUB feature", placeHolder: "Select a feature" }
  );
  if (!feature) return;
  const args = ["add", feature];
  if (feature === "crud") {
    const name = await vscode.window.showInputBox({ title: "Resource name", prompt: "Example: product" });
    if (!name) return;
    args.push(name);
  }
  runInteractive(args, "GOKUB Features");
}

async function openMCPConfig() {
  const root = workspacePath();
  if (!root) return;
  for (const relative of [".codex/config.toml", ".mcp.json"]) {
    const uri = vscode.Uri.joinPath(vscode.Uri.file(root), relative);
    try {
      await vscode.workspace.fs.stat(uri);
      await vscode.window.showTextDocument(uri, { preview: false });
      return;
    } catch (_) {}
  }
  vscode.window.showWarningMessage("No GOKUB MCP configuration found in this workspace.");
}

async function installSkills() {
  const agent = await vscode.window.showQuickPick(
    ["all", "codex", "claude", "copilot", "gemini", "portable"],
    { title: "Install GOKUB agent skills", placeHolder: "Select an agent target" }
  );
  if (!agent) return;
  runInteractive(["skill", "install", "--agent", agent], "GOKUB Agent Skills");
}

function activate(context) {
  const commands = {
    "gokub.newProject": () => runInteractive(["new"], "GOKUB New Project"),
    "gokub.addTemplate": addTemplate,
    "gokub.listTemplates": () => runInteractive(["template", "list"], "GOKUB Templates"),
    "gokub.addFeature": addFeature,
    "gokub.status": () => runInteractive(["status"], "GOKUB Status"),
    "gokub.doctor": () => runInteractive(["doctor"], "GOKUB Doctor"),
    "gokub.openMCPConfig": openMCPConfig,
    "gokub.installSkills": installSkills,
    "gokub.uninstall": () => runInteractive(["uninstall"], "GOKUB Uninstall")
  };
  for (const [name, handler] of Object.entries(commands)) {
    context.subscriptions.push(vscode.commands.registerCommand(name, handler));
  }

  const status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 50);
  status.text = "$(tools) GOKUB";
  status.tooltip = "Create a GOKUB project";
  status.command = "gokub.newProject";
  status.show();
  context.subscriptions.push(status);
}

function deactivate() {}

module.exports = { activate, deactivate };
