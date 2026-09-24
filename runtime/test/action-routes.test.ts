import { test, expect } from "bun:test";
import { connect } from "node:net";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { success, value, type Completion } from "../completion.ts";
import { runOwnedRoot, resourceStatus } from "../owner.ts";
import {
  compileActionRoutes,
  matchActionRoute,
  buildActionURL,
  bunRouteKeys,
  ActionRouteIssue,
  type ActionRouteTable,
  type ActionRouteCapture,
} from "../platform/action-routes.ts";
import { createServer, type ActionInvoke } from "../platform/server.ts";
import { createRouter } from "../platform/router.ts";
import { createRequests, createResponses } from "../platform/http.ts";

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
  "stream::close_failed": ["reason"],
  "stream::write_failed": ["reason"],
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
const responses = createResponses(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  invalidData: id("can.std.codec@1::invalid_data"),
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  limit: id("can.std.http@1::body_limit"),
});
const requests = createRequests(domain, {
  invalid: id("can.std.http@1::invalid_request"),
  limit: id("can.std.http@1::body_limit"),
  invalidData: id("can.std.codec@1::invalid_data"),
  header: "header",
  close: id("can.std.stream@1::close_failed"),
  writeFailed: id("can.std.stream@1::write_failed"),
  multipartForm: "unused",
  multipartField: "unused",
  multipartFile: "unused",
});
const router = createRouter(domain, {
  invalid: id("can.std.http@1::invalid_route"),
  duplicate: id("can.std.http@1::duplicate_route"),
  ambiguous: id("can.std.http@1::ambiguous_route"),
});

function issueOf(run: () => unknown): ActionRouteIssue {
  try {
    run();
  } catch (error) {
    expect(error).toBeInstanceOf(ActionRouteIssue);
    return error as ActionRouteIssue;
  }
  throw new Error("expected ActionRouteIssue");
}

const entries = [
  { identity: "web::load_new", method: "GET", path: "/invoices/new", captures: [] },
  {
    identity: "web::load_one",
    method: "GET",
    path: "/invoices/{invoice_id}",
    captures: [{ name: "invoice_id", type: "str" }],
  },
  {
    identity: "web::load_line",
    method: "GET",
    path: "/invoices/{invoice_id}/lines/{line}",
    captures: [
      { name: "invoice_id", type: "str" },
      { name: "line", type: "int" },
    ],
  },
  {
    identity: "web::save_one",
    method: "POST",
    path: "/invoices/{invoice_id}",
    captures: [{ name: "invoice_id", type: "str" }],
  },
  { identity: "web::save_wire", method: "POST", path: "/invoices/save", captures: [] },
] as const;

function entryOf(
  path: string,
  captures: readonly { name: string; type: "str" | "int" }[] = [],
  identity = "web::x",
  method = "GET",
): Record<string, unknown> {
  return { identity, method, path, captures: [...captures] };
}

test("compiles a guarded table with canonical bun keys", () => {
  const table = compileActionRoutes([...entries]);
  expect(Object.isFrozen(table)).toBe(true);
  expect(Object.isFrozen(table.routes)).toBe(true);
  expect(table.routes.map((route) => route.identity)).toEqual([
    "web::load_new",
    "web::load_one",
    "web::load_line",
    "web::save_one",
    "web::save_wire",
  ]);
  expect(bunRouteKeys(table)).toEqual([
    "/invoices/new",
    "/invoices/:invoice_id",
    "/invoices/:invoice_id/lines/:line",
    "/invoices/:invoice_id",
    "/invoices/save",
  ]);
  expect(table.routes.map((route) => route.method)).toEqual(["GET", "GET", "GET", "POST", "POST"]);
  expect(table.routes.map((route) => route.statics)).toEqual([2, 1, 2, 1, 2]);
  expect(Object.isFrozen(table.routes[2]!.segments)).toBe(true);
  expect(table.routes[2]!.segments.map((segment) => segment.kind)).toEqual([
    "static",
    "capture",
    "static",
    "capture",
  ]);
});

