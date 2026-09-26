// D02 conformance: Op A storage reviewed adapter. The first seven legs
// re-run the D01 T2 legs verbatim against the delivered adapter; the
// trailing legs cover the D02 native host binding without a browser.
// Live-native legs (real localStorage in Chromium/WebKit/Firefox) are
// pinned under live/ and recorded as conformance debt until C01
// unblocks — see live/README.md.
import { test, expect } from "bun:test";
import {
  createFakeStorageHost,
  createNativeStorageHost,
  createStorageAdapter,
  STORAGE_KEY_MAX_CHARS,
  STORAGE_VALUE_MAX_CHARS,
  type NativeStorageShape,
} from "../adapters/storage.ts";

test("roundtrip: set, get, remove, missing reads null", () => {
  const adapter = createStorageAdapter(createFakeStorageHost());
  expect(adapter.localSet("theme", "dark")).toEqual({ ok: true, value: null });
  expect(adapter.localGet("theme")).toEqual({ ok: true, value: "dark" });
  expect(adapter.localRemove("theme")).toEqual({ ok: true, value: null });
  expect(adapter.localGet("theme")).toEqual({ ok: true, value: null });
});

test("invalid keys and values reject before host contact", () => {
  const host = createFakeStorageHost();
  const adapter = createStorageAdapter(host);
  const badKeys: unknown[] = ["", "k".repeat(STORAGE_KEY_MAX_CHARS + 1), 42, null, {}, "a�b\ud800"];
  for (const key of badKeys) {
    const got = adapter.localGet(key);
    expect(got.ok).toBe(false);
    if (!got.ok) expect(got.failure.code).toBe("storage::invalid_key");
  }
  const setBad = adapter.localSet("k", "v".repeat(STORAGE_VALUE_MAX_CHARS + 1));
  expect(setBad.ok).toBe(false);
  if (!setBad.ok) expect(setBad.failure.code).toBe("storage::invalid_key");
  expect(host.calls.length).toBe(0);
});

test("quota breach maps to quota_exceeded with no native text", () => {
  const host = createFakeStorageHost(4);
  const adapter = createStorageAdapter(host);
  expect(adapter.localSet("a", "1234").ok).toBe(true);
  const over = adapter.localSet("b", "x");
  expect(over.ok).toBe(false);
  if (!over.ok) {
    expect(over.failure.code).toBe("storage::quota_exceeded");
    expect(JSON.stringify(over)).not.toContain("QuotaExceededError");
  }
  host.injectQuota();
  const injected = adapter.localSet("a", "y");
  expect(injected.ok).toBe(false);
  if (!injected.ok) expect(injected.failure.code).toBe("storage::quota_exceeded");
});

test("unavailable target fails closed without host contact", () => {
  const host = createFakeStorageHost();
  host.setAvailable(false);
  const adapter = createStorageAdapter(host);
  for (const result of [adapter.localGet("k"), adapter.localSet("k", "v"), adapter.localRemove("k")]) {
    expect(result.ok).toBe(false);
    if (!result.ok) expect(result.failure.code).toBe("storage::unavailable");
  }
  expect(host.calls.length).toBe(0);
});

test("native security failure maps to unavailable with no native text", () => {
  const host = createFakeStorageHost();
  host.injectSecurity();
  const adapter = createStorageAdapter(host);
  const got = adapter.localGet("k");
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("storage::unavailable");
    expect(JSON.stringify(got)).not.toContain("SecurityError");
  }
});

test("immutable copies: results are frozen primitives with no aliasing", () => {
  const adapter = createStorageAdapter(createFakeStorageHost());
  expect(adapter.localSet("k", "v").ok).toBe(true);
  const first = adapter.localGet("k");
  const second = adapter.localGet("k");
  expect(first.ok && second.ok && first.value === second.value).toBe(true);
  expect(Object.isFrozen(first)).toBe(true);
  // Strings are primitives: overwriting the store cannot mutate a past result.
  expect(adapter.localSet("k", "changed").ok).toBe(true);
  expect(first.ok && first.value).toBe("v");
});

