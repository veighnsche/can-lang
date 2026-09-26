// F05 companion: operator entry point.
//
// Usage:
//   CARRIER_SECRET=... bun companion/main.ts --base http://127.0.0.1:18490 \
//     --worker worker-1 --config deliveries.json --policy policy.json [--once]
//
// --config maps subscriptions to destination URLs, e.g.
// {"deliveries": {"sub-9": "https://hooks.example.com/tenant/acme/hook"}}.
// --policy is a C-G destination policy (see runtime/outbound/
// destination-policy.ts and the shipped companion-policy.json
// fixture). --once runs a single batch and exits; otherwise the
// supervised worker loops until SIGTERM/SIGINT.
//
// Secrets arrive via environment only: CARRIER_SECRET authenticates
// the Can side, and per-rule downstream credentials resolve from
// their bound variables at send time. Nothing secret is logged:
// batch reports print as JSON lines on stdout, diagnostics on stderr.
import { destinationPolicy } from "../../../runtime/outbound/destination-policy.ts";
import { CarrierHttp, CarrierTransportError, type CarrierTransport } from "./protocol.ts";
import {
  defaultWorkerOptions,
  runBatch,
  runWorker,
  type BatchReport,
  type WorkerOptions,
} from "./worker.ts";
import { defaultSupervisorOptions, runSupervised } from "./supervisor.ts";

function usage(): string {
  return "usage: main.ts --base <url> --worker <id> --config <path> --policy <path> [--once]";
}

function flag(args: readonly string[], name: string): string | undefined {
  const index = args.indexOf(name);
  if (index < 0 || index + 1 >= args.length) return undefined;
  return args[index + 1];
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

async function loadDeliveries(path: string): Promise<Record<string, string>> {
  let raw: unknown;
  try {
    raw = JSON.parse(await Bun.file(path).text());
  } catch {
    throw new Error(`invalid deliveries file ${path}`);
  }
  if (!isRecord(raw) || !isRecord(raw["deliveries"]))
    throw new Error(`invalid deliveries file ${path}`);
  const entries = raw["deliveries"];
  for (const [subscription, url] of Object.entries(entries)) {
    if (subscription === "" || typeof url !== "string" || url === "")
      throw new Error(`invalid deliveries file ${path}`);
    try {
      new URL(url);
    } catch {
      throw new Error(`invalid deliveries file ${path}`);
    }
  }
  return entries as Record<string, string>;
}

function httpTransport(baseUrl: string): CarrierTransport {
  const base = baseUrl.endsWith("/") ? baseUrl.slice(0, -1) : baseUrl;
  return async (path, headers, body) => {
    let response: Response;
    try {
      response = await fetch(base + path, { method: "POST", headers, body });
    } catch {
      throw new CarrierTransportError();
    }
    let json: unknown = null;
    try {
      json = await response.json();
    } catch {
      json = null;
    }
    return { status: response.status, json };
  };
}

async function main(args: readonly string[]): Promise<number> {
  const baseUrl = flag(args, "--base");
  const workerId = flag(args, "--worker");
  const configPath = flag(args, "--config");
  const policyPath = flag(args, "--policy");
  const once = args.includes("--once");
  if (
    baseUrl === undefined ||
    workerId === undefined ||
    configPath === undefined ||
    policyPath === undefined
  ) {
    console.error(usage());
    return 2;
  }
  const secret = process.env["CARRIER_SECRET"];
  if (secret === undefined || secret === "") {
    console.error("main.ts: CARRIER_SECRET is not set");
    return 2;
  }
  let deliveries: Record<string, string>;
  let policy: ReturnType<typeof destinationPolicy>;
  try {
    deliveries = await loadDeliveries(configPath);
    policy = destinationPolicy(JSON.parse(await Bun.file(policyPath).text()));
  } catch (cause) {
    console.error(`main.ts: ${cause instanceof Error ? cause.message : String(cause)}`);
    return 2;
  }
  if (workerId.length < 1 || workerId.length > 128) {
    console.error("main.ts: worker id must be 1..128 characters");
    return 2;
  }
  let base: URL;
  try {
    base = new URL(baseUrl);
  } catch {
    console.error("main.ts: base is not a URL");
    return 2;
  }
  if (base.protocol !== "http:" && base.protocol !== "https:") {
    console.error("main.ts: base must be http(s)");
    return 2;
  }
  const controller = new AbortController();
  const stop = (): void => controller.abort();
  process.on("SIGTERM", stop);
  process.on("SIGINT", stop);
  const report = (seen: BatchReport): void => console.log(JSON.stringify(seen));
  const options: WorkerOptions = defaultWorkerOptions({
    baseUrl: base.href,
    workerId,
    secret,
    deliveries,
    policy,
    readEnvironment: (key: string) => process.env[key],
    transport: httpTransport(base.href),
    postDownstream: async (url, body, headers, signal) => {
      const response = await fetch(url, {
        method: "POST",
        headers,
        body,
        signal,
        redirect: "manual",
      });
      await response.arrayBuffer().catch(() => undefined);
      return { status: response.status, location: response.headers.get("location") };
    },
    dnsLookup: async (host: string) => {
      if (/^[0-9.]+$/.test(host) || host.includes(":")) return host;
      return (await Bun.dns.lookup(host))[0]?.address ?? host;
    },
    signal: controller.signal,
  });
  const supervised = defaultSupervisorOptions({
    onRestart: (restart: number, cause: unknown) => {
      const detail = cause instanceof CarrierHttp ? `http ${cause.status}` : "failure";
      console.error(`main.ts: restart ${restart} after ${detail}`);
    },
  });
  try {
    if (once) {
      report(await runBatch(options));
      return 0;
    }
    await runSupervised(() => runWorker(options, report), supervised);
    return 0;
  } catch (cause) {
    if (cause instanceof CarrierHttp)
      console.error(`main.ts: carrier http ${cause.status}: ${cause.bodyStatus}`);
    else console.error(`main.ts: ${cause instanceof Error ? cause.message : String(cause)}`);
    return 1;
  } finally {
    process.off("SIGTERM", stop);
    process.off("SIGINT", stop);
  }
}

const status = await main(Bun.argv.slice(2));
process.exit(status);
