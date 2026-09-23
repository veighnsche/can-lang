import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { isStandardFailure, standardFailureKind, standardFailureDiagnostics } from "../failure.ts";
import {
  createS3,
  isS3Value,
  S3_CLIENT_KIND,
  S3_UPLOAD_KIND,
  S3_PRESIGNED_KIND,
  S3_CONTINUATION_KIND,
} from "../platform/s3.ts";
import { createStreamReads, openByteCell, openLineCell } from "../transport/stream/readable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { createDateTimes } from "../platform/datetime.ts";
import { value, success, failure } from "../completion.ts";
import { record, dataArray, dataProperty, recordIdentity } from "../data.ts";
import { ownBytes, copyBytes } from "../bytes.ts";
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
  [1305, 1316, 1318, 1319, 1329, 1330, 1331, 1332, 1343, 1344, 1345, 1346, 1347, 1348].includes(
    e.id,
  ),
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
function check(result: Completion, id: number, payload: object) {
  const d = failOf(result);
  expect(d.declaration.id).toBe(id);
  expect(d.payload).toMatchObject(payload);
}
function standardOf(thrown: unknown) {
  expect(isStandardFailure(thrown)).toBe(true);
  return standardFailureKind(thrown as never);
}
const METADATA = "can.std.s3@1::metadata",
  ENTRY = "can.std.s3@1::entry",
  PAGE = "can.std.s3@1::page",
  INFO = "can.std.s3@1::presigned_info";
const METHOD_GET = "can.std.s3@1::method_get",
  METHOD_PUT = "can.std.s3@1::method_put",
  METHOD_DELETE = "can.std.s3@1::method_delete",
  METHOD_HEAD = "can.std.s3@1::method_head";
const SOME_TEXT = "can.std.option@1::some<str>",
  SOME_INT = "can.std.option@1::some<int>",
  SOME_CONT = "can.std.option@1::some<s3::continuation>",
  NONE = "can.std.option@1::none";
const s3 = createS3(domain, {
  invalid: identity("can.std.s3@1::invalid_config"),
  missing: identity("can.std.s3@1::missing_key"),
  denied: identity("can.std.s3@1::access_denied"),
  service: identity("can.std.s3@1::service_error"),
  closed: identity("can.std.s3@1::upload_closed"),
  overLimit: identity("can.std.s3@1::over_limit"),
  metadata: METADATA,
  entry: ENTRY,
  page: PAGE,
  info: INFO,
  methodGet: METHOD_GET,
  methodPut: METHOD_PUT,
  methodDelete: METHOD_DELETE,
  methodHead: METHOD_HEAD,
  someText: SOME_TEXT,
  someInt: SOME_INT,
  someContinuation: SOME_CONT,
  none: NONE,
  readFailed: identity("can.std.stream@1::read_failed"),
  cancelled: identity("can.std.stream@1::cancelled"),
  closeFailed: identity("can.std.stream@1::close_failed"),
});
const reads = createStreamReads(domain, {
  readFailed: identity("can.std.stream@1::read_failed"),
  cancelled: identity("can.std.stream@1::cancelled"),
  closeFailed: identity("can.std.stream@1::close_failed"),
  limitExceeded: identity("can.std.files@1::limit_exceeded"),
});
const times = createDateTimes(domain, {
  outOfRange: identity("can.std.time@1::out_of_range"),
  invalidZone: identity("can.std.time@1::invalid_zone"),
  nonexistent: identity("can.std.time@1::nonexistent_time"),
  invalidOption: identity("can.std.time@1::invalid_option"),
});
const origin = { source: "test", start: 0, end: 0, invocation: [] };
const fail = (id: string, fields: readonly (readonly [string, unknown])[], cause?: unknown) =>
  failure(domain.create(id, record(id, fields), origin, cause));
const someText = (v: unknown) => record(SOME_TEXT, [["value", v]]);
const someInt = (v: unknown) => record(SOME_INT, [["value", v]]);
const someCont = (v: unknown) => record(SOME_CONT, [["value", v]]);
const none = () => record(NONE, []);
const writeOpts = (contentType: unknown) =>
  record("can.std.s3@1::write_options", [["content_type", contentType]]);
