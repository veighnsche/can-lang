import { test, expect } from "bun:test";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { mkdtemp, rm, writeFile, readFile } from "node:fs/promises";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import {
  createStreamReads,
  drainStream,
  openByteCell,
  openLineCell,
} from "../transport/stream/readable.ts";
import { createStreamWrites, registerSink } from "../transport/stream/writable.ts";
import { registerReader } from "../transport/stream/lifecycle.ts";
import { createFileStreams } from "../platform/files/stream.ts";
import { copyBytes, ownBytes } from "../bytes.ts";
import { dataArray, record } from "../data.ts";
import { failure, success, type Completion } from "../completion.ts";
import { isStandardFailure, standardFailureKind } from "../failure.ts";
import { runOwnedRoot } from "../owner.ts";
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
const declarations = catalogue.errors.filter((e) =>
  [
    "files::not_found",
    "files::denied",
    "files::invalid_path",
    "files::io_error",
    "files::limit_exceeded",
    "files::unexpected_kind",
    "stream::read_failed",
    "stream::write_failed",
    "stream::cancelled",
    "stream::close_failed",
  ].includes(e.name),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, int, ...errors],
});
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
const reads = createStreamReads(domain, {
  readFailed: identity("can.std.stream@1::read_failed"),
  cancelled: identity("can.std.stream@1::cancelled"),
  closeFailed: identity("can.std.stream@1::close_failed"),
  limitExceeded: identity("can.std.files@1::limit_exceeded"),
});
const writes = createStreamWrites(domain, {
  writeFailed: identity("can.std.stream@1::write_failed"),
  closeFailed: identity("can.std.stream@1::close_failed"),
});
const files = createFileStreams(domain, {
  notFound: identity("can.std.files@1::not_found"),
  denied: identity("can.std.files@1::denied"),
  invalidPath: identity("can.std.files@1::invalid_path"),
  unexpectedKind: identity("can.std.files@1::unexpected_kind"),
  limitExceeded: identity("can.std.files@1::limit_exceeded"),
  ioError: identity("can.std.files@1::io_error"),
  closeFailed: identity("can.std.stream@1::close_failed"),
});
const origin = { source: "test", start: 0, end: 0, invocation: [] };
const fail = (id: string, fields: readonly (readonly [string, unknown])[], cause?: unknown) =>
  failure(domain.create(id, record(id, fields), origin, cause));
