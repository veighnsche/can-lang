import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createBrowser, isBrowserValue, type BrowserDocument } from "../platform/browser.ts";
import { success, value, type Completion } from "../completion.ts";
import { dataProperty, recordIdentity } from "../data.ts";
// C02 native control surface: checked state, multiselect, file metadata,
// modifiers, composition and caret projections plus live property and
// selection calls. The fake host below models control IDLs (value,
// checked, options, files, selection) while listener dispatch stays
// native Bun EventTarget behavior.
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
  ["browser::missing_root", "browser::disposed", "browser::rejected"].includes(e.name),
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
  invalidQuery: "test-browser-invalid-query",
  some: "test-browser-some",
  none: "test-browser-none",
  modifiers: "test-browser-modifiers",
  selection: "test-browser-selection",
  file: "test-browser-file",
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
  focus(): void {}
}
class FakeInput extends FakeElement {
  value = "";
  checked = false;
  files: { name: string; size: number; type: string }[] | null = null;
  selectionStart = 0;
  selectionEnd = 0;
  selectionDirection = "forward";
  setSelectionRange(start: number, end: number, direction: string): void {
    this.selectionStart = start;
    this.selectionEnd = end;
    this.selectionDirection = direction;
  }
}
class FakeSelect extends FakeElement {
  options: { value: string; selected: boolean }[] = [];
}
class FakeLi extends FakeElement {
  value: unknown = 0;
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
    if (tag === "input") return new FakeInput(tag);
    if (tag === "select") return new FakeSelect(tag);
    if (tag === "li") return new FakeLi(tag);
    return new FakeElement(tag);
  }
  createTextNode(text: string): FakeText {
    const node = new FakeText();
    node.textContent = text;
    return node;
  }
}
// asCheckbox models native checkbox selection behavior: the value IDL
// reads "on" and every selection accessor throws InvalidStateError.
const asCheckbox = (input: FakeInput): FakeInput => {
  input.value = "on";
  const throwing = (): never => {
    throw new Error("InvalidStateError");
  };
  Object.defineProperty(input, "selectionStart", { get: throwing });
  Object.defineProperty(input, "selectionEnd", { get: throwing });
  Object.defineProperty(input, "selectionDirection", { get: throwing });
  input.setSelectionRange = throwing;
  return input;
};
const setup = () => {
  const document = new FakeDocument();
  const browser = createBrowser(domain, contracts, document as unknown as BrowserDocument);
  return { document, browser };
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
const reasonOf = async (result: Completion<unknown>): Promise<string> => {
  expect(failureName(result)).toBe("browser::rejected");
  return dataProperty(failurePayload(result), "reason") as string;
};
const flush = () => new Promise((resolve) => setTimeout(resolve, 0));
const control = async (
  browser: ReturnType<typeof setup>["browser"],
  view: unknown,
  root: unknown,
  tag: string,
  id: string,
): Promise<unknown> => {
  const node = value(await browser.createElement(view, tag));
  value(await browser.setAttribute(node, "id", id));
  value(await browser.appendChild(root, node));
  return node;
};
const watch = (seen: unknown[]) => async (event: unknown) => {
  seen.push(event);
  return success(undefined);
};
const emit = (target: FakeElement, type: string, extra: Record<string, unknown> = {}): void => {
  const event = new Event(type);
  for (const [name, field] of Object.entries(extra))
    Object.defineProperty(event, name, { value: field });
  target.dispatchEvent(event);
};

test("checkbox snapshots distinguish checked state while value stays on", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = await control(browser, view, root, "input", "agree");
  const seen: unknown[] = [];
  value(await browser.onEvent(view, box, "change", watch(seen)));
  const host = asCheckbox(document.getElementById("agree") as unknown as FakeInput);
  emit(host, "change");
  host.checked = true;
  emit(host, "change");
  await flush();
  expect(seen.length).toBe(2);
  expect(dataProperty(seen[0], "checked")).toBe(false);
  expect(dataProperty(seen[1], "checked")).toBe(true);
  expect(dataProperty(seen[0], "value")).toBe("on");
  expect(dataProperty(seen[1], "value")).toBe("on");
  expect(dataProperty(seen[0], "kind")).toBe("change");
  expect(dataProperty(seen[0], "target")).toBe("agree");
});