const uploadOpts = (contentType: unknown, partSize: unknown) =>
  record("can.std.s3@1::upload_options", [
    ["content_type", contentType],
    ["part_size", partSize],
  ]);
const listOpts = (prefix: string, limit: bigint, delimiter: unknown, continuation: unknown) =>
  record("can.std.s3@1::list_options", [
    ["prefix", prefix],
    ["limit", limit],
    ["delimiter", delimiter],
    ["continuation", continuation],
  ]);
const settled = async () => {
  // Absence after cancel must settle: cleanup completes the native
  // upload synchronously and deletes the key, so only a settled
  // check catches a completion leak.
  try {
    (globalThis as unknown as { Bun?: { gc?: (force?: boolean) => void } }).Bun?.gc?.(true);
  } catch {
    /* best effort */
  }
  await new Promise((resolve) => setTimeout(resolve, 1200));
};
const buf = (text: string) => ownBytes(new TextEncoder().encode(text));
const textOf = (body: unknown) => new TextDecoder().decode(copyBytes(body, origin));
const metaOf = async (meta: unknown) => ({
  size: dataProperty(meta, "size") as bigint,
  etag: dataProperty(meta, "etag") as string,
  type: dataProperty(meta, "content_type") as string,
  modified: value(await times.instantEpochMillis(dataProperty(meta, "last_modified"))) as bigint,
});
async function owned(body: () => Promise<void>): Promise<void> {
  const out = await runOwnedRoot(async () => {
    await body();
    return success(undefined);
  });
  if (out.completion.kind !== "ok" || out.cleanupFailed) {
    const cause =
      out.completion.kind === "standard" && isStandardFailure(out.completion.value)
        ? (standardFailureDiagnostics(out.completion.value as never).cause as {
            message?: unknown;
            expected?: unknown;
            actual?: unknown;
            diff?: unknown;
            stack?: unknown;
          })
        : undefined;
    const detail =
      out.completion.kind === "domain"
        ? JSON.stringify(domainFailureDiagnostics(out.completion.value).payload)
        : cause !== undefined
          ? JSON.stringify({
              message: String(cause.message),
              stack: typeof cause.stack === "string" ? cause.stack.slice(0, 600) : "",
            })
          : out.completion.kind;
    throw Error(`owned body failed: ${detail} cleanupFailed=${out.cleanupFailed}`);
  }
}
// Live tests run only with a provisioned S3-compatible endpoint; without
// it they skip visibly instead of passing vacuously. Credentials never
// appear in assertions, messages, or snapshots: only variable names travel.
const ENDPOINT = process.env["CAN_TEST_S3_ENDPOINT"],
  ACCESS = process.env["CAN_TEST_S3_ACCESS_KEY"],
  SECRET = process.env["CAN_TEST_S3_SECRET_KEY"];
const BUCKET = process.env["CAN_TEST_S3_BUCKET"] ?? "can-b1-10";
const live = test.skipIf(ENDPOINT === undefined || ACCESS === undefined || SECRET === undefined);
const PREFIX = `s3t/${Date.now().toString(36)}/`;
const openLive = async () =>
  value(await s3.clientOpen(ENDPOINT!, "us-east-1", BUCKET, ACCESS!, SECRET!)) as object;
