// URL parsing and query handling over WHATWG URL/URLSearchParams. Only
// http and https are admitted: anything else is invalid_url, and userinfo
// never enters a projection. Parsed URLs are immutable records, never
// live native objects: scheme, host, port (0 when absent, including
// default ports), path, query, and fragment. Query work uses form
// decoding (application/x-www-form-urlencoded: "+" reads as a space),
// which differs deliberately from path percent-decoding; repeated keys
// keep document order, and "?a" and "?a=" both read as ("a", "").
import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { record, array, dataProperty } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
const origin = Object.freeze({
  source: "can:url",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type Parts = {
  scheme: string;
  host: string;
  port: bigint;
  path: string;
  query: string;
  fragment: string;
};
export function createURLs(
  domain: ReturnType<typeof createDomainRuntime>,
  ids: { invalidURL: string; parts: string; pair: string },
) {
  const reject = (reason: string) =>
    failure(domain.create(ids.invalidURL, record(ids.invalidURL, [["reason", reason]]), origin));
  const project = (native: URL) => {
    const port = native.port === "" ? 0n : BigInt(native.port);
    return record(ids.parts, [
      ["scheme", native.protocol.slice(0, -1)],
      ["host", native.hostname],
      ["port", port],
      ["path", native.pathname],
      ["query", native.search === "" ? "" : native.search.slice(1)],
      ["fragment", native.hash === "" ? "" : native.hash.slice(1)],
    ]);
  };
  const read = (value: unknown): Parts => {
    const field = (name: string): unknown => {
      try {
        return dataProperty(value, name);
      } catch {
        throw new TypeError("invalid compiler url parts");
      }
    };
    const scheme = field("scheme"),
      host = field("host"),
      port = field("port"),
      path = field("path"),
      query = field("query"),
      fragment = field("fragment");
    if (
      typeof scheme !== "string" ||
      typeof host !== "string" ||
      typeof port !== "bigint" ||
      typeof path !== "string" ||
      typeof query !== "string" ||
      typeof fragment !== "string"
    )
      throw new TypeError("invalid compiler url parts");
    return { scheme, host, port, path, query, fragment };
  };
  const parse = (text: string, base?: string): URL | undefined => {
    try {
      const native = base === undefined ? new URL(text) : new URL(text, base);
      if (native.protocol !== "http:" && native.protocol !== "https:") return undefined;
      return native;
    } catch {
      return undefined;
    }
  };
  const serialize = (parts: Parts, query: string): URL | undefined =>
    parse(
      `${parts.scheme}://${parts.host}${parts.port === 0n ? "" : ":" + parts.port}${parts.path}${query === "" ? "" : "?" + query}${parts.fragment === "" ? "" : "#" + parts.fragment}`,
    );
  return Object.freeze({
    async parseURL(text: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      if (typeof text !== "string") throw new TypeError("invalid compiler url input");
      const native = parse(text);
      if (native === undefined)
        return reject(
          /^[a-zA-Z][a-zA-Z0-9+.-]*:/.test(text) && !/^https?:/i.test(text) ? "scheme" : "syntax",
        );
      return success(project(native));
    },
    async resolveURL(
      base: unknown,
      input: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      if (typeof base !== "string" || typeof input !== "string")
        throw new TypeError("invalid compiler url input");
      const anchor = parse(base);
      if (anchor === undefined) return reject("syntax");
      const native = parse(input, anchor.href);
      if (native === undefined) return reject("syntax");
      return success(project(native));
    },
    async urlToString(parts: unknown, _context?: AssertionContext): Promise<Completion<string>> {
      const fields = read(parts);
      const native = serialize(fields, fields.query);
      if (native === undefined) return reject("syntax");
      return success(native.href);
    },
    async queryAll(
      parts: unknown,
      name: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<readonly string[]>> {
      const fields = read(parts);
      if (typeof name !== "string") throw new TypeError("invalid compiler url input");
      return success(array(new URLSearchParams(fields.query).getAll(name)));
    },
    async queryPairs(
      parts: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<readonly unknown[]>> {
      const fields = read(parts);
      return success(
        array(
          [...new URLSearchParams(fields.query).entries()].map(([name, value]) =>
            record(ids.pair, [
              ["name", name],
              ["value", value],
            ]),
          ),
        ),
      );
    },
    async withQuery(
      parts: unknown,
      pairs: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const fields = read(parts);
      if (!Array.isArray(pairs)) throw new TypeError("invalid compiler url input");
      const params = new URLSearchParams();
      for (const pair of pairs) {
        let name: unknown, value: unknown;
        try {
          name = dataProperty(pair, "name");
          value = dataProperty(pair, "value");
        } catch {
          throw new TypeError("invalid compiler url input");
        }
        if (typeof name !== "string" || typeof value !== "string")
          throw new TypeError("invalid compiler url input");
        params.append(name, value);
      }
      const native = serialize(fields, params.toString());
      if (native === undefined) return reject("syntax");
      return success(project(native));
    },
  });
}
