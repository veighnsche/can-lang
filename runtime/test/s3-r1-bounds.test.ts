// R1 (retained X-R15-3): every S3 await races a layered bound.
//
// S3-PROTOCOL scope: live legs run against the provisioned MinIO
// endpoint (never AWS-real) under an isolated `s3r1/` prefix. Without
// the endpoint/credentials the live legs skip visibly. Blackhole legs
// hold service traffic at the tap so hangs are deterministic; recovery
// legs drop the held connections and prove deferred convergence
// (absent key, no MPU orphans, one late settlement record).
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createS3, s3RaceBoundMs, S3_AWAIT_CEILING_MS, type S3Bounds } from "../platform/s3.ts";
import { openByteCell } from "../transport/stream/readable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { createRequestScope, runWithRequestScope } from "../transport/request-scope.ts";
import { value, success, failure, type Completion } from "../completion.ts";
import { record } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { runOwnedRoot } from "../owner.ts";
import { startTap, listUploads, type Tap, type S3Identity } from "./s3-e07-tap.ts";

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
const listOpts = (prefix: string, limit: bigint, delimiter: unknown, continuation: unknown) =>
  record("can.std.s3@1::list_options", [
    ["prefix", prefix],
    ["limit", limit],
    ["delimiter", delimiter],
    ["continuation", continuation],
  ]);
const buf = (text: string) => ownBytes(new TextEncoder().encode(text));
// 6MB forces a part push on flush (H2 shape): small chunks buffer
// client-side and never hang the flush against a blackhole.
const bigChunk = () => {
  const out = new Uint8Array(6 * 1024 * 1024);
  for (let i = 0; i < out.length; i++) out[i] = i % 251;
  return ownBytes(out);
};

const ENDPOINT = process.env["CAN_TEST_S3_ENDPOINT"],
  ACCESS = process.env["CAN_TEST_S3_ACCESS_KEY"],
  SECRET = process.env["CAN_TEST_S3_SECRET_KEY"];
const BUCKET = process.env["CAN_TEST_S3_BUCKET"] ?? "can-b1-10";
const REGION = process.env["CAN_TEST_S3_REGION"] ?? "us-east-1";
const live = test.skipIf(ENDPOINT === undefined || ACCESS === undefined || SECRET === undefined);
const PREFIX = `s3r1/r1harness/${Date.now().toString(36)}/`;
const openLive = async () =>
  value(await s3.clientOpen(ENDPOINT!, REGION, BUCKET, ACCESS!, SECRET!)) as object;
const openTap = async (tap: Tap) =>
  value(
    await s3.clientOpen(`http://127.0.0.1:${tap.port}`, REGION, BUCKET, ACCESS!, SECRET!),
  ) as object;
const upstreamOf = (): { host: string; port: number } => {
  const url = new URL(ENDPOINT!);
  return { host: url.hostname, port: Number(url.port) };
};
const ident = (): S3Identity => ({
  endpoint: ENDPOINT!,
  region: REGION,
  bucket: BUCKET,
  accessKey: ACCESS!,
  secretKey: SECRET!,
});
const opTriples = (tap: Tap): string[] =>
  tap.ops.map((entry) => `${entry.method} ${entry.op} ${entry.verdict}`);

// One scoped race: returns the owned outcome plus the scope that
// collected its escalation and late-settlement records.
const raceOp = async (invoke: () => Promise<Completion<unknown>>) => {
  const scope = createRequestScope();
  const started = Date.now();
  const outcome = await runWithRequestScope(scope, () => runOwnedRoot(invoke));
  return { scope, outcome, elapsed: Date.now() - started };
};
const awaitLates = async (
  scope: ReturnType<typeof createRequestScope>,
  count: number,
  budgetMs: number,
): Promise<boolean> => {
  const started = Date.now();
  for (;;) {
    if (scope.collected.lates.length >= count) return true;
    if (Date.now() - started >= budgetMs) return false;
    await Bun.sleep(20);
  }
};
const blackholeAll = (tap: Tap): void => {
  tap.setRules([{ action: "blackhole", match: () => true }]);
};

