// F05 companion: delivery worker (scheduling, concurrency, backoff,
// destination enforcement, batch reporting).
//
// The worker pages single-row claims until the outbox is empty, then
// delivers the batch under bounded concurrency: destination policy
// admission, resolved-literal recheck, credential binding, redirect
// handling, heartbeat for slow sends, idempotent ack, N-attempt poison
// routing, and step-indexed batch reports that mirror the C-A
// diagnostics shape for F06.
//
// At-least-once discipline: every row ends delivered, failed (acked
// for retry), dead (terminal), or error (left leased to redeliver
// after expiry). Fatal carrier failures (auth, version, shape, clock)
// throw CarrierFatal instead of spinning: misconfiguration must fail
// loudly under the supervisor, never retry forever.
//
// Reports never carry secrets or destination URLs: steps name the
// delivery, the finite outcome, and the policy/failure reason code.
// Operators map subscriptions to URLs from their own config.
import {
  credentialValue,
  evaluateDestination,
  evaluateRedirect,
  isLoopbackAddressLiteral,
  isPrivateAddressLiteral,
  type DestinationPolicy,
  type EnvName,
} from "../../../runtime/outbound/destination-policy.ts";
import {
  MAX_ATTEMPTS,
  PROTOCOL_VERSION,
  CarrierHttp,
  CarrierTransportError,
  createCarrierClient,
  type AckResult,
  type CarrierClient,
  type CarrierTransport,
  type ClaimItem,
  type DeadResult,
  type HeartbeatResult,
} from "./protocol.ts";

export type StepOutcome = "delivered" | "failed" | "dead" | "error";

export type FailureIdentity =
  | "ok"
  | "downstream-status"
  | "downstream-transport"
  | "downstream-timeout"
  | "credential-missing"
  | "redirect-denied"
  | "redirect-hops"
  | "poison"
  | "destination-denied"
  | "no-destination"
  | "lease-lost"
  | "protocol-error"
  | "store-unavailable";

export type StepFailure = Readonly<{
  identity: FailureIdentity;
  occurrence: string;
}>;

export type BatchStep = Readonly<{
  step: number;
  deliveryId: string;
  outcome: StepOutcome;
  attempts: number;
  version: number;
  failure: StepFailure;
}>;

export type BatchReport = Readonly<{
  protocol: typeof PROTOCOL_VERSION;
  workerId: string;
  batch: number;
  steps: readonly BatchStep[];
  batchError?: StepFailure;
}>;

// CarrierFatal marks misconfiguration and worker bugs: wrong secret,
// unknown protocol, malformed shapes, skewed clock, reused nonces.
// The supervisor restarts bounded times and then dies loudly; the
// worker never retries these inside the batch loop.
export class CarrierFatal extends Error {
  readonly occurrence: string;
  constructor(occurrence: string) {
    super(`carrier fatal: ${occurrence}`);
    this.name = "CarrierFatal";
    this.occurrence = occurrence;
  }
}

export type DownstreamResponse = Readonly<{
  status: number;
  location: string | null;
}>;

export type DownstreamPost = (
  url: string,
  body: string,
  headers: Record<string, string>,
  signal: AbortSignal,
) => Promise<DownstreamResponse>;

export type WorkerOptions = Readonly<{
  baseUrl: string;
  workerId: string;
  secret: string;
  leaseMs: number;
  maxAttempts: number;
  concurrency: number;
  maxBatch: number;
  backoffBaseMs: number;
  backoffMaxMs: number;
  idleMs: number;
  downstreamTimeoutMs: number;
  deliveries: Readonly<Record<string, string>>;
  policy: DestinationPolicy;
  readEnvironment: (key: string) => string | undefined;
  transport: CarrierTransport;
  postDownstream: DownstreamPost;
  dnsLookup: (host: string) => Promise<string>;
  now: () => number;
  sleep: (ms: number) => Promise<void>;
  batch: number;
  signal?: AbortSignal;
}>;

export function defaultWorkerOptions(
  init: Pick<
    WorkerOptions,
    | "baseUrl"
    | "workerId"
    | "secret"
    | "deliveries"
    | "policy"
    | "readEnvironment"
    | "transport"
    | "postDownstream"
    | "dnsLookup"
  > &
    Partial<WorkerOptions>,
): WorkerOptions {
  return Object.freeze({
    leaseMs: 30000,
    maxAttempts: MAX_ATTEMPTS,
    concurrency: 4,
    maxBatch: 64,
    backoffBaseMs: 1000,
    backoffMaxMs: 30000,
    idleMs: 1000,
    downstreamTimeoutMs: 10000,
    now: Date.now,
    sleep: (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms)),
    batch: 1,
    ...init,
  });
}

