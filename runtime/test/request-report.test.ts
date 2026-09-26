import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, failure, value, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { captureStandard, standardFailureDiagnostics } from "../failure.ts";
import {
  runOwnedRoot,
  runExplicitRoot,
  launchOwnedWithContext,
  registerResource,
  resourceStatus,
  type OwnerDiagnostic,
} from "../owner.ts";
import { runEntry } from "../entry.ts";
import { reportBrowserDiagnostic } from "../browser/diagnostics.ts";
import {
  correlationFor,
  defaultRequestReportSink,
  noteRequestSource,
  reportRequestFailure,
  requestContext,
  requestSource,
  setRequestReportSink,
  type RequestReportContext,
} from "../transport/request-report.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { createResponses } from "../platform/http.ts";
import { compileActionRoutes } from "../platform/action-routes.ts";
import {
  createJsonActions,
  type ActionJsonHandler,
  type EmittedActionEntry,
} from "../platform/action-json.ts";
import type { Schema } from "../codec/json.ts";

const origin = { source: "can:test", start: 0, end: 0, invocation: [] };
const identity = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const textShape: FailureShape = {
  identity: identity("primitive", "str"),
  kind: "primitive",
  declaration: "str",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const intShape: FailureShape = {
  identity: identity("primitive", "int"),
  kind: "primitive",
  declaration: "int",
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
};
const fieldNames: Record<string, string[]> = {
  "http::invalid_server_config": ["reason"],
  "http::bind_failed": ["address"],
  "http::shutdown_failed": ["phase"],
  "http::invalid_route": ["reason"],
  "http::duplicate_route": ["method", "path"],
  "http::ambiguous_route": ["first", "second"],
  "http::invalid_request": ["reason"],
  "http::body_limit": ["limit"],
  "codec::invalid_data": ["path", "reason"],
  "stream::read_failed": ["reason"],
  "stream::write_failed": ["reason"],
  "stream::cancelled": ["reason"],
  "stream::close_failed": ["reason"],
};
const declarations = catalogue.errors
  .filter((e) => fieldNames[e.name] !== undefined)
  .map((e) => ({ identity: e.identity, name: e.name, parameters: 0 }));
const fieldKinds = new Map(
  catalogue.errors.flatMap((e) => e.fields.map((f) => [e.name + ":" + f.name, f.type])),
);
const errorShapes: FailureShape[] = declarations.map((e) => ({
  identity: identity("error", e.identity),
  kind: "error",
  declaration: e.identity,
  arguments: [],
  fields: fieldNames[e.name]!.map((name) => ({
    name,
    type: fieldKinds.get(e.name + ":" + name) === "int" ? intShape.identity : textShape.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({ declarations, shapes: [textShape, intShape, ...errorShapes] });
const id = (declaration: string) => identity("error", declaration);
const server = createServer(domain, {
  invalidConfig: id("can.std.http@1::invalid_server_config"),
  bindFailed: id("can.std.http@1::bind_failed"),
  shutdownFailed: id("can.std.http@1::shutdown_failed"),
});
const router = createRouter(domain, {
  invalid: id("can.std.http@1::invalid_route"),
  duplicate: id("can.std.http@1::duplicate_route"),
  ambiguous: id("can.std.http@1::ambiguous_route"),
});
const responses = createResponses(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  limit: id("can.std.http@1::body_limit"),
});
const jsonActions = createJsonActions(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  limit: id("can.std.http@1::body_limit"),
});

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/;

function capture() {
  const lines: string[] = [];
  setRequestReportSink((line) => {
    lines.push(line);
  });
  return { lines, restore: () => setRequestReportSink(defaultRequestReportSink) };
}

function testContext(): RequestReportContext {
  return requestContext(undefined);
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

async function textResult(body: string): Promise<Completion<unknown>> {
  return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), body);
}

test("legacy handler throw reports once, redacted, with a fixed 500", async () => {
  const messageSecret = "e06-native-message-7q2x";
  const querySecret = "e06-query-token-9d4v";
  const bodySecret = "e06-body-secret-3m8k";
  const { lines, restore } = capture();
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await server.makeConfig("127.0.0.1", 18741n, 1048576n, 5000n));
      const thrown = value(
        await router.post("/boom", async () => {
          throw new Error("handler fault carries " + messageSecret);
        }),
      );
      const table = value(await router.make([thrown]));
      const token = value(await server.start(config, table));
      for (let round = 0; round < 2; round++) {
        const response = await fetch(`http://127.0.0.1:18741/boom?token=${querySecret}`, {
          method: "POST",
          headers: { "content-type": "text/plain", "x-secret": bodySecret },
          body: "payload " + bodySecret,
        });
        expect(response.status).toBe(500);
        expect(await response.text()).toBe("Internal Server Error");
        expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
      }
      expect(resourceStatus(token)).toMatchObject({ state: "open", leases: 0 });
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    expect(lines.length).toBe(2);
    const first = JSON.parse(lines[0]!),
      second = JSON.parse(lines[1]!);
    for (const parsed of [first, second]) {
      expect(parsed).toMatchObject({
        schemaVersion: 1,
        kind: "can.runtime-failure",
        phase: "request",
        channel: "standard",
        category: "native_exception",
        source: "route:POST /boom",
      });
      expect(typeof parsed.occurrence).toBe("string");
      expect(parsed.occurrence).toMatch(/^\d+$/);
      expect(parsed.correlation).toMatch(uuidPattern);
      expect(Array.isArray(parsed.frames)).toBe(true);
    }
    expect(first.occurrence).not.toBe(second.occurrence);
    expect(first.correlation).not.toBe(second.correlation);
    for (const line of lines) {
      expect(line).not.toContain("e06-native-message");
      expect(line).not.toContain("e06-query-token");
      expect(line).not.toContain("e06-body-secret");
      expect(line).not.toContain(messageSecret);
      expect(line).not.toContain(querySecret);
      expect(line).not.toContain(bodySecret);
    }
  } finally {
    restore();
  }
});

test("domain handler failure reports declaration identity with a redacted payload", async () => {
  const payloadSecret = "e06-payload-secret-5t6r";
  const declaration = "can.std.codec@1::invalid_data";
  const occurrence = domain.create(
    id(declaration),
    record(id(declaration), [
      ["path", "seal"],
      ["reason", payloadSecret],
    ]),
    origin,
  );
  const { lines, restore } = capture();
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await server.makeConfig("127.0.0.1", 18742n, 1048576n, 5000n));
      const failed = value(await router.get("/domain", async () => failure(occurrence)));
      const table = value(await router.make([failed]));
      const token = value(await server.start(config, table));
      const response = await fetch("http://127.0.0.1:18742/domain");
      expect(response.status).toBe(500);
      expect(await response.text()).toBe("Internal Server Error");
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    expect(lines.length).toBe(1);
    const parsed = JSON.parse(lines[0]!);
    const catalogueIdentity = catalogue.errors.find((e) => e.name === "codec::invalid_data")!;
    expect(parsed).toMatchObject({
      schemaVersion: 1,
      kind: "can.runtime-failure",
      phase: "request",
      channel: "domain",
      identity: "can.error.v2:" + catalogueIdentity.identity,
      error: "codec::invalid_data",
      typeIdentity: id(declaration),
      occurrence: String(domainFailureDiagnostics(occurrence).occurrenceID),
      payload: "<redacted>",
      source: "route:GET /domain",
    });
    expect(parsed.correlation).toMatch(uuidPattern);
    expect(lines[0]).not.toContain(payloadSecret);
    expect(lines[0]).not.toContain("e06-payload-secret");
  } finally {
    restore();
  }
});

