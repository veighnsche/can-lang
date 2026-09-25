// Verified browser bundling (UP15). Fixed offline tool: it runs the native
// `Bun.build({ target: "browser" })` over exactly the driver-staged,
// pre-audited generation tree and reports the emitted bytes. No plugins, no
// aliases, no shims, no externals, no test imports: the tree on disk is the
// only input, and any bundler failure exits nonzero with the native logs on
// stderr. Byte shaping (map normalization, trailers, diagnostic table,
// manifest) and all audits stay in the Go driver; this tool only invokes the
// pinned bundler and digests what it emitted.
import "../../runtime/environment.ts";
import { createHash } from "node:crypto";
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, resolve, sep } from "node:path";

const raw = await Bun.stdin.text();
const input = JSON.parse(raw);
if (
  input === null ||
  typeof input !== "object" ||
  input.schemaVersion !== 1 ||
  input.kind !== "can.browser-bundle-request" ||
  typeof input.sourceDir !== "string" ||
  typeof input.entry !== "string" ||
  typeof input.outDir !== "string" ||
  !Array.isArray(input.files) ||
  input.expected === null ||
  typeof input.expected !== "object" ||
  typeof input.expected.version !== "string" ||
  typeof input.expected.revision !== "string"
)
  throw new Error("invalid browser bundle request");

if (Bun.version !== input.expected.version || Bun.revision !== input.expected.revision)
  throw new Error(
    `bundler toolchain mismatch: have bun ${Bun.version} ${Bun.revision}`,
  );

const sourceDir = resolve(input.sourceDir);
const outDir = resolve(input.outDir);
if (sourceDir === outDir || outDir.startsWith(sourceDir + sep))
  throw new Error("bundle output must not overlap the staged tree");

// The staged tree must be exactly the audited inventory: no missing, extra,
// or substituted bytes between the pre-bundle audit and the bundler.
const want = new Map<string, string>();
for (const file of input.files) {
  if (
    file === null ||
    typeof file !== "object" ||
    typeof file.path !== "string" ||
    typeof file.sha256 !== "string" ||
    !/^[0-9a-f]{64}$/.test(file.sha256)
  )
    throw new Error("invalid bundle inventory entry");
  const resolved = resolve(sourceDir, file.path);
  if (resolved !== sourceDir && !resolved.startsWith(sourceDir + sep))
    throw new Error("bundle inventory escapes the staged tree");
  want.set(file.path, file.sha256);
}
const seen: string[] = [];
const walk = (dir: string): void => {
  for (const entry of readdirSync(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(full);
      continue;
    }
    if (!entry.isFile()) throw new Error("staged tree holds a non-regular file");
    seen.push(relative(sourceDir, full).split(sep).join("/"));
  }
};
walk(sourceDir);
if (seen.length !== want.size || seen.some((path) => !want.has(path)))
  throw new Error("staged tree does not match the audited inventory");
for (const [path, sha256] of want) {
  const bytes = readFileSync(join(sourceDir, path));
  if (createHash("sha256").update(bytes).digest("hex") !== sha256)
    throw new Error(`staged file changed after audit: ${path}`);
}

const entry = resolve(sourceDir, input.entry);
if (!want.has(input.entry)) throw new Error("bundle entry is not staged");
if (statSync(outDir, { throwIfNoEntry: false })?.isDirectory() !== true)
  throw new Error("bundle output directory is missing");

const result = await Bun.build({
  entrypoints: [entry],
  outdir: outDir,
  target: "browser",
  sourcemap: "external",
  minify: false,
});
if (!result.success) {
  for (const log of result.logs) console.error(log);
  throw new Error("browser bundle failed");
}

const files: { path: string; sha256: string; bytes: number }[] = [];
const collect = (dir: string): void => {
  for (const entry of readdirSync(dir, { withFileTypes: true }).toSorted((a, b) =>
    a.name < b.name ? -1 : 1,
  )) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) {
      collect(full);
      continue;
    }
    const bytes = readFileSync(full);
    files.push({
      path: relative(outDir, full).split(sep).join("/"),
      sha256: createHash("sha256").update(bytes).digest("hex"),
      bytes: bytes.length,
    });
  }
};
collect(outDir);
console.log(
  JSON.stringify({
    schemaVersion: 1,
    kind: "can.browser-bundle-report",
    requestSHA256: createHash("sha256").update(raw).digest("hex"),
    bun: { version: Bun.version, revision: Bun.revision },
    files,
  }),
);
