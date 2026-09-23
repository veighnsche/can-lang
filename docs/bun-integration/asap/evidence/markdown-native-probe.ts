// Native probe: Bun.markdown callback model for B1-12 safe renderer design.
//
// Run: bun docs/bun-integration/asap/evidence/markdown-native-probe.ts
// Pins: Bun 1.4.2. Records render()/html() behavior the safe renderer relies on.

const seen: Array<[string, string, string]> = [];
const cb = (n: string) => (c: string, m?: unknown) => {
  seen.push([n, JSON.stringify(c).slice(0, 100), m === undefined ? "" : JSON.stringify(m)]);
  return n + "{" + c + "}";
};
const callbacks = {
  text: (t: string) => "T[" + t + "]",
  heading: cb("h"), paragraph: cb("p"), blockquote: cb("bq"), code: cb("code"),
  list: cb("list"), listItem: cb("li"), hr: cb("hr"), table: cb("table"),
  thead: cb("thead"), tbody: cb("tbody"), tr: cb("tr"),
  th: cb("th"), td: cb("td"), html: cb("html"), strong: cb("st"),
  emphasis: cb("em"), link: cb("a"), image: cb("img"),
  codespan: cb("cs"), strikethrough: cb("strike"),
};

console.log("=== render: lists + tables (camelCase names) ===");
seen.length = 0;
Bun.markdown.render("- one\n- two\n\n1. a\n2. b\n\n- [ ] t1\n- [x] t2\n\n| h1 | h2 |\n|:---|---:|\n| c1 | c2 |\n", callbacks);
for (const s of seen) console.log(" ", s[0], "|", s[1], "|", s[2].slice(0, 140));

console.log("=== render: breaks / inline / hr ===");
seen.length = 0;
Bun.markdown.render("one\ntwo  \nthree\n\n---\n\n*em* **st** ~~d~~ `c` __u__\n\n[t](https://x.test/u \"ti\") ![alt](https://x.test/i.png)\n", callbacks);
for (const s of seen) console.log(" ", s[0], "|", s[1], "|", s[2].slice(0, 140));

console.log("=== html: edge elements ===");
const H = Bun.markdown.html;
console.log("HARD:", JSON.stringify(H("a  \nb\n")));
console.log("SOFT:", JSON.stringify(H("a\nb\n")));
console.log("UNDERLINE-IGNORED:", JSON.stringify(H("__x__\n", { underline: true })));
console.log("TASK:", JSON.stringify(H("- [ ] a\n- [x] b\n")));
console.log("AUTO:", JSON.stringify(H("see https://x.test/a and www.y.test b@c.test\n", { autolinks: true })));
console.log("HTMLBLK:", JSON.stringify(H('<div class="q">raw & <b>b</b></div>\n')));
console.log("TAGF:", JSON.stringify(H("<script>alert(1)</script><div>ok</div>\n", { tagFilter: true })));
console.log("WIKI-HTML:", JSON.stringify(H("[[Tgt|lbl]] [[Bare]]\n", { wikiLinks: true })));
console.log("MATH-IGNORED:", JSON.stringify(H("$x^2$ and $$y$$\n", { latexMath: true })));
console.log("HEAD-IDS:", JSON.stringify(H("## Hello World!\n", { headings: { ids: true } })));
console.log("HEAD-AUTOLINK:", JSON.stringify(H("## Hello\n", { headings: true })));

console.log("=== render: wiki target dropped / code meta / heading id ===");
seen.length = 0;
Bun.markdown.render("[[T| l]]\n\n    ind\n\n```js\nf\n```\n\n## Head One\n", callbacks, { wikiLinks: true, headings: { ids: true } });
for (const s of seen) console.log(" ", s[0], "|", s[1], "|", s[2].slice(0, 140));
