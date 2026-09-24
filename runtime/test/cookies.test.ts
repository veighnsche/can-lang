import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createCookies, isCookieValue, COOKIE_KIND } from "../platform/cookies.ts";
import { createCSRF } from "../platform/csrf.ts";
import { createResponses, nativeResponse } from "../platform/http.ts";
import { value, success } from "../completion.ts";
import { record, array, dataArray, dataProperty, recordIdentity } from "../data.ts";
import { runOwnedRoot } from "../owner.ts";
import { assertionContext, closeContext } from "../assert/context.ts";
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
  [
    "http::invalid_request",
    "codec::invalid_data",
    "cookie::invalid_cookie",
    "csrf::invalid_config",
  ].includes(e.name),
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
const failOf = (result: Completion) => {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain");
  return domainFailureDiagnostics(result.value);
};
function check(result: Completion, name: string, payload: object) {
  const d = failOf(result);
  expect(d.declaration.name).toBe(name);
  expect(d.payload).toMatchObject(payload);
}
const COLLECTION = "can.std.cookie@1::collection",
  PAIR = "can.std.cookie@1::pair",
  SOME = "can.std.option@1::some",
  NONE = "can.std.option@1::none",
  STRICT = "can.std.cookie@1::strict",
  LAX = "can.std.cookie@1::lax",
  NOSITE = "can.std.cookie@1::none";
const cookies = createCookies(domain, {
  invalid: identity("can.std.cookie@1::invalid_cookie"),
  collection: COLLECTION,
  pair: PAIR,
  some: SOME,
  none: NONE,
  samesiteStrict: STRICT,
  samesiteLax: LAX,
  samesiteNone: NOSITE,
});
const csrf = createCSRF(domain, { invalid: identity("can.std.csrf@1::invalid_config") });
const responses = createResponses(domain, {
  invalid: identity("can.std.http@1::invalid_request"),
  invalidData: identity("can.std.codec@1::invalid_data"),
  close: "can.std.stream@1::close_failed",
  writeFailed: "can.std.stream@1::write_failed",
  limit: "can.std.http@1::body_limit",
});
const some = (v: unknown) => record(SOME, [["value", v]]);
const none = () => record(NONE, []);
const attrs = (
  path: string,
  domainName: unknown,
  secure: boolean,
  httpOnly: boolean,
  site: unknown,
  maxAge: unknown,
  expiresMs: unknown,
) =>
  record("can.std.cookie@1::attributes", [
    ["path", path],
    ["domain", domainName],
    ["secure", secure],
    ["http_only", httpOnly],
    ["same_site", site],
    ["max_age", maxAge],
    ["expires_ms", expiresMs],
  ]);
const lax = () => record(LAX, []);
const pairsOf = (collection: unknown) =>
  dataArray(dataProperty(collection, "pairs")).map(
    (entry) => [dataProperty(entry, "name"), dataProperty(entry, "value")] as const,
  );