test("server action-table failure reports the checked action identity", async () => {
  const messageSecret = "e06-action-message-2w9n";
  const { lines, restore } = capture();
  const actions = createServer(
    domain,
    {
      invalidConfig: id("can.std.http@1::invalid_server_config"),
      bindFailed: id("can.std.http@1::bind_failed"),
      shutdownFailed: id("can.std.http@1::shutdown_failed"),
    },
    undefined,
    {
      table: compileActionRoutes([
        { identity: "t16::boom", method: "GET", path: "/boom", captures: [] },
      ]),
      invoke: async () => {
        throw new Error("action fault carries " + messageSecret);
      },
    },
  );
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await actions.makeConfig("127.0.0.1", 18743n, 1048576n, 5000n));
      const legacy = value(await router.get("/legacy", async () => textResult("legacy")));
      const table = value(await router.make([legacy]));
      const token = value(await actions.start(config, table));
      const response = await fetch("http://127.0.0.1:18743/boom");
      expect(response.status).toBe(500);
      expect(await response.text()).toBe("Internal Server Error");
      expect((await actions.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    expect(lines.length).toBe(1);
    const parsed = JSON.parse(lines[0]!);
    expect(parsed).toMatchObject({
      channel: "standard",
      category: "native_exception",
      source: "action:t16::boom",
    });
    expect(parsed.correlation).toMatch(uuidPattern);
    expect(lines[0]).not.toContain(messageSecret);
    expect(lines[0]).not.toContain("e06-action-message");
  } finally {
    restore();
  }
});

