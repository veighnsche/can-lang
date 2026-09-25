import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import {
  createBrowser,
  createBrowserState,
  isBrowserValue,
  isBrowserStateValue,
  type BrowserDocument,
} from "../platform/browser.ts";
import { success, value, type Completion } from "../completion.ts";
import { dataProperty, recordIdentity } from "../data.ts";
// T22 bounded browser catalogue. The element tree is a minimal fake host,
// but listener, timer and disposal semantics run through native Bun
// EventTarget/Event/AbortController/setTimeout behavior: dispatch,
// abort removal and timer clearing are real, not simulated.
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
const declarations = catalogue.errors.filter((e) =>
  [
    "browser::missing_root",
    "browser::disposed",
    "browser::rejected",
    "browser::stale_version",
    "browser::invalid_query",
  ].includes(e.name),
);
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
const contracts = {
  missingRoot: errors[0]!.identity,
  disposed: errors[1]!.identity,
  rejected: errors[2]!.identity,
  event: "test-browser-event",
  invalidQuery: errors[4]!.identity,
  some: "test-browser-some",
  none: "test-browser-none",
};
const stateContracts = {
  disposed: errors[1]!.identity,
  stale: errors[3]!.identity,
  state: "test-browser-state",
  snapshot: "test-browser-snapshot",
};
class FakeNode extends EventTarget {
  parentNode: FakeElement | null = null;
  textContent: string | null = null;
  remove(): void {
    if (this.parentNode) {
      this.parentNode.children = this.parentNode.children.filter((child) => child !== this);
      this.parentNode = null;
    }
  }
  addEventListener(
    type: string,
    listener: (event: Event) => void,
    options?: { signal: AbortSignal },
  ): void {
    super.addEventListener(type, listener, options);
  }
}
class FakeElement extends FakeNode {
  tagName: string;
  id = "";
  attributes = new Map<string, string>();
  children: FakeNode[] = [];
  focused = 0;
  constructor(tag: string) {
    super();
    this.tagName = tag;
  }
  setAttribute(name: string, value: string): void {
    this.attributes.set(name, value);
    if (name === "id") this.id = value;
  }
  removeAttribute(name: string): void {
    this.attributes.delete(name);
    if (name === "id") this.id = "";
  }
  appendChild(node: FakeNode): void {
    node.remove();
    this.children.push(node);
    node.parentNode = this;
  }
  focus(): void {
    this.focused++;
  }
}
class FakeText extends FakeNode {}
class FakeDocument {
  roots: FakeElement[];
  constructor(ids: string[] = ["app"]) {
    this.roots = ids.map((id) => {
      const root = new FakeElement("div");
      root.setAttribute("id", id);
      return root;
    });
  }
  getElementById(id: string): FakeElement | null {
    const visit = (node: FakeNode): FakeElement | null => {
      if (node instanceof FakeElement) {
        if (node.id === id) return node;
        for (const child of node.children) {
          const found = visit(child);
          if (found) return found;
        }
      }
      return null;
    };
    for (const root of this.roots) {
      const found = visit(root);
      if (found) return found;
    }
    return null;
  }
  createElement(tag: string): FakeElement {
    return new FakeElement(tag);
  }
  createTextNode(text: string): FakeText {
    const node = new FakeText();
    node.textContent = text;
    return node;
  }
}
const setup = () => {
  const document = new FakeDocument();
  const browser = createBrowser(domain, contracts, document as unknown as BrowserDocument);
  const states = createBrowserState(domain, stateContracts);
  return { document, browser, states };
};
const failureName = (result: Completion<unknown>): string => {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw new Error("expected domain failure");
  return domainFailureDiagnostics(result.value).declaration.name;
};
const failurePayload = (result: Completion<unknown>): unknown => {
  if (result.kind !== "domain") throw new Error("expected domain failure");
  return domainFailureDiagnostics(result.value).payload;
};
const flush = () => new Promise((resolve) => setTimeout(resolve, 0));
const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