test("R1-0: s3RaceBoundMs layers explicit, trailing, and ceiling", () => {
  expect(S3_AWAIT_CEILING_MS).toBe(30000);
  expect(s3RaceBoundMs(undefined, undefined)).toBe(S3_AWAIT_CEILING_MS);
  expect(s3RaceBoundMs(100, undefined)).toBe(100);
  expect(s3RaceBoundMs(undefined, { boundMs: 50 })).toBe(50);
  expect(s3RaceBoundMs(100, { boundMs: 50 })).toBe(50);
  expect(s3RaceBoundMs(50, { boundMs: 100 })).toBe(50);
  expect(s3RaceBoundMs(100000, { boundMs: 60000 })).toBe(S3_AWAIT_CEILING_MS);
  // An exhausted remainder clamps to a ~immediate race, never below 1.
  expect(s3RaceBoundMs(0, undefined)).toBe(1);
  expect(s3RaceBoundMs(-5, { boundMs: 50 })).toBe(1);
});

live(
  "R1-1: data awaits answer timeout under blackhole",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      const seedKey = `${PREFIX}seed.bin`;
      await runOwnedRoot(async () => {
        const direct = await openLive();
        value(await s3.writeBytes(direct, seedKey, buf("seed-bytes"), writeOpts(none())));
        return success(undefined);
      });
      blackholeAll(tap);
      const rows: {
        op: string;
        invoke: (client: object) => Promise<Completion<unknown>>;
      }[] = [
        { op: "stat", invoke: (client) => s3.stat(client, seedKey, undefined, { boundMs: 50 }) },
        {
          op: "exists",
          invoke: (client) => s3.exists(client, seedKey, undefined, { boundMs: 50 }),
        },
        {
          op: "delete",
          invoke: (client) => s3.remove(client, `${PREFIX}gone.bin`, undefined, { boundMs: 50 }),
        },
        {
          op: "list",
          invoke: (client) =>
            s3.list(client, listOpts(PREFIX, 10n, none(), none()), undefined, { boundMs: 50 }),
        },
        {
          op: "read_bytes",
          invoke: (client) => s3.readBytes(client, seedKey, 1024n, undefined, { boundMs: 50 }),
        },
        {
          op: "read_range",
          invoke: (client) => s3.readRange(client, seedKey, 0n, 4n, undefined, { boundMs: 50 }),
        },
        {
          op: "read_stream",
          invoke: (client) => s3.readStream(client, seedKey, 1024n, undefined, { boundMs: 50 }),
        },
        {
          op: "write_bytes",
          invoke: (client) =>
            s3.writeBytes(client, `${PREFIX}w.bin`, buf("x"), writeOpts(none()), undefined, {
              boundMs: 50,
            }),
        },
      ];
      for (const row of rows) {
        const client = await openTap(tap);
        const { scope, outcome, elapsed } = await raceOp(() => row.invoke(client));
        expect(outcome.cleanupFailed).toBe(false);
        check(outcome.completion, "s3::service_error", { code: "timeout", operation: row.op });
        expect(elapsed).toBeLessThan(2000);
        expect(scope.collected.escalations).toHaveLength(1);
        expect(scope.collected.escalations[0]).toMatchObject({ cause: "budget" });
        expect(scope.collected.escalations[0]!.marker.source).toBe("s3");
      }
      console.log(JSON.stringify({ leg: "R1-1", ops: opTriples(tap) }));
      expect(tap.desyncs()).toBe(0);
      // The blackholed delete never reached the service: timeout is not deletion.
      await runOwnedRoot(async () => {
        const direct = await openLive();
        expect(value(await s3.exists(direct, seedKey))).toBe(true);
        return success(undefined);
      });
    } finally {
      tap.close();
    }
  },
  60000,
);

live(
  "R1-2: ambient budget remainder shortens the race",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      blackholeAll(tap);
      const scope = createRequestScope({ totalMs: 300 });
      await Bun.sleep(250);
      const started = Date.now();
      const outcome = await runWithRequestScope(scope, () =>
        runOwnedRoot(async () => {
          const client = await openTap(tap);
          return s3.stat(client, `${PREFIX}b.bin`, undefined, { boundMs: 5000 });
        }),
      );
      const elapsed = Date.now() - started;
      expect(outcome.cleanupFailed).toBe(false);
      check(outcome.completion, "s3::service_error", { code: "timeout", operation: "stat" });
      // The ~50ms budget remainder won over the 5000ms caller bound.
      expect(elapsed).toBeLessThan(2000);
      expect(scope.collected.escalations).toHaveLength(1);
      expect(scope.collected.escalations[0]!.effectiveMs).toBeLessThan(1000);
    } finally {
      tap.close();
    }
  },
  30000,
);

