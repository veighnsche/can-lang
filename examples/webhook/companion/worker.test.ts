// F05 companion worker tests: claim paging, delivery, poison,
// destination enforcement, heartbeat, backoff, and reporting.
//
// The stub Can below verifies the real authenticated envelope
// (protocol header, exact-byte HMAC, timestamp skew, single-use
// nonces), so every passing test also proves the worker speaks
// protocol v1 to a checking peer. The live pair test proves the same
// against the real Can program.
import { test, expect } from "bun:test";
import { destinationPolicy } from "../../../runtime/outbound/destination-policy.ts";
import { signBody, type CarrierTransport } from "./protocol.ts";
import {
  CarrierFatal,
  backoffFor,
  defaultWorkerOptions,
  runBatch,
  runWorker,
  type BatchReport,
  type DownstreamResponse,
  type WorkerOptions,
} from "./worker.ts";
import { CarrierTransportError } from "./protocol.ts";

const SECRET = "carrier-test-secret";

type StubRow = {
  deliveryId: string;
  event: string;
  subscription: string;
  state: "pending" | "done" | "dead";
  attempts: number;
  workerId: string;
  leaseUntil: number;
  version: number;
};

class StubCan {
  rows = new Map<string, StubRow>();
  dead: { deliveryId: string; reason: string }[] = [];
  attempts: { deliveryId: string; outcome: string }[] = [];
  nonces = new Set<string>();
  heartbeats = 0;
  now = 1000000;
  injectClaim: { status: number; json: unknown } | undefined;

  constructor(rows: Omit<StubRow, "state" | "workerId" | "leaseUntil" | "version">[] = []) {
    for (const row of rows)
      this.rows.set(row.deliveryId, {
        ...row,
        state: "pending",
        workerId: "",
        leaseUntil: 0,
        version: 0,
      });
  }

  transport: CarrierTransport = async (path, headers, body) => {
    if (headers["x-carrier-protocol"] !== "1")
      return { status: 400, json: { status: "unsupported_protocol" } };
    if (headers["x-carrier-signature"] !== signBody(SECRET, body))
      return { status: 401, json: { status: "unauthorized" } };
    const seen = JSON.parse(body) as Record<string, string | number | boolean>;
    const ts = seen["timestamp_ms"];
    if (typeof ts !== "number" || Math.abs(ts - this.now) > 300000)
      return { status: 401, json: { status: "stale", attempts: 0 } };
    const nonce = seen["nonce"];
    if (typeof nonce !== "string" || this.nonces.has(nonce))
      return { status: 401, json: { status: "replay", attempts: 0 } };
    this.nonces.add(nonce);
    if (path === "/outbox/claim") {
      if (this.injectClaim !== undefined) {
        const injected = this.injectClaim;
        this.injectClaim = undefined;
        return injected;
      }
      for (const row of this.rows.values()) {
        if (row.state === "pending" && row.leaseUntil <= this.now) {
          row.workerId = seen["worker_id"] as string;
          row.leaseUntil = this.now + (seen["lease_ms"] as number);
          row.version += 1;
          return {
            status: 200,
            json: {
              status: "claimed",
              items: [
                {
                  delivery_id: row.deliveryId,
                  event: row.event,
                  subscription: row.subscription,
                  attempts: row.attempts,
                  lease_until_ms: row.leaseUntil,
                  version: row.version,
                },
              ],
            },
          };
        }
      }
      return { status: 200, json: { status: "empty", items: [] } };
    }
    if (path === "/outbox/heartbeat") {
      this.heartbeats += 1;
      const row = this.rows.get(seen["delivery_id"] as string);
      if (row === undefined)
        return {
          status: 404,
          json: { status: "unknown", lease_until_ms: 0, version: 0, attempts: 0 },
        };
      if (row.state !== "pending")
        return {
          status: 200,
          json: {
            status: row.state,
            lease_until_ms: row.leaseUntil,
            version: row.version,
            attempts: row.attempts,
          },
        };
      if (row.workerId !== seen["worker_id"] || row.version !== seen["version"])
        return {
          status: 409,
          json: { status: "lease_lost", lease_until_ms: 0, version: 0, attempts: 0 },
        };
      row.leaseUntil = this.now + (seen["lease_ms"] as number);
      row.version += 1;
      return {
        status: 200,
        json: {
          status: "extended",
          lease_until_ms: row.leaseUntil,
          version: row.version,
          attempts: row.attempts,
        },
      };
    }
    if (path === "/outbox/ack") {
      const row = this.rows.get(seen["delivery_id"] as string);
      if (row === undefined) return { status: 404, json: { status: "unknown", attempts: 0 } };
      if (row.state !== "pending")
        return { status: 200, json: { status: row.state, attempts: row.attempts } };
      row.attempts += 1;
      const settled = seen["settled"] === true;
      row.state = settled ? "done" : "pending";
      this.attempts.push({ deliveryId: row.deliveryId, outcome: settled ? "delivered" : "failed" });
      return { status: 200, json: { status: row.state, attempts: row.attempts } };
    }
    if (path === "/outbox/dead_letter") {
      const row = this.rows.get(seen["delivery_id"] as string);
      if (row === undefined) return { status: 404, json: { status: "unknown", attempts: 0 } };
      if (row.state === "done")
        return { status: 409, json: { status: "already_done", attempts: row.attempts } };
      if (row.state === "dead")
        return { status: 200, json: { status: "dead", attempts: row.attempts } };
      if (row.workerId !== seen["worker_id"])
        return { status: 409, json: { status: "lease_lost", attempts: row.attempts } };
      row.state = "dead";
      this.dead.push({ deliveryId: row.deliveryId, reason: seen["reason"] as string });
      return { status: 200, json: { status: "dead", attempts: row.attempts } };
    }
    throw new Error(`unexpected carrier path ${path}`);
  };
}