test("dirty live values reset through set_value, not set_attribute", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "qty");
  const host = document.getElementById("qty") as unknown as FakeInput;
  host.value = "user edit";
  value(await browser.setAttribute(node, "value", "normalized"));
  expect(host.value).toBe("user edit");
  expect(host.attributes.get("value")).toBe("normalized");
  value(await browser.setValue(node, "reset default"));
  expect(host.value).toBe("reset default");
  expect(value(await browser.readValue(node))).toBe("reset default");
  const seen: unknown[] = [];
  value(await browser.onEvent(view, node, "input", watch(seen)));
  emit(host, "input");
  await flush();
  expect(dataProperty(seen[0], "value")).toBe("reset default");
});

test("checked state writes and reads live", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "agree");
  const host = asCheckbox(document.getElementById("agree") as unknown as FakeInput);
  expect(value(await browser.readChecked(node))).toBe(false);
  value(await browser.setChecked(node, true));
  expect(host.checked).toBe(true);
  expect(value(await browser.readChecked(node))).toBe(true);
  value(await browser.setChecked(node, false));
  expect(value(await browser.readChecked(node))).toBe(false);
  const seen: unknown[] = [];
  value(await browser.onEvent(view, node, "change", watch(seen)));
  host.checked = true;
  emit(host, "change");
  await flush();
  expect(dataProperty(seen[0], "checked")).toBe(true);
});

test("multiselect snapshots, reads and resets by value", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "select", "tags");
  const host = document.getElementById("tags") as unknown as FakeSelect;
  host.options = [
    { value: "a", selected: false },
    { value: "b", selected: true },
    { value: "c", selected: true },
  ];
  const seen: unknown[] = [];
  value(await browser.onEvent(view, node, "change", watch(seen)));
  emit(host, "change");
  await flush();
  expect(dataProperty(seen[0], "selected")).toEqual(["b", "c"]);
  expect(value(await browser.readSelected(node))).toEqual(["b", "c"]);
  value(await browser.setSelected(node, ["c", "missing"]));
  expect(host.options.map((option) => option.selected)).toEqual([false, false, true]);
  expect(value(await browser.readSelected(node))).toEqual(["c"]);
  value(await browser.setSelected(node, []));
  expect(value(await browser.readSelected(node))).toEqual([]);
});

test("file selections project bounded metadata without bytes", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "upload");
  const host = document.getElementById("upload") as unknown as FakeInput;
  host.files = [
    { name: "a.csv", size: 12, type: "text/csv" },
    { name: "b.png", size: 300, type: "image/png" },
  ];
  const seen: unknown[] = [];
  value(await browser.onEvent(view, node, "change", watch(seen)));
  emit(host, "change");
  await flush();
  const files = dataProperty(seen[0], "files") as unknown[];
  expect(files.length).toBe(2);
  for (const file of files) expect(recordIdentity(file)).toBe(contracts.file);
  expect(dataProperty(files[0], "name")).toBe("a.csv");
  expect(dataProperty(files[0], "size")).toBe(12n);
  expect(dataProperty(files[0], "mime")).toBe("text/csv");
  const live = value(await browser.readFiles(node)) as unknown[];
  expect(live.length).toBe(2);
  expect(dataProperty(live[1], "name")).toBe("b.png");
  host.files = Array.from({ length: 130 }, (_, index) => ({
    name: "f" + index,
    size: index,
    type: "text/plain",
  }));
  emit(host, "change");
  await flush();
  expect((dataProperty(seen[1], "files") as unknown[]).length).toBe(128);
  expect((value(await browser.readFiles(node)) as unknown[]).length).toBe(128);
  host.files = [
    { name: "ok.txt", size: 1, type: "text/plain" },
    { name: "bad", size: Number.NaN, type: "text/plain" },
  ];
  emit(host, "change");
  await flush();
  expect((dataProperty(seen[2], "files") as unknown[]).length).toBe(1);
});

test("modifier keys and composition project per event", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "field");
  const host = document.getElementById("field") as unknown as FakeInput;
  const seen: unknown[] = [];
  value(await browser.onEvent(view, node, "keydown", watch(seen)));
  value(await browser.onEvent(view, node, "input", watch(seen)));
  emit(host, "keydown", { key: "s", ctrlKey: true, shiftKey: true });
  emit(host, "input", { isComposing: true });
  await flush();
  expect(seen.length).toBe(2);
  const held = dataProperty(seen[0], "modifiers") as unknown;
  expect(recordIdentity(held)).toBe(contracts.modifiers);
  expect(dataProperty(held, "alt")).toBe(false);
  expect(dataProperty(held, "ctrl")).toBe(true);
  expect(dataProperty(held, "meta")).toBe(false);
  expect(dataProperty(held, "shift")).toBe(true);
  expect(dataProperty(seen[0], "composing")).toBe(false);
  expect(dataProperty(seen[1], "composing")).toBe(true);
  const plain = dataProperty(seen[1], "modifiers") as unknown;
  expect(dataProperty(plain, "ctrl")).toBe(false);
});

