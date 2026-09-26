// H08 eval protocol loader: frozen registration + labeled triage case
// sets. The registration pins the feature, categories, provider
// identity, measures, and pass threshold before any live result; the
// loader refuses to run when a case file no longer matches its
// registered hash (edits require re-registration, never silent drift).
import { createHash } from "node:crypto";
import { readFile } from "node:fs/promises";
import { join } from "node:path";
import type { Schema } from "../../../runtime/codec/json.ts";
import { record } from "../../../runtime/data.ts";
import type { ChoiceDescriptor } from "../../../runtime/ai/questions.ts";

export type TriageCategory = Readonly<{ key: string; description: string }>;
export type TriageCase = Readonly<{
  id: string;
  subject: string;
  body: string;
  label: string;
  note?: string;
}>;
export type TriageRegistration = Readonly<{
  protocolVersion: string;
  task: string;
  experiment: string;
  workload: string;
  feature: string;
  instructions: string;
  categories: readonly TriageCategory[];
  stateCaps: Readonly<{ subjectChars: number; bodyChars: number }>;
  pinned: Readonly<{
    provider: string;
    model: string;
    version: string;
    protocol: string;
    endpoint: string;
    timeoutMs: number;
    maxBodyBytes: number;
  }>;
  abstainConfidenceFloor: number;
  thresholds: Readonly<{
    appliesTo: string;
    pass: Readonly<{
      of: number;
      correctMin: number;
      abstainMax: number;
      invalidOutputMax: number;
      rejectedMax: number;
    }>;
  }>;
  caseSets: Readonly<{
    representative: Readonly<{ file: string; sha256: string; count: number }>;
    heldout: Readonly<{ file: string; sha256: string; count: number }>;
  }>;
  boundStatus: Readonly<{ status: string; detail: string; upperBound?: unknown }>;
  priceTable: unknown;
}>;

export type TriageProtocol = Readonly<{
  registration: TriageRegistration;
  representative: readonly TriageCase[];
  heldout: readonly TriageCase[];
  question: ChoiceDescriptor;
}>;

const nonEmpty = (value: unknown, what: string): string => {
  if (typeof value !== "string" || value.trim() === "")
    throw new TypeError(`invalid protocol ${what}`);
  return value;
};

const sha256 = (bytes: Uint8Array): string => createHash("sha256").update(bytes).digest("hex");

function checkCases(value: unknown, file: string, registered: { count: number }): TriageCase[] {
  if (typeof value !== "object" || value === null || Array.isArray(value))
    throw new TypeError(`invalid protocol ${file}`);
  const cases = (value as { cases?: unknown }).cases;
  if (!Array.isArray(cases) || cases.length !== registered.count)
    throw new TypeError(`invalid protocol ${file} case count`);
  const seen = new Set<string>();
  return cases.map((entry): TriageCase => {
    if (typeof entry !== "object" || entry === null || Array.isArray(entry))
      throw new TypeError(`invalid protocol ${file} case shape`);
    const shaped = entry as Record<string, unknown>;
    const id = nonEmpty(shaped.id, `${file} case id`);
    if (seen.has(id)) throw new TypeError(`invalid protocol ${file} duplicate case`);
    seen.add(id);
    const note = shaped.note;
    if (note !== undefined && typeof note !== "string")
      throw new TypeError(`invalid protocol ${file} note`);
    return {
      id,
      subject: nonEmpty(shaped.subject, `${file} subject`),
      body: nonEmpty(shaped.body, `${file} body`),
      label: nonEmpty(shaped.label, `${file} label`),
      ...(note === undefined ? {} : { note: note as string }),
    };
  });
}

