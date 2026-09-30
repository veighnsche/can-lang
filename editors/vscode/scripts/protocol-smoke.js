const fs = require("node:fs");
const path = require("node:path");
const cp = require("node:child_process");
const assert = require("node:assert/strict");
const { extension } = require("./source-identity");
const { verifyServer } = require("./verify-server");

async function smoke() {
  const stamp = verifyServer();
  const child = cp.spawn(path.join(extension, "bin", "canlc"), ["lsp"], { stdio: ["pipe", "pipe", "pipe"] });
  let buffer = Buffer.alloc(0);
  let stderr = "";
  let done = false;
  const deadline = setTimeout(() => child.kill(), 8000);
  child.stderr.on("data", (chunk) => { stderr += chunk.toString(); });
  try {
    const result = await new Promise((resolve, reject) => {
      child.once("error", reject);
      child.once("exit", (code) => { if (!done) reject(new Error(`server exited ${code}: ${stderr}`)); });
      child.stdout.on("data", (chunk) => {
        buffer = Buffer.concat([buffer, chunk]);
        for (;;) {
          const boundary = buffer.indexOf("\r\n\r\n");
          if (boundary < 0) return;
          const header = buffer.subarray(0, boundary).toString();
          const match = /Content-Length:\s*(\d+)/i.exec(header);
          if (!match) { reject(new Error(`missing Content-Length: ${header}`)); return; }
          const length = Number(match[1]);
          if (buffer.length < boundary + 4 + length) return;
          const payload = JSON.parse(buffer.subarray(boundary + 4, boundary + 4 + length).toString());
          buffer = buffer.subarray(boundary + 4 + length);
          if (payload.id === 1) { done = true; resolve(payload); return; }
        }
      });
      const request = JSON.stringify({ jsonrpc: "2.0", id: 1, method: "initialize", params: { processId: process.pid, rootUri: null, capabilities: {}, workspaceFolders: null } });
      child.stdin.write(`Content-Length: ${Buffer.byteLength(request)}\r\n\r\n${request}`);
    });
    assert.equal(result.result?.serverInfo?.version, stamp.version);
    assert.ok(result.result?.capabilities?.textDocumentSync);
    assert.equal(typeof result.result?.capabilities?.definitionProvider, "boolean");
    console.log(`Protocol smoke: ${stamp.version}, ${Object.keys(result.result.capabilities).length} capabilities`);
  } finally {
    clearTimeout(deadline);
    child.kill();
  }
}
smoke().catch((error) => { console.error(error); process.exitCode = 1; });