test("caret snapshots share the live selection projection", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "field");
  const host = document.getElementById("field") as unknown as FakeInput;
  host.value = "hello";
  host.selectionStart = 1;
  host.selectionEnd = 4;
  host.selectionDirection = "backward";
  const seen: unknown[] = [];
  value(await browser.onEvent(view, node, "input", watch(seen)));
  emit(host, "input");
  await flush();
  const caret = dataProperty(seen[0], "selection") as unknown;
  expect(recordIdentity(caret)).toBe(contracts.selection);
  expect(dataProperty(caret, "start")).toBe(1n);
  expect(dataProperty(caret, "end")).toBe(4n);
  expect(dataProperty(caret, "direction")).toBe("backward");
  const live = value(await browser.readSelection(node)) as unknown;
  expect(dataProperty(live, "start")).toBe(1n);
  expect(dataProperty(live, "end")).toBe(4n);
  expect(dataProperty(live, "direction")).toBe("backward");
});

test("controls without a text selection read the neutral caret", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = await control(browser, view, root, "div", "panel");
  const check = await control(browser, view, root, "input", "agree");
  const text = value(await browser.createText(view, "x"));
  asCheckbox(document.getElementById("agree") as unknown as FakeInput);
  const seen: unknown[] = [];
  value(await browser.onEvent(view, box, "click", watch(seen)));
  value(await browser.onEvent(view, check, "change", watch(seen)));
  emit(document.getElementById("panel")!, "click");
  emit(document.getElementById("agree")!, "change");
  await flush();
  for (const event of seen) {
    const caret = dataProperty(event, "selection") as unknown;
    expect(dataProperty(caret, "start")).toBe(-1n);
    expect(dataProperty(caret, "end")).toBe(-1n);
    expect(dataProperty(caret, "direction")).toBe("none");
  }
  for (const node of [box, check, text]) {
    const caret = value(await browser.readSelection(node)) as unknown;
    expect(dataProperty(caret, "start")).toBe(-1n);
    expect(dataProperty(caret, "direction")).toBe("none");
  }
});

test("set_selection writes caret natively and rejects bad bounds", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "field");
  const host = document.getElementById("field") as unknown as FakeInput;
  host.value = "hello";
  value(await browser.setSelection(node, 1n, 3n, "Backward"));
  expect(host.selectionStart).toBe(1);
  expect(host.selectionEnd).toBe(3);
  expect(host.selectionDirection).toBe("backward");
  expect(await reasonOf(await browser.setSelection(node, 3n, 1n, "none"))).toBe("selection");
  expect(await reasonOf(await browser.setSelection(node, -1n, 1n, "none"))).toBe("selection");
  expect(await reasonOf(await browser.setSelection(node, 0n, 6n, "none"))).toBe("selection");
  expect(await reasonOf(await browser.setSelection(node, 0n, 0n, "sideways"))).toBe("direction");
  const box = await control(browser, view, root, "div", "panel");
  expect(await reasonOf(await browser.setSelection(box, 0n, 0n, "none"))).toBe("property");
  const check = await control(browser, view, root, "input", "agree");
  asCheckbox(document.getElementById("agree") as unknown as FakeInput);
  expect(await reasonOf(await browser.setSelection(check, 0n, 0n, "none"))).toBe("selection");
  const text = value(await browser.createText(view, "x"));
  expect(await reasonOf(await browser.setSelection(text, 0n, 0n, "none"))).toBe("text_node");
});

