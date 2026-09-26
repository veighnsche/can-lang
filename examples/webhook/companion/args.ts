// F06 companion: operator CLI parsing for main.ts.
//
// --base/--worker/--config/--policy select the Can side, the worker
// identity, and the operator-owned config files; --once runs a single
// batch and exits. The tuning flags mirror defaultWorkerOptions so
// live qualification (and operators) can shorten leases, bound
// batches, and shrink sleeps without editing code; the defaults below
// must equal worker.ts, pinned by args.test.ts.
//
// Every value is validated here, before any file or network touch:
// unknown flags and positionals reject (a typo'd flag must never
// silently run with defaults), the lease mirrors the protocol bounds
// so an unclaimable ask fails locally, and worker/base keep the F05
// admission rules. main.ts prints the thrown message plus usage and
// exits 2.
import { LEASE_MAX_MS, LEASE_MIN_MS } from "./protocol.ts";

export type CompanionArgs = Readonly<{
  baseUrl: string;
  workerId: string;
  configPath: string;
  policyPath: string;
  once: boolean;
  leaseMs: number;
  concurrency: number;
  maxBatch: number;
  backoffBaseMs: number;
  backoffMaxMs: number;
  idleMs: number;
  downstreamTimeoutMs: number;
}>;

export function usage(): string {
  return "usage: main.ts --base <url> --worker <id> --config <path> --policy <path> [--once] [--lease-ms <n>] [--concurrency <n>] [--max-batch <n>] [--backoff-base-ms <n>] [--backoff-max-ms <n>] [--idle-ms <n>] [--timeout-ms <n>]";
}

const VALUE_FLAGS = new Set([
  "--base",
  "--worker",
  "--config",
  "--policy",
  "--lease-ms",
  "--concurrency",
  "--max-batch",
  "--backoff-base-ms",
  "--backoff-max-ms",
  "--idle-ms",
  "--timeout-ms",
]);

function parseCount(raw: string, name: string, min: number, max: number): number {
  if (!/^[0-9]+$/.test(raw)) throw new Error(`${name} must be an integer`);
  const value = Number(raw);
  if (!Number.isSafeInteger(value) || value < min || value > max)
    throw new Error(`${name} must be ${min}..${max}`);
  return value;
}

function checkBase(raw: string): string {
  let base: URL;
  try {
    base = new URL(raw);
  } catch {
    throw new Error("base is not a URL");
  }
  if (base.protocol !== "http:" && base.protocol !== "https:")
    throw new Error("base must be http(s)");
  return raw;
}

function checkWorker(raw: string): string {
  if (raw.length < 1 || raw.length > 128)
    throw new Error("worker id must be 1..128 characters");
  return raw;
}

export function parseCompanionArgs(args: readonly string[]): CompanionArgs {
  const seen = new Map<string, string>();
  let once = false;
  for (let index = 0; index < args.length; index += 1) {
    const token = args[index];
    if (token === "--once") {
      once = true;
      continue;
    }
    if (!VALUE_FLAGS.has(token)) {
      if (token.startsWith("--")) throw new Error(`unknown flag ${token}`);
      throw new Error(`unexpected argument ${token}`);
    }
    const value = args[index + 1];
    if (value === undefined || value.startsWith("--")) throw new Error(`${token} needs a value`);
    index += 1;
    seen.set(token, value);
  }
  const baseUrl = seen.get("--base");
  const workerId = seen.get("--worker");
  const configPath = seen.get("--config");
  const policyPath = seen.get("--policy");
  if (
    baseUrl === undefined ||
    workerId === undefined ||
    configPath === undefined ||
    policyPath === undefined
  )
    throw new Error("missing required --base/--worker/--config/--policy");
  const at = (name: string): string | undefined => seen.get(name);
  return Object.freeze({
    baseUrl: checkBase(baseUrl),
    workerId: checkWorker(workerId),
    configPath,
    policyPath,
    once,
    leaseMs:
      at("--lease-ms") === undefined
        ? 30000
        : parseCount(at("--lease-ms")!, "--lease-ms", LEASE_MIN_MS, LEASE_MAX_MS),
    concurrency:
      at("--concurrency") === undefined ? 4 : parseCount(at("--concurrency")!, "--concurrency", 1, 1024),
    maxBatch:
      at("--max-batch") === undefined ? 64 : parseCount(at("--max-batch")!, "--max-batch", 1, 1024),
    backoffBaseMs:
      at("--backoff-base-ms") === undefined
        ? 1000
        : parseCount(at("--backoff-base-ms")!, "--backoff-base-ms", 0, 3600000),
    backoffMaxMs:
      at("--backoff-max-ms") === undefined
        ? 30000
        : parseCount(at("--backoff-max-ms")!, "--backoff-max-ms", 0, 3600000),
    idleMs:
      at("--idle-ms") === undefined ? 1000 : parseCount(at("--idle-ms")!, "--idle-ms", 0, 3600000),
    downstreamTimeoutMs:
      at("--timeout-ms") === undefined
        ? 10000
        : parseCount(at("--timeout-ms")!, "--timeout-ms", 1, 3600000),
  });
}