test("surface census: three ops, three failure leaves, zero callbacks", () => {
  const adapter = createStorageAdapter(createFakeStorageHost());
  expect(Object.keys(adapter).sort()).toEqual(["localGet", "localRemove", "localSet"]);
  const leaves = new Set<string>();
  const collect = (result: { ok: boolean; failure?: { code: string } }) => {
    if (!result.ok && result.failure) leaves.add(result.failure.code);
  };
  collect(adapter.localGet(""));
  const quotaHost = createFakeStorageHost(1);
  const quotaAdapter = createStorageAdapter(quotaHost);
  expect(quotaAdapter.localSet("a", "b").ok).toBe(true);
  collect(quotaAdapter.localSet("c", "d"));
  const downHost = createFakeStorageHost();
  downHost.setAvailable(false);
  collect(createStorageAdapter(downHost).localGet("k"));
  expect([...leaves].sort()).toEqual([
    "storage::invalid_key",
    "storage::quota_exceeded",
    "storage::unavailable",
  ]);
});

function mapBackedShape(store: Map<string, string>): NativeStorageShape & { detached(): boolean } {
  let detached = false;
  const shape: NativeStorageShape = {
    getItem(key: string): string | null {
      // Native Storage methods throw when detached from their
      // receiver; the double reproduces that so the binding leg
      // proves receiver-preserving delegation.
      if (detached) throw new DOMException("detached", "TypeError");
      return store.has(key) ? store.get(key)! : null;
    },
    setItem(key: string, value: string): void {
      if (detached) throw new DOMException("detached", "TypeError");
      store.set(key, value);
    },
    removeItem(key: string): void {
      if (detached) throw new DOMException("detached", "TypeError");
      store.delete(key);
    },
  };
  return Object.assign(shape, {
    detached(): boolean {
      detached = true;
      return detached;
    },
  });
}

test("native binding: absent storage reads unavailable with no host contact", () => {
  for (const scope of [{}, { localStorage: null }, { localStorage: undefined }, null, 42, "x"]) {
    const adapter = createStorageAdapter(createNativeStorageHost(scope));
    for (const result of [adapter.localGet("k"), adapter.localSet("k", "v"), adapter.localRemove("k")]) {
      expect(result.ok).toBe(false);
      if (!result.ok) expect(result.failure.code).toBe("storage::unavailable");
    }
  }
});

test("native binding: throwing scope access fails closed", () => {
  const scope = {};
  Object.defineProperty(scope, "localStorage", {
    get(): never {
      // Opaque-origin shape: accessing the property itself throws.
      throw new DOMException("blocked", "SecurityError");
    },
  });
  const adapter = createStorageAdapter(createNativeStorageHost(scope));
  const got = adapter.localGet("k");
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("storage::unavailable");
    expect(JSON.stringify(got)).not.toContain("SecurityError");
  }
});

test("native binding: partial native shape fails closed", () => {
  const adapter = createStorageAdapter(createNativeStorageHost({ localStorage: { getItem: () => null } }));
  const got = adapter.localSet("k", "v");
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("storage::unavailable");
});

test("native binding: probe failure fails closed", () => {
  const shape = mapBackedShape(new Map());
  shape.detached();
  const adapter = createStorageAdapter(createNativeStorageHost({ localStorage: shape }));
  const got = adapter.localGet("k");
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("storage::unavailable");
});

test("native binding: well-formed shape roundtrips with receiver preserved", () => {
  const adapter = createStorageAdapter(createNativeStorageHost({ localStorage: mapBackedShape(new Map()) }));
  expect(adapter.localSet("theme", "dark")).toEqual({ ok: true, value: null });
  expect(adapter.localGet("theme")).toEqual({ ok: true, value: "dark" });
  expect(adapter.localRemove("theme")).toEqual({ ok: true, value: null });
  expect(adapter.localGet("theme")).toEqual({ ok: true, value: null });
});

test("native binding: quota throw through the binding maps with no native text", () => {
  const shape = mapBackedShape(new Map());
  const adapter = createStorageAdapter(createNativeStorageHost({ localStorage: shape }));
  shape.setItem = (): never => {
    throw new DOMException("quota", "QuotaExceededError");
  };
  const over = adapter.localSet("b", "x");
  expect(over.ok).toBe(false);
  if (!over.ok) {
    expect(over.failure.code).toBe("storage::quota_exceeded");
    expect(JSON.stringify(over)).not.toContain("QuotaExceededError");
  }
});