test("mount resolves the root and reports a missing root", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  expect(isBrowserValue("app", app)).toBe(true);
  const root = value(await browser.root(app));
  expect(isBrowserValue("node", root)).toBe(true);
  const ghost = await browser.mount("ghost");
  expect(failureName(ghost)).toBe("browser::missing_root");
  expect(dataProperty(failurePayload(ghost), "root")).toBe("ghost");
  expect(document.getElementById("app")).not.toBeNull();
});

test("mount without a document host fails closed", async () => {
  expect((globalThis as { document?: unknown }).document).toBeUndefined();
  const browser = createBrowser(domain, contracts);
  const result = await browser.mount("app");
  expect(failureName(result)).toBe("browser::missing_root");
});

test("elements, text and attributes use native text semantics", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = value(await browser.createElement(view, "DIV"));
  const label = value(await browser.createText(view, "<script>alert(1)</script>"));
  value(await browser.setText(label, "hello <b>world</b>"));
  value(await browser.setAttribute(box, "id", "panel"));
  value(await browser.setAttribute(box, "aria-label", "Panel"));
  value(await browser.setAttribute(box, "href", "https://example.com/x"));
  value(await browser.appendChild(root, box));
  value(await browser.appendChild(box, label));
  const panel = document.getElementById("panel")!;
  expect(panel.tagName).toBe("div");
  expect(panel.attributes.get("href")).toBe("https://example.com/x");
  expect(panel.children.length).toBe(1);
  expect(panel.children[0]!.textContent).toBe("hello <b>world</b>");
  value(await browser.removeAttribute(box, "aria-label"));
  value(await browser.removeAttribute(box, "title"));
  expect(panel.attributes.has("aria-label")).toBe(false);
  value(await browser.focus(box));
  expect(panel.focused).toBe(1);
  expect(isBrowserValue("node", box)).toBe(true);
  expect(isBrowserValue("view", view)).toBe(true);
});

test("admission rejects unsafe names and URLs", async () => {
  const { browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  for (const tag of ["script", "style", "iframe", "marquee", ""]) {
    const result = await browser.createElement(view, tag);
    expect(failureName(result)).toBe("browser::rejected");
    expect(dataProperty(failurePayload(result), "reason")).toBe("tag");
  }
  const box = value(await browser.createElement(view, "div"));
  for (const name of ["onclick", "onload", "style", "srcset", "data-x"]) {
    const result = await browser.setAttribute(box, name, "x");
    expect(failureName(result)).toBe("browser::rejected");
    expect(dataProperty(failurePayload(result), "reason")).toBe("attribute");
  }
  for (const href of [
    "javascript:alert(1)",
    "JaVaScRiPt:x",
    "//example.com/x",
    "http://example.com/",
    "",
  ]) {
    const result = await browser.setAttribute(box, "href", href);
    expect(failureName(result)).toBe("browser::rejected");
    expect(dataProperty(failurePayload(result), "reason")).toBe("url");
  }
  const text = value(await browser.createText(view, "x"));
  expect(failureName(await browser.setAttribute(text, "id", "x"))).toBe("browser::rejected");
  expect(
    failureName(
      await browser.onEvent(view, box, "mouseover", async () => value(await browser.focus(box))),
    ),
  ).toBe("browser::rejected");
  expect(failureName(await browser.setTimeout(view, -1n, async () => {}))).toBe(
    "browser::rejected",
  );
  expect(failureName(await browser.setTimeout(view, 2147483648n, async () => {}))).toBe(
    "browser::rejected",
  );
});

test("tree operations enforce scope and detachment rules", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const first = value(await browser.openView(app));
  const second = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = value(await browser.createElement(first, "div"));
  const label = value(await browser.createText(first, "x"));
  const foreign = value(await browser.createElement(second, "span"));
  expect(failureName(await browser.appendChild(box, foreign))).toBe("browser::rejected");
  expect(failureName(await browser.appendChild(label, box))).toBe("browser::rejected");
  expect(failureName(await browser.appendChild(box, root))).toBe("browser::rejected");
  expect(failureName(await browser.removeNode(root))).toBe("browser::rejected");
  expect(failureName(await browser.appendChild(box, box))).toBe("browser::rejected");
  value(await browser.appendChild(root, box));
  value(await browser.appendChild(box, label));
  const host = document.getElementById("app")!;
  expect(host.children.length).toBe(1);
  value(await browser.removeNode(label));
  expect(host.children.length).toBe(1);
  value(await browser.appendChild(box, label));
  expect(host.children.length).toBe(1);
});

