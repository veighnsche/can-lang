import { success, failure, type Completion, type AssertionContext } from "../completion.ts";
import { createDomainRuntime } from "../domain.ts";
import { record } from "../data.ts";
import { maxNodes, standaloneBytes } from "../codec/budget.ts";
import { createHTML, renderSafe as renderSafeHTML } from "./html.ts";

type Ids = Readonly<{ overLimit: string; htmlStructure: string; htmlURL: string }>;
const origin = Object.freeze({
  source: "can:markdown",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
const encoder = new TextEncoder();
const languageClass = /^[A-Za-z0-9_-]+$/;
const tokenOf = (id: number) => `\0${id}\0`;
const aligns = new Set(["left", "center", "right"]);

// Captured element trees. Parts mix raw text segments with references to
// earlier nodes; children always arrive before parents, so a single
// ascending build pass suffices.
type Attr = Readonly<
  { kind: "text"; name: string; value: string } | { kind: "url"; name: string; value: string }
>;
type Tree = Readonly<{ tag: string; attrs: readonly Attr[]; parts: readonly (string | number)[] }>;
type NativeCallbacks = NonNullable<Parameters<typeof Bun.markdown.render>[1]>;

export function createMarkdown(domain: ReturnType<typeof createDomainRuntime>, ids: Ids) {
  const html = createHTML(domain, {
    structure: ids.htmlStructure,
    url: ids.htmlURL,
    target: "",
    interval: "",
  });
  const fail = (identity: string, fields: readonly (readonly [string, unknown])[]) =>
    failure(domain.create(identity, record(identity, fields), origin));
  const over = (limit: number, size: number) =>
    fail(ids.overLimit, [
      ["limit", BigInt(limit)],
      ["size", BigInt(size)],
    ]);
  const invariant = (what: string): never => {
    throw new TypeError(`unreachable markdown ${what}`);
  };
  return Object.freeze({
    async renderTextHTML(
      source: unknown,
      _context?: AssertionContext,
    ): Promise<Completion<string>> {
      if (typeof source !== "string") throw new TypeError("invalid markdown source");
      const size = encoder.encode(source).length;
      if (size > standaloneBytes) return over(standaloneBytes, size);
      // Default native options preserve raw HTML by contract: the result
      // stays an ordinary str and the type system keeps it out of
      // html::safe and trusted responses.
      const rendered = Bun.markdown.html(source);
      const out = encoder.encode(rendered).length;
      if (out > standaloneBytes) return over(standaloneBytes, out);
      return success(rendered);
    },
    async renderSafe(source: unknown, _context?: AssertionContext): Promise<Completion<unknown>> {
      if (typeof source !== "string") throw new TypeError("invalid markdown source");
      const size = encoder.encode(source).length;
      if (size > standaloneBytes) return over(standaloneBytes, size);
      const trees: Tree[] = [];
      let capped = false;
      const split = (children: string): (string | number)[] => {
        if (!children.includes("\0")) return children === "" ? [] : [children];
        const out: (string | number)[] = [];
        const cells = children.split(/\0(\d+)\0/);
        for (let i = 0; i < cells.length; i += 2) {
          const text = cells[i]!;
          if (text !== "") {
            if (text.includes("\0")) invariant("nul");
            out.push(text);
          }
          if (i + 1 < cells.length) {
            const id = Number(cells[i + 1]);
            if (!Number.isSafeInteger(id) || id < 0 || id >= trees.length) invariant("token");
            out.push(id);
          }
        }
        return out;
      };
      const emit = (tag: string, attrs: readonly Attr[], children: string): string => {
        if (capped) return "";
        if (trees.length >= maxNodes) {
          capped = true;
          return "";
        }
        trees.push({ tag, attrs, parts: split(children) });
        return tokenOf(trees.length - 1);
      };
      // Plain-text extraction for image alt attributes. References only
      // point at earlier trees, so iterative pre-order terminates.
      const textOf = (id: number): string => {
        let out = "";
        const stack = [id];
        while (stack.length > 0) {
          const next = stack.pop()!;
          if (next < 0 || next >= trees.length) invariant("alt");
          const parts = trees[next]!.parts;
          for (let i = parts.length - 1; i >= 0; i--) {
            const part = parts[i]!;
            if (typeof part === "string") out += part;
            else stack.push(part);
          }
        }
        return out;
      };
      const altText = (children: string): string => {
        if (!children.includes("\0")) return children;
        let out = "";
        for (const part of split(children)) out += typeof part === "string" ? part : textOf(part);
        return out;
      };
      const cell = (tag: string, children: string, align: unknown): string => {
        if (align !== undefined && (typeof align !== "string" || !aligns.has(align.toLowerCase())))
          invariant("align");
        const attrs: Attr[] =
          typeof align === "string" ? [{ kind: "text", name: "align", value: align }] : [];
        return emit(tag, attrs, children);
      };
      // Phase 1 captures structure synchronously. Every callback returns
      // either raw text (escaped exactly once in phase 2) or an
      // unforgeable token; the native layer never emits NUL in text, and
      // any violation fails closed here. In-paragraph newlines pass
      // through untouched: soft and hard breaks normalize to soft.
      const text = (value: string): string => {
        if (value.includes("\0")) invariant("text");
        return value;
      };
      const callbacks: NativeCallbacks = {
        text,
        heading: (children, meta) => {
          if (!Number.isInteger(meta.level) || meta.level < 1 || meta.level > 6)
            invariant("heading");
          const id = meta.id;
          if (id !== undefined && typeof id !== "string") invariant("heading-id");
          const attrs: Attr[] = id === undefined ? [] : [{ kind: "text", name: "id", value: id }];
          return emit(`h${meta.level}`, attrs, children);
        },
        paragraph: (children) => emit("p", [], children),
        blockquote: (children) => emit("blockquote", [], children),
        strong: (children) => emit("strong", [], children),
        emphasis: (children) => emit("em", [], children),
        strikethrough: (children) => emit("del", [], children),
        codespan: (children) => emit("code", [], children),
        code: (children, meta) => {
          const language = meta?.language;
          if (language !== undefined && typeof language !== "string") invariant("code-language");
          const attrs: Attr[] =
            language !== undefined && language !== "" && languageClass.test(language)
              ? [{ kind: "text", name: "class", value: `language-${language}` }]
              : [];
          return emit("pre", [], emit("code", attrs, children));
        },
        link: (children, meta) => {
          if (typeof meta.href !== "string") invariant("link-href");
          const title = meta.title;
          if (title !== undefined && typeof title !== "string") invariant("link-title");
          const attrs: Attr[] = [{ kind: "url", name: "href", value: meta.href }];
          if (title !== undefined) attrs.push({ kind: "text", name: "title", value: title });
          return emit("a", attrs, children);
        },
        image: (children, meta) => {
          if (typeof meta.src !== "string") invariant("image-src");
          const title = meta.title;
          if (title !== undefined && typeof title !== "string") invariant("image-title");
          const attrs: Attr[] = [
            { kind: "url", name: "src", value: meta.src },
            { kind: "text", name: "alt", value: altText(children) },
          ];
          if (title !== undefined) attrs.push({ kind: "text", name: "title", value: title });
          return emit("img", attrs, "");
        },
        hr: () => emit("hr", [], ""),
        // Unreachable with noHtmlBlocks/noHtmlSpans: the parser demotes raw
        // markup to text callbacks. Escape closed if that ever changes.
        html: (children) => emit("p", [], children),
        list: (children, meta) => {
          if (typeof meta.ordered !== "boolean") invariant("list-ordered");
          if (!meta.ordered) return emit("ul", [], children);
          const start = meta.start;
          const attrs: Attr[] =
            typeof start === "number" && Number.isSafeInteger(start) && start >= 0 && start !== 1
              ? [{ kind: "text", name: "start", value: String(start) }]
              : [];
          return emit("ol", attrs, children);
        },
        listItem: (children, meta) => {
          const checked = meta.checked;
          if (checked !== undefined && typeof checked !== "boolean") invariant("task");
          if (checked === undefined) return emit("li", [], children);
          const box = emit(
            "input",
            checked
              ? [
                  { kind: "text", name: "type", value: "checkbox" },
                  { kind: "text", name: "disabled", value: "disabled" },
                  { kind: "text", name: "checked", value: "checked" },
                ]
              : [
                  { kind: "text", name: "type", value: "checkbox" },
                  { kind: "text", name: "disabled", value: "disabled" },
                ],
            "",
          );
          return emit("li", [], box + children);
        },
        table: (children) => {
          const parts = split(children);
          let head = "",
            body = "";
          for (const part of parts) {
            if (typeof part === "string") invariant("table-text");
            else {
              const tag = trees[part]!.tag;
              if (tag === "thead") {
                if (head !== "") invariant("table");
                head = tokenOf(part);
              } else if (tag === "tbody") {
                if (body !== "") invariant("table");
                body = tokenOf(part);
              } else invariant("table");
            }
          }
          if (head === "") invariant("table");
          // A header-only table fires no tbody callback; the gate requires
          // thead plus tbody, so synthesize the empty body.
          if (body === "") body = emit("tbody", [], "");
          return emit("table", [], head + body);
        },
        thead: (children) => emit("thead", [], children),
        tbody: (children) => emit("tbody", [], children),
        tr: (children) => emit("tr", [], children),
        th: (children, meta) => cell("th", children, meta?.align),
        td: (children, meta) => cell("td", children, meta?.align),
      };
      const rendered = Bun.markdown.render(source, callbacks, {
        tables: true,
        strikethrough: true,
        tasklists: true,
        headings: { ids: true },
        noHtmlBlocks: true,
        noHtmlSpans: true,
        wikiLinks: false,
        underline: false,
        latexMath: false,
        autolinks: false,
      });
      if (capped) return over(maxNodes, trees.length);
      // Phase 2 rebuilds trusted nodes bottom-up through the html
      // factory. URL rejections propagate as html::invalid_url; every
      // other factory failure is unreachable by construction.
      const nodes: unknown[] = [];
      for (let i = 0; i < trees.length; i++) {
        const tree = trees[i]!;
        const kids: unknown[] = [];
        for (const part of tree.parts) {
          if (typeof part === "string") {
            const made = await html.text(part);
            if (made.kind !== "ok") invariant("text-node");
            kids.push(made.value);
          } else {
            if (part < 0 || part >= i) invariant("order");
            kids.push(nodes[part]);
          }
        }
        const attrs: unknown[] = [];
        for (const attr of tree.attrs) {
          if (attr.kind === "text") {
            const made = await html.textAttribute(attr.name, attr.value);
            if (made.kind !== "ok") invariant("attribute");
            attrs.push(made.value);
          } else {
            const parsed = await html.parseURL(attr.value);
            if (parsed.kind !== "ok") return parsed;
            const made = await html.urlAttribute(attr.name, parsed.value);
            if (made.kind !== "ok") invariant("url-attribute");
            attrs.push(made.value);
          }
        }
        const tag = await html.makeTag(tree.tag);
        if (tag.kind !== "ok") invariant("tag");
        const made = await html.element(tag.value, attrs, kids);
        if (made.kind !== "ok") invariant("element");
        nodes.push(made.value);
      }
      const top: unknown[] = [];
      for (const part of split(rendered)) {
        if (typeof part === "string") {
          const made = await html.text(part);
          if (made.kind !== "ok") invariant("top-text");
          top.push(made.value);
        } else {
          if (part < 0 || part >= nodes.length) invariant("top-order");
          top.push(nodes[part]);
        }
      }
      const frag = await html.fragment(top);
      if (frag.kind !== "ok") invariant("fragment");
      const out = renderSafeHTML(frag.value);
      const bytes = encoder.encode(out).length;
      if (bytes > standaloneBytes) return over(standaloneBytes, bytes);
      return success(frag.value);
    },
  });
}
