// F06 companion: CLI parsing tests. Pins the tuning defaults against
// defaultWorkerOptions (args.ts and worker.ts must agree), plus every
// rejection: missing required flags, unknown flags, positionals,
// out-of-range tuning, and the F05 worker/base admission rules.
import { describe, expect, test } from "bun:test";
import { defaultWorkerOptions } from "./worker.ts";
import { parseCompanionArgs } from "./args.ts";

const REQUIRED = ["--base", "http://127.0.0.1:18490", "--worker", "worker-1", "--config", "deliveries.json", "--policy", "policy.json"];

function stubWorker() {
  return defaultWorkerOptions({
    baseUrl: "http://127.0.0.1:18490",
    workerId: "worker-1",
    secret: "secret",
    deliveries: {},
    policy: { version: "test", rules: [], redirect: "deny", maxRedirectHops: 0, loopback: "deny", privateNetworks: "deny" },
    readEnvironment: () => undefined,
    transport: async () => ({ status: 500, json: null }),
    postDownstream: async () => ({ status: 500, location: null }),
    dnsLookup: async (host: string) => host,
  });
}

describe("parseCompanionArgs", () => {
  test("required flags parse with worker defaults", () => {
    const parsed = parseCompanionArgs(REQUIRED);
    const worker = stubWorker();
    expect(parsed.once).toBe(false);
    expect(parsed.leaseMs).toBe(worker.leaseMs);
    expect(parsed.concurrency).toBe(worker.concurrency);
    expect(parsed.maxBatch).toBe(worker.maxBatch);
    expect(parsed.backoffBaseMs).toBe(worker.backoffBaseMs);
    expect(parsed.backoffMaxMs).toBe(worker.backoffMaxMs);
    expect(parsed.idleMs).toBe(worker.idleMs);
    expect(parsed.downstreamTimeoutMs).toBe(worker.downstreamTimeoutMs);
  });

  test("tuning flags override every default", () => {
    const parsed = parseCompanionArgs([
      ...REQUIRED,
      "--once",
      "--lease-ms", "2000",
      "--concurrency", "1",
      "--max-batch", "8",
      "--backoff-base-ms", "100",
      "--backoff-max-ms", "500",
      "--idle-ms", "250",
      "--timeout-ms", "1500",
    ]);
    expect(parsed.once).toBe(true);
    expect(parsed.leaseMs).toBe(2000);
    expect(parsed.concurrency).toBe(1);
    expect(parsed.maxBatch).toBe(8);
    expect(parsed.backoffBaseMs).toBe(100);
    expect(parsed.backoffMaxMs).toBe(500);
    expect(parsed.idleMs).toBe(250);
    expect(parsed.downstreamTimeoutMs).toBe(1500);
  });

  test("missing required flags reject", () => {
    expect(() => parseCompanionArgs(["--base", "http://127.0.0.1:18490"])).toThrow("missing required");
  });

  test("unknown flags and positionals reject", () => {
    expect(() => parseCompanionArgs([...REQUIRED, "--bogus"])).toThrow("unknown flag --bogus");
    expect(() => parseCompanionArgs([...REQUIRED, "stray"])).toThrow("unexpected argument stray");
    expect(() => parseCompanionArgs([...REQUIRED, "--lease-ms"])).toThrow("--lease-ms needs a value");
  });

  test("out-of-range tuning rejects", () => {
    expect(() => parseCompanionArgs([...REQUIRED, "--lease-ms", "999"])).toThrow("--lease-ms must be 1000..600000");
    expect(() => parseCompanionArgs([...REQUIRED, "--lease-ms", "600001"])).toThrow("--lease-ms must be 1000..600000");
    expect(() => parseCompanionArgs([...REQUIRED, "--concurrency", "0"])).toThrow("--concurrency must be 1..1024");
    expect(() => parseCompanionArgs([...REQUIRED, "--max-batch", "0"])).toThrow("--max-batch must be 1..1024");
    expect(() => parseCompanionArgs([...REQUIRED, "--timeout-ms", "0"])).toThrow("--timeout-ms must be 1..3600000");
    expect(() => parseCompanionArgs([...REQUIRED, "--idle-ms", "nope"])).toThrow("--idle-ms must be an integer");
  });

  test("worker and base keep F05 admission", () => {
    expect(() => parseCompanionArgs([...REQUIRED.slice(0, 3), "", ...REQUIRED.slice(4)])).toThrow(
      "worker id must be 1..128 characters",
    );
    expect(() =>
      parseCompanionArgs(["--base", "notaurl", ...REQUIRED.slice(2)]),
    ).toThrow("base is not a URL");
    expect(() =>
      parseCompanionArgs(["--base", "ftp://127.0.0.1:1", ...REQUIRED.slice(2)]),
    ).toThrow("base must be http(s)");
  });
});
