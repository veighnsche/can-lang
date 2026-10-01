// NT-I01 native-values adapter. Builds NativeValueRequest envelopes for the
// external N owner (schema.ts checkRequest contract) and maps
// NativeValueResult facts back to nominal Can records and opaque handles.
// Wire rule (merge-defined, the N owner honors it): replies carry wire
// handles under their handlesOut keys (open.session, make.value,
// invoke.value/action, settle.value, gate.gate, fault.fault); the adapter
// round-trips wire objects verbatim and never invents ids. The transport is
// injected: the default dispatch throws, so no call can silently succeed
// before the N owner lands (P11/P14/P23 scope). Non-completed outcomes throw
// with outcome/kind; a Can failure vocabulary lands with the N owner.
import { createHash, randomBytes } from "node:crypto";
import { success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, dataArray, dataProperty, record, recordIdentity } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import type { TestOwner } from "../../owner.ts";

const sessionBrand = Symbol("can.native.session");
const valueBrand = Symbol("can.native.value");
const actionBrand = Symbol("can.native.action");
const gateBrand = Symbol("can.native.gate");
const faultBrand = Symbol("can.native.fault");

const MAX_HANDLES = 1024;
const TRANSPORT_DEFAULT_DEADLINE_MS = 30000;

export type NativeValuesErrors = Readonly<{
  some: string;
  none: string;
  inertFacts: string;
  observation: string;
  observationResult: string;
  taggedEntry: string;
  taggedValue: string;
  aliasGroup: string;
  observeCounters: string;
  descriptorOrUnknown: string;
  handleOrPendingAction: string;
  settlementOrPending: string;
  releaseFacts: string;
  restoreOutcome: string;
  closeReceipt: string;
}>;

export type NativeDispatch = (
  request: Readonly<Record<string, unknown>>,
) => Promise<Readonly<Record<string, unknown>>>;

function defaultDispatch(): Promise<Readonly<Record<string, unknown>>> {
  return Promise.reject(new Error("native-values N transport is not bound"));
}

// canonicalize copies the observe.ts digestFacts algorithm (sorted keys,
// compact JSON) so arguments_digest matches the service computation.
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
  throw new TypeError("native arguments_digest needs inert args");
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

function readOptional(
  value: unknown,
  someIdentity: string,
  noneIdentity: string,
  what: string,
): unknown {
  const identity = recordIdentity(value);
  if (identity === noneIdentity) return undefined;
  if (identity !== someIdentity) throw new TypeError(`${what} needs an option`);
  return dataProperty(value, "value");
}

function optionalOf(someIdentity: string, noneIdentity: string, value: unknown): unknown {
  if (value === null || value === undefined) return record(noneIdentity, []);
  return record(someIdentity, [["value", value]]);
}

function isBranded(value: unknown, brand: symbol): value is Record<symbol, number> {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as Record<symbol, unknown>)[brand] === "number"
  );
}