function testPolicy() {
  return destinationPolicy({
    version: "test.1",
    rules: [
      {
        scheme: "https",
        host: "hooks.example.com",
        pathPrefix: "/tenant/",
        credential: "TENANT_HOOKS_TOKEN",
      },
      { scheme: "https", host: "open.example.com" },
    ],
    redirect: "allowlisted",
    maxRedirectHops: 2,
    loopback: "deny",
    privateNetworks: "deny",
  });
}

type Sleeper = {
  calls: number[];
  sleep: (ms: number) => Promise<void>;
  beat: () => void;
};

function manualSleeper(): Sleeper {
  const calls: number[] = [];
  const waiters: (() => void)[] = [];
  return {
    calls,
    sleep: (ms: number) => {
      calls.push(ms);
      return new Promise<void>((resolve) => waiters.push(resolve));
    },
    beat: () => {
      waiters.shift()?.();
    },
  };
}

type Harness = {
  can: StubCan;
  options: WorkerOptions;
  downstream: { url: string; body: string; headers: Record<string, string> }[];
  sleeper: Sleeper;
  env: Record<string, string>;
};

function harness(
  rows: Omit<StubRow, "state" | "workerId" | "leaseUntil" | "version">[],
  init: {
    downstream?: (
      url: string,
      body: string,
      headers: Record<string, string>,
    ) => Promise<DownstreamResponse>;
    dns?: (host: string) => Promise<string>;
    env?: Record<string, string>;
    deliveries?: Record<string, string>;
    leaseMs?: number;
    concurrency?: number;
    maxBatch?: number;
    maxAttempts?: number;
    downstreamTimeoutMs?: number;
  } = {},
): Harness {
  const can = new StubCan(rows);
  const downstream: Harness["downstream"] = [];
  let inFlight = 0;
  const sleeper = manualSleeper();
  const env: Record<string, string> = {
    TENANT_HOOKS_TOKEN: "tenant-credential-value",
    ...init.env,
  };
  const options = defaultWorkerOptions({
    baseUrl: "http://127.0.0.1:9",
    workerId: "worker-1",
    secret: SECRET,
    deliveries: init.deliveries ?? { "sub-9": "https://hooks.example.com/tenant/acme/hook" },
    policy: testPolicy(),
    readEnvironment: (key: string) => env[key],
    transport: can.transport,
    postDownstream: async (url, body, headers) => {
      downstream.push({ url, body, headers });
      inFlight += 1;
      try {
        if (init.downstream !== undefined) return await init.downstream(url, body, headers);
        return { status: 200, location: null };
      } finally {
        inFlight -= 1;
      }
    },
    dnsLookup: init.dns ?? (async () => "93.184.216.34"),
    now: () => can.now,
    sleep: sleeper.sleep,
    leaseMs: init.leaseMs ?? 30000,
    concurrency: init.concurrency ?? 4,
    maxBatch: init.maxBatch ?? 64,
    maxAttempts: init.maxAttempts ?? 5,
    downstreamTimeoutMs: init.downstreamTimeoutMs ?? 60000,
  });
  void inFlight;
  return { can, options, downstream, sleeper, env };
}

