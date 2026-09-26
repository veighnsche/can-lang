// E08 (R15 remedy): destructive discard_upload + between-awaits deadline legs.
//
// S3-PROTOCOL scope: every live leg runs against the provisioned
// MinIO endpoint (never AWS-real) under an isolated `s3e08/` prefix;
// the evidence record carries that caveat. Without the
// endpoint/credentials the live legs skip visibly instead of passing
// vacuously.
//
// Both E07 branches went negative, so this suite pins the E08 remedy:
//   - `s3::cancel_upload` is gone from the catalogue; the explicit
//     `s3::discard_upload` carries the destructive contract (complete
//     the pending upload — transiently visible, overwriting any
//     pre-existing key — then delete it) with honest cleanup-failure
//     reporting.
//   - `s3::write_stream` keeps only the qualified between-awaits
//     deadline: it fires at the top of each pump iteration, never
//     inside a hung await (the H-term E07 legs remain the documented
//     unbounded remainder; W5 stays blocked on it).
// Destructive-overwrite, transient-window, and cleanup-injection
// validation lives in the rebased E07 legs (C0/C4/F1–F7); this file
// adds the rename/contract legs, the terminal guards, and the
// timeout-path scrub wire shape.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue, operation } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createS3 } from "../platform/s3.ts";
import { openByteCell } from "../transport/stream/readable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { value, success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { runOwnedRoot } from "../owner.ts";
import { startTap, type Tap } from "./s3-e07-tap.ts";

const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update(`can-concrete-type-v1\0${JSON.stringify([kind, name])}`)
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
    "files::limit_exceeded",
    "stream::read_failed",
    "stream::cancelled",
    "stream::close_failed",
    "s3::invalid_config",
    "s3::missing_key",
    "s3::access_denied",
    "s3::service_error",
    "s3::upload_closed",
    "s3::over_limit",
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
const SOME_TEXT = "can.std.option@1::some<str>",
  SOME_INT = "can.std.option@1::some<int>",
  NONE = "can.std.option@1::none";
const s3 = createS3(domain, {
  invalid: identity("can.std.s3@1::invalid_config"),
  missing: identity("can.std.s3@1::missing_key"),
  denied: identity("can.std.s3@1::access_denied"),
  service: identity("can.std.s3@1::service_error"),
  closed: identity("can.std.s3@1::upload_closed"),
  overLimit: identity("can.std.s3@1::over_limit"),
  metadata: "can.std.s3@1::metadata",
  entry: "can.std.s3@1::entry",
  page: "can.std.s3@1::page",
  info: "can.std.s3@1::presigned_info",
  methodGet: "can.std.s3@1::method_get",
  methodPut: "can.std.s3@1::method_put",
  methodDelete: "can.std.s3@1::method_delete",
  methodHead: "can.std.s3@1::method_head",
  someText: SOME_TEXT,
  someInt: SOME_INT,
  someContinuation: "can.std.option@1::some<s3::continuation>",
  none: NONE,
  readFailed: identity("can.std.stream@1::read_failed"),
  cancelled: identity("can.std.stream@1::cancelled"),
  closeFailed: identity("can.std.stream@1::close_failed"),
});
const origin = { source: "test", start: 0, end: 0, invocation: [] };
const fail = (id: string, fields: readonly (readonly [string, unknown])[], cause?: unknown) =>
  failure(domain.create(id, record(id, fields), origin, cause));
const none = () => record(NONE, []);
const writeOpts = (contentType: unknown) =>
  record("can.std.s3@1::write_options", [["content_type", contentType]]);
const uploadOpts = (contentType: unknown, partSize: unknown) =>
  record("can.std.s3@1::upload_options", [
    ["content_type", contentType],
    ["part_size", partSize],
  ]);
const buf = (text: string) => ownBytes(new TextEncoder().encode(text));
async function owned(body: () => Promise<void>): Promise<void> {
  const out = await runOwnedRoot(async () => {
    await body();
    return success(undefined);
  });
  if (out.completion.kind !== "ok" || out.cleanupFailed) {
    const detail =
      out.completion.kind === "domain"
        ? JSON.stringify(domainFailureDiagnostics(out.completion.value).payload)
        : out.completion.kind;
    throw Error(`owned body failed: ${detail} cleanupFailed=${out.cleanupFailed}`);
  }
}

const ENDPOINT = process.env["CAN_TEST_S3_ENDPOINT"],
  ACCESS = process.env["CAN_TEST_S3_ACCESS_KEY"],
  SECRET = process.env["CAN_TEST_S3_SECRET_KEY"];
const BUCKET = process.env["CAN_TEST_S3_BUCKET"] ?? "can-b1-10";
const REGION = process.env["CAN_TEST_S3_REGION"] ?? "us-east-1";
const live = test.skipIf(ENDPOINT === undefined || ACCESS === undefined || SECRET === undefined);
const PREFIX = `s3e08/e08harness/${Date.now().toString(36)}/`;