test("rejects malformed route entries", () => {
  const bad: [string, unknown][] = [
    ["null entry", null],
    ["empty identity", { ...entryOf("/a"), identity: "" }],
    ["missing identity", { method: "GET", path: "/a", captures: [] }],
    ["bad method", entryOf("/a", [], "web::x", "PUT")],
    ["lowercase method", entryOf("/a", [], "web::x", "get")],
    ["missing slash", entryOf("a")],
    ["leading double slash", entryOf("//a")],
    ["space", entryOf("/a b")],
    ["query", entryOf("/a?b")],
    ["fragment", entryOf("/a#b")],
    ["wildcard", entryOf("/a*b")],
    ["backslash", entryOf("/a\\b")],
    ["legacy capture", entryOf("/a/:b")],
    ["partial capture", entryOf("/a/{b}.json")],
    ["empty capture", entryOf("/a/{}")],
    ["capital capture", entryOf("/a/{B}")],
    ["underscore capture", entryOf("/a/{_b}")],
    ["duplicate capture", entryOf("/a/{b}/{b}")],
    ["non-array captures", { identity: "web::x", method: "GET", path: "/a" }],
    ["bad row type", entryOf("/a/{b}", [{ name: "b", type: "bool" as "str" }])],
    [
      "duplicate rows",
      entryOf("/a/{b}", [
        { name: "b", type: "str" },
        { name: "b", type: "str" },
      ]),
    ],
    ["capture without row", entryOf("/a/{b}", [])],
    ["row without capture", entryOf("/a", [{ name: "b", type: "str" }])],
    ["reserved prefix", entryOf("/__can/x")],
    ["reserved exact", entryOf("/__can")],
    ["malformed escape", entryOf("/a/%ZZ")],
    ["decoded control", entryOf("/a/%00")],
    ["decoded backslash", entryOf("/a/%5c")],
    ["non-string path", { identity: "web::x", method: "GET", path: 42, captures: [] }],
    ["lone surrogate", entryOf("/a\ud800")],
  ];
  expect(bad.length).toBe(30);
  for (const [name, entry] of bad) {
    const issue = issueOf(() => compileActionRoutes([entry]));
    expect(`${name}: ${issue.code}`).toBe(`${name}: invalid-route`);
  }
});

test("rejects duplicate and ambiguous collisions", () => {
  const duped = issueOf(() =>
    compileActionRoutes([
      entryOf("/a/{x}", [{ name: "x", type: "str" }], "web::first"),
      entryOf("/a/{y}", [{ name: "y", type: "str" }], "web::second"),
    ]),
  );
  expect(duped.code).toBe("duplicate-route");
  expect(duped.detail).toContain("duplicates the GET /a/{} route");
  const staticDuped = issueOf(() =>
    compileActionRoutes([entryOf("/a/b", [], "web::first"), entryOf("/a/b", [], "web::second")]),
  );
  expect(staticDuped.code).toBe("duplicate-route");
  const encodedDuped = issueOf(() =>
    compileActionRoutes([entryOf("/a%41b", [], "web::first"), entryOf("/aAb", [], "web::second")]),
  );
  expect(encodedDuped.code).toBe("duplicate-route");
  expect(encodedDuped.detail).toContain("duplicates the GET /aAb route");
  const split = compileActionRoutes([
    entryOf("/a/{x}", [{ name: "x", type: "str" }], "web::first", "GET"),
    entryOf("/a/{y}", [{ name: "y", type: "str" }], "web::second", "POST"),
  ]);
  expect(split.routes.length).toBe(2);
  const ambiguous = issueOf(() =>
    compileActionRoutes([
      entryOf("/a/{x}/c", [{ name: "x", type: "str" }], "web::first"),
      entryOf("/a/b/{y}", [{ name: "y", type: "str" }], "web::second"),
    ]),
  );
  expect(ambiguous.code).toBe("ambiguous-route");
  expect(ambiguous.detail).toContain("ambiguously overlaps the GET /a/b/{} route");
  const prioritized = compileActionRoutes([
    entryOf("/a/b", [], "web::static"),
    entryOf("/a/{x}", [{ name: "x", type: "str" }], "web::capture"),
  ]);
  expect(prioritized.routes.length).toBe(2);
  const covered = compileActionRoutes([
    entryOf("/a/{x}/c", [{ name: "x", type: "str" }], "web::general"),
    entryOf("/a/b/c", [], "web::specific"),
  ]);
  expect(covered.routes.length).toBe(2);
  const arities = compileActionRoutes([
    entryOf("/a/{x}", [{ name: "x", type: "str" }], "web::short"),
    entryOf("/a/{x}/c", [{ name: "x", type: "str" }], "web::long"),
  ]);
  expect(arities.routes.length).toBe(2);
});