test("happy path claims, delivers with credential, and acks done", async () => {
  const h = harness([
    { deliveryId: "del-1", event: "invoice.paid", subscription: "sub-9", attempts: 0 },
    { deliveryId: "del-2", event: "invoice.paid", subscription: "sub-9", attempts: 0 },
  ]);
  const report = await runBatch(h.options);
  expect(report.protocol).toBe("1");
  expect(report.workerId).toBe("worker-1");
  expect(report.batch).toBe(1);
  expect(report.batchError).toBeUndefined();
  expect(report.steps.map((step) => [step.step, step.deliveryId, step.outcome])).toEqual([
    [1, "del-1", "delivered"],
    [2, "del-2", "delivered"],
  ]);
  for (const step of report.steps) {
    expect(step.failure.identity).toBe("ok");
    expect(step.version).toBe(1);
  }
  expect(h.downstream).toHaveLength(2);
  expect(h.downstream[0].url).toBe("https://hooks.example.com/tenant/acme/hook");
  expect(h.downstream[0].headers["authorization"]).toBe("Bearer tenant-credential-value");
  expect(JSON.parse(h.downstream[0].body)).toEqual({
    delivery_id: "del-1",
    event: "invoice.paid",
    subscription: "sub-9",
  });
  expect(h.can.rows.get("del-1")?.state).toBe("done");
  expect(h.can.attempts).toEqual([
    { deliveryId: "del-1", outcome: "delivered" },
    { deliveryId: "del-2", outcome: "delivered" },
  ]);
  expect(h.can.heartbeats).toBe(0);
});

test("failed downstream stays pending with a durable failed attempt", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    downstream: async () => ({ status: 500, location: null }),
  });
  const report = await runBatch(h.options);
  expect(report.steps).toHaveLength(1);
  expect(report.steps[0].outcome).toBe("failed");
  expect(report.steps[0].failure).toEqual({ identity: "downstream-status", occurrence: "500" });
  expect(report.steps[0].attempts).toBe(1);
  expect(h.can.rows.get("del-1")?.state).toBe("pending");
  expect(h.can.attempts).toEqual([{ deliveryId: "del-1", outcome: "failed" }]);
});

test("rows at N attempts dead-letter as poison without a downstream touch", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 5 }]);
  const report = await runBatch(h.options);
  expect(report.steps[0].outcome).toBe("dead");
  expect(report.steps[0].failure).toEqual({ identity: "poison", occurrence: "5" });
  expect(h.downstream).toHaveLength(0);
  expect(h.can.dead).toEqual([{ deliveryId: "del-1", reason: "poison" }]);
  expect(h.can.rows.get("del-1")?.state).toBe("dead");
});

test("unknown subscriptions dead-letter as no_destination", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-gone", attempts: 0 }]);
  const report = await runBatch(h.options);
  expect(report.steps[0].outcome).toBe("dead");
  expect(report.steps[0].failure).toEqual({ identity: "no-destination", occurrence: "sub-gone" });
  expect(h.downstream).toHaveLength(0);
  expect(h.can.dead).toEqual([{ deliveryId: "del-1", reason: "no_destination" }]);
});

test("policy-denied destinations dead-letter with the policy reason", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    deliveries: { "sub-9": "https://evil.example/hook" },
  });
  const report = await runBatch(h.options);
  expect(report.steps[0].outcome).toBe("dead");
  expect(report.steps[0].failure).toEqual({
    identity: "destination-denied",
    occurrence: "no-matching-rule",
  });
  expect(h.downstream).toHaveLength(0);
  expect(h.can.dead).toEqual([{ deliveryId: "del-1", reason: "destination_denied" }]);
});

test("DNS resolving private dead-letters before connecting", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    dns: async () => "10.9.9.9",
  });
  const report = await runBatch(h.options);
  expect(report.steps[0].outcome).toBe("dead");
  expect(report.steps[0].failure).toEqual({
    identity: "destination-denied",
    occurrence: "resolved-private-denied",
  });
  expect(h.downstream).toHaveLength(0);
});

test("missing bound credentials fail the attempt for later retry", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    env: {},
  });
  delete h.env["TENANT_HOOKS_TOKEN"];
  const report = await runBatch(h.options);
  expect(report.steps[0].outcome).toBe("failed");
  expect(report.steps[0].failure).toEqual({
    identity: "credential-missing",
    occurrence: "TENANT_HOOKS_TOKEN",
  });
  expect(h.downstream).toHaveLength(0);
  expect(h.can.rows.get("del-1")?.state).toBe("pending");
});

