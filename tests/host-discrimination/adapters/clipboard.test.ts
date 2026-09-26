import { test, expect } from "bun:test";
import { createClipboardAdapter, createFakeClipboardHost, CLIPBOARD_TEXT_MAX_CHARS } from "./clipboard.ts";

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
