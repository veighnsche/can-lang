// Compiler-owned SQL descriptor materialization. The checker validates each
// manifest descriptor once per compile; this factory turns the emitted table
// into inert descriptor values plus tag-call templates. Values are never
// concatenated into SQL: template() pairs static strings with caller values
// for native binding. Misuse (unknown names, forged values, arity breaks,
// corrupt tables) throws, exactly like declareAsset: compiled code cannot
// trigger it because every name and value shape is compiler-derived.
export interface SQLSegment {
  text?: string;
  param?: number;
}
export interface SQLDescriptorEntry {
  cardinality: "one" | "optional" | "many" | "execute";
  kind: string;
  segments: SQLSegment[];
  params: string[];
  paramType: string;
  rowType: string;
  limit: number;
  total: number;
  version: number;
}
export interface SQLDescriptor {
  readonly owner: string;
  readonly name: string;
  readonly cardinality: SQLDescriptorEntry["cardinality"];
  readonly kind: string;
  readonly params: readonly string[];
  readonly paramType: string;
  readonly rowType: string;
  readonly limit: number;
  readonly total: number;
  readonly version: number;
  readonly strings: TemplateStringsArray;
  readonly numbers: readonly number[];
}
export interface SQLTemplate {
  strings: TemplateStringsArray;
  values: readonly unknown[];
}

// Pinned by I36 alongside compiler/internal/sql: libpg_query 17.7.
const parserVersion = 170007;

export function createSQLDescriptors(table: Record<string, Record<string, SQLDescriptorEntry>>) {
  const descriptors = new Map<string, SQLDescriptor>();
  if (table === null || typeof table !== "object" || Array.isArray(table)) throw new TypeError("invalid sql descriptor table");
  const key = (owner: string, name: string) => `${owner.length}:${owner}${name}`;
  for (const [owner, names] of Object.entries(table)) {
    if (names === null || typeof names !== "object" || Array.isArray(names)) throw new TypeError("invalid sql descriptor table");
    for (const [name, entry] of Object.entries(names)) {
      if (typeof name !== "string" || name === "") throw new TypeError("invalid sql descriptor name");
    if (entry === null || typeof entry !== "object") throw new TypeError("invalid sql descriptor entry");
    const { cardinality, kind, segments, params, paramType, rowType, limit, total, version } = entry;
    if (cardinality !== "one" && cardinality !== "optional" && cardinality !== "many" && cardinality !== "execute") throw new TypeError("invalid sql cardinality");
    if (typeof kind !== "string" || kind === "") throw new TypeError("invalid sql statement kind");
    if (!Array.isArray(segments) || segments.length === 0) throw new TypeError("invalid sql segments");
    if (!Array.isArray(params) || !params.every((p) => typeof p === "string" && p !== "")) throw new TypeError("invalid sql params");
    if (typeof paramType !== "string" || paramType === "" || typeof rowType !== "string" || rowType === "") throw new TypeError("invalid sql identities");
    if (!Number.isInteger(limit) || !Number.isInteger(total) || total < 0 || limit < 0 || limit > total) throw new TypeError("invalid sql parameter count");
    if ((cardinality === "execute") !== (limit === 0)) throw new TypeError("invalid sql limit");
    if (params.length !== total - (limit === 0 ? 0 : 1)) throw new TypeError("invalid sql parameter names");
    if (version !== parserVersion) throw new TypeError("sql parser version mismatch");
    const literals: string[] = [];
    const numbers: number[] = [];
    let literal = "";
    for (const segment of segments) {
      if (segment === null || typeof segment !== "object") throw new TypeError("invalid sql segment");
      if (typeof segment.text === "string" && segment.param === undefined) {
        literal += segment.text;
      } else if (typeof segment.param === "number" && segment.text === undefined && Number.isInteger(segment.param) && segment.param >= 1 && segment.param <= total) {
        literals.push(literal);
        literal = "";
        numbers.push(segment.param);
      } else {
        throw new TypeError("invalid sql segment");
      }
    }
    literals.push(literal);
    if (literals.length !== numbers.length + 1) throw new TypeError("invalid sql segments");
    const strings = Object.freeze(Object.assign(literals.slice(), { raw: Object.freeze(literals.slice()) })) as unknown as TemplateStringsArray;
      descriptors.set(key(owner, name), Object.freeze({ owner, name, cardinality, kind, params: Object.freeze(params.slice()), paramType, rowType, limit, total, version, strings, numbers: Object.freeze(numbers.slice()) }));
    }
  }
  return Object.freeze({
    declareDescriptor(owner: string, name: string): SQLDescriptor {
      if (typeof owner !== "string" || typeof name !== "string") throw new TypeError("invalid sql descriptor name");
      const found = descriptors.get(key(owner, name));
      if (found === undefined) throw new TypeError("undeclared sql descriptor");
      return found;
    },
    template(descriptor: SQLDescriptor, values: readonly unknown[]): SQLTemplate {
      if (descriptor === null || typeof descriptor !== "object" || typeof descriptor.owner !== "string" || typeof descriptor.name !== "string" || descriptors.get(key(descriptor.owner, descriptor.name)) !== descriptor) throw new TypeError("forged sql descriptor");
      if (!Array.isArray(values)) throw new TypeError("invalid sql values");
      // One prepared value per parameter number; repeated sites expand to
      // the same value structurally, so callers cannot bind $1 twice over.
      if (values.length !== descriptor.total) throw new TypeError("sql descriptor arity");
      return { strings: descriptor.strings, values: descriptor.numbers.map((n) => values[n - 1]) };
    },
  });
}