export function createNativeValues(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: NativeValuesErrors,
  owner: TestOwner,
  dispatch: NativeDispatch = defaultDispatch,
) {
  void domain;
  const runId = `run-${hex(8)}`;
  let operationSeq = 0;
  const sessions = new Map<number, { wire: unknown; grant: string; closed: boolean }>();
  const values = new Map<number, unknown>();
  // Action, gate and fault rows carry their session grant: settle, release
  // and restore take no session argument, so the grant cannot come from a
  // paired session the way value operations do.
  const actions = new Map<number, { wire: unknown; grant: string }>();
  const gates = new Map<number, { wire: unknown; grant: string }>();
  const faults = new Map<number, { wire: unknown; grant: string }>();
  let nextId = 0;

  function mint(table: Map<number, unknown>, brand: symbol, kind: string, wire: unknown) {
    if (table.size >= MAX_HANDLES) throw new TypeError(`native ${kind} table cap reached`);
    if (wire === null || typeof wire !== "object" || Array.isArray(wire)) {
      throw new TypeError(`native ${kind} reply carries no wire handle`);
    }
    nextId += 1;
    table.set(nextId, wire);
    return Object.freeze({ [brand]: nextId });
  }

  function lookup(
    table: Map<number, unknown>,
    brand: symbol,
    kind: string,
    value: unknown,
    what: string,
  ): unknown {
    if (!isBranded(value, brand)) throw new TypeError(`${what} needs a native::${kind} handle`);
    const wire = table.get(value[brand]);
    if (wire === undefined) throw new TypeError(`${what} names a foreign native::${kind} handle`);
    return wire;
  }

  function mintGrant(
    table: Map<number, { wire: unknown; grant: string }>,
    brand: symbol,
    kind: string,
    wire: unknown,
    grant: string,
  ) {
    if (table.size >= MAX_HANDLES) throw new TypeError(`native ${kind} table cap reached`);
    if (wire === null || typeof wire !== "object" || Array.isArray(wire)) {
      throw new TypeError(`native ${kind} reply carries no wire handle`);
    }
    nextId += 1;
    table.set(nextId, { wire, grant });
    return Object.freeze({ [brand]: nextId });
  }

  function lookupGrant(
    table: Map<number, { wire: unknown; grant: string }>,
    brand: symbol,
    kind: string,
    value: unknown,
    what: string,
  ): { wire: unknown; grant: string } {
    if (!isBranded(value, brand)) throw new TypeError(`${what} needs a native::${kind} handle`);
    const row = table.get(value[brand]);
    if (row === undefined) throw new TypeError(`${what} names a foreign native::${kind} handle`);
    return row;
  }

  function lookupSession(value: unknown, what: string): { wire: unknown; grant: string } {
    if (!isBranded(value, sessionBrand)) {
      throw new TypeError(`${what} needs a native::session handle`);
    }
    const row = sessions.get(value[sessionBrand]);
    if (row === undefined) throw new TypeError(`${what} names a foreign native::session handle`);
    if (row.closed) throw new TypeError(`${what} names a closed native::session handle`);
    return row;
  }

  function envelope(
    operation: string,
    args: Readonly<Record<string, unknown>>,
    grant: string,
    deadline?: Readonly<{ clock: string; ms: number }>,
  ): Readonly<Record<string, unknown>> {
    operationSeq += 1;
    return Object.freeze({
      schema_version: "1",
      run_id: runId,
      operation_id: `op-${operationSeq}-${hex(4)}`,
      owner_grant: grant,
      operation,
      arguments_digest: digestArgs(args),
      deadline_ms: Object.freeze(
        deadline ?? { clock: "n-monotonic", ms: TRANSPORT_DEFAULT_DEADLINE_MS },
      ),
      args: Object.freeze({ ...args }),
    });
  }

  function tagOf(value: unknown): string {
    return typeof value === "string" ? value : "?";
  }

  function factsOf(
    reply: Readonly<Record<string, unknown>>,
    operation: string,
  ): Record<string, unknown> {
    if (reply["outcome"] !== "completed") {
      throw new Error(`native.${operation} ${tagOf(reply["outcome"])}/${tagOf(reply["kind"])}`);
    }
    const facts = reply["facts"];
    if (facts === null || typeof facts !== "object" || Array.isArray(facts)) {
      throw new TypeError(`native.${operation} reply carries no facts map`);
    }
    return facts as Record<string, unknown>;
  }

  function readDeadline(value: unknown, what: string): { clock: string; ms: number } {
    return {
      clock: readStr(dataProperty(value, "clock"), `${what}.clock`),
      ms: readInt(dataProperty(value, "ms"), `${what}.ms`),
    };
  }

  function readLiteralValue(value: unknown, what: string): string | number | boolean {
    const identity = recordIdentity(value);
    if (identity === undefined) throw new TypeError(`${what} needs a native::literal_value`);
    const leaf = identity.split("::").pop();
    if (leaf === "literal_int") return readInt(dataProperty(value, "value"), `${what}.value`);
    if (leaf === "literal_text") return readStr(dataProperty(value, "value"), `${what}.value`);
    if (leaf === "literal_bool") return readBool(dataProperty(value, "value"), `${what}.value`);
    throw new TypeError(`${what} needs a native::literal_value leaf`);
  }

  function taggedValueRecord(tagged: unknown): unknown {
    if (tagged === null || typeof tagged !== "object" || Array.isArray(tagged)) {
      throw new TypeError("native observe reply carries a malformed tagged value");
    }
    const get = (key: string): unknown => (tagged as Record<string, unknown>)[key] ?? null;
    const optInt = (key: string): unknown => {
      const raw = get(key);
      return raw === null ? null : BigInt(raw as number);
    };
    return record(errors.taggedValue, [
      ["tag", readStr(get("tag"), "tagged_value.tag")],
      [
        "value",
        optionalOf(
          errors.some,
          errors.none,
          get("value") === null ? null : readBool(get("value"), "tagged_value.value"),
        ),
      ],
      ["text", optionalOf(errors.some, errors.none, get("text"))],
      ["lexeme", optionalOf(errors.some, errors.none, get("lexeme"))],
      ["hi", optionalOf(errors.some, errors.none, optInt("hi"))],
      ["lo", optionalOf(errors.some, errors.none, optInt("lo"))],
      ["class", optionalOf(errors.some, errors.none, get("class"))],
      ["sign", optionalOf(errors.some, errors.none, optInt("sign"))],
      ["exponent", optionalOf(errors.some, errors.none, optInt("exponent"))],
      ["mantissa_hi", optionalOf(errors.some, errors.none, optInt("mantissa_hi"))],
      ["mantissa_lo", optionalOf(errors.some, errors.none, optInt("mantissa_lo"))],
      ["gap", optionalOf(errors.some, errors.none, get("gap"))],
    ]);
  }

  function observationRecord(entry: unknown): unknown {
    if (entry === null || typeof entry !== "object" || Array.isArray(entry)) {
      throw new TypeError("native observe reply carries a malformed observation");
    }
    const fields = entry as Record<string, unknown>;
    const result = fields["result"];
    if (result === null || typeof result !== "object" || Array.isArray(result)) {
      throw new TypeError("native observe reply carries a malformed observation result");
    }
    const get = (key: string): unknown => (result as Record<string, unknown>)[key] ?? null;
    const optInt = (key: string): unknown => {
      const raw = get(key);
      return raw === null ? null : BigInt(raw as number);
    };
    const entriesRaw = get("entries");
    const entries =
      entriesRaw === null
        ? null
        : array(
            (entriesRaw as unknown[]).map((item) => {
              if (item === null || typeof item !== "object" || Array.isArray(item)) {
                throw new TypeError("native observe reply carries a malformed entry");
              }
              const pair = item as Record<string, unknown>;
              return record(errors.taggedEntry, [
                ["key", readStr(pair["key"], "tagged_entry.key")],
                ["value", taggedValueRecord(pair["value"])],
              ]);
            }),
          );
    return record(errors.observation, [
      ["handle", readStr(fields["handle"], "observation.handle")],
      ["cell", readStr(fields["cell"], "observation.cell")],
      [
        "result",
        record(errors.observationResult, [
          ["scalar", optionalOf(errors.some, errors.none, get("scalar"))],
          ["descriptor", optionalOf(errors.some, errors.none, get("descriptor"))],
          ["bytes", optionalOf(errors.some, errors.none, optInt("bytes"))],
          ["cell", optionalOf(errors.some, errors.none, get("cell"))],
          ["gap", optionalOf(errors.some, errors.none, get("gap"))],
          ["tag", optionalOf(errors.some, errors.none, get("tag"))],
          ["hi", optionalOf(errors.some, errors.none, optInt("hi"))],
          ["lo", optionalOf(errors.some, errors.none, optInt("lo"))],
          ["class", optionalOf(errors.some, errors.none, get("class"))],
          ["sign", optionalOf(errors.some, errors.none, optInt("sign"))],
          ["exponent", optionalOf(errors.some, errors.none, optInt("exponent"))],
          ["mantissa_hi", optionalOf(errors.some, errors.none, optInt("mantissa_hi"))],
          ["mantissa_lo", optionalOf(errors.some, errors.none, optInt("mantissa_lo"))],
          ["lexeme", optionalOf(errors.some, errors.none, get("lexeme"))],
          ["text", optionalOf(errors.some, errors.none, get("text"))],
          ["omitted", optionalOf(errors.some, errors.none, optInt("omitted"))],
          ["entries", optionalOf(errors.some, errors.none, entries)],
        ]),
      ],
    ]);
  }

  return Object.freeze({
    async open(
      ownerHandle: unknown,
      runtime: unknown,
      observer: unknown,
      scope: unknown,
      limits: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      if (!owner.ownerLive(ownerHandle)) {
        throw new TypeError("native::open needs a live test::owner handle");
      }
      const grant = owner.grantOf(ownerHandle);
      if (grant === null) throw new TypeError("native::open needs a live test::owner handle");
      const args = {
        runtime: readStr(runtime, "native::open runtime"),
        observer: readStr(observer, "native::open observer"),
        scope: readStr(scope, "native::open scope"),
        limits: {
          max_sessions: readInt(
            dataProperty(limits, "max_sessions"),
            "native::limits max_sessions",
          ),
          max_handles_per_session: readInt(
            dataProperty(limits, "max_handles_per_session"),
            "native::limits max_handles_per_session",
          ),
          maxPendingActions: readInt(
            dataProperty(limits, "maxPendingActions"),
            "native::limits maxPendingActions",
          ),
          maxObserveEntries: readInt(
            dataProperty(limits, "maxObserveEntries"),
            "native::limits maxObserveEntries",
          ),
          maxObserveBytes: readInt(
            dataProperty(limits, "maxObserveBytes"),
            "native::limits maxObserveBytes",
          ),
          maxGatesPerSession: readInt(
            dataProperty(limits, "maxGatesPerSession"),
            "native::limits maxGatesPerSession",
          ),
          maxFaultsPerSession: readInt(
            dataProperty(limits, "maxFaultsPerSession"),
            "native::limits maxFaultsPerSession",
          ),
        },
      };
      const facts = factsOf(await dispatch(envelope("native.open", args, grant)), "open");
      if (sessions.size >= MAX_HANDLES) throw new TypeError("native session table cap reached");
      const wire = facts["session"];
      if (wire === null || typeof wire !== "object" || Array.isArray(wire)) {
        throw new TypeError("native open reply carries no wire session");
      }
      nextId += 1;
      sessions.set(nextId, { wire, grant, closed: false });
      return success(Object.freeze({ [sessionBrand]: nextId }));
    },

    async describe(
      session: unknown,
      api: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const row = lookupSession(session, "native::describe");
      const args = { session: row.wire, api: readStr(api, "native::describe api") };
      const facts = factsOf(
        await dispatch(envelope("native.describe", args, row.grant)),
        "describe",
      );
      return success(
        record(errors.descriptorOrUnknown, [
          ["api", readStr(facts["api"] ?? args["api"], "descriptor_or_unknown.api")],
          ["known", readBool(facts["known"], "descriptor_or_unknown.known")],
          ["presence", optionalOf(errors.some, errors.none, facts["presence"] ?? null)],
          ["descriptor", optionalOf(errors.some, errors.none, facts["descriptor"] ?? null)],
        ]),
      );
    },

    async make(
      session: unknown,
      kind: unknown,
      payload: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const row = lookupSession(session, "native::make");
      const tag = readStr(dataProperty(payload, "tag"), "native::inert_literal tag");
      const wirePayload: Record<string, unknown> = { tag };
      const hi = readOptional(
        dataProperty(payload, "hi"),
        errors.some,
        errors.none,
        "inert_literal.hi",
      );
      const lo = readOptional(
        dataProperty(payload, "lo"),
        errors.some,
        errors.none,
        "inert_literal.lo",
      );
      const text = readOptional(
        dataProperty(payload, "text"),
        errors.some,
        errors.none,
        "inert_literal.text",
      );
      const entries = readOptional(
        dataProperty(payload, "entries"),
        errors.some,
        errors.none,
        "inert_literal.entries",
      );
      const descriptor = readOptional(
        dataProperty(payload, "descriptor"),
        errors.some,
        errors.none,
        "inert_literal.descriptor",
      );
      if (hi !== undefined) wirePayload["hi"] = readInt(hi, "inert_literal.hi");
      if (lo !== undefined) wirePayload["lo"] = readInt(lo, "inert_literal.lo");
      if (text !== undefined) wirePayload["text"] = readStr(text, "inert_literal.text");
      if (entries !== undefined) {
        wirePayload["entries"] = dataArray(entries).map((entry) => [
          readStr(dataProperty(entry, "key"), "native::entry key"),
          readLiteralValue(dataProperty(entry, "value"), "native::entry value"),
        ]);
      }
      if (descriptor !== undefined)
        wirePayload["descriptor"] = readStr(descriptor, "inert_literal.descriptor");
      const args = {
        session: row.wire,
        kind: readStr(kind, "native::make kind"),
        payload: wirePayload,
      };
      const facts = factsOf(await dispatch(envelope("native.make", args, row.grant)), "make");
      return success(mint(values, valueBrand, "value", facts["value"]));
    },

    async invoke(
      session: unknown,
      operation: unknown,
      receiver: unknown,
      args: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const row = lookupSession(session, "native::invoke");
      const wireArgs: Record<string, unknown> = {
        session: row.wire,
        operation: readStr(operation, "native::invoke operation"),
        arguments: dataArray(args).map((entry) =>
          lookup(values, valueBrand, "value", entry, "native::invoke arguments"),
        ),
      };
      const receiverWire = readOptional(
        receiver,
        errors.some,
        errors.none,
        "native::invoke receiver",
      );
      if (receiverWire !== undefined) {
        wireArgs["receiver"] = lookup(
          values,
          valueBrand,
          "value",
          receiverWire,
          "native::invoke receiver",
        );
      }
      const facts = factsOf(
        await dispatch(envelope("native.invoke", wireArgs, row.grant)),
        "invoke",
      );
      const settled = facts["value"] !== undefined && facts["value"] !== null;
      return success(
        record(errors.handleOrPendingAction, [
          ["settled", settled],
          [
            "handle",
            optionalOf(
              errors.some,
              errors.none,
              settled ? mint(values, valueBrand, "value", facts["value"]) : null,
            ),
          ],
          [
            "action",
            optionalOf(
              errors.some,
              errors.none,
              settled
                ? null
                : mintGrant(actions, actionBrand, "pending_action", facts["action"], row.grant),
            ),
          ],
        ]),
      );
    },

    async settle(
      action: unknown,
      deadline: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const row = lookupGrant(actions, actionBrand, "pending_action", action, "native::settle");
      const limit = readDeadline(deadline, "native::deadline");
      const args = { action: row.wire, deadline: { clock: limit.clock, ms: limit.ms } };
      const facts = factsOf(
        await dispatch(envelope("native.settle", args, row.grant, limit)),
        "settle",
      );
      const settled = facts["value"] !== undefined && facts["value"] !== null;
      return success(
        record(errors.settlementOrPending, [
          ["settled", settled],
          [
            "handle",
            optionalOf(
              errors.some,
              errors.none,
              settled ? mint(values, valueBrand, "value", facts["value"]) : null,
            ),
          ],
        ]),
      );
    },

    async observe(
      session: unknown,
      handles: unknown,
      kind: unknown,
      bounds: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const row = lookupSession(session, "native::observe");
      const wireBounds = {
        maxEntries: readInt(
          dataProperty(bounds, "maxEntries"),
          "native::observe_bounds maxEntries",
        ),
        maxBytes: readInt(dataProperty(bounds, "maxBytes"), "native::observe_bounds maxBytes"),
      };
      const args = {
        session: row.wire,
        handles: dataArray(handles).map((entry) =>
          lookup(values, valueBrand, "value", entry, "native::observe handles"),
        ),
        kind: readStr(kind, "native::observe kind"),
        bounds: wireBounds,
      };
      const facts = factsOf(await dispatch(envelope("native.observe", args, row.grant)), "observe");
      const groups = facts["alias_groups"];
      const counters = facts["counters"];
      return success(
        record(errors.inertFacts, [
          ["kind", readStr(facts["kind"], "inert_facts.kind")],
          [
            "observations",
            array(((facts["observations"] ?? []) as unknown[]).map(observationRecord)),
          ],
          [
            "alias_groups",
            optionalOf(
              errors.some,
              errors.none,
              groups === undefined || groups === null
                ? null
                : array(
                    (groups as unknown[]).map((group) => {
                      if (group === null || typeof group !== "object" || Array.isArray(group)) {
                        throw new TypeError("native observe reply carries a malformed alias group");
                      }
                      const cells = group as Record<string, unknown>;
                      return record(errors.aliasGroup, [
                        ["cell", readStr(cells["cell"], "alias_group.cell")],
                        [
                          "handles",
                          array(
                            ((cells["handles"] ?? []) as unknown[]).map((entry) =>
                              readStr(entry, "alias_group.handles"),
                            ),
                          ),
                        ],
                      ]);
                    }),
                  ),
            ),
          ],
          [
            "counters",
            optionalOf(
              errors.some,
              errors.none,
              counters === undefined || counters === null
                ? null
                : record(errors.observeCounters, [
                    ["makes", BigInt((counters as Record<string, number>)["makes"] ?? 0)],
                    ["aliases", BigInt((counters as Record<string, number>)["aliases"] ?? 0)],
                    [
                      "observations",
                      BigInt((counters as Record<string, number>)["observations"] ?? 0),
                    ],
                    [
                      "explicitReads",
                      BigInt((counters as Record<string, number>)["explicitReads"] ?? 0),
                    ],
                    ["sealed", Boolean((counters as Record<string, unknown>)["sealed"])],
                    [
                      "getterReads",
                      BigInt((counters as Record<string, number>)["getterReads"] ?? 0),
                    ],
                    ["thenCalls", BigInt((counters as Record<string, number>)["thenCalls"] ?? 0)],
                    ["proxyTraps", BigInt((counters as Record<string, number>)["proxyTraps"] ?? 0)],
                  ]),
            ),
          ],
        ]),
      );
    },

    async allocateGate(
      session: unknown,
      spec: unknown,
      ref: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const row = lookupSession(session, "native::allocate_gate");
      const actionRow = lookupGrant(
        actions,
        actionBrand,
        "pending_action",
        dataProperty(ref, "action"),
        "native::bounded_action_ref action",
      );
      const args = {
        session: row.wire,
        gate: {
          name: readStr(dataProperty(spec, "name"), "native::gate_spec name"),
          max_waiters: readInt(dataProperty(spec, "max_waiters"), "native::gate_spec max_waiters"),
        },
        action: {
          action: actionRow.wire,
          bound: readInt(dataProperty(ref, "bound"), "native::bounded_action_ref bound"),
        },
      };
      const facts = factsOf(await dispatch(envelope("native.gate", args, row.grant)), "gate");
      return success(mintGrant(gates, gateBrand, "gate", facts["gate"], row.grant));
    },

    async release(gate: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const row = lookupGrant(gates, gateBrand, "gate", gate, "native::release");
      const args = { gate: row.wire };
      const facts = factsOf(await dispatch(envelope("native.release", args, row.grant)), "release");
      return success(
        record(errors.releaseFacts, [
          ["released", readBool(facts["released"], "release_facts.released")],
          ["waiters", BigInt((facts["waiters"] as number) ?? 0)],
          ["joined", readBool(facts["joined"], "release_facts.joined")],
        ]),
      );
    },

    async installFault(
      session: unknown,
      target: unknown,
      mode: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const row = lookupSession(session, "native::install_fault");
      const args = {
        session: row.wire,
        target: readStr(target, "native::install_fault target"),
        mode: readStr(mode, "native::install_fault mode"),
      };
      const facts = factsOf(await dispatch(envelope("native.fault", args, row.grant)), "fault");
      return success(mintGrant(faults, faultBrand, "fault", facts["fault"], row.grant));
    },

    async restore(fault: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      const row = lookupGrant(faults, faultBrand, "fault", fault, "native::restore");
      const args = { fault: row.wire };
      const facts = factsOf(await dispatch(envelope("native.restore", args, row.grant)), "restore");
      return success(
        record(errors.restoreOutcome, [
          ["restored", readBool(facts["restored"], "restore_outcome.restored")],
          ["timing", readStr(facts["timing"], "restore_outcome.timing")],
          ["outcome", readStr(facts["outcome"], "restore_outcome.outcome")],
        ]),
      );
    },

    async close(
      session: unknown,
      deadline: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      if (!isBranded(session, sessionBrand)) {
        throw new TypeError("native::close needs a native::session handle");
      }
      const row = sessions.get((session as Record<symbol, number>)[sessionBrand]);
      if (row === undefined)
        throw new TypeError("native::close names a foreign native::session handle");
      const limit = readDeadline(deadline, "native::deadline");
      const args = { session: row.wire, deadline: { clock: limit.clock, ms: limit.ms } };
      const facts = factsOf(
        await dispatch(envelope("native.close", args, row.grant, limit)),
        "close",
      );
      row.closed = true;
      return success(
        record(errors.closeReceipt, [
          ["sessionId", readStr(facts["sessionId"], "close_receipt.sessionId")],
          ["released", BigInt((facts["released"] as number) ?? 0)],
          ["remaining", BigInt((facts["remaining"] as number) ?? 0)],
          [
            "forced",
            array(
              ((facts["forced"] ?? []) as unknown[]).map((entry) =>
                readStr(entry, "close_receipt.forced"),
              ),
            ),
          ],
          ["joined", readBool(facts["joined"], "close_receipt.joined")],
          ["cellsReleased", BigInt((facts["cellsReleased"] as number) ?? 0)],
        ]),
      );
    },
  });
}
