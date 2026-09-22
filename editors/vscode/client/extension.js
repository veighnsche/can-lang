// can-lang VSCode/Cursor client: coloring comes from the TextMate grammar,
// squiggles come from `canlc lsp` over stdio (see compiler/lsp.go).
// Server resolution: `canlc.serverPath` setting, else bundled bin/canlc,
// else `canlc` on PATH.
const fs = require("fs");
const path = require("path");
const vscode = require("vscode");
const { LanguageClient, TransportKind } = require("vscode-languageclient/node");

let client;

function serverCommand(context) {
  const configured = vscode.workspace.getConfiguration("canlc").get("serverPath", "");
  if (configured) {
    return configured;
  }
  const bundled = path.join(context.extensionPath, "bin", "canlc");
  if (fs.existsSync(bundled)) {
    return bundled;
  }
  return "canlc";
}

function activate(context) {
  const command = serverCommand(context);
  client = new LanguageClient("canlc", "can-lang", {
    command,
    args: ["lsp"],
    transport: TransportKind.stdio,
  }, {
    documentSelector: [{ scheme: "file", language: "can" }],
  });
  client.start();
  context.subscriptions.push({ dispose: () => client && client.stop() });
}

function deactivate() {
  return client ? client.stop() : undefined;
}

module.exports = { activate, deactivate };