live(
  "R1-3: trailing boundMs shortens the race",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      blackholeAll(tap);
      const scope = createRequestScope({ totalMs: 30000 });
      const started = Date.now();
      const outcome = await runWithRequestScope(scope, () =>
        runOwnedRoot(async () => {
          const client = await openTap(tap);
          return s3.stat(client, `${PREFIX}b.bin`, undefined, { boundMs: 50 });
        }),
      );
      const elapsed = Date.now() - started;
      expect(outcome.cleanupFailed).toBe(false);
      check(outcome.completion, "s3::service_error", { code: "timeout", operation: "stat" });
      expect(elapsed).toBeLessThan(2000);
      expect(scope.collected.escalations).toHaveLength(1);
      expect(scope.collected.escalations[0]!.effectiveMs).toBe(50);
    } finally {
      tap.close();
    }
  },
  30000,
);

live(
  "R1-4: upload expiry poisons the handle; recovery converges",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      blackholeAll(tap);
      const key = `${PREFIX}poison.bin`;
      const scope = createRequestScope();
      const outcome = await runWithRequestScope(scope, () =>
        runOwnedRoot(async () => {
          const client = await openTap(tap);
          const upload = value(await s3.beginUpload(client, key, uploadOpts(none(), none())));
          const started = Date.now();
          const timed = await s3.uploadWrite(upload, bigChunk(), undefined, { boundMs: 50 });
          check(timed, "s3::service_error", { code: "timeout", operation: "upload_write" });
          expect(Date.now() - started).toBeLessThan(2000);
          // Poisoned: every reuse answers upload_closed.
          check(await s3.uploadWrite(upload, buf("chunk-2")), "s3::upload_closed", {
            operation: "upload_write",
            state: "discarded",
          });
          check(await s3.uploadFinish(upload), "s3::upload_closed", {
            operation: "upload_finish",
            state: "discarded",
          });
          // Discard short-circuits promptly with no new wire calls
          // (E08 D3 shape: upload_closed on the dead handle).
          const opsBefore = tap.ops.length;
          const discardStarted = Date.now();
          check(await s3.discardUpload(upload), "s3::upload_closed", {
            operation: "discard_upload",
            state: "discarded",
          });
          expect(Date.now() - discardStarted).toBeLessThan(1500);
          expect(tap.ops.length).toBe(opsBefore);
          return success(undefined);
        }),
      );
      expect(outcome.cleanupFailed).toBe(false);
      expect(outcome.completion.kind).toBe("ok");
      expect(scope.collected.escalations).toHaveLength(1);
      // Recovery: the service drops the hung connection, the write
      // rejects, the deferred scrub converges, and the late record lands.
      tap.setRules([]);
      const dropped = tap.dropHeld();
      expect(dropped).toBeGreaterThanOrEqual(1);
      expect(await awaitLates(scope, 1, 15000)).toBe(true);
      // H2-qualified: a reset part upload is not retried.
      expect(scope.collected.lates[0]).toMatchObject({ settled: "rejected" });
      await runOwnedRoot(async () => {
        const direct = await openLive();
        expect(value(await s3.exists(direct, key))).toBe(false);
        return success(undefined);
      });
      const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
      expect(orphans).toEqual([]);
      console.log(JSON.stringify({ leg: "R1-4", dropped, ops: opTriples(tap) }));
      expect(tap.desyncs()).toBe(0);
    } finally {
      tap.close();
    }
  },
  60000,
);