test("builds canonical URLs that round-trip through dispatch", () => {
  const table = compileActionRoutes([...entries]);
  const cases: [string, readonly (string | bigint)[], string][] = [
    ["web::load_new", [], "/invoices/new"],
    ["web::load_one", ["acme"], "/invoices/acme"],
    ["web::load_one", ["a b"], "/invoices/a%20b"],
    ["web::load_one", ["é"], "/invoices/%C3%A9"],
    ["web::load_one", ["a%"], "/invoices/a%25"],
    ["web::load_one", ["a+b"], "/invoices/a%2Bb"],
    ["web::load_line", ["9", 3n], "/invoices/9/lines/3"],
    ["web::load_line", ["x", 0n], "/invoices/x/lines/0"],
    ["web::load_line", ["x", -1n], "/invoices/x/lines/-1"],
    ["web::load_line", ["x", 9223372036854775807n], "/invoices/x/lines/9223372036854775807"],
    ["web::load_line", ["x", -(2n ** 63n)], "/invoices/x/lines/-9223372036854775808"],
    ["web::save_wire", [], "/invoices/save"],
  ];
  expect(cases.length).toBe(12);
  for (const [identity, values, url] of cases) {
    expect(buildActionURL(table, identity, values)).toBe(url);
    const match = matchActionRoute(table, identity === "web::save_wire" ? "POST" : "GET", url);
    expect(match.kind).toBe("match");
    if (match.kind !== "match") throw new Error("built URL does not dispatch");
    expect(match.identity).toBe(identity);
    expect(match.captures.map((capture) => capture.value)).toEqual([...values]);
  }
});

test("rejects unbuildable URLs", () => {
  const table = compileActionRoutes([...entries]);
  expect(issueOf(() => buildActionURL(table, "web::missing", [])).code).toBe("unknown-action");
  expect(issueOf(() => buildActionURL(table, "web::load_one", [])).code).toBe("capture-arity");
  expect(issueOf(() => buildActionURL(table, "web::load_one", ["a", "b"])).code).toBe(
    "capture-arity",
  );
  expect(issueOf(() => buildActionURL(table, "web::load_one", "acme" as never)).code).toBe(
    "capture-arity",
  );
  expect(issueOf(() => buildActionURL(table, "web::load_one", [42])).code).toBe("capture-type");
  expect(issueOf(() => buildActionURL(table, "web::load_line", ["x", 3])).code).toBe(
    "capture-type",
  );
  expect(issueOf(() => buildActionURL(table, "web::load_line", ["x", "3"])).code).toBe(
    "capture-type",
  );
  expect(issueOf(() => buildActionURL(table, "web::load_line", [7n, 3n])).code).toBe(
    "capture-type",
  );
  const badValues: [string, unknown][] = [
    ["empty", ""],
    ["separator", "a/b"],
    ["dot", "."],
    ["dotdot", ".."],
    ["control", "a\u0000b"],
    ["backslash", "a\\b"],
    ["lone surrogate", "\ud800"],
    ["too large", 2n ** 63n],
    ["too small", -(2n ** 63n) - 1n],
  ];
  expect(badValues.length).toBe(9);
  for (const [name, value] of badValues) {
    const values: unknown[] = typeof value === "bigint" ? ["x", value] : [value];
    const identity = typeof value === "bigint" ? "web::load_line" : "web::load_one";
    const issue = issueOf(() => buildActionURL(table, identity, values));
    expect(`${name}: ${issue.code}`).toBe(`${name}: capture-value`);
  }
});