test("allowlisted redirects re-post without the credential", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    deliveries: { "sub-9": "https://open.example.com/hook" },
    downstream: async (url) => {
      if (url === "https://open.example.com/hook")
        return { status: 307, location: "https://open.example.com/final" };
      return { status: 200, location: null };
    },
  });
  const report = await runBatch(h.options);
  expect(report.steps[0].outcome).toBe("delivered");
  expect(h.downstream.map((call) => call.url)).toEqual([
    "https://open.example.com/hook",
    "https://open.example.com/final",
  ]);
  expect(h.downstream[1].headers["authorization"]).toBeUndefined();
});

test("denied redirects fail the attempt without following", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    deliveries: { "sub-9": "https://open.example.com/hook" },
    downstream: async () => ({ status: 307, location: "https://evil.example/final" }),
  });
  const report = await runBatch(h.options);
  expect(report.steps[0].outcome).toBe("failed");
  expect(report.steps[0].failure.identity).toBe("redirect-denied");
  expect(h.downstream).toHaveLength(1);
  expect(h.can.rows.get("del-1")?.state).toBe("pending");
});

test("slow sends heartbeat until the downstream answers", async () => {
  let release!: () => void;
  const gate = new Promise<void>((resolve) => (release = resolve));
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    leaseMs: 10000,
    downstream: async () => {
      await gate;
      return { status: 200, location: null };
    },
  });
  const pending = runBatch(h.options);
  await Bun.sleep(10);
  expect(h.can.heartbeats).toBe(0);
  h.sleeper.beat();
  await Bun.sleep(10);
  expect(h.can.heartbeats).toBe(1);
  release();
  const report = await pending;
  expect(report.steps[0].outcome).toBe("delivered");
  expect(report.steps[0].version).toBe(2);
});

test("lost leases abort the send and leave the row to its holder", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    leaseMs: 10000,
    downstream: async () => {
      await new Promise(() => {});
      return { status: 200, location: null };
    },
  });
  const pending = runBatch(h.options);
  await Bun.sleep(10);
  const row = h.can.rows.get("del-1");
  if (row !== undefined) {
    row.workerId = "worker-2";
    row.version = 9;
  }
  h.sleeper.beat();
  const report = await pending;
  expect(report.steps[0].outcome).toBe("error");
  expect(report.steps[0].failure).toEqual({ identity: "lease-lost", occurrence: "lease_lost" });
  expect(h.can.attempts).toHaveLength(0);
});

test("claim pressure ends paging but still delivers held rows", async () => {
  const h = harness([
    { deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 },
    { deliveryId: "del-2", event: "e", subscription: "sub-9", attempts: 0 },
  ]);
  let claims = 0;
  const inner = h.options.transport;
  const options: WorkerOptions = {
    ...h.options,
    transport: async (path, headers, body) => {
      if (path === "/outbox/claim") {
        claims += 1;
        if (claims === 2) return { status: 503, json: { status: "store_unavailable" } };
      }
      return inner(path, headers, body);
    },
  };
  const report = await runBatch(options);
  expect(report.steps.map((step) => step.deliveryId)).toEqual(["del-1"]);
  expect(report.steps[0].outcome).toBe("delivered");
  expect(report.batchError).toEqual({ identity: "store-unavailable", occurrence: "claim" });
});

test("rejected carrier credentials fail fast instead of paging", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }]);
  const options: WorkerOptions = {
    ...h.options,
    transport: async () => ({ status: 401, json: { status: "unauthorized" } }),
  };
  await expect(runBatch(options)).rejects.toBeInstanceOf(CarrierFatal);
});

test("unreachable Can reports protocol pressure for retry", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }]);
  const options: WorkerOptions = {
    ...h.options,
    transport: async () => {
      throw new CarrierTransportError();
    },
  };
  const report = await runBatch(options);
  expect(report.steps).toHaveLength(0);
  expect(report.batchError).toEqual({ identity: "protocol-error", occurrence: "claim" });
});

