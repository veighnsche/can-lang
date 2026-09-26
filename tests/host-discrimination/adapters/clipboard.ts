// X-R01-1 (D01) UNREVIEWED PROTOTYPE — reviewed-adapter tier, Op B.
//
// Clipboard text exchange through the W3C Clipboard API shape
// (`navigator.clipboard.readText/writeText`): async, permission-gated,
// secure-context-only. The adapter injects its host so every leg runs
// under Bun; native-call fidelity is pinned to the specification
// (Promise<string>/Promise<void>, NotAllowedError on denial,
// SecurityError outside secure contexts) with two documented
// uncertainties deferred to D02 live verification: the exact empty-
// clipboard read shape and the engagement-gating matrix. Both are
// non-discriminating: every tier shares the same native facts.
//
// Audit-contract shape exercised here: declared I/O types, immutable
// copying, failure mapping, target availability, permission gating,
// async reentrancy rules (explicit no-ordering rule), and stateless
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
