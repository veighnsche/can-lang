const test = require("node:test");
const assert = require("node:assert/strict");
const { CanClientController } = require("../client/controller");
const { minimatch } = require("minimatch");

function harness(options = {}) {
  const instances = [];
  const commands = new Map();
  const messages = [];
  const lines = [];
  let configListener;
  let serverPath = options.serverPath ?? process.execPath;
  let inlayHints = true;
  class FakeClient {
    constructor(id, name, server, clientOptions) {
      this.server = server;
      this.options = clientOptions;
      this.initializeResult = { serverInfo: { name: "canlc", version: "0.2.0+test" } };
      this.stops = 0;
      this.notifications = [];
      this.handlers = new Map();
      this.stateHandlers = new Set();
      instances.push(this);
    }
    async start() { if (options.startError) throw options.startError; }
    async stop() { this.stops++; if (options.stopError) throw options.stopError; }
    async sendNotification(method, params) { if (options.notificationError) throw options.notificationError; this.notifications.push({ method, params }); }
    onNotification(method, callback) { this.handlers.set(method, callback); return { dispose() { } }; }
    onDidChangeState(callback) { this.stateHandlers.add(callback); return { dispose: () => this.stateHandlers.delete(callback) }; }
    emitState(newState) { for (const callback of this.stateHandlers) callback({ newState }); }
  }
  const status = { show() {}, dispose() {} };
  const output = { appendLine(s) { lines.push(s); }, show() {}, dispose() {} };
  const vscode = {
    RelativePattern: class { constructor(base, pattern) { this.base = base; this.pattern = pattern; } },
    StatusBarAlignment: { Left: 1 },
    window: {
      createStatusBarItem: () => status,
      createOutputChannel: () => output,
      showInformationMessage: async (message) => { messages.push(message); },
    },
    commands: { registerCommand: (name, callback) => { commands.set(name, callback); return { dispose() {} }; } },
    workspace: {
      createFileSystemWatcher: (pattern) => ({
        pattern, dispose() {},
        onDidCreate: () => ({ dispose() {} }),
        onDidChange: () => ({ dispose() {} }),
        onDidDelete: () => ({ dispose() {} }),
      }),
      findFiles: async () => options.manifests || [],
      textDocuments: options.textDocuments || [],
      onDidChangeTextDocument: () => ({ dispose() {} }),
      onDidCloseTextDocument: () => ({ dispose() {} }),
      getConfiguration: () => ({ get: (key, fallback) => key === "serverPath" ? serverPath : key === "inlayHints" ? inlayHints : fallback }),
      onDidChangeConfiguration: (listener) => { configListener = listener; return { dispose() {} }; },
    },
  };
  const context = { extensionPath: "/no/bundled/server", subscriptions: [] };
  const controller = new CanClientController(vscode, FakeClient, { stdio: 1 }, context, { Stopped: 1, Running: 2, StartFailed: 4 });
  return {
    controller, instances, commands, messages, lines, status,
    setServerPath(value) { serverPath = value; },
    setInlayHints(value) { inlayHints = value; },
    configChanged(key) { configListener({ affectsConfiguration: (candidate) => candidate === key || key === "canlc" && candidate === "canlc" }); },
  };
}

test("activation selects Can, scratch and named configuration, watches inputs, shows build", async () => {
  const h = harness();
  await h.controller.activate();
  assert.equal(h.instances.length, 1);
  const c = h.instances[0];
  assert.equal(c.server.command, process.execPath);
  assert.equal(c.options.synchronize.fileEvents.pattern, "**/*");
  assert.deepEqual(c.options.documentSelector, [
    { scheme: "file", language: "can" },
    { scheme: "untitled", language: "can" },
    { scheme: "file", language: "json", pattern: "**/can.project.json" },
    { scheme: "file", language: "json", pattern: "**/can.lock.json" },
    { scheme: "file", language: "json", pattern: "**/can.errors.json" },
  ]);
  assert.deepEqual(c.options.initializationOptions, { inlayHints: true });
  assert.match(h.status.text, /0\.2\.0\+test/);
  await h.commands.get("canlc.showServerStatus")();
  assert.match(h.messages[0], /0\.2\.0\+test/);
  await h.controller.deactivate();
  assert.equal(c.stops, 1);
});

test("serverPath change awaits stop before replacement and deactivation stops last client", async () => {
  const h = harness();
  await h.controller.activate();
  h.setServerPath(process.execPath);
  h.configChanged("canlc.serverPath");
  await h.controller.queue;
  // restart schedules its start after its stop; flush the chained start.
  await new Promise((resolve) => setImmediate(resolve));
  await h.controller.queue;
  assert.equal(h.instances.length, 2);
  assert.equal(h.instances[0].stops, 1);
  await h.controller.deactivate();
  assert.equal(h.instances[1].stops, 1);
});