const refusedEndpoint = () => {
  const match = ENDPOINT!.match(/^(https?:\/\/[^:]+)(:\d+)?(.*)$/);
  return match![1] + ":9" + (match![3] ?? "");
};
async function liveOwned(body: (keys: string[]) => Promise<void>): Promise<void> {
  const keys: string[] = [];
  await owned(async () => {
    try {
      await body(keys);
    } finally {
      if (keys.length > 0) {
        try {
          const client = await openLive();
          for (const key of keys) {
            try {
              value(await s3.remove(client, key));
            } catch {
              /* best effort */
            }
          }
        } catch {
          /* best effort */
        }
      }
    }
  });
}
const byteReader = (chunks: Uint8Array[]): object => {
  let index = 0;
  const stream = new ReadableStream<Uint8Array>({
    pull(controller) {
      if (index < chunks.length) controller.enqueue(chunks[index++]);
      else controller.close();
    },
  });
  return registerReader(
    openByteCell(stream, 65536n),
    fail,
    identity("can.std.stream@1::close_failed"),
    { scopeManaged: true },
  );
};
test("client_open validates endpoint, region, bucket and credentials", async () => {
  await owned(async () => {
    check(await s3.clientOpen("not a url", "us-east-1", "b", "a", "s"), 1343, {
      reason: "endpoint",
    });
    check(await s3.clientOpen("ftp://x", "us-east-1", "b", "a", "s"), 1343, { reason: "endpoint" });
    check(await s3.clientOpen("http://127.0.0.1:9", "", "b", "a", "s"), 1343, { reason: "region" });
    check(await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "", "a", "s"), 1343, {
      reason: "bucket",
    });
    check(await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "", "s"), 1343, {
      reason: "credentials",
    });
    check(await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", ""), 1343, {
      reason: "credentials",
    });
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    expect(isS3Value(S3_CLIENT_KIND, client)).toBe(true);
    expect(isS3Value(S3_UPLOAD_KIND, client)).toBe(false);
    expect(isS3Value(S3_PRESIGNED_KIND, client)).toBe(false);
    expect(isS3Value(undefined, client)).toBe(false);
    expect(isS3Value(S3_CLIENT_KIND, Object.freeze({}))).toBe(false);
  });
});
test("read validation fails before any wire call", async () => {
  await owned(async () => {
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    check(await s3.readBytes(client, "", 8n), 1343, { reason: "key" });
    check(await s3.readBytes(client, "k", 0n), 1343, { reason: "limit" });
    check(await s3.readBytes(client, "k", -1n), 1343, { reason: "limit" });
    check(await s3.readBytes(client, "k", 67108865n), 1343, { reason: "limit" });
    check(await s3.readRange(client, "", 0n, 1n), 1343, { reason: "key" });
    check(await s3.readRange(client, "k", -1n, 1n), 1343, { reason: "offset" });
    check(await s3.readRange(client, "k", 0n, -1n), 1343, { reason: "length" });
    check(await s3.readStream(client, "", 8n), 1343, { reason: "key" });
    check(await s3.readStream(client, "k", 0n), 1343, { reason: "max_bytes" });
    check(await s3.stat(client, ""), 1343, { reason: "key" });
    check(await s3.exists(client, ""), 1343, { reason: "key" });
    check(await s3.remove(client, ""), 1343, { reason: "key" });
    // Zero-length ranges short-circuit without touching the wire, even
    // against an endpoint that refuses every connection.
    expect(textOf(value(await s3.readRange(client, "k", 100n, 0n)))).toBe("");
  });
});
test("write validation fails before any wire call", async () => {
  await owned(async () => {
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    check(await s3.writeBytes(client, "", buf("x"), writeOpts(none())), 1343, { reason: "key" });
    check(await s3.writeBytes(client, "k", buf("x"), writeOpts(someText(""))), 1343, {
      reason: "content_type",
    });
    check(
      await s3.writeBytes(client, "k", buf("x"), writeOpts(someText("text/plain\r\nX: y"))),
      1343,
      { reason: "content_type" },
    );
    check(await s3.list(client, listOpts("p", 0n, none(), none())), 1343, { reason: "max_keys" });
    check(await s3.list(client, listOpts("p", 10n, someText(""), none())), 1343, {
      reason: "delimiter",
    });
    check(await s3.presign(client, record(METHOD_GET, []), "", 300n, none()), 1343, {
      reason: "key",
    });
    check(await s3.presign(client, record(METHOD_GET, []), "k", 0n, none()), 1343, {
      reason: "expires",
    });
    check(await s3.presign(client, record(METHOD_GET, []), "k", -5n, none()), 1343, {
      reason: "expires",
    });
    check(await s3.presign(client, record(METHOD_GET, []), "k", 604801n, none()), 1343, {
      reason: "expires",
    });
    check(await s3.beginUpload(client, "", uploadOpts(none(), none())), 1343, { reason: "key" });
    check(await s3.beginUpload(client, "k", uploadOpts(none(), someInt(1024n))), 1343, {
      reason: "part_size",
    });
    check(await s3.beginUpload(client, "k", uploadOpts(someText(""), none())), 1343, {
      reason: "content_type",
    });
  });
});
test("presign mints locally and describe reveals method and expiry", async () => {
  await owned(async () => {
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    for (const leaf of [METHOD_GET, METHOD_PUT, METHOD_DELETE, METHOD_HEAD] as const) {
      const token = value(
        await s3.presign(client, record(leaf, []), "dir/obj.bin", 300n, none()),
      ) as object;
      expect(isS3Value(S3_PRESIGNED_KIND, token)).toBe(true);
      const info = value(await s3.describe(token));
      const url = dataProperty(info, "url") as string;
      expect(url).toContain("dir/obj.bin");
      expect(url).toContain("X-Amz-Signature");
      expect(recordIdentity(dataProperty(info, "method"))).toBe(leaf);
      const expires = value(
        await times.instantEpochMillis(dataProperty(info, "expires_at")),
      ) as bigint;
      const now = BigInt(Date.now());
      expect(expires - now > 290000n && expires - now <= 300000n).toBe(true);
    }
    let thrown: unknown;
    try {
      value(await s3.describe(Object.freeze({})));
    } catch (cause) {
      thrown = cause;
    }
    expect(thrown instanceof TypeError).toBe(true);
  });
});
test("upload terminal states reject reuse without wire calls", async () => {
  await owned(async () => {
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    const upload = value(await s3.beginUpload(client, "k", uploadOpts(none(), none()))) as object;
    expect(isS3Value(S3_UPLOAD_KIND, upload)).toBe(true);
    value(await s3.cancelUpload(upload));
    check(await s3.uploadWrite(upload, buf("x")), 1347, {
      operation: "upload_write",
      state: "cancelled",
    });
    check(await s3.uploadFinish(upload), 1347, { operation: "upload_finish", state: "cancelled" });
    check(await s3.cancelUpload(upload), 1347, { operation: "cancel_upload", state: "cancelled" });
    let thrown: unknown;
    try {
      value(await s3.uploadWrite(Object.freeze({}), buf("x")));
    } catch (cause) {
      thrown = cause;
    }
    expect(thrown instanceof TypeError).toBe(true);
  });
});
test("write_stream rejects non-bytes readers before uploading", async () => {
  await owned(async () => {
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    const stream = new ReadableStream<Uint8Array>({
      start(c) {
        c.enqueue(new TextEncoder().encode("hi\n"));
        c.close();
      },
    });
    const lines = registerReader(
      openLineCell(stream, 1024n),
      fail,
      identity("can.std.stream@1::close_failed"),
      { scopeManaged: true },
    );
    check(await s3.writeStream(client, "k", lines, writeOpts(none()), 1024n, 5000n), 1343, {
      reason: "reader",
    });
    check(await s3.writeStream(client, "", byteReader([]), writeOpts(none()), 1024n, 5000n), 1343, {
      reason: "key",
    });
    check(await s3.writeStream(client, "k", byteReader([]), writeOpts(none()), 0n, 5000n), 1343, {
      reason: "max_bytes",
    });
    check(await s3.writeStream(client, "k", byteReader([]), writeOpts(none()), 1024n, 0n), 1343, {
      reason: "deadline",
    });
  });
});
test("supplied operations deny the live assertion boundary", async () => {
  await owned(async () => {
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    const context = assertionContext({
      package: "can.project.root/app",
      declaration: "can.project.root/app::probe",
      name: "sample",
    });
    try {
      for (const run of [
        () => s3.readBytes(client, "k", 8n, context),
        () => s3.readRange(client, "k", 0n, 1n, context),
        () => s3.readStream(client, "k", 8n, context),
        () => s3.writeBytes(client, "k", buf("x"), writeOpts(none()), context),
        () => s3.writeStream(client, "k", byteReader([]), writeOpts(none()), 8n, 1000n, context),
        () => s3.stat(client, "k", context),
        () => s3.exists(client, "k", context),
        () => s3.remove(client, "k", context),
        () => s3.list(client, listOpts("", 10n, none(), none()), context),
        () => s3.presign(client, record(METHOD_GET, []), "k", 60n, none(), context),
        () => s3.beginUpload(client, "k", uploadOpts(none(), none()), context),
        () => s3.cancelUpload(Object.freeze(Object.create(null)), context),
      ]) {
        let thrown: unknown;
        try {
          await run();
        } catch (cause) {
          thrown = cause;
        }
        expect(thrown).toBeDefined();
      }
      // Real operations run under assertion context.
      const local = value(
        await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s", context),
      ) as object;
      expect(isS3Value(S3_CLIENT_KIND, local)).toBe(true);
      const token = value(
        await s3.presign(client, record(METHOD_GET, []), "k", 60n, none()),
      ) as object;
      expect(typeof dataProperty(value(await s3.describe(token, context)), "url")).toBe("string");
    } finally {
      closeContext(context);
    }
  });
});
live("roundtrip writes, stats, reads and deletes bytes", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "round.bin";
    keys.push(key);
    const meta = await metaOf(
      value(await s3.writeBytes(client, key, buf("hello s3"), writeOpts(someText("text/plain")))),
    );
    expect(meta.size).toBe(8n);
    expect(meta.etag).toBe('"b60d991fa6dbd33863c7cec70d15dbac"');
    expect(meta.type.startsWith("text/plain")).toBe(true);
    expect(meta.modified > 0n).toBe(true);
    const again = await metaOf(value(await s3.stat(client, key)));
    expect(again).toEqual(meta);
    expect(value(await s3.exists(client, key))).toBe(true);
    expect(textOf(value(await s3.readBytes(client, key, 64n)))).toBe("hello s3");
    value(await s3.remove(client, key));
    expect(value(await s3.exists(client, key))).toBe(false);
    keys.pop();
  });
});
live("read_bytes fails over_limit without downloading", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "over.bin";
    keys.push(key);
    value(await s3.writeBytes(client, key, buf("12345678"), writeOpts(none())));
    check(await s3.readBytes(client, key, 4n), 1348, { limit: 4n, size: 8n });
    expect(textOf(value(await s3.readBytes(client, key, 8n)))).toBe("12345678");
  });
});
live("read_range serves exact windows and pins range faults", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "range.bin";
    keys.push(key);
    value(await s3.writeBytes(client, key, buf("0123456789"), writeOpts(none())));
    expect(textOf(value(await s3.readRange(client, key, 0n, 4n)))).toBe("0123");
    expect(textOf(value(await s3.readRange(client, key, 8n, 10n)))).toBe("89");
    check(await s3.readRange(client, key, 100n, 10n), 1346, {
      code: "InvalidRange",
      operation: "read_range",
    });
    check(await s3.readRange(client, PREFIX + "absent.bin", 0n, 4n), 1344, {
      key: PREFIX + "absent.bin",
    });
  });
});
live("read_stream reassembles bytes and cancels terminally", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "stream.bin";
    keys.push(key);
    const body = new Uint8Array(200000);
    for (let i = 0; i < body.length; i++) body[i] = i % 251;
    value(await s3.writeBytes(client, key, ownBytes(body), writeOpts(none())));
    check(await s3.readStream(client, key, 100n), 1348, { limit: 100n, size: 200000n });
    const token = value(await s3.readStream(client, key, 300000n)) as object;
    const chunks: Uint8Array[] = [];
    for (;;) {
      const batch = dataArray(value(await reads.readMany(token, 8n)));
      if (batch.length === 0) break;
      for (const item of batch) chunks.push(copyBytes(item, origin));
    }
    const joined = new Uint8Array(chunks.reduce((n, c) => n + c.byteLength, 0));
    let offset = 0;
    for (const chunk of chunks) {
      joined.set(chunk, offset);
      offset += chunk.byteLength;
    }
    expect(joined).toEqual(body);
    value(await reads.closeReader(token));
    const token2 = value(await s3.readStream(client, key, 300000n)) as object;
    expect(dataArray(value(await reads.readMany(token2, 1n))).length).toBe(1);
    value(await reads.cancelReader(token2, "test-done"));
    let thrown: unknown;
    try {
      value(await reads.readMany(token2, 1n));
    } catch (cause) {
      thrown = cause;
    }
    expect(standardOf(thrown)).toBe("resource_state");
  });
});
live("write_stream pumps readers under byte budgets", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "pump.bin";
    keys.push(key);
    const meta = await metaOf(
      value(
        await s3.writeStream(
          client,
          key,
          byteReader([new TextEncoder().encode("ab"), new TextEncoder().encode("cd")]),
          writeOpts(someText("application/octet-stream")),
          64n,
          30000n,
        ),
      ),
    );
    expect(meta.size).toBe(4n);
    expect(meta.type).toBe("application/octet-stream");
    expect(textOf(value(await s3.readBytes(client, key, 64n)))).toBe("abcd");
    const overKey = PREFIX + "pump-over.bin";
    keys.push(overKey);
    check(
      await s3.writeStream(
        client,
        overKey,
        byteReader([new TextEncoder().encode("abcdef")]),
        writeOpts(none()),
        4n,
        30000n,
      ),
      1348,
      { limit: 4n, size: 6n },
    );
    await settled();
    expect(value(await s3.exists(client, overKey))).toBe(false);
  });
});
live("write_stream cancels the upload when the reader fails", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "pump-fail.bin";
    keys.push(key);
    const stream = new ReadableStream<Uint8Array>({
      pull() {
        throw Object.assign(new Error("reader boom"), { code: "reader_boom" });
      },
    });
    const reader = registerReader(
      openByteCell(stream, 65536n),
      fail,
      identity("can.std.stream@1::close_failed"),
      { scopeManaged: true },
    );
    check(await s3.writeStream(client, key, reader, writeOpts(none()), 1024n, 30000n), 1316, {
      reason: "reader_boom",
    });
    await settled();
    expect(value(await s3.exists(client, key))).toBe(false);
  });
});
live("write_stream fails timeout past the deadline", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "pump-slow.bin";
    keys.push(key);
    let pulls = 0;
    const stream = new ReadableStream<Uint8Array>({
      pull: async (controller) => {
        pulls++;
        await new Promise((resolve) => setTimeout(resolve, 500));
        controller.enqueue(new TextEncoder().encode("x"));
      },
    });
    const reader = registerReader(
      openByteCell(stream, 65536n),
      fail,
      identity("can.std.stream@1::close_failed"),
      { scopeManaged: true },
    );
    check(await s3.writeStream(client, key, reader, writeOpts(none()), 1024n, 50n), 1346, {
      code: "timeout",
      operation: "write_stream",
    });
    // One consumed chunk plus at most one high-water-mark prefetch: the
    // pump stops pulling once the deadline fires.
    expect(pulls <= 2).toBe(true);
    await settled();
    expect(value(await s3.exists(client, key))).toBe(false);
  });
});
live("multipart upload finishes, guards terminals and abandons cleanly", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "multi.bin";
    keys.push(key);
    const upload = value(
      await s3.beginUpload(
        client,
        key,
        uploadOpts(someText("application/octet-stream"), someInt(5242880n)),
      ),
    ) as object;
    const part = new Uint8Array(3145728);
    for (let i = 0; i < part.length; i++) part[i] = i % 251;
    expect(value(await s3.uploadWrite(upload, ownBytes(part)))).toBe(BigInt(part.length));
    expect(value(await s3.uploadWrite(upload, ownBytes(part)))).toBe(BigInt(part.length));
    const meta = await metaOf(value(await s3.uploadFinish(upload)));
    expect(meta.size).toBe(6291456n);
    expect(meta.etag.endsWith('-2"')).toBe(true);
    check(await s3.uploadWrite(upload, ownBytes(new Uint8Array([1]))), 1347, {
      operation: "upload_write",
      state: "finished",
    });
    check(await s3.uploadFinish(upload), 1347, { operation: "upload_finish", state: "finished" });
    check(await s3.cancelUpload(upload), 1347, { operation: "cancel_upload", state: "finished" });
    const emptyKey = PREFIX + "multi-empty.bin";
    keys.push(emptyKey);
    const emptyMeta = await metaOf(
      value(
        await s3.uploadFinish(
          value(await s3.beginUpload(client, emptyKey, uploadOpts(none(), none()))) as object,
        ),
      ),
    );
    expect(emptyMeta.size).toBe(0n);
    const dropKey = PREFIX + "multi-drop.bin";
    keys.push(dropKey);
    const dropped = value(
      await s3.beginUpload(client, dropKey, uploadOpts(none(), none())),
    ) as object;
    value(await s3.uploadWrite(dropped, ownBytes(new Uint8Array([7]))));
    value(await s3.cancelUpload(dropped));
    await settled();
    expect(value(await s3.exists(client, dropKey))).toBe(false);
  });
});
live("abandoned uploads never materialize their key", async () => {
  const client = await openLive();
  const key = PREFIX + "multi-abandon.bin";
  await owned(async () => {
    const upload = value(await s3.beginUpload(client, key, uploadOpts(none(), none()))) as object;
    value(await s3.uploadWrite(upload, ownBytes(new Uint8Array([7]))));
    // No finish or cancel: scope drain retires the handle.
  });
  await owned(async () => {
    await settled();
    expect(value(await s3.exists(client, key))).toBe(false);
    value(await s3.remove(client, key));
  });
});
live("list pages, continues and groups with opaque tokens", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const base = PREFIX + "ls/";
    for (const name of ["a.txt", "b.txt", "dir/c.txt"]) {
      keys.push(base + name);
      value(await s3.writeBytes(client, base + name, buf("x"), writeOpts(none())));
    }
    const first = value(await s3.list(client, listOpts(base, 1n, none(), none())));
    expect(dataProperty(first, "truncated")).toBe(true);
    const entries = dataArray(dataProperty(first, "entries"));
    expect(entries.length).toBe(1);
    const cont = dataProperty(first, "continuation");
    expect(recordIdentity(cont)).toBe(SOME_CONT);
    const token = dataProperty(cont, "value") as object;
    expect(isS3Value(S3_CONTINUATION_KIND, token)).toBe(true);
    const second = value(await s3.list(client, listOpts(base, 10n, none(), someCont(token))));
    const keys2 = dataArray(dataProperty(second, "entries")).map((e) => dataProperty(e, "key"));
    expect(dataProperty(entries[0], "key") !== keys2[0]).toBe(true);
    const full = value(await s3.list(client, listOpts(base, 10n, none(), none())));
    expect(dataProperty(full, "truncated")).toBe(false);
    expect(dataArray(dataProperty(full, "entries")).length).toBe(3);
    expect(recordIdentity(dataProperty(full, "continuation"))).toBe(NONE);
    const grouped = value(await s3.list(client, listOpts(base, 10n, someText("/"), none())));
    expect(dataArray(dataProperty(grouped, "prefixes"))).toEqual([base + "dir/"]);
    const entry = dataArray(dataProperty(full, "entries"))[0];
    expect(typeof dataProperty(entry, "etag")).toBe("string");
    expect(
      (value(await times.instantEpochMillis(dataProperty(entry, "last_modified"))) as bigint) > 0n,
    ).toBe(true);
    const bare = value(await s3.list(client, listOpts(base + "nope/", 10n, none(), none())));
    expect(dataArray(dataProperty(bare, "entries"))).toEqual([]);
    expect(dataProperty(bare, "truncated")).toBe(false);
  });
});
live("presigned urls roundtrip and fail closed on misuse", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const key = PREFIX + "signed.bin";
    keys.push(key);
    value(await s3.writeBytes(client, key, buf("signed"), writeOpts(none())));
    const urlOf = async (method: string, k: string, expires: bigint) => {
      const token = value(
        await s3.presign(client, record(method, []), k, expires, none()),
      ) as object;
      return dataProperty(value(await s3.describe(token)), "url") as string;
    };
    const getUrl0 = await urlOf(METHOD_GET, key, 300n);
    expect(getUrl0.includes(SECRET!)).toBe(false);
    const get = await fetch(getUrl0);
    expect(get.status).toBe(200);
    expect(await get.text()).toBe("signed");
    const putKey = PREFIX + "signed-put.bin";
    keys.push(putKey);
    const put = await fetch(await urlOf(METHOD_PUT, putKey, 300n), {
      method: "PUT",
      body: "via-url",
    });
    expect(put.status).toBe(200);
    expect(textOf(value(await s3.readBytes(client, putKey, 64n)))).toBe("via-url");
    const putUrl = await urlOf(METHOD_PUT, key, 300n);
    expect((await fetch(putUrl, { method: "GET" })).status).toBe(403);
    const getUrl = await urlOf(METHOD_GET, key, 300n);
    expect((await fetch(getUrl, { method: "PUT", body: "x" })).status).toBe(403);
    expect(
      (await fetch(getUrl.replace(/X-Amz-Signature=[^&]+/, "X-Amz-Signature=00"))).status,
    ).toBe(403);
    const delKey = PREFIX + "signed-del.bin";
    keys.push(delKey);
    value(await s3.writeBytes(client, delKey, buf("d"), writeOpts(none())));
    expect(
      (await fetch(await urlOf(METHOD_DELETE, delKey, 300n), { method: "DELETE" })).status,
    ).toBe(204);
    expect(value(await s3.exists(client, delKey))).toBe(false);
    const head = await fetch(await urlOf(METHOD_HEAD, key, 300n), { method: "HEAD" });
    expect(head.status).toBe(200);
    expect(head.headers.get("content-length")).toBe("6");
    expect((await fetch(await urlOf(METHOD_GET, PREFIX + "absent-signed.bin", 300n))).status).toBe(
      404,
    );
  });
});
live("denied credentials, refused endpoints and missing keys map exactly", async () => {
  await liveOwned(async (keys) => {
    const client = await openLive();
    const bad = value(
      await s3.clientOpen(ENDPOINT!, "us-east-1", BUCKET, "nope", "nope"),
    ) as object;
    check(await s3.stat(bad, PREFIX + "round.bin"), 1345, { operation: "stat" });
    check(await s3.readBytes(bad, PREFIX + "round.bin", 8n), 1345, { operation: "read_bytes" });
    const refused = value(
      await s3.clientOpen(refusedEndpoint(), "us-east-1", BUCKET, ACCESS!, SECRET!),
    ) as object;
    check(await s3.stat(refused, "k"), 1346, { code: "ConnectionRefused", operation: "stat" });
    check(await s3.readBytes(refused, "k", 8n), 1346, {
      code: "ConnectionRefused",
      operation: "read_bytes",
    });
    check(await s3.stat(client, PREFIX + "absent.bin"), 1344, { key: PREFIX + "absent.bin" });
    check(await s3.readBytes(client, PREFIX + "absent.bin", 8n), 1344, {
      key: PREFIX + "absent.bin",
    });
    expect(value(await s3.exists(client, PREFIX + "absent.bin"))).toBe(false);
    value(await s3.remove(client, PREFIX + "absent.bin"));
    const noBucket = value(
      await s3.clientOpen(ENDPOINT!, "us-east-1", "no-such-bucket-zzz", ACCESS!, SECRET!),
    ) as object;
    check(await s3.stat(noBucket, "k"), 1344, { key: "k" });
    check(await s3.list(noBucket, listOpts("", 5n, none(), none())), 1346, {
      code: "NoSuchBucket",
      operation: "list",
    });
    expect(keys).toEqual([]);
  });
});
