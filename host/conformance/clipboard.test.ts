// D02 conformance: Op B clipboard reviewed adapter. The first eight
// legs re-run the D01 T2 legs verbatim against the delivered adapter;
// the trailing legs cover the D02 native host binding without a
// browser. Live-native legs (real navigator.clipboard on
// Chromium/WebKit/Firefox, incl. the permission/engagement matrix and
// the empty-read shape) are pinned under live/ and recorded as
// conformance debt until C01 unblocks — see live/README.md.
import { test, expect } from "bun:test";
import {
  createClipboardAdapter,
  createFakeClipboardHost,
  createNativeClipboardHost,
  CLIPBOARD_TEXT_MAX_CHARS,
  type NativeClipboardShape,
} from "../adapters/clipboard.ts";

test("roundtrip: write then read returns the text", async () => {
  const adapter = createClipboardAdapter(createFakeClipboardHost());
  expect(await adapter.writeText("invoice-42")).toEqual({ ok: true, value: null });
  expect(await adapter.readText()).toEqual({ ok: true, value: "invoice-42" });
});

test("empty clipboard reads the empty leaf", async () => {
  const adapter = createClipboardAdapter(createFakeClipboardHost());
  const got = await adapter.readText();
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("clipboard::empty");
});

test("invalid writes reject before host contact", async () => {
  const host = createFakeClipboardHost();
  const adapter = createClipboardAdapter(host);
  for (const text of ["", "x".repeat(CLIPBOARD_TEXT_MAX_CHARS + 1), 7, null, "\ud800"]) {
    const got = await adapter.writeText(text);
    expect(got.ok).toBe(false);
    if (!got.ok) expect(got.failure.code).toBe("clipboard::invalid_text");
  }
  expect(host.calls.length).toBe(0);
});

test("denied permission fails closed before clipboard contact", async () => {
  const host = createFakeClipboardHost();
  host.setPermission("read", "denied");
  host.setPermission("write", "denied");
  const adapter = createClipboardAdapter(host);
  const read = await adapter.readText();
  expect(read.ok).toBe(false);
  if (!read.ok) expect(read.failure.code).toBe("clipboard::denied");
  const write = await adapter.writeText("x");
  expect(write.ok).toBe(false);
  if (!write.ok) expect(write.failure.code).toBe("clipboard::denied");
  expect(host.calls.every((call) => call.op.startsWith("queryPermission"))).toBe(true);
});

test("unavailable target fails closed without host contact", async () => {
  const host = createFakeClipboardHost();
  host.setAvailable(false);
  const adapter = createClipboardAdapter(host);
  const read = await adapter.readText();
  expect(read.ok).toBe(false);
  if (!read.ok) expect(read.failure.code).toBe("clipboard::unavailable");
  const write = await adapter.writeText("x");
  expect(write.ok).toBe(false);
  if (!write.ok) expect(write.failure.code).toBe("clipboard::unavailable");
  expect(host.calls.length).toBe(0);
});

test("native denial and native failure map with no native text", async () => {
  const host = createFakeClipboardHost();
  host.setPermission("read", "prompt");
  host.injectNativeFailure("NotAllowedError");
  const adapter = createClipboardAdapter(host);
  const denied = await adapter.readText();
  expect(denied.ok).toBe(false);
  if (!denied.ok) {
    expect(denied.failure.code).toBe("clipboard::denied");
    expect(JSON.stringify(denied)).not.toContain("NotAllowedError");
  }
  host.injectNativeFailure("SecurityError");
  const failed = await adapter.writeText("x");
  expect(failed.ok).toBe(false);
  if (!failed.ok) {
    expect(failed.failure.code).toBe("clipboard::unavailable");
    expect(JSON.stringify(failed)).not.toContain("SecurityError");
  }
});

test("reentrancy: concurrent writes settle independently in host order", async () => {
  const adapter = createClipboardAdapter(createFakeClipboardHost());
  const [first, second] = await Promise.all([adapter.writeText("a"), adapter.writeText("b")]);
  expect(first.ok).toBe(true);
  expect(second.ok).toBe(true);
  // Documented no-ordering rule: the leg pins fake order (call order)
  // while the contract promises none across implementations.
  expect(await adapter.readText()).toEqual({ ok: true, value: "b" });
});

