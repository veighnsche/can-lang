// E04: dispatch propagation legs — disconnect/shutdown expiry, bounded
// ingress, and the republished signal-path shutdown bound.
//
// Disconnect (X-R04-3, qualified) expires the per-request operation
// scope; every budgeted adapter resolves it implicitly, so in-flight
// work returns its unknown-write outcome at the boundary while staying
// owned until settlement. Ingress aborts map to 408; corrupt reads
// stay 400. Header stalls never dispatch (Bun-owned, no Can hook);
// that limitation is pinned, not worked around.
import { describe, expect, test } from "bun:test";
import { spawn } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { runOwnedRoot } from "../owner.ts";
import { createServer } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import {
  createResponses,
  snapshotRequest,
  snapshotRequestLazy,
  snapshotBodyBytes,
} from "../platform/http.ts";
import { createSQLDescriptors, type SQLDescriptorEntry } from "../platform/sql/descriptor.ts";
import { createSQLPools } from "../platform/sql/pool.ts";
import type { SQLPlan } from "../platform/sql/values.ts";
import { raceBoundary } from "../transport/operation-budget.ts";
import { createRequestScope, currentRequestScope } from "../transport/request-scope.ts";
import type { RequestScope } from "../transport/request-scope.ts";

const PG_URL = process.env["CAN_TEST_POSTGRES_URL"];
const RUN_DB = process.env["CAN_E04_DB"] ?? "can_e04_run2";
const pgLive = test.skipIf(PG_URL === undefined);

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
  "http::credentials_missing": ["variable"],
  "codec::invalid_data": ["path", "reason"],
  "sql::connection_failed": ["phase"],
  "sql::query_failed": ["operation", "code"],
  "sql::row_missing": ["query"],
  "sql::row_count": ["query", "actual"],
  "sql::schema_mismatch": ["path", "reason"],
  "sql::constraint_failed": ["constraint"],
  "sql::close_failed": ["reason"],
  "sql::row_limit": ["limit"],
  "sql::unsupported_value": ["path", "reason"],
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
  close: "unused",
  writeFailed: "unused",
  limit: id("can.std.http@1::body_limit"),
});

async function textResult(body: string): Promise<Completion<unknown>> {
  return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), body);
}

async function startOn(
  port: number,
  handler: (request?: unknown) => Promise<Completion<unknown>>,
  options?: { shutdownMs?: bigint; bodyLimit?: bigint },
): Promise<{ token: unknown; config: unknown }> {
  const config = value(
    await server.makeConfig(
      "127.0.0.1",
      BigInt(port),
      options?.bodyLimit ?? 1048576n,
      options?.shutdownMs ?? 5000n,
    ),
  );
  const get = value(await router.get("/x", handler));
  const post = value(await router.post("/x", handler));
  const table = value(await router.make([get, post]));
  const started = await server.start(config, table);
  if (started.kind !== "ok") throw new Error("test server failed to start");
  return { token: value(started), config };
}

async function waitFor(label: string, ready: () => boolean, budgetMs: number): Promise<void> {
  const started = Date.now();
  while (!ready()) {
    if (Date.now() - started > budgetMs) throw new Error(`timed out waiting for ${label}`);
    await Bun.sleep(20);
  }
}

// SQL slice for the live dispatch leg (own isolated database, env only).
const pgLookup = (name: string): string | undefined =>
  name === "CAN_E04_PG" && PG_URL !== undefined ? PG_URL.replace("/<db>", `/${RUN_DB}`) : undefined;