function matched(
  table: ActionRouteTable,
  method: string,
  pathname: string,
): { identity: string; captures: readonly ActionRouteCapture[] } {
  const match = matchActionRoute(table, method, pathname);
  expect(match.kind).toBe("match");
  if (match.kind !== "match") throw new Error(`no match for ${method} ${pathname}`);
  expect(Object.isFrozen(match.captures)).toBe(true);
  return match;
}

test("dispatches method-first with static priority", () => {
  const table = compileActionRoutes([...entries]);
  expect(matched(table, "GET", "/invoices/new").identity).toBe("web::load_new");
  const one = matched(table, "GET", "/invoices/42");
  expect(one.identity).toBe("web::load_one");
  expect(one.captures).toEqual([{ name: "invoice_id", value: "42" }]);
  // An encoded static spelling still resolves to the static route: matching
  // compares decoded segments, so the static wins over the capture.
  expect(matched(table, "GET", "/invoices/%6eew").identity).toBe("web::load_new");
  expect(matched(table, "GET", "/invoices/n%65w").identity).toBe("web::load_new");
  const encoded = matched(table, "GET", "/invoices/%41dmin");
  expect(encoded.captures).toEqual([{ name: "invoice_id", value: "Admin" }]);
  const spaced = matched(table, "GET", "/invoices/a%20b");
  expect(spaced.captures).toEqual([{ name: "invoice_id", value: "a b" }]);
  // Static priority applies within one method only.
  const save = matched(table, "GET", "/invoices/save");
  expect(save.identity).toBe("web::load_one");
  expect(save.captures).toEqual([{ name: "invoice_id", value: "save" }]);
  const created = matched(table, "POST", "/invoices/new");
  expect(created.identity).toBe("web::save_one");
  expect(created.captures).toEqual([{ name: "invoice_id", value: "new" }]);
  expect(matched(table, "POST", "/invoices/save").identity).toBe("web::save_wire");
  expect(matched(table, "POST", "/invoices/42").identity).toBe("web::save_one");
});

test("classifies unknown paths and methods as 404 and 405", () => {
  const table = compileActionRoutes([...entries]);
  for (const pathname of [
    "/nope",
    "/",
    "",
    "/invoices",
    "/invoices/42/",
    "/INVOICES/42",
    "/invoices/42/lines",
    "/invoices/9/lines/",
    "/invoices/a/b/c/d",
  ]) {
    expect(matchActionRoute(table, "GET", pathname)).toEqual({ kind: "not-found" });
  }
  const put = matchActionRoute(table, "PUT", "/invoices/42");
  expect(put).toEqual({ kind: "method-not-allowed", allow: "GET, POST" });
  expect(matchActionRoute(table, "DELETE", "/invoices/save")).toEqual({
    kind: "method-not-allowed",
    allow: "GET, POST",
  });
  expect(matchActionRoute(table, "HEAD", "/invoices/new")).toEqual({
    kind: "method-not-allowed",
    allow: "GET, POST",
  });
  expect(matchActionRoute(table, "PUT", "/invoices/9/lines/3")).toEqual({
    kind: "method-not-allowed",
    allow: "GET",
  });
  expect(matchActionRoute(table, "get", "/invoices/42")).toEqual({
    kind: "method-not-allowed",
    allow: "GET, POST",
  });
  expect(matchActionRoute(table, "PUT", "/nope")).toEqual({ kind: "not-found" });
});

test("types int captures as strict canonical int64", () => {
  const table = compileActionRoutes([...entries]);
  const line = matched(table, "GET", "/invoices/9/lines/3");
  expect(line.identity).toBe("web::load_line");
  expect(line.captures).toEqual([
    { name: "invoice_id", value: "9" },
    { name: "line", value: 3n },
  ]);
  expect(matched(table, "GET", "/invoices/9/lines/0").captures[1]).toEqual({
    name: "line",
    value: 0n,
  });
  expect(matched(table, "GET", "/invoices/9/lines/-42").captures[1]).toEqual({
    name: "line",
    value: -42n,
  });
  expect(matched(table, "GET", "/invoices/9/lines/9223372036854775807").captures[1]).toEqual({
    name: "line",
    value: 9223372036854775807n,
  });
  expect(matched(table, "GET", "/invoices/9/lines/-9223372036854775808").captures[1]).toEqual({
    name: "line",
    value: -(2n ** 63n),
  });
  for (const line of [
    "007",
    "+3",
    "-0",
    "3.0",
    "0x3",
    "3 ",
    " 3",
    "4_2",
    "9223372036854775808",
    "-9223372036854775809",
    "99999999999999999999999",
    "three",
  ]) {
    expect(matchActionRoute(table, "GET", `/invoices/9/lines/${line}`)).toEqual({
      kind: "bad-request",
    });
  }
});