test("appending an ancestor into its own subtree fails instead of throwing", async () => {
  const { browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const outer = value(await browser.createElement(view, "div"));
  const middle = value(await browser.createElement(view, "div"));
  const leaf = value(await browser.createElement(view, "span"));
  value(await browser.appendChild(root, outer));
  value(await browser.appendChild(outer, middle));
  value(await browser.appendChild(middle, leaf));
  const cycle = await browser.appendChild(leaf, outer);
  expect(failureName(cycle)).toBe("browser::rejected");
  expect(dataProperty(failurePayload(cycle), "reason")).toBe("cycle");
  const self = await browser.appendChild(middle, middle);
  expect(failureName(self)).toBe("browser::rejected");
  expect(dataProperty(failurePayload(self), "reason")).toBe("cycle");
});

test("event snapshots capture kind, target, value and key", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const button = value(await browser.createElement(view, "button"));
  value(await browser.setAttribute(button, "id", "save"));
  const field = value(await browser.createElement(view, "input"));
  value(await browser.setAttribute(field, "id", "name"));
  value(await browser.appendChild(root, button));
  value(await browser.appendChild(root, field));
  const seen: unknown[] = [];
  const watch = async (event: unknown) => {
    seen.push(event);
    return success(undefined);
  };
  value(await browser.onEvent(view, button, "click", watch));
  value(await browser.onEvent(view, field, "input", watch));
  value(await browser.onEvent(view, field, "keydown", watch));
  const save = document.getElementById("save")!;
  const name = document.getElementById("name")!;
  save.dispatchEvent(new Event("click"));
  (name as unknown as { value: string }).value = "ann";
  name.dispatchEvent(new Event("input"));
  const key = new Event("keydown");
  (key as unknown as { key: string }).key = "Enter";
  name.dispatchEvent(key);
  await flush();
  expect(seen.length).toBe(3);
  for (const event of seen) expect(recordIdentity(event)).toBe(contracts.event);
  const fields = (event: unknown): string[] =>
    ["kind", "target", "value", "key"].map((name) => dataProperty(event, name) as string);
  expect(fields(seen[0])).toEqual(["click", "save", "", ""]);
  expect(fields(seen[1])).toEqual(["input", "name", "ann", ""]);
  expect(fields(seen[2])).toEqual(["keydown", "name", "ann", "Enter"]);
});

test("listener and timer disposal is view-scoped without leaks", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = value(await browser.createElement(view, "div"));
  value(await browser.appendChild(root, box));
  const host = document.getElementById("app")!;
  const inner = host.children[0] as FakeElement;
  let clicks = 0;
  let ticks = 0;
  value(
    await browser.onEvent(view, box, "click", async () => {
      clicks++;
      return success(undefined);
    }),
  );
  value(
    await browser.setTimeout(view, 5n, async () => {
      ticks++;
      return success(undefined);
    }),
  );
  inner.dispatchEvent(new Event("click"));
  await flush();
  expect(clicks).toBe(1);
  value(await browser.disposeView(view));
  inner.dispatchEvent(new Event("click"));
  inner.dispatchEvent(new Event("click"));
  await sleep(30);
  expect(clicks).toBe(1);
  expect(ticks).toBe(0);
  expect(host.children.length).toBe(0);
  expect(inner.parentNode).toBeNull();
  expect(failureName(await browser.createElement(view, "div"))).toBe("browser::disposed");
  value(await browser.disposeView(view));
});