test("settings notification carries the current inlay preference", async () => {
  const h = harness();
  await h.controller.activate();
  h.setInlayHints(false);
  h.configChanged("canlc");
  await new Promise((resolve) => setImmediate(resolve));
  assert.deepEqual(h.instances[0].notifications, [{
    method: "workspace/didChangeConfiguration",
    params: { settings: { canlc: { inlayHints: false } } },
  }]);
  await h.controller.deactivate();
});

test("invalid configured path and launch failures report actionable status", async () => {
  const invalid = harness({ serverPath: "/missing/canlc" });
  await invalid.controller.activate();
  assert.equal(invalid.instances.length, 0);
  assert.match(invalid.status.tooltip, /missing or not executable/);
  await invalid.controller.deactivate();
  const failing = harness({ startError: new Error("protocol rejected") });
  await failing.controller.activate();
  assert.equal(failing.instances[0].stops, 1);
  assert.match(failing.lines.join("\n"), /protocol rejected/);
  assert.match(failing.status.text, /failed/);
  await failing.controller.deactivate();
});

test("project status is visible and clears when the server resolves it", async () => {
  const h = harness();
  await h.controller.activate();
  const notify = h.instances[0].handlers.get("can/projectStatus");
  assert.equal(typeof notify, "function");
  notify({ rootUri: "file:///project", messages: ["registry missing"] });
  assert.match(h.status.tooltip, /registry missing/);
  assert.match(h.lines.join("\n"), /registry missing/);
  notify({ rootUri: "file:///project", messages: [] });
  assert.doesNotMatch(h.status.tooltip, /registry missing/);
  await h.controller.deactivate();
});

test("custom registry selector is narrowed to the declaring manifest", async () => {
  const manifest = require("node:path").join(__dirname, "fixtures", "can.project.json");
  const h = harness({ manifests: [{ fsPath: manifest }] });
  await h.controller.activate();
  const selector = h.instances[0].options.documentSelector.at(-1);
  assert.equal(selector.language, "json");
  assert.equal(selector.pattern.base, require("node:path").dirname(manifest));
  assert.equal(selector.pattern.pattern, "diagnostics/custom-errors.json");
  await h.controller.deactivate();
});

test("failed stop refuses replacement and retains the old client", async () => {
  const h = harness({ stopError: new Error("shutdown refused") });
  await h.controller.activate();
  await h.commands.get("canlc.restartServer")();
  assert.equal(h.instances.length, 1);
  assert.equal(h.controller.client, h.instances[0]);
  assert.match(h.status.tooltip, /shutdown refused/);
  assert.match(h.lines.join("\n"), /refusing replacement/);
});

test("unsaved manifest chooses its overlay registry path", async () => {
  const manifest = require("node:path").join(__dirname, "fixtures", "can.project.json");
  const h = harness({
    manifests: [{ fsPath: manifest }],
    textDocuments: [{ uri: { fsPath: manifest }, getText: () => '{"error_registry":"diagnostics/unsaved.json"}' }],
  });
  await h.controller.activate();
  assert.equal(h.instances[0].options.documentSelector.at(-1).pattern.pattern, "diagnostics/unsaved.json");
  await h.controller.deactivate();
});

test("unexpected server stop updates visible status and restart recovers", async () => {
  const h = harness();
  await h.controller.activate();
  h.instances[0].emitState(1);
  assert.match(h.status.text, /failed/);
  assert.match(h.status.tooltip, /stopped unexpectedly/);
  await h.commands.get("canlc.restartServer")();
  assert.equal(h.instances.length, 2);
  assert.equal(h.instances[0].stops, 1);
  assert.match(h.status.text, /0\.2\.0\+test/);
  await h.controller.deactivate();
});

test("registry filenames containing glob syntax stay literal", async () => {
  const manifest = require("node:path").join(__dirname, "fixtures", "can.project.json");
  const h = harness({
    manifests: [{ fsPath: manifest }],
    textDocuments: [{ uri: { fsPath: manifest }, getText: () => '{"error_registry":"diagnostics/errors[1].json"}' }],
  });
  await h.controller.activate();
  const pattern = h.instances[0].options.documentSelector.at(-1).pattern.pattern;
  assert.equal(minimatch("diagnostics/errors[1].json", pattern), true);
  assert.equal(minimatch("diagnostics/errors1.json", pattern), false);
  await h.controller.deactivate();
});


test("settings delivery failure is handled during a server transition", async () => {
  const h = harness({ notificationError: new Error("connection closed") });
  await h.controller.activate();
  h.configChanged("canlc");
  await new Promise((resolve) => setImmediate(resolve));
  assert.match(h.lines.join("\n"), /Can settings update failed: connection closed/);
  await h.controller.deactivate();
});
