import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import {
  createBrowser,
  type BrowserDocument,
} from "../platform/browser.ts";
import { success, value, type Completion } from "../completion.ts";
import { dataProperty, recordIdentity } from "../data.ts";

// UP13 cancel policy: registration-time synchronous cancellation with
// exactly one immutable snapshot. Listener dispatch runs through native
// Bun EventTarget/Event/AbortController behavior; defaultPrevented is
// asserted synchronously after dispatchEvent, before any Can handler
// continuation settles.
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
const str = shape("primitive", "str");
const declarations = catalogue.errors.filter((e) =>
  [
    "browser::missing_root",
    "browser::disposed",
    "browser::rejected",
    "browser::invalid_query",
  ].includes(e.name),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, ...errors],
});
const contracts = {
  missingRoot: errors[0]!.identity,
  disposed: errors[1]!.identity,
  rejected: errors[2]!.identity,
  event: "test-browser-event",
  invalidQuery: errors[3]!.identity,
  some: "test-browser-some",
  none: "test-browser-none",
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
const flush = () => new Promise((resolve) => setTimeout(resolve, 0));
const keyEvent = (kind: string, key: string, cancelable: boolean): Event => {
  const event = new Event(kind, { cancelable });
  (event as unknown as { key: string }).key = key;
  return event;
};

test("matching cancelable keys cancel synchronously with one snapshot", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const field = value(await browser.createElement(view, "input"));
  value(await browser.setAttribute(field, "id", "name"));
  value(await browser.appendChild(root, field));
  const seen: unknown[] = [];
  value(
    await browser.onCancelKey(view, field, "keydown", "Enter", async (event: unknown) => {
      seen.push(event);
      return success(undefined);
    }),
  );
  const target = document.getElementById("name")!;
  const event = keyEvent("keydown", "Enter", true);
  target.dispatchEvent(event);
  expect(event.defaultPrevented).toBe(true);
  await flush();
  expect(seen.length).toBe(1);
  expect(recordIdentity(seen[0])).toBe(contracts.event);
  expect(Object.isFrozen(seen[0])).toBe(true);
  expect(dataProperty(seen[0], "kind")).toBe("keydown");
  expect(dataProperty(seen[0], "target")).toBe("name");
  expect(dataProperty(seen[0], "key")).toBe("Enter");
  value(await browser.disposeView(view));
});

test("unmatched keys and noncancelable events dispatch once without cancellation", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const field = value(await browser.createElement(view, "input"));
  value(await browser.setAttribute(field, "id", "name"));
  value(await browser.appendChild(root, field));
  let calls = 0;
  value(
    await browser.onCancelKey(view, field, "KEYDOWN", "Enter", async () => {
      calls++;
      return success(undefined);
    }),
  );
  const target = document.getElementById("name")!;
  const other = keyEvent("keydown", "Escape", true);
  target.dispatchEvent(other);
  expect(other.defaultPrevented).toBe(false);
  const shifted = keyEvent("keydown", "enter", true);
  target.dispatchEvent(shifted);
  expect(shifted.defaultPrevented).toBe(false);
  const fixed = keyEvent("keydown", "Enter", false);
  target.dispatchEvent(fixed);
  expect(fixed.defaultPrevented).toBe(false);
  await flush();
  expect(calls).toBe(3);
  value(await browser.disposeView(view));
});

test("submit cancellation follows the same policy, ordinary events never cancel", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const form = value(await browser.createElement(view, "form"));
  value(await browser.setAttribute(form, "id", "edit"));
  value(await browser.appendChild(root, form));
  const button = value(await browser.createElement(view, "button"));
  value(await browser.setAttribute(button, "id", "save"));
  value(await browser.appendChild(root, button));
  let submits = 0;
  let clicks = 0;
  value(
    await browser.onCancelEvent(view, form, "submit", async () => {
      submits++;
      return success(undefined);
    }),
  );
  value(
    await browser.onEvent(view, button, "click", async () => {
      clicks++;
      return success(undefined);
    }),
  );
  const formTarget = document.getElementById("edit")!;
  const submit = new Event("submit", { cancelable: true });
  formTarget.dispatchEvent(submit);
  expect(submit.defaultPrevented).toBe(true);
  const plain = new Event("submit", { cancelable: false });
  formTarget.dispatchEvent(plain);
  expect(plain.defaultPrevented).toBe(false);
  const saveTarget = document.getElementById("save")!;
  const click = new Event("click", { cancelable: true });
  saveTarget.dispatchEvent(click);
  expect(click.defaultPrevented).toBe(false);
  await flush();
  expect(submits).toBe(2);
  expect(clicks).toBe(1);
  value(await browser.disposeView(view));
});