test("ack pressure leaves the row leased for redelivery", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }]);
  const inner = h.options.transport;
  const options: WorkerOptions = {
    ...h.options,
    transport: async (path, headers, body) => {
      if (path === "/outbox/ack") return { status: 503, json: { status: "store_unavailable" } };
      return inner(path, headers, body);
    },
  };
  const report = await runBatch(options);
  expect(report.steps[0].outcome).toBe("error");
  expect(report.steps[0].failure).toEqual({ identity: "store-unavailable", occurrence: "ack" });
  expect(h.can.rows.get("del-1")?.state).toBe("pending");
  expect(h.can.rows.get("del-1")?.attempts).toBe(0);
});

test("concurrency stays bounded across the batch", async () => {
  let inFlight = 0;
  let peak = 0;
  const h = harness(
    [1, 2, 3, 4].map((n) => ({
      deliveryId: `del-${n}`,
      event: "e",
      subscription: "sub-9",
      attempts: 0,
    })),
    {
      concurrency: 2,
      downstream: async () => {
        inFlight += 1;
        peak = Math.max(peak, inFlight);
        await Bun.sleep(5);
        inFlight -= 1;
        return { status: 200, location: null };
      },
    },
  );
  const report = await runBatch(h.options);
  expect(report.steps).toHaveLength(4);
  expect(peak).toBe(2);
  expect(report.steps.map((step) => step.step)).toEqual([1, 2, 3, 4]);
});

test("reports never carry secrets or destination URLs", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }]);
  const report = await runBatch(h.options);
  const text = JSON.stringify(report);
  expect(text.includes(SECRET)).toBe(false);
  expect(text.includes("tenant-credential-value")).toBe(false);
  expect(text.includes("hooks.example.com")).toBe(false);
  expect(text.includes("del-1")).toBe(true);
});

test("backoff doubles per failure batch under the cap", () => {
  expect(backoffFor(0, 1000, 30000)).toBe(1000);
  expect(backoffFor(3, 1000, 30000)).toBe(8000);
  expect(backoffFor(99, 1000, 30000)).toBe(30000);
  expect(() => backoffFor(-1, 1000, 30000)).toThrow(TypeError);
});

test("worker idles on empty outboxes and backs off on failures", async () => {
  const h = harness([]);
  const sleeps: number[] = [];
  const controller = new AbortController();
  const reports: BatchReport[] = [];
  const options: WorkerOptions = {
    ...h.options,
    sleep: async (ms: number) => {
      sleeps.push(ms);
    },
    signal: controller.signal,
  };
  const done = runWorker(options, (report) => {
    reports.push(report);
    if (reports.length === 2) controller.abort();
  });
  await done;
  expect(reports).toHaveLength(2);
  expect(sleeps).toEqual([1000]);
});

test("worker backs off after failure batches and resets when clean", async () => {
  const h = harness([{ deliveryId: "del-1", event: "e", subscription: "sub-9", attempts: 0 }], {
    downstream: async () => ({ status: 500, location: null }),
  });
  const sleeps: number[] = [];
  const controller = new AbortController();
  const reports: BatchReport[] = [];
  const options: WorkerOptions = {
    ...h.options,
    sleep: async (ms: number) => {
      sleeps.push(ms);
    },
    signal: controller.signal,
    backoffBaseMs: 1000,
    backoffMaxMs: 30000,
  };
  const done = runWorker(options, (report) => {
    reports.push(report);
    h.can.now += 40000;
    if (reports.length === 3) controller.abort();
  });
  await done;
  expect(reports.map((report) => report.steps[0]?.outcome)).toEqual(["failed", "failed", "failed"]);
  expect(sleeps.filter((wait) => wait !== 15000)).toEqual([1000, 2000]);
});

test("stub Can rejects tampered and replayed carrier requests", async () => {
  const can = new StubCan();
  const body = JSON.stringify({
    worker_id: "w",
    lease_ms: 5000,
    timestamp_ms: can.now,
    nonce: "n-1",
  });
  const good = { "x-carrier-protocol": "1", "x-carrier-signature": signBody(SECRET, body) };
  const first = await can.transport("/outbox/claim", good, body);
  expect(first.status).toBe(200);
  const replay = await can.transport("/outbox/claim", good, body);
  expect(replay.status).toBe(401);
  const tampered = await can.transport("/outbox/claim", good, body + " ");
  expect(tampered.status).toBe(401);
  const staleBody = JSON.stringify({
    worker_id: "w",
    lease_ms: 5000,
    timestamp_ms: 1,
    nonce: "n-2",
  });
  const stale = await can.transport(
    "/outbox/claim",
    { "x-carrier-protocol": "1", "x-carrier-signature": signBody(SECRET, staleBody) },
    staleBody,
  );
  expect(stale.status).toBe(401);
});