test("app disposal cascades while the root element stays mounted", async () => {
  const { document, browser, states } = setup();
  const app = value(await browser.mount("app"));
  const first = value(await browser.openView(app));
  const second = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = value(await browser.createElement(first, "div"));
  value(await browser.appendChild(root, box));
  const host = document.getElementById("app")!;
  const inner = host.children[0] as FakeElement;
  let clicks = 0;
  let ticks = 0;
  value(
    await browser.onEvent(first, box, "click", async () => {
      clicks++;
      return success(undefined);
    }),
  );
  value(
    await browser.setTimeout(second, 5n, async () => {
      ticks++;
      return success(undefined);
    }),
  );
  const cell = value(await states.createState(first, "draft"));
  value(await browser.disposeApp(app));
  inner.dispatchEvent(new Event("click"));
  await sleep(30);
  expect(clicks).toBe(0);
  expect(ticks).toBe(0);
  expect(host.children.length).toBe(0);
  expect(document.getElementById("app")).not.toBeNull();
  expect(failureName(await browser.openView(app))).toBe("browser::disposed");
  expect(failureName(await browser.root(app))).toBe("browser::disposed");
  expect(failureName(await browser.createElement(second, "div"))).toBe("browser::disposed");
  expect(failureName(await states.readState(cell))).toBe("browser::disposed");
  value(await browser.disposeApp(app));
});

test("disposed views leave sibling views usable", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const first = value(await browser.openView(app));
  const second = value(await browser.openView(app));
  value(await browser.disposeView(first));
  const root = value(await browser.root(app));
  const box = value(await browser.createElement(second, "div"));
  value(await browser.appendChild(root, box));
  let clicks = 0;
  value(
    await browser.onEvent(second, box, "click", async () => {
      clicks++;
      return success(undefined);
    }),
  );
  const host = document.getElementById("app")!;
  (host.children[0] as FakeElement).dispatchEvent(new Event("click"));
  await flush();
  expect(clicks).toBe(1);
});

test("versioned state replaces atomically and reports staleness", async () => {
  const { browser, states } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const cell = value(await states.createState(view, "draft"));
  expect(isBrowserStateValue(stateContracts.state, cell)).toBe(true);
  expect(isBrowserStateValue("other-identity", cell)).toBe(false);
  expect(isBrowserValue("node", cell)).toBe(false);
  const first = value(await states.readState(cell));
  expect(recordIdentity(first)).toBe(stateContracts.snapshot);
  expect(dataProperty(first, "version")).toBe(0n);
  expect(dataProperty(first, "value")).toBe("draft");
  expect(value(await states.replaceState(cell, 0n, "saved"))).toBe(1n);
  const second = value(await states.readState(cell));
  expect(dataProperty(second, "version")).toBe(1n);
  expect(dataProperty(second, "value")).toBe("saved");
  const stale = await states.replaceState(cell, 0n, "late");
  expect(failureName(stale)).toBe("browser::stale_version");
  expect(dataProperty(failurePayload(stale), "expected")).toBe(0n);
  expect(dataProperty(failurePayload(stale), "actual")).toBe(1n);
  const current = value(await states.readState(cell));
  expect(dataProperty(current, "value")).toBe("saved");
});

test("failed handler completions report once without breaking later events", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = value(await browser.createElement(view, "div"));
  value(await browser.appendChild(root, box));
  const host = document.getElementById("app")!;
  const inner = host.children[0] as FakeElement;
  let calls = 0;
  value(
    await browser.onEvent(view, box, "click", async () => {
      calls++;
      if (calls === 1) throw new Error("boom");
      return success(undefined);
    }),
  );
  const reports: unknown[] = [];
  const original = console.error;
  console.error = (...args: unknown[]) => {
    reports.push(args.length === 1 ? args[0] : args);
  };
  try {
    inner.dispatchEvent(new Event("click"));
    await flush();
    inner.dispatchEvent(new Event("click"));
    await flush();
  } finally {
    console.error = original;
  }
  expect(calls).toBe(2);
  expect(reports.length).toBe(1);
  const report = reports[0] as Record<string, unknown>;
  expect(report["kind"]).toBe("can.runtime-diagnostic");
  expect(report["phase"]).toBe("handler");
  expect(typeof report["category"]).toBe("string");
  expect(typeof report["occurrence"]).toBe("string");
  expect(Object.keys(report).sort()).toEqual([
    "category",
    "column",
    "file",
    "kind",
    "line",
    "occurrence",
    "phase",
  ]);
  expect(JSON.stringify(report)).not.toContain("boom");
  value(await browser.disposeView(view));
});

