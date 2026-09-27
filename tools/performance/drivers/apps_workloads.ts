import { createDomainRuntime, domainFailureDiagnostics } from "../../../runtime/domain.ts";
import { createFileWrites } from "../../../runtime/platform/files/write.ts";
import { ownBytes, copyBytes } from "../../../runtime/bytes.ts";
import { catalogue } from "../../../runtime/catalogue.ts";
import { createHash } from "node:crypto";
import { writeFile } from "node:fs/promises";
import { createFileReads } from "../../../runtime/platform/files/read.ts";
import {
  snapshotRequest,
  requestSnapshot,
  nativeResponse,
  ownedResponse,
  ownedJsonResponse,
  revokeRequest,
} from "../../../runtime/platform/http.ts";
import { value } from "../../../runtime/completion.ts";
import { writeFileSync } from "node:fs";
const [
  mode,
  work,
  profile,
  iterationsRaw = "1",
  warmupsRaw = "0",
  sizeRaw = "10",
  url,
  resultPath,
] = process.argv.slice(2);
const iterations = Number(iterationsRaw),
  warmups = Number(warmupsRaw),
  size = Number(sizeRaw);
const batchCount = profile === "standard" ? 7 : 3;
function grouped(raw: number[]) {
  const output: number[][] = [];
  for (let offset = 0; offset < raw.length; offset += iterations)
    output.push(raw.slice(offset, offset + iterations));
  return output;
}
function averages(raw: number[]) {
  return grouped(raw).map((batch) => batch.reduce((a, b) => a + b, 0) / iterations);
}
const digest = (kind: string, declaration: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, declaration]))
    .digest("hex");
const primitives = ["str", "int"].map((declaration) => ({
  identity: digest("primitive", declaration),
  kind: "primitive",
  declaration,
  arguments: [],
  fields: [],
  leaves: [],
  inputs: [],
  errors: [],
}));
const declarations = catalogue.errors.filter(
  (e) => e.name.startsWith("files::") || e.name === "codec::invalid_data",
);
const shapes = declarations.map((e) => ({
  identity: digest("error", e.identity),
  kind: "error",
  declaration: e.identity,
  arguments: [],
  fields: e.fields.map((f) => ({
    name: f.name,
    type: primitives[f.type === "int" ? 1 : 0]!.identity,
  })),
  leaves: [],
  inputs: [],
  errors: [],
}));
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [...primitives, ...shapes],
});
const fileError = (name: string) =>
  shapes[declarations.findIndex((e) => e.name === (name.includes("::") ? name : `files::${name}`))]!
    .identity;
