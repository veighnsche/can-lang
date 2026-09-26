// E07 (X-R15-1/X-R15-3): sink event-loop pin child probe.
//
// Runs ONE scenario selected by argv[2], prints exactly one JSON line,
// then falls off the end with NO explicit exit. The parent leg
// distinguishes release (exit 0) from pin (alive at watchdog). Env
// credentials only; argv carries the mode alone. Blackhole/fault
// servers are in-process so each child is self-contained.
import { startTap } from "./s3-e07-tap.ts";

const mode = process.argv[2] ?? "missing";
const ENDPOINT = process.env["CAN_TEST_S3_ENDPOINT"] ?? "";
const BUCKET = process.env["CAN_TEST_S3_BUCKET"] ?? "can-b1-10";
const ACCESS = process.env["CAN_TEST_S3_ACCESS_KEY"] ?? "";
const SECRET = process.env["CAN_TEST_S3_SECRET_KEY"] ?? "";

const say = (record: unknown): void => console.log(JSON.stringify(record));

const openClient = (endpoint: string): InstanceType<typeof Bun.S3Client> =>
  new Bun.S3Client({
    endpoint,
    region: "us-east-1",
    bucket: BUCKET,
    accessKeyId: ACCESS,
    secretAccessKey: SECRET,
  });

// Accept-and-hold blackhole: connections succeed, nothing responds.
function startBlackhole(): { port: number; close: () => void } {
  const server = Bun.listen({
    hostname: "127.0.0.1",
    port: 0,
    socket: {
      open() {},
      data() {},
      drain() {},
      close() {},
      error() {},
    },
  });
  return {
    port: server.port,
    close: () => server.stop(true),
  };
}

const keyFor = (stem: string): string => `s3e07/e07harness/pin-${stem}-${process.pid}.bin`;
const bytes = (text: string): Uint8Array => new TextEncoder().encode(text);

if (mode === "bare-end" || mode === "error-end") {
  const key = keyFor(mode);
  const writer = openClient(ENDPOINT).file(key).writer({ type: "application/octet-stream" });
  await writer.write(bytes("pin-probe"));
  const settled =
    mode === "bare-end"
      ? String(await writer.end())
      : String(await writer.end(new Error("e07-abort")));
  say({ mode, end: settled, key });
} else if (mode === "write-abandon") {
  const writer = openClient(ENDPOINT)
    .file(keyFor(mode))
    .writer({ type: "application/octet-stream" });
  await writer.write(bytes("pin-probe"));
  say({ mode, abandoned: true });
} else if (mode === "writer-abandon") {
  openClient(ENDPOINT).file(keyFor(mode)).writer({ type: "application/octet-stream" });
  say({ mode, abandoned: true });
} else if (mode === "hung-flush-abandon") {
  const hole = startBlackhole();
  const key = keyFor(mode);
  const big = new Uint8Array(6 * 1024 * 1024);
  big.fill(9);
  const writer = openClient(`http://127.0.0.1:${hole.port}`).file(key).writer({
    type: "application/octet-stream",
  });
  await writer.write(big);
  // Deliberately un-awaited: the part upload hangs in the blackhole.
  void Promise.resolve(writer.flush()).then(
    () => say({ mode, late: "flush-resolved" }),
    () => say({ mode, late: "flush-rejected" }),
  );
  say({ mode, pending: "flush", key });
} else if (mode === "hung-end-abandon") {
  const upstream = new URL(ENDPOINT);
  const tap = startTap(upstream.hostname, Number(upstream.port));
  const key = keyFor(mode);
  tap.setRules([{ action: "blackhole", match: (op) => op === "complete-mpu" }]);
  const big = new Uint8Array(6 * 1024 * 1024);
  big.fill(9);
  const writer = openClient(`http://127.0.0.1:${tap.port}`)
    .file(key)
    .writer({ type: "application/octet-stream" });
  await writer.write(big);
  say({ mode, flushed: String(await writer.flush()), key });
  // Deliberately un-awaited: CompleteMultipartUpload is blackholed.
  void Promise.resolve(writer.end()).then(
    () => console.log(JSON.stringify({ mode, late: "end-resolved" })),
    () => console.log(JSON.stringify({ mode, late: "end-rejected" })),
  );
} else if (mode === "hung-stat-abandon") {
  const hole = startBlackhole();
  const key = keyFor(mode);
  const file = openClient(`http://127.0.0.1:${hole.port}`).file(key);
  // Deliberately un-awaited: the HEAD hangs in the blackhole.
  void file.stat().then(
    () => console.log(JSON.stringify({ mode, late: "stat-resolved" })),
    () => console.log(JSON.stringify({ mode, late: "stat-rejected" })),
  );
  say({ mode, pending: "stat", key });
} else if (mode === "end-error-during-hung-flush") {
  const hole = startBlackhole();
  const key = keyFor(mode);
  const big = new Uint8Array(6 * 1024 * 1024);
  big.fill(9);
  const writer = openClient(`http://127.0.0.1:${hole.port}`).file(key).writer({
    type: "application/octet-stream",
  });
  await writer.write(big);
  void Promise.resolve(writer.flush()).then(
    () => {},
    () => {},
  );
  await Bun.sleep(500);
  const outcome = await Promise.race([
    Promise.resolve(writer.end(new Error("e07-abort"))).then(
      (value) => `resolved:${String(value)}`,
      (cause: unknown) => `rejected:${cause instanceof Error ? cause.name : typeof cause}`,
    ),
    Bun.sleep(5000).then(() => "WATCHDOG"),
  ]);
  say({ mode, outcome, key });
  // Fall off the end: the hung flush plus the un-ended writer decide
  // whether the loop releases.
} else {
  say({ mode, error: "unknown-mode" });
}