test("rejects malformed targets before consulting routes", () => {
  const table = compileActionRoutes([...entries]);
  for (const pathname of [
    "invoices/42",
    "/invoices/%ZZ",
    "/invoices/%",
    "/invoices/%A",
    "/invoices/%c3%28",
    "/invoices/%ed%a0%80",
    "/invoices/a%2Fb",
    "/invoices/a%2fb",
    "/invoices/%2F",
    "/invoices/%00",
    "/invoices/a%00b",
    "/invoices/%5c",
    "/invoices/%5C",
    "/invoices/%2e",
    "/invoices/%2E%2E",
    "/invoices/%2e./x",
    "/%2e%2e",
  ]) {
    expect(`${pathname}: ${matchActionRoute(table, "GET", pathname).kind}`).toBe(
      `${pathname}: bad-request`,
    );
  }
  expect(matchActionRoute(table, "GET", "/%ZZ").kind).toBe("bad-request");
  expect(matchActionRoute(table, "PUT", "/invoices/%ZZ").kind).toBe("bad-request");
  expect(matchActionRoute(table, "GET", undefined as never).kind).toBe("bad-request");
});

type Call = { identity: string; captures: readonly ActionRouteCapture[]; path: string };

function echoInvoke(calls: Call[]): ActionInvoke {
  return async (identity, captures, request) => {
    const path = value(await requests.path(request));
    calls.push({ identity, captures, path });
    const echo = `${identity}|${captures.map((capture) => `${capture.name}=${String(capture.value)}`).join(",")}`;
    return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), echo);
  };
}

function actionServer(
  calls: Call[],
  assets?: { serve(native: Request): Promise<Response | undefined> },
) {
  return createServer(
    domain,
    {
      invalidConfig: id("can.std.http@1::invalid_server_config"),
      bindFailed: id("can.std.http@1::bind_failed"),
      shutdownFailed: id("can.std.http@1::shutdown_failed"),
    },
    assets,
    { table: compileActionRoutes([...entries]), invoke: echoInvoke(calls) },
  );
}

function expectPolicy(response: Response): void {
  expect(response.headers.get("content-security-policy")).toContain("script-src 'self'");
  expect(response.headers.get("content-security-policy")).not.toContain("unsafe-eval");
  expect(response.headers.get("x-content-type-options")).toBe("nosniff");
}

async function textResult(body: string): Promise<Completion<unknown>> {
  return responses.text(value(await responses.ok()), value(await responses.emptyHeaders()), body);
}

