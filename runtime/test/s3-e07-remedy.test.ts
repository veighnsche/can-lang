// E07 (X-R15-1/X-R15-3): isolated S3 remedy experiments.
//
// S3-PROTOCOL scope: every live leg runs against the provisioned
// MinIO endpoint (never AWS-real); the evidence record carries that
// caveat and flags legs needing AWS-real semantics. Without the
// endpoint/credentials legs skip visibly instead of passing vacuously.
//
// Two independent branches, both outcomes honored per branch:
//   - Cancel (X-R15-1): does end(Error) release the sink without
//     publishing, and what does each cleanup-await failure do?
//   - Deadline (X-R15-3): which hung awaits (reader, writer, flush,
//     end, stat) admit a bounded return, and at what pin/orphan cost?
// Legs pin the QUALIFIED behavior: if a future Bun changes the
// native surface, the legs fail loudly and E07 must be re-qualified
// before E08 maps any contract onto it.
//
// E08 rebase: both branches went negative, so the adapter legs now
// call discard_upload (cancel_upload is removed from the catalogue)
// and F1/F2/F4/F5 assert the honest cleanup-failure posture
// (service_error) instead of silent resolve. Wire observations and
// `leg:` tags are unchanged from the E07 record.
import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { spawn } from "node:child_process";
import { fileURLToPath } from "node:url";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createS3 } from "../platform/s3.ts";
import { openByteCell } from "../transport/stream/readable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { value, success, failure, type Completion } from "../completion.ts";
import { record, dataProperty } from "../data.ts";
import { ownBytes, copyBytes } from "../bytes.ts";
import { runOwnedRoot } from "../owner.ts";
import {
  startTap,
  listUploads,
  listParts,
  abortUpload,
  type S3Identity,
  type Tap,
} from "./s3-e07-tap.ts";

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
const METADATA = "can.std.s3@1::metadata";
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
  entry: "can.std.s3@1::entry",
  page: "can.std.s3@1::page",
  info: "can.std.s3@1::presigned_info",
  methodGet: "can.std.s3@1::method_get",
  methodPut: "can.std.s3@1::method_put",
  methodDelete: "can.std.s3@1::method_delete",
  methodHead: "can.std.s3@1::method_head",
  someText: SOME_TEXT,
  someInt: SOME_INT,
  someContinuation: SOME_CONT,
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
const textOf = (body: unknown) => new TextDecoder().decode(copyBytes(body, origin));
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

const ENDPOINT = process.env["CAN_TEST_S3_ENDPOINT"],
  ACCESS = process.env["CAN_TEST_S3_ACCESS_KEY"],
  SECRET = process.env["CAN_TEST_S3_SECRET_KEY"];
const BUCKET = process.env["CAN_TEST_S3_BUCKET"] ?? "can-b1-10";
const REGION = process.env["CAN_TEST_S3_REGION"] ?? "us-east-1";
const live = test.skipIf(ENDPOINT === undefined || ACCESS === undefined || SECRET === undefined);
const PREFIX = `s3e07/e07harness/${Date.now().toString(36)}/`;

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

const PROBE_FILE = fileURLToPath(new URL("./s3-e07-pin-probe.ts", import.meta.url));

type PinOutcome = Readonly<{
  mode: string;
  exited: boolean;
  code: number | null;
  signal: string | null;
  elapsedMs: number;
  firstLine: string;
  errors: string;
}>;

async function runPinChild(mode: string, watchdogMs: number): Promise<PinOutcome> {
  const started = Date.now();
  const child = spawn(process.execPath, ["--no-install", PROBE_FILE, mode], {
    stdio: ["ignore", "pipe", "pipe"],
  });
  let output = "",
    errors = "";
  child.stdout.on("data", (chunk) => (output += chunk));
  child.stderr.on("data", (chunk) => (errors += chunk));
  const deadline = setTimeout(() => child.kill("SIGKILL"), watchdogMs);
  try {
    const status = await new Promise<{ code: number | null; signal: string | null }>(
      (resolve, reject) => {
        child.once("error", reject);
        child.once("exit", (code, signal) => resolve({ code, signal }));
      },
    );
    const elapsedMs = Date.now() - started;
    return {
      mode,
      exited: status.signal === null,
      code: status.code,
      signal: status.signal,
      elapsedMs,
      firstLine: output.split("\n")[0] ?? "",
      errors,
    };
  } finally {
    clearTimeout(deadline);
    child.kill("SIGKILL");
  }
}