live(
  "R1-5: write_stream post-sink expiry converges deferred on recovery",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      blackholeAll(tap);
      const key = `${PREFIX}deferred.bin`;
      const scope = createRequestScope();
      const outcome = await runWithRequestScope(scope, () =>
        runOwnedRoot(async () => {
          const client = await openTap(tap);
          const stream = new ReadableStream<Uint8Array>({
            start(controller) {
              controller.enqueue(new TextEncoder().encode("aaa"));
              controller.enqueue(new TextEncoder().encode("bbb"));
              controller.close();
            },
          });
          const reader = registerReader(
            openByteCell(stream, 65536n),
            fail,
            identity("can.std.stream@1::close_failed"),
            { scopeManaged: true },
          );
          const started = Date.now();
          // Generous deadline: the 100ms trailing bound drives expiry
          // inside the hung first write (post-sink).
          const timed = await s3.writeStream(
            client,
            key,
            reader,
            writeOpts(none()),
            1024n,
            30000n,
            undefined,
            {
              boundMs: 100,
            },
          );
          check(timed, "s3::service_error", { code: "timeout", operation: "write_stream" });
          expect(Date.now() - started).toBeLessThan(2000);
          return success(undefined);
        }),
      );
      expect(outcome.cleanupFailed).toBe(false);
      expect(outcome.completion.kind).toBe("ok");
      expect(scope.collected.escalations).toHaveLength(1);
      tap.setRules([]);
      const dropped = tap.dropHeld();
      expect(dropped).toBeGreaterThanOrEqual(1);
      // The hung flush settles on recovery (resolved via retry or
      // rejected — the native retry kind is not R1's contract); the
      // late record lands either way and the deferred scrub converges.
      expect(await awaitLates(scope, 1, 15000)).toBe(true);
      expect(scope.collected.lates).toHaveLength(1);
      await runOwnedRoot(async () => {
        const direct = await openLive();
        expect(value(await s3.exists(direct, key))).toBe(false);
        return success(undefined);
      });
      const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
      expect(orphans).toEqual([]);
      console.log(JSON.stringify({ leg: "R1-5", dropped, ops: opTriples(tap) }));
      expect(tap.desyncs()).toBe(0);
    } finally {
      tap.close();
    }
  },
  60000,
);

live(
  "R1-6: healthy awaits settle under generous bounds",
  async () => {
    const key = `${PREFIX}roundtrip.bin`;
    const upKey = `${PREFIX}roundtrip-up.bin`;
    const streamKey = `${PREFIX}roundtrip-stream.bin`;
    const outcome = await runOwnedRoot(async () => {
      const client = await openLive();
      const bounds: S3Bounds = { boundMs: 5000 };
      value(
        await s3.writeBytes(
          client,
          key,
          buf("roundtrip-bytes"),
          writeOpts(none()),
          undefined,
          bounds,
        ),
      );
      expect(value(await s3.exists(client, key, undefined, bounds))).toBe(true);
      const meta = value(await s3.stat(client, key, undefined, bounds));
      expect(meta).toBeDefined();
      value(await s3.readBytes(client, key, 1024n, undefined, bounds));
      value(await s3.readRange(client, key, 0n, 4n, undefined, bounds));
      value(await s3.readStream(client, key, 1024n, undefined, bounds));
      const page = value(
        await s3.list(client, listOpts(PREFIX, 10n, none(), none()), undefined, bounds),
      );
      expect(page).toBeDefined();
      const upload = value(await s3.beginUpload(client, upKey, uploadOpts(none(), none())));
      expect(value(await s3.uploadWrite(upload, buf("up-chunk"), undefined, bounds))).toBe(8n);
      value(await s3.uploadFinish(upload, undefined, bounds));
      expect(value(await s3.exists(client, upKey, undefined, bounds))).toBe(true);
      const stream = new ReadableStream<Uint8Array>({
        start(controller) {
          controller.enqueue(new TextEncoder().encode("ss"));
          controller.close();
        },
      });
      const reader = registerReader(
        openByteCell(stream, 65536n),
        fail,
        identity("can.std.stream@1::close_failed"),
        { scopeManaged: true },
      );
      value(
        await s3.writeStream(
          client,
          streamKey,
          reader,
          writeOpts(none()),
          1024n,
          5000n,
          undefined,
          bounds,
        ),
      );
      expect(value(await s3.exists(client, streamKey, undefined, bounds))).toBe(true);
      value(await s3.remove(client, key, undefined, bounds));
      value(await s3.remove(client, upKey, undefined, bounds));
      value(await s3.remove(client, streamKey, undefined, bounds));
      expect(value(await s3.exists(client, key, undefined, bounds))).toBe(false);
      return success(undefined);
    });
    expect(outcome.cleanupFailed).toBe(false);
    expect(outcome.completion.kind).toBe("ok");
  },
  60000,
);
