import { success, failure, type AssertionContext } from "../completion.ts";
import { record, dataArray } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
const origin = Object.freeze({
  source: "can:html",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
type Node = Readonly<{ html: string; tag: string; head: boolean; anchor: boolean; form: boolean }>;
type Attribute = Readonly<{ name: string; value: string; kind: "text" | "url" | "htmx" }>;
type Address = Readonly<{ value: string; local: boolean }>;
// One compiler-derived swap policy: an HTML action's method plus its path
// segments ("{}" marks a capture) and the declared non-2xx swap cases.
// The emitter supplies the per-program table; authors cannot forge it.
export type SwapCase = Readonly<{ status: number; swap: "inner" | "outer" }>;
export type SwapPolicy = Readonly<{
  method: "GET" | "POST";
  segments: readonly string[];
  cases: readonly SwapCase[];
}>;
const nodes = new WeakMap<object, Node>(),
  safe = new WeakMap<object, string>(),
  tags = new WeakMap<object, string>(),
  attributes = new WeakMap<object, Attribute>(),
  urls = new WeakMap<object, Address>(),
  targets = new WeakMap<object, string>();
function token<T>(map: WeakMap<object, T>, data: T): unknown {
  const value = Object.freeze(Object.create(null));
  map.set(value, data);
  return value;
}
function read<T>(map: WeakMap<object, T>, value: unknown): T {
  if (
    value === null ||
    (typeof value !== "object" && typeof value !== "function") ||
    !map.has(value)
  )
    throw resourceStateFailure(undefined, origin);
  return map.get(value)!;
}
export function renderSafe(value: unknown): string {
  return read(safe, value);
}
// Portable HTML escaping, byte-identical to the native escaper on every
// input: exactly &<>"' map to their entities and all other code units,
// including astral pairs and lone surrogates, pass through untouched.
// Both profiles share this one implementation.
const htmlEscapes: Readonly<Record<string, string>> = Object.freeze({
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  '"': "&quot;",
  "'": "&#x27;",
});
export function escapeHTML(value: string): string {
  return value.replace(/[&<>"']/g, (char) => htmlEscapes[char] ?? char);
}
export function isHTMLValue(kind: string | undefined, value: unknown): boolean {
  const map =
    kind === "node"
      ? nodes
      : kind === "safe"
        ? safe
        : kind === "url"
          ? urls
          : kind === "tag"
            ? tags
            : kind === "attribute"
              ? attributes
              : kind === "target"
                ? targets
                : undefined;
  return (
    map !== undefined &&
    value !== null &&
    (typeof value === "object" || typeof value === "function") &&
    map.has(value)
  );
}
function string(value: string): string {
  if (typeof value !== "string") throw new TypeError("invalid HTML string");
  return value;
}
const lower = (value: string) => string(value).replace(/[A-Z]/g, (c) => c.toLowerCase());
// Shared with the live-DOM browser catalogue (platform/browser.ts): the
// author tag and attribute vocabularies are one definition, and
// runtime/test/browser-names.json pins their effective admission.
export const authorTags = new Set(
  "main header footer nav section article aside h1 h2 h3 h4 h5 h6 p div span ul ol li a form label input textarea select option button table thead tbody tr th td dl dt dd strong em small br hr code pre blockquote img del".split(
    " ",
  ),
);
const voidTags = new Set(["input", "br", "hr", "img"]);
export const globals = new Set("id class title lang dir hidden tabindex role".split(" "));
export const applicability: Readonly<Record<string, readonly string[]>> = Object.freeze({
  name: ["form", "input", "textarea", "select", "button"],
  value: ["input", "option", "button", "li"],
  type: ["input", "button", "a", "ol"],
  placeholder: ["input", "textarea"],
  autocomplete: ["form", "input", "textarea", "select"],
  for: ["label"],
  method: ["form"],
  rel: ["a", "form"],
  checked: ["input"],
  selected: ["option"],
  disabled: ["input", "textarea", "select", "option", "button"],
  required: ["input", "textarea", "select"],
  multiple: ["input", "select"],
  rows: ["textarea"],
  cols: ["textarea"],
  scope: ["th"],
  colspan: ["td", "th"],
  rowspan: ["td", "th"],
  alt: ["img"],
  align: ["td", "th"],
  start: ["ol"],
});
const inputTypes = new Set(
  "hidden text search tel url email password date month week time datetime-local number range color checkbox radio file submit image reset button".split(
    " ",
  ),
);
const relations = new Set(
  "alternate author bookmark external help license next nofollow noopener noreferrer opener prev privacy-policy search tag terms-of-service".split(
    " ",
  ),
);
const autocompleteFields = new Set(
  "name honorific-prefix given-name additional-name family-name honorific-suffix nickname username new-password current-password one-time-code organization-title organization street-address address-line1 address-line2 address-line3 address-level4 address-level3 address-level2 address-level1 country country-name postal-code cc-name cc-given-name cc-additional-name cc-family-name cc-number cc-exp cc-exp-month cc-exp-year cc-csc cc-type transaction-currency transaction-amount language bday bday-day bday-month bday-year sex url photo tel tel-country-code tel-national tel-area-code tel-local tel-local-prefix tel-local-suffix tel-extension email impp".split(
    " ",
  ),
);
function autocomplete(value: string): boolean {
  const parts = lower(value)
    .replace(/^[\t\n\f\r ]+|[\t\n\f\r ]+$/g, "")
    .split(/[\t\n\f\r ]+/);
  if (parts.length === 1 && (parts[0] === "on" || parts[0] === "off")) return true;
  if (parts[0]?.startsWith("section-") && parts[0].length > 8) parts.shift();
  if (parts[0] === "shipping" || parts[0] === "billing") parts.shift();
  if (["home", "work", "mobile", "fax", "pager"].includes(parts[0] ?? "")) {
    parts.shift();
    if (!/^(tel(?:-[a-z-]+)?|email|impp)$/.test(parts[0] ?? "")) return false;
  }
  if (!autocompleteFields.has(parts.shift() ?? "")) return false;
  if (parts[0] === "webauthn") parts.shift();
  return parts.length === 0;
}
function validValue(name: string, value: string, tag?: string): boolean {
  const v = lower(value);
  if (name === "dir") return ["ltr", "rtl", "auto"].includes(v);
  if (name === "hidden") return ["", "hidden", "until-found"].includes(v);
  if (["checked", "selected", "disabled", "required", "multiple"].includes(name))
    return v === "" || v === name;
  if (name === "method") return ["get", "post", "dialog"].includes(v);
  if (name === "scope") return ["row", "col", "rowgroup", "colgroup"].includes(v);
  if (name === "align") return ["left", "center", "right"].includes(v);
  if (name === "autocomplete")
    return tag === "form" ? ["on", "off"].includes(v) : autocomplete(value);
  if (name === "rel") {
    const parts = v.replace(/^[\t\n\f\r ]+|[\t\n\f\r ]+$/g, "").split(/[\t\n\f\r ]+/);
    return (
      parts.length > 0 &&
      parts.every(
        (p) =>
          relations.has(p) &&
          !(
            tag === "form" &&
            [
              "alternate",
              "author",
              "bookmark",
              "privacy-policy",
              "tag",
              "terms-of-service",
            ].includes(p)
          ),
      )
    );
  }
  if (name === "type" && tag !== undefined) {
    if (tag === "input") return inputTypes.has(v);
    if (tag === "button") return ["submit", "reset", "button"].includes(v);
    if (tag === "ol") return ["1", "a", "A", "i", "I"].includes(value);
  }
  if (["rows", "cols", "colspan", "rowspan"].includes(name)) {
    if (!/^[0-9]+$/.test(value)) return false;
    const n = BigInt(value);
    return name === "rowspan" ? n <= 65534n : n >= 1n && (name !== "colspan" || n <= 1000n);
  }
  if (name === "tabindex" || name === "start" || (name === "value" && tag === "li"))
    return /^-?[0-9]+$/.test(value);
  return true;
}
const attr = (name: string, value: string, kind: Attribute["kind"] = "htmx") =>
  token(attributes, Object.freeze({ name, value, kind }));
const node = (html: string, tag = "", head = false, anchor = false, form = false) =>
  token(nodes, Object.freeze({ html, tag, head, anchor, form }));
const children = (input: readonly unknown[]) => dataArray(input).map((v) => read(nodes, v));
const serialize = (a: Attribute) => ` ${a.name}="${escapeHTML(a.value)}"`;
const selectorID = (value: string) => /^[A-Za-z_][A-Za-z0-9_-]*$/.test(string(value));
// Validate the emitter-supplied swap table once at factory time. Malformed
// entries fail fast: only generated code provides this table.
function readSwapTable(input: readonly unknown[]): readonly SwapPolicy[] {
  if (!Array.isArray(input)) throw new TypeError("invalid swap policy table");
  return Object.freeze(
    input.map((entry) => {
      if (typeof entry !== "object" || entry === null)
        throw new TypeError("invalid swap policy entry");
      const { method, segments, cases } = entry as Record<string, unknown>;
      if (method !== "GET" && method !== "POST") throw new TypeError("invalid swap policy method");
      if (
        !Array.isArray(segments) ||
        segments.length === 0 ||
        segments.some(
          (segment) => typeof segment !== "string" || segment === "" || segment.includes("/"),
        )
      )
        throw new TypeError("invalid swap policy path");
      if (!Array.isArray(cases)) throw new TypeError("invalid swap policy cases");
      const frozen = cases.map((item) => {
        if (typeof item !== "object" || item === null)
          throw new TypeError("invalid swap policy case");
        const { status, swap } = item as Record<string, unknown>;
        if (typeof status !== "number" || !Number.isInteger(status) || status < 200 || status > 599)
          throw new TypeError("invalid swap policy status");
        if (swap !== "inner" && swap !== "outer") throw new TypeError("invalid swap policy style");
        return Object.freeze({ status, swap });
      });
      return Object.freeze({
        method,
        segments: Object.freeze([...(segments as string[])]),
        cases: Object.freeze(frozen),
      });
    }),
  );
}
// Derive hx-status swap exceptions for htmx verbs bound to checked HTML
// actions. Each hx-post/hx-get URL is matched structurally against the
// compiler table (static segments byte-equal after decoding, captures match
// any nonempty segment); matching entries union their declared non-2xx swap
// cases, first style wins per status, and output sorts ascending so renders
// are deterministic. 2xx cases swap under the global policy already, and
// unmatched URLs serialize exactly as before.
function deriveSwapAttributes(
  policies: readonly SwapPolicy[],
  attrs: readonly Attribute[],
): Attribute[] {
  if (policies.length === 0) return [];
  const merged = new Map<number, "inner" | "outer">();
  for (const attr of attrs) {
    if (attr.kind !== "htmx" || (attr.name !== "hx-post" && attr.name !== "hx-get")) continue;
    const verb = attr.name === "hx-post" ? "POST" : "GET";
    const path = attr.value.split(/[?#]/)[0] ?? "";
    if (!path.startsWith("/")) continue;
    let decoded: string[];
    try {
      decoded = path
        .split("/")
        .slice(1)
        .map((segment) => decodeURIComponent(segment));
    } catch {
      continue;
    }
    for (const policy of policies) {
      if (policy.method !== verb || policy.segments.length !== decoded.length) continue;
      let hit = true;
      for (let index = 0; index < decoded.length; index++) {
        const want = policy.segments[index] ?? "";
        const got = decoded[index] ?? "";
        if (want === "{}" ? got === "" : want !== got) {
          hit = false;
          break;
        }
      }
      if (!hit) continue;
      for (const item of policy.cases) {
        if (item.status >= 200 && item.status <= 299) continue;
        if (!merged.has(item.status)) merged.set(item.status, item.swap);
      }
    }
  }
  return [...merged.entries()]
    .sort(([left], [right]) => left - right)
    .map(([status, swap]) =>
      Object.freeze({
        name: `hx-status:${status}`,
        value: `{"swap":"${swap === "outer" ? "outerHTML" : "innerHTML"}"}`,
        kind: "htmx" as const,
      }),
    );
}
type Contracts = Readonly<{ structure: string; url: string; target: string; interval: string }>;
export function createHTML(
  domain: ReturnType<typeof createDomainRuntime>,
  types: Contracts,
  declared: readonly string[] = [],
  swaps: readonly unknown[] = [],
) {
  const policies = readSwapTable(swaps);
  const bad = (identity: string, reason: string) =>
    failure(domain.create(identity, record(identity, [["reason", reason]]), origin));
  const structure = (reason: string) => bad(types.structure, reason);
  const interval = (n: bigint) =>
    failure(domain.create(types.interval, record(types.interval, [["milliseconds", n]]), origin));
  const local = (input: unknown, name: string) => {
    const url = read(urls, input);
    return url.local ? success(attr(name, url.value)) : bad(types.url, "same_origin");
  };
  const declaredURLs = new Set(declared);
  return Object.freeze({
    async makeTag(name: string, _context?: AssertionContext) {
      name = lower(name);
      return authorTags.has(name) ? success(token(tags, name)) : structure("tag");
    },
    async text(value: string, _context?: AssertionContext) {
      return success(node(escapeHTML(string(value))));
    },
    async textFragment(value: string, _context?: AssertionContext) {
      return success(token(safe, escapeHTML(string(value))));
    },
    async parseURL(value: string, _context?: AssertionContext) {
      // oxlint-disable no-control-regex -- Reject ASCII controls, DEL, and backslash before URL parsing.
      string(value);
      if (!value.isWellFormed() || /[\x00-\x1f\x7f\\]/.test(value) || value.startsWith("//"))
        return bad(types.url, "syntax");
      // oxlint-enable no-control-regex
      const local = value.startsWith("/");
      if (!local && !/^https:\/\//i.test(value)) return bad(types.url, "scheme");
      let parsed: URL;
      try {
        parsed = new URL(value, "https://can.invalid/");
      } catch (cause) {
        if (!(cause instanceof TypeError)) throw cause;
        return bad(types.url, "syntax");
      }
      if (parsed.protocol !== "https:" || parsed.username !== "" || parsed.password !== "")
        return bad(types.url, "authority");
      const output = local ? parsed.pathname + parsed.search + parsed.hash : parsed.href;
      if (local && (parsed.origin !== "https://can.invalid" || output.startsWith("//")))
        return bad(types.url, "same_origin");
      return success(token(urls, Object.freeze({ value: output, local })));
    },
    async textAttribute(name: string, value: string, _context?: AssertionContext) {
      name = lower(name);
      string(value);
      if (
        !globals.has(name) &&
        !/^aria-[a-z][a-z0-9-]*$/.test(name) &&
        !Object.hasOwn(applicability, name)
      )
        return structure("attribute");
      if (!validValue(name, value)) return structure("attribute_value");
      return success(attr(name, value, "text"));
    },
    async urlAttribute(name: string, url: unknown, _context?: AssertionContext) {
      name = lower(name);
      const value = read(urls, url);
      return ["href", "action", "formaction", "src"].includes(name)
        ? success(attr(name, value.value, "url"))
        : structure("attribute");
    },
    async element(
      tag: unknown,
      inputAttributes: readonly unknown[],
      inputChildren: readonly unknown[],
      _context?: AssertionContext,
    ) {
      const name = read(tags, tag),
        attrs = dataArray(inputAttributes).map((v) => read(attributes, v)),
        kids = children(inputChildren),
        seen = new Set<string>();
      for (const a of attrs) {
        if (seen.has(a.name)) return structure("duplicate_attribute");
        seen.add(a.name);
        if (
          a.kind === "text" &&
          Object.hasOwn(applicability, a.name) &&
          !applicability[a.name]!.includes(name)
        )
          return structure("attribute_tag");
        if (a.kind === "text" && !validValue(a.name, a.value, name))
          return structure("attribute_value");
        if (
          a.kind === "url" &&
          (
            { href: "a", action: "form", formaction: "button", src: "img" } as Record<
              string,
              string
            >
          )[a.name] !== name
        )
          return structure("attribute_tag");
      }
      if (kids.some((k) => k.head)) return structure("head_context");
      if (voidTags.has(name) && kids.length !== 0) return structure("void_children");
      const required: Readonly<Record<string, readonly string[]>> = {
        ul: ["li"],
        ol: ["li"],
        select: ["option"],
        thead: ["tr"],
        tbody: ["tr"],
        tr: ["th", "td"],
      };
      if (Object.hasOwn(required, name) && kids.some((k) => !required[name]!.includes(k.tag)))
        return structure("children");
      if (
        name === "table" &&
        !(
          (kids.length === 1 && kids[0]!.tag === "tbody") ||
          (kids.length === 2 && kids[0]!.tag === "thead" && kids[1]!.tag === "tbody")
        )
      )
        return structure("children");
      if (
        name === "dl" &&
        (kids.length % 2 !== 0 || kids.some((k, i) => k.tag !== (i % 2 === 0 ? "dt" : "dd")))
      )
        return structure("children");
      if (
        (name === "a" && kids.some((k) => k.anchor)) ||
        (name === "form" && kids.some((k) => k.form))
      )
        return structure("nested_element");
      const output =
        `<${name}${[...attrs, ...deriveSwapAttributes(policies, attrs)].map(serialize).join("")}>` +
        (voidTags.has(name) ? "" : kids.map((k) => k.html).join("") + `</${name}>`);
      return success(
        node(
          output,
          name,
          false,
          name === "a" || kids.some((k) => k.anchor),
          name === "form" || kids.some((k) => k.form),
        ),
      );
    },
    async fragment(input: readonly unknown[], _context?: AssertionContext) {
      const kids = children(input);
      return kids.some((k) => k.head)
        ? structure("head_context")
        : success(token(safe, kids.map((k) => k.html).join("")));
    },
    async stylesheet(input: unknown, _context?: AssertionContext) {
      const url = read(urls, input);
      return success(node(`<link rel="stylesheet" href="${escapeHTML(url.value)}">`, "link", true));
    },
    async metaViewport(_context?: AssertionContext) {
      return success(
        node('<meta name="viewport" content="width=device-width, initial-scale=1">', "meta", true),
      );
    },
    async document(
      title: string,
      head: readonly unknown[],
      body: readonly unknown[],
      _context?: AssertionContext,
    ) {
      const h = children(head),
        b = children(body);
      if (h.some((n) => !n.head) || b.some((n) => n.head)) return structure("document_context");
      return success(
        token(
          safe,
          `<!doctype html><html><head><title>${escapeHTML(string(title))}</title>${h.map((n) => n.html).join("")}</head><body>${b.map((n) => n.html).join("")}</body></html>`,
        ),
      );
    },
    async get(url: unknown, _context?: AssertionContext) {
      return local(url, "hx-get");
    },
    async post(url: unknown, _context?: AssertionContext) {
      return local(url, "hx-post");
    },
    async targetID(id: string, _context?: AssertionContext) {
      return selectorID(id) ? success(token(targets, "#" + id)) : bad(types.target, "id");
    },
    async targetAttribute(target: unknown, _context?: AssertionContext) {
      return success(attr("hx-target", read(targets, target)));
    },
    async indicatorID(id: string, _context?: AssertionContext) {
      return selectorID(id) ? success(attr("hx-indicator", "#" + id)) : bad(types.target, "id");
    },
    async swapInner(_context?: AssertionContext) {
      return success(attr("hx-swap", "innerHTML"));
    },
    async swapOuter(_context?: AssertionContext) {
      return success(attr("hx-swap", "outerHTML"));
    },
    async triggerChange(_context?: AssertionContext) {
      return success(attr("hx-trigger", "change"));
    },
    async triggerInputChanged(delay: bigint, _context?: AssertionContext) {
      return delay < 0n || delay > 60000n
        ? interval(delay)
        : success(attr("hx-trigger", `input changed delay:${delay}ms`));
    },
    async triggerEvery(period: bigint, _context?: AssertionContext) {
      return period < 1000n || period > 3600000n
        ? interval(period)
        : success(attr("hx-trigger", `every ${period}ms`));
    },
    async disableThis(_context?: AssertionContext) {
      return success(attr("hx-disable", "this"));
    },
    async declareAsset(url: string, _context?: AssertionContext) {
      if (typeof url !== "string" || !declaredURLs.has(url))
        throw new TypeError("undeclared asset url");
      return success(token(urls, Object.freeze({ value: url, local: true })));
    },
    async rejectAsset(reason: string, _context?: AssertionContext) {
      if (reason !== "missing" && reason !== "unowned") throw new TypeError("invalid asset reason");
      return bad(types.url, reason);
    },
    async runtimeHead(_context?: AssertionContext) {
      // htmx 4 swaps every status except noSwap entries. The compiler-owned
      // policy keeps 204/304 quiet and every 4xx/5xx out of swaps; exact
      // hx-status attributes generated only from a checked HTML action's
      // cases re-admit declared error statuses per source element, with
      // same-origin fetch pinned explicitly. The owned guard module loads
      // after htmx and enforces target identity, response-control, and
      // task-shape policy. Its integrity pins the Bun-transpiled bytes of
      // runtime/platform/htmx-guard.ts; re-pin after any guard edit.
      const noSwap = [204, 304, "4xx", "5xx"];
      const config = JSON.stringify({ mode: "same-origin", noSwap });
      return success(
        node(
          `<meta name="htmx-config" content="${escapeHTML(config)}"><script defer src="/__can/assets/htmx-4.0.0.min.js" integrity="sha384-BvJpBiO8Kh31EqtJe5DRIeWrHWnCGkwytKs9NKFi86Hhw96dEqdEMzZDeK9iEGTc"></script><script type="module" src="/__can/assets/htmx-guard.js" integrity="sha384-mr/IRfJgLjok38ftBi21o/T8c9cnFZrvEKtiwVjIOFAlo3Z7h1rGMYsWvebDJ8kG"></script>`,
          "runtime",
          true,
        ),
      );
    },
  });
}