const byteOrigin = { source: "apps-performance", start: 0, end: 0, invocation: [] };
function result(suite: string, cases: any[], notes: string[] = []) {
  const failed = cases.some((c) => !c.correctness.passed);
  const output = {
    schema_version: 1,
    suite,
    status: failed ? "failed" : "complete",
    ...(failed ? { reason: "Workload correctness failed; raw trial records retained." } : {}),
    cases,
    notes,
    artifacts: [],
  };
  if (resultPath) writeFileSync(resultPath, JSON.stringify(output) + "\n");
  else console.log(JSON.stringify(output));
}
function one(
  name: string,
  samples: number[],
  warmup_samples: number[],
  scope: string,
  metrics: any = {},
  unit = "ms/op",
  parameters: any = {},
) {
  return {
    name,
    unit,
    samples: averages(samples),
    warmup_samples: averages(warmup_samples),
    iterations_per_sample: iterations,
    timing_scope:
      scope +
      " Each reported sample is the mean of operations in one batch; individual operation times are retained.",
    parameters: { size, profile, ...parameters },
    correctness: { passed: true, checks: ["exact deterministic payload"] },
    metrics: {
      ...metrics,
      operation_batches_ms: grouped(samples),
      warmup_operation_batches_ms: grouped(warmup_samples),
    },
  };
}
if (mode === "server-host") {
  const text = "can-performance:" + "x".repeat(size);
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    async fetch(req) {
      if (new URL(req.url).pathname === "/ready") return new Response("ready");
      const snapshot = await snapshotRequest(req, 1048576);
      if (snapshot.kind !== "request") return new Response("rejected", { status: 400 });
      try {
        const captured = requestSnapshot(snapshot.value);
        if (captured.method === "GET") return nativeResponse(ownedResponse(200, text, false));
        if (captured.method !== "POST") return new Response("method", { status: 405 });
        const request = JSON.parse(
          new TextDecoder("utf-8", { fatal: true }).decode(copyBytes(captured.body, byteOrigin)),
        );
        const response = JSON.stringify({
          id: request.id,
          message: request.message,
          length: request.message.length,
        });
        return nativeResponse(ownedJsonResponse(200, ownBytes(new TextEncoder().encode(response))));
      } finally {
        revokeRequest(snapshot.value);
      }
    },
  });
  console.log(JSON.stringify({ url: `http://127.0.0.1:${server.port}`, text }));
  await new Promise(() => {});
} else if (mode === "server") {
  const expected = "can-performance:" + "x".repeat(size),
    cases: any[] = [];
  // Closed loop has a fixed population; open loop schedules independently and reports dropped generator work.
  for (const method of ["GET", "POST"])
    for (const scenario of ["bounded-clients", "arrival-rate"]) {
      const samples: number[] = [],
        warm: number[] = [],
        trials: any[] = [];
      const count = profile === "standard" ? 200 : 20,
        clients = profile === "standard" ? 16 : 4,
        rate = profile === "standard" ? 500 : 100;
      for (let sample = 0; sample < (warmups + batchCount) * iterations; sample++) {
        const start = performance.now(),
          records: any[] = [],
          pending = new Set<Promise<void>>();
        let next = 0,
          dropped = 0;
        async function request(id: number, scheduled: number) {
          const sent = performance.now();
          try {
            const response = await fetch(`${url}/payload`, {
              method,
              ...(method === "POST"
                ? {
                    headers: { "content-type": "application/json" },
                    body: JSON.stringify({ id, message: expected }),
                  }
                : {}),
              signal: AbortSignal.timeout(10000),
            });
            const text = await response.text();
            records.push({
              id,
              scheduled_ms: scheduled - start,
              send_ms: sent - start,
              completed_ms: performance.now() - start,
              latency_ms: performance.now() - sent,
              status: response.status,
              correct:
                response.status === 200 &&
                text ===
                  (method === "GET"
                    ? expected
                    : JSON.stringify({ id, message: expected, length: expected.length })),
              error: null,
            });
          } catch (e) {
            records.push({
              id,
              scheduled_ms: scheduled - start,
              send_ms: sent - start,
              completed_ms: performance.now() - start,
              latency_ms: performance.now() - sent,
              status: null,
              correct: false,
              error: String(e),
            });
          }
        }
        if (scenario === "bounded-clients") {
          await Promise.all(
            Array.from({ length: clients }, async () => {
              for (;;) {
                const id = next++;
                if (id >= count) return;
                await request(id, performance.now());
              }
            }),
          );
        } else {
          for (let id = 0; id < count; id++) {
            const scheduled = start + (id * 1000) / rate;
            const delay = scheduled - performance.now();
            if (delay > 0) await Bun.sleep(delay);
            if (pending.size >= clients) {
              dropped++;
              continue;
            }
            let task: Promise<void>;
            task = request(id, scheduled).finally(() => pending.delete(task));
            pending.add(task);
          }
          await Promise.all(pending);
        }
        const elapsed = performance.now() - start;
        (sample < warmups * iterations ? warm : samples).push(elapsed);
        trials.push({
          warmup: sample < warmups * iterations,
          batch: Math.floor(sample / iterations),
          operation_in_batch: sample % iterations,
          scheduled_count: count,
          requests: records,
          elapsed_ms: elapsed,
          completed: records.length,
          errors: records.filter((r) => r.error).length,
          dropped_by_generator: dropped,
          achieved_requests_per_second: (records.length * 1000) / elapsed,
          requested_rate_per_second: scenario === "arrival-rate" ? rate : null,
          max_inflight: clients,
        });
      }
      cases.push(
        one(
          `runtime-http.${method.toLowerCase()}.${scenario}`,
          samples,
          warm,
          "Generator starts scheduling through final response body consumption; server is prestarted; per-request latency separately recorded.",
          {
            trials,
            generator:
              "Bun fetch in a separate process on the same host; fixed cap; local scheduling can limit achieved load",
            server_cpu: "unavailable",
          },
          "ms/trial",
          {
            request_count: count,
            clients,
            method,
            contract:
              method === "POST"
                ? "bounded-json-body-native-codec-runtime-response-v1"
                : "runtime-text-response-v1",
          },
        ),
      );
      cases.at(-1).correctness.passed = trials.every(
        (t) =>
          t.completed > 0 &&
          t.completed + t.dropped_by_generator === count &&
          new Set(t.requests.map((r: any) => r.id)).size === t.completed &&
          t.requests.every(
            (r: any) => Number.isInteger(r.id) && r.id >= 0 && r.id < count && r.correct,
          ) &&
          (scenario !== "bounded-clients" ||
            (t.dropped_by_generator === 0 && t.completed === count)),
      );
      cases
        .at(-1)
        .correctness.checks.push(
          "nonempty exact scheduled accounting",
          "unique valid request IDs",
          "bounded-client completes every request without drops",
        );
    }
  result("server", cases, [
    "Initial scope: Bun listener with production snapshotRequest, ownedResponse and nativeResponse adapters. Does not measure createServer ownership, routing, TLS or remote service capacity.",
  ]);
} else if (mode === "io") {
  const expected = "a".repeat(size) + "é",
    path = `${work}/payload.txt`,
    bytes = new TextEncoder().encode(expected),
    limit = bytes.length;
  writeFileSync(path, bytes);
  const reads = createFileReads(domain, {
    notFound: fileError("not_found"),
    denied: fileError("denied"),
    invalidPath: fileError("invalid_path"),
    unexpectedKind: fileError("unexpected_kind"),
    limitExceeded: fileError("limit_exceeded"),
    invalidData: fileError("codec::invalid_data"),
    ioError: fileError("io_error"),
  });
  async function nativeRead(maxBytes = limit, binary = false) {
    const reader = Bun.file(path).stream().getReader();
    const chunks: Uint8Array[] = [];
    let length = 0;
    try {
      for (;;) {
        const chunk = await reader.read();
        if (chunk.done) break;
        if (chunk.value.length > maxBytes - length) throw Error("limit");
        length += chunk.value.length;
        chunks.push(new Uint8Array(chunk.value));
      }
    } finally {
      await reader.cancel();
      reader.releaseLock();
    }
    const output = new Uint8Array(length);
    let offset = 0;
    for (const chunk of chunks) {
      output.set(chunk, offset);
      offset += chunk.length;
    }
    if (binary) return new Uint8Array(output);
    return new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(
      new Uint8Array(new Uint8Array(output)),
    );
  }
  const cases: any[] = [];
  for (const impl of ["runtime", "native-contract"]) {
    const samples: number[] = [],
      warm: number[] = [];
    for (let i = 0; i < (warmups + batchCount) * iterations; i++) {
      const start = performance.now();
      const text =
        impl === "runtime" ? value(await reads.readText(path, BigInt(limit))) : await nativeRead();
      const elapsed = performance.now() - start;
      if (text !== expected) throw Error("readText mismatch");
      (i < warmups * iterations ? warm : samples).push(elapsed);
    }
    cases.push(
      one(
        `${impl}.bounded-stream-utf8`,
        samples,
        warm,
        "Open cached file stream, enforce byte cap, collect copied chunks, materialize owned/copy boundary and fatal UTF8 decode; setup/write excluded.",
        {
          bytes: limit,
          cache_state: "warm/OS-managed; no cold-cache claim",
          per_operation_bytes_per_second: samples.map((t) => (limit * 1000) / t),
        },
        "ms/op",
        { max_bytes: limit, contract: "bounded-copy-fatal-utf8-v1" },
      ),
    );
  }
  const writes = createFileWrites(domain, {
    notFound: fileError("not_found"),
    alreadyExists: fileError("already_exists"),
    denied: fileError("denied"),
    invalidPath: fileError("invalid_path"),
    unexpectedKind: fileError("unexpected_kind"),
    notEmpty: fileError("not_empty"),
    crossDevice: fileError("cross_device"),
    ioError: fileError("io_error"),
  });
  const binary = Uint8Array.from({ length: size }, (_, i) => i % 256),
    owned = ownBytes(binary),
    binaryPath = `${work}/binary.bin`;
  for (const impl of ["runtime", "native-contract"])
    for (const scenario of ["binary-write-read", "bounded-read-rejection"]) {
      const raw: number[] = [],
        rawWarm: number[] = [];
      for (let op = 0; op < (warmups + batchCount) * iterations; op++) {
        let output: Uint8Array | undefined;
        let rejected = false;
        let elapsed: number | undefined;
        let outcome: any;
        let caught = false;
        let nativeError: unknown;
        const start = performance.now();
        if (scenario === "binary-write-read") {
          if (impl === "runtime") {
            value(await writes.writeBytes(binaryPath, owned, true));
            output = copyBytes(value(await reads.readBytes(binaryPath, BigInt(size))), byteOrigin);
          } else {
            await writeFile(binaryPath, new Uint8Array(binary), { flag: "w" });
            const reader = Bun.file(binaryPath).stream().getReader();
            const chunks: Uint8Array[] = [];
            let length = 0;
            try {
              for (;;) {
                const next = await reader.read();
                if (next.done) break;
                if (next.value.length > size - length) throw Error("limit");
                length += next.value.length;
                chunks.push(new Uint8Array(next.value));
              }
            } finally {
              reader.releaseLock();
            }
            const assembled = new Uint8Array(length);
            let offset = 0;
            for (const chunk of chunks) {
              assembled.set(chunk, offset);
              offset += chunk.length;
            }
            output = new Uint8Array(new Uint8Array(assembled));
          }
        } else if (impl === "runtime") {
          outcome = await reads.readText(path, BigInt(limit - 1));
          elapsed = performance.now() - start;
        } else {
          try {
            await nativeRead(limit - 1);
          } catch (error) {
            elapsed = performance.now() - start;
            nativeError = error;
            caught = true;
          }
        }
        elapsed ??= performance.now() - start;
        if (scenario === "bounded-read-rejection") {
          if (impl === "runtime") {
            const details =
              outcome.kind === "domain" ? domainFailureDiagnostics(outcome.value) : undefined;
            rejected =
              details?.declaration.name === "files::limit_exceeded" &&
              (details.payload as any).limit === BigInt(limit - 1);
          } else if (caught) {
            if (String(nativeError) !== "Error: limit") throw nativeError;
            rejected = true;
          }
        }
        if (
          scenario === "binary-write-read"
            ? output!.length !== binary.length ||
              output!.some((n, i) => n !== binary[i]) ||
              copyBytes(owned, byteOrigin).some((n, i) => n !== binary[i])
            : !rejected
        )
          throw Error("I/O lifecycle/bound correctness failed");
        (op < warmups * iterations ? rawWarm : raw).push(elapsed);
      }
      cases.push(
        one(
          `${impl}.${scenario}`,
          raw,
          rawWarm,
          scenario === "binary-write-read"
            ? "Overwrite bytes, bounded stream read, ownership copies and outbound materialization through completion; excludes fsync/durability and validation."
            : "Open bounded stream and reject before retaining bytes over cap, including stream cancellation; error diagnostic verification and decoding success excluded.",
          {
            bytes: scenario === "binary-write-read" ? size : limit,
            max_bytes: scenario === "binary-write-read" ? size : limit - 1,
            cache_state: "OS-managed/warm",
          },
          "ms/op",
          {
            contract:
              scenario === "binary-write-read"
                ? "copied-bounded-binary-overwrite-v1"
                : "bounded-read-rejection-v1",
          },
        ),
      );
      cases
        .at(-1)
        .correctness.checks.push(
          scenario === "binary-write-read"
            ? "exact all-byte roundtrip and source ownership"
            : "expected byte-cap rejection",
        );
    }
  result("io", cases, [
    "Initial scope: cached local bounded text read through real runtime file/stream/UTF8 adapters and equivalent native contract. Binary overwrite/read lifecycle and byte-bound failure are included; no fsync durability or network storage coverage.",
  ]);
} else if (mode === "journeys") {
  const { runJourney, runInvoice } = await import(`${work}/journey-adapter.ts`);
  const request = JSON.stringify(Array.from({ length: size }, (_, i) => String(i)));
  const samples: number[] = [],
    warm: number[] = [];
  for (let i = 0; i < (warmups + batchCount) * iterations; i++) {
    const start = performance.now();
    const input = (JSON.parse(request) as string[]).map(BigInt);
    const transformed = await runJourney(input);
    const response = JSON.stringify(transformed.map(String));
    const output = JSON.parse(response) as string[];
    const elapsed = performance.now() - start;
    if (
      !Object.isFrozen(transformed) ||
      input.some((n, j) => n !== BigInt(j)) ||
      output.length !== size ||
      output.some((n: string, j: number) => n !== String(BigInt(j) * 2n))
    )
      throw Error("emitted journey mismatch");
    (i < warmups * iterations ? warm : samples).push(elapsed);
  }
  const invoiceCases: any[] = [];
  for (const invalid of [false, true]) {
    const rows = Array.from({ length: size }, (_, i) => ({
      sku: `SKU-${i}`,
      quantity: String(invalid && i === size - 1 ? 0 : (i % 5) + 1),
      unit_minor: String(100 + (i % 11)),
    }));
    const request = JSON.stringify({ lines: rows });
    const expected = JSON.stringify({
      accepted: !invalid,
      total_minor: String(
        rows.reduce((total, row) => total + BigInt(row.quantity) * BigInt(row.unit_minor), 0n),
      ),
      invalid_lines: invalid ? 1 : 0,
    });
    const raw: number[] = [],
      rawWarm: number[] = [];
    for (let op = 0; op < (warmups + batchCount) * iterations; op++) {
      const start = performance.now();
      const payload = JSON.parse(request);
      const observed = await runInvoice(payload.lines);
      const summary = observed.summary;
      const response = JSON.stringify({
        accepted: summary.accepted,
        total_minor: String(summary.total_minor),
        invalid_lines: Number(summary.invalid_lines),
      });
      const elapsed = performance.now() - start;
      if (
        response !== expected ||
        JSON.stringify(payload.lines) !== JSON.stringify(rows) ||
        !Object.isFrozen(summary) ||
        !Object.isFrozen(observed.input) ||
        observed.lines.length !== rows.length ||
        observed.lines.some(
          (line: any, i: number) =>
            !Object.isFrozen(line) ||
            observed.input[i] !== line ||
            line.sku !== rows[i].sku ||
            line.quantity !== BigInt(rows[i].quantity) ||
            line.unit_minor !== BigInt(rows[i].unit_minor),
        )
      )
        throw Error("invoice response/input correctness failed");
      (op < warmups * iterations ? rawWarm : raw).push(elapsed);
    }
    invoiceCases.push(
      one(
        `emitted-can.invoice.${invalid ? "invalid-line" : "valid"}`,
        raw,
        rawWarm,
        "Decode invoice request, construct actual emitted Can line records, validate/map/fold in emitted Can and serialize exact response; minimal adapter observation return included, full correctness/immutability verification excluded; compile/setup excluded.",
        {
          emission: "compiler/perfemit",
          source_inspiration: "examples/invoice line records",
          database_authorization_and_browser: "unavailable",
        },
        "ms/journey",
        { contract: "invoice-record-validation-reduction-v1", invalid },
      ),
    );
    invoiceCases
      .at(-1)
      .correctness.checks.push(
        "exact serialized response",
        "immutable result/input/line records",
        "source request unchanged",
      );
  }
  result(
    "journeys",
    [
      ...invoiceCases,
      one(
        "emitted-can.serialized-batch-request",
        samples,
        warm,
        "Decode serialized application request, create integer input, call actual compiler-emitted Can doubled function, serialize response and consume decoded result; emission and request construction excluded.",
        { emission: "compiler/perfemit", full_browser_journey: "unavailable" },
        "ms/journey",
      ),
    ],
    [
      "Emitted serialized batch and invoice record validation/map/reduction request journeys, including valid and invalid-line outcomes; no whole-product UI, authorization/database or remote network journey coverage.",
    ],
  );
} else throw Error(`unknown workload ${mode}`);
