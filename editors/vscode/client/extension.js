const vscode = require("vscode");
const { LanguageClient, TransportKind, State } = require("vscode-languageclient/node");
const { CanClientController } = require("./controller");

let controller;

async function activate(context) {
  controller = new CanClientController(vscode, LanguageClient, TransportKind, context, State);
  await controller.activate();
}

async function deactivate() {
  if (controller) await controller.deactivate();
  controller = undefined;
}

module.exports = { activate, deactivate };
