const fs = require("node:fs");
const path = require("node:path");
const crypto = require("node:crypto");
const cp = require("node:child_process");

const root = path.resolve(__dirname, "..", "..", "..");
const extension = path.join(root, "editors", "vscode");

function walk(directory, predicate, out) {
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    if (entry.name === "node_modules" || entry.name === "bin" || entry.name === "dist") continue;
    const absolute = path.join(directory, entry.name);
    if (entry.isDirectory()) walk(absolute, predicate, out);
    else if (entry.isFile() && predicate(absolute)) out.push(absolute);
  }
}

function sourceFiles() {
  const files = [path.join(root, "go.mod"), path.join(root, "go.sum")];
  // Hash every compiler input, including files reached by go:embed and
  // generated catalogue sources. This deliberately invalidates on any
  // compiler-tree change rather than maintaining an incomplete allowlist.
  walk(path.join(root, "compiler"), () => true, files);
  // The compiler also imports the local distribution package, whose Go and
  // embedded target/asset files affect the host binary and its contracts.
  walk(path.join(root, "distribution"), () => true, files);
  for (const directory of ["client", "syntaxes", "scripts"]) {
    walk(path.join(extension, directory), () => true, files);
  }
  for (const name of ["language-configuration.json", "package.json", "bun.lock", ".vscodeignore"]) {
    files.push(path.join(extension, name));
  }
  files.sort();
  return files;
}

function sourceIdentity() {
  const files = sourceFiles();
  const hash = crypto.createHash("sha256");
  for (const file of files) {
    hash.update(path.relative(root, file).replaceAll(path.sep, "/"));
    hash.update("\0");
    hash.update(fs.readFileSync(file));
    hash.update("\0");
  }
  const digest = hash.digest("hex");
  const revision = cp.execFileSync("git", ["rev-parse", "--short=12", "HEAD"], { cwd: root, encoding: "utf8" }).trim();
  const packageVersion = require(path.join(extension, "package.json")).version;
  return { digest, revision, version: `${packageVersion}+${revision}.${digest.slice(0, 12)}`, files: files.length };
}

module.exports = { root, extension, sourceFiles, sourceIdentity };
