// E07 (X-R15-1/X-R15-3): S3 fault-injection tap and orphan-accounting helpers.
//
// A TCP-level forward proxy to the provisioned S3-protocol endpoint
// (MinIO; S3-protocol scope, not AWS-real). Bytes pass through
// unmodified so SigV4 survives; the tap classifies each request head,
// applies per-operation fault/blackhole rules, and logs sanitized
// request lines (no credentials, no signatures, upload IDs elided).
// The SigV4 list/abort helpers below account orphaned multipart
// uploads; the signer output is cross-checked against `aws s3api` in
// the E07 evidence record.
import { createHash, createHmac } from "node:crypto";

export type TapVerdict = "forwarded" | "fault500" | "blackholed";

export type TapOp = Readonly<{
  conn: number;
  method: string;
  op: string;
  target: string;
  bodyLen: number;
  verdict: TapVerdict;
}>;

export type TapAction = "fault500" | "blackhole";

export type TapRule = Readonly<{
  action: TapAction;
  match: (op: string, method: string, rawTarget: string) => boolean;
}>;

export type Tap = Readonly<{
  port: number;
  ops: readonly TapOp[];
  conns: () => number;
  desyncs: () => number;
  setRules: (rules: readonly TapRule[]) => void;
  clearOps: () => void;
  // Reset every currently blackholed downstream connection so the
  // client's hung awaits fail fast (service-drop recovery shape);
  // returns the number of connections reset.
  dropHeld: () => number;
  close: () => void;
}>;

const hasParam = (target: string, name: string): boolean => {
  const q = target.indexOf("?");
  if (q < 0) return false;
  return target
    .slice(q + 1)
    .split("&")
    .some((pair) => pair.split("=")[0] === name);
};

export function classifyOp(method: string, rawTarget: string): string {
  if (method === "POST" && hasParam(rawTarget, "uploads")) return "create-mpu";
  if (method === "PUT" && hasParam(rawTarget, "partNumber")) return "upload-part";
  if (method === "POST" && hasParam(rawTarget, "uploadId")) return "complete-mpu";
  if (method === "DELETE" && hasParam(rawTarget, "uploadId")) return "abort-mpu";
  if (method === "DELETE") return "delete-object";
  if (method === "PUT") return "put-object";
  if (method === "HEAD") return "head-object";
  if (method === "GET" && hasParam(rawTarget, "uploads")) return "list-mpu";
  if (method === "GET") return "get-object";
  return "other";
}

export function sanitizeTarget(rawTarget: string): string {
  const q = rawTarget.indexOf("?");
  if (q < 0) return rawTarget;
  const path = rawTarget.slice(0, q);
  const kept = rawTarget
    .slice(q + 1)
    .split("&")
    .filter((pair) => !/^X-Amz-/i.test(pair) && !/^Signature=/i.test(pair))
    .map((pair) => (/^uploadId=/i.test(pair) ? "uploadId=…" : pair));
  return kept.length > 0 ? `${path}?${kept.join("&")}` : path;
}

type Pump = (bytes: Uint8Array) => void;

// Bun Socket.write may accept fewer bytes than offered; the remainder
// waits for `drain`. Dropping the return value silently truncates
// large part uploads, so every direction pumps through a queue.
function makePump(
  write: (bytes: Uint8Array) => number,
  onFlush: (flush: () => void) => void,
): Pump {
  let queue: Uint8Array[] = [];
  const flush = (): void => {
    while (queue.length > 0) {
      const head = queue[0]!;
      const accepted = write(head);
      if (accepted < head.length) {
        queue[0] = head.slice(Math.max(0, accepted));
        return;
      }
      queue.shift();
    }
  };
  onFlush(flush);
  return (bytes: Uint8Array): void => {
    if (queue.length === 0) {
      const accepted = write(bytes);
      if (accepted >= bytes.length) return;
      queue.push(bytes.slice(Math.max(0, accepted)));
      return;
    }
    queue.push(bytes);
  };
}

const FAULT_XML =
  '<?xml version="1.0" encoding="UTF-8"?><Error><Code>InternalError</Code><Message>E07 injected fault</Message></Error>';

type ConnState = {
  ready: boolean;
  toUp: Pump;
  pending: Uint8Array[];
  buf: Buffer;
  skipBody: number;
  decidedVerdict: TapVerdict | undefined;
  opaque: boolean;
};

