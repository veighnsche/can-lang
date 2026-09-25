// Bundles the shared wire-codec vectors for browser loading with the
// compiler's own sealed browser overlay and nothing else. Usage:
//   bun build-vectors.mjs <entry.ts> <outdir> <runtime-dir>
// Prints the bundle path. Every module the vector graph reaches
// resolves to a committed runtime source: the fourteen overlay edges
// below mirror compiler/internal/browser/audit.go browserOverlay
// exactly (the same substitution the verified browser pipeline
// stages), and any other node: import fails the build loudly. No
// test-authored shim or semantic alias remains: the browsers execute
// the shipped browser-profile modules.
import { dirname, join, resolve } from "node:path";

const entry = process.argv[2];
const outdir = process.argv[3];
const runtime = resolve(process.argv[4] ?? "");
if (!entry || !outdir || !process.argv[4]) {
  console.error("usage: bun build-vectors.mjs <entry.ts> <outdir> <runtime-dir>");
  process.exit(2);
}
// Mirror of browserOverlay in compiler/internal/browser/audit.go. The
// Go parity test asserts every edge still matches the committed
// table before trusting a bundle built here.
const overlay = {
  "reflect.ts": "browser/reflect.ts",
  "domain.ts": "browser/domain.ts",
  "diagnostics.ts": "browser/diagnostics.ts",
  "owner.ts": "browser/owner.ts",
  "callable.ts": "browser/callable.ts",
  "coordination.ts": "browser/coordination.ts",
  "entry.ts": "browser/entry.ts",
  "codec/formats.ts": "browser/formats.ts",
  "platform/cookies.ts": "browser/cookies.ts",
  "platform/csrf.ts": "browser/csrf.ts",
  "platform/clock.ts": "browser/clock.ts",
  "platform/log.ts": "browser/log.ts",
  "platform/markdown.ts": "browser/markdown.ts",
  "platform/assets.ts": "browser/assets.ts",
};
const canonical = new Map(
  Object.entries(overlay).map(([from, to]) => [join(runtime, from), join(runtime, to)]),
);
const plugin = {
  name: "can-browser-overlay",
  setup(build) {
    build.onResolve({ filter: /.*/ }, (args) => {
      if (args.path.startsWith("node:")) {
        throw new Error(`node builtin reached the vector graph: ${args.path} from ${args.importer}`);
      }
      if (args.path.startsWith(".") && args.importer) {
        const resolved = join(dirname(args.importer), args.path);
        const alternate = canonical.get(resolved) ?? canonical.get(`${resolved}.ts`);
        if (alternate !== undefined) return { path: alternate };
      }
      return undefined;
    });
  },
};
const result = await Bun.build({ entrypoints: [entry], outdir, target: "browser", plugins: [plugin] });
if (!result.success) {
  for (const log of result.logs) console.error(log);
  process.exit(1);
}
const [{ path }] = result.outputs;
console.log(path);