test("serves actions over native HTTP through the server lifecycle", async () => {
  const calls: Call[] = [];
  const server = actionServer(calls);
  const owned = await runOwnedRoot(async () => {
    const config = value(await server.makeConfig("127.0.0.1", 18511n, 1048576n, 5000n));
    const legacy = value(await router.get("/legacy/x", async () => textResult("legacy")));
    const routes = value(await router.make([legacy]));
    const token = value(await server.start(config, routes));
    const base = "http://127.0.0.1:18511";
    const fresh = await fetch(base + "/invoices/new");
    expect(fresh.status).toBe(200);
    expect(await fresh.text()).toBe("web::load_new|");
    expectPolicy(fresh);
    const one = await fetch(base + "/invoices/42");
    expect(one.status).toBe(200);
    expect(await one.text()).toBe("web::load_one|invoice_id=42");
    // Bun fires the dynamic route for an encoded static spelling; the
    // canonical re-dispatch still serves the static action.
    const encoded = await fetch(base + "/invoices/%6eew");
    expect(encoded.status).toBe(200);
    expect(await encoded.text()).toBe("web::load_new|");
    const spaced = await fetch(base + "/invoices/a%20b");
    expect(spaced.status).toBe(200);
    expect(await spaced.text()).toBe("web::load_one|invoice_id=a b");
    const line = await fetch(base + "/invoices/9/lines/3");
    expect(line.status).toBe(200);
    expect(await line.text()).toBe("web::load_line|invoice_id=9,line=3");
    const saved = await fetch(base + "/invoices/save", { method: "POST", body: "hi" });
    expect(saved.status).toBe(200);
    expect(await saved.text()).toBe("web::save_wire|");
    const created = await fetch(base + "/invoices/42", { method: "POST" });
    expect(created.status).toBe(200);
    expect(await created.text()).toBe("web::save_one|invoice_id=42");
    const query = await fetch(base + "/invoices/42?x=1&y=2");
    expect(query.status).toBe(200);
    expect(await query.text()).toBe("web::load_one|invoice_id=42");
    expect(calls.length).toBe(8);
    expect(calls.map((call) => call.path)).toEqual([
      "/invoices/new",
      "/invoices/42",
      "/invoices/new",
      "/invoices/a b",
      "/invoices/9/lines/3",
      "/invoices/save",
      "/invoices/42",
      "/invoices/42",
    ]);
    const wrong = await fetch(base + "/invoices/42", { method: "PUT" });
    expect(wrong.status).toBe(405);
    expect(wrong.headers.get("allow")).toBe("GET, POST");
    expect(await wrong.text()).toBe("Method Not Allowed");
    expectPolicy(wrong);
    const missing = await fetch(base + "/nope");
    expect(missing.status).toBe(404);
    expect(await missing.text()).toBe("Not Found");
    expectPolicy(missing);
    const slashed = await fetch(base + "/invoices/42/");
    expect(slashed.status).toBe(404);
    const malformed = await fetch(base + "/invoices/%ZZ");
    expect(malformed.status).toBe(400);
    expect(await malformed.text()).toBe("Bad Request");
    expectPolicy(malformed);
    const separator = await fetch(base + "/invoices/a%2Fb");
    expect(separator.status).toBe(400);
    const leading = await fetch(base + "/invoices/9/lines/007");
    expect(leading.status).toBe(400);
    const head = await fetch(base + "/invoices/new", { method: "HEAD" });
    expect(head.status).toBe(405);
    expect(head.headers.get("allow")).toBe("GET, POST");
    const legacyHit = await fetch(base + "/legacy/x");
    expect(legacyHit.status).toBe(200);
    expect(await legacyHit.text()).toBe("legacy");
    const legacyMissing = await fetch(base + "/legacy/nope");
    expect(legacyMissing.status).toBe(404);
    expect(calls.length).toBe(8);
    expect(resourceStatus(token)).toMatchObject({ kind: "server", state: "open", leases: 0 });
    expect((await server.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("enforces the body limit before action entry", async () => {
  const calls: Call[] = [];
  const server = actionServer(calls);
  const owned = await runOwnedRoot(async () => {
    const config = value(await server.makeConfig("127.0.0.1", 18512n, 8n, 5000n));
    const routes = value(await router.make([]));
    const token = value(await server.start(config, routes));
    const tooLarge = await fetch("http://127.0.0.1:18512/invoices/save", {
      method: "POST",
      body: "nine-byte",
    });
    expect(tooLarge.status).toBe(413);
    expect(await tooLarge.text()).toBe("Payload Too Large");
    expect(calls.length).toBe(0);
    const exact = await fetch("http://127.0.0.1:18512/invoices/save", {
      method: "POST",
      body: "eight123",
    });
    expect(exact.status).toBe(200);
    expect(await exact.text()).toBe("web::save_wire|");
    expect(calls.length).toBe(1);
    expect((await server.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

test("serves reserved assets ahead of native action callbacks", async () => {
  const calls: Call[] = [];
  const server = actionServer(calls, {
    serve: async (native: Request): Promise<Response | undefined> =>
      new URL(native.url).pathname === "/invoices/new" ? new Response("asset") : undefined,
  });
  const owned = await runOwnedRoot(async () => {
    const config = value(await server.makeConfig("127.0.0.1", 18513n, 1048576n, 5000n));
    const routes = value(await router.make([]));
    const token = value(await server.start(config, routes));
    const reserved = await fetch("http://127.0.0.1:18513/invoices/new");
    expect(reserved.status).toBe(200);
    expect(await reserved.text()).toBe("asset");
    expectPolicy(reserved);
    expect(calls.length).toBe(0);
    const passed = await fetch("http://127.0.0.1:18513/invoices/42");
    expect(passed.status).toBe(200);
    expect(await passed.text()).toBe("web::load_one|invoice_id=42");
    expect(calls.length).toBe(1);
    expect((await server.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});

function rawRequest(
  port: number,
  target: string,
  method = "GET",
): Promise<{ status: number; body: string }> {
  return new Promise((resolve, reject) => {
    const sock = connect(port, "127.0.0.1", () => {
      sock.write(`${method} ${target} HTTP/1.1\r\nHost: x\r\nConnection: close\r\n\r\n`);
    });
    let data = "";
    sock.on("data", (chunk) => {
      data += chunk.toString("latin1");
    });
    sock.on("close", () => {
      const head = data.split("\r\n\r\n")[0] ?? "";
      const body = data.split("\r\n\r\n").slice(1).join("\r\n\r\n");
      resolve({ status: Number(head.split(" ")[1]), body });
    });
    sock.on("error", reject);
  });
}

test("dots, separators and malformed escapes never reach the handler", async () => {
  const calls: Call[] = [];
  const server = actionServer(calls);
  const owned = await runOwnedRoot(async () => {
    const config = value(await server.makeConfig("127.0.0.1", 18514n, 1048576n, 5000n));
    const routes = value(await router.make([]));
    const token = value(await server.start(config, routes));
    // Resolved dots carry the resolved path: the handler observes values,
    // never dot segments.
    const dotted = await rawRequest(18514, "/invoices/./42");
    expect(dotted.status).toBe(200);
    expect(dotted.body).toBe("web::load_one|invoice_id=42");
    const escaped = await rawRequest(18514, "/%2e%2e/invoices/42");
    expect(escaped.status).toBe(200);
    expect(escaped.body).toBe("web::load_one|invoice_id=42");
    const dotValue = await rawRequest(18514, "/invoices/%2e");
    expect(dotValue.status).toBe(404);
    expect(dotValue.body).toBe("Not Found");
    const dotEscape = await rawRequest(18514, "/invoices/%2e%2e/x");
    expect(dotEscape.status).toBe(404);
    const separator = await rawRequest(18514, "/invoices/a%2Fb");
    expect(separator.status).toBe(400);
    expect(separator.body).toBe("Bad Request");
    const malformed = await rawRequest(18514, "/invoices/%ZZ");
    expect(malformed.status).toBe(400);
    expect(malformed.body).toBe("Bad Request");
    const nul = await rawRequest(18514, "/invoices/%00");
    expect(nul.status).toBe(400);
    const badUtf8 = await rawRequest(18514, "/invoices/%c3%28");
    expect(badUtf8.status).toBe(400);
    expect(badUtf8.body).toBe("Bad Request");
    const leading = await rawRequest(18514, "/invoices/9/lines/007");
    expect(leading.status).toBe(400);
    expect(leading.body).toBe("Bad Request");
    const line = await rawRequest(18514, "/invoices/9/lines/3");
    expect(line.status).toBe(200);
    expect(line.body).toBe("web::load_line|invoice_id=9,line=3");
    expect(calls.length).toBe(3);
    expect(calls.flatMap((call) => call.captures.map((capture) => String(capture.value)))).toEqual([
      "42",
      "42",
      "9",
      "3",
    ]);
    expect((await server.stop(token)).kind).toBe("ok");
    return success(undefined);
  });
  expect(owned.cleanupFailed).toBe(false);
  expect(owned.completion.kind).toBe("ok");
});