test("failed timer completions report once without breaking later timers", async () => {
  const { browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  let calls = 0;
  value(
    await browser.setTimeout(view, 5n, async () => {
      calls++;
      throw new Error("timer-boom");
    }),
  );
  value(
    await browser.setTimeout(view, 20n, async () => {
      calls++;
      return success(undefined);
    }),
  );
  const reports: unknown[] = [];
  const original = console.error;
  console.error = (...args: unknown[]) => {
    reports.push(args.length === 1 ? args[0] : args);
  };
  try {
    await sleep(60);
  } finally {
    console.error = original;
  }
  expect(calls).toBe(2);
  expect(reports.length).toBe(1);
  const report = reports[0] as Record<string, unknown>;
  expect(report["kind"]).toBe("can.runtime-diagnostic");
  expect(report["phase"]).toBe("handler");
  expect(JSON.stringify(report)).not.toContain("timer-boom");
  value(await browser.disposeView(view));
});

test("shared admission corpus matches the checker", async () => {
  const corpus = (await Bun.file(new URL("./browser-names.json", import.meta.url)).json()) as {
    tags: { accept: string[]; reject: string[] };
    attributes: { accept: string[]; reject: string[] };
    events: { accept: string[]; reject: string[] };
    urls: { accept: string[]; reject: string[] };
    delays: { accept: number[]; reject: number[] };
  };
  expect(corpus.tags.accept.length).toBeGreaterThan(0);
  const { browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  for (const tag of corpus.tags.accept) {
    const result = await browser.createElement(view, tag);
    expect(result.kind).toBe("ok");
  }
  for (const tag of corpus.tags.reject) {
    expect(failureName(await browser.createElement(view, tag))).toBe("browser::rejected");
  }
  const box = value(await browser.createElement(view, "div"));
  const urlValue = (name: string): string =>
    ["href", "action", "formaction", "src"].includes(name.toLowerCase()) ? "/x" : "x";
  for (const name of corpus.attributes.accept) {
    const result = await browser.setAttribute(box, name, urlValue(name));
    expect(result.kind).toBe("ok");
  }
  for (const name of corpus.attributes.reject) {
    expect(failureName(await browser.setAttribute(box, name, "x"))).toBe("browser::rejected");
  }
  for (const kind of corpus.events.accept) {
    const result = await browser.onEvent(view, box, kind, async () => {});
    expect(result.kind).toBe("ok");
  }
  for (const kind of corpus.events.reject) {
    expect(failureName(await browser.onEvent(view, box, kind, async () => {}))).toBe(
      "browser::rejected",
    );
  }
  const link = value(await browser.createElement(view, "a"));
  for (const url of corpus.urls.accept) {
    const result = await browser.setAttribute(link, "href", url);
    expect(result.kind).toBe("ok");
  }
  for (const url of corpus.urls.reject) {
    expect(failureName(await browser.setAttribute(link, "href", url))).toBe("browser::rejected");
  }
  for (const delay of corpus.delays.accept) {
    const result = await browser.setTimeout(view, BigInt(delay), async () => {});
    expect(result.kind).toBe("ok");
  }
  for (const delay of corpus.delays.reject) {
    expect(failureName(await browser.setTimeout(view, BigInt(delay), async () => {}))).toBe(
      "browser::rejected",
    );
  }
  value(await browser.disposeView(view));
});
