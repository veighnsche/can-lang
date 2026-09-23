// B1-10 live S3 qualification against a provisioned S3-compatible service
// (MinIO in CI/dev). Bun 1.4.2 is the only wire client; usage:
//   CAN_TEST_S3_ENDPOINT=http://127.0.0.1:9000 CAN_TEST_S3_ACCESS_KEY=... \
//     CAN_TEST_S3_SECRET_KEY=... bun s3-native-probe.ts <out.json>
// The probe creates a unique scratch prefix, runs every row, deletes the
// scratch keys, and writes a deterministic report (no credentials, no
// absolute timestamps, no raw presigned URLs).
import { S3Client } from "bun";

const outPath = process.argv[2];
const endpoint = process.env.CAN_TEST_S3_ENDPOINT;
const accessKeyId = process.env.CAN_TEST_S3_ACCESS_KEY;
const secretAccessKey = process.env.CAN_TEST_S3_SECRET_KEY;
if (typeof outPath !== "string" || outPath === "") throw new Error("missing output path argument");
if (typeof endpoint !== "string" || endpoint === "") throw new Error("CAN_TEST_S3_ENDPOINT is not set");
if (typeof accessKeyId !== "string" || accessKeyId === "") throw new Error("CAN_TEST_S3_ACCESS_KEY is not set");
if (typeof secretAccessKey !== "string" || secretAccessKey === "") throw new Error("CAN_TEST_S3_SECRET_KEY is not set");

const bucket = "can-b1-10";
const prefix = `probe/${Date.now().toString(36)}-${Math.floor(Math.random() * 1e6).toString(36)}/`;
const rows: Record<string, unknown> = {};
const attempt = async (label: string, fn: () => unknown) => {
  try { rows[label] = { status: "observed", value: await fn() }; }
  catch (e) {
    const err = e as { name?: unknown; code?: unknown; message?: unknown };
    rows[label] = { status: "error", name: String(err.name ?? "?"), code: String(err.code ?? "?"), message: String(err.message ?? "").slice(0, 200) };
  }
};
const fail = async (label: string, fn: () => unknown) => {
  try { rows[label] = { status: "unexpected-ok", value: await fn() }; }
  catch (e) {
    const err = e as { name?: unknown; code?: unknown; message?: unknown };
    rows[label] = { status: "throws", name: String(err.name ?? "?"), code: String(err.code ?? "?"), message: String(err.message ?? "").slice(0, 200) };
  }
};

