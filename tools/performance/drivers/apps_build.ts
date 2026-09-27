import { resolve } from "node:path";
const [repo, out] = process.argv.slice(2);
const overlay = new Set([
  "domain",
  "reflect",
  "diagnostics",
  "owner",
  "callable",
  "coordination",
  "entry",
]);
const result = await Bun.build({
  entrypoints: [resolve(repo!, "tools/performance/drivers/apps_browser.ts")],
  outdir: out!,
  target: "browser",
  naming: "browser.js",
  plugins: [
    {
      name: "can-browser-profile",
      setup(build) {
        build.onResolve({ filter: /\.ts$/ }, (args) => {
          const path = resolve(args.resolveDir, args.path);
          const name = path.split("/").pop()!.replace(".ts", "");
          if (path === resolve(repo!, `runtime/${name}.ts`) && overlay.has(name))
            return { path: resolve(repo!, `runtime/browser/${name}.ts`) };
        });
      },
    },
  ],
});
if (!result.success) throw new Error(result.logs.join("\n"));
