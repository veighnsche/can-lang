import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createMarkdown } from "../platform/markdown.ts";
import { renderSafe, isHTMLValue } from "../platform/html.ts";
import { value, type Completion } from "../completion.ts";
import { dataProperty } from "../data.ts";
const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const str = shape("primitive", "str"),
  int = shape("primitive", "int");
const declarations = catalogue.errors.filter((e) => [1220, 1221, 1349].includes(e.id));
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, int, ...errors],
});
const identity = (name: string) => errors.find((e) => e.declaration === name)!.identity;
const md = createMarkdown(domain, {
  overLimit: identity("can.std.markdown@1::over_limit"),
  htmlStructure: identity("can.std.html@1::invalid_structure"),
  htmlURL: identity("can.std.html@1::invalid_url"),
});
function check(result: Completion, id: number) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error();
  expect(domainFailureDiagnostics(result.value).declaration.id).toBe(id);
}
const failPayload = (result: Completion) => {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error();
  return domainFailureDiagnostics(result.value).payload;
};
const safe = async (source: string) => {
  const out = value(await md.renderSafe(source));
  expect(isHTMLValue("safe", out)).toBe(true);
  return renderSafe(out as never);
};
test("safe rendering covers blocks, spans and code", async () => {
  expect(await safe("# Hello World!\n\nSome *em* **strong** ~~del~~ `code` text.\n")).toBe(
    '<h1 id="hello-world">Hello World!</h1><p>Some <em>em</em> <strong>strong</strong> <del>del</del> <code>code</code> text.</p>',
  );
  expect(await safe("> quote *em*\n\n---\n")).toBe(
    "<blockquote><p>quote <em>em</em></p></blockquote><hr>",
  );
  expect(await safe("Setext\n===\n\ntext\n---\n")).toBe(
    '<h1 id="setext">Setext</h1><h2 id="text">text</h2>',
  );
  expect(await safe("```js\nf()\n```\n\n    ind\n")).toBe(
    '<pre><code class="language-js">f()\n</code></pre><pre><code>ind\n</code></pre>',
  );
  expect(await safe("```c++\nx\n```\n")).toBe("<pre><code>x\n</code></pre>");
  expect(await safe("## Dup\n## Dup\n")).toBe('<h2 id="dup">Dup</h2><h2 id="dup-1">Dup</h2>');
  expect(await safe("**a *b* c**\n")).toBe("<p><strong>a <em>b</em> c</strong></p>");
  expect(await safe("")).toBe("");
});
test("lists, task items and ordered starts render fully", async () => {
  expect(await safe("- one\n- two\n")).toBe("<ul><li>one</li><li>two</li></ul>");
  expect(await safe("1. a\n2. b\n")).toBe("<ol><li>a</li><li>b</li></ol>");
  expect(await safe("5. five\n6. six\n")).toBe('<ol start="5"><li>five</li><li>six</li></ol>');
  expect(await safe("- a\n  - b\n")).toBe("<ul><li>a<ul><li>b</li></ul></li></ul>");
  expect(await safe("- [ ] t1\n- [x] t2\n")).toBe(
    '<ul><li><input type="checkbox" disabled="disabled">t1</li><li><input type="checkbox" disabled="disabled" checked="checked">t2</li></ul>',
  );
  expect(await safe("1. [x] done\n2. [ ] todo\n")).toBe(
    '<ol><li><input type="checkbox" disabled="disabled" checked="checked">done</li><li><input type="checkbox" disabled="disabled">todo</li></ol>',
  );
});
test("tables keep header/body shape and alignment", async () => {
  expect(await safe("| h1 | h2 |\n|:---|---:|\n| c1 | c2 |\n")).toBe(
    '<table><thead><tr><th align="left">h1</th><th align="right">h2</th></tr></thead><tbody><tr><td align="left">c1</td><td align="right">c2</td></tr></tbody></table>',
  );
  expect(await safe("| x |\n|:-:|\n| 1 |\n")).toBe(
    '<table><thead><tr><th align="center">x</th></tr></thead><tbody><tr><td align="center">1</td></tr></tbody></table>',
  );
  expect(await safe("| a |\n|---|\n| 1 |\n")).toBe(
    "<table><thead><tr><th>a</th></tr></thead><tbody><tr><td>1</td></tr></tbody></table>",
  );
  expect(await safe("| a |\n|---|\n")).toBe(
    "<table><thead><tr><th>a</th></tr></thead><tbody></tbody></table>",
  );
});
test("links and images enforce the strict URL policy", async () => {
  expect(await safe('[t](https://x.test/u "ti")\n')).toBe(
    '<p><a href="https://x.test/u" title="ti">t</a></p>',
  );
  expect(await safe("[l](/p?q#f)\n")).toBe('<p><a href="/p?q#f">l</a></p>');
  expect(await safe('![a](https://i.test/p.png "ti")\n')).toBe(
    '<p><img src="https://i.test/p.png" alt="a" title="ti"></p>',
  );
  expect(await safe("[![t](https://i.test/p.png)](https://v.test)\n")).toBe(
    '<p><a href="https://v.test/"><img src="https://i.test/p.png" alt="t"></a></p>',
  );
  for (const [href, reason] of [
    ["javascript:alert(1)", "scheme"],
    ["data:text/html,x", "scheme"],
    ["http://a.test", "scheme"],
    ["mailto:a@b.test", "scheme"],
    ["#sec", "scheme"],
    ["", "scheme"],
    ["u", "scheme"],
    ["//evil.test", "syntax"],
  ]) {
    const result = await md.renderSafe(`[t](${href})\n`);
    check(result, 1221);
    expect(dataProperty(failPayload(result), "reason")).toBe(reason);
  }
  check(await md.renderSafe("![empty]()\n"), 1221);
});
test("raw HTML degrades to escaped text, never markup", async () => {
  const out = await safe(
    '<script>alert(1)</script>\n\ntext <b on=x=y>bold</b> end & "quoted"\n<!-- c -->\n',
  );
  expect(out).toBe(
    "<p>&lt;script&gt;alert(1)&lt;/script&gt;</p><p>text &lt;b on=x=y&gt;bold&lt;/b&gt; end &amp; &quot;quoted&quot;\n&lt;!-- c --&gt;</p>",
  );
  expect(out).not.toContain("<script");
  expect(await safe('<a href="https://x.test">raw link</a>\n')).toBe(
    "<p>&lt;a href=&quot;https://x.test&quot;&gt;raw link&lt;/a&gt;</p>",
  );
});
test("breaks normalize to soft newlines", async () => {
  expect(await safe("line one\nline two  \nline three\n")).toBe(
    "<p>line one\nline two\nline three</p>",
  );
  expect(await safe("a\\\nb\n")).toBe("<p>a\nb</p>");
  expect(await safe("\\\n")).toBe("<p>\\</p>");
});
test("disabled extensions stay literal text", async () => {
  expect(await safe("[[Tgt|lbl]] and __u__ and $x$ and https://bare.test/x\n")).toBe(
    "<p>[[Tgt|lbl]] and <strong>u</strong> and $x$ and https://bare.test/x</p>",
  );
  expect(await safe("#nospace\n")).toBe("<p>#nospace</p>");
  // noHtmlSpans demotes <url> autolinks to text: visible, fail-closed.
  expect(await safe("<https://auto.test/x>\n")).toBe("<p>&lt;https://auto.test/x&gt;</p>");
  expect(await safe("a\0b\n")).toBe("<p>a�b</p>");
});
test("render_text_html preserves native output as an ordinary string", async () => {
  const out = value(await md.renderTextHTML("# Hi <b>raw</b>\n\n[bad](javascript:alert(1))\n"));
  expect(typeof out).toBe("string");
  expect(out).toBe('<h1>Hi <b>raw</b></h1>\n<p><a href="javascript:alert(1)">bad</a></p>\n');
  expect(isHTMLValue("safe", out)).toBe(false);
});
test("budgets cap input, output and node count", async () => {
  const big = "x".repeat(8_388_608 + 1);
  const over = await md.renderSafe(big);
  check(over, 1349);
  expect(failPayload(over)).toMatchObject({ limit: 8388608n, size: 8388609n });
  const overText = await md.renderTextHTML(big);
  check(overText, 1349);
  const fat = "x".repeat(8_388_608);
  const outOver = await md.renderSafe(fat);
  check(outOver, 1349);
  expect(failPayload(outOver)).toMatchObject({ limit: 8388608n, size: 8388615n });
  const many = "# a\n".repeat(1_000_001);
  const nodeOver = await md.renderSafe(many);
  check(nodeOver, 1349);
  expect(failPayload(nodeOver)).toMatchObject({ limit: 1000000n, size: 1000000n });
  const ok = value(await md.renderTextHTML("# a\n"));
  expect(ok).toBe("<h1>a</h1>\n");
});
test("non-string sources reject without parsing", async () => {
  for (const bad of [1, null, undefined, {}]) {
    try {
      await md.renderSafe(bad as never);
      expect(true).toBe(false);
    } catch (error) {
      expect(error).toBeInstanceOf(TypeError);
    }
    try {
      await md.renderTextHTML(bad as never);
      expect(true).toBe(false);
    } catch (error) {
      expect(error).toBeInstanceOf(TypeError);
    }
  }
});
