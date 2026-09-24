import { describe, expect, test } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { value, type Completion } from "../completion.ts";
import { dataProperty, record, recordIdentity } from "../data.ts";
import { createURLs } from "../platform/url.ts";

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
const declaration = catalogue.errors.find((e) => e.name === "url::invalid_url")!;
const errorShape: FailureShape = {
  identity: identity("error", declaration.identity),
  kind: "error",
  declaration: declaration.identity,
  arguments: [],
  fields: [{ name: "reason", type: textShape.identity }],
  leaves: [],
  inputs: [],
  errors: [],
};
const domain = createDomainRuntime({
  declarations: [{ identity: declaration.identity, name: declaration.name, parameters: 0 }],
  shapes: [textShape, intShape, errorShape],
});
const api = createURLs(domain, {
  invalidURL: errorShape.identity,
  parts: "url::parts",
  pair: "url::query_pair",
});

function reason(completion: Completion<unknown>): string {
  expect(completion.kind).toBe("domain");
  if (completion.kind !== "domain") throw new Error("wrong outcome");
  const details = domainFailureDiagnostics(completion.value);
  expect(details.declaration.name).toBe("url::invalid_url");
  return (details.payload as Record<string, string>).reason;
}
const fields = (parts: unknown) => ({
  scheme: dataProperty(parts, "scheme"),
  host: dataProperty(parts, "host"),
  port: dataProperty(parts, "port"),
  path: dataProperty(parts, "path"),
  query: dataProperty(parts, "query"),
  fragment: dataProperty(parts, "fragment"),
});

describe("url parsing", () => {
  test("absolute urls project normalized immutable parts", async () => {
    const parts = value(await api.parseURL("HTTPS://Example.COM:8080/a/../b?q=1#f"));
    expect(recordIdentity(parts)).toBe("url::parts");
    expect(fields(parts)).toEqual({
      scheme: "https",
      host: "example.com",
      port: 8080n,
      path: "/b",
      query: "q=1",
      fragment: "f",
    });
    const bare = value(await api.parseURL("http://h"));
    expect(fields(bare)).toEqual({
      scheme: "http",
      host: "h",
      port: 0n,
      path: "/",
      query: "",
      fragment: "",
    });
    const deflated = value(await api.parseURL("https://h:443/x"));
    expect(dataProperty(deflated, "port")).toBe(0n);
  });
  test("other schemes and malformed text reject distinctly; userinfo never projects", async () => {
    expect(reason(await api.parseURL("ftp://h/x"))).toBe("scheme");
    expect(reason(await api.parseURL("notaurl"))).toBe("syntax");
    expect(reason(await api.parseURL("http://[::1"))).toBe("syntax");
    expect(reason(await api.parseURL("http://"))).toBe("syntax");
    const parts = value(await api.parseURL("https://user:pass@h/p"));
    expect(dataProperty(parts, "host")).toBe("h");
    expect(value(await api.urlToString(parts))).toBe("https://h/p");
  });
  test("relative references resolve against absolute bases", async () => {
    const parts = value(await api.resolveURL("https://base.example/dir/", "/x?q=1#f"));
    expect(value(await api.urlToString(parts))).toBe("https://base.example/x?q=1#f");
    const absolute = value(await api.resolveURL("https://base.example/", "http://other.example/y"));
    expect(dataProperty(absolute, "host")).toBe("other.example");
    expect(reason(await api.resolveURL("notaurl", "/x"))).toBe("syntax");
    expect(reason(await api.resolveURL("https://base.example/", "ftp://h/x"))).toBe("syntax");
  });
});

describe("url queries", () => {
  test("repeated keys keep order with form decoding and unicode escapes", async () => {
    const parts = value(await api.parseURL("https://h/p?a=1&a=2&b=+x&u=%E2%82%AC&e&f="));
    expect(value(await api.queryAll(parts, "a"))).toEqual(["1", "2"]);
    expect(value(await api.queryAll(parts, "b"))).toEqual([" x"]);
    expect(value(await api.queryAll(parts, "u"))).toEqual(["€"]);
    expect(value(await api.queryAll(parts, "e"))).toEqual([""]);
    expect(value(await api.queryAll(parts, "missing"))).toEqual([]);
    const pairs = value(await api.queryPairs(parts)) as unknown[];
    expect(pairs.map((p) => [dataProperty(p, "name"), dataProperty(p, "value")])).toEqual([
      ["a", "1"],
      ["a", "2"],
      ["b", " x"],
      ["u", "€"],
      ["e", ""],
      ["f", ""],
    ]);
  });
  test("rebuilt queries preserve order and duplicates with form encoding", async () => {
    const parts = value(await api.parseURL("https://h/p?old=0#keep"));
    const pair = (name: string, val: string) =>
      record("url::query_pair", [
        ["name", name],
        ["value", val],
      ]);
    const rebuilt = value(
      await api.withQuery(parts, [pair("a", "1"), pair("a", "2"), pair("s", "x y")]),
    );
    expect(recordIdentity(rebuilt)).toBe("url::parts");
    expect(value(await api.urlToString(rebuilt))).toBe("https://h/p?a=1&a=2&s=x+y#keep");
  });
});