const openTap = async (tap: Tap) =>
  value(
    await s3.clientOpen(`http://127.0.0.1:${tap.port}`, REGION, BUCKET, ACCESS!, SECRET!),
  ) as object;
const upstreamOf = (): { host: string; port: number } => {
  const url = new URL(ENDPOINT!);
  return { host: url.hostname, port: Number(url.port) };
};
const opTriples = (tap: Tap): string[] =>
  tap.ops.map((entry) => `${entry.method} ${entry.op} ${entry.verdict}`);

// --- Rename atomicity: no silent delete may keep the cancel name ---

test("E08 E1: s3::cancel_upload is absent from the catalogue", () => {
  expect(() => operation("s3::cancel_upload")).toThrow();
  // Widen to string: the generated literal unions no longer contain
  // the removed name, which is itself compile-time proof of removal.
  const names = catalogue.operations.map((item): string => item.name);
  const identities = catalogue.operations.map((item): string => item.identity);
  expect(names.some((name) => name === "s3::cancel_upload")).toBe(false);
  expect(identities.some((id) => id === "can.std.s3@1::cancel_upload")).toBe(false);
});

test("E08 E2: s3::discard_upload carries the destructive contract", () => {
  const found = operation("s3::discard_upload");
  expect(found.identity).toBe("can.std.s3@1::discard_upload");
  expect(found.inputs).toEqual([{ name: "upload", type: "s3::upload" }]);
  expect(found.result).toBe("void");
  expect([...found.emits].sort()).toEqual([
    "s3::access_denied",
    "s3::service_error",
    "s3::upload_closed",
  ]);
  // The contract must name the destruction, the transient window,
  // and the orphan path: no silent-delete reading may survive.
  for (const phrase of [
    "Destructively",
    "transiently visible",
    "pre-existing key",
    "service errors",
    "operator abort recipe",
  ]) {
    expect(found.lowering.adapter).toContain(phrase);
  }
  expect(found.lowering.adapter.toLowerCase()).not.toContain("never deletes");
  expect(found.lowering.adapter.toLowerCase()).not.toContain("never materializes");
});

test("E08 E3: write_stream documents only the between-awaits deadline", () => {
  const pump = operation("s3::write_stream");
  expect(pump.lowering.adapter).toContain("only between awaits");
  expect(pump.lowering.adapter).toContain("never a hung await");
  expect(pump.lowering.adapter).toContain("discards the upload destructively");
  expect(pump.lowering.adapter.toLowerCase()).not.toContain("cancel");
  const append = operation("s3::upload_write");
  expect(append.lowering.adapter).toContain("finish or discard");
});

// --- Terminal guards: discarded handles reject every reuse ---

test("E08 D3: discarded uploads reject reuse without wire calls", async () => {
  await owned(async () => {
    const client = value(
      await s3.clientOpen("http://127.0.0.1:9", "us-east-1", "b", "a", "s"),
    ) as object;
    const upload = value(await s3.beginUpload(client, "k", uploadOpts(none(), none()))) as object;
    // No sink exists yet, so the discard retires the handle with no I/O.
    expect(value(await s3.discardUpload(upload))).toBe(undefined);
    check(await s3.uploadWrite(upload, buf("x")), "s3::upload_closed", {
      operation: "upload_write",
      state: "discarded",
    });
    check(await s3.uploadFinish(upload), "s3::upload_closed", {
      operation: "upload_finish",
      state: "discarded",
    });
    check(await s3.discardUpload(upload), "s3::upload_closed", {
      operation: "discard_upload",
      state: "discarded",
    });
  });
});

// --- Deadline posture: the timeout path scrubs destructively ---

live(
  "E08 D4: write_stream timeout completes then deletes the partial key",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}d4.bin`;
        // 60 ms sleeps against a 100 ms deadline: the third loop-top
        // check fires regardless of scheduling jitter, after at
        // least one chunk landed in the sink buffer.
        let pulls = 0;
        const stream = new ReadableStream<Uint8Array>({
          pull: async (controller) => {
            pulls++;
            await Bun.sleep(60);
            controller.enqueue(new TextEncoder().encode("x"));
          },
        });
        const reader = registerReader(
          openByteCell(stream, 65536n),
          fail,
          identity("can.std.stream@1::close_failed"),
          { scopeManaged: true },
        );
        tap.clearOps();
        check(
          await s3.writeStream(client, key, reader, writeOpts(none()), 1024n, 100n),
          "s3::service_error",
          { code: "timeout", operation: "write_stream" },
        );
        expect(tap.desyncs()).toBe(0);
        console.log(JSON.stringify({ leg: "D4", pulls, ops: opTriples(tap) }));
        expect(pulls).toBeGreaterThanOrEqual(1);
        // The timed-out pump scrubs through the shared destructive
        // cleanup: one completion of the buffered bytes, then the
        // delete. Nothing else touches the wire on this path.
        expect(opTriples(tap)).toEqual([
          "PUT put-object forwarded",
          "DELETE delete-object forwarded",
        ]);
        expect(value(await s3.exists(client, key))).toBe(false);
      });
    } finally {
      tap.close();
    }
  },
  20000,
);