const sealStr = { identity: "t16::str", kind: "primitive", name: "str" };
const sealInt = { identity: "t16::int", kind: "primitive", name: "int" };
const sealRequest: Schema = {
  root: "t16::seal_wire",
  nodes: [
    {
      identity: "t16::seal_wire",
      kind: "record",
      name: "seal_wire",
      fields: [
        { name: "label", type: "t16::str" },
        { name: "seats", type: "t16::int" },
      ],
    },
    sealStr,
    sealInt,
  ],
};
const sealResponse: Schema = {
  root: "t16::seal_outcome",
  nodes: [
    {
      identity: "t16::seal_outcome",
      kind: "variant",
      name: "seal_outcome",
      leaves: ["t16::sealed", "t16::seal_failed"],
    },
    {
      identity: "t16::sealed",
      kind: "record",
      name: "sealed",
      fields: [{ name: "label", type: "t16::str" }],
    },
    {
      identity: "t16::seal_failed",
      kind: "record",
      name: "seal_failed",
      fields: [{ name: "reason", type: "t16::str" }],
    },
    sealStr,
  ],
};
const sealEntry: EmittedActionEntry = {
  identity: "t16::seal_invoice",
  method: "POST",
  path: "/seal",
  captures: [],
  body: { mode: "json", type: "t16::seal_wire", schema: sealRequest },
  handler: "t16::seal_validated",
  result: "t16::seal_outcome",
  cases: [
    { leaf: "t16::sealed", status: 200 },
    { leaf: "t16::seal_failed", status: 422 },
  ],
  responseSchema: sealResponse,
};

test("JSON adapter handler failure reports once at the adapter", async () => {
  const messageSecret = "e06-adapter-message-8h3j";
  const broken: ActionJsonHandler = async () => {
    throw new Error("adapter fault carries " + messageSecret);
  };
  const { lines, restore } = capture();
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await server.makeConfig("127.0.0.1", 18744n, 65536n, 5000n));
      const route = value(await jsonActions.mount(router, sealEntry, broken));
      const table = value(await router.make([route]));
      const token = value(await server.start(config, table));
      const response = await fetch("http://127.0.0.1:18744/seal", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ label: "inv-1", seats: 2 }),
      });
      expect(response.status).toBe(500);
      expect(response.headers.get("content-type")).toBe("text/plain; charset=utf-8");
      expect(await response.text()).toBe("Internal Server Error");
      // Expected client rejections stay silent: the malformed body below
      // adds no report.
      const bad = await fetch("http://127.0.0.1:18744/seal", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: "{oops",
      });
      expect(bad.status).toBe(400);
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    // Exactly one record: the adapter reports, and the boundary sees a
    // successful 500 conversion, never a second report.
    expect(lines.length).toBe(1);
    const parsed = JSON.parse(lines[0]!);
    expect(parsed).toMatchObject({
      channel: "standard",
      category: "native_exception",
      source: "action:t16::seal_invoice",
    });
    expect(parsed.correlation).toMatch(uuidPattern);
    expect(lines[0]).not.toContain(messageSecret);
    expect(lines[0]).not.toContain("e06-adapter-message");
  } finally {
    restore();
  }
});

