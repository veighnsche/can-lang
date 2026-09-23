// B1-12 native Markdown qualification on the pinned Bun 1.4.2 binary.
// Run: bun docs/bun-integration/asap/evidence/markdown-native-probe.ts
// Rows pin the string renderer, the render callback protocol (names,
// payloads, composition order), the list/table flattening gate, raw
// HTML routing, href rawness, NUL handling, throw propagation and the
// ignored-unknown policy.
const M = (Bun as any).markdown;
if (Bun.version !== "1.4.2") throw new Error(`pinned Bun 1.4.2 required, saw ${Bun.version}`);
if (typeof M?.html !== "function" || typeof M?.render !== "function") throw new Error("missing native markdown");
type Row = { status: "ok"; value: unknown } | { status: "error"; name: string; message: string };
const rows: Record<string, Row> = {};
const attempt = (k: string, f: () => unknown) => {
  try { rows[k] = { status: "ok", value: f() }; }
  catch (e: any) { rows[k] = { status: "error", name: e?.name ?? "Error", message: String(e?.message ?? e).slice(0, 200) }; }
};
attempt("surface", () => ({
  keys: Object.getOwnPropertyNames(M).sort(), htmlArity: M.html.length, renderArity: M.render.length,
}));
attempt("string_default", () => M.html("# Hello\n\n<script>alert(1)</script>\n\n[x](javascript:alert(1))"));
attempt("string_nohtml", () => M.html("<b>hello</b>", { noHtmlBlocks: true, noHtmlSpans: true }));
attempt("string_unknown_option", () => M.html("- a", { bogusOption: true }));
attempt("string_full", () => M.html("- a\n- b\n\n| h |\n|---|\n| c |\n"));
attempt("callbacks", () => {
  const calls: unknown[] = [];
  const out = M.render("# T\n\nPara with [lnk](javascript:alert(1)) and ![alt](https://img/x.png) plus `cs`.\n\n```js\nconst x = 1;\n```\n\n> quote\n\n- li1\n\n<div>raw</div>\n", {
    text: (t: string) => t,
    heading: (c: string, m: unknown) => { calls.push(["heading", c, m]); return c; },
    paragraph: (c: string) => { calls.push(["paragraph", c]); return c; },
    blockquote: (c: string) => { calls.push(["blockquote", c]); return c; },
    strong: (c: string) => { calls.push(["strong", c]); return c; },
    emphasis: (c: string) => { calls.push(["emphasis", c]); return c; },
    strikethrough: (c: string) => { calls.push(["strikethrough", c]); return c; },
    codespan: (c: string) => { calls.push(["codespan", c]); return c; },
    code: (c: string, m: unknown) => { calls.push(["code", c, m]); return c; },
    link: (c: string, m: unknown) => { calls.push(["link", c, m]); return c; },
    image: (c: string, m: unknown) => { calls.push(["image", c, m]); return c; },
    list: (c: string, m: unknown) => { calls.push(["list", c, m]); return c; },
    table: (c: string, m: unknown) => { calls.push(["table", c, m]); return c; },
    hr: (c: string) => { calls.push(["hr", c]); return c; },
    html: (c: string) => { calls.push(["html", c]); return c; },
  });
  return { calls, out };
});
attempt("flattening", () => {
  const seen: unknown[] = [];
  const cbs: any = { text: (t: string) => "T[" + t + "]" };
  for (const n of ["list", "table", "listitem", "item", "tablerow", "tablecell", "softbreak", "hardbreak", "linebreak"])
    cbs[n] = (...a: any[]) => { seen.push([n, ...a]); return `<${n}>`; };
  const out = M.render("- one\n- two\n\n| h1 | h2 |\n|---|---|\n| c1 | c2 |\n\nline1\nline2  \nline3\n", cbs);
  return { seen, out };
});
attempt("span_routing", () => {
  const seen: unknown[] = [];
  const out = M.render("para <b>span</b> end\n", {
    html: (...a: unknown[]) => { seen.push(["html", ...a]); return "H"; },
    text: (t: string) => { seen.push(["text", t]); return t; },
  });
  return { seen, out };
});
attempt("href_rawness", () => {
  const hrefs: unknown[] = [];
  M.render("[a](javascript:alert(1)) [b](JaVaScRiPt:x) [c](  https://ok/x  ) [d](./rel) [e](#frag) [f](java&#x09;script:y) [g](data:text/html,x) [t](https://u \"ti\")", {
    link: (c: string, m: unknown) => { hrefs.push(m); return c; }, text: (t: string) => t,
  });
  return hrefs;
});
attempt("meta_edges", () => {
  const seen: unknown[] = [];
  M.render("####### h7\n\n```\nplain\n```\n\n```js extra\nc\n```\n\n- a\n  - b\n", {
    heading: (c: string, m: unknown) => { seen.push(["heading", m]); return c; },
    code: (c: string, m: unknown) => { seen.push(["code", m]); return c; },
    list: (c: string, m: unknown) => { seen.push(["list", c, m]); return c; },
    text: (t: string) => t,
  });
  return seen;
});
attempt("throw_propagates", () => M.render("# x", { heading: () => { throw new Error("cb-boom"); }, text: (t: string) => t }));
attempt("unknown_callback_ignored", () => M.render("# x", { nope: () => "X", text: (t: string) => t }));
attempt("nul_text", () => {
  const got: unknown[] = [];
  const out = M.render("a\0b", { text: (t: string) => { got.push(t); return "[" + t + "]"; } });
  return { got, out, html: M.html("a\0b") };
});
attempt("empty", () => ({ html: M.html(""), render: M.render("", { text: (t: string) => t }) }));
attempt("no_gfm_extras", () => {
  const seen: unknown[] = [];
  const cbs: any = { text: (t: string) => t };
  for (const n of ["footnote", "math", "alert", "task", "tasklist", "del", "sub", "sup", "mark"])
    cbs[n] = (...a: any[]) => { seen.push([n, ...a]); return a[0] ?? ""; };
  const out = M.render("[^1] note\n\n[^1]: foot\n\n$math$ and - [ ] task\n", cbs);
  return { seen, out };
});
attempt("escaping", () => {
  const got: unknown[] = [];
  M.render("a < b & \"q\"\n", { text: (t: string) => { got.push(t); return t; } });
  return got;
});
console.log(JSON.stringify({ bun: Bun.version, rows }, null, 1));
