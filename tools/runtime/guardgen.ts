// Guard generator: transpiles the owned HTMX guard to its servable bytes.
//
// The checked-in distribution/assets/htmx-guard.js must stay byte-identical
// to `new Bun.Transpiler({ loader: "ts" }).transformSync` over
// runtime/platform/htmx-guard.ts — the same settings the pin test in
// runtime/test/html.test.ts ("the runtime head loads the pinned guard
// module after htmx") uses to derive the served integrity. The generated
// file carries no header or trailer: every extra byte would invalidate
// the pins below.
//
// Regenerate with `bun tools/runtime/guardgen.ts` (or `make guard`);
// verify with the same command plus `--check` (or `make guard-check`).
// The freshness test in runtime/test/guard-freshness.test.ts also rejects
// stale bytes. After regenerating, re-pin together:
//
// - runtime/platform/html.ts runtimeHead guard integrity
// - runtime/platform/assets.ts guardIntegrity
// - compiler/internal/emit/assets.go guard Integrity literal
// - distribution/manifest.go GuardAsset digests
// - tests/integration/browser/assets.mjs pinned-scripts guard integrity
// - distribution/guard_test.go digest literals
import { join } from "node:path";

const usage = "usage: bun tools/runtime/guardgen.ts [--check] [root]";

const args = Bun.argv.slice(2);
const check = args.includes("--check");
const positional = args.filter((arg) => arg !== "--check");
if (positional.length > 1) {
  console.error(usage);
  process.exit(2);
}
const root = positional[0] ?? ".";
const sourcePath = join(root, "runtime/platform/htmx-guard.ts");
const outputPath = join(root, "distribution/assets/htmx-guard.js");

const source = await Bun.file(sourcePath).text();
const served = new Bun.Transpiler({ loader: "ts" }).transformSync(source);

if (check) {
  const actual = await Bun.file(outputPath)
    .text()
    .catch(() => undefined);
  if (actual !== served) {
    console.error(
      `stale guard mirror distribution/assets/htmx-guard.js; run bun tools/runtime/guardgen.ts`,
    );
    process.exit(1);
  }
} else {
  await Bun.write(outputPath, served);
}