// backoffFor sleeps base*2^failures capped at max. Deterministic on
// purpose: retries stay reproducible in tests and in operator logs.
export function backoffFor(failures: number, baseMs: number, maxMs: number): number {
  if (!Number.isSafeInteger(failures) || failures < 0) throw new TypeError("invalid backoff count");
  if (!Number.isSafeInteger(baseMs) || baseMs < 0) throw new TypeError("invalid backoff base");
  if (!Number.isSafeInteger(maxMs) || maxMs < 0) throw new TypeError("invalid backoff cap");
  return Math.min(maxMs, baseMs * 2 ** Math.min(failures, 16));
}

function fail(identity: FailureIdentity, occurrence: string): StepFailure {
  return Object.freeze({ identity, occurrence });
}

const okFailure: StepFailure = Object.freeze({ identity: "ok", occurrence: "" });

// carrierVerdict maps a thrown carrier call to a retryable verdict or
// a fatal stop. 500/503 answers are Can-side pressure the next batch
// retries after backoff; wrapped transport failures (Can unreachable)
// retry the same way; 400/401/413 are fatal misconfiguration; parser
// violations and anything unrecognized are fatal bugs, never silent
// retries.
function carrierVerdict(cause: unknown): { kind: "store" } | { kind: "transport" } {
  if (cause instanceof CarrierFatal) throw cause;
  if (cause instanceof CarrierTransportError) return { kind: "transport" };
  if (cause instanceof CarrierHttp) {
    if (cause.status === 500 || cause.status === 503) return { kind: "store" };
    if (cause.status === 400 || cause.status === 401 || cause.status === 413)
      throw new CarrierFatal(cause.bodyStatus);
    throw new CarrierFatal(`http-${cause.status}`);
  }
  if (cause instanceof TypeError) throw new CarrierFatal("invalid-body");
  throw new CarrierFatal("carrier-error");
}

function verdictFailure(
  verdict: { kind: "store" } | { kind: "transport" },
  detail: string,
): StepFailure {
  if (verdict.kind === "store") return fail("store-unavailable", detail);
  return fail("protocol-error", detail);
}

type HeldRow = Readonly<{
  item: ClaimItem;
  leaseUntilMs: number;
  version: number;
}>;

async function takeClaim(
  client: CarrierClient,
  options: WorkerOptions,
): Promise<
  { kind: "page"; item: ClaimItem | undefined } | { kind: "store"; failure: StepFailure }
> {
  let page: { status: string; items: readonly ClaimItem[] };
  try {
    page = await client.claim(options.workerId, options.leaseMs);
  } catch (cause) {
    const verdict = carrierVerdict(cause);
    return { kind: "store", failure: verdictFailure(verdict, "claim") };
  }
  if (page.status === "claimed" && page.items.length === 1)
    return { kind: "page", item: page.items[0] };
  if (page.status === "claimed" || page.status === "empty")
    return { kind: "page", item: undefined };
  if (page.status === "store_unavailable" || page.status === "uncertain")
    return { kind: "store", failure: fail("store-unavailable", page.status) };
  throw new CarrierFatal(page.status);
}

// resolveChecked admits one send URL: policy admission, then a DNS
// recheck of the resolved literal against the loopback/private scopes.
// DNS names can resolve anywhere after the allowlist check, so the
// literal check closes the obvious hole. Residual TOCTOU between this
// lookup and connect (rebinding) is a documented limit: the policy
// still denies direct literals, and high-risk deployments pin IPs.
async function resolveChecked(
  options: WorkerOptions,
  url: string,
): Promise<
  | { kind: "ok"; credentialEnv: EnvName | undefined }
  | { kind: "denied"; reason: string }
  | { kind: "failed"; failure: StepFailure }