async function sweepKeys(client: InstanceType<typeof Bun.S3Client>, keys: string[]): Promise<void> {
  for (const key of keys) {
    try {
      await client.file(key).delete();
    } catch {
      /* best effort */
    }
  }
}

// Abort every pending upload under keyPrefix; returns aborted keys.
async function sweepOrphans(keyPrefix: string): Promise<string[]> {
  const aborted: string[] = [];
  const pending = await listUploads(ident());
  for (const upload of pending) {
    if (!upload.key.startsWith(keyPrefix)) continue;
    const status = await abortUpload(ident(), upload.key, upload.uploadId);
    if (status === 204) aborted.push(upload.key);
  }
  return aborted;
}

const opTriples = (tap: Tap): string[] =>
  tap.ops.map((entry) => `${entry.method} ${entry.op} ${entry.verdict}`);
const countOp = (tap: Tap, op: string, verdict?: string): number =>
  tap.ops.filter((entry) => entry.op === op && (verdict === undefined || entry.verdict === verdict))
    .length;

// --- Cancel branch (X-R15-1) ---

live(
  "X-R15-1 C0-small: discard_upload after first byte deletes a pre-existing key",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}c0-small.bin`;
        const before = await metaOf(
          value(await s3.writeBytes(client, key, buf("original-bytes"), writeOpts(none()))),
        );
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        expect(value(await s3.uploadWrite(upload, buf("replacement")))).toBe(11n);
        tap.clearOps();
        expect(value(await s3.discardUpload(upload))).toBe(undefined);
        expect(tap.desyncs()).toBe(0);
        console.log(JSON.stringify({ leg: "C0-small", ops: opTriples(tap) }));
        // The scrub completes the replacement, then deletes the key:
        // the original is gone, not preserved.
        expect(opTriples(tap)).toEqual([
          "PUT put-object forwarded",
          "DELETE delete-object forwarded",
        ]);
        expect(value(await s3.exists(client, key))).toBe(false);
        check(await s3.readBytes(client, key, 64n), "s3::missing_key", { key });
        expect(before.size).toBe(14n);
      });
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-1 C0-multipart: discard_upload after parts deletes a pre-existing key",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}c0-multi.bin`;
        await s3.writeBytes(client, key, buf("original-bytes"), writeOpts(none()));
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        expect(value(await s3.uploadWrite(upload, ownBytes(bigBytes(6 * 1024 * 1024))))).toBe(
          BigInt(6 * 1024 * 1024),
        );
        tap.clearOps();
        expect(value(await s3.discardUpload(upload))).toBe(undefined);
        expect(tap.desyncs()).toBe(0);
        console.log(JSON.stringify({ leg: "C0-multipart", ops: opTriples(tap) }));
        expect(opTriples(tap)).toEqual([
          "PUT upload-part forwarded",
          "POST complete-mpu forwarded",
          "DELETE delete-object forwarded",
        ]);
        expect(value(await s3.exists(client, key))).toBe(false);
        const orphans = await listUploads(ident());
        expect(orphans.filter((upload) => upload.key === key)).toEqual([]);
      });
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-1 C1: end(Error) completes a small buffered write",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      const client = native(`http://127.0.0.1:${tap.port}`);
      const key = `${PREFIX}c1.bin`;
      try {
        const writer = client.file(key).writer({ type: "application/octet-stream" });
        expect(String(await writer.write(new TextEncoder().encode("hello-e07")))).toBe("9");
        tap.clearOps();
        // The Error argument neither rejects nor aborts: the PUT is
        // issued exactly as with a bare end().
        expect(String(await writer.end(new Error("e07-abort")))).toBe("9");
        expect(tap.desyncs()).toBe(0);
        console.log(JSON.stringify({ leg: "C1", ops: opTriples(tap) }));
        expect(opTriples(tap)).toEqual(["PUT put-object forwarded"]);
        const stat = await client.file(key).stat();
        expect(stat.size).toBe(9);
      } finally {
        await sweepKeys(client, [key]);
      }
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-1 C2: end(Error) completes a multipart upload without aborting",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      const client = native(`http://127.0.0.1:${tap.port}`);
      const key = `${PREFIX}c2.bin`;
      try {
        const writer = client.file(key).writer({ type: "application/octet-stream" });
        expect(String(await writer.write(bigBytes(6 * 1024 * 1024)))).toBe(String(6 * 1024 * 1024));
        expect(String(await writer.flush())).toBe(String(5 * 1024 * 1024));
        tap.clearOps();
        expect(String(await writer.end(new Error("e07-abort")))).toBe(String(6 * 1024 * 1024));
        expect(tap.desyncs()).toBe(0);
        console.log(JSON.stringify({ leg: "C2", ops: opTriples(tap) }));
        expect(opTriples(tap)).toEqual([
          "PUT upload-part forwarded",
          "POST complete-mpu forwarded",
        ]);
        const stat = await client.file(key).stat();
        expect(stat.size).toBe(6 * 1024 * 1024);
        expect(stat.etag.endsWith('-2"')).toBe(true);
        const orphans = await listUploads(ident());
        expect(orphans.filter((upload) => upload.key === key)).toEqual([]);
      } finally {
        await sweepKeys(client, [key]);
      }
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-1 C3: end(Error) while replacing overwrites the original",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      const direct = native(ENDPOINT!);
      const key = `${PREFIX}c3.bin`;
      try {
        await direct.file(key).write("original-bytes");
        const before = await direct.file(key).stat();
        const client = native(`http://127.0.0.1:${tap.port}`);
        const writer = client.file(key).writer({ type: "application/octet-stream" });
        await writer.write(new TextEncoder().encode("replacement!!"));
        await writer.end(new Error("e07-abort"));
        const after = await direct.file(key).stat();
        const body = await direct.file(key).text();
        console.log(
          JSON.stringify({
            leg: "C3",
            beforeEtag: before.etag,
            afterEtag: after.etag,
            ops: opTriples(tap),
          }),
        );
        // Preservation fails: the original bytes are replaced.
        expect(body).toBe("replacement!!");
        expect(after.etag).not.toBe(before.etag);
      } finally {
        await sweepKeys(direct, [key]);
      }
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-1 C4: observer never sees partial bytes during discard-while-replacing",
  async () => {
    await owned(async () => {
      const client = await openLive();
      const observer = native(ENDPOINT!);
      for (let run = 1; run <= 3; run++) {
        const key = `${PREFIX}c4-${run}.bin`;
        const originalTag = `bytes:14:${Buffer.from("original-bytes").toString("hex").slice(0, 16)}`;
        const replacementTag = `bytes:11:${Buffer.from("replacement").toString("hex").slice(0, 16)}`;
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
        console.log(JSON.stringify({ leg: "C4", run, seen: [...seen].sort() }));
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

live(
  "X-R15-1 C5: ended sinks release the loop, un-ended writers pin it",
  async () => {
    const direct = native(ENDPOINT!);
    const pinKeys: string[] = [];
    const ended = await runPinChild("bare-end", 6000);
    const errored = await runPinChild("error-end", 6000);
    const writeAbandon = await runPinChild("write-abandon", 6000);
    const writerAbandon = await runPinChild("writer-abandon", 6000);
    console.log(JSON.stringify({ leg: "C5", ended, errored, writeAbandon, writerAbandon }));
    for (const outcome of [ended, errored]) {
      expect(outcome.exited).toBe(true);
      expect(outcome.code).toBe(0);
      expect(outcome.errors).toBe("");
      const parsed = JSON.parse(outcome.firstLine) as { end?: string; key?: string };
      expect(parsed.end).toBe("9");
      if (parsed.key !== undefined) pinKeys.push(parsed.key);
    }
    // Pinned means the work finished (first line printed) but the
    // loop never released: the parent SIGKILLs at the watchdog.
    for (const outcome of [writeAbandon, writerAbandon]) {
      expect(outcome.exited).toBe(false);
      expect(outcome.signal).toBe("SIGKILL");
      expect(outcome.firstLine).toContain("abandoned");
    }
    await sweepKeys(direct, pinKeys);
  },
  60000,
);

live(
  "X-R15-1 C6: loopback latency samples for end, discard and stat",
  async () => {
    const direct = native(ENDPOINT!);
    await owned(async () => {
      const client = await openLive();
      const samples: Record<string, number[]> = {
        writeBytes: [],
        discardUpload: [],
        nativeEndError: [],
        stat: [],
      };
      for (let i = 0; i < 3; i++) {
        const key = `${PREFIX}c6-${i}.bin`;
        let started = Date.now();
        await s3.writeBytes(client, key, buf("sample-bytes"), writeOpts(none()));
        samples["writeBytes"]!.push(Date.now() - started);
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        await s3.uploadWrite(upload, buf("sample-bytes"));
        started = Date.now();
        await s3.discardUpload(upload);
        samples["discardUpload"]!.push(Date.now() - started);
        const writer = direct.file(`${PREFIX}c6e-${i}.bin`).writer();
        await writer.write(new TextEncoder().encode("sample-bytes"));
        started = Date.now();
        await writer.end(new Error("e07-abort"));
        samples["nativeEndError"]!.push(Date.now() - started);
        started = Date.now();
        await s3.stat(client, `${PREFIX}c6e-${i}.bin`);
        samples["stat"]!.push(Date.now() - started);
        for (const series of Object.values(samples))
          expect(series[series.length - 1]!).toBeLessThan(30000);
        try {
          value(await s3.remove(client, `${PREFIX}c6e-${i}.bin`));
        } catch {
          /* best effort */
        }
      }
      console.log(JSON.stringify({ leg: "C6", loopbackMs: samples }));
    });
  },
  30000,
);

async function metaOf(meta: unknown): Promise<{ size: bigint; etag: string }> {
  return {
    size: dataProperty(meta, "size") as bigint,
    etag: dataProperty(meta, "etag") as string,
  };
}

// --- Cleanup-await failure injection (one leg per scrub await) ---

live(
  "X-R15-1 F1: injected complete-mpu failure during discard fails honestly",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "fault500", match: (op) => op === "complete-mpu" }]);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}f1.bin`;
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        expect(value(await s3.uploadWrite(upload, ownBytes(bigBytes(6 * 1024 * 1024))))).toBe(
          BigInt(6 * 1024 * 1024),
        );
        tap.clearOps();
        // E08: the failed completion surfaces (first failed await)
        // and the scrub still deletes: the key never materializes
        // but discard reports the cleanup failure.
        check(await s3.discardUpload(upload), "s3::service_error", {
          code: "InternalError",
          operation: "discard_upload",
        });
        expect(tap.desyncs()).toBe(0);
        const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
        console.log(
          JSON.stringify({
            leg: "F1",
            ops: opTriples(tap),
            completeAttempts: countOp(tap, "complete-mpu"),
            abortIssued: countOp(tap, "abort-mpu"),
            orphaned: orphans.length,
          }),
        );
        // The client retries the failed completion (1 + 3, each on
        // a fresh connection) and attempts no abort, so the MPU —
        // parts included — is orphaned server-side.
        expect(countOp(tap, "complete-mpu")).toBe(4);
        expect(countOp(tap, "abort-mpu")).toBe(0);
        expect(orphans.length).toBe(1);
        const held = await listParts(ident(), key, orphans[0]!.uploadId);
        console.log(JSON.stringify({ leg: "F1-parts", status: held.status, parts: held.parts }));
        expect(held.status).toBe(200);
        expect(held.parts).toEqual([
          { number: "1", size: 5 * 1024 * 1024 },
          { number: "2", size: 1024 * 1024 },
        ]);
        expect(value(await s3.exists(client, key))).toBe(false);
        for (const orphan of orphans) await abortUpload(ident(), orphan.key, orphan.uploadId);
      });
    } finally {
      tap.close();
    }
  },
  30000,
);

live(
  "X-R15-1 F2: injected delete-object failure leaves the completed key",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "fault500", match: (op) => op === "delete-object" }]);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}f2.bin`;
        try {
          const upload = value(
            await s3.beginUpload(client, key, uploadOpts(none(), none())),
          ) as object;
          expect(value(await s3.uploadWrite(upload, buf("replacement")))).toBe(11n);
          tap.clearOps();
          // E08: the failed delete surfaces: the transient
          // completion stays in place permanently and discard
          // reports the cleanup failure.
          check(await s3.discardUpload(upload), "s3::service_error", {
            code: "InternalError",
            operation: "discard_upload",
          });
          expect(tap.desyncs()).toBe(0);
          console.log(JSON.stringify({ leg: "F2", ops: opTriples(tap) }));
          expect(opTriples(tap)).toEqual([
            "PUT put-object forwarded",
            "DELETE delete-object fault500",
          ]);
          expect(value(await s3.exists(client, key))).toBe(true);
          expect(textOf(value(await s3.readBytes(client, key, 64n)))).toBe("replacement");
        } finally {
          // Clear the delete fault first: cleanup through the
          // faulted tap would fail and leak the key.
          tap.setRules([]);
          try {
            value(await s3.remove(client, key));
          } catch {
            /* best effort */
          }
        }
      });
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-1 F3: injected upload-part failure surfaces then discards",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "fault500", match: (op) => op === "upload-part" }]);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}f3.bin`;
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        const written = await s3.uploadWrite(upload, ownBytes(bigBytes(6 * 1024 * 1024)));
        check(written, "s3::service_error", { code: "InternalError", operation: "upload_write" });
        console.log(JSON.stringify({ leg: "F3-write", ops: opTriples(tap) }));
        // 1 initial + 3 retries, each on a fresh connection.
        expect(countOp(tap, "upload-part")).toBe(4);
        tap.clearOps();
        expect(value(await s3.discardUpload(upload))).toBe(undefined);
        expect(tap.desyncs()).toBe(0);
        const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
        console.log(
          JSON.stringify({
            leg: "F3-cancel",
            ops: opTriples(tap),
            abortIssued: countOp(tap, "abort-mpu"),
            orphaned: orphans.length,
          }),
        );
        // The scrub end() is a no-op after part failure, but the
        // scrub delete aborts the tracked MPU first (plus ~3
        // background aborts within a second): no orphan accumulates.
        // The background burst races the leg window, so only the
        // delete-coupled abort (deterministic) is pinned by count.
        expect(countOp(tap, "abort-mpu")).toBeGreaterThanOrEqual(1);
        expect(countOp(tap, "delete-object")).toBe(1);
        expect(orphans.length).toBe(0);
        expect(value(await s3.exists(client, key))).toBe(false);
        for (const orphan of orphans) await abortUpload(ident(), orphan.key, orphan.uploadId);
      });
    } finally {
      tap.close();
    }
  },
  30000,
);

