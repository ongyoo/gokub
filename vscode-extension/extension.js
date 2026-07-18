const vscode = require("vscode");

function executable() {
  return vscode.workspace.getConfiguration("gokub").get("executablePath", "gokub");
}

function shellQuote(value) {
  if (process.platform === "win32") {
    return `'${String(value).replaceAll("'", "''")}'`;
  }
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

async function installTemplate() {
  const repository = await vscode.window.showInputBox({
    title: "Community template repository",
    prompt: "GitHub owner/repository or HTTPS URL",
    placeHolder: "owner/go-service-template"
  });
  if (!repository) return;
  runInteractive(["template", "install", repository], "GOKUB Community Templates");
}

async function searchTemplates() {
  const query = await vscode.window.showInputBox({
    title: "Search community templates",
    prompt: "Optional keywords; leave empty to browse popular templates",
    placeHolder: "api postgres"
  });
  if (query === undefined) return;
  const args = ["template", "search"];
  if (query.trim()) args.push(query.trim());
  args.push("--install");
  runInteractive(args, "GOKUB Community Templates");
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

async function generateModel() {
  const selected = await vscode.window.showOpenDialog({
    canSelectFiles: true,
    canSelectFolders: false,
    canSelectMany: false,
    filters: { JSON: ["json"] },
    openLabel: "Generate Go Model"
  });
  if (!selected?.length) return;
  const suggested = selected[0].path.split("/").at(-1).replace(/(?:\.schema)?\.json$/i, "");
  const name = await vscode.window.showInputBox({
    title: "Go model name",
    value: suggested,
    prompt: "Example: user or payment-event"
  });
  if (!name) return;
  runInteractive(["add", "model", name, "--from", selected[0].fsPath], "GOKUB Model Generator");
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

async function initProject() {
  if (!workspacePath()) {
    vscode.window.showWarningMessage("Open a Go project folder before initializing GOKUB.");
    return;
  }
  const provider = await vscode.window.showQuickPick(
    ["all", "codex", "claude", "copilot", "gemini", "portable"],
    { title: "Initialize project for AI agents", placeHolder: "Select an agent target" }
  );
  if (!provider) return;
  runInteractive(["init", "--provider", provider], "GOKUB Initialize Project");
}

async function createPlugin() {
  const name = await vscode.window.showInputBox({
    title: "Plugin name",
    prompt: "Lowercase letters, numbers, and hyphens",
    placeHolder: "api-audit"
  });
  if (!name) return;
  runInteractive(["plugin", "create", name], "GOKUB Plugins");
}

async function installPlugin() {
  const selected = await vscode.window.showOpenDialog({
    canSelectFiles: false,
    canSelectFolders: true,
    canSelectMany: false,
    openLabel: "Install GOKUB Plugin"
  });
  if (!selected?.length) return;
  runInteractive(["plugin", "install", selected[0].fsPath], "GOKUB Plugins");
}

async function packPlugin() {
  const selected = await vscode.window.showOpenDialog({
    canSelectFiles: false,
    canSelectFolders: true,
    canSelectMany: false,
    openLabel: "Package GOKUB Plugin"
  });
  if (!selected?.length) return;
  runInteractive(["plugin", "pack", selected[0].fsPath], "GOKUB Plugins");
}

async function qualityGate() {
  const minimum = await vscode.window.showInputBox({
    title: "Minimum GOKUB project score",
    value: "80",
    validateInput: (value) => {
      const score = Number(value);
      return Number.isInteger(score) && score >= 0 && score <= 100 ? undefined : "Enter an integer from 0 to 100.";
    }
  });
  if (minimum === undefined) return;
  runInteractive(["score", "--fail-under", minimum], "GOKUB Quality Gate");
}

function activate(context) {
  const commands = {
    "gokub.open": () => runInteractive([], "GOKUB Command Center"),
    "gokub.initProject": initProject,
    "gokub.newProject": () => runInteractive(["new"], "GOKUB New Project"),
    "gokub.addTemplate": addTemplate,
    "gokub.installTemplate": installTemplate,
    "gokub.searchTemplates": searchTemplates,
    "gokub.listTemplates": () => runInteractive(["template", "list"], "GOKUB Templates"),
    "gokub.addFeature": addFeature,
    "gokub.generateModel": generateModel,
    "gokub.status": () => runInteractive(["status"], "GOKUB Status"),
    "gokub.doctor": () => runInteractive(["doctor"], "GOKUB Doctor"),
    "gokub.score": () => runInteractive(["score"], "GOKUB Project Score"),
    "gokub.qualityGate": qualityGate,
    "gokub.graph": () => runInteractive(["graph"], "GOKUB Dependency Graph"),
    "gokub.architectureCheck": () => runInteractive(["graph", "--check"], "GOKUB Architecture Check"),
    "gokub.upgrade": () => runInteractive(["upgrade"], "GOKUB Project Upgrade"),
    "gokub.updateCLI": () => runInteractive(["update"], "GOKUB CLI Update"),
    "gokub.openMCPConfig": openMCPConfig,
    "gokub.installSkills": installSkills,
    "gokub.createPlugin": createPlugin,
    "gokub.installPlugin": installPlugin,
    "gokub.packPlugin": packPlugin,
    "gokub.listPlugins": () => runInteractive(["plugin", "list"], "GOKUB Plugins"),
    "gokub.uninstall": () => runInteractive(["uninstall"], "GOKUB Uninstall")
  };
  for (const [name, handler] of Object.entries(commands)) {
    context.subscriptions.push(vscode.commands.registerCommand(name, handler));
  }

  const status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left, 50);
  status.text = "$(tools) GOKUB";
  status.tooltip = "Open GOKUB Command Center";
  status.command = "gokub.open";
  status.show();
  context.subscriptions.push(status);
}

function deactivate() {}

module.exports = { activate, deactivate };
