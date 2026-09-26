// E09: W5 storage qualification on the final S3 adapters.
//
// S3-PROTOCOL scope: every live leg runs against the provisioned
// MinIO endpoint (never AWS-real) under an isolated `s3e09/` prefix.
// Without the endpoint/credentials the live legs skip visibly.
//
// Branch posture (inherited, not relitigated):
//   - X-R15-1 NEGATIVE → the O2 destructive branch: `cancel_upload`
//     is gone; `discard_upload` completes-then-deletes with honest
//     cleanup-failure reporting. Per the X-R15-2 experiment row, W5
//     asserts name removal, explicit call sites and honest outcomes —
//     never preservation.
//   - X-R15-3 NEGATIVE → hung writer/flush/end/stat awaits admit no
//     bound. W5-S5 pins that negative with the exact evidence and the
//     verdict stays BLOCKED: it must never become an accepted pass.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue, operation } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createS3 } from "../platform/s3.ts";
import { openByteCell } from "../transport/stream/readable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { value, success, failure, type Completion } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { ownBytes } from "../bytes.ts";
import { runOwnedRoot } from "../owner.ts";
import { startTap, listUploads, abortUpload, type Tap, type S3Identity } from "./s3-e07-tap.ts";

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
const bigBytes = (size: number): Uint8Array => {
  const out = new Uint8Array(size);
  for (let i = 0; i < out.length; i++) out[i] = i % 251;
  return out;
};
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
async function metaOf(meta: unknown): Promise<{ size: bigint; etag: string }> {
  return {
    size: dataProperty(meta, "size") as bigint,
    etag: dataProperty(meta, "etag") as string,
  };
}

const ENDPOINT = process.env["CAN_TEST_S3_ENDPOINT"],
  ACCESS = process.env["CAN_TEST_S3_ACCESS_KEY"],
  SECRET = process.env["CAN_TEST_S3_SECRET_KEY"];
const BUCKET = process.env["CAN_TEST_S3_BUCKET"] ?? "can-b1-10";
const REGION = process.env["CAN_TEST_S3_REGION"] ?? "us-east-1";
const live = test.skipIf(ENDPOINT === undefined || ACCESS === undefined || SECRET === undefined);
const PREFIX = `s3e09/e09harness/${Date.now().toString(36)}/`;

const ident = (): S3Identity => ({
  endpoint: ENDPOINT!,
  region: REGION,
  bucket: BUCKET,
  accessKey: ACCESS!,
  secretKey: SECRET!,
});
const openLive = async () =>
  value(await s3.clientOpen(ENDPOINT!, REGION, BUCKET, ACCESS!, SECRET!)) as object;
const openTap = async (tap: Tap) =>
  value(
    await s3.clientOpen(`http://127.0.0.1:${tap.port}`, REGION, BUCKET, ACCESS!, SECRET!),
  ) as object;
const native = (endpoint: string): InstanceType<typeof Bun.S3Client> =>
  new Bun.S3Client({
    endpoint,
    region: REGION,
    bucket: BUCKET,
    accessKeyId: ACCESS!,
    secretAccessKey: SECRET!,
  });
const upstreamOf = (): { host: string; port: number } => {
  const url = new URL(ENDPOINT!);
  return { host: url.hostname, port: Number(url.port) };
};
const opTriples = (tap: Tap): string[] =>
  tap.ops.map((entry) => `${entry.method} ${entry.op} ${entry.verdict}`);

type RaceWon = Readonly<
  | { status: "resolved"; value: unknown }
  | { status: "rejected"; name: string; code: string }
  | { status: "watchdog" }
>;

async function raceWatch(promise: Promise<unknown>, ms: number): Promise<RaceWon> {
  return Promise.race([
    promise.then(
      (val) => ({ status: "resolved", value: val }) as const,
      (cause: unknown) => {
        const code = (cause as { code?: unknown }).code;
        return {
          status: "rejected",
          name: cause instanceof Error ? cause.name : typeof cause,
          code: typeof code === "string" ? code : "",
        } as const;
      },
    ),
    Bun.sleep(ms).then(() => ({ status: "watchdog" }) as const),
  ]);
}