live(
  "X-R15-1 F4: failed complete leaves an orphan; the client attempts no abort",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([
      { action: "fault500", match: (op) => op === "complete-mpu" },
      { action: "fault500", match: (op) => op === "abort-mpu" },
    ]);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}f4.bin`;
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        expect(value(await s3.uploadWrite(upload, ownBytes(bigBytes(6 * 1024 * 1024))))).toBe(
          BigInt(6 * 1024 * 1024),
        );
        tap.clearOps();
        // E08: the failed completion surfaces like F1; the
        // abort-fault rule still never triggers.
        check(await s3.discardUpload(upload), "s3::service_error", {
          code: "InternalError",
          operation: "discard_upload",
        });
        expect(tap.desyncs()).toBe(0);
        const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
        console.log(
          JSON.stringify({
            leg: "F4",
            ops: opTriples(tap),
            abortAttempts: countOp(tap, "abort-mpu"),
            orphaned: orphans.length,
          }),
        );
        // A failed completion accumulates server-side: the client
        // attempts no abort (the abort-fault rule below never
        // triggers), so parts — and their cost — survive the
        // failed discard until an out-of-band abort lands.
        expect(countOp(tap, "abort-mpu")).toBe(0);
        expect(orphans.length).toBe(1);
        const held = await listParts(ident(), key, orphans[0]!.uploadId);
        console.log(JSON.stringify({ leg: "F4-parts", status: held.status, parts: held.parts }));
        expect(held.status).toBe(200);
        expect(held.parts).toEqual([
          { number: "1", size: 5 * 1024 * 1024 },
          { number: "2", size: 1024 * 1024 },
        ]);
        for (const orphan of orphans) {
          expect(await abortUpload(ident(), orphan.key, orphan.uploadId)).toBe(204);
        }
        const after = (await listUploads(ident())).filter((upload) => upload.key === key);
        expect(after).toEqual([]);
      });
    } finally {
      tap.close();
    }
  },
  30000,
);

live(
  "X-R15-1 F5: injected put-object failure during small discard fails absent",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "fault500", match: (op) => op === "put-object" }]);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}f5.bin`;
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        expect(value(await s3.uploadWrite(upload, buf("replacement")))).toBe(11n);
        tap.clearOps();
        // E08: the failed end surfaces even though the follow-up
        // delete settles the key absent — the caller learns the
        // cleanup did not complete cleanly.
        check(await s3.discardUpload(upload), "s3::service_error", {
          code: "InternalError",
          operation: "discard_upload",
        });
        expect(tap.desyncs()).toBe(0);
        console.log(JSON.stringify({ leg: "F5", ops: opTriples(tap) }));
        expect(countOp(tap, "put-object")).toBe(4);
        expect(countOp(tap, "delete-object")).toBe(1);
        expect(value(await s3.exists(client, key))).toBe(false);
      });
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-1 F6: end(Error) surfaces a failed PUT as rejection",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "fault500", match: (op) => op === "put-object" }]);
    try {
      const client = native(`http://127.0.0.1:${tap.port}`);
      const key = `${PREFIX}f6.bin`;
      try {
        const writer = client.file(key).writer({ type: "application/octet-stream" });
        await writer.write(new TextEncoder().encode("hello-e07"));
        tap.clearOps();
        const outcome = await raceWatch(Promise.resolve(writer.end(new Error("e07-abort"))), 10000);
        console.log(
          JSON.stringify({ leg: "F6", outcome: describeRace(outcome), ops: opTriples(tap) }),
        );
        expect(tap.desyncs()).toBe(0);
        expect(outcome.status).toBe("rejected");
        if (outcome.status === "rejected") {
          expect(outcome.name).toBe("S3Error");
          expect(outcome.code).toBe("InternalError");
        }
        expect(countOp(tap, "put-object")).toBe(4);
        expect(await client.file(key).exists()).toBe(false);
      } finally {
        await sweepKeys(client, [key]);
      }
    } finally {
      tap.close();
    }
  },
  20000,
);