export function startTap(upstreamHost: string, upstreamPort: number): Tap {
  const ops: TapOp[] = [];
  let connCount = 0;
  let desyncCount = 0;
  let rules: readonly TapRule[] = [];
  const liveUpstream: { end: () => void }[] = [];
  const liveDownstream: { end: () => void }[] = [];

  const decide = (method: string, rawTarget: string): TapAction | undefined => {
    const op = classifyOp(method, rawTarget);
    for (const rule of rules) {
      if (rule.match(op, method, rawTarget)) return rule.action;
    }
    return undefined;
  };
  const heldDownstream = new Set<{ end: () => void }>();

  const server = Bun.listen({
    hostname: "127.0.0.1",
    port: 0,
    socket: {
      open(sock) {
        const id = ++connCount;
        const box = sock as unknown as {
          __st: ConnState;
          __flushDown: () => void;
          __upSock: { end: () => void } | undefined;
        };
        const st: ConnState = {
          ready: false,
          toUp: () => {},
          pending: [],
          buf: Buffer.alloc(0),
          skipBody: 0,
          decidedVerdict: undefined,
          opaque: false,
        };
        box.__st = st;
        let flushDown: () => void = () => {};
        box.__flushDown = () => flushDown();
        liveDownstream.push(sock);
        const toDown = makePump(
          (bytes) => sock.write(bytes),
          (flush) => (flushDown = flush),
        );

        const feed = (bytes: Uint8Array): void => {
          if (st.decidedVerdict === "fault500" || st.decidedVerdict === "blackholed") {
            // Taken over: bodies of faulted/blackholed requests are
            // consumed for logging only, never forwarded.
            st.buf = Buffer.concat([st.buf, bytes]);
            drainParser(true);
            return;
          }
          st.buf = Buffer.concat([st.buf, bytes]);
          drainParser(false);
        };

        const drainParser = (takenOver: boolean): void => {
          for (;;) {
            if (st.opaque) {
              if (!takenOver) forwardAll();
              return;
            }
            if (st.skipBody > 0) {
              const take = Math.min(st.skipBody, st.buf.length);
              if (!takenOver) forwardBytes(st.buf.subarray(0, take));
              st.buf = st.buf.subarray(take);
              st.skipBody -= take;
              if (st.buf.length === 0) return;
              continue;
            }
            const end = st.buf.indexOf("\r\n\r\n");
            if (end < 0) return;
            const head = st.buf.subarray(0, end).toString("latin1");
            const lines = head.split("\r\n");
            const parts = (lines[0] ?? "").split(" ");
            const method = parts[0] ?? "?";
            const rawTarget = parts[1] ?? "?";
            const op = classifyOp(method, rawTarget);
            let bodyLen = 0;
            let chunked = false;
            for (const line of lines.slice(1)) {
              const i = line.indexOf(":");
              if (i < 0) continue;
              const key = line.slice(0, i).trim().toLowerCase();
              const val = line.slice(i + 1).trim();
              if (key === "content-length") bodyLen = Number.parseInt(val, 10) || 0;
              if (key === "transfer-encoding" && /chunked/i.test(val)) chunked = true;
            }
            if (chunked) {
              // The observed client always sends Content-Length; a
              // chunked request cannot be framed by this tap, so the
              // rest of this connection forwards blindly and the leg
              // integrity check (desyncs() === 0) fails loudly.
              desyncCount++;
              ops.push({
                conn: id,
                method,
                op: "CHUNKED-DESYNC",
                target: sanitizeTarget(rawTarget),
                bodyLen: -1,
                verdict: "forwarded",
              });
              st.opaque = true;
              if (!takenOver) forwardAll();
              return;
            }
            const action = takenOver ? undefined : decide(method, rawTarget);
            const verdict: TapVerdict =
              action === "fault500"
                ? "fault500"
                : action === "blackhole"
                  ? "blackholed"
                  : "forwarded";
            ops.push({
              conn: id,
              method,
              op,
              target: sanitizeTarget(rawTarget),
              bodyLen,
              verdict,
            });
            const headLen = end + 4;
            if (action === "fault500") {
              st.decidedVerdict = "fault500";
              st.buf = st.buf.subarray(headLen);
              st.skipBody = bodyLen;
              const body = Buffer.from(FAULT_XML, "utf8");
              const response =
                `HTTP/1.1 500 Internal Server Error\r\nContent-Type: application/xml\r\n` +
                `Content-Length: ${body.length}\r\nConnection: close\r\n\r\n`;
              toDown(Buffer.from(response, "latin1"));
              toDown(body);
              // Delayed FIN: the 500 above must be delivered first.
              setTimeout(() => {
                try {
                  sock.end();
                } catch {
                  /* already closing */
                }
              }, 100);
              try {
                box.__upSock?.end();
              } catch {
                /* already closing */
              }
              return;
            }
            if (action === "blackhole") {
              st.decidedVerdict = "blackholed";
              heldDownstream.add(sock);
              st.buf = st.buf.subarray(headLen);
              st.skipBody = bodyLen;
              return;
            }
            if (!takenOver) forwardBytes(st.buf.subarray(0, headLen));
            st.buf = st.buf.subarray(headLen);
            st.skipBody = bodyLen;
          }
        };

        const forwardBytes = (bytes: Uint8Array): void => {
          if (bytes.length === 0) return;
          if (st.ready) st.toUp(bytes);
          else st.pending.push(bytes.slice());
        };
        const forwardAll = (): void => {
          forwardBytes(st.buf);
          st.buf = Buffer.alloc(0);
        };
        (box as unknown as { __feed: (bytes: Uint8Array) => void }).__feed = feed;

        Bun.connect({
          hostname: upstreamHost,
          port: upstreamPort,
          socket: {
            data(_upSock, data) {
              toDown(data);
            },
            drain(upSock) {
              (upSock as unknown as { __flush?: () => void }).__flush?.();
            },
            close() {
              try {
                sock.end();
              } catch {
                /* already closing */
              }
            },
            error() {},
          },
        }).then(
          (up) => {
            const toUp = makePump(
              (bytes) => up.write(bytes),
              (flush) => ((up as unknown as { __flush: () => void }).__flush = flush),
            );
            const held = st.pending;
            st.pending = [];
            st.toUp = toUp;
            st.ready = true;
            box.__upSock = up;
            liveUpstream.push(up);
            for (const chunk of held) toUp(chunk);
          },
          () => {
            try {
              sock.end();
            } catch {
              /* already closing */
            }
          },
        );
      },
      data(sock, data) {
        const fed = (sock as unknown as { __feed?: (bytes: Uint8Array) => void }).__feed;
        try {
          fed?.(data);
        } catch {
          desyncCount++;
        }
      },
      drain(sock) {
        (sock as unknown as { __flushDown?: () => void }).__flushDown?.();
      },
      close(sock) {
        heldDownstream.delete(sock);
      },
      error() {},
    },
  });

  return Object.freeze({
    port: server.port,
    ops,
    conns: () => connCount,
    desyncs: () => desyncCount,
    setRules: (next: readonly TapRule[]): void => {
      rules = next;
    },
    clearOps: (): void => {
      ops.length = 0;
    },
    dropHeld: (): number => {
      const held = [...heldDownstream];
      heldDownstream.clear();
      for (const sock of held) {
        try {
          sock.end();
        } catch {
          /* already closing */
        }
      }
      return held.length;
    },
    close: (): void => {
      for (const sock of liveUpstream) {
        try {
          sock.end();
        } catch {
          /* best effort */
        }
      }
      for (const sock of liveDownstream) {
        try {
          sock.end();
        } catch {
          /* best effort */
        }
      }
      server.stop(true);
    },
  });
}