// --- X-R15-2, O2 branch: name removal, explicit sites, honest outcomes ---

test("W5-S1: cancel_upload is absent; discard_upload carries the destructive contract", () => {
  expect(() => operation("s3::cancel_upload")).toThrow();
  const names = catalogue.operations.map((item): string => item.name);
  const identities = catalogue.operations.map((item): string => item.identity);
  expect(names.some((name) => name === "s3::cancel_upload")).toBe(false);
  expect(identities.some((id) => id === "can.std.s3@1::cancel_upload")).toBe(false);
  const found = operation("s3::discard_upload");
  expect(found.identity).toBe("can.std.s3@1::discard_upload");
  expect(found.inputs).toEqual([{ name: "upload", type: "s3::upload" }]);
  expect(found.result).toBe("void");
  expect([...found.emits].sort()).toEqual([
    "s3::access_denied",
    "s3::service_error",
    "s3::upload_closed",
  ]);
  for (const phrase of [
    "Destructively",
    "transiently visible",
    "pre-existing key",
    "service errors",
    "operator abort recipe",
  ]) {
    expect(found.lowering.adapter).toContain(phrase);
  }
});

live(
  "W5-S2: discard-while-replacing destroys the seeded bytes+etag (never preserved)",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}s2.bin`;
        const before = await metaOf(
          value(await s3.writeBytes(client, key, buf("original-bytes"), writeOpts(none()))),
        );
        expect(before.size).toBe(14n);
        expect(before.etag.length).toBeGreaterThan(0);
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        expect(value(await s3.uploadWrite(upload, buf("replacement")))).toBe(11n);
        tap.clearOps();
        expect(value(await s3.discardUpload(upload))).toBe(undefined);
        expect(tap.desyncs()).toBe(0);
        console.log(JSON.stringify({ leg: "W5-S2", ops: opTriples(tap) }));
        // The scrub completes the replacement, then deletes the key:
        // the seeded bytes+etag are gone, not preserved.
        expect(opTriples(tap)).toEqual([
          "PUT put-object forwarded",
          "DELETE delete-object forwarded",
        ]);
        expect(value(await s3.exists(client, key))).toBe(false);
        check(await s3.readBytes(client, key, 64n), "s3::missing_key", { key });
      });
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "W5-S3: a concurrent observer never sees partial bytes during discard-while-replacing",
  async () => {
    await owned(async () => {
      const client = await openLive();
      const observer = native(ENDPOINT!);
      const originalTag = `bytes:14:${Buffer.from("original-bytes").toString("hex").slice(0, 16)}`;
      const replacementTag = `bytes:11:${Buffer.from("replacement").toString("hex").slice(0, 16)}`;
      for (let run = 1; run <= 2; run++) {
        const key = `${PREFIX}s3-${run}.bin`;
        await s3.writeBytes(client, key, buf("original-bytes"), writeOpts(none()));
        const seen = new Set<string>();
        let stop = false;
        const poller = (async () => {
          for (;;) {
            if (stop) return;
            try {
              const body = await observer.file(key).bytes();
              seen.add(`bytes:${body.length}:${Buffer.from(body).toString("hex").slice(0, 16)}`);
            } catch (cause) {
              const code = (cause as { code?: unknown }).code;
              seen.add(`missing:${typeof code === "string" ? code : "?"}`);
            }
            await Bun.sleep(2);
          }
        })();
        try {
          const upload = value(
            await s3.beginUpload(client, key, uploadOpts(none(), none())),
          ) as object;
          expect(value(await s3.uploadWrite(upload, buf("replacement")))).toBe(11n);
          expect(value(await s3.discardUpload(upload))).toBe(undefined);
        } finally {
          stop = true;
          await poller;
        }
        await Bun.sleep(500);
        console.log(JSON.stringify({ leg: "W5-S3", run, seen: [...seen].sort() }));
        // Only whole states are ever visible: the original bytes, the
        // transient replacement bytes, or missing. Partial/mixed bytes
        // would fail here.
        for (const tag of seen) {
          expect([originalTag, replacementTag, "missing:NoSuchKey"].includes(tag)).toBe(true);
        }
        expect(value(await s3.exists(client, key))).toBe(false);
        try {
          value(await s3.remove(client, key));
        } catch {
          /* already absent */
        }
      }
    });
  },
  30000,
);

// --- Orphan/service failures: honest error plus the operator recipe ---

live(
  "W5-S4: a failed completion surfaces service_error and the orphan aborts out-of-band",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "fault500", match: (op) => op === "complete-mpu" }]);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}s4.bin`;
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        expect(value(await s3.uploadWrite(upload, ownBytes(bigBytes(6 * 1024 * 1024))))).toBe(
          BigInt(6 * 1024 * 1024),
        );
        tap.clearOps();
        // The first failed cleanup await surfaces: the caller learns
        // the scrub failed instead of reading a silent success.
        check(await s3.discardUpload(upload), "s3::service_error", {
          code: "InternalError",
          operation: "discard_upload",
        });
        expect(tap.desyncs()).toBe(0);
        const orphans = (await listUploads(ident())).filter((entry) => entry.key === key);
        console.log(
          JSON.stringify({ leg: "W5-S4", ops: opTriples(tap), orphaned: orphans.length }),
        );
        // Failed completion strands real bytes server-side (the client
        // never attempts an abort on this path).
        expect(orphans.length).toBe(1);
        // Operator recipe: list unfiltered + client-side prefix filter
        // + abort each + re-list to verify (never trust the abort
        // status on MinIO — the re-list is the proof).
        for (const orphan of orphans) {
          await abortUpload(ident(), orphan.key, orphan.uploadId);
        }
        const again = (await listUploads(ident())).filter((entry) => entry.key === key);
        expect(again).toEqual([]);
        expect(value(await s3.exists(client, key))).toBe(false);
      });
    } finally {
      tap.close();
    }
  },
  30000,
);

