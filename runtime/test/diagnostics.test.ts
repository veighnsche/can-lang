import { test, expect } from "bun:test";
import { mkdtempSync, mkdirSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { GenMapping, addMapping, toEncodedMap } from "../../tools/runtime/vendor/source-maps.mjs";
import { diagnosticFrames } from "../diagnostics.ts";

test.each([true, false])(
  "direct Bun TS frames compose through upstream maps (callback=%p) without private details",
  async (callback) => {
    const root = mkdtempSync(join(tmpdir(), "can-maps-"));
    try {
      mkdirSync(join(root, "diagnostics"));
      mkdirSync(join(root, "runtime"));
      writeFileSync(
        join(root, "runtime/helper.ts"),
        'export function native(value: number): never { throw new Error("bearer-secret /private/machine"); }\n',
      );
      const runtime = new URL("../diagnostics.ts", import.meta.url).href;
      const lines = [
        `import {configureDiagnostics,diagnosticFrames} from ${JSON.stringify(runtime)};`,
        'import {native} from "./runtime/helper.ts";',
        callback
          ? 'async function inner(): Promise<void> { await Promise.resolve().then(async (): Promise<void> => { const astral: string = "😀"; await Promise.resolve(); native(1); }); }'
          : 'async function inner(): Promise<void> { const astral: string = "😀"; await Promise.resolve(); native(1); }',
        "async function outer(): Promise<void> { await inner(); }",
        "configureDiagnostics(import.meta.url);",
        'try { await outer(); } catch(cause) { const origin={source:"can.project.root/app/unicode.can",start:100,end:120,invocation:[]};console.log(JSON.stringify({frames:diagnosticFrames(cause,origin),synthetic:diagnosticFrames("private primitive cause",origin)})); }',
      ];
      const source = "can.project.root/app/unicode.can";
      const prefix = '    let s = "😀"; ';
      const span = {
        start: 100,
        end: 120,
        line: 4,
        column: prefix.length,
        endLine: 4,
        endColumn: prefix.length + 8,
        operation: "call",
      };
      const outer = {
        start: 130,
        end: 150,
        line: 8,
        column: 4,
        endLine: 8,
        endColumn: 15,
        operation: "call",
      };
      const segments = [
        { line: 3, column: lines[2].indexOf("native(1)"), source, name: "call:100:120" },
        { line: 4, column: lines[3].indexOf("await inner()"), source, name: "call:130:150" },
      ];
      const index = {
        schemaVersion: 1,
        kind: "can.source-index",
        sources: [
          {
            id: source,
            path: "unicode.can",
            spans: { "call:100:120": span, "call:130:150": outer },
          },
        ],
        modules: [{ path: "entry.ts", segments }],
      };
      const map = new GenMapping({ file: "entry.ts" });
      for (const p of segments) {
        const original = p.line === 3 ? span : outer;
        addMapping(map, {
          generated: { line: p.line, column: p.column },
          source,
          original: { line: original.line, column: original.column },
          name: p.name,
        });
      }
      writeFileSync(join(root, "diagnostics/source-index.json"), JSON.stringify(index));
      writeFileSync(join(root, "entry.ts.map"), JSON.stringify(toEncodedMap(map)));
      writeFileSync(
        join(root, "entry.ts"),
        lines.join("\n") + "\n//# sourceMappingURL=entry.ts.map\n",
      );
      const child = Bun.spawn(
        [process.execPath, "--no-install", "--no-env-file", join(root, "entry.ts")],
        { stdout: "pipe", stderr: "pipe", env: { PATH: "/nonexistent" } },
      );
      const [out, err, status] = await Promise.all([
        new Response(child.stdout).text(),
        new Response(child.stderr).text(),
        child.exited,
      ]);
      expect(status).toBe(0);
      expect(err).toBe("");
      const { frames, synthetic } = JSON.parse(out);
      expect(synthetic).toEqual([
        {
          ...span,
          source,
          file: "unicode.can",
          column: span.column + 1,
          endColumn: span.endColumn + 1,
          synthetic: true,
        },
      ]);
      expect(frames).toContainEqual({
        ...span,
        source,
        file: "unicode.can",
        column: span.column + 1,
        endColumn: span.endColumn + 1,
        synthetic: false,
      });
      if (!callback) expect(frames.some((f: any) => f.line === 8 && !f.synthetic)).toBe(true);
      for (const secret of [
        root,
        "bearer-secret",
        "/private/machine",
        "helper.ts",
        "native(",
        "Error:",
      ])
        expect(out).not.toContain(secret);
    } finally {
      rmSync(root, { recursive: true, force: true });
    }
  },
);
test("stack inspection rejects proxies and accessors without calling authored behavior", () => {
  let calls = 0;
  const accessor = new Error("hidden");
  Object.defineProperty(accessor, "stack", {
    get() {
      calls++;
      throw "bad";
    },
  });
  const native = new Error("hidden");
  Object.defineProperty(native, "name", {
    get() {
      calls++;
      return "secret";
    },
  });
  Object.defineProperty(native, "message", {
    get() {
      calls++;
      return "secret";
    },
  });
  const proxy = new Proxy(new Error("hidden"), {
    get() {
      calls++;
      throw "bad";
    },
    getOwnPropertyDescriptor() {
      calls++;
      throw "bad";
    },
    getPrototypeOf() {
      calls++;
      throw "bad";
    },
  });
  for (const value of [accessor, native, proxy, "bearer-secret", { stack: "secret" }])
    expect(
      diagnosticFrames(value, { source: "unknown", start: 0, end: 0, invocation: [] }),
    ).toEqual([]);
  expect(calls).toBe(0);
});