function describeRace(won: RaceWon): string {
  if (won.status === "resolved") return `resolved:${String(won.value)}`;
  if (won.status === "rejected") return `rejected:${won.name}:${won.code}`;
  return "WATCHDOG";
}

live(
  "X-R15-1 F7: faulted abort still deletes the key but orphans the MPU",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([
      { action: "fault500", match: (op) => op === "upload-part" },
      { action: "fault500", match: (op) => op === "abort-mpu" },
    ]);
    try {
      await owned(async () => {
        const client = await openTap(tap);
        const key = `${PREFIX}f7.bin`;
        const upload = value(
          await s3.beginUpload(client, key, uploadOpts(none(), none())),
        ) as object;
        check(
          await s3.uploadWrite(upload, ownBytes(bigBytes(6 * 1024 * 1024))),
          "s3::service_error",
          { code: "InternalError", operation: "upload_write" },
        );
        tap.clearOps();
        // E08 blind spot, retained: the native delete swallows its
        // faulted coupled abort and resolves, so discard resolves
        // while the MPU orphans. No in-band signal exists; the E08
        // operator recipe (list/abort/re-list) is the only detection.
        expect(value(await s3.discardUpload(upload))).toBe(undefined);
        expect(tap.desyncs()).toBe(0);
        const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
        console.log(
          JSON.stringify({
            leg: "F7",
            ops: opTriples(tap),
            orphaned: orphans.length,
          }),
        );
        // The delete-coupled abort fails fast, the key delete still
        // lands, and the MPU is orphaned: no in-band path retries
        // the abort once the service refuses it.
        expect(countOp(tap, "abort-mpu", "fault500")).toBeGreaterThanOrEqual(1);
        expect(countOp(tap, "delete-object")).toBe(1);
        expect(orphans.length).toBe(1);
        const held = await listParts(ident(), key, orphans[0]!.uploadId);
        console.log(JSON.stringify({ leg: "F7-parts", status: held.status, parts: held.parts }));
        // No part ever landed (every part PUT faulted), so the
        // orphan is an empty MPU shell: listed, but holding no bytes.
        expect(held.status).toBe(200);
        expect(held.parts).toEqual([]);
        expect(value(await s3.exists(client, key))).toBe(false);
        for (const orphan of orphans) {
          expect(await abortUpload(ident(), orphan.key, orphan.uploadId)).toBe(204);
        }
      });
    } finally {
      tap.close();
    }
  },
  30000,
);