async function owned(body: () => Promise<void>): Promise<void> {
  const out = await runOwnedRoot(async () => {
    await body();
    return success(undefined);
  });
  expect(out.completion.kind).toBe("ok");
  expect(out.cleanupFailed).toBe(false);
}
test("parse keeps ordered pairs with first-wins lookup", async () => {
  await owned(async () => {
    const parsed = value(await cookies.parse("a=1; b=2; a=3"));
    expect(pairsOf(parsed)).toEqual([
      ["a", "1"],
      ["b", "2"],
      ["a", "3"],
    ]);
    expect(dataProperty(value(await cookies.get(parsed, "a")), "value")).toBe("1");
    expect(dataProperty(value(await cookies.get(parsed, "b")), "value")).toBe("2");
    const missing = value(await cookies.get(parsed, "zzz"));
    expect(recordIdentity(missing)).toBe(NONE);
  });
});
test("parse tolerates malformed segments and decodes values", async () => {
  await owned(async () => {
    expect(pairsOf(value(await cookies.parse("")))).toEqual([]);
    expect(pairsOf(value(await cookies.parse("  a  =  1 ;b=2 ")))).toEqual([
      ["a", "1"],
      ["b", "2"],
    ]);
    expect(pairsOf(value(await cookies.parse("a=1; bare; b=2")))).toEqual([
      ["a", "1"],
      ["b", "2"],
    ]);
    expect(pairsOf(value(await cookies.parse("=x; b=2")))).toEqual([["b", "2"]]);
    expect(pairsOf(value(await cookies.parse("a=; b=2")))).toEqual([
      ["a", ""],
      ["b", "2"],
    ]);
    expect(pairsOf(value(await cookies.parse("a=b=c")))).toEqual([["a", "b=c"]]);
    expect(pairsOf(value(await cookies.parse("a=%20%3B")))).toEqual([["a", " ;"]]);
    expect(pairsOf(value(await cookies.parse("A=1; a=2")))).toEqual([
      ["A", "1"],
      ["a", "2"],
    ]);
  });
});
test("make and serialize render full attributes", async () => {
  await owned(async () => {
    const jar = value(
      await cookies.make(
        "s",
        "v",
        attrs("/admin", some("example.com"), true, true, record(STRICT, []), some(3600n), none()),
      ),
    );
    expect(value(await cookies.serialize(jar))).toBe(
      "s=v; Domain=example.com; Path=/admin; Max-Age=3600; Secure; HttpOnly; SameSite=Strict",
    );
    const dated = value(
      await cookies.make(
        "e",
        "v",
        attrs("/", none(), false, false, lax(), none(), some(1893456000000n)),
      ),
    );
    expect(value(await cookies.serialize(dated))).toBe(
      "e=v; Path=/; Expires=Tue, 01 Jan 2030 00:00:00 GMT; SameSite=Lax",
    );
  });
});
test("serialize emits native defaults", async () => {
  await owned(async () => {
    const jar = value(
      await cookies.make("a", "1", attrs("/", none(), false, false, lax(), none(), none())),
    );
    expect(value(await cookies.serialize(jar))).toBe("a=1; Path=/; SameSite=Lax");
  });
});
test("make rejects invalid names paths and domains per field", async () => {
  await owned(async () => {
    const good = attrs("/", none(), false, false, lax(), none(), none());
    for (const name of ["", "a\r\nX", "a;b", "a b", "a=b"])
      check(await cookies.make(name, "1", good), "cookie::invalid_cookie", { reason: "name" });
    check(
      await cookies.make("a", "1", attrs("/\r\nX", none(), false, false, lax(), none(), none())),
      "cookie::invalid_cookie",
      { reason: "path" },
    );
    check(
      await cookies.make(
        "a",
        "1",
        attrs("/", some("ex ample.com"), false, false, lax(), none(), none()),
      ),
      "cookie::invalid_cookie",
      { reason: "domain" },
    );
    check(
      await cookies.make(
        "a",
        "1",
        attrs("/", none(), false, false, lax(), none(), some(8640000000000001n)),
      ),
      "cookie::invalid_cookie",
      { reason: "expires" },
    );
    check(
      await cookies.make(
        "a",
        "1",
        attrs("/", none(), false, false, lax(), some(9007199254740993n), none()),
      ),
      "cookie::invalid_cookie",
      { reason: "max_age" },
    );
  });
});
test("values encode while prefixes pass through unenforced", async () => {
  await owned(async () => {
    const encoded = value(
      await cookies.make(
        "a",
        "1\r\nX: y;€",
        attrs("/", none(), false, false, lax(), none(), none()),
      ),
    );
    expect(value(await cookies.serialize(encoded))).toBe(
      "a=1%0D%0AX%3A%20y%3B%E2%82%AC; Path=/; SameSite=Lax",
    );
    const open = value(
      await cookies.make(
        "__Host-s",
        "v",
        attrs("/", some("example.com"), true, false, lax(), none(), none()),
      ),
    );
    expect(value(await cookies.serialize(open))).toBe(
      "__Host-s=v; Domain=example.com; Path=/; Secure; SameSite=Lax",
    );
    const insecure = value(
      await cookies.make(
        "a",
        "1",
        attrs("/", none(), false, false, record(NOSITE, []), none(), none()),
      ),
    );
    expect(value(await cookies.serialize(insecure))).toBe("a=1; Path=/; SameSite=None");
  });
});
test("expire builds scoped tombstones", async () => {
  await owned(async () => {
    const tomb = value(await cookies.remove("a", "/", none()));
    expect(value(await cookies.serialize(tomb))).toBe(
      "a=; Path=/; Expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Lax",
    );
    const scoped = value(await cookies.remove("b", "/admin", some("example.com")));
    expect(value(await cookies.serialize(scoped))).toBe(
      "b=; Domain=example.com; Path=/admin; Expires=Thu, 01 Jan 1970 00:00:00 GMT; SameSite=Lax",
    );
    check(await cookies.remove("a;b", "/", none()), "cookie::invalid_cookie", { reason: "name" });
  });
});
test("cookie handles reject foreign values", async () => {
  await owned(async () => {
    const jar = value(
      await cookies.make("a", "1", attrs("/", none(), false, false, lax(), none(), none())),
    );
    expect(isCookieValue(COOKIE_KIND, jar)).toBe(true);
    expect(isCookieValue(COOKIE_KIND, record("can.std.cookie@1::cookie", []))).toBe(false);
    expect(isCookieValue("other", jar)).toBe(false);
    let thrown: unknown;
    try {
      await cookies.serialize(record("can.std.cookie@1::cookie", []));
    } catch (cause) {
      thrown = cause;
    }
    expect(thrown).toBeInstanceOf(TypeError);
  });
});
test("repeated set-cookie survives headers into native responses", async () => {
  await owned(async () => {
    const first = value(
      await cookies.serialize(
        value(
          await cookies.make("a", "1", attrs("/", none(), false, false, lax(), none(), none())),
        ),
      ),
    );
    const second = value(
      await cookies.serialize(
        value(
          await cookies.make(
            "e",
            "v",
            attrs("/", none(), false, false, lax(), none(), some(1893456000000n)),
          ),
        ),
      ),
    );
    const headers = value(
      await responses.makeHeaders(
        array([
          record("can.std.http@1::header", [
            ["name", "set-cookie"],
            ["value", first as string],
          ]),
          record("can.std.http@1::header", [
            ["name", "x-t"],
            ["value", "1"],
          ]),
          record("can.std.http@1::header", [
            ["name", "Set-Cookie"],
            ["value", second as string],
          ]),
        ]),
      ),
    );
    const response = value(await responses.text(value(await responses.ok()), headers, "ok"));
    const native = nativeResponse(response);
    expect(native.headers.getSetCookie()).toEqual([first, second]);
    expect(native.headers.get("x-t")).toBe("1");
  });
});
test("generate mints distinct session-bound tokens", async () => {
  await owned(async () => {
    const a = value(await csrf.generate("s3cret", "A", 60000n)) as string;
    const b = value(await csrf.generate("s3cret", "A", 60000n)) as string;
    expect(typeof a).toBe("string");
    expect(a.length).toBe(86);
    expect(a).not.toBe(b);
    expect(value(await csrf.verify("s3cret", "A", a, 60000n))).toBe(true);
  });
});
test("verify answers false for token faults", async () => {
  await owned(async () => {
    const token = value(await csrf.generate("s3cret", "A", 60000n)) as string;
    expect(value(await csrf.verify("s3cret", "B", token, 60000n))).toBe(false);
    expect(value(await csrf.verify("other", "A", token, 60000n))).toBe(false);
    expect(value(await csrf.verify("s3cret", "A", "not-a-token", 60000n))).toBe(false);
    expect(value(await csrf.verify("s3cret", "A", token.slice(0, 8), 60000n))).toBe(false);
    expect(
      value(
        await csrf.verify(
          "s3cret",
          "A",
          token.slice(0, -2) + (token.endsWith("AA") ? "BB" : "AA"),
          60000n,
        ),
      ),
    ).toBe(false);
  });
});
test("invalid config fails without secret or token material", async () => {
  await owned(async () => {
    for (const [secret, session, duration] of [
      ["", "A", 60000n],
      ["s3cret", "", 60000n],
    ] as const) {
      const failed = failOf(await csrf.generate(secret, session, duration));
      expect(failed.declaration.name).toBe("csrf::invalid_config");
      expect(Object.keys(failed.payload as object).sort()).toEqual(["reason"]);
    }
    check(await csrf.generate("s3cret", "A", -1n), "csrf::invalid_config", { reason: "duration" });
    check(await csrf.generate("s3cret", "A", 9007199254740992n), "csrf::invalid_config", {
      reason: "duration",
    });
    check(await csrf.verify("", "A", "t", 60000n), "csrf::invalid_config", { reason: "secret" });
    check(await csrf.verify("s3cret", "", "t", 60000n), "csrf::invalid_config", {
      reason: "session",
    });
    check(await csrf.verify("s3cret", "A", "t", -1n), "csrf::invalid_config", {
      reason: "duration",
    });
  });
});
test("expiry is absolute over max age", async () => {
  await owned(async () => {
    const fresh = value(await csrf.generate("s3cret", "A", 60000n)) as string;
    expect(value(await csrf.verify("s3cret", "A", fresh, 0n))).toBe(true);
    const tiny = value(await csrf.generate("s3cret", "A", 50n)) as string;
    expect(value(await csrf.verify("s3cret", "A", tiny, 60000n))).toBe(true);
    await new Promise((resolve) => setTimeout(resolve, 250));
    expect(value(await csrf.verify("s3cret", "A", tiny, 60000n))).toBe(false);
  });
});
test("generate denies the live assertion boundary", async () => {
  await owned(async () => {
    const context = assertionContext({
      package: "can.project.root/app",
      declaration: "can.project.root/app::probe",
      name: "sample",
    });
    try {
      let thrown: unknown;
      try {
        await csrf.generate("s3cret", "A", 60000n, context);
      } catch (cause) {
        thrown = cause;
      }
      expect(thrown).toBeDefined();
      expect(value(await csrf.verify("s3cret", "A", "bogus", 60000n, context))).toBe(false);
    } finally {
      closeContext(context);
    }
  });
});
