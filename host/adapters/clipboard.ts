// D02 delivered reviewed-adapter tier, Op B (PENDING-REVIEW — not admitted).
//
// Clipboard text exchange through the W3C Clipboard API shape
// (`navigator.clipboard.readText/writeText`): async, permission-gated,
// secure-context-only. Validators and failure leaves below are
// inherited verbatim from the D01 prototype
// (`tests/host-discrimination/adapters/clipboard.ts`); the only
// addition is the native host binding at the end of this file, which
// the live-browser conformance leg exercises. Native-call fidelity is
// pinned to the specification (Promise<string>/Promise<void>,
// NotAllowedError on denial, SecurityError outside secure contexts)
// with two documented uncertainties owned by D02 live verification:
// the exact empty-clipboard read shape and the engagement-gating
// matrix. Both are non-discriminating: every tier shares the same
// native facts.
//
// Audit contract: declared I/O types, immutable copying, failure
// mapping, target availability, permission gating, async reentrancy
// rules (explicit no-ordering rule), and stateless
// ownership/disposal.
export const CLIPBOARD_TEXT_MAX_CHARS = 1048576;

export type ClipboardFailureCode =
  | "clipboard::invalid_text"
  | "clipboard::denied"
  | "clipboard::empty"
  | "clipboard::unavailable";

export type ClipboardFailure = Readonly<{ code: ClipboardFailureCode; detail: string }>;

export type ClipboardResult<T> =
  | Readonly<{ ok: true; value: T }>
  | Readonly<{ ok: false; failure: ClipboardFailure }>;

function ok<T>(value: T): ClipboardResult<T> {
  return Object.freeze({ ok: true, value }) as ClipboardResult<T>;
}

function fail<T>(code: ClipboardFailureCode, detail: string): ClipboardResult<T> {
  return Object.freeze({ ok: false, failure: Object.freeze({ code, detail }) }) as ClipboardResult<T>;
}

export type ClipboardPermission = "granted" | "denied" | "prompt";

// ClipboardHost is the exact native surface the adapter needs.
// `available` is false outside secure contexts and where the API is
// absent. `queryPermission` mirrors the Permissions API
// (`clipboard-read`/`clipboard-write`); a denied verdict fails closed
// before any clipboard call. `readText` resolves `null` when no text
// is available (empty-clipboard shape; D02 verifies against live
// browsers where implementations may reject instead).
export type ClipboardHost = {
  readonly available: boolean;
  queryPermission(kind: "read" | "write"): ClipboardPermission;
  readText(): Promise<string | null>;
  writeText(text: string): Promise<void>;
};

export type ClipboardAdapter = Readonly<{
  readText: () => Promise<ClipboardResult<string>>;
  writeText: (text: unknown) => Promise<ClipboardResult<null>>;
}>;

function mapNative(error: unknown): ClipboardFailure {
  const name = typeof error === "object" && error !== null && "name" in error ? String(error.name) : "";
  if (name === "NotAllowedError") return { code: "clipboard::denied", detail: "clipboard access denied" };
  return { code: "clipboard::unavailable", detail: "clipboard unavailable" };
}

export function createClipboardAdapter(host: ClipboardHost): ClipboardAdapter {
  return Object.freeze({
    async readText(): Promise<ClipboardResult<string>> {
      if (!host.available) return fail("clipboard::unavailable", "clipboard unavailable");
      if (host.queryPermission("read") === "denied") return fail("clipboard::denied", "clipboard access denied");
      try {
        const text = await host.readText();
        if (text === null) return fail("clipboard::empty", "clipboard holds no text");
        return ok(text);
      } catch (error) {
        const mapped = mapNative(error);
        return fail(mapped.code, mapped.detail);
      }
    },
    async writeText(text: unknown): Promise<ClipboardResult<null>> {
      if (typeof text !== "string" || text.length === 0 || !text.isWellFormed() || text.length > CLIPBOARD_TEXT_MAX_CHARS) {
        return fail("clipboard::invalid_text", "text must be well-formed non-empty text of at most 1048576 chars");
      }
      if (!host.available) return fail("clipboard::unavailable", "clipboard unavailable");
      if (host.queryPermission("write") === "denied") return fail("clipboard::denied", "clipboard access denied");
      try {
        await host.writeText(text);
        return ok(null);
      } catch (error) {
        const mapped = mapNative(error);
        return fail(mapped.code, mapped.detail);
      }
    },
  });
}