// --- Deadline branch (X-R15-3): one leg per hung await ---

live(
  "X-R15-3 H1: hung source.read hangs write_stream past its deadline",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      const client = await openTap(tap);
      const key = `${PREFIX}h1.bin`;
      // A stream that never produces: pull returns without enqueueing.
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
      console.log(
        JSON.stringify({ leg: "H1", outcome: describeRace(outcome), ops: opTriples(tap) }),
      );
      expect(tap.desyncs()).toBe(0);
      // The between-awaits deadline never fires: no loop iteration
      // completes while the first read is pending.
      expect(outcome.status).toBe("watchdog");
      expect(tap.ops).toEqual([]);
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-3 H2: hung part upload hangs flush and end(Error) alike",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "blackhole", match: () => true }]);
    try {
      const client = native(`http://127.0.0.1:${tap.port}`);
      const key = `${PREFIX}h2.bin`;
      try {
        const writer = client.file(key).writer({ type: "application/octet-stream" });
        expect(String(await writer.write(bigBytes(6 * 1024 * 1024)))).toBe(String(6 * 1024 * 1024));
        const flushPromise = Promise.resolve(writer.flush());
        const flushOutcome = await raceWatch(flushPromise, 3000);
        const endPromise = Promise.resolve(writer.end(new Error("e07-abort")));
        const endOutcome = await raceWatch(endPromise, 5000);
        console.log(
          JSON.stringify({
            leg: "H2-hung",
            flush: describeRace(flushOutcome),
            endError: describeRace(endOutcome),
            ops: opTriples(tap),
          }),
        );
        // No client-side abort exists: end(Error) waits on the same
        // hung multipart state as the flush.
        expect(flushOutcome.status).toBe("watchdog");
        expect(endOutcome.status).toBe("watchdog");
        // Recovery: the service drops the hung connections and comes
        // back; the client retries on fresh connections and the SAME
        // pending awaits settle, so the writer ends and the pin —
        // which a settled (resolved or rejected) end always
        // releases — lets the file exit.
        tap.setRules([]);
        const dropped = tap.dropHeld();
        const flushSettled = await raceWatch(flushPromise, 15000);
        const endSettled = await raceWatch(endPromise, 15000);
        console.log(
          JSON.stringify({
            leg: "H2-recovered",
            dropped,
            flush: describeRace(flushSettled),
            end: describeRace(endSettled),
            ops: opTriples(tap),
          }),
        );
        expect(dropped).toBeGreaterThanOrEqual(1);
        // A reset connection is NOT retried (unlike a 500): both
        // awaits reject with the transport failure and no new
        // request is issued. The creation never reached the
        // service, so neither key nor orphan exists.
        expect(flushSettled.status).toBe("rejected");
        expect(endSettled.status).toBe("rejected");
        if (flushSettled.status === "rejected") {
          expect(flushSettled.name).toBe("S3Error");
          expect(flushSettled.code).toBe("ConnectionClosed");
        }
        if (endSettled.status === "rejected") {
          expect(endSettled.name).toBe("S3Error");
          expect(endSettled.code).toBe("ConnectionClosed");
        }
        expect(countOp(tap, "create-mpu")).toBe(1);
        expect(await client.file(key).exists()).toBe(false);
        const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
        expect(orphans).toEqual([]);
        expect(tap.desyncs()).toBe(0);
      } finally {
        await sweepKeys(client, [key]);
      }
    } finally {
      tap.close();
    }
  },
  40000,
);