// --- X-R15-3: BLOCKED. Hung awaits admit no bound. ---

test("W5-S5a: write_stream documents only the between-awaits deadline", () => {
  const pump = operation("s3::write_stream");
  expect(pump.lowering.adapter).toContain("only between awaits");
  expect(pump.lowering.adapter).toContain("never a hung await");
  expect(pump.lowering.adapter).toContain("discards the upload destructively");
});

live(
  "W5-S5b: BLOCKED — a hung source.read hangs write_stream past its deadline",
  async () => {
    // BLOCKED (X-R15-3 negative, must never become an accepted pass):
    // this leg pins the exact negative — the between-awaits deadline
    // never fires while the first read is pending — and the evidence
    // record carries the BLOCKED verdict. E07 H2–H5 pin the same
    // negative for hung flush/end/stat/abandonment on these final
    // adapters; see the E09 evidence record.
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      const client = await openTap(tap);
      const key = `${PREFIX}s5b.bin`;
      const hung = new ReadableStream<Uint8Array>({ pull() {} });
      // The owned root is deliberately abandoned at the watchdog: the
      // hung read holds only a pending promise (no sink exists yet —
      // creation is lazy — so nothing pins the loop for later legs).
      const root = runOwnedRoot(async () => {
        const reader = registerReader(
          openByteCell(hung, 65536n),
          fail,
          identity("can.std.stream@1::close_failed"),
          { scopeManaged: true },
        );
        return s3.writeStream(client, key, reader, writeOpts(none()), 1024n, 100n);
      });
      const outcome = await raceWatch(root, 3000);
      console.log(JSON.stringify({ leg: "W5-S5b", outcome: outcome.status, ops: opTriples(tap) }));
      expect(tap.desyncs()).toBe(0);
      expect(outcome.status).toBe("watchdog");
      expect(tap.ops).toEqual([]);
    } finally {
      tap.close();
    }
  },
  20000,
);
