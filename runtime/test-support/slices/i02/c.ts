// NT-I02 C-ingress adapter. Owner-bound parse_module/check_module over the
// ported K04 seam checker (c-service.ts). Pure local compute: no transport
// and no injected dispatch, because the seam functions are candidate-visible
// by the I02 split decision while the K05 witness stays host-side with no
// catalogue operations. A revoked or foreign owner fails test::stale_handle
// without touching the checker. Wire records map verbatim: snake_case Can
// fields to the service camelCase, manifest options to absent-or-present
// service fields, and problem strings to a str array.
import { failure, success, type AssertionContext, type Completion } from "../../../completion.ts";
import { array, dataArray, dataProperty, record, recordIdentity } from "../../../data.ts";
import { createDomainRuntime } from "../../../domain.ts";
import { ownerTag, type TestOwner } from "../../owner.ts";
import {
  checkModule,
  parseModule,
  type EntryKind,
  type IngressManifest,
  type ParsedEdge,
  type ParsedModule,
  type ParsedName,
} from "./c-service.ts";

const origin = Object.freeze({
  source: "can:c",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

export type CIngressErrors = Readonly<{
  staleHandle: string;
  some: string;
  none: string;
  parsedModule: string;
  parsedEdge: string;
  parsedName: string;
}>;

const ENTRY_KINDS: ReadonlyArray<EntryKind> = ["executable", "assertion_root", "assertion_case"];

function readStr(value: unknown, what: string): string {
  if (typeof value !== "string") throw new TypeError(`${what} needs a str`);
  return value;
}

function readBool(value: unknown, what: string): boolean {
  if (typeof value !== "boolean") throw new TypeError(`${what} needs a bool`);
  return value;
}

function readStrArray(value: unknown, what: string): string[] {
  return dataArray(value, what).map((entry) => readStr(entry, `${what} entry`));
}

function readOptionalStr(
  value: unknown,
  someIdentity: string,
  noneIdentity: string,
  what: string,
): string | undefined {
  const identity = recordIdentity(value);
  if (identity === noneIdentity) return undefined;
  if (identity !== someIdentity) throw new TypeError(`${what} needs an option`);
  return readStr(dataProperty(value, "value"), `${what} value`);
}

export function createCIngress(
  domain: ReturnType<typeof createDomainRuntime>,
  errors: CIngressErrors,
  owner: TestOwner,
) {
  function failStale(handleTag: string): Completion<never> {
    return failure(
      domain.create(
        errors.staleHandle,
        record(errors.staleHandle, [["handle", handleTag]]),
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

  function readFixedEdgeNames(value: unknown, what: string): Record<string, string> {
    const names: Record<string, string> = {};
    for (const entry of dataArray(value, `${what} names`)) {
      names[readStr(dataProperty(entry, "exported"), `${what} exported`)] = readStr(
        dataProperty(entry, "local"),
        `${what} local`,
      );
    }
    return names;
  }

  function readManifestEntry(
    value: unknown,
    what: string,
  ): IngressManifest["entries"]["executable"] {
    return {
      path: readStr(dataProperty(value, "path"), `${what} path`),
      body_pattern: readStr(dataProperty(value, "body_pattern"), `${what} body_pattern`),
      fixed_edges: dataArray(dataProperty(value, "fixed_edges"), `${what} fixed_edges`).map(
        (edge) => {
          const runtimeModule = readOptionalStr(
            dataProperty(edge, "runtime_module"),
            errors.some,
            errors.none,
            `${what} runtime_module`,
          );
          const resolved = readOptionalStr(
            dataProperty(edge, "resolved"),
            errors.some,
            errors.none,
            `${what} resolved`,
          );
          return {
            ...(runtimeModule === undefined ? {} : { runtime_module: runtimeModule }),
            ...(resolved === undefined ? {} : { resolved }),
            type_only: readBool(dataProperty(edge, "type_only"), `${what} type_only`),
            names: readFixedEdgeNames(dataProperty(edge, "names"), what),
          };
        },
      ),
    };
  }

  function readManifest(value: unknown): IngressManifest {
    const assertionCase = dataProperty(value, "assertion_case");
    return {
      local_prefix: readStr(dataProperty(value, "local_prefix"), "c::manifest local_prefix"),
      allowed_program_roots: readStrArray(
        dataProperty(value, "program_roots"),
        "c::manifest program_roots",
      ),
      runtime_modules: readStrArray(
        dataProperty(value, "runtime_modules"),
        "c::manifest runtime_modules",
      ),
      entries: {
        executable: readManifestEntry(dataProperty(value, "executable"), "c::manifest executable"),
        assertion_root: readManifestEntry(
          dataProperty(value, "assertion_root"),
          "c::manifest assertion_root",
        ),
        assertion_case: {
          required_runtime_edges: dataArray(
            dataProperty(assertionCase, "runtime_edges"),
            "c::case_requirements runtime_edges",
          ).map((edge) => ({
            runtime_module: readStr(
              dataProperty(edge, "runtime_module"),
              "c::runtime_edge runtime_module",
            ),
            type_only: readBool(dataProperty(edge, "type_only"), "c::runtime_edge type_only"),
          })),
          required_resolved_edges: dataArray(
            dataProperty(assertionCase, "resolved_edges"),
            "c::case_requirements resolved_edges",
          ).map((edge) => ({
            resolved: readStr(dataProperty(edge, "resolved"), "c::resolved_edge resolved"),
            type_only: readBool(dataProperty(edge, "type_only"), "c::resolved_edge type_only"),
          })),
          exports: readStrArray(
            dataProperty(assertionCase, "exports"),
            "c::case_requirements exports",
          ),
        },
      },
    };
  }

  function parsedNameRecord(name: ParsedName): unknown {
    return record(errors.parsedName, [
      ["exported", name.exported],
      ["local", name.local],
      ["aliased", name.aliased],
    ]);
  }

  function parsedEdgeRecord(edge: ParsedEdge): unknown {
    return record(errors.parsedEdge, [
      ["specifier", edge.specifier],
      ["type_only", edge.typeOnly],
      ["names", array(edge.names.map((name) => parsedNameRecord(name)))],
      ["side_effect", edge.sideEffect],
    ]);
  }

  function parsedModuleRecord(parsed: ParsedModule): unknown {
    return record(errors.parsedModule, [
      ["imports", array(parsed.imports.map((edge) => parsedEdgeRecord(edge)))],
      ["exports", array(parsed.exports)],
      ["dynamic_import", parsed.dynamicImport],
      ["body", parsed.body],
    ]);
  }

  return Object.freeze({
    async parseModule(
      ownerHandle: unknown,
      source: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "c::parse_module");
      if (typeof admitted !== "string") return admitted;
      return success(parsedModuleRecord(parseModule(readStr(source, "c::parse_module source"))));
    },

    async checkModule(
      ownerHandle: unknown,
      manifest: unknown,
      kind: unknown,
      modulePath: unknown,
      source: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<unknown>> {
      const admitted = admitOwner(ownerHandle, "c::check_module");
      if (typeof admitted !== "string") return admitted;
      const entryKind = readStr(kind, "c::check_module kind");
      if (!ENTRY_KINDS.includes(entryKind as EntryKind)) {
        throw new TypeError(`c entry kind ${JSON.stringify(entryKind)} is not admitted`);
      }
      return success(
        array(
          checkModule(
            readManifest(manifest),
            entryKind as EntryKind,
            readStr(modulePath, "c::check_module module_path"),
            readStr(source, "c::check_module source"),
          ),
        ),
      );
    },
  });
}
