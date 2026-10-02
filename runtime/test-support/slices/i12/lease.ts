// NT-I12 inherited generation-lease probe adapter. Builds the
// descriptor.probe_lease envelope for the external N owner (JSON codec
// contract) and maps the probe verdict back to the nominal
// descriptor::lease_report record. Wire rule (merge-defined, the N
// owner honors it): replies carry the probe report under facts.lease;
// the adapter round-trips the path echo verbatim and never invents
// verdicts. The transport is injected: the default dispatch throws,
// so no call can silently succeed before the N owner lands (P14/P23
// scope). A gone lease path is a completed outcome with
// released=true (the K19 exclusive-probe verdict relayed), not an
// error. Non-completed outcomes return Can failures:
// wrong-owner/stale-handle/closed-handle/not-found map to
// test::stale_handle; every other kind maps verbatim to
// descriptor::descriptor_fault{kind, reason}. The envelope,
// canonical digest, admission and routing helpers copy the NT-I11
// descriptor adapter so this slice stays disjoint from it.
import { createHash, randomBytes } from "node:crypto";
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { record } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";

const TRANSPORT_DEFAULT_DEADLINE_MS = 30000;

const origin = Object.freeze({
  source: "can:descriptor",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type LeaseProbeErrors = Readonly<{
  staleHandle: string;
  descriptorFault: string;
  leaseReport: string;
}>;

export type LeaseProbeDispatch = (
  request: Readonly<Record<string, unknown>>,
) => Promise<Readonly<Record<string, unknown>>>;

function defaultDispatch(): Promise<Readonly<Record<string, unknown>>> {
  return Promise.reject(new Error("descriptor N transport is not bound"));
}

// canonicalize copies the K02 digestFacts algorithm (sorted keys, compact
// JSON) so arguments_digest matches the owner computation.
function canonicalize(value: unknown): string {
  if (value === null) return "null";
  if (typeof value === "number" || typeof value === "string" || typeof value === "boolean") {
    return JSON.stringify(value) as string;
  }
  if (Array.isArray(value)) {
    return `[${value.map((item) => canonicalize(item)).join(",")}]`;
  }
  if (typeof value === "object") {
    const entries = Object.keys(value as Record<string, unknown>)
      .sort()
      .map(
        (key) => `${JSON.stringify(key)}:${canonicalize((value as Record<string, unknown>)[key])}`,
      )
      .join(",");
    return `{${entries}}`;
  }
  throw new TypeError("descriptor arguments_digest needs inert args");
}

function digestArgs(args: Readonly<Record<string, unknown>>): string {
  return `sha256:${createHash("sha256").update(canonicalize(args), "utf8").digest("hex")}`;
}

function hex(bytes: number): string {
  return randomBytes(bytes).toString("hex");
}

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

type Outcome =
  | { readonly ok: true; readonly facts: Record<string, unknown> }
  | { readonly ok: false; readonly completion: Completion<never> };

const STALE_KINDS = new Set(["wrong-owner", "stale-handle", "closed-handle", "not-found"]);

export function createLeaseProbe(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: LeaseProbeErrors,
  owner: TestOwner,
  dispatch: LeaseProbeDispatch = defaultDispatch,
) {
  const runId = `run-${hex(8)}`;
  let operationSeq = 0;

  function failStale(handleTag: string): Completion<never> {
    return failure(
      domain.create(
        errors.staleHandle,
        record(errors.staleHandle, [["handle", handleTag]]),
        origin,
      ),
    );
  }

  function failFault(kind: string, reason: string): Completion<never> {
    return failure(
      domain.create(
        errors.descriptorFault,
        record(errors.descriptorFault, [
          ["kind", kind],
          ["reason", reason],
        ]),
        origin,
      ),
    );
  }

  function admitOwner(ownerHandle: unknown, what: string): string | Completion<never> {
    const tag = ownerTag(ownerHandle);
    if (tag === null) throw new TypeError(`${what} needs a test::owner handle`);
    const grant = owner.grantOf(ownerHandle);
    if (grant === null) return failStale(tag);
    return grant;
  }

  function envelope(
    operation: string,
    args: Readonly<Record<string, unknown>>,
    grant: string,
    deadlineMs: number,
  ): Readonly<Record<string, unknown>> {
    operationSeq += 1;
    return Object.freeze({
      schema_version: "1",
      run_id: runId,
      operation_id: `op-${operationSeq}-${hex(4)}`,
      owner_grant: grant,
      operation,
      arguments_digest: digestArgs(args),
      deadline_ms: Object.freeze({ clock: "n-monotonic", ms: deadlineMs }),
      args: Object.freeze({ ...args }),
    });
  }

  function route(operation: string, reply: Readonly<Record<string, unknown>>): Outcome {
    if (reply["outcome"] !== "completed") {
      const kind = typeof reply["kind"] === "string" ? (reply["kind"] as string) : "?";
      const rawReason = reply["message"];
      const reason =
        typeof rawReason === "string" && rawReason !== ""
          ? rawReason
          : `${typeof reply["outcome"] === "string" ? (reply["outcome"] as string) : "?"}/${kind}`;
      if (STALE_KINDS.has(kind)) return { ok: false, completion: failStale("lease") };
      return { ok: false, completion: failFault(kind, reason) };
    }
    const facts = reply["facts"];
    if (facts === null || typeof facts !== "object" || Array.isArray(facts)) {
      throw new TypeError(`descriptor.${operation} reply carries no facts map`);
    }
    return { ok: true, facts: facts as Record<string, unknown> };
  }

  function leaseRecord(raw: unknown): unknown {
    if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
      throw new TypeError("descriptor reply carries a malformed lease report");
    }
    const fields = raw as Record<string, unknown>;
    return record(errors.leaseReport, [
      ["path", readStr(fields["path"], "lease.path")],
      ["inherited", fields["inherited"] as boolean],
      ["held", fields["held"] as boolean],
      ["released", fields["released"] as boolean],
    ]);
  }

  return Object.freeze({
    async probeLease(
      ownerHandle: unknown,
      path: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "descriptor::probe_lease");
      if (typeof admitted !== "string") return admitted;
      const args = { path: readStr(path, "descriptor::probe_lease path") };
      const routed = route(
        "probe_lease",
        await dispatch(
          envelope("descriptor.probe_lease", args, admitted, TRANSPORT_DEFAULT_DEADLINE_MS),
        ),
      );
      if (!routed.ok) return routed.completion;
      return success(leaseRecord(routed.facts["lease"]));
    },
  });
}
