const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const cp = require("node:child_process");
const { root, extension, sourceIdentity } = require("./source-identity");

const identity = sourceIdentity();
const binDir = path.join(extension, "bin");
fs.mkdirSync(binDir, { recursive: true });
const temporary = path.join(binDir, `.canlc-build-${process.pid}`);
const metadata = path.join(binDir, "server-provenance.json");
const metadataTemp = path.join(binDir, `.server-provenance-${process.pid}.json`);
fs.rmSync(metadata, { force: true });
try {
  const built = cp.spawnSync("go", ["build", "-p", "1", "-ldflags", `-X main.version=${identity.version}`, "-o", temporary, "./compiler"], { cwd: root, stdio: "inherit", timeout: 10 * 60 * 1000 });
  if (built.error) throw built.error;
  if (built.status !== 0) throw new Error(`go build failed with status ${built.status}`);
  if (process.platform === "darwin") {
    const signed = cp.spawnSync("codesign", ["--force", "--sign", "-", temporary], { stdio: "inherit", timeout: 30000 });
    if (signed.error || signed.status !== 0) throw signed.error || new Error(`codesign failed with status ${signed.status}`);
  }
  const binary = path.join(binDir, "canlc");
  fs.renameSync(temporary, binary);
  const binarySHA256 = crypto.createHash("sha256").update(fs.readFileSync(binary)).digest("hex");
  fs.writeFileSync(metadataTemp, JSON.stringify({ ...identity, binarySHA256, platform: process.platform, arch: process.arch }, null, 2) + "\n");
  fs.renameSync(metadataTemp, metadata);
  process.stdout.write(`Built ${identity.version} (${identity.files} source files)\n`);
} finally {
  fs.rmSync(temporary, { force: true });
  fs.rmSync(metadataTemp, { force: true });
}
