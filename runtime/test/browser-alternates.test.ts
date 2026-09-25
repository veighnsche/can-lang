import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { decodeJson5, decodeToml, decodeYaml } from "../browser/formats.ts";
import { COOKIE_KIND, createCookies, isCookieValue } from "../browser/cookies.ts";
import { createCSRF } from "../browser/csrf.ts";
import { createClock } from "../browser/clock.ts";
import { createLog } from "../browser/log.ts";
import { createMarkdown } from "../browser/markdown.ts";
import { createAssets } from "../browser/assets.ts";
import type { Completion } from "../completion.ts";

const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const str = shape("primitive", "str"),
  int = shape("primitive", "int");
const decls = catalogue.errors.filter((e) =>
  ["clock::invalid_duration", "log::write_failed"].includes(e.name),
);
const declarations = decls.map((e) => ({ ...e, parameters: 0 }));
const errors = decls.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({ declarations, shapes: [str, int, ...errors] });
const identity = (name: string) => errors.find((e) => e.declaration === name)!.identity;

test("format stubs fail closed without a parser", () => {
  for (const [name, fn] of [
    ["TOML", decodeToml],
    ["YAML", decodeYaml],
    ["JSON5", decodeJson5],
  ] as const) {
    expect(() => fn({} as never, new Uint8Array())).toThrow(
      `${name} parsing is unavailable in the browser profile`,
    );
  }
});

test("cookie stubs fail closed and mint no handles", async () => {
  expect(COOKIE_KIND).toBe("cookie");
  expect(isCookieValue("cookie", {})).toBe(false);
  expect(isCookieValue(undefined, {})).toBe(false);
  const cookies = createCookies(domain, {
    invalid: "invalid",
    collection: "collection",
    pair: "pair",
    some: "some",
    none: "none",
    samesiteStrict: "strict",
    samesiteLax: "lax",
    samesiteNone: "none",
  });
  await expect(cookies.parse("a=b")).rejects.toThrow(
    "cookie handling is unavailable in the browser profile",
  );
  await expect(cookies.get({}, "a")).rejects.toThrow(
    "cookie handling is unavailable in the browser profile",
  );
  await expect(cookies.make("a", "b", {})).rejects.toThrow(
    "cookie handling is unavailable in the browser profile",
  );
  await expect(cookies.serialize({})).rejects.toThrow(
    "cookie handling is unavailable in the browser profile",
  );
  await expect(cookies.remove("a", "/", {})).rejects.toThrow(
    "cookie handling is unavailable in the browser profile",
  );
});

test("csrf stubs fail closed without a secret", async () => {
  const csrf = createCSRF(domain, { invalid: "invalid" });
  await expect(csrf.generate("secret", "session", 1000n)).rejects.toThrow(
    "CSRF tokens are unavailable in the browser profile",
  );
  await expect(csrf.verify("secret", "session", "token", 1000n)).rejects.toThrow(
    "CSRF tokens are unavailable in the browser profile",
  );
});

test("markdown stubs fail closed without an engine", async () => {
  const markdown = createMarkdown(domain, {
    overLimit: "over",
    htmlStructure: "structure",
    htmlURL: "url",
  });
  await expect(markdown.renderTextHTML("# hi")).rejects.toThrow(
    "Markdown rendering is unavailable in the browser profile",
  );
  await expect(markdown.renderSafe("# hi")).rejects.toThrow(
    "Markdown rendering is unavailable in the browser profile",
  );
});

test("asset stub serves no routes", async () => {
  const assets = createAssets(
    { htmx: {} as never, guard: {} as never, project: [] },
    new URL("file:///"),
  );
  await expect(assets.serve(new Request("https://example.com/__can/"))).rejects.toThrow(
    "asset serving is unavailable in the browser profile",
  );
});

test("browser clock reads and sleeps natively", async () => {
  const clock = createClock(domain, identity("can.std.clock@1::invalid_duration"));
  const wall = await clock.wallMillis();
  expect(wall.kind).toBe("ok");
  const mono = await clock.monotonicMillis();
  expect(mono.kind).toBe("ok");
  const start = Date.now();
  const slept = await clock.sleepMillis(20n);
  expect(slept.kind).toBe("ok");
  expect(Date.now() - start).toBeGreaterThanOrEqual(5);
  for (const bad of [-1n, 2147483648n]) {
    const failed = await clock.sleepMillis(bad);
    expect(failed.kind).toBe("domain");
    if (failed.kind !== "domain") throw new Error("expected domain");
    const d = domainFailureDiagnostics(failed.value);
    expect(d.declaration.name).toBe("clock::invalid_duration");
  }
});

test("browser log routes levels and keeps line bytes", async () => {
  const log = createLog(domain, identity("can.std.log@1::write_failed"));
  const lines: { level: string; line: string }[] = [];
  const host = {
    encode: JSON.stringify,
    info: (line: string) => void lines.push({ level: "info", line }),
    error: (line: string) => void lines.push({ level: "error", line }),
  };
  const injected = createLog(domain, identity("can.std.log@1::write_failed"), host);
  expect((await injected.writeInfo("hello")).kind).toBe("ok");
  expect((await injected.writeError("boom")).kind).toBe("ok");
  expect(lines).toEqual([
    { level: "info", line: '{"level":"info","message":"hello"}' },
    { level: "error", line: '{"level":"error","message":"boom"}' },
  ]);
  const failing = createLog(domain, identity("can.std.log@1::write_failed"), {
    encode: () => {
      throw new RangeError("encode");
    },
    info: () => {},
    error: () => {},
  });
  const failed: Completion = await failing.writeInfo("x");
  expect(failed.kind).toBe("domain");
  await expect(log.writeInfo(7 as never)).rejects.toThrow("invalid log text");
});

test("browser log defaults to the console by level", async () => {
  const log = createLog(domain, identity("can.std.log@1::write_failed"));
  const seen: { sink: string; line: unknown }[] = [];
  const origLog = console.log;
  const origError = console.error;
  console.log = (line: unknown) => void seen.push({ sink: "log", line });
  console.error = (line: unknown) => void seen.push({ sink: "error", line });
  try {
    await log.writeInfo("hello");
    await log.writeError("boom");
  } finally {
    console.log = origLog;
    console.error = origError;
  }
  expect(seen).toEqual([
    { sink: "log", line: '{"level":"info","message":"hello"}' },
    { sink: "error", line: '{"level":"error","message":"boom"}' },
  ]);
});