test("live calls deny controls without the matching IDL", async () => {
  const { browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const box = await control(browser, view, root, "div", "panel");
  const item = await control(browser, view, root, "li", "row");
  const field = await control(browser, view, root, "input", "field");
  const text = value(await browser.createText(view, "x"));
  expect(await reasonOf(await browser.setValue(box, "x"))).toBe("property");
  expect(await reasonOf(await browser.readValue(box))).toBe("property");
  expect(await reasonOf(await browser.setChecked(box, true))).toBe("property");
  expect(await reasonOf(await browser.readChecked(box))).toBe("property");
  expect(await reasonOf(await browser.setSelected(box, []))).toBe("property");
  expect(await reasonOf(await browser.readSelected(box))).toBe("property");
  expect(await reasonOf(await browser.readFiles(field))).toBe("property");
  expect(await reasonOf(await browser.setValue(item, "x"))).toBe("property");
  expect(await reasonOf(await browser.readValue(item))).toBe("property");
  expect(await reasonOf(await browser.setValue(text, "x"))).toBe("text_node");
  expect(await reasonOf(await browser.readChecked(text))).toBe("text_node");
  expect(await reasonOf(await browser.setSelected(text, []))).toBe("text_node");
  expect(await reasonOf(await browser.readSelected(text))).toBe("text_node");
  expect(await reasonOf(await browser.readFiles(text))).toBe("text_node");
  await expect(browser.setValue(field, 7)).rejects.toThrow(TypeError);
  await expect(browser.setChecked(field, "yes")).rejects.toThrow(TypeError);
  const tags = await control(browser, view, root, "select", "tags");
  await expect(browser.setSelected(tags, "a")).rejects.toThrow(TypeError);
  await expect(browser.setSelected(tags, ["a", 7])).rejects.toThrow(TypeError);
  await expect(browser.setSelection(field, 0, 0n, "none")).rejects.toThrow(TypeError);
  // Input shape faults precede control admission, like set_attribute:
  // mistyped inputs throw even where the control would deny.
  await expect(browser.setValue(box, 7)).rejects.toThrow(TypeError);
  await expect(browser.setChecked(box, "yes")).rejects.toThrow(TypeError);
  await expect(browser.setSelected(box, "a")).rejects.toThrow(TypeError);
  await expect(browser.setSelection(box, 0, 0n, "none")).rejects.toThrow(TypeError);
  await expect(browser.setSelection(field, 0n, 0n, 7)).rejects.toThrow(TypeError);
});

test("autofilled content is observable and reset restores defaults", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "email");
  const host = document.getElementById("email") as unknown as FakeInput;
  const seen: unknown[] = [];
  value(await browser.onEvent(view, node, "input", watch(seen)));
  host.value = "ann@example.com";
  emit(host, "input");
  await flush();
  expect(dataProperty(seen[0], "value")).toBe("ann@example.com");
  expect(value(await browser.readValue(node))).toBe("ann@example.com");
  value(await browser.setValue(node, ""));
  value(await browser.setSelection(node, 0n, 0n, "none"));
  expect(value(await browser.readValue(node))).toBe("");
  emit(host, "input");
  await flush();
  expect(dataProperty(seen[1], "value")).toBe("");
});

test("live projections escape as frozen copies only", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const select = await control(browser, view, root, "select", "tags");
  const upload = await control(browser, view, root, "input", "upload");
  const selectHost = document.getElementById("tags") as unknown as FakeSelect;
  const uploadHost = document.getElementById("upload") as unknown as FakeInput;
  selectHost.options = [{ value: "a", selected: true }];
  uploadHost.files = [{ name: "a.csv", size: 1, type: "text/csv" }];
  const first = value(await browser.readSelected(select)) as unknown[];
  const second = value(await browser.readSelected(select)) as unknown[];
  expect(first).not.toBe(second);
  expect(Object.isFrozen(first)).toBe(true);
  expect(() => (first as string[]).push("b")).toThrow();
  const files = value(await browser.readFiles(upload)) as unknown[];
  expect(Object.isFrozen(files)).toBe(true);
  expect(Object.isFrozen(files[0])).toBe(true);
  expect(() => (files as unknown[]).push(files[0])).toThrow();
  const seen: unknown[] = [];
  value(await browser.onEvent(view, select, "change", watch(seen)));
  emit(selectHost, "change");
  await flush();
  const snapshot = dataProperty(seen[0], "selected") as unknown[];
  expect(Object.isFrozen(snapshot)).toBe(true);
  expect(snapshot).not.toBe(first);
});

test("control calls fail closed on disposed views and apps", async () => {
  const { browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const node = await control(browser, view, root, "input", "field");
  value(await browser.disposeView(view));
  for (const result of [
    await browser.setValue(node, "x"),
    await browser.setChecked(node, true),
    await browser.setSelected(node, []),
    await browser.setSelection(node, 0n, 0n, "none"),
    await browser.readValue(node),
    await browser.readChecked(node),
    await browser.readSelected(node),
    await browser.readSelection(node),
    await browser.readFiles(node),
  ]) {
    expect(failureName(result)).toBe("browser::disposed");
  }
  expect(isBrowserValue("node", node)).toBe(true);
});
