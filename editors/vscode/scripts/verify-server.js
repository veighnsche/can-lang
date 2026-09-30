const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const cp = require("node:child_process");
const { extension, sourceIdentity } = require("./source-identity");

function fail(message) { throw new Error(`server package rejected: ${message}`); }
function verifyServer(options = {}) {
  const expected = sourceIdentity();
  const binary = options.binary || path.join(extension, "bin", "canlc");
  const metadataPath = options.metadataPath || path.join(extension, "bin", "server-provenance.json");
  if (!fs.existsSync(binary) || !fs.existsSync(metadataPath)) fail("binary or provenance missing; run build:server after review");
  const stamp = JSON.parse(fs.readFileSync(metadataPath, "utf8"));
  if (stamp.digest !== expected.digest || stamp.revision !== expected.revision || stamp.version !== expected.version) fail("bundled binary belongs to different source contents");
  if (stamp.platform !== process.platform || stamp.arch !== process.arch) fail("bundled binary is for a different host");
  const actualSHA = crypto.createHash("sha256").update(fs.readFileSync(binary)).digest("hex");
  if (stamp.binarySHA256 !== actualSHA) fail("bundled binary bytes changed after build");
  const result = cp.spawnSync(binary, ["--version"], { encoding: "utf8", timeout: 5000 });
  if (result.error || result.status !== 0 || result.stdout.trim() !== `canlc ${expected.version}`) fail(`binary reports ${JSON.stringify(result.stdout?.trim())}`);
  if (process.platform === "darwin") {
    const archs = cp.execFileSync("lipo", ["-archs", binary], { encoding: "utf8", timeout: 5000 }).trim().split(/\s+/);
    if (!archs.includes(process.arch)) fail(`Mach-O architecture ${archs.join(",")} does not include ${process.arch}`);
    const signed = cp.spawnSync("codesign", ["--verify", "--verbose=2", binary], { encoding: "utf8", timeout: 5000 });
    if (signed.error || signed.status !== 0) fail(`macOS signature invalid: ${signed.stderr}`);
  }
  return stamp;
}

if (require.main === module) {
  try { const stamp = verifyServer(); console.log(`Verified ${stamp.version} (${stamp.platform}/${stamp.arch})`); }
  catch (error) { console.error(error.message); process.exitCode = 1; }
}
module.exports = { verifyServer };