test("JSON adapter adaptation failure reports without reflecting the leaf", async () => {
  const forged: ActionJsonHandler = async () => success(record("t16::archived", [["label", "x"]]));
  const { lines, restore } = capture();
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await server.makeConfig("127.0.0.1", 18745n, 65536n, 5000n));
      const route = value(
        await jsonActions.mount(router, { ...sealEntry, path: "/forged" }, forged),
      );
      const table = value(await router.make([route]));
      const token = value(await server.start(config, table));
      const response = await fetch("http://127.0.0.1:18745/forged", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ label: "inv-1", seats: 2 }),
      });
      expect(response.status).toBe(500);
      expect(await response.text()).toBe("Internal Server Error");
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    expect(lines.length).toBe(1);
    const parsed = JSON.parse(lines[0]!);
    expect(parsed).toMatchObject({
      channel: "standard",
      category: "native_exception",
      source: "action:t16::seal_invoice",
    });
    // The synthesized record carries no handler value: the undeclared
    // leaf never reflects into the report.
    expect(lines[0]).not.toContain("t16::archived");
  } finally {
    restore();
  }
});

test("scope drain failure keeps the cleanup record single with a fixed 500", async () => {
  const closeSecret = "e06-close-secret-4f7d";
  const { lines, restore } = capture();
  const diagnostics: OwnerDiagnostic[] = [];
  try {
    const owned = await runOwnedRoot(
      async () => {
        const config = value(await server.makeConfig("127.0.0.1", 18746n, 1048576n, 5000n));
        const leaking = value(
          await router.get("/leak", async () => {
            registerResource("probe", {}, async () => {
              throw new Error("close fault carries " + closeSecret);
            });
            return textResult("done");
          }),
        );
        const table = value(await router.make([leaking]));
        const token = value(await server.start(config, table));
        const response = await fetch("http://127.0.0.1:18746/leak");
        expect(response.status).toBe(500);
        expect(await response.text()).toBe("Internal Server Error");
        expect((await server.stop(token)).kind).toBe("ok");
        return success(undefined);
      },
      (diagnostic) => {
        diagnostics.push(diagnostic);
      },
    );
    expect(owned.completion.kind).toBe("ok");
    // The failing auto-close marks the root; the occurrence was already
    // delivered as a cleanup diagnostic, so the request hook dedups.
    expect(owned.cleanupFailed).toBe(true);
    expect(diagnostics.length).toBeGreaterThanOrEqual(1);
    for (const diagnostic of diagnostics) {
      expect(diagnostic.phase).toBe("cleanup");
      expect(diagnostic.category).toBe("cleanup");
    }
    expect(JSON.stringify(diagnostics)).not.toContain(closeSecret);
    expect(lines.length).toBe(0);
  } finally {
    restore();
  }
});

test("a throwing wrapper still yields one redacted record", async () => {
  const innerSecret = "e06-inner-fault-6g2s";
  const wrapperSecret = "e06-wrapper-fault-1p5w";
  const { lines, restore } = capture();
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await server.makeConfig("127.0.0.1", 18747n, 1048576n, 5000n));
      const wrapped = value(
        await router.post("/wrapped", async () => {
          try {
            throw new Error("handler fault carries " + innerSecret);
          } catch {
            // The author logging wrapper faults while handling the fault
            // and masks it, like a failing manual log around a throw.
            throw new Error("wrapper fault carries " + wrapperSecret);
          }
        }),
      );
      const table = value(await router.make([wrapped]));
      const token = value(await server.start(config, table));
      const response = await fetch("http://127.0.0.1:18747/wrapped", { method: "POST" });
      expect(response.status).toBe(500);
      expect(await response.text()).toBe("Internal Server Error");
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    expect(lines.length).toBe(1);
    const parsed = JSON.parse(lines[0]!);
    expect(parsed).toMatchObject({
      channel: "standard",
      category: "native_exception",
      source: "route:POST /wrapped",
    });
    expect(lines[0]).not.toContain(innerSecret);
    expect(lines[0]).not.toContain(wrapperSecret);
  } finally {
    restore();
  }
});

test("a throwing sink never breaks the boundary", async () => {
  setRequestReportSink(() => {
    throw new Error("sink down");
  });
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await server.makeConfig("127.0.0.1", 18748n, 1048576n, 5000n));
      const thrown = value(
        await router.get("/boom", async () => {
          throw new Error("fault under a broken sink");
        }),
      );
      const table = value(await router.make([thrown]));
      const token = value(await server.start(config, table));
      const response = await fetch("http://127.0.0.1:18748/boom");
      expect(response.status).toBe(500);
      expect(await response.text()).toBe("Internal Server Error");
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  } finally {
    setRequestReportSink(defaultRequestReportSink);
  }
});