const pgTable: Record<string, Record<string, SQLDescriptorEntry>> = {
  pg: {
    drop_probe: {
      dialect: "postgresql",
      cardinality: "execute",
      kind: "execute_statement",
      segments: [{ text: "DROP TABLE IF EXISTS e04_probe" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: 170007,
    },
    setup_probe: {
      dialect: "postgresql",
      cardinality: "execute",
      kind: "execute_statement",
      segments: [{ text: "CREATE TABLE e04_probe (id INTEGER PRIMARY KEY, note TEXT)" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: 170007,
    },
    seed_probe: {
      dialect: "postgresql",
      cardinality: "execute",
      kind: "execute_statement",
      segments: [{ text: "INSERT INTO e04_probe (id, note) VALUES (1, 'seed')" }],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 0,
      total: 0,
      version: 170007,
    },
    sleep_mark: {
      dialect: "postgresql",
      cardinality: "many",
      kind: "select_statement",
      segments: [
        { text: "SELECT 'slept' AS mark FROM e04_probe WHERE pg_sleep(2) IS NOT NULL LIMIT " },
        { param: 1 },
      ],
      params: [],
      paramType: "p",
      rowType: "r",
      limit: 1,
      total: 1,
      version: 170007,
    },
  },
};
const sqlDescriptors = createSQLDescriptors(pgTable);
const pools = createSQLPools(
  domain,
  {
    connectionFailed: id("can.std.sql@1::connection_failed"),
    queryFailed: id("can.std.sql@1::query_failed"),
    rowMissing: id("can.std.sql@1::row_missing"),
    rowCount: id("can.std.sql@1::row_count"),
    schemaMismatch: id("can.std.sql@1::schema_mismatch"),
    constraintFailed: id("can.std.sql@1::constraint_failed"),
    rowLimit: id("can.std.sql@1::row_limit"),
    unsupportedValue: id("can.std.sql@1::unsupported_value"),
    credentialsMissing: id("can.std.http@1::credentials_missing"),
    closeFailed: id("can.std.sql@1::close_failed"),
  },
  pgLookup,
  sqlDescriptors,
);
const markPlan: SQLPlan = {
  params: { root: "app::empty", fields: [] },
  rows: { root: "app::mark_row", fields: [{ name: "mark", kind: "str" }] },
};
const emptyPlan: SQLPlan = { params: { root: "app::empty", fields: [] } };
const emptyParams = record("app::empty", []);

describe("dispatch propagation", () => {
  test("disconnect expires the scope; the boundary returns unknown while work stays owned", async () => {
    const seen: Record<string, unknown> = {};
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18761, async () => {
        const scope = currentRequestScope();
        seen.scopeInstalled = scope !== undefined;
        seen.scope = scope;
        const start = Date.now();
        const outcome = await raceBoundary(
          {
            source: "worker",
            boundMs: 30000,
            budget: scope?.budget,
            scopeSignal: scope?.signal,
            sink: scope?.sink,
          },
          () => Bun.sleep(2000).then(() => "slow-value"),
        );
        seen.boundaryAt = Date.now() - start;
        if (outcome.kind === "unknown") {
          seen.cause = outcome.escalation.cause;
          return textResult("unknown");
        }
        return textResult("settled");
      });
      const stop = new AbortController();
      const outcome = fetch("http://127.0.0.1:18761/x", { signal: stop.signal }).then(
        () => "responded",
        (cause: unknown) => (cause instanceof Error ? cause.name : typeof cause),
      );
      await Bun.sleep(300);
      stop.abort();
      expect(await outcome).toBe("AbortError");
      // The handler observed the disconnect promptly at its own
      // boundary — long before the two-second operation settled.
      await waitFor("handler boundary", () => seen.boundaryAt !== undefined, 5000);
      expect(seen.scopeInstalled).toBe(true);
      expect(seen.boundaryAt as number).toBeLessThan(1500);
      expect(seen.cause).toBe("disconnect");
      const scope = seen.scope as RequestScope;
      expect(scope.collected.escalations).toHaveLength(1);
      // The operation ran to settlement under ownership, then the
      // server kept serving: no corruption, no leaked scope.
      await waitFor("late settlement", () => scope.collected.lates.length === 1, 8000);
      expect(scope.collected.lates[0]).toMatchObject({ settled: "resolved" });
      const again = await fetch("http://127.0.0.1:18761/x");
      expect(again.status).toBe(200);
      expect(await again.text()).toBe("settled");
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  pgLive(
    "disconnect into a live SQL query returns the budget outcome at the boundary",
    async () => {
      const seen: Record<string, unknown> = {};
      const owned = await runOwnedRoot(async () => {
        const pool = value(await pools.open("CAN_E04_PG", 5n));
        for (const name of ["drop_probe", "setup_probe", "seed_probe"]) {
          value(
            await pools.execute(
              sqlDescriptors.declareDescriptor("pg", name),
              emptyPlan,
              pool,
              emptyParams,
            ),
          );
        }
        const { token } = await startOn(18762, async () => {
          const scope = currentRequestScope();
          seen.scope = scope;
          const start = Date.now();
          // No explicit bounds: the ambient request scope alone
          // propagates the disconnect into the query.
          const outcome = await pools.queryRows(
            sqlDescriptors.declareDescriptor("pg", "sleep_mark"),
            markPlan,
            pool,
            emptyParams,
            10n,
          );
          seen.boundaryAt = Date.now() - start;
          if (outcome.kind !== "domain") return textResult("settled");
          const details = domainFailureDiagnostics(outcome.value);
          seen.failure = details.declaration.name;
          // Object.entries skips the nominal brand symbol toEqual
          // would otherwise trip over.
          seen.payload = Object.fromEntries(
            Object.entries(details.payload as Record<string, unknown>),
          );
          return textResult("budget");
        });
        const stop = new AbortController();
        const outcome = fetch("http://127.0.0.1:18762/x", { signal: stop.signal }).then(
          () => "responded",
          (cause: unknown) => (cause instanceof Error ? cause.name : typeof cause),
        );
        await Bun.sleep(400);
        stop.abort();
        expect(await outcome).toBe("AbortError");
        await waitFor("sql boundary", () => seen.boundaryAt !== undefined, 5000);
        expect(seen.boundaryAt as number).toBeLessThan(1500);
        expect(seen.failure).toBe("sql::query_failed");
        expect(seen.payload).toEqual({ operation: "query_rows", code: "budget" });
        const scope = seen.scope as RequestScope;
        expect(scope.collected.escalations).toHaveLength(1);
        expect(scope.collected.escalations[0]).toMatchObject({ cause: "disconnect" });
        await waitFor("late sql settlement", () => scope.collected.lates.length === 1, 8000);
        // The pool stayed usable throughout: a second request runs
        // the same query to full settlement.
        const again = await fetch("http://127.0.0.1:18762/x");
        expect(again.status).toBe(200);
        expect(await again.text()).toBe("settled");
        expect((await server.stop(token)).kind).toBe("ok");
        value(await pools.close(pool, 5000n));
        return success(undefined);
      });
      expect(owned.cleanupFailed).toBe(false);
      expect(owned.completion.kind).toBe("ok");
    },
    30000,
  );

  test("stop expires scopes; the live peer gets the bounded response", async () => {
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(
        18763,
        async () => {
          const scope = currentRequestScope();
          const outcome = await raceBoundary(
            {
              source: "worker",
              boundMs: 30000,
              budget: scope?.budget,
              scopeSignal: scope?.signal,
              sink: scope?.sink,
            },
            () => Bun.sleep(5000).then(() => "slow-value"),
          );
          if (outcome.kind === "unknown") return textResult(`unknown:${outcome.escalation.cause}`);
          return textResult("settled");
        },
        { shutdownMs: 5000n },
      );
      const response = await Promise.all([
        fetch("http://127.0.0.1:18763/x").then((r) => r.text()),
        Bun.sleep(200).then(() => server.stop(token)),
      ]);
      // The peer stayed connected and received the bounded shutdown
      // outcome instead of waiting out the five-second operation.
      expect(response[0]).toBe("unknown:shutdown");
      expect(response[1].kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("header stalls never dispatch (Bun-owned, no Can hook)", async () => {
    let fires = 0;
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18764, async () => {
        fires++;
        return textResult("x");
      });
      const sock = await Bun.connect({
        hostname: "127.0.0.1",
        port: 18764,
        socket: { data() {}, close() {}, connectError() {}, timeout() {}, error() {} },
      });
      // Partial headers, never completed: Bun holds the connection
      // without ever invoking dispatch.
      sock.write("POST /x HTTP/1.1\r\nhost: x\r\ncontent-length: 100\r\nx-partial: abc");
      await Bun.sleep(2000);
      expect(fires).toBe(0);
      sock.end();
      expect((await server.stop(token)).kind).toBe("ok");
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);

  test("shutdown during ingress answers the stalled body with 408", async () => {
    const owned = await runOwnedRoot(async () => {
      const { token } = await startOn(18767, async () => textResult("x"));
      const chunks: string[] = [];
      const decoder = new TextDecoder();
      const sock = await Bun.connect({
        hostname: "127.0.0.1",
        port: 18767,
        socket: {
          data(_socket, data) {
            chunks.push(decoder.decode(data));
          },
          close() {},
          connectError() {},
          timeout() {},
          error() {},
        },
      });
      // Full headers, zero body bytes: ingress pends in the drain.
      sock.write("POST /x HTTP/1.1\r\nhost: x\r\ncontent-length: 100\r\n\r\n");
      await Bun.sleep(300);
      expect((await server.stop(token)).kind).toBe("ok");
      await waitFor("408 bytes", () => chunks.join("").includes("408"), 5000);
      sock.end();
      return success(undefined);
    });
    expect(owned.cleanupFailed).toBe(false);
    expect(owned.completion.kind).toBe("ok");
  }, 30000);
});

describe("bounded ingress", () => {
  test("scope expiry abandons a stalled drain with 408", async () => {
    const scope = createRequestScope();
    const request = new Request("http://127.0.0.1/x", {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: new ReadableStream<Uint8Array>({}),
    });
    const draining = snapshotRequest(request, 1024, scope.signal);
    await Bun.sleep(50);
    scope.expire("shutdown");
    const started = Date.now();
    const snapshot = await draining;
    expect(Date.now() - started).toBeLessThan(1500);
    expect(snapshot).toEqual({ kind: "rejected", status: 408 });
  });

  test("a pre-expired scope rejects without reading", async () => {
    const scope = createRequestScope();
    scope.expire("disconnect");
    let pulls = 0;
    const request = new Request("http://127.0.0.1/x", {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: new ReadableStream<Uint8Array>({
        pull() {
          pulls++;
        },
      }),
    });
    expect(await snapshotRequest(request, 1024, scope.signal)).toEqual({
      kind: "rejected",
      status: 408,
    });
    expect(pulls).toBe(0);
  });

  test("Bun-side stream aborts map to 408, corrupt reads stay 400", async () => {
    const aborted = new Request("http://127.0.0.1/x", {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: new ReadableStream<Uint8Array>({
        start(controller) {
          controller.error(new DOMException("timed out", "AbortError"));
        },
      }),
    });
    expect(await snapshotRequest(aborted, 1024)).toEqual({ kind: "rejected", status: 408 });
    const corrupt = new Request("http://127.0.0.1/x", {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: new ReadableStream<Uint8Array>({
        start(controller) {
          controller.error(new Error("corrupt"));
        },
      }),
    });
    expect(await snapshotRequest(corrupt, 1024)).toEqual({ kind: "rejected", status: 400 });
  });

  test("over-limit bodies still reject with 413", async () => {
    const request = new Request("http://127.0.0.1/x", {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: new Uint8Array(100),
    });
    expect(await snapshotRequest(request, 16)).toEqual({ kind: "rejected", status: 413 });
  });

  test("complete bodies drain identically with a live scope", async () => {
    const scope = createRequestScope();
    const request = new Request("http://127.0.0.1/x", {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: new Uint8Array([1, 2, 3]),
    });
    const snapshot = await snapshotRequest(request, 1024, scope.signal);
    expect(snapshot.kind).toBe("request");
  });

  test("scope expiry abandons a lazy live read as failed", async () => {
    const scope = createRequestScope();
    const request = new Request("http://127.0.0.1/x", {
      method: "POST",
      headers: { "content-type": "text/plain" },
      body: new ReadableStream<Uint8Array>({}),
    });
    const snapshot = await snapshotRequestLazy(request, 1024, scope.signal);
    expect(snapshot.kind).toBe("request");
    if (snapshot.kind !== "request") throw new Error("wrong outcome");
    const reading = snapshotBodyBytes(snapshot.value);
    await Bun.sleep(50);
    scope.expire("shutdown");
    expect(await reading).toMatchObject({ kind: "failed" });
  });
});

// Child-process signal legs: signals cannot target the runner itself.
const childRuntime = fileURLToPath(new URL("../", import.meta.url));
function childPrelude(errors: readonly string[]): string {
  return `import {createHash} from "node:crypto";
import {catalogue} from ${JSON.stringify(join(childRuntime, "catalogue.ts"))};
import {createDomainRuntime,domainFailureDiagnostics} from ${JSON.stringify(join(childRuntime, "domain.ts"))};
import {success,value} from ${JSON.stringify(join(childRuntime, "completion.ts"))};
import {runOwnedRoot} from ${JSON.stringify(join(childRuntime, "owner.ts"))};
import {createServer} from ${JSON.stringify(join(childRuntime, "platform/server.ts"))};
import {createRouter} from ${JSON.stringify(join(childRuntime, "platform/router.ts"))};
import {createResponses} from ${JSON.stringify(join(childRuntime, "platform/http.ts"))};
import {raceBoundary} from ${JSON.stringify(join(childRuntime, "transport/operation-budget.ts"))};
import {currentRequestScope} from ${JSON.stringify(join(childRuntime, "transport/request-scope.ts"))};
const identity=(kind,declaration)=>createHash("sha256").update("can-concrete-type-v1\\0"+JSON.stringify([kind,declaration])).digest("hex");
const textShape={identity:identity("primitive","str"),kind:"primitive",declaration:"str",arguments:[],fields:[],leaves:[],inputs:[],errors:[]};
const wanted=new Set(${JSON.stringify(errors)});
const declarations=catalogue.errors.filter(e=>wanted.has(e.name)).map(e=>({identity:e.identity,name:e.name,parameters:0}));
const lookup=Object.fromEntries(catalogue.errors.filter(e=>wanted.has(e.name)).map(e=>[e.name,e.fields.map(f=>f.name)]));
const shapes=declarations.map(e=>({identity:identity("error",e.identity),kind:"error",declaration:e.identity,arguments:[],fields:lookup[e.name].map(name=>({name,type:textShape.identity})),leaves:[],inputs:[],errors:[]}));
const domain=createDomainRuntime({declarations,shapes:[textShape,...shapes]});
const id=declaration=>identity("error",declaration);
const server=createServer(domain,{invalidConfig:id("can.std.http@1::invalid_server_config"),bindFailed:id("can.std.http@1::bind_failed"),shutdownFailed:id("can.std.http@1::shutdown_failed")});
const router=createRouter(domain,{invalid:id("can.std.http@1::invalid_route"),duplicate:id("can.std.http@1::duplicate_route"),ambiguous:id("can.std.http@1::ambiguous_route")});
const responses=createResponses(domain,{invalid:id("can.std.http@1::invalid_request"),invalidData:id("can.std.codec@1::invalid_data")});
`;
}
async function runChild(
  name: string,
  script: string,
  interact: (child: ReturnType<typeof spawn>, output: () => string) => Promise<void>,
  killAfterMs = 10000,
): Promise<{ output: string; errors: string; code: number | null; signal: NodeJS.Signals | null }> {
  const directory = mkdtempSync(join(tmpdir(), "can-server-budget-" + name + "-"));
  const file = join(directory, name + ".ts");
  writeFileSync(file, script);
  const child = spawn(process.execPath, ["--no-install", file], {
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "",
    errors = "";
  child.stdout.on("data", (chunk) => {
    output += chunk;
  });
  child.stderr.on("data", (chunk) => {
    errors += chunk;
  });
  const deadline = setTimeout(() => child.kill("SIGKILL"), killAfterMs);
  try {
    await interact(child, () => output);
    const status = await new Promise<{ code: number | null; signal: NodeJS.Signals | null }>(
      (resolve, reject) => {
        child.once("error", reject);
        child.once("exit", (code, signal) => resolve({ code, signal }));
      },
    );
    return { output, errors, ...status };
  } finally {
    clearTimeout(deadline);
    child.kill("SIGKILL");
    rmSync(directory, { recursive: true, force: true });
  }
}
async function waitForOutput(output: () => string, marker: string): Promise<void> {
  const begin = Date.now();
  while (!output().includes(marker)) {
    if (Date.now() - begin > 8000) throw new Error("missing child marker " + marker);
    await Bun.sleep(20);
  }
}

describe("signal propagation", () => {
  test("SIGTERM propagates shutdown; the live peer gets the bounded response", async () => {
    const script =
      childPrelude([
        "http::invalid_server_config",
        "http::bind_failed",
        "http::shutdown_failed",
        "http::invalid_route",
        "http::duplicate_route",
        "http::ambiguous_route",
        "http::invalid_request",
        "codec::invalid_data",
      ]) +
      `
await runOwnedRoot(async ()=>{
 const config=value(await server.makeConfig("127.0.0.1",18765n,1048576n,5000n));
 const route=value(await router.get("/slow",async ()=>{
   console.log("handler-enter");
   const scope=currentRequestScope();
   const outcome=await raceBoundary({source:"worker",boundMs:30000,budget:scope?.budget,scopeSignal:scope?.signal,sink:scope?.sink},()=>Bun.sleep(1500).then(()=>"slow-value"));
   if(outcome.kind==="unknown"){console.log("boundary:"+outcome.escalation.cause);return responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),"unknown:"+outcome.escalation.cause);}
   return responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),"settled");
 }));
 const table=value(await router.make([route]));
 const token=value(await server.start(config,table));
 console.log("listening");
 const pending=server.wait(token).then(c=>{console.log("wait:"+c.kind);return c;});
 const _keep=token;
 await pending;
 return success(undefined);
});`;
    const result = await runChild("sigterm", script, async (child, output) => {
      await waitForOutput(output, "listening");
      const pending = fetch("http://127.0.0.1:18765/slow").then((r) => r.text());
      await waitForOutput(output, "handler-enter");
      await Bun.sleep(200);
      child.kill("SIGTERM");
      // The connected peer receives the bounded shutdown outcome.
      expect(await pending).toBe("unknown:shutdown");
    });
    expect(result.output).toContain("listening");
    expect(result.output).toContain("handler-enter");
    expect(result.output).toContain("boundary:shutdown");
    expect(result.output).toContain("wait:ok");
    expect(result.errors).toBe("");
    expect(result.signal).toBe(null);
    expect(result.code).toBe(0);
  }, 30000);

  test("a signal past shutdownMs reports the deadline with the close owned", async () => {
    const script =
      childPrelude([
        "http::invalid_server_config",
        "http::bind_failed",
        "http::shutdown_failed",
        "http::invalid_route",
        "http::duplicate_route",
        "http::ambiguous_route",
        "http::invalid_request",
        "codec::invalid_data",
      ]) +
      `
await runOwnedRoot(async ()=>{
 const never=new Promise(()=>{});
 const config=value(await server.makeConfig("127.0.0.1",18766n,1048576n,200n));
 const route=value(await router.get("/hung",async ()=>{console.log("entered");await never;return responses.text(value(await responses.ok()),value(await responses.emptyHeaders()),"late");}));
 const table=value(await router.make([route]));
 const token=value(await server.start(config,table));
 console.log("listening");
 const pending=fetch("http://127.0.0.1:18766/hung").catch(()=>{});
 const waited=await server.wait(token);
 console.log("wait:"+waited.kind+(waited.kind==="domain"?":"+JSON.stringify(domainFailureDiagnostics(waited.value).payload):""));
 return success(undefined);
});`;
    const result = await runChild(
      "sigdeadline",
      script,
      async (child, output) => {
        await waitForOutput(output, "listening");
        // The child fetches itself, so the handler is entered before
        // the parent signals; the unbudgeted hang ignores the scope.
        await waitForOutput(output, "entered");
        await Bun.sleep(100);
        const signalled = Date.now();
        child.kill("SIGTERM");
        await waitForOutput(output, "wait:domain");
        // Bounded by shutdownMs (200ms), not by the hung handler.
        expect(Date.now() - signalled).toBeLessThan(5000);
        await Bun.sleep(100);
        // The close stays owned past the deadline: only the external
        // supervisor bounds the nonsettling work.
        child.kill("SIGKILL");
      },
      15000,
    );
    expect(result.output).toContain("listening");
    expect(result.output).toContain("wait:domain");
    expect(result.output).toContain("deadline");
    expect(result.errors).toBe("");
    expect(result.signal).toBe("SIGKILL");
  }, 30000);
});
