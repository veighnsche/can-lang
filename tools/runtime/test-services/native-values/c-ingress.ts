// K04 generated-C ingress seam checker: fixed entry/export/signature/import
// shapes plus same-realm completion handling.
//
// A generated module is clean when every import specifier is relative, every
// resolved target is a manifest runtime module under ONE shared runtime root
// or an allowed program path, every imported local carries the $can prefix,
// no dynamic import() appears, entry bodies match the supervisor shape, and
// exports match the entry kind. Anything else is a seam violation: a second
// runtime root would swap the runtime under the supervisor, and a dynamic
// import would admit an arbitrary importer.
export interface IngressManifest {
  readonly local_prefix: string;
  readonly allowed_program_roots: ReadonlyArray<string>;
  readonly runtime_modules: ReadonlyArray<string>;
  readonly entries: {
    readonly executable: {
      readonly path: string;
      readonly body_pattern: string;
      readonly fixed_edges: ReadonlyArray<{
        readonly runtime_module?: string;
        readonly resolved?: string;
        readonly type_only?: boolean;
        readonly names: Record<string, string>;
      }>;
    };
    readonly assertion_root: {
      readonly path: string;
      readonly body_pattern: string;
      readonly fixed_edges: ReadonlyArray<{
        readonly runtime_module?: string;
        readonly resolved?: string;
        readonly type_only?: boolean;
        readonly names: Record<string, string>;
      }>;
    };
    readonly assertion_case: {
      readonly required_runtime_edges: ReadonlyArray<{
        readonly runtime_module: string;
        readonly type_only: boolean;
      }>;
      readonly required_resolved_edges: ReadonlyArray<{
        readonly resolved: string;
        readonly type_only: boolean;
      }>;
      readonly exports: ReadonlyArray<string>;
    };
  };
}

export interface ParsedName {
  readonly exported: string;
  readonly local: string;
  readonly aliased: boolean;
}

export interface ParsedEdge {
  readonly specifier: string;
  readonly typeOnly: boolean;
  readonly names: ReadonlyArray<ParsedName>;
  readonly sideEffect: boolean;
}

export interface ParsedModule {
  readonly imports: ReadonlyArray<ParsedEdge>;
  readonly exports: ReadonlyArray<string>;
  readonly dynamicImport: boolean;
  readonly body: string;
}

const IMPORT_PATTERN = /^import\s+(?:type\s+)?(?:\{([^}]*)\}\s+from\s+)?["']([^"']+)["'];?\s*$/;
const EXPORT_PATTERN =
  /^export\s+(?:const|let|var|function|class|async\s+function)\s+([A-Za-z_$][\w$]*)/;
