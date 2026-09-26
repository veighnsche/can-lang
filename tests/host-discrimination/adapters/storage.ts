// X-R01-1 (D01) UNREVIEWED PROTOTYPE — reviewed-adapter tier, Op A.
//
// Persistent client key-value storage through the WHATWG Storage shape
// (`localStorage.getItem/setItem/removeItem`): synchronous, string-only,
// per-origin, quota-bound. The adapter injects its host so every leg
// runs under Bun; native-call fidelity is pinned to the living standard
// (string keys/values, `null` for missing keys, `QuotaExceededError` on
// quota breach, `SecurityError`/opaque origin when storage is blocked)
// with live-browser verification deferred to D02 conformance.
//
// Audit-contract shape exercised here: declared I/O types, immutable
// copying, failure mapping, target availability, callback/reentrancy
// rules (none: the surface is synchronous), ownership/disposal
// (none: the adapter is stateless).
export const STORAGE_KEY_MAX_CHARS = 256;
export const STORAGE_VALUE_MAX_CHARS = 1048576;

export type StorageFailureCode = "storage::invalid_key" | "storage::quota_exceeded" | "storage::unavailable";

export type StorageFailure = Readonly<{
  code: StorageFailureCode;
  detail: string;
}>;

export type StorageResult<T> =
  | Readonly<{ ok: true; value: T }>
  | Readonly<{ ok: false; failure: StorageFailure }>;

export function ok<T>(value: T): StorageResult<T> {
  return Object.freeze({ ok: true, value }) as StorageResult<T>;
}

function fail<T>(code: StorageFailureCode, detail: string): StorageResult<T> {
  return Object.freeze({ ok: false, failure: Object.freeze({ code, detail }) }) as StorageResult<T>;
}

// StorageHost is the exact native surface the adapter needs. `available`
// is false for opaque origins and blocked storage (the standard's
// SecurityError shape), in which case the adapter never touches the host.
export type StorageHost = {
  readonly available: boolean;
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
};

export type StorageAdapter = Readonly<{
  localGet: (key: unknown) => StorageResult<string | null>;
  localSet: (key: unknown, value: unknown) => StorageResult<null>;
  localRemove: (key: unknown) => StorageResult<null>;
}>;

function checkKey(key: unknown): string | StorageFailure {
  if (typeof key !== "string" || key.length === 0 || key.length > STORAGE_KEY_MAX_CHARS || !key.isWellFormed()) {
    return { code: "storage::invalid_key", detail: "key must be well-formed text of 1..256 chars" };
  }
  return key;
}

function checkValue(value: unknown): string | StorageFailure {
  if (typeof value !== "string" || !value.isWellFormed() || value.length > STORAGE_VALUE_MAX_CHARS) {
    return { code: "storage::invalid_key", detail: "value must be well-formed text of at most 1048576 chars" };
  }
  return value;
}

// Native throws never cross the boundary: quota maps to its leaf and
// every other native failure (SecurityError, unknown) maps to the fixed
// unavailable leaf with no native message. No native text is logged.
function mapNative(error: unknown): StorageFailure {
  const name = typeof error === "object" && error !== null && "name" in error ? String(error.name) : "";
  if (name === "QuotaExceededError") return { code: "storage::quota_exceeded", detail: "origin quota exceeded" };
  return { code: "storage::unavailable", detail: "storage unavailable for this origin" };
}

export function createStorageAdapter(host: StorageHost): StorageAdapter {
  const gate = (): StorageFailure | null =>
    host.available ? null : { code: "storage::unavailable", detail: "storage unavailable for this origin" };
  return Object.freeze({
    localGet(key: unknown): StorageResult<string | null> {
      const checked = checkKey(key);
      if (typeof checked !== "string") return fail(checked.code, checked.detail);
      const blocked = gate();
      if (blocked !== null) return fail(blocked.code, blocked.detail);
      try {
        // Strings are immutable primitives: no copy boundary to cross.
        return ok(host.getItem(checked));
      } catch (error) {
        const mapped = mapNative(error);
        return fail(mapped.code, mapped.detail);
      }
    },
    localSet(key: unknown, value: unknown): StorageResult<null> {
      const checkedKey = checkKey(key);
      if (typeof checkedKey !== "string") return fail(checkedKey.code, checkedKey.detail);
      const checkedValue = checkValue(value);
      if (typeof checkedValue !== "string") return fail(checkedValue.code, checkedValue.detail);
      const blocked = gate();
      if (blocked !== null) return fail(blocked.code, blocked.detail);
      try {
        host.setItem(checkedKey, checkedValue);
        return ok(null);
      } catch (error) {
        const mapped = mapNative(error);
        return fail(mapped.code, mapped.detail);
      }
    },
    localRemove(key: unknown): StorageResult<null> {
      const checked = checkKey(key);
      if (typeof checked !== "string") return fail(checked.code, checked.detail);
      const blocked = gate();
      if (blocked !== null) return fail(blocked.code, blocked.detail);
      try {
        host.removeItem(checked);
        return ok(null);
      } catch (error) {
        const mapped = mapNative(error);
        return fail(mapped.code, mapped.detail);
      }
    },
  });
}

// FakeStorageHost is the injected double for Bun legs: a Map-backed
// store with a quota budget, an availability flag, fault injection,
// and a call log proving no-contact legs.
export type FakeStorageHost = StorageHost & {
  calls: Array<Readonly<{ op: string; key: string }>>;
  injectQuota(): void;
  injectSecurity(): void;
  clearFault(): void;
  setAvailable(available: boolean): void;
  storedChars(): number;
};

export function createFakeStorageHost(quotaChars = 1024): FakeStorageHost {
  const store = new Map<string, string>();
  const calls: Array<Readonly<{ op: string; key: string }>> = [];
  let available = true;
  let fault: "" | "quota" | "security" = "";
  return {
    calls,
    get available(): boolean {
      return available;
    },
    getItem(key: string): string | null {
      calls.push({ op: "getItem", key });
      if (fault === "security") throw new DOMException("blocked", "SecurityError");
      return store.has(key) ? store.get(key)! : null;
    },
    setItem(key: string, value: string): void {
      calls.push({ op: "setItem", key });
      if (fault === "security") throw new DOMException("blocked", "SecurityError");
      if (fault === "quota") throw new DOMException("quota", "QuotaExceededError");
      const current = store.has(key) ? store.get(key)!.length : 0;
      let total = 0;
      for (const entry of store.values()) total += entry.length;
      if (total - current + value.length > quotaChars) throw new DOMException("quota", "QuotaExceededError");
      store.set(key, value);
    },
    removeItem(key: string): void {
      calls.push({ op: "removeItem", key });
      if (fault === "security") throw new DOMException("blocked", "SecurityError");
      store.delete(key);
    },
    injectQuota(): void {
      fault = "quota";
    },
    injectSecurity(): void {
      fault = "security";
    },
    clearFault(): void {
      fault = "";
    },
    setAvailable(next: boolean): void {
      available = next;
    },
    storedChars(): number {
      let total = 0;
      for (const entry of store.values()) total += entry.length;
      return total;
    },
  };
}