// Reentrancy rule (explicit, weak): concurrent calls settle
// independently in host order; the adapter provides no serialization
// and no cross-call ordering guarantee. Callers needing atomicity
// must sequence awaits. The fake below resolves in call order so the
// leg is deterministic.

export type FakeClipboardHost = ClipboardHost & {
  calls: Array<Readonly<{ op: string }>>;
  setAvailable(available: boolean): void;
  setPermission(kind: "read" | "write", permission: ClipboardPermission): void;
  injectNativeFailure(name: string): void;
  clearFault(): void;
  seed(text: string | null): void;
};

export function createFakeClipboardHost(): FakeClipboardHost {
  const calls: Array<Readonly<{ op: string }>> = [];
  let available = true;
  const permission: Record<"read" | "write", ClipboardPermission> = { read: "granted", write: "granted" };
  let fault = "";
  let cell: string | null = null;
  const maybeFault = (): void => {
    if (fault !== "") throw new DOMException("injected", fault);
  };
  return {
    calls,
    get available(): boolean {
      return available;
    },
    queryPermission(kind: "read" | "write"): ClipboardPermission {
      calls.push({ op: `queryPermission:${kind}` });
      return permission[kind];
    },
    async readText(): Promise<string | null> {
      calls.push({ op: "readText" });
      maybeFault();
      return cell;
    },
    async writeText(text: string): Promise<void> {
      calls.push({ op: "writeText" });
      maybeFault();
      cell = text;
    },
    setAvailable(next: boolean): void {
      available = next;
    },
    setPermission(kind: "read" | "write", next: ClipboardPermission): void {
      permission[kind] = next;
    },
    injectNativeFailure(name: string): void {
      fault = name;
    },
    clearFault(): void {
      fault = "";
    },
    seed(text: string | null): void {
      cell = text;
    },
  };
}

// NativeClipboardShape is the minimal surface
// `createNativeClipboardHost` needs from `scope.navigator.clipboard`.
// The native `readText` resolves a string; the binding normalizes an
// empty resolution to `null` (the adapter's empty leaf), because an
// empty string on the device clipboard is indistinguishable from no
// text and no implementation reports it as a successful read of text.
// The live leg verifies this normalization per browser; if a browser
// rejects empty reads instead, the rejection name is pinned there and
// mapped here in the live follow-up (recorded, not assumed).
export type NativeClipboardShape = {
  readText(): Promise<string>;
  writeText(text: string): Promise<void>;
};

function closedNativeClipboardHost(): ClipboardHost {
  const closed = (): never => {
    throw new Error("unreachable: adapter gates on available before calling the host");
  };
  return {
    available: false,
    queryPermission: closed,
    readText: closed,
    writeText: closed,
  };
}

export function createNativeClipboardHost(scope: unknown): ClipboardHost {
  try {
    if (scope === null || typeof scope !== "object") return closedNativeClipboardHost();
    const holder = scope as Record<string, unknown>;
    // An explicitly insecure context has no clipboard API by
    // construction; anything else defers to shape detection so
    // doubles without the flag still bind.
    if (holder["isSecureContext"] === false) return closedNativeClipboardHost();
    const navigator = holder["navigator"];
    if (navigator === null || typeof navigator !== "object") return closedNativeClipboardHost();
    const clipboard = (navigator as Record<string, unknown>)["clipboard"];
    if (clipboard === null || typeof clipboard !== "object") return closedNativeClipboardHost();
    const shape = clipboard as Partial<NativeClipboardShape>;
    if (typeof shape.readText !== "function" || typeof shape.writeText !== "function") {
      return closedNativeClipboardHost();
    }
    const native = shape as NativeClipboardShape;
    return {
      available: true,
      // The Permissions API is async while the host contract's
      // pre-gate is sync, so the native binding cannot query it:
      // it reports "prompt" (unknown) and lets the attempt decide.
      // Denial then surfaces as NotAllowedError, which the
      // adapter maps to the denied leaf. Injected doubles keep
      // the synchronous pre-gate for policy-driven tests.
      queryPermission(): ClipboardPermission {
        return "prompt";
      },
      // Closures keep the native receiver.
      readText: async (): Promise<string | null> => {
        const text = await native.readText();
        return text === "" ? null : text;
      },
      writeText: async (text: string): Promise<void> => {
        await native.writeText(text);
      },
    };
  } catch {
    return closedNativeClipboardHost();
  }
}