> {
  const decision = evaluateDestination(options.policy, url);
  if (decision.decision === "denied") return { kind: "denied", reason: decision.reason };
  let host: string;
  try {
    host = new URL(url).hostname;
  } catch {
    return { kind: "denied", reason: "invalid-url" };
  }
  let literal: string;
  try {
    literal = await options.dnsLookup(host);
  } catch {
    return { kind: "failed", failure: fail("downstream-transport", "dns") };
  }
  if (isLoopbackAddressLiteral(literal) && options.policy.loopback === "deny")
    return { kind: "denied", reason: "resolved-loopback-denied" };
  if (isPrivateAddressLiteral(literal) && options.policy.privateNetworks === "deny")
    return { kind: "denied", reason: "resolved-private-denied" };
  return { kind: "ok", credentialEnv: decision.credential };
}

async function deadStep(
  client: CarrierClient,
  options: WorkerOptions,
  held: HeldRow,
  step: number,
  reason: "poison" | "destination_denied" | "no_destination",
  failure: StepFailure,
): Promise<BatchStep> {
  const finish = (outcome: StepOutcome, attempts: number, done: StepFailure): BatchStep =>
    Object.freeze({
      step,
      deliveryId: held.item.deliveryId,
      outcome,
      attempts,
      version: held.version,
      failure: done,
    });
  let dead: DeadResult;
  try {
    dead = await client.deadLetter(held.item.deliveryId, options.workerId, reason);
  } catch (cause) {
    const verdict = carrierVerdict(cause);
    return finish("error", held.item.attempts, verdictFailure(verdict, "dead-letter"));
  }
  if (dead.status === "dead") return finish("dead", dead.attempts, failure);
  if (dead.status === "already_done")
    return finish("delivered", dead.attempts, fail("ok", "already-done"));
  if (dead.status === "lease_lost")
    return finish("error", dead.attempts, fail("lease-lost", dead.status));
  if (dead.status === "unknown")
    return finish("error", dead.attempts, fail("protocol-error", "unknown-dead"));
  if (dead.status === "store_unavailable" || dead.status === "uncertain")
    return finish("error", held.item.attempts, fail("store-unavailable", dead.status));
  throw new CarrierFatal(dead.status);
}

async function ackStep(
  client: CarrierClient,
  held: HeldRow,
  step: number,
  settled: boolean,
  failed: StepFailure,
): Promise<BatchStep> {
  const finish = (outcome: StepOutcome, attempts: number, done: StepFailure): BatchStep =>
    Object.freeze({
      step,
      deliveryId: held.item.deliveryId,
      outcome,
      attempts,
      version: held.version,
      failure: done,
    });
  let ack: AckResult;
  try {
    ack = await client.ack(held.item.deliveryId, settled);
  } catch (cause) {
    const verdict = carrierVerdict(cause);
    return finish("error", held.item.attempts, verdictFailure(verdict, "ack"));
  }
  if (ack.status === "done") return finish("delivered", ack.attempts, okFailure);
  if (ack.status === "pending") {
    if (settled)
      return finish("error", ack.attempts, fail("protocol-error", "pending-after-settled"));
    return finish("failed", ack.attempts, failed);
  }
  if (ack.status === "dead") return finish("dead", ack.attempts, fail("poison", "already-dead"));
  if (ack.status === "unknown")
    return finish("error", ack.attempts, fail("protocol-error", "unknown-ack"));
  if (ack.status === "store_unavailable" || ack.status === "uncertain")
    return finish("error", held.item.attempts, fail("store-unavailable", ack.status));
  throw new CarrierFatal(ack.status);
}

// sendDownstream POSTs one payload with manual redirect handling: each
// hop re-admits through the redirect policy, rechecks the resolved
// literal, and re-posts the same webhook body (webhook redirect
// semantics, documented in PROTOCOL.md). Redirects never inherit the
// credential header: only the configured destination is credentialed.
async function sendDownstream(
  options: WorkerOptions,
  url: string,
  body: string,
  credential: string | undefined,
  signal: AbortSignal,
  timedOut: () => boolean,
): Promise<
  | { kind: "status"; status: number }
  | { kind: "failed"; failure: StepFailure }
  | { kind: "aborted" }