live(
  "X-R15-3 H3: hung complete-mpu hangs end()",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    tap.setRules([{ action: "blackhole", match: (op) => op === "complete-mpu" }]);
    try {
      const client = native(`http://127.0.0.1:${tap.port}`);
      const key = `${PREFIX}h3.bin`;
      try {
        const writer = client.file(key).writer({ type: "application/octet-stream" });
        await writer.write(bigBytes(6 * 1024 * 1024));
        expect(String(await writer.flush())).toBe(String(5 * 1024 * 1024));
        tap.clearOps();
        const endPromise = Promise.resolve(writer.end());
        const endOutcome = await raceWatch(endPromise, 3000);
        console.log(
          JSON.stringify({ leg: "H3-hung", end: describeRace(endOutcome), ops: opTriples(tap) }),
        );
        expect(endOutcome.status).toBe("watchdog");
        expect(countOp(tap, "complete-mpu", "blackholed")).toBe(1);
        tap.setRules([]);
        const dropped = tap.dropHeld();
        const endSettled = await raceWatch(endPromise, 15000);
        console.log(
          JSON.stringify({
            leg: "H3-recovered",
            dropped,
            end: describeRace(endSettled),
            ops: opTriples(tap),
          }),
        );
        expect(dropped).toBe(1);
        // Unlike the dropped creation, the dropped completion IS
        // retried on a fresh connection and the upload completes.
        expect(endSettled.status).toBe("resolved");
        if (endSettled.status === "resolved") {
          expect(String(endSettled.value)).toBe(String(6 * 1024 * 1024));
        }
        expect(countOp(tap, "complete-mpu", "forwarded")).toBe(1);
        expect((await client.file(key).stat()).size).toBe(6 * 1024 * 1024);
        // The retry completes the one open MPU: no orphan remains.
        // (One full-run observation listed a leftover here that never
        // reproduced in six follow-ups; the leg pins the qualified
        // shape and fails loudly if it recurs. See E07 evidence.)
        const orphans = (await listUploads(ident())).filter((upload) => upload.key === key);
        expect(orphans).toEqual([]);
        expect(tap.desyncs()).toBe(0);
      } finally {
        await sweepKeys(client, [key]);
      }
    } finally {
      tap.close();
    }
  },
  30000,
);