const client = new S3Client({ endpoint, region: "us-east-1", bucket, accessKeyId, secretAccessKey });
try {
  const settle = async () => {
    // Abandonment verdicts must settle: close() completes uploads
    // asynchronously, so an immediate check reports a false absent.
    try { (Bun as unknown as { gc?: (force?: boolean) => void }).gc?.(true); } catch { /* best effort */ }
    await new Promise((r) => setTimeout(r, 1500));
  };
  await attempt("methods", () => {
    const names = (o: object) => Object.getOwnPropertyNames(Object.getPrototypeOf(o)).filter((m) => m !== "constructor").sort();
    const w = client.file(`${prefix}methods.bin`).writer();
    const out = {
      writer: { ctor: w.constructor.name, methods: names(w) },
      file: { ctor: client.file(`${prefix}methods.bin`).constructor.name, methods: names(client.file(`${prefix}methods.bin`)) },
      client: { ctor: client.constructor.name, methods: names(client) },
    };
    void w;
    return out;
  });
  await attempt("roundtrip", async () => {
    const f = client.file(`${prefix}round.txt`);
    const bytes = await f.write("hello s3", { type: "text/plain" });
    const st = await f.stat() as unknown as Record<string, unknown>;
    const out = {
      bytes, text: await client.file(`${prefix}round.txt`).text(),
      jsonText: await client.file(`${prefix}round.json`).write("{\"a\":1}").then(() => client.file(`${prefix}round.json`).json()),
      size: st.size, type: st.type, etag: st.etag,
      lastModifiedIsDate: st.lastModified instanceof Date,
      lastModifiedMsPositive: st.lastModified instanceof Date && st.lastModified.getTime() > 0,
      statKeysEnumerable: Object.keys(st),
      existsBefore: await f.exists(),
    };
    await f.delete();
    return { ...out, existsAfter: await f.exists() };
  });
  await attempt("write_inputs", async () => {
    const n1 = await client.file(`${prefix}w-str.bin`).write("abc");
    const n2 = await client.file(`${prefix}w-bytes.bin`).write(new Uint8Array([1, 2, 3, 4]));
    const n3 = await client.file(`${prefix}w-blob.bin`).write(new Blob(["blob-body"], { type: "application/json" }));
    const n4 = await client.file(`${prefix}w-resp.bin`).write(new Response("resp-body", { headers: { "content-type": "text/plain" } }));
    const n5 = await client.file(`${prefix}w-buf.bin`).write(new Uint8Array([9, 9]).buffer as ArrayBuffer);
    const blobType = (await client.file(`${prefix}w-blob.bin`).stat() as { type: string }).type;
    const back = await client.file(`${prefix}w-bytes.bin`).bytes();
    return { n1, n2, n3, n4, n5, blobStoredType: blobType, bytesRoundtrip: Array.from(back) };
  });
  await attempt("range", async () => {
    const data = new Uint8Array(12 * 1024 * 1024);
    for (let i = 0; i < data.length; i++) data[i] = i % 251;
    await client.file(`${prefix}big.bin`).write(data);
    const f = client.file(`${prefix}big.bin`);
    const first = new Uint8Array(await f.slice(0, 4).arrayBuffer());
    const mid = new Uint8Array(await f.slice(6 * 1024 * 1024, 6 * 1024 * 1024 + 4).arrayBuffer());
    const tail = new Uint8Array(await f.slice(data.length - 4, data.length).arrayBuffer());
    const empty = new Uint8Array(await f.slice(100, 100).arrayBuffer());
    const clamp = new Uint8Array(await f.slice(data.length - 2, data.length + 1000).arrayBuffer());
    const stream = f.stream() as ReadableStream<Uint8Array>;
    const reader = stream.getReader();
    const head = await reader.read();
    await reader.cancel("probe-cancel");
    const after = await reader.read();
    reader.releaseLock();
    return {
      first: Array.from(first), mid: Array.from(mid), tail: Array.from(tail),
      emptyLen: empty.length, clampLen: clamp.length,
      headBytes: head.value?.length ?? -1, headDone: head.done,
      afterCancelDone: after.done, lockedAfterRelease: stream.locked,
    };
  });
  await attempt("list", async () => {
    for (let i = 0; i < 5; i++) await client.file(`${prefix}page-${i}.txt`).write(`p${i}`);
    await client.file(`${prefix}dir/a.txt`).write("a");
    await client.file(`${prefix}dir/b.txt`).write("b");
    const flat = await client.list({ prefix, maxKeys: 2 }) as unknown as Record<string, unknown>;
    const full = await client.list({ prefix }) as unknown as { contents?: Array<{ key?: string; size?: number }>; keyCount?: number; isTruncated?: boolean; continuationToken?: unknown; nextContinuationToken?: unknown };
    const grouped = await client.list({ prefix, delimiter: "/" }) as unknown as Record<string, unknown>;
    const rel = (k: string | undefined) => (k ?? "").startsWith(prefix) ? (k ?? "").slice(prefix.length) : k;
    return {
      page: { keys: Object.keys(flat).sort(), isTruncated: flat.isTruncated, keyCount: flat.keyCount, contents: (flat.contents as Array<{ key?: string }> | undefined)?.map((c) => rel(c.key)), nextTokenType: typeof flat.nextContinuationToken, nextTokenEmpty: flat.nextContinuationToken === "" },
      fullCount: full.contents?.length ?? -1, fullTruncated: full.isTruncated,
      groupedKeys: Object.keys(grouped).sort(),
      commonPrefixes: (grouped.commonPrefixes as Array<{ prefix?: string }> | undefined)?.map((p) => rel(p.prefix)),
      entryShape: (() => { const e = full.contents?.[0] as unknown as Record<string, unknown> | undefined; return e ? Object.keys(e).sort() : []; })(),
    };
  });
  await attempt("list_continuation", async () => {
    const rel = (k: string | undefined) => (k ?? "").startsWith(prefix) ? (k ?? "").slice(prefix.length) : k;
    const p1 = await client.list({ prefix, maxKeys: 2 }) as unknown as { contents?: Array<{ key?: string }>; isTruncated?: boolean; nextContinuationToken?: string };
    const token = p1.nextContinuationToken;
    const p2 = await client.list({ prefix, maxKeys: 2, continuationToken: token }) as unknown as { contents?: Array<{ key?: string }>; isTruncated?: boolean; nextContinuationToken?: string };
    const empty = await client.list({ prefix: `${prefix}no-such-dir/` }) as unknown as { contents?: unknown[]; isTruncated?: boolean; keyCount?: number };
    return {
      page1: (p1.contents ?? []).map((c) => rel(c.key)), truncated1: p1.isTruncated, tokenType: typeof token, tokenEmpty: token === "",
      page2: (p2.contents ?? []).map((c) => rel(c.key)), truncated2: p2.isTruncated,
      overlap: (p1.contents ?? []).some((a) => (p2.contents ?? []).some((b) => a.key === b.key)),
      emptyCount: (empty.contents ?? []).length, emptyTruncated: empty.isTruncated, emptyKeyCount: empty.keyCount,
    };
  });
  await attempt("slice_edges", async () => {
    const f = client.file(`${prefix}big.bin`);
    const at100 = new Uint8Array(await f.slice(100, 100).arrayBuffer());
    const at0 = new Uint8Array(await f.slice(0, 0).arrayBuffer());
    const one = new Uint8Array(await f.slice(100, 101).arrayBuffer());
    const sliced = f.slice(10, 14);
    return {
      emptyAt100: Array.from(at100), emptyAt0: Array.from(at0), oneAt100: Array.from(one),
      expected100: 100 % 251, slicedSize: sliced.size, slicedType: sliced.type,
    };
  });
  await attempt("presign", async () => {
    await client.file(`${prefix}get.bin`).write("presigned");
    const url = String(client.presign(`${prefix}get.bin`, { expiresIn: 300 }));
    const parsed = new URL(url);
    const get = await fetch(url);
    const tampered = url.replace(/X-Amz-Signature=[^&]+/, "X-Amz-Signature=00");
    const tamp = await fetch(tampered);
    const tampPutUrl = String(client.presign(`${prefix}tampered-put.bin`, { method: "PUT", expiresIn: 300 })).replace(/X-Amz-Signature=[^&]+/, "X-Amz-Signature=00");
    const tampPut = await fetch(tampPutUrl, { method: "PUT", body: "x" });
    const putTyped = String(client.presign(`${prefix}put.bin`, { method: "PUT", expiresIn: 300, type: "application/json" }));
    const putMatch = await fetch(putTyped, { method: "PUT", body: "{}", headers: { "content-type": "application/json" } });
    const putMismatch = await fetch(putTyped, { method: "PUT", body: "{}", headers: { "content-type": "text/plain" } });
    const putUrlGet = await fetch(String(client.presign(`${prefix}get.bin`, { method: "PUT", expiresIn: 300 })), { method: "GET" });
    const getUrlPut = await fetch(String(client.presign(`${prefix}get.bin`, { expiresIn: 300 })), { method: "PUT", body: "x" });
    await client.file(`${prefix}del.bin`).write("d");
    const delUrl = String(client.presign(`${prefix}del.bin`, { method: "DELETE", expiresIn: 300 }));
    const del = await fetch(delUrl, { method: "DELETE" });
    const head = await fetch(String(client.presign(`${prefix}get.bin`, { method: "HEAD", expiresIn: 300 })), { method: "HEAD" });
    return {
      params: [...parsed.searchParams.keys()].sort(),
      leaksSecret: url.includes(secretAccessKey),
      getStatus: get.status, getBody: await get.text(),
      tamperedStatus: tamp.status,
      tamperedPutStatus: tampPut.status, tamperedPutCreated: await client.file(`${prefix}tampered-put.bin`).exists(),
      putMatchStatus: putMatch.status, putMismatchStatus: putMismatch.status,
      putUrlWithGet: putUrlGet.status, getUrlWithPut: getUrlPut.status,
      deleteStatus: del.status, deletedGone: !(await client.file(`${prefix}del.bin`).exists()),
      headStatus: head.status, headLength: head.headers.get("content-length"),
    };
  });
  await attempt("multipart", async () => {
    const data = new Uint8Array(6 * 1024 * 1024);
    for (let i = 0; i < data.length; i++) data[i] = i % 251;
    const w = client.file(`${prefix}multi.bin`).writer({ partSize: 5 * 1024 * 1024 });
    w.write(data.subarray(0, 5 * 1024 * 1024)); await w.flush();
    w.write(data.subarray(5 * 1024 * 1024)); await w.flush();
    const endBytes = await w.end();
    const st = await client.file(`${prefix}multi.bin`).stat() as unknown as { size?: number; etag?: string };
    {
      // Multipart abandonment: parts flushed, writer dropped without
      // close or end. Never call close() here: it async-completes.
      const w2 = client.file(`${prefix}abandon.bin`).writer({ partSize: 5 * 1024 * 1024 });
      w2.write(data); await w2.flush();
    }
    {
      // Single-part abandonment: bytes flushed, writer dropped.
      const w3 = client.file(`${prefix}abandon-small.bin`).writer();
      w3.write(new Uint8Array([7])); await w3.flush();
    }
    {
      // Empty abandonment: writer dropped without any write.
      client.file(`${prefix}abandon-empty.bin`).writer();
    }
    {
      // Guard row: close() without end() still materializes the key
      // asynchronously, so cancellation must never close.
      const w4 = client.file(`${prefix}close-completes.bin`).writer();
      w4.write(new Uint8Array([7])); await w4.flush();
      w4.close();
    }
    await settle();
    const w5 = client.file(`${prefix}empty.bin`).writer();
    const emptyEnd = await w5.end();
    return {
      endBytes, endBytesType: typeof endBytes, size: st.size, sizeMatches: st.size === data.length,
      storedEtag: st.etag, storedEtagMultipart: typeof st.etag === "string" && /-[0-9]+\"$/.test(st.etag),
      abandonedExists: await client.file(`${prefix}abandon.bin`).exists(),
      abandonedSmallExists: await client.file(`${prefix}abandon-small.bin`).exists(),
      abandonedEmptyExists: await client.file(`${prefix}abandon-empty.bin`).exists(),
      closeCompletesSettled: await client.file(`${prefix}close-completes.bin`).exists(),
      emptyEnd, emptySize: (await client.file(`${prefix}empty.bin`).stat() as { size?: number }).size,
    };
  });
  await fail("multipart_partsize_floor", async () => client.file(`${prefix}floor.bin`).writer({ partSize: 1024 }));
  await attempt("writer_after_end", async () => {
    const w = client.file(`${prefix}afterend.bin`).writer();
    w.write("x"); await w.flush();
    await w.end();
    let writeAfter = "no-throw";
    try { w.write("y"); } catch (e) { writeAfter = (e as Error).name; }
    let endAfter = "no-throw";
    try { await w.end(); } catch (e) { endAfter = (e as Error).name; }
    return { writeAfter, endAfter, text: await client.file(`${prefix}afterend.bin`).text() };
  });
  await attempt("missing", async () => {
    const f = client.file(`${prefix}absent.bin`);
    const grab = async (fn: () => unknown) => {
      try { return { ok: await fn() }; }
      catch (e) { const err = e as { name?: unknown; code?: unknown; message?: unknown }; return { throws: `${String(err.name)}:${String(err.code ?? "?")}:${String(err.message).slice(0, 120)}` }; }
    };
    return {
      exists: await grab(() => f.exists()),
      stat: await grab(() => f.stat()),
      text: await grab(() => f.text()),
      bytes: await grab(() => f.bytes()),
      size: await grab(() => f.size),
      json: await grab(() => f.json()),
      stream: await grab(async () => { const r = (f.stream() as ReadableStream).getReader(); const n = await r.read(); r.releaseLock(); return { done: n.done }; }),
      unlink: await grab(() => f.unlink()),
      delete: await grab(() => f.delete()),
    };
  });
  await attempt("auth_config", async () => {
    const grab = async (fn: () => unknown) => {
      try { return { ok: await fn() }; }
      catch (e) { const err = e as { name?: unknown; code?: unknown; message?: unknown }; return { throws: `${String(err.name)}:${String(err.code ?? "?")}:${String(err.message).slice(0, 120)}` }; }
    };
    const badCreds = new S3Client({ endpoint, region: "us-east-1", bucket, accessKeyId: "nope", secretAccessKey: "nope" });
    const badBucket = new S3Client({ endpoint, region: "us-east-1", bucket: "no-such-bucket-zzz", accessKeyId, secretAccessKey });
    const badEndpoint = new S3Client({ endpoint: "http://127.0.0.1:9", region: "us-east-1", bucket, accessKeyId, secretAccessKey });
    return {
      badCredsStat: await grab(() => badCreds.file(`${prefix}get.bin`).stat()),
      badCredsExists: await grab(() => badCreds.file(`${prefix}get.bin`).exists()),
      badBucketStat: await grab(() => badBucket.file("x").stat()),
      badEndpointStat: await grab(() => badEndpoint.file("x").stat()),
    };
  });
  await attempt("presign_bounds", async () => {
    const grab = async (fn: () => unknown) => {
      try { const u = String(await fn()); return { ok: u.length > 0 }; }
      catch (e) { const err = e as { name?: unknown; code?: unknown; message?: unknown }; return { throws: `${String(err.name)}:${String(err.code ?? "?")}:${String(err.message).slice(0, 120)}` }; }
    };
    const missing = await fetch(String(client.presign(`${prefix}absent-presign.bin`, { expiresIn: 300 })));
    return {
      expiresZero: await grab(() => client.presign(`${prefix}get.bin`, { expiresIn: 0 })),
      expiresNegative: await grab(() => client.presign(`${prefix}get.bin`, { expiresIn: -5 })),
      expiresHuge: await grab(() => client.presign(`${prefix}get.bin`, { expiresIn: 9007199254740991 })),
      unknownMethod: await grab(() => client.presign(`${prefix}get.bin`, { method: "FOO" as "GET", expiresIn: 300 })),
      fileLevel: await grab(() => client.file(`${prefix}get.bin`).presign({ expiresIn: 300 })),
      missingKeyGetStatus: missing.status,
    };
  });
  await attempt("keys", async () => {
    await client.file(`${prefix}sp ace.txt`).write("s");
    await client.file(`${prefix}ünïcode.txt`).write("u");
    const listed = await client.list({ prefix }) as unknown as { contents?: Array<{ key?: string }> };
    return {
      space: await client.file(`${prefix}sp ace.txt`).text(),
      unicode: await client.file(`${prefix}ünïcode.txt`).text(),
      listedHasSpace: (listed.contents ?? []).some((c) => c.key === `${prefix}sp ace.txt`),
      listedHasUnicode: (listed.contents ?? []).some((c) => c.key === `${prefix}ünïcode.txt`),
    };
  });
} finally {
  const listed = await client.list({ prefix }) as unknown as { contents?: Array<{ key?: string }> };
  for (const entry of listed.contents ?? []) {
    if (entry.key) { try { await client.file(entry.key).delete(); } catch { /* best effort */ } }
  }
}
const report = {
  bun: Bun.version, revision: Bun.revision,
  service: { endpointHost: new URL(endpoint).host, bucket, prefixNote: "unique scratch prefix per run, deleted after" },
  rows,
};
await Bun.write(outPath, JSON.stringify(report, null, 1) + "\n");
console.log(`s3 probe: ${Object.keys(rows).length} rows -> ${outPath}`);
