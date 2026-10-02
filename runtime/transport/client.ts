import { denyLiveBoundary, type AssertionContext } from "../assert/context.ts";
import { providerHTTP } from "../assert/provider.ts";
import { invoke, success, failure, type Completion } from "../completion.ts";
import { record, array, dataArray, dataProperty } from "../data.ts";
import { ownBytes, copyBytes, type Bytes } from "../bytes.ts";
import { createDomainRuntime } from "../domain.ts";
import type { FailureOrigin } from "../failure.ts";
import { transportProblem } from "./deadline.ts";
import { performRequest, type NativeRequest, type ResponseMetadata } from "./fetch.ts";
import type { Connection } from "./request.ts";

const origin: FailureOrigin = Object.freeze({
  source: "can:http-client",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

// Only identities the op contract references: every slot must resolve to
// a program type. Unreferenced slots (timeout, credential, status) would
// arrive empty and fail closed at creation, so fault mapping lives here,
// over thrown transport problems, instead of reusing the shared wrapper.
export type HTTPClientTypes = Readonly<{
  invalid: string;
  transport: string;
  limit: string;
  header: string;
  bytesResponse: string;
}>;

const methods = new Set(["GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"]);

export function createHTTPClient(
  domain: ReturnType<typeof createDomainRuntime>,
  types: HTTPClientTypes,
) {
  const operation = "fetch_bytes";
  const native = { boundary: "native", operation } as const;
  function fail(identity: string, fields: [string, unknown][]): Completion<never> {
    return failure(domain.create(identity, record(identity, fields), origin, undefined, native));
  }
  function invalid(reason: string): Completion<never> {
    return fail(types.invalid, [["reason", reason]]);
  }
  return Object.freeze({
    async fetchBytes(
      method: unknown,
      url: unknown,
      headers: unknown,
      body: unknown,
      timeoutMs: unknown,
      maxBodyBytes: unknown,
      context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      if (typeof method !== "string" || !methods.has(method)) return invalid("method");
      if (typeof url !== "string" || url === "") return invalid("url");
      let endpoint: string;
      const query: { name: string; value: string }[] = [];
      try {
        const parsed = new URL(url);
        // The shared transport takes the query apart from the endpoint;
        // split here so one op serves plain and parameterized URLs.
        // Repeats and order survive via URLSearchParams iteration.
        for (const [name, value] of parsed.searchParams) query.push({ name, value });
        parsed.search = "";
        endpoint = parsed.href;
      } catch {
        return invalid("url");
      }
      const entries: { name: string; value: string }[] = [];
      for (const item of dataArray(headers)) {
        const name = dataProperty(item, "name"),
          value = dataProperty(item, "value");
        if (typeof name !== "string" || typeof value !== "string") return invalid("header_value");
        entries.push({ name, value });
      }
      const timeout = timeoutMs as bigint,
        cap = maxBodyBytes as bigint;
      if (timeout < 1n || timeout > 2147483647n) return invalid("timeout_ms");
      if (cap < 1n || cap > 67108864n) return invalid("max_body_bytes");
      const payload = copyBytes(body as Bytes, origin);
      const hasBody = payload.byteLength > 0;
      if (hasBody && (method === "GET" || method === "HEAD")) return invalid("method_body");
      const connection: Connection = {
        endpoint,
        timeoutMilliseconds: Number(timeout),
        maxBodyBytes: Number(cap),
        headers: [],
      };
      const request: NativeRequest = {
        path: "",
        method: method as NativeRequest["method"],
        query,
        headers: entries,
        body: hasBody ? payload : undefined,
        bodyEncoding: hasBody ? "bytes" : undefined,
        envelope: true,
      };
      const exchange = providerHTTP(context, origin, operation, Number(cap));
      if (exchange === undefined) denyLiveBoundary(context, origin);
      return invoke(async () => {
        try {
          return await performRequest(
            connection,
            { ...request, exchange },
            () => undefined,
            (bytes: Uint8Array, metadata: ResponseMetadata) =>
              success(
                record(types.bytesResponse, [
                  ["status", BigInt(metadata.status)],
                  [
                    "headers",
                    array(
                      metadata.headers.map((header) =>
                        record(types.header, [
                          ["name", header.name],
                          ["value", header.value],
                        ]),
                      ),
                    ),
                  ],
                  ["body", ownBytes(bytes)],
                ]),
              ),
          );
        } catch (cause) {
          const problem = transportProblem(cause);
          if (!problem) throw cause;
          switch (problem.kind) {
            case "invalid":
              return invalid(problem.reason);
            case "transport":
              return fail(types.transport, [["phase", problem.phase]]);
            case "timeout":
              // Same no-commit-knowledge rule as the JSON action client:
              // expiry aborts the wire request under its own phase.
              return fail(types.transport, [["phase", "timeout"]]);
            case "limit":
              return fail(types.limit, [["limit", BigInt(problem.limit)]]);
            case "credential":
            case "status":
              // Unreachable by construction: no bearer environment is
              // configured and the envelope accepts every status. A
              // failure here is a defect, so it stays loud.
              throw cause;
          }
        }
      }, origin);
    },
  });
}
