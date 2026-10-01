// NT-I11 descriptor-delivery adapter. Builds descriptor operation envelopes
// for the external N owner (JSON codec contract) and maps result facts back
// to nominal Can records and the opaque launch handle. Wire rule
// (merge-defined, the N owner honors it): replies carry the launch wire
// object under facts.launch and operation payloads under facts.exit,
// facts.facts, facts.report and facts.ack; the adapter round-trips wire
// objects verbatim and never invents ids. The transport is injected: the
// default dispatch throws, so no call can silently succeed before the N
// owner lands (P14/P23 scope). Non-completed outcomes return Can failures:
// wrong-owner/stale-handle/closed-handle/not-found map to
// test::stale_handle; every other kind maps verbatim to
// descriptor::descriptor_fault{kind, reason}.
import { createHash, randomBytes } from "node:crypto";
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, dataArray, dataProperty, record } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";

const launchBrand = Symbol("can.descriptor.launch");

const MAX_LAUNCHES = 1024;
const TRANSPORT_DEFAULT_DEADLINE_MS = 30000;

const origin = Object.freeze({
  source: "can:descriptor",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type DescriptorDeliveryErrors = Readonly<{
  staleHandle: string;
  descriptorFault: string;
  some: string;
  none: string;
  exit: string;
  facts: string;
  leaseReport: string;
  report: string;
}>;

export type DescriptorDispatch = (
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

function readInt(value: unknown, what: string): number {
  if (typeof value !== "bigint") throw new TypeError(`${what} needs an int`);
  return Number(value);
}

function readBool(value: unknown, what: string): boolean {
  if (typeof value !== "boolean") throw new TypeError(`${what} needs a bool`);
  return value;
}

function readStrArray(value: unknown, what: string): string[] {
  return dataArray(value, what).map((entry) => readStr(entry, `${what} entry`));
}

function optionalOf(someIdentity: string, noneIdentity: string, value: unknown): unknown {
  if (value === null || value === undefined) return record(noneIdentity, []);
  return record(someIdentity, [["value", value]]);
}

function tagOf(value: unknown): string {
  return typeof value === "string" ? value : "?";
}

type Outcome =
  | { readonly ok: true; readonly facts: Record<string, unknown> }
  | { readonly ok: false; readonly completion: Completion<never> };

const STALE_KINDS = new Set(["wrong-owner", "stale-handle", "closed-handle", "not-found"]);

export function createDescriptorDelivery(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: DescriptorDeliveryErrors,
  owner: TestOwner,
  dispatch: DescriptorDispatch = defaultDispatch,
) {
  const runId = `run-${hex(8)}`;
  let operationSeq = 0;
  const launches = new Map<number, { wire: unknown; grant: string }>();
  let nextId = 0;

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

  function mintLaunch(wire: unknown, grant: string): unknown {
    if (launches.size >= MAX_LAUNCHES) throw new TypeError("descriptor launch table cap reached");
    if (wire === null || typeof wire !== "object" || Array.isArray(wire)) {
      throw new TypeError("descriptor launch reply carries no wire handle");
    }
    nextId += 1;
    launches.set(nextId, { wire, grant });
    return Object.freeze({ [launchBrand]: nextId });
  }

  function launchRow(value: unknown, what: string): { wire: unknown; grant: string } | null {
    if (typeof value !== "object" || value === null) {
      throw new TypeError(`${what} needs a descriptor::launch handle`);
    }
    const id = (value as Record<symbol, unknown>)[launchBrand];
    if (typeof id !== "number") throw new TypeError(`${what} needs a descriptor::launch handle`);
    return launches.get(id) ?? null;
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
      const kind = tagOf(reply["kind"]);
      const rawReason = reply["message"];
      const reason =
        typeof rawReason === "string" && rawReason !== ""
          ? rawReason
          : `${tagOf(reply["outcome"])}/${kind}`;
      if (STALE_KINDS.has(kind)) return { ok: false, completion: failStale("launch") };
      return { ok: false, completion: failFault(kind, reason) };
    }
    const facts = reply["facts"];
    if (facts === null || typeof facts !== "object" || Array.isArray(facts)) {
      throw new TypeError(`descriptor.${operation} reply carries no facts map`);
    }
    return { ok: true, facts: facts as Record<string, unknown> };
  }

  function exitRecord(raw: unknown): unknown {
    if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
      throw new TypeError("descriptor reply carries a malformed exit");
    }
    const fields = raw as Record<string, unknown>;
    return record(errors.exit, [
      ["code", BigInt(fields["code"] as number)],
      ["signaled", fields["signaled"] as boolean],
      ["signal", readStr(fields["signal"], "exit.signal")],
    ]);
  }

  function factsRecord(raw: unknown): unknown {
    if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
      throw new TypeError("descriptor reply carries malformed facts");
    }
    const fields = raw as Record<string, unknown>;
    return record(errors.facts, [
      ["offered", fields["offered"] as boolean],
      ["offered_bytes", BigInt(fields["offered_bytes"] as number)],
      ["writer_done", fields["writer_done"] as boolean],
      ["writer_err", readStr(fields["writer_err"], "facts.writer_err")],
      ["accepted", fields["accepted"] as boolean],
      ["malformed", fields["malformed"] as boolean],
      ["eof", fields["eof"] as boolean],
      ["reaped", fields["reaped"] as boolean],
      [
        "child_exit",
        optionalOf(
          errors.some,
          errors.none,
          fields["child_exit"] === null || fields["child_exit"] === undefined
            ? null
            : exitRecord(fields["child_exit"]),
        ),
      ],
    ]);
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

  function reportRecord(raw: unknown): unknown {
    if (raw === null || typeof raw !== "object" || Array.isArray(raw)) {
      throw new TypeError("descriptor reply carries a malformed report");
    }
    const fields = raw as Record<string, unknown>;
    return record(errors.report, [
      ["executable", readStr(fields["executable"], "report.executable")],
      ["facts", factsRecord(fields["facts"])],
      ["lease", leaseRecord(fields["lease"])],
      ["orphan", fields["orphan"] as boolean],
      ["clean", fields["clean"] as boolean],
      ["reason", readStr(fields["reason"], "report.reason")],
    ]);
  }

  function readSpec(value: unknown): Record<string, unknown> {
    return {
      executable: readStr(dataProperty(value, "executable"), "descriptor::launch_child executable"),
      args: readStrArray(dataProperty(value, "args"), "descriptor::launch_child args"),
      env: dataArray(dataProperty(value, "env"), "descriptor::launch_child env").map((entry) => ({
        key: readStr(dataProperty(entry, "key"), "descriptor::env_entry key"),
        value: readStr(dataProperty(entry, "value"), "descriptor::env_entry value"),
      })),
      with_lease: readBool(
        dataProperty(value, "with_lease"),
        "descriptor::launch_child with_lease",
      ),
      detached: readBool(dataProperty(value, "detached"), "descriptor::launch_child detached"),
      dir: readStr(dataProperty(value, "dir"), "descriptor::launch_child dir"),
    };
  }

  return Object.freeze({
    async launchChild(
      ownerHandle: unknown,
      opId: unknown,
      spec: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "descriptor::launch_child");
      if (typeof admitted !== "string") return admitted;
      const grant = admitted;
      const args = {
        op_id: readStr(opId, "descriptor::launch_child op_id"),
        spec: readSpec(spec),
      };
      const routed = route(
        "launch_child",
        await dispatch(
          envelope("descriptor.launch_child", args, grant, TRANSPORT_DEFAULT_DEADLINE_MS),
        ),
      );
      if (!routed.ok) return routed.completion;
      return success(mintLaunch(routed.facts["launch"], grant));
    },

    async collectStatus(
      ownerHandle: unknown,
      launch: unknown,
      ackTimeoutMs: unknown,
      eofTimeoutMs: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "descriptor::collect_status");
      if (typeof admitted !== "string") return admitted;
      const row = launchRow(launch, "descriptor::collect_status");
      if (row === null) return failStale("launch");
      const ackMs = readInt(ackTimeoutMs, "descriptor::collect_status ack_timeout_ms");
      const eofMs = readInt(eofTimeoutMs, "descriptor::collect_status eof_timeout_ms");
      const routed = route(
        "collect_status",
        await dispatch(
          envelope(
            "descriptor.collect_status",
            { launch: row.wire, ack_timeout_ms: ackMs, eof_timeout_ms: eofMs },
            row.grant,
            Math.max(ackMs + eofMs, TRANSPORT_DEFAULT_DEADLINE_MS),
          ),
        ),
      );
      if (!routed.ok) return routed.completion;
      return success(undefined);
    },

    async waitChild(
      ownerHandle: unknown,
      launch: unknown,
      timeoutMs: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "descriptor::wait_child");
      if (typeof admitted !== "string") return admitted;
      const row = launchRow(launch, "descriptor::wait_child");
      if (row === null) return failStale("launch");
      const timeout = readInt(timeoutMs, "descriptor::wait_child timeout_ms");
      const routed = route(
        "wait_child",
        await dispatch(
          envelope(
            "descriptor.wait_child",
            { launch: row.wire, timeout_ms: timeout },
            row.grant,
            Math.max(timeout, TRANSPORT_DEFAULT_DEADLINE_MS),
          ),
        ),
      );
      if (!routed.ok) return routed.completion;
      return success(exitRecord(routed.facts["exit"]));
    },

    async killChild(
      ownerHandle: unknown,
      launch: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<void>> {
      const admitted = admitOwner(ownerHandle, "descriptor::kill_child");
      if (typeof admitted !== "string") return admitted;
      const row = launchRow(launch, "descriptor::kill_child");
      if (row === null) return failStale("launch");
      const routed = route(
        "kill_child",
        await dispatch(
          envelope(
            "descriptor.kill_child",
            { launch: row.wire },
            row.grant,
            TRANSPORT_DEFAULT_DEADLINE_MS,
          ),
        ),
      );
      if (!routed.ok) return routed.completion;
      return success(undefined);
    },

    async releaseLaunch(
      ownerHandle: unknown,
      launch: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "descriptor::release_launch");
      if (typeof admitted !== "string") return admitted;
      const row = launchRow(launch, "descriptor::release_launch");
      if (row === null) return failStale("launch");
      const routed = route(
        "release_launch",
        await dispatch(
          envelope(
            "descriptor.release_launch",
            { launch: row.wire },
            row.grant,
            TRANSPORT_DEFAULT_DEADLINE_MS,
          ),
        ),
      );
      if (!routed.ok) return routed.completion;
      return success(reportRecord(routed.facts["report"]));
    },

    async readFacts(
      ownerHandle: unknown,
      launch: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "descriptor::read_facts");
      if (typeof admitted !== "string") return admitted;
      const row = launchRow(launch, "descriptor::read_facts");
      if (row === null) return failStale("launch");
      const routed = route(
        "read_facts",
        await dispatch(
          envelope(
            "descriptor.read_facts",
            { launch: row.wire },
            row.grant,
            TRANSPORT_DEFAULT_DEADLINE_MS,
          ),
        ),
      );
      if (!routed.ok) return routed.completion;
      return success(factsRecord(routed.facts["facts"]));
    },

    async expectedAck(
      ownerHandle: unknown,
      launch: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "descriptor::expected_ack");
      if (typeof admitted !== "string") return admitted;
      const row = launchRow(launch, "descriptor::expected_ack");
      if (row === null) return failStale("launch");
      const routed = route(
        "expected_ack",
        await dispatch(
          envelope(
            "descriptor.expected_ack",
            { launch: row.wire },
            row.grant,
            TRANSPORT_DEFAULT_DEADLINE_MS,
          ),
        ),
      );
      if (!routed.ok) return routed.completion;
      const raw = routed.facts["ack"];
      if (!Array.isArray(raw))
        throw new TypeError("descriptor expected_ack reply carries no ack array");
      return success(array(raw.map((byte) => BigInt(byte as number))));
    },
  });
}
