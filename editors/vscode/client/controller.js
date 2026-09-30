const fs = require("node:fs");
const path = require("node:path");

function escapeGlobLiteral(value) {
  const escaped = { "[": "[[]", "]": "[]]", "{": "[{]", "}": "[}]", "?": "[?]", "*": "[*]", "!": "[\\!]" };
  return Array.from(value, (character) => escaped[character] || character).join("");
}

class CanClientController {
  constructor(vscode, LanguageClient, TransportKind, context, State) {
    this.vscode = vscode;
    this.LanguageClient = LanguageClient;
    this.TransportKind = TransportKind;
    this.State = State;
    this.context = context;
    this.client = undefined;
    this.build = undefined;
    this.crashMessage = undefined;
    this.intentionalStop = false;
    this.clientSubscriptions = [];
    this.projectMessages = new Map();
    this.command = undefined;
    this.queue = Promise.resolve();
    this.disposed = false;
    this.manifestRefresh = undefined;
    this.output = vscode.window.createOutputChannel("Can language server", { log: true });
    this.status = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Left);
    this.status.command = "canlc.showServerStatus";
    this.status.text = "$(sync~spin) Can";
    this.status.show();
    context.subscriptions.push(this.output, this.status);
  }

  resolveServer() {
    const configured = this.vscode.workspace.getConfiguration("canlc").get("serverPath", "").trim();
    const command = configured || path.join(this.context.extensionPath, "bin", "canlc");
    if (!path.isAbsolute(command)) {
      throw new Error(`Can server path must be absolute: ${command}`);
    }
    let stat;
    try {
      stat = fs.statSync(command);
      fs.accessSync(command, fs.constants.X_OK);
    } catch (error) {
      throw new Error(`Can server is missing or not executable at ${command}: ${error.message}`);
    }
    if (!stat.isFile()) {
      throw new Error(`Can server path is not a file: ${command}`);
    }
    return command;
  }

  updateStatus() {
    if (this.crashMessage) {
      this.status.text = "$(error) Can server failed";
      this.status.tooltip = this.crashMessage;
      return;
    }
    const messages = [...this.projectMessages.values()].flat();
    if (messages.length) {
      this.status.text = `$(warning) Can ${this.build || "server"}`;
      this.status.tooltip = `${this.command || "Can server"}\n${messages.join("\n")}`;
    } else if (this.client && this.build) {
      this.status.text = `$(check) Can ${this.build}`;
      this.status.tooltip = `${this.command}\n${this.build}`;
    }
  }

  activate() {
    const { workspace, commands } = this.vscode;
    const watcher = workspace.createFileSystemWatcher("**/*");
    this.context.subscriptions.push(watcher);
    this.watcher = watcher;
    const manifests = workspace.createFileSystemWatcher("**/can.project.json");
    for (const event of ["onDidCreate", "onDidChange", "onDidDelete"]) {
      this.context.subscriptions.push(manifests[event](() => { void this.restart(); }));
    }
    this.context.subscriptions.push(manifests);
    for (const event of ["onDidChangeTextDocument", "onDidCloseTextDocument"]) {
      this.context.subscriptions.push(workspace[event](({ document, uri }) => {
        const file = document?.uri?.fsPath || uri?.fsPath;
        if (!file || path.basename(file) !== "can.project.json") return;
        clearTimeout(this.manifestRefresh);
        this.manifestRefresh = setTimeout(() => { void this.restart(); }, 350);
      }));
    }
    this.context.subscriptions.push(commands.registerCommand("canlc.restartServer", () => this.restart()));
    this.context.subscriptions.push(commands.registerCommand("canlc.showServerStatus", () => this.showStatus()));
    this.context.subscriptions.push(workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration("canlc.serverPath")) {
        void this.restart();
      } else if (event.affectsConfiguration("canlc")) {
        void this.client?.sendNotification("workspace/didChangeConfiguration", { settings: { canlc: { inlayHints: workspace.getConfiguration("canlc").get("inlayHints", true) } } }).catch((error) => {
          this.output.appendLine(`Can settings update failed: ${error.message || error}`);
        });
      }
    }));
    return this.start();
  }

  schedule(work) {
    this.queue = this.queue.catch(() => undefined).then(work);
    return this.queue;
  }

  async registrySelectors() {
    const selectors = [];
    const manifests = await this.vscode.workspace.findFiles("**/can.project.json");
    for (const uri of manifests) {
      try {
        const overlay = this.vscode.workspace.textDocuments.find((doc) => doc.uri.fsPath === uri.fsPath);
        const manifest = JSON.parse(overlay ? overlay.getText() : fs.readFileSync(uri.fsPath, "utf8"));
        const registry = manifest.error_registry;
        if (typeof registry !== "string" || !registry || path.isAbsolute(registry)) continue;
        const base = path.dirname(uri.fsPath);
        const resolved = path.resolve(base, registry);
        if (resolved !== base && !resolved.startsWith(base + path.sep)) continue;
        selectors.push({ scheme: "file", language: "json", pattern: new this.vscode.RelativePattern(base, escapeGlobLiteral(registry)) });
      } catch (error) {
        this.output.appendLine(`Could not inspect ${uri.fsPath} for its error registry: ${error.message}`);
      }
    }
    return selectors;
  }

  async stopClient(client) {
    this.intentionalStop = true;
    try {
      if (client.needsStop()) await client.stop();
      for (const subscription of this.clientSubscriptions) subscription.dispose();
      this.clientSubscriptions = [];
    } finally {
      this.intentionalStop = false;
    }
  }

  start() {
    return this.schedule(async () => {
      if (this.disposed || this.client) return;
      this.status.text = "$(sync~spin) Can";
      try {
        const command = this.resolveServer();
        const registrySelectors = await this.registrySelectors();
        const client = new this.LanguageClient("canlc", "Can language server", {
          command, args: ["lsp"], transport: this.TransportKind.stdio,
        }, {
          documentSelector: [
            { scheme: "file", language: "can" },
            { scheme: "untitled", language: "can" },
            { scheme: "file", language: "json", pattern: "**/can.project.json" },
            { scheme: "file", language: "json", pattern: "**/can.lock.json" },
            { scheme: "file", language: "json", pattern: "**/can.errors.json" },
            ...registrySelectors,
          ],
          synchronize: { fileEvents: this.watcher },
          initializationOptions: { inlayHints: this.vscode.workspace.getConfiguration("canlc").get("inlayHints", true) },
          outputChannel: this.output,
        });
        this.client = client;
        this.command = command;
        this.clientSubscriptions.push(client.onNotification("can/projectStatus", ({ rootUri, messages = [] }) => {
          if (messages.length) {
            this.projectMessages.set(rootUri, messages);
            this.output.appendLine(`Can project ${rootUri}: ${messages.join("; ")}`);
          } else {
            this.projectMessages.delete(rootUri);
          }
          this.updateStatus();
        }));
        this.clientSubscriptions.push(client.onDidChangeState(({ newState }) => {
          if (this.client !== client || this.intentionalStop) return;
          if (newState === this.State.Stopped || newState === this.State.StartFailed) {
            this.crashMessage = `Can language server stopped unexpectedly (${this.command}). Use Can: Restart Language Server.`;
            this.output.appendLine(this.crashMessage);
            this.updateStatus();
          } else if (newState === this.State.Running) {
            this.crashMessage = undefined;
            this.updateStatus();
          }
        }));
        await client.start();
        this.build = client.initializeResult?.serverInfo?.version || "unknown build";
        if (!this.crashMessage) this.updateStatus();
        this.output.appendLine(`Can server ready: ${this.build} (${command})`);
      } catch (error) {
        this.output.appendLine(`Can server failed: ${error.stack || error.message || error}`);
        this.crashMessage = String(error.message || error);
        this.updateStatus();
        const client = this.client;
        this.build = undefined;
        if (client) {
          try { await this.stopClient(client); this.client = undefined; } catch (stopError) {
            this.output.appendLine(`Can server cleanup failed; refusing replacement: ${stopError.stack || stopError}`);
            this.status.tooltip = `Can server cleanup failed: ${stopError.message || stopError}`;
          }
        }
      }
    });
  }

  restart() {
    return this.schedule(async () => {
      if (this.disposed) return false;
      const client = this.client;
      if (client) {
        try {
          await this.stopClient(client);
        } catch (error) {
          this.output.appendLine(`Can server shutdown failed; refusing replacement: ${error.stack || error}`);
          this.crashMessage = `Can server shutdown failed: ${error.message || error}`;
          this.updateStatus();
          return false;
        }
      }
      this.client = undefined;
      this.build = undefined;
      this.crashMessage = undefined;
      this.projectMessages.clear();
      return true;
    }).then((stopped) => stopped ? this.start() : undefined);
  }

  showStatus() {
    const message = this.client && this.build && !this.status.text.includes("failed")
      ? `Can server ${this.build} at ${this.command}${[...this.projectMessages.values()].flat().length ? `; project issues: ${[...this.projectMessages.values()].flat().join("; ")}` : ""}`
      : `Can server unavailable: ${this.status.tooltip || "not started"}`;
    return this.vscode.window.showInformationMessage(message, "Show Output").then((choice) => {
      if (choice === "Show Output") this.output.show();
    });
  }

  deactivate() {
    this.disposed = true;
    clearTimeout(this.manifestRefresh);
    return this.schedule(async () => {
      const client = this.client;
      if (client) await this.stopClient(client);
      this.client = undefined;
      this.build = undefined;
    });
  }
}

module.exports = { CanClientController, escapeGlobLiteral };
