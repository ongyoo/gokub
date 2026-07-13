const assert = require("node:assert/strict");
const fs = require("node:fs");

const manifest = JSON.parse(fs.readFileSync("package.json", "utf8"));
const source = fs.readFileSync("extension.js", "utf8");
const declared = manifest.contributes.commands.map((item) => item.command).sort();
const registered = [...source.matchAll(/^    "(gokub\.[^"]+)":/gm)].map((match) => match[1]).sort();

assert.deepEqual(registered, declared, "contributed commands and registered handlers must match");
assert.match(source, /status\.command = "gokub\.open"/, "status bar must open the context-aware command center");
assert.match(source, /process\.platform === "win32"/, "terminal arguments must support PowerShell quoting");

console.log(`Verified ${declared.length} VS Code command handlers.`);