// Minimal SigV4 for bucket-level GET (ListMultipartUploads) and key
// DELETE with uploadId (AbortMultipartUpload), path-style, against
// the provisioned endpoint. Env credentials only; nothing logged.
const enc = (value: string): string =>
  encodeURIComponent(value).replace(
    /[!'()*]/g,
    (c) => `%${c.charCodeAt(0).toString(16).toUpperCase()}`,
  );

function signV4(
  method: string,
  path: string,
  query: readonly (readonly [string, string])[],
  host: string,
  region: string,
  accessKey: string,
  secretKey: string,
): Record<string, string> {
  const stamp = `${new Date().toISOString().replace(/[-:.]/g, "").slice(0, 15)}Z`;
  const date = stamp.slice(0, 8);
  const canonicalQuery = [...query]
    .sort(([a], [b]) => (a < b ? -1 : a > b ? 1 : 0))
    .map(([key, val]) => `${enc(key)}=${enc(val)}`)
    .join("&");
  const canonicalHeaders = `host:${host}\nx-amz-content-sha256:UNSIGNED-PAYLOAD\nx-amz-date:${stamp}\n`;
  const canonical = [
    method,
    path,
    canonicalQuery,
    canonicalHeaders,
    "host;x-amz-content-sha256;x-amz-date",
    "UNSIGNED-PAYLOAD",
  ].join("\n");
  const scope = `${date}/${region}/s3/aws4_request`;
  const stringToSign = [
    "AWS4-HMAC-SHA256",
    stamp,
    scope,
    createHash("sha256").update(canonical).digest("hex"),
  ].join("\n");
  const keyDate = createHmac("sha256", `AWS4${secretKey}`).update(date).digest();
  const keyRegion = createHmac("sha256", keyDate).update(region).digest();
  const keyService = createHmac("sha256", keyRegion).update("s3").digest();
  const keySigning = createHmac("sha256", keyService).update("aws4_request").digest();
  const signature = createHmac("sha256", keySigning).update(stringToSign).digest("hex");
  return {
    "x-amz-date": stamp,
    "x-amz-content-sha256": "UNSIGNED-PAYLOAD",
    authorization:
      `AWS4-HMAC-SHA256 Credential=${accessKey}/${scope}, ` +
      `SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=${signature}`,
  };
}

export type S3Identity = Readonly<{
  endpoint: string;
  region: string;
  bucket: string;
  accessKey: string;
  secretKey: string;
}>;

export type PendingUpload = Readonly<{ key: string; uploadId: string }>;

// MinIO honors neither `prefix` nor `max-uploads` on this API (E07
// evidence), so callers filter client-side by key prefix.
export async function listUploads(identity: S3Identity): Promise<PendingUpload[]> {
  const host = new URL(identity.endpoint).host;
  const path = `/${identity.bucket}/`;
  const headers = signV4(
    "GET",
    path,
    [["uploads", ""]],
    host,
    identity.region,
    identity.accessKey,
    identity.secretKey,
  );
  const response = await fetch(`${identity.endpoint}${path}?uploads=`, { headers });
  const xml = await response.text();
  if (!response.ok) throw new Error(`list-uploads failed with status ${response.status}`);
  const out: PendingUpload[] = [];
  for (const match of xml.matchAll(/<Upload>([\s\S]*?)<\/Upload>/g)) {
    const key = /<Key>(.*?)<\/Key>/.exec(match[1]!)?.[1];
    const uploadId = /<UploadId>(.*?)<\/UploadId>/.exec(match[1]!)?.[1];
    if (key !== undefined && uploadId !== undefined) out.push({ key, uploadId });
  }
  return out;
}

export type ListedPart = Readonly<{ number: string; size: number }>;

export async function listParts(
  identity: S3Identity,
  key: string,
  uploadId: string,
): Promise<{ status: number; parts: ListedPart[] }> {
  const host = new URL(identity.endpoint).host;
  const path = `/${identity.bucket}/${key.split("/").map(enc).join("/")}`;
  const headers = signV4(
    "GET",
    path,
    [["uploadId", uploadId]],
    host,
    identity.region,
    identity.accessKey,
    identity.secretKey,
  );
  const response = await fetch(`${identity.endpoint}${path}?uploadId=${enc(uploadId)}`, {
    headers,
  });
  const xml = await response.text();
  if (!response.ok) return { status: response.status, parts: [] };
  const parts: ListedPart[] = [];
  for (const match of xml.matchAll(/<Part>([\s\S]*?)<\/Part>/g)) {
    const number = /<PartNumber>(.*?)<\/PartNumber>/.exec(match[1]!)?.[1];
    const size = /<Size>(.*?)<\/Size>/.exec(match[1]!)?.[1];
    if (number !== undefined && size !== undefined) parts.push({ number, size: Number(size) });
  }
  return { status: response.status, parts };
}

// MinIO returns 204 even for a nonexistent upload (E07 evidence),
// so the status is not cleanup proof: legs must verify with a
// follow-up listing instead. AWS-real legs would assert 404
// NoSuchUpload on ghosts; that semantic needs AWS-real qualification.
export async function abortUpload(
  identity: S3Identity,
  key: string,
  uploadId: string,
): Promise<number> {
  const host = new URL(identity.endpoint).host;
  const path = `/${identity.bucket}/${key.split("/").map(enc).join("/")}`;
  const headers = signV4(
    "DELETE",
    path,
    [["uploadId", uploadId]],
    host,
    identity.region,
    identity.accessKey,
    identity.secretKey,
  );
  const response = await fetch(`${identity.endpoint}${path}?uploadId=${enc(uploadId)}`, {
    method: "DELETE",
    headers,
  });
  await response.text();
  return response.status;
}