test("surface census: two ops, four failure leaves, zero callbacks", async () => {
  const adapter = createClipboardAdapter(createFakeClipboardHost());
  expect(Object.keys(adapter).sort()).toEqual(["readText", "writeText"]);
  const leaves = new Set<string>();
  const invalid = await adapter.writeText("");
  if (!invalid.ok) leaves.add(invalid.failure.code);
  const empty = await adapter.readText();
  if (!empty.ok) leaves.add(empty.failure.code);
  const deniedHost = createFakeClipboardHost();
  deniedHost.setPermission("read", "denied");
  const denied = await createClipboardAdapter(deniedHost).readText();
  if (!denied.ok) leaves.add(denied.failure.code);
  const downHost = createFakeClipboardHost();
  downHost.setAvailable(false);
  const down = await createClipboardAdapter(downHost).readText();
  if (!down.ok) leaves.add(down.failure.code);
  expect([...leaves].sort()).toEqual([
    "clipboard::denied",
    "clipboard::empty",
    "clipboard::invalid_text",
    "clipboard::unavailable",
  ]);
});

function cellBackedShape(cell: { text: string | null }): NativeClipboardShape {
  return {
    async readText(): Promise<string> {
      // Native resolves a string; the double resolves "" for an
      // empty cell so the binding's normalization is exercised.
      return cell.text ?? "";
    },
    async writeText(text: string): Promise<void> {
      cell.text = text;
    },
  };
}

function nativeScope(shape: unknown, secure?: unknown): unknown {
  const scope: Record<string, unknown> = { navigator: { clipboard: shape } };
  if (secure !== undefined) scope["isSecureContext"] = secure;
  return scope;
}

test("native binding: absent clipboard reads unavailable with no host contact", async () => {
  const scopes: unknown[] = [
    {},
    { navigator: null },
    { navigator: {} },
    { navigator: { clipboard: null } },
    { navigator: { clipboard: { readText: async () => "x" } } },
    nativeScope(cellBackedShape({ text: "x" }), false),
    null,
    42,
  ];
  for (const scope of scopes) {
    const adapter = createClipboardAdapter(createNativeClipboardHost(scope));
    const read = await adapter.readText();
    expect(read.ok).toBe(false);
    if (!read.ok) expect(read.failure.code).toBe("clipboard::unavailable");
    const write = await adapter.writeText("x");
    expect(write.ok).toBe(false);
    if (!write.ok) expect(write.failure.code).toBe("clipboard::unavailable");
  }
});

test("native binding: throwing scope access fails closed", async () => {
  const scope = {};
  Object.defineProperty(scope, "navigator", {
    get(): never {
      throw new DOMException("blocked", "SecurityError");
    },
  });
  const adapter = createClipboardAdapter(createNativeClipboardHost(scope));
  const got = await adapter.readText();
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("clipboard::unavailable");
    expect(JSON.stringify(got)).not.toContain("SecurityError");
  }
});

test("native binding: well-formed shape roundtrips; pre-gate reports prompt", async () => {
  const host = createNativeClipboardHost(nativeScope(cellBackedShape({ text: null })));
  expect(host.queryPermission("read")).toBe("prompt");
  expect(host.queryPermission("write")).toBe("prompt");
  const adapter = createClipboardAdapter(host);
  expect(await adapter.writeText("invoice-42")).toEqual({ ok: true, value: null });
  expect(await adapter.readText()).toEqual({ ok: true, value: "invoice-42" });
});

test("native binding: empty resolution normalizes to the empty leaf", async () => {
  const adapter = createClipboardAdapter(
    createNativeClipboardHost(nativeScope(cellBackedShape({ text: null }))),
  );
  const got = await adapter.readText();
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("clipboard::empty");
});

test("native binding: native denial maps with no native text", async () => {
  const shape = cellBackedShape({ text: "x" });
  shape.readText = async (): Promise<string> => {
    throw new DOMException("denied", "NotAllowedError");
  };
  const adapter = createClipboardAdapter(createNativeClipboardHost(nativeScope(shape)));
  const got = await adapter.readText();
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("clipboard::denied");
    expect(JSON.stringify(got)).not.toContain("NotAllowedError");
  }
});