async function withTemp(run: (root: string) => Promise<void>): Promise<void> {
  const root = await mkdtemp(join(tmpdir(), "can-streams-test-"));
  try {
    await run(root);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}
const bytesOf = (value: unknown) => Array.from(copyBytes(value, origin));
function standardOf(thrown: unknown) {
  expect(isStandardFailure(thrown)).toBe(true);
  return standardFailureKind(thrown as never);
}
test("bytes write/read preserves order across chunks with exact end", async () => {
  await withTemp(async (root) => {
    const file = join(root, "a.bin");
    const result = await runOwnedRoot(async () => {
      const opened = await files.writeStream(file);
      if (opened.kind !== "ok") throw Error("open failed");
      const writer = opened.value;
      expect(await writes.writeSome(writer, ownBytes(new Uint8Array([1, 2, 3])))).toEqual(
        success(3n),
      );
      expect(await writes.writeSome(writer, ownBytes(new Uint8Array([4, 5])))).toEqual(success(2n));
      expect(await writes.closeWriter(writer)).toEqual(success(undefined));
      const ropened = await files.readStream(file, 2n);
      if (ropened.kind !== "ok") throw Error("read open failed");
      const reader = ropened.value;
      const first = await reads.readMany(reader, 10n);
      if (first.kind !== "ok") throw Error("read failed");
      const chunks = dataArray(first.value).map(bytesOf);
      const end = await reads.readMany(reader, 10n);
      if (end.kind !== "ok") throw Error("end read failed");
      expect(dataArray(end.value)).toEqual([]);
      expect(await reads.closeReader(reader)).toEqual(success(undefined));
      return success(chunks);
    });
    if (result.completion.kind !== "ok") throw Error();
    expect(result.completion.value).toEqual([[1, 2], [3, 4], [5]]);
  });
});
test("lines frame CRLF, trailing segments and split UTF-8; malformed input fails", async () => {
  await withTemp(async (root) => {
    const file = join(root, "lines.txt");
    await writeFile(file, "one\r\ntwo\n€\npartial");
    const result = await runOwnedRoot(async () => {
      const opened = await files.readLinesStream(file, 64n);
      if (opened.kind !== "ok") throw Error("open failed");
      const reader = opened.value;
      const lines: string[] = [];
      for (;;) {
        const next = await reads.readMany(reader, 2n);
        if (next.kind !== "ok") throw Error("read failed");
        const batch = dataArray(next.value) as string[];
        if (batch.length === 0) break;
        lines.push(...batch);
      }
      await reads.closeReader(reader);
      return success(lines);
    });
    if (result.completion.kind !== "ok") throw Error();
    expect(result.completion.value).toEqual(["one", "two", "€", "partial"]);
    const split = new ReadableStream<Uint8Array>({
      start(c) {
        c.enqueue(new Uint8Array([0xe2]));
        c.enqueue(new Uint8Array([0x82, 0xac, 0x0a]));
        c.close();
      },
    });
    const forced = await runOwnedRoot(async () => {
      const token = registerReader(
        openLineCell(split, 64n),
        fail,
        identity("can.std.stream@1::close_failed"),
      );
      const next = await reads.readMany(token, 4n);
      if (next.kind !== "ok") throw Error("split read failed");
      const lines = [...(dataArray(next.value) as string[])];
      const end = await reads.readMany(token, 4n);
      if (end.kind !== "ok" || dataArray(end.value).length !== 0) throw Error("split end failed");
      await reads.closeReader(token);
      return success(lines);
    });
    if (forced.completion.kind !== "ok") throw Error();
    expect(forced.completion.value).toEqual(["€"]);
    const bad = join(root, "bad.txt");
    await writeFile(bad, new Uint8Array([0xff, 0xfe, 0x0a]));
    await runOwnedRoot(async () => {
      const opened = await files.readLinesStream(bad, 64n);
      if (opened.kind !== "ok") throw Error("open failed");
      check(await reads.readMany(opened.value, 4n), "stream::read_failed", { reason: "utf8" });
      expect(await reads.closeReader(opened.value)).toEqual(success(undefined));
      return success(undefined);
    });
  });
});
test("line over cap reports limit_exceeded once and ends the reader", async () => {
  await withTemp(async (root) => {
    const file = join(root, "long.txt");
    await writeFile(file, "0123456789\n");
    await runOwnedRoot(async () => {
      const opened = await files.readLinesStream(file, 4n);
      if (opened.kind !== "ok") throw Error("open failed");
      const reader = opened.value;
      check(await reads.readMany(reader, 4n), "files::limit_exceeded", { limit: 4n });
      let thrown: unknown;
      try {
        await reads.readMany(reader, 4n);
      } catch (cause) {
        thrown = cause;
      }
      expect(standardOf(thrown)).toBe("resource_state");
      expect(await reads.closeReader(reader)).toEqual(success(undefined));
      return success(undefined);
    });
  });
});
test("cancel during a pending read reports cancelled and delivers nothing", async () => {
  let release!: () => void;
  let started = false;
  const gate = new Promise<void>((resolve) => {
    release = resolve;
  });
  const stream = new ReadableStream<Uint8Array>({
    async pull(c) {
      started = true;
      await gate;
      try {
        c.enqueue(new Uint8Array([9]));
      } catch {}
    },
  });
  const out = await runOwnedRoot(async () => {
    const token = registerReader(
      openByteCell(stream, 8n),
      fail,
      identity("can.std.stream@1::close_failed"),
    );
    const pending = reads.readMany(token, 4n);
    while (!started) await Promise.resolve();
    expect(await reads.cancelReader(token, "deadline")).toEqual(success(undefined));
    release();
    return pending;
  });
  check(out.completion, "stream::cancelled", { reason: "deadline" });
});
test("close twice, use-after-close and foreign handles throw resource-state", async () => {
  await runOwnedRoot(async () => {
    const stream = new ReadableStream<Uint8Array>({
      start(c) {
        c.enqueue(new Uint8Array([1]));
        c.close();
      },
    });
    const token = registerReader(
      openByteCell(stream, 8n),
      fail,
      identity("can.std.stream@1::close_failed"),
    );
    expect(await reads.closeReader(token)).toEqual(success(undefined));
    for (const attempt of [
      () => reads.closeReader(token),
      () => reads.readMany(token, 1n),
      () => reads.cancelReader(token, "x"),
    ]) {
      let thrown: unknown;
      try {
        await attempt();
      } catch (cause) {
        thrown = cause;
      }
      expect(standardOf(thrown)).toBe("resource_state");
    }
    const sink = {
      chunks: [] as Uint8Array[],
      write(c: Uint8Array) {
        this.chunks.push(c);
        return c.byteLength;
      },
      flush() {},
      end() {},
    };
    const writer = registerSink(sink, fail, identity("can.std.stream@1::close_failed"));
    let thrown: unknown;
    try {
      await reads.readMany(writer, 1n);
    } catch (cause) {
      thrown = cause;
    }
    expect(standardOf(thrown)).toBe("resource_state");
    try {
      await reads.readMany({}, 1n);
    } catch (cause) {
      thrown = cause;
    }
    expect(standardOf(thrown)).toBe("resource_state");
    expect(await writes.closeWriter(writer)).toEqual(success(undefined));
    return success(undefined);
  });
});
test("native read failure reports once; close still succeeds; end is sticky", async () => {
  const failing = new ReadableStream<Uint8Array>({
    pull() {
      throw Object.assign(new Error("boom"), { code: "EIO" });
    },
  });
  await runOwnedRoot(async () => {
    const token = registerReader(
      openByteCell(failing, 8n),
      fail,
      identity("can.std.stream@1::close_failed"),
    );
    check(await reads.readMany(token, 4n), "stream::read_failed", { reason: "EIO" });
    let thrown: unknown;
    try {
      await reads.readMany(token, 4n);
    } catch (cause) {
      thrown = cause;
    }
    expect(standardOf(thrown)).toBe("resource_state");
    expect(await reads.closeReader(token)).toEqual(success(undefined));
    return success(undefined);
  });
  const ending = new ReadableStream<Uint8Array>({
    start(c) {
      c.enqueue(new Uint8Array([7]));
      c.close();
    },
  });
  await runOwnedRoot(async () => {
    const token = registerReader(
      openByteCell(ending, 8n),
      fail,
      identity("can.std.stream@1::close_failed"),
    );
    const first = await reads.readMany(token, 4n);
    if (first.kind !== "ok") throw Error();
    expect(dataArray(first.value).map(bytesOf)).toEqual([[7]]);
    for (let i = 0; i < 2; i++) {
      const end = await reads.readMany(token, 4n);
      if (end.kind !== "ok") throw Error();
      expect(dataArray(end.value)).toEqual([]);
    }
    expect(await reads.closeReader(token)).toEqual(success(undefined));
    return success(undefined);
  });
});
test("data then error delivers data first, then fails", async () => {
  let pulls = 0;
  const stream = new ReadableStream<Uint8Array>({
    pull(c) {
      pulls++;
      if (pulls === 1) c.enqueue(new Uint8Array([3]));
      else throw Object.assign(new Error("late"), { code: "EIO" });
    },
  });
  await runOwnedRoot(async () => {
    const token = registerReader(
      openByteCell(stream, 8n),
      fail,
      identity("can.std.stream@1::close_failed"),
    );
    const first = await reads.readMany(token, 1n);
    if (first.kind !== "ok") throw Error();
    expect(dataArray(first.value).map(bytesOf)).toEqual([[3]]);
    check(await reads.readMany(token, 1n), "stream::read_failed", { reason: "EIO" });
    expect(await reads.closeReader(token)).toEqual(success(undefined));
    return success(undefined);
  });
});
test("boundary copies detach both directions", async () => {
  await withTemp(async (root) => {
    const file = join(root, "copy.bin");
    await runOwnedRoot(async () => {
      const opened = await files.writeStream(file);
      if (opened.kind !== "ok") throw Error();
      const writer = opened.value;
      const backing = new Uint8Array([10, 20]);
      const chunk = ownBytes(backing);
      backing[0] = 99;
      expect(await writes.writeSome(writer, chunk)).toEqual(success(2n));
      expect(await writes.closeWriter(writer)).toEqual(success(undefined));
      return success(undefined);
    });
    expect(Array.from(await readFile(file))).toEqual([10, 20]);
    const result = await runOwnedRoot(async () => {
      const opened = await files.readStream(file, 8n);
      if (opened.kind !== "ok") throw Error();
      const reader = opened.value;
      const first = await reads.readMany(reader, 4n);
      if (first.kind !== "ok") throw Error();
      const held = dataArray(first.value)[0];
      await writeFile(file, new Uint8Array([30, 40]));
      const end = await reads.readMany(reader, 4n);
      if (end.kind !== "ok") throw Error();
      await reads.closeReader(reader);
      return success(bytesOf(held));
    });
    if (result.completion.kind !== "ok") throw Error();
    expect(result.completion.value).toEqual([10, 20]);
  });
});
test("short writes report partial counts; failing ends report close_failed", async () => {
  let calls = 0;
  const short = {
    write(c: Uint8Array) {
      calls++;
      return calls === 1 ? 1 : c.byteLength - 1;
    },
    flush() {},
    end() {},
  };
  await runOwnedRoot(async () => {
    const writer = registerSink(short, fail, identity("can.std.stream@1::close_failed"));
    expect(await writes.writeSome(writer, ownBytes(new Uint8Array([1, 2, 3])))).toEqual(
      success(1n),
    );
    expect(await writes.closeWriter(writer)).toEqual(success(undefined));
    return success(undefined);
  });
  const badEnd = {
    write(c: Uint8Array) {
      return c.byteLength;
    },
    flush() {},
    end() {
      throw Object.assign(new Error("flush lost"), { code: "EIO" });
    },
  };
  await runOwnedRoot(async () => {
    const writer = registerSink(badEnd, fail, identity("can.std.stream@1::close_failed"));
    check(await writes.closeWriter(writer), "stream::close_failed", { reason: "close" });
    return success(undefined);
  });
});
test("open failures attribute acquisition; writers truncate at open", async () => {
  await withTemp(async (root) => {
    await runOwnedRoot(async () => {
      check(await files.readStream(join(root, "missing.bin"), 8n), "files::not_found", {
        path: join(root, "missing.bin"),
      });
      check(await files.readStream(root, 8n), "files::unexpected_kind", {
        path: root,
        operation: "open",
      });
      check(await files.readStream(join(root, "x"), 0n), "files::limit_exceeded", { limit: 0n });
      check(await files.readLinesStream(join(root, "x"), -2n), "files::limit_exceeded", {
        limit: -2n,
      });
      check(await files.writeStream(join(root, "nope", "f.bin")), "files::not_found", {
        path: join(root, "nope", "f.bin"),
      });
      check(await files.writeStream(root), "files::unexpected_kind", {
        path: root,
        operation: "open",
      });
      return success(undefined);
    });
    const file = join(root, "trunc.bin");
    await writeFile(file, new Uint8Array([1, 2, 3]));
    await runOwnedRoot(async () => {
      const opened = await files.writeStream(file);
      if (opened.kind !== "ok") throw Error();
      expect(await writes.closeWriter(opened.value)).toEqual(success(undefined));
      return success(undefined);
    });
    expect(Array.from(await readFile(file))).toEqual([]);
  });
});
test("shared drain overflows before retaining and concatenates within cap", async () => {
  const big = new ReadableStream<Uint8Array>({
    start(c) {
      c.enqueue(new Uint8Array([1, 2, 3, 4]));
      c.close();
    },
  });
  await runOwnedRoot(async () => {
    const over = await drainStream(big, 3n, fail, identity("can.std.files@1::io_error"));
    expect(over.overflow).toBe(true);
    return success(undefined);
  });
  const small = new ReadableStream<Uint8Array>({
    start(c) {
      c.enqueue(new Uint8Array([5]));
      c.enqueue(new Uint8Array([6, 7]));
      c.close();
    },
  });
  const result = await runOwnedRoot(async () => {
    const ok = await drainStream(small, 8n, fail, identity("can.std.files@1::io_error"));
    if (ok.overflow) throw Error("unexpected overflow");
    return success(Array.from(ok.data));
  });
  if (result.completion.kind !== "ok") throw Error();
  expect(result.completion.value).toEqual([5, 6, 7]);
});