> {
  let current = url;
  let boundCredential = credential;
  redirects: for (let hops = 0; ; hops += 1) {
    const headers: Record<string, string> = { "content-type": "application/json" };
    if (boundCredential !== undefined) headers["authorization"] = `Bearer ${boundCredential}`;
    let response: DownstreamResponse;
    try {
      response = await options.postDownstream(current, body, headers, signal);
    } catch {
      if (timedOut()) return { kind: "failed", failure: fail("downstream-timeout", "timeout") };
      if (signal.aborted) return { kind: "aborted" };
      return { kind: "failed", failure: fail("downstream-transport", "transport") };
    }
    if (timedOut()) return { kind: "failed", failure: fail("downstream-timeout", "timeout") };
    if (signal.aborted) return { kind: "aborted" };
    if (
      (response.status === 301 ||
        response.status === 302 ||
        response.status === 303 ||
        response.status === 307 ||
        response.status === 308) &&
      response.location !== null
    ) {
      const hop = evaluateRedirect(options.policy, current, response.location, hops);
      if (hop.decision === "denied") {
        const identity = hop.reason === "max-hops" ? "redirect-hops" : "redirect-denied";
        return { kind: "failed", failure: fail(identity, hop.reason) };
      }
      const target = hop.target ?? "";
      const recheck = await resolveChecked(options, target);
      if (recheck.kind === "denied")
        return { kind: "failed", failure: fail("redirect-denied", `target-${recheck.reason}`) };
      if (recheck.kind === "failed") return { kind: "failed", failure: recheck.failure };
      current = target;
      boundCredential = undefined;
      continue redirects;
    }
    return { kind: "status", status: response.status };
  }
}

// deliverOne runs one claimed row to a terminal step: poison routing,
// destination admission, one downstream send with heartbeat cover,
// and the settling ack. Heartbeat extends the lease while the send is
// in flight; a lost lease aborts the send and leaves the row to its
// new holder.
async function deliverOne(
  client: CarrierClient,
  options: WorkerOptions,
  held: HeldRow,
  step: number,
): Promise<BatchStep> {
  const item = held.item;
  if (item.attempts >= options.maxAttempts)
    return deadStep(client, options, held, step, "poison", fail("poison", String(item.attempts)));
  const destination = options.deliveries[item.subscription];
  if (destination === undefined)
    return deadStep(
      client,
      options,
      held,
      step,
      "no_destination",
      fail("no-destination", item.subscription),
    );
  const admitted = await resolveChecked(options, destination);
  if (admitted.kind === "denied")
    return deadStep(
      client,
      options,
      held,
      step,
      "destination_denied",
      fail("destination-denied", admitted.reason),
    );
  if (admitted.kind === "failed") return ackStep(client, held, step, false, admitted.failure);
  let credential: string | undefined;
  if (admitted.credentialEnv !== undefined) {
    credential = credentialValue(admitted.credentialEnv, options.readEnvironment);
    if (credential === undefined)
      return ackStep(client, held, step, false, fail("credential-missing", admitted.credentialEnv));
  }
  const body = JSON.stringify({
    delivery_id: item.deliveryId,
    event: item.event,
    subscription: item.subscription,
  });
  const controller = new AbortController();
  let expired = false;
  const timeout = setTimeout(() => {
    expired = true;
    controller.abort();
  }, options.downstreamTimeoutMs);
  const heartbeatAt = Math.max(1000, Math.floor(options.leaseMs / 2));
  let leaseUntilMs = held.leaseUntilMs;
  let version = held.version;
  const finish = (outcome: StepOutcome, attempts: number, done: StepFailure): BatchStep =>
    Object.freeze({ step, deliveryId: item.deliveryId, outcome, attempts, version, failure: done });
  try {
    const send = sendDownstream(
      options,
      destination,
      body,
      credential,
      controller.signal,
      () => expired,
    );
    for (;;) {
      const raced = await Promise.race([
        send.then((done) => ({ kind: "sent" as const, done })),
        options.sleep(heartbeatAt).then(() => ({ kind: "beat" as const })),
      ]);
      if (raced.kind === "sent") {
        const done = raced.done;
        if (done.kind === "aborted")
          return ackStep(
            client,
            { item, leaseUntilMs, version },
            step,
            false,
            fail("downstream-timeout", "timeout"),
          );
        if (done.kind === "failed")
          return ackStep(client, { item, leaseUntilMs, version }, step, false, done.failure);
        if (done.status === 200)
          return ackStep(client, { item, leaseUntilMs, version }, step, true, okFailure);
        return ackStep(
          client,
          { item, leaseUntilMs, version },
          step,
          false,
          fail("downstream-status", String(done.status)),
        );
      }
      let beat: HeartbeatResult;
      try {
        beat = await client.heartbeat(options.workerId, item.deliveryId, version, options.leaseMs);
      } catch (cause) {
        controller.abort();
        const verdict = carrierVerdict(cause);
        return finish("error", item.attempts, verdictFailure(verdict, "heartbeat"));
      }
      if (beat.status === "extended") {
        leaseUntilMs = beat.leaseUntilMs;
        version = beat.version;
        continue;
      }
      controller.abort();
      if (beat.status === "done")
        return finish("delivered", beat.attempts, fail("ok", "already-done"));
      if (beat.status === "dead")
        return finish("dead", beat.attempts, fail("poison", "already-dead"));
      if (beat.status === "lease_lost")
        return finish("error", beat.attempts, fail("lease-lost", beat.status));
      if (beat.status === "unknown")
        return finish("error", beat.attempts, fail("protocol-error", "unknown-heartbeat"));
      if (beat.status === "store_unavailable" || beat.status === "uncertain")
        return finish("error", item.attempts, fail("store-unavailable", beat.status));
      throw new CarrierFatal(beat.status);
    }
  } finally {
    clearTimeout(timeout);
  }
}

