// Browser-profile host-object predicates. Standard JavaScript has no trap-free
// proxy or native-error test, so these checks are bounded and contained: each
// call fires at most a fixed number of getPrototypeOf invocations inside
// try/catch, never reads an attacker getter, and never serializes a cause.
// A proxy whose getPrototypeOf throws (or is revoked) is detected; a proxy
// whose traps stay silent is missed and handled by the caller's ordinary
// fail-closed path. Cross-realm errors are missed. All divergences from the
// canonical module stay sanitized: no Can or native secret is disclosed.
const nativeErrorPrototypes: ReadonlySet<object> = new Set([
  Error.prototype,
  TypeError.prototype,
  RangeError.prototype,
  ReferenceError.prototype,
  SyntaxError.prototype,
  URIError.prototype,
  EvalError.prototype,
  AggregateError.prototype,
]);
const maxPrototypeDepth = 64;

function objectLike(value: unknown): value is object | Function {
  return value !== null && (typeof value === "object" || typeof value === "function");
}

export function isHostProxy(value: unknown): boolean {
  if (!objectLike(value)) return false;
  try {
    // Genuine objects never fail here; only a proxy (or revoked proxy) can.
    Object.getPrototypeOf(value);
    return false;
  } catch {
    return true;
  }
}

export function isHostNativeError(value: unknown): boolean {
  if (!objectLike(value)) return false;
  try {
    // A manual walk, not instanceof: instanceof would hang on a proxy whose
    // getPrototypeOf trap returns a cycle. The seen-set stops cycles after
    // one repeat; the depth cap stops fresh-object floods.
    let current: unknown = value;
    const seen = new Set<unknown>([value]);
    for (let depth = 0; depth < maxPrototypeDepth; depth++) {
      current = Object.getPrototypeOf(current);
      if (current === null) return false;
      if (nativeErrorPrototypes.has(current as object)) return true;
      if (seen.has(current)) return false;
      seen.add(current);
    }
    return false;
  } catch {
    return false;
  }
}
