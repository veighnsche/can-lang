// Builds the browser-target wire-vector bundle with `Bun.build`. Run with bun:
//   bun build-vectors.mjs <entry.ts> <outdir>
// The exact runtime sources are bundled except two engine-only modules no
// browser implements: `node:async_hooks` gets a synchronous
// stack-disciplined AsyncLocalStorage stub, and `node:util` gets a `types`
// object with a faithful same-realm isNativeError plus an isProxy that
// reports false. Every vector is synchronous, proxy-free and never enters
// an owner execution context, and the parity test additionally runs the same
// bundle under bun, so any observable shim divergence fails the comparison
// instead of hiding.
import { basename, join } from "node:path";

const entry = process.argv[2];
const outdir = process.argv[3];
if (!entry || !outdir) {
  console.error("usage: bun build-vectors.mjs <entry.ts> <outdir>");
  process.exit(2);
}
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
const plugin = {
  name: "t21-engine-shims",
  setup(build) {
    build.onResolve({ filter: /^node:async_hooks$/ }, (args) => ({
      path: args.path,
      namespace: "t21-als",
    }));
    build.onLoad({ filter: /.*/, namespace: "t21-als" }, () => ({
      contents: asyncHooksShim,
      loader: "js",
    }));
    build.onResolve({ filter: /^node:util$/ }, (args) => ({
      path: args.path,
      namespace: "t21-util",
    }));
    build.onLoad({ filter: /.*/, namespace: "t21-util" }, () => ({
      contents: utilShim,
      loader: "js",
    }));
  },
};
const result = await Bun.build({
  entrypoints: [entry],
  outdir,
  target: "browser",
  plugins: [plugin],
});
if (!result.success) {
  for (const log of result.logs) console.error(log);
  process.exit(1);
}
console.log(join(outdir, `${basename(entry, ".ts")}.js`));