const EXPORT_LIST_PATTERN = /^export\s*\{([^}]*)\}/;
const DYNAMIC_IMPORT_PATTERN = /import\s*\(/;

export function parseModule(source: string): ParsedModule {
  const imports: ParsedEdge[] = [];
  const exports: string[] = [];
  const body: string[] = [];
  let dynamicImport = false;
  for (const line of source.split("\n")) {
    const trimmed = line.trim();
    if (trimmed === "") {
      continue;
    }
    const imported = IMPORT_PATTERN.exec(trimmed);
    if (imported !== null) {
      const typeOnly = /^import\s+type\s+/.test(trimmed);
      const names: ParsedName[] = [];
      const block = imported[1];
      if (block !== undefined) {
        for (const part of block.split(",")) {
          const cells = part.split(/\s+as\s+/).map((cell) => cell.trim());
          if (cells.length === 2 && cells[0] !== "" && cells[1] !== "") {
            names.push({ exported: cells[0] as string, local: cells[1] as string, aliased: true });
          } else if (part.trim() !== "") {
            names.push({ exported: part.trim(), local: part.trim(), aliased: false });
          }
        }
      }
      imports.push({
        specifier: imported[2] as string,
        typeOnly,
        names,
        sideEffect: block === undefined,
      });
      continue;
    }
    const exported = EXPORT_PATTERN.exec(trimmed);
    if (exported !== null) {
      exports.push(exported[1] as string);
      continue;
    }
    const listed = EXPORT_LIST_PATTERN.exec(trimmed);
    if (listed !== null) {
      for (const part of (listed[1] as string).split(",")) {
        const cells = part.split(/\s+as\s+/).map((cell) => cell.trim());
        const local = cells[cells.length - 1] as string;
        if (local !== "") {
          exports.push(local);
        }
      }
      continue;
    }
    if (DYNAMIC_IMPORT_PATTERN.test(trimmed)) {
      dynamicImport = true;
    }
    body.push(trimmed);
  }
  return { imports, exports, dynamicImport, body: body.join("\n") };
}

function resolveSpecifier(modulePath: string, specifier: string): string | null {
  if (!specifier.startsWith(".")) {
    return null;
  }
  const base = modulePath.split("/").slice(0, -1);
  for (const cell of specifier.split("/")) {
    if (cell === "." || cell === "") {
      continue;
    }
    if (cell === "..") {
      if (base.length === 0) {
        return null;
      }
      base.pop();
      continue;
    }
    base.push(cell);
  }
  return base.join("/");
}

interface ClassifiedEdge extends ParsedEdge {
  readonly resolved: string;
  readonly runtimeModule: string | null;
}

function splitRuntime(resolved: string): { root: string; rest: string } {
  const slash = resolved.indexOf("/");
  if (slash < 0) {
    return { root: resolved, rest: "" };
  }
  return { root: resolved.slice(0, slash), rest: resolved.slice(slash + 1) };
}

function isProgramPath(manifest: IngressManifest, resolved: string): boolean {
  return manifest.allowed_program_roots.some((prefix) => resolved.startsWith(prefix));
}

export type EntryKind = "executable" | "assertion_root" | "assertion_case";

// runtimeRootsOf lists the distinct runtime roots a module imports. A clean
// closure keeps one root across the entry supervisor and every case module:
// completions then resolve in the same realm as the supervisor that awaits
// them, and no second runtime copy can be swapped underneath.
export function runtimeRootsOf(
  manifest: IngressManifest,
  modulePath: string,
  source: string,
): string[] {
  const roots = new Set<string>();
  for (const edge of parseModule(source).imports) {
    const resolved = resolveSpecifier(modulePath, edge.specifier);
    if (resolved === null) {
      continue;
    }
    const { root, rest } = splitRuntime(resolved);
    if (manifest.runtime_modules.includes(rest)) {
      roots.add(root);
    }
  }
  return [...roots].sort();
}

export function checkModule(
  manifest: IngressManifest,
  kind: EntryKind,
  modulePath: string,
  source: string,
): string[] {
  const problems: string[] = [];
  const parsed = parseModule(source);
  if (parsed.dynamicImport) {
    problems.push("dynamic import() is not a seam import");
  }
  const runtimeRoots = new Set<string>();
  const edges: ClassifiedEdge[] = [];
  for (const edge of parsed.imports) {
    if (!edge.specifier.startsWith(".")) {
      problems.push(`non-relative specifier ${edge.specifier}`);
      continue;
    }
    const resolved = resolveSpecifier(modulePath, edge.specifier);
    if (resolved === null) {
      problems.push(`specifier escapes the generation root: ${edge.specifier}`);
      continue;
    }
    const { root, rest } = splitRuntime(resolved);
    const runtimeModule = manifest.runtime_modules.includes(rest) ? rest : null;
    if (runtimeModule !== null) {
      runtimeRoots.add(root);
    } else if (!isProgramPath(manifest, resolved)) {
      problems.push(`target outside the seam: ${edge.specifier} resolves to ${resolved}`);
      continue;
    }
    if (edge.sideEffect) {
      problems.push(`side-effect import without bindings: ${edge.specifier}`);
    }
    for (const name of edge.names) {
      if (!name.aliased) {
        problems.push(`unaliased binding ${name.exported} from ${edge.specifier}`);
      }
      if (!name.local.startsWith(manifest.local_prefix)) {
        problems.push(`local ${name.local} misses the ${manifest.local_prefix} prefix`);
      }
    }
    edges.push({ ...edge, resolved, runtimeModule });
  }
  if (runtimeRoots.size > 1) {
    problems.push(`second runtime root swaps the runtime: ${[...runtimeRoots].sort().join(", ")}`);
  }
  if (kind === "executable") {
    checkExecutable(manifest, modulePath, parsed, edges, problems);
  } else if (kind === "assertion_root") {
    checkAssertionRoot(manifest, modulePath, parsed, edges, problems);
  } else {
    checkAssertionCase(manifest, modulePath, parsed, edges, problems);
  }
  return problems;
}

function namesMatch(edge: ParsedEdge, wanted: Record<string, string>): boolean {
  if (edge.names.length !== Object.keys(wanted).length) {
    return false;
  }
  const names = new Map(edge.names.map((name) => [name.exported, name.local] as const));
  return Object.entries(wanted).every(([exported, local]) => names.get(exported) === local);
}

function checkExecutable(
  manifest: IngressManifest,
  modulePath: string,
  parsed: ParsedModule,
  edges: ReadonlyArray<ClassifiedEdge>,
  problems: string[],
): void {
  const entry = manifest.entries.executable;
  if (modulePath !== entry.path) {
    problems.push(`executable entry must be ${entry.path}`);
  }
  const rest = [...edges];
  for (const fixed of entry.fixed_edges) {
    const index = rest.findIndex(
      (edge) =>
        (fixed.runtime_module === undefined || edge.runtimeModule === fixed.runtime_module) &&
        (fixed.resolved === undefined || edge.resolved === fixed.resolved) &&
        namesMatch(edge, fixed.names),
    );
    if (index < 0) {
      problems.push(`missing fixed edge ${fixed.runtime_module ?? fixed.resolved ?? "?"}`);
    } else {
      rest.splice(index, 1);
    }
  }
  const main = rest.filter((edge) => edge.names.some((name) => name.local === "$canMain"));
  const other = rest.filter((edge) => !edge.names.some((name) => name.local === "$canMain"));
  if (main.length !== 1 || main[0]?.names.length !== 1) {
    problems.push("executable entry needs exactly one $canMain edge");
  } else if (main[0]?.runtimeModule !== null) {
    problems.push("main edge must be a program module, not a runtime module");
  }
  for (const edge of other) {
    problems.push(`unexpected executable edge ${edge.specifier}`);
  }
  if (parsed.exports.length > 0) {
    problems.push(`executable entry exports ${parsed.exports.join(", ")}`);
  }
  if (!new RegExp(entry.body_pattern).test(parsed.body)) {
    problems.push("executable body is not the supervisor shape");
  }
}

function checkAssertionRoot(
  manifest: IngressManifest,
  modulePath: string,
  parsed: ParsedModule,
  edges: ReadonlyArray<ClassifiedEdge>,
  problems: string[],
): void {
  const entry = manifest.entries.assertion_root;
  if (modulePath !== entry.path) {
    problems.push(`assertion entry must be ${entry.path}`);
  }
  const rest = [...edges];
  for (const fixed of entry.fixed_edges) {
    const index = rest.findIndex(
      (edge) =>
        (fixed.runtime_module === undefined || edge.runtimeModule === fixed.runtime_module) &&
        (fixed.resolved === undefined || edge.resolved === fixed.resolved) &&
        namesMatch(edge, fixed.names),
    );
    if (index < 0) {
      problems.push(`missing fixed edge ${fixed.runtime_module ?? fixed.resolved ?? "?"}`);
    } else {
      rest.splice(index, 1);
    }
  }
  if (rest.length < 1) {
    problems.push("assertion entry needs at least one case edge");
  }
  for (const edge of rest) {
    if (!edge.resolved.startsWith("assertions/")) {
      problems.push(`case edge outside assertions/: ${edge.specifier}`);
      continue;
    }
    if (
      edge.names.length !== 1 ||
      edge.names[0]?.exported !== "$canCase" ||
      !/^\$canCase\d+$/.test(edge.names[0]?.local ?? "")
    ) {
      problems.push(`case edge needs $canCase as $canCase<index>: ${edge.specifier}`);
    }
  }
  if (parsed.exports.length > 0) {
    problems.push(`assertion entry exports ${parsed.exports.join(", ")}`);
  }
  if (!new RegExp(entry.body_pattern).test(parsed.body)) {
    problems.push("assertion body is not the supervisor shape");
  }
}

function checkAssertionCase(
  manifest: IngressManifest,
  modulePath: string,
  parsed: ParsedModule,
  edges: ReadonlyArray<ClassifiedEdge>,
  problems: string[],
): void {
  if (!modulePath.startsWith("assertions/")) {
    problems.push("assertion case must live under assertions/");
  }
  const required = manifest.entries.assertion_case;
  const rest = [...edges];
  for (const fixed of required.required_runtime_edges) {
    const index = rest.findIndex(
      (edge) => edge.runtimeModule === fixed.runtime_module && edge.typeOnly === fixed.type_only,
    );
    if (index < 0) {
      problems.push(
        `missing runtime edge ${fixed.runtime_module}${fixed.type_only ? " (type-only)" : ""}`,
      );
    } else {
      rest.splice(index, 1);
    }
  }
  for (const fixed of required.required_resolved_edges) {
    const index = rest.findIndex(
      (edge) => edge.resolved === fixed.resolved && edge.typeOnly === fixed.type_only,
    );
    if (index < 0) {
      problems.push(`missing resolved edge ${fixed.resolved}`);
    } else {
      rest.splice(index, 1);
    }
  }
  for (const edge of rest) {
    if (edge.runtimeModule !== null) {
      problems.push(`unexpected runtime edge ${edge.specifier}`);
    }
  }
  const exported = [...parsed.exports].sort();
  const wanted = [...required.exports].sort();
  if (exported.join(",") !== wanted.join(",")) {
    problems.push(`assertion case must export exactly ${wanted.join(", ")}`);
  }
}
