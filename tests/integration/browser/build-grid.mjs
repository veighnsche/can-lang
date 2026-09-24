// Bundles one `canlc build --target browser` grid output for real browsers.
// Run with the toolchain bun:
//   bun build-grid.mjs <build-dir>/browser.ts <outdir>
// Prints the bundle path. The bundle holds the exact emitted Can modules;
// only four engine-only node: modules no browser implements are shimmed,
// following the T21 build-vectors.mjs precedent:
//
//   node:async_hooks  synchronous stack-disciplined AsyncLocalStorage. The
//                     grid never enters an owner execution context (no
//                     coordination, no owned resources on the browser path),
//                     so the shim is never consulted for a live context.
//   node:util         faithful same-realm isNativeError plus an isProxy that
//                     reports false (the grid creates no proxies).
//   node:fs           readFileSync answers only diagnostics/source-index.json,
//                     with the build's real source spans baked in and zero
//                     module maps: bundled stack frames can never match
//                     emitted paths, so native-frame mapping is unavailable by
//                     construction while origin frames stay exact. Every other
//                     path throws ENOENT, so environment/credential reads fail
//                     closed instead of returning fixture data.
//   node:crypto       createHash("sha256").update(string).digest("hex") from
//                     the sibling sha256-shim.mjs, machine-checked against an
//                     independent implementation by the gate test.
//
// The boot entry below is the entire harness-authored client code: three
// lines that read the invoice id from the query string and invoke the
// emitted $canBrowserMain. It is generated verbatim (only the absolute build
// path varies) and asserted byte-for-byte by the gate test.
import { dirname, join } from "node:path";
import { readFileSync, writeFileSync } from "node:fs";

const browserTs = process.argv[2];
const outdir = process.argv[3];
if (!browserTs || !outdir) {
  console.error("usage: bun build-grid.mjs <browser.ts> <outdir>");
  process.exit(2);
}
const here = dirname(new URL(import.meta.url).pathname);
const buildDir = dirname(browserTs);
const index = JSON.parse(readFileSync(join(buildDir, "diagnostics/source-index.json"), "utf8"));
if (index.schemaVersion !== 1 || index.kind !== "can.source-index") {
  console.error("build-grid.mjs: invalid diagnostic index");
  process.exit(1);
}
const browserIndex = JSON.stringify({
  schemaVersion: index.schemaVersion,
  kind: index.kind,
  sources: index.sources,
  modules: [],
});

const bootTemplate = (path) =>
  `import { $canBrowserMain } from ${JSON.stringify(path)};
const invoice = new URLSearchParams(location.search).get("invoice") ?? "inv-1";
globalThis.__canGridBoot = $canBrowserMain([invoice]);
`;
const bootTs = join(outdir, "grid-boot.ts");
writeFileSync(bootTs, bootTemplate(browserTs));

const asyncHooksShim = `
export class AsyncLocalStorage {
  #store = undefined;
  getStore() { return this.#store; }
  run(store, fn) {
    const prev = this.#store;
    this.#store = store;
    try { return fn(); } finally { this.#store = prev; }
  }
  enterWith(store) { this.#store = store; }
  disable() { this.#store = undefined; }
}
`;
const utilShim = `
function isNativeError(value) {
  if (value === null || (typeof value !== "object" && typeof value !== "function")) return false;
  let proto = value;
  while (proto !== null) {
    if (proto === Error.prototype) return true;
    proto = Object.getPrototypeOf(proto);
  }
  return false;
}
export const types = {
  isNativeError,
  isProxy: () => false,
};
`;
const fsShim = `
const indexText = ${JSON.stringify(browserIndex)};
export function readFileSync(path, encoding) {
  const href = String(path?.href ?? path);
  if (href.endsWith("diagnostics/source-index.json")) {
    if (encoding !== undefined && encoding !== "utf8" && encoding !== "utf-8") throw new TypeError("unsupported encoding");
    return indexText;
  }
  const error = new Error("ENOENT: no such file in the browser bundle, open '" + href + "'");
  error.code = "ENOENT";
  throw error;
}
export function closeSync() { throw new Error("closeSync is unavailable in browsers"); }
`;
const cryptoShim = readFileSync(join(here, "sha256-shim.mjs"), "utf8");

const plugin = {
  name: "t25-engine-shims",
  setup(build) {
    const shim = (mod, ns, contents) => {
      build.onResolve({ filter: new RegExp(`^node:${mod}$`) }, (args) => ({
        path: args.path,
        namespace: ns,
      }));
      build.onLoad({ filter: /.*/, namespace: ns }, () => ({ contents, loader: "js" }));
    };
    shim("async_hooks", "t25-als", asyncHooksShim);
    shim("util", "t25-util", utilShim);
    shim("fs", "t25-fs", fsShim);
    shim("crypto", "t25-crypto", cryptoShim);
  },
};
// Shim adequacy is proven by the matrix itself: Bun stubs unshimmed node:
// imports, so a new engine-only import the set does not cover breaks the
// grid observably in a real browser instead of hiding.
const result = await Bun.build({
  entrypoints: [bootTs],
  outdir,
  target: "browser",
  plugins: [plugin],
});
if (!result.success) {
  for (const log of result.logs) console.error(log);
  process.exit(1);
}
console.log(join(outdir, "grid-boot.js"));
