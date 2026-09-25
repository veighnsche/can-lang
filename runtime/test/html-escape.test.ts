import { test, expect } from "bun:test";
import { escapeHTML } from "../platform/html.ts";

// The portable escaper must match the native one byte for byte on every
// input: both profiles share this single implementation.
const vectors: string[] = [
  "",
  "&<>\"'=/`",
  "a&b<c>d\"e'f",
  "\u00e9\u{1f600}",
  "\u0000\u0007'<>&\"",
  "\ud800",
  "\udc00",
  "\ud800x\udc00",
  "&amp;",
  "&lt;&gt;&quot;&#x27;",
  "a".repeat(1000) + "<" + "b".repeat(1000),
  "<script>alert('x')</script>",
  "\"quoted\" and 'singles'",
];
for (let i = 0; i < 128; i++) vectors.push(String.fromCharCode(i));
for (let i = 0; i < 256; i++)
  vectors.push("x" + String.fromCharCode(i) + "&" + String.fromCharCode(255 - i));

test("portable escape matches native on adversarial vectors", () => {
  expect(vectors.length).toBeGreaterThan(350);
  for (const input of vectors) {
    expect(escapeHTML(input)).toBe(Bun.escapeHTML(input));
  }
});

test("only the five entities escape", () => {
  expect(escapeHTML("&<>\"'")).toBe("&amp;&lt;&gt;&quot;&#x27;");
  expect(escapeHTML("=/`é😀 \t\n")).toBe("=/`é😀 \t\n");
});