async function eachBounded<T>(
  items: readonly T[],
  limit: number,
  run: (item: T, index: number) => Promise<BatchStep>,
): Promise<BatchStep[]> {
  const bound = Math.max(1, Math.floor(limit));
  const done: { index: number; step: BatchStep }[] = [];
  let next = 0;
  async function lane(): Promise<void> {
    for (;;) {
      const index = next;
      next += 1;
      if (index >= items.length) return;
      done.push({ index, step: await run(items[index], index) });
    }
  }
  const lanes: Promise<void>[] = [];
  for (let slot = 0; slot < Math.min(bound, items.length); slot += 1) lanes.push(lane());
  await Promise.all(lanes);
  done.sort((left, right) => left.index - right.index);
  return done.map((entry) => entry.step);
}

// runBatch pages claims until the outbox is empty (or maxBatch rows
// are held), then delivers under bounded concurrency. Steps keep
// claim order with 1-based indexes. A store-level claim failure ends
// paging with batchError; already-held rows still deliver, so one
// slow store never strands a claimed lease.
export async function runBatch(options: WorkerOptions): Promise<BatchReport> {
  const client = createCarrierClient(options.transport, options.secret, options.now);
  const held: HeldRow[] = [];
  let batchError: StepFailure | undefined;
  for (let page = 0; page < options.maxBatch; page += 1) {
    if (options.signal?.aborted) break;
    const taken = await takeClaim(client, options);
    if (taken.kind === "store") {
      batchError = taken.failure;
      break;
    }
    if (taken.item === undefined) break;
    held.push({
      item: taken.item,
      leaseUntilMs: taken.item.leaseUntilMs,
      version: taken.item.version,
    });
  }
  const steps =
    held.length === 0
      ? []
      : await eachBounded(held, options.concurrency, (row, index) =>
          deliverOne(client, options, row, index + 1),
        );
  steps.sort((left, right) => left.step - right.step);
  return Object.freeze({
    protocol: PROTOCOL_VERSION,
    workerId: options.workerId,
    batch: options.batch,
    steps: Object.freeze(steps),
    ...(batchError === undefined ? {} : { batchError }),
  });
}

// runWorker loops batches until aborted: idle outboxes sleep idleMs,
// failure batches back off exponentially, clean batches reset the
// ladder. onReport observes every batch (logging, tests). Fatal
// carrier failures propagate to the supervisor; the loop never
// swallows them into another batch.
export async function runWorker(
  options: WorkerOptions,
  onReport: (report: BatchReport) => void = () => {},
): Promise<void> {
  let failures = 0;
  let batch = options.batch;
  for (;;) {
    if (options.signal?.aborted) return;
    const report = await runBatch({ ...options, batch });
    onReport(report);
    batch += 1;
    if (options.signal?.aborted) return;
    const clean =
      report.batchError === undefined &&
      report.steps.every((step) => step.outcome === "delivered" || step.outcome === "dead");
    if (report.steps.length === 0 && report.batchError === undefined) {
      failures = 0;
      await options.sleep(options.idleMs);
      continue;
    }
    if (clean) {
      failures = 0;
      continue;
    }
    const wait = backoffFor(failures, options.backoffBaseMs, options.backoffMaxMs);
    failures += 1;
    await options.sleep(wait);
  }
}