test("pre-dispatch asset failure reports under the fallback source", async () => {
  const assetSecret = "e06-asset-secret-9k1m";
  const { lines, restore } = capture();
  const assets = createServer(
    domain,
    {
      invalidConfig: id("can.std.http@1::invalid_server_config"),
      bindFailed: id("can.std.http@1::bind_failed"),
      shutdownFailed: id("can.std.http@1::shutdown_failed"),
    },
    {
      serve: async () => {
        throw new Error("asset fault carries " + assetSecret);
      },
    },
  );
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await assets.makeConfig("127.0.0.1", 18749n, 1048576n, 5000n));
      const route = value(await router.get("/ok", async () => textResult("ok")));
      const table = value(await router.make([route]));
      const token = value(await assets.start(config, table));
      const response = await fetch("http://127.0.0.1:18749/ok");
      expect(response.status).toBe(500);
      expect(await response.text()).toBe("Internal Server Error");
      expect((await assets.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    expect(lines.length).toBe(1);
    const parsed = JSON.parse(lines[0]!);
    expect(parsed).toMatchObject({ channel: "standard", source: "http" });
    expect(parsed.correlation).toMatch(uuidPattern);
    expect(lines[0]).not.toContain(assetSecret);
  } finally {
    restore();
  }
});

test("expected rejections and successes stay silent", async () => {
  const { lines, restore } = capture();
  try {
    const owned = await runOwnedRoot(async () => {
      const config = value(await server.makeConfig("127.0.0.1", 18750n, 16n, 5000n));
      const route = value(await router.get("/ok", async () => textResult("ok")));
      const table = value(await router.make([route]));
      const token = value(await server.start(config, table));
      const base = "http://127.0.0.1:18750";
      expect((await fetch(base + "/nope")).status).toBe(404);
      expect((await fetch(base + "/ok", { method: "POST" })).status).toBe(405);
      expect((await fetch(base + "/ok", { method: "POST", body: "x".repeat(1024) })).status).toBe(
        413,
      );
      const ok = await fetch(base + "/ok");
      expect(ok.status).toBe(200);
      expect(await ok.text()).toBe("ok");
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
    expect(lines.length).toBe(0);
  } finally {
    restore();
  }
});

test("request reporter delivers once per occurrence", async () => {
  const { lines, restore } = capture();
  try {
    const occurrence = captureStandard(new Error("once fault"), origin);
    const first = await reportRequestFailure(failure(occurrence), testContext());
    expect(first).toBeDefined();
    expect(Object.isFrozen(first)).toBe(true);
    expect(first).toMatchObject({
      schemaVersion: 1,
      kind: "can.runtime-failure",
      phase: "request",
      channel: "standard",
      category: "native_exception",
      occurrence: String(standardFailureDiagnostics(occurrence).occurrenceID),
      source: "http",
    });
    expect(first!.correlation).toMatch(uuidPattern);
    expect(await reportRequestFailure(failure(occurrence), testContext())).toBeUndefined();
    expect(lines.length).toBe(1);
    expect(lines[0]).toBe(JSON.stringify(first));
  } finally {
    restore();
  }
});

test("late-owner report claims before the request hook", async () => {
  const { lines, restore } = capture();
  try {
    const loserFault = captureStandard(new Error("late loser fault"), origin);
    const winner = deferred<Completion>(),
      loser = deferred<Completion>();
    const diagnostics: OwnerDiagnostic[] = [];
    const pending = runExplicitRoot(
      async (ctx) => {
        const group = launchOwnedWithContext(ctx, [
          { run: () => winner.promise, captures: [] },
          { run: () => loser.promise, captures: [] },
        ]);
        const first = await group.promises[0];
        group.publish([0]);
        return first;
      },
      (diagnostic) => {
        diagnostics.push(diagnostic);
      },
    );
    winner.resolve(success(1n));
    loser.resolve(failure(loserFault));
    const result = await pending;
    expect(value(result.completion)).toBe(1n);
    expect(result.cleanupFailed).toBe(false);
    expect(diagnostics.length).toBe(1);
    expect(diagnostics[0]).toMatchObject({
      phase: "late",
      category: "native_exception",
      occurrence: String(standardFailureDiagnostics(loserFault).occurrenceID),
    });
    // The late diagnostic delivered the occurrence; the request hook skips.
    expect(await reportRequestFailure(failure(loserFault), testContext())).toBeUndefined();
    expect(lines.length).toBe(0);
  } finally {
    restore();
  }
});

test("browser report claims before the request hook", async () => {
  const { lines, restore } = capture();
  const errors: unknown[][] = [];
  const realError = console.error;
  console.error = ((...args: unknown[]) => {
    errors.push(args);
  }) as typeof console.error;
  try {
    const occurrence = captureStandard(new Error("browser fault"), origin);
    const record = reportBrowserDiagnostic(failure(occurrence), "handler");
    expect(record?.category).toBe("native_exception");
    expect(errors.length).toBe(1);
    expect(await reportRequestFailure(failure(occurrence), testContext())).toBeUndefined();
    expect(lines.length).toBe(0);
  } finally {
    console.error = realError;
    restore();
  }
});

test("main report marks the claim; terminal delivery stays total", async () => {
  const { lines, restore } = capture();
  try {
    const mainFirst = captureStandard(new Error("main first"), origin);
    const entryLines: string[] = [];
    const mainExit = await runEntry(
      () => {},
      async () => failure(mainFirst),
      [],
      (line) => {
        entryLines.push(line);
      },
    );
    expect(mainExit).toBe(1);
    expect(entryLines.length).toBe(1);
    // Main marked the occurrence, so the request hook skips it.
    expect(await reportRequestFailure(failure(mainFirst), testContext())).toBeUndefined();
    expect(lines.length).toBe(0);

    // The reverse stays total: the terminal report always delivers,
    // even for an occurrence the request hook already delivered.
    const requestFirst = captureStandard(new Error("request first"), origin);
    const delivered = await reportRequestFailure(failure(requestFirst), testContext());
    expect(delivered?.channel).toBe("standard");
    expect(lines.length).toBe(1);
    const requestExit = await runEntry(
      () => {},
      async () => failure(requestFirst),
      [],
      (line) => {
        entryLines.push(line);
      },
    );
    expect(requestExit).toBe(1);
    expect(entryLines.length).toBe(2);
  } finally {
    restore();
  }
});

test("correlation and source resolve per request", () => {
  const native = new Request("http://127.0.0.1/native");
  expect(correlationFor(native)).toBe(correlationFor(native));
  expect(correlationFor(native)).toMatch(uuidPattern);
  expect(correlationFor(new Request("http://127.0.0.1/other"))).not.toBe(correlationFor(native));
  expect(requestSource(native)).toBe("http");
  noteRequestSource(native, "route:GET /native");
  expect(requestSource(native)).toBe("route:GET /native");
  // Invalid annotations never poison the record or inject a line break.
  noteRequestSource(native, "bad\nsource");
  noteRequestSource(native, "");
  expect(requestSource(native)).toBe("route:GET /native");
  noteRequestSource(undefined, "action:x");
  expect(requestSource(undefined)).toBe("http");
  expect(requestContext(native)).toMatchObject({
    correlation: correlationFor(native),
    source: "route:GET /native",
  });
  expect(requestContext(undefined)).toMatchObject({ source: "http" });
  expect(requestContext(undefined).correlation).toMatch(uuidPattern);
});

test("successful completions report nothing", async () => {
  const { lines, restore } = capture();
  try {
    expect(await reportRequestFailure(success(1n), testContext())).toBeUndefined();
    expect(lines.length).toBe(0);
  } finally {
    restore();
  }
});

test("raw thrown values box into redacted records under the fallback context", async () => {
  const errorSecret = "e06-raw-error-3v8c";
  const primitiveSecret = "e06-primitive-7b4n";
  const { lines, restore } = capture();
  try {
    const boxed = await reportRequestFailure(
      new Error("raw fault carries " + errorSecret),
      requestContext(undefined),
    );
    expect(boxed).toMatchObject({
      channel: "standard",
      category: "native_exception",
      source: "http",
    });
    const primitive = await reportRequestFailure(
      "primitive fault carries " + primitiveSecret,
      requestContext(undefined),
    );
    expect(primitive?.channel).toBe("standard");
    expect(lines.length).toBe(2);
    for (const line of lines) {
      expect(line).not.toContain(errorSecret);
      expect(line).not.toContain(primitiveSecret);
    }
  } finally {
    restore();
  }
});
