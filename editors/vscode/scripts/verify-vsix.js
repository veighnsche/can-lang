const fs = require("node:fs");
const cp = require("node:child_process");
const path = require("node:path");
const crypto = require("node:crypto");
const { extension, sourceIdentity } = require("./source-identity");
const { verifyServer } = require("./verify-server");

const sourceAssets = [
  "package.json", "client/extension.js", "client/controller.js",
  "syntaxes/can.tmGrammar.json", "language-configuration.json", "README.md",
];

function verifySourceAssets(readPacked) {
  for (const asset of sourceAssets) {
    const packed = readPacked(`extension/${asset}`);
    const reviewed = fs.readFileSync(path.join(extension, asset));
    if (!packed.equals(reviewed)) throw new Error(`VSIX ${asset} differs from reviewed source`);
  }
}

function verifyVSIX(vsix) {
if (!fs.existsSync(vsix)) throw new Error("VSIX archive missing");
const stamp = verifyServer();
const entries = cp.execFileSync("unzip", ["-Z1", vsix], { encoding: "utf8" }).trim().split("\n");
for (const required of ["extension/package.json", "extension/bin/canlc", "extension/bin/server-provenance.json", "extension/client/extension.js", "extension/client/controller.js", "extension/node_modules/vscode-languageclient/lib/node/main.js", "extension/node_modules/vscode-languageserver-protocol/lib/node/main.js", "extension/syntaxes/can.tmGrammar.json", "extension/language-configuration.json"]) {
  if (!entries.includes(required)) throw new Error(`VSIX missing ${required}`);
}
if (entries.some((entry) => /(?:test|scripts|vscode-textmate|vscode-oniguruma|@vscode\/vsce)\//.test(entry))) throw new Error("VSIX contains development-only payload");
const payload = (entry) => cp.execFileSync("unzip", ["-p", vsix, entry], { maxBuffer: 128 * 1024 * 1024 });
verifySourceAssets(payload);
const packageJSON = JSON.parse(payload("extension/package.json"));
const packedStamp = JSON.parse(payload("extension/bin/server-provenance.json"));
if (packageJSON.version !== require(path.join(extension, "package.json")).version || packedStamp.digest !== sourceIdentity().digest || packedStamp.version !== stamp.version) throw new Error("VSIX identity differs from reviewed source");
const binarySHA = crypto.createHash("sha256").update(payload("extension/bin/canlc")).digest("hex");
if (binarySHA !== stamp.binarySHA256) throw new Error("VSIX bundled binary differs from verified host build");
console.log(`Verified VSIX ${packageJSON.version} with ${stamp.version}`);
}

if (require.main === module) verifyVSIX(path.resolve(process.argv[2] || ""));
module.exports = { verifySourceAssets, verifyVSIX, sourceAssets };