export async function loadProtocol(dir: string): Promise<TriageProtocol> {
  const registrationRaw = await readFile(join(dir, "registration.json"), "utf8");
  const registration = JSON.parse(registrationRaw) as TriageRegistration;
  if (typeof registration !== "object" || registration === null)
    throw new TypeError("invalid protocol registration");
  nonEmpty(registration.instructions, "instructions");
  if (!Array.isArray(registration.categories) || registration.categories.length < 2)
    throw new TypeError("invalid protocol categories");
  const keys = new Set<string>();
  for (const category of registration.categories) {
    const key = nonEmpty(category?.key, "category key");
    nonEmpty(category?.description, "category description");
    if (keys.has(key)) throw new TypeError("invalid protocol duplicate category");
    keys.add(key);
  }
  const floor = registration.abstainConfidenceFloor;
  if (typeof floor !== "number" || !Number.isFinite(floor) || floor < 0 || floor > 1)
    throw new TypeError("invalid protocol abstain floor");
  const pinned = registration.pinned;
  for (const field of ["provider", "model", "version", "protocol", "endpoint"] as const)
    nonEmpty(pinned?.[field], `pinned ${field}`);
  if (!Number.isSafeInteger(pinned?.timeoutMs) || (pinned?.timeoutMs as number) < 1)
    throw new TypeError("invalid protocol timeout");
  if (!Number.isSafeInteger(pinned?.maxBodyBytes) || (pinned?.maxBodyBytes as number) < 1)
    throw new TypeError("invalid protocol body cap");

  const sets = ["representative", "heldout"] as const;
  const loaded = {} as Record<(typeof sets)[number], readonly TriageCase[]>;
  for (const set of sets) {
    const entry = registration.caseSets?.[set];
    nonEmpty(entry?.file, `${set} file`);
    nonEmpty(entry?.sha256, `${set} hash`);
    const raw = await readFile(join(dir, entry.file));
    if (sha256(raw) !== entry.sha256)
      throw new TypeError(`protocol ${set} cases changed without re-registration`);
    loaded[set] = checkCases(JSON.parse(new TextDecoder().decode(raw)), entry.file, entry);
    for (const triage of loaded[set]) {
      if (!keys.has(triage.label)) throw new TypeError(`protocol ${set} unknown label`);
      if (triage.subject.length > registration.stateCaps.subjectChars)
        throw new TypeError(`protocol ${set} subject over cap`);
      if (triage.body.length > registration.stateCaps.bodyChars)
        throw new TypeError(`protocol ${set} body over cap`);
    }
  }
  const question: ChoiceDescriptor = {
    kind: "choice",
    instructions: registration.instructions,
    options: registration.categories.map((category) => ({
      key: category.key,
      description: category.description,
    })),
  };
  return { registration, representative: loaded.representative, heldout: loaded.heldout, question };
}

// stateSchema is the fixed triage state shape shared by every case:
// short subject plus bounded body, both opaque strings.
export function stateSchema(): Schema {
  return {
    root: "triage-state",
    nodes: [
      {
        identity: "triage-state",
        kind: "record",
        name: "state",
        fields: [
          { name: "subject", type: "triage-str" },
          { name: "body", type: "triage-str" },
        ],
      },
      { identity: "triage-str", kind: "primitive", name: "str" },
    ],
  };
}

// qualifiedBound reads the registration's qualified complete-call
// upper bound U. Only status "qualified" with a valid positive bound
// counts; anything else leaves the profile unqualified and live
// dispatch mechanically impossible.
export function qualifiedBound(registration: TriageRegistration): number | undefined {
  if (registration.boundStatus?.status !== "qualified") return undefined;
  const value = registration.boundStatus.upperBound;
  if (!Number.isSafeInteger(value) || (value as number) < 1) return undefined;
  return value as number;
}

// pricePerToken reads the registration's pinned USD price. Only an
// explicit finite positive per-token price counts; anything else
// leaves USD cost unknown and live dispatch blocked.
export function pricePerToken(registration: TriageRegistration): number | undefined {
  const table = registration.priceTable as { usdPerToken?: unknown } | null | undefined;
  if (typeof table !== "object" || table === null) return undefined;
  const price = table.usdPerToken;
  if (typeof price !== "number" || !Number.isFinite(price) || price <= 0) return undefined;
  return price;
}

// caseState encodes one frozen case as SystemOne state. Over-cap
// input refuses here, before any reservation or dispatch.
export function caseState(
  triage: TriageCase,
  caps: { subjectChars: number; bodyChars: number },
): unknown {
  if (triage.subject.length > caps.subjectChars || triage.body.length > caps.bodyChars)
    throw new TypeError("triage case over registered state cap");
  return record("triage-state", [
    ["subject", triage.subject],
    ["body", triage.body],
  ]);
}