live(
  "X-R15-3 H4: hung head-object hangs stat",
  async () => {
    const tap = startTap(upstreamOf().host, upstreamOf().port);
    try {
      const client = native(`http://127.0.0.1:${tap.port}`);
      const key = `${PREFIX}h4.bin`;
      try {
        await client.file(key).write("hello-e07");
        tap.setRules([{ action: "blackhole", match: (op) => op === "head-object" }]);
        tap.clearOps();
        const statOutcome = await raceWatch(client.file(key).stat(), 3000);
        console.log(
          JSON.stringify({ leg: "H4-hung", stat: describeRace(statOutcome), ops: opTriples(tap) }),
        );
        expect(statOutcome.status).toBe("watchdog");
        expect(countOp(tap, "head-object", "blackholed")).toBe(1);
        tap.setRules([]);
        const statSettled = await raceWatch(client.file(key).stat(), 10000);
        expect(statSettled.status).toBe("resolved");
        expect(tap.desyncs()).toBe(0);
      } finally {
        await sweepKeys(client, [key]);
      }
    } finally {
      tap.close();
    }
  },
  20000,
);

live(
  "X-R15-3 H5: abandoned hung sinks pin the loop; end(Error) cannot release them",
  async () => {
    const hungFlush = await runPinChild("hung-flush-abandon", 6000);
    const hungEnd = await runPinChild("hung-end-abandon", 8000);
    const hungStat = await runPinChild("hung-stat-abandon", 6000);
    const endDuringHung = await runPinChild("end-error-during-hung-flush", 9000);
    console.log(JSON.stringify({ leg: "H5", hungFlush, hungEnd, hungStat, endDuringHung }));
    for (const outcome of [hungFlush, hungEnd, hungStat]) {
      expect(outcome.exited).toBe(false);
      expect(outcome.signal).toBe("SIGKILL");
      expect(outcome.firstLine.length).toBeGreaterThan(0);
    }
    expect(hungFlush.firstLine).toContain("flush");
    expect(hungEnd.firstLine).toContain("flushed");
    expect(hungStat.firstLine).toContain("stat");
    // The in-child end(Error) race never settles either, and the
    // child stays pinned afterwards.
    expect(endDuringHung.firstLine).toContain("WATCHDOG");
    expect(endDuringHung.exited).toBe(false);
    // The hung-end child uploaded real parts through its in-process
    // tap before its completion blackholed: abort that orphan here.
    const aborted = await sweepOrphans("s3e07/e07harness/pin-");
    console.log(JSON.stringify({ leg: "H5-cleanup", aborted }));
  },
  90000,
);

live(
  "X-R15-3 Z99: harness leaves no orphaned uploads behind",
  async () => {
    const aborted = await sweepOrphans("s3e07/e07harness/");
    const remaining = (await listUploads(ident())).filter((upload) =>
      upload.key.startsWith("s3e07/e07harness/"),
    );
    console.log(JSON.stringify({ leg: "Z99", aborted, remaining }));
    expect(remaining).toEqual([]);
  },
  20000,
);
