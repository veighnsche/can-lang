const test = require("node:test");
const assert = require("node:assert/strict");
const path = require("node:path");
const { root, sourceFiles, sourceIdentity } = require("../scripts/source-identity");

test("source provenance covers actual embedded catalogue inputs", () => {
  const files = new Set(sourceFiles());
  for (const name of ["catalogue.json", "runtime-header.txt", "runtime-footer.txt"]) {
    assert.ok(files.has(path.join(root, "compiler/internal/catalogue", name)), `${name} absent from source fingerprint`);
  }
  for (const name of ["manifest.go", "target.json", "target-linux-amd64.json", "assets/htmx-4.0.0.min.js", "assets/htmx-guard.js"]) {
    assert.ok(files.has(path.join(root, "distribution", name)), `${name} absent from local dependency fingerprint`);
  }
  for (const name of ["bun.lock", ".vscodeignore"]) {
    assert.ok(files.has(path.join(root, "editors/vscode", name)), `${name} absent from package fingerprint`);
  }
  assert.match(sourceIdentity().digest, /^[a-f0-9]{64}$/);
});

test("packaging rejects an old binary before launch", (t) => {
  const fs = require("node:fs");
  const os = require("node:os");
  const { verifyServer } = require("../scripts/verify-server");
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "can-vsix-test-"));
  t.after(() => fs.rmSync(dir, { recursive: true, force: true }));
  const binary = path.join(dir, "canlc");
  const metadataPath = path.join(dir, "server-provenance.json");
  fs.writeFileSync(binary, "old binary");
  fs.writeFileSync(metadataPath, JSON.stringify({ digest: "stale", version: "old" }));
  assert.throws(() => verifyServer({ binary, metadataPath }), /different source contents/);
});

test("VSIX asset gate rejects changed grammar, client and language configuration bytes", () => {
  const { extension } = require("../scripts/source-identity");
  const { verifySourceAssets, sourceAssets } = require("../scripts/verify-vsix");
  const baseline = new Map(sourceAssets.map((asset) => [`extension/${asset}`, require("node:fs").readFileSync(path.join(extension, asset))]));
  // Match VSCE's real archive spelling rather than the checkout's spelling.
  baseline.set("extension/readme.md", baseline.get("extension/README.md"));
  baseline.delete("extension/README.md");
  const read = (name) => baseline.get(name);
  assert.doesNotThrow(() => verifySourceAssets(read));
  for (const asset of ["syntaxes/can.tmGrammar.json", "client/controller.js", "language-configuration.json"]) {
    const name = `extension/${asset}`;
    const original = baseline.get(name);
    baseline.set(name, Buffer.concat([original, Buffer.from("\nchanged")]));
    assert.throws(() => verifySourceAssets(read), /differs from reviewed source/);
    baseline.set(name, original);
  }
});