test("cancel admission rejects unadmitted kinds and empty keys", async () => {
  const { browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const field = value(await browser.createElement(view, "input"));
  for (const kind of ["click", "submit", "input", ""]) {
    const result = await browser.onCancelKey(view, field, kind, "Enter", async () => {});
    expect(failureName(result)).toBe("browser::rejected");
    expect(dataProperty(failurePayload(result), "reason")).toBe("event");
  }
  const empty = await browser.onCancelKey(view, field, "keydown", "", async () => {});
  expect(failureName(empty)).toBe("browser::rejected");
  expect(dataProperty(failurePayload(empty), "reason")).toBe("key");
  for (const kind of ["click", "keydown", "keyup", ""]) {
    const result = await browser.onCancelEvent(view, field, kind, async () => {});
    expect(failureName(result)).toBe("browser::rejected");
    expect(dataProperty(failurePayload(result), "reason")).toBe("event");
  }
  const second = value(await browser.openView(app));
  const foreign = value(await browser.createElement(second, "input"));
  expect(failureName(await browser.onCancelKey(view, foreign, "keydown", "Enter", async () => {}))).toBe(
    "browser::rejected",
  );
  value(await browser.disposeView(view));
  expect(failureName(await browser.onCancelKey(view, field, "keydown", "Enter", async () => {}))).toBe(
    "browser::disposed",
  );
  expect(failureName(await browser.onCancelEvent(view, field, "submit", async () => {}))).toBe(
    "browser::disposed",
  );
});

test("disposed views cannot cancel or dispatch", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const field = value(await browser.createElement(view, "input"));
  value(await browser.setAttribute(field, "id", "name"));
  value(await browser.appendChild(root, field));
  const host = document.getElementById("app")!;
  const inner = host.children[0] as FakeElement;
  let calls = 0;
  value(
    await browser.onCancelKey(view, field, "keydown", "Enter", async () => {
      calls++;
      return success(undefined);
    }),
  );
  value(await browser.disposeView(view));
  const event = keyEvent("keydown", "Enter", true);
  inner.dispatchEvent(event);
  expect(event.defaultPrevented).toBe(false);
  await flush();
  expect(calls).toBe(0);
});

test("a late handler failure neither undoes cancellation nor hides its report", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const field = value(await browser.createElement(view, "input"));
  value(await browser.setAttribute(field, "id", "name"));
  value(await browser.appendChild(root, field));
  let calls = 0;
  value(
    await browser.onCancelKey(view, field, "keydown", "Enter", async () => {
      calls++;
      throw new Error("late-cancel-boom");
    }),
  );
  const reports: unknown[] = [];
  const original = console.error;
  console.error = (...args: unknown[]) => {
    reports.push(args.length === 1 ? args[0] : args);
  };
  let event!: Event;
  try {
    event = keyEvent("keydown", "Enter", true);
    document.getElementById("name")!.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(true);
    await flush();
  } finally {
    console.error = original;
  }
  expect(calls).toBe(1);
  expect(event.defaultPrevented).toBe(true);
  expect(reports.length).toBe(1);
  const report = reports[0] as Record<string, unknown>;
  expect(report["kind"]).toBe("can.runtime-diagnostic");
  expect(report["phase"]).toBe("handler");
  expect(JSON.stringify(report)).not.toContain("late-cancel-boom");
  value(await browser.disposeView(view));
});

test("ordinary and cancel registrations dispatch independently once each", async () => {
  const { document, browser } = setup();
  const app = value(await browser.mount("app"));
  const view = value(await browser.openView(app));
  const root = value(await browser.root(app));
  const field = value(await browser.createElement(view, "input"));
  value(await browser.setAttribute(field, "id", "name"));
  value(await browser.appendChild(root, field));
  let ordinary = 0;
  let first = 0;
  let second = 0;
  value(
    await browser.onEvent(view, field, "keydown", async () => {
      ordinary++;
      return success(undefined);
    }),
  );
  value(
    await browser.onCancelKey(view, field, "keydown", "Enter", async () => {
      first++;
      return success(undefined);
    }),
  );
  value(
    await browser.onCancelKey(view, field, "keydown", "Enter", async () => {
      second++;
      return success(undefined);
    }),
  );
  const event = keyEvent("keydown", "Enter", true);
  document.getElementById("name")!.dispatchEvent(event);
  expect(event.defaultPrevented).toBe(true);
  await flush();
  expect([ordinary, first, second]).toEqual([1, 1, 1]);
  value(await browser.disposeView(view));
});
