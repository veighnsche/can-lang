import { success, failure, invoke, type Completion, type AssertionContext } from "../completion.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import { isAssertScope } from "../assert/context.ts";
import { authorTags, globals, applicability } from "./html.ts";
// Bounded explicit browser catalogue (T22). Native DOM, Event and
// AbortController operations do the work; there is no virtual DOM,
// reducer or reactive runtime. Opaque app/view/node/state handles gate
// every operation: the checker admits literal tags, attributes, events,
// URL values and delays statically, and this factory re-checks every
// dynamic name at runtime. Listener and timer lifetimes belong to their
// view scope: view disposal aborts listeners natively, clears pending
// timers and detaches nodes. State cells are versioned compare-and-swap
// slots; snapshots are immutable data copies.
// Elided harness scope arguments (T24) fail closed as disposed instead
// of throwing a resource-state fault, so asserted handlers exercise
// their disposal arms; disposals no-op. Can code cannot name a
// handle-typed value, so only elided arguments ever meet this path.
const origin = Object.freeze({
  source: "can:browser",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
// The native setTimeout range. Delays outside it fail instead of
// silently clamping to the native overflow behavior.
const maxDelayMs = 2147483647;
export type BrowserDomNode = {
  textContent: string | null;
  readonly parentNode: unknown;
  remove(): void;
  addEventListener(
    type: string,
    listener: (event: Event) => void,
    options?: { signal: AbortSignal },
  ): void;
};
export type BrowserElement = BrowserDomNode & {
  readonly tagName: string;
  id: string;
  setAttribute(name: string, value: string): void;
  removeAttribute(name: string): void;
  appendChild(node: BrowserDomNode): void;
  focus(): void;
};
export type BrowserText = BrowserDomNode;
export type BrowserDocument = {
  getElementById(id: string): BrowserElement | null;
  createElement(tag: string): BrowserElement;
  createTextNode(value: string): BrowserText;
};
type App = {
  document: BrowserDocument;
  root: BrowserElement;
  scope: View;
  views: Set<View>;
  disposed: boolean;
};
type View = {
  app: App;
  controller: AbortController;
  timers: Set<ReturnType<typeof setTimeout>>;
  nodes: Set<NodeRecord>;
  states: Set<StateRecord>;
  disposed: boolean;
};
type NodeRecord = {
  dom: BrowserDomNode;
  view: View;
  text: boolean;
  root: boolean;
  disposed: boolean;
};
type StateRecord = {
  view: View;
  identity: string;
  version: bigint;
  value: unknown;
  disposed: boolean;
};
const apps = new WeakMap<object, App>(),
  views = new WeakMap<object, View>(),
  nodes = new WeakMap<object, NodeRecord>(),
  states = new WeakMap<object, StateRecord>();
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
export function isBrowserValue(kind: string | undefined, value: unknown): boolean {
  const map = kind === "app" ? apps : kind === "view" ? views : kind === "node" ? nodes : undefined;
  return (
    map !== undefined &&
    value !== null &&
    (typeof value === "object" || typeof value === "function") &&
    map.has(value)
  );
}
export function isBrowserStateValue(identity: string, value: unknown): boolean {
  if (value === null || (typeof value !== "object" && typeof value !== "function")) return false;
  return states.get(value)?.identity === identity;
}
function string(value: unknown): string {
  if (typeof value !== "string") throw new TypeError("invalid browser string");
  return value;
}
function integer(value: unknown): bigint {
  if (typeof value !== "bigint") throw new TypeError("invalid browser integer");
  return value;
}
const asciiLower = (value: string): string => value.replace(/[A-Z]/g, (c) => c.toLowerCase());
const urlAttributes = new Set(["href", "action", "formaction", "src"]);
const events = new Set([
  "click",
  "dblclick",
  "input",
  "change",
  "keydown",
  "keyup",
  "focus",
  "blur",
  "submit",
]);
const ariaName = (name: string): boolean => /^aria-[a-z][a-z0-9-]*$/.test(name);
function admitTag(tag: string): boolean {
  return authorTags.has(asciiLower(tag));
}
function admitAttribute(name: string): boolean {
  const lowered = asciiLower(name);
  return (
    globals.has(lowered) ||
    Object.hasOwn(applicability, lowered) ||
    urlAttributes.has(lowered) ||
    ariaName(lowered)
  );
}
function admitEvent(kind: string): boolean {
  return events.has(asciiLower(kind));
}
function admitURL(value: string): boolean {
  // oxlint-disable no-control-regex -- Reject C0 controls, space, DEL, and backslash before scheme checks.
  if (value === "" || !value.isWellFormed() || /[\x00-\x20\x7f\\]/.test(value)) return false;
  // oxlint-enable no-control-regex
  if (value.startsWith("//")) return false;
  if (value.startsWith("/")) return true;
  return value.length >= 8 && value.slice(0, 8).toLowerCase() === "https://";
}
function defaultDocument(): BrowserDocument | undefined {
  const candidate = (globalThis as { document?: unknown }).document;
  if (candidate === null || typeof candidate !== "object") return undefined;
  const host = candidate as Partial<BrowserDocument>;
  if (
    typeof host.getElementById !== "function" ||
    typeof host.createElement !== "function" ||
    typeof host.createTextNode !== "function"
  )
    return undefined;
  return host as BrowserDocument;
}
function stringField(holder: unknown, name: string): string {
  if (holder === null || (typeof holder !== "object" && typeof holder !== "function")) return "";
  const value = (holder as Record<string, unknown>)[name];
  return typeof value === "string" ? value : "";
}
// covers reports whether ancestor is node itself or one of its parents.
// Appending an ancestor into its own subtree would throw natively, so the
// factory rejects the cycle as a Can failure instead.
function covers(ancestor: BrowserDomNode, node: BrowserDomNode): boolean {
  let current: unknown = node;
  while (current !== null && (typeof current === "object" || typeof current === "function")) {
    if (current === ancestor) return true;
    current = (current as { parentNode?: unknown }).parentNode ?? null;
  }
  return false;
}
type EventHandler = (
  event: unknown,
  context: AssertionContext | undefined,
) => Completion<unknown> | Promise<Completion<unknown>>;
type TimerHandler = (
  context: AssertionContext | undefined,
) => Completion<unknown> | Promise<Completion<unknown>>;
function checkHandler(value: unknown): EventHandler & TimerHandler {
  if (typeof value !== "function") throw new TypeError("invalid browser callback");
  return value as EventHandler & TimerHandler;
}
type Contracts = Readonly<{
  missingRoot: string;
  disposed: string;
  rejected: string;
  event: string;
}>;
export function createBrowser(
  domain: ReturnType<typeof createDomainRuntime>,
  contracts: Contracts,
  host?: BrowserDocument,
) {
  const missing = (root: string): Completion<never> =>
    failure(
      domain.create(contracts.missingRoot, record(contracts.missingRoot, [["root", root]]), origin),
    );
  const gone = (): Completion<never> =>
    failure(domain.create(contracts.disposed, record(contracts.disposed, []), origin));
  const denied = (reason: string): Completion<never> =>
    failure(
      domain.create(contracts.rejected, record(contracts.rejected, [["reason", reason]]), origin),
    );
  const live = (view: View): boolean => !view.disposed && !view.app.disposed;
  const liveNode = (found: NodeRecord): boolean => !found.disposed && live(found.view);
  const destroyView = (view: View): void => {
    if (view.disposed) return;
    view.disposed = true;
    view.controller.abort();
    for (const timer of view.timers) clearTimeout(timer);
    view.timers.clear();
    for (const node of view.nodes) {
      node.disposed = true;
      if (!node.root) node.dom.remove();
    }
    view.nodes.clear();
    for (const state of view.states) state.disposed = true;
    view.states.clear();
    view.app.views.delete(view);
  };
  // Settle drops the handler completion once awaited: a failed handler
  // ends that dispatch without affecting later events or timers. There is
  // no reporting channel beyond the consumed event itself.
  const settle = (call: () => Completion<unknown> | Promise<Completion<unknown>>): void => {
    void invoke(call, origin).then(
      () => undefined,
      () => undefined,
    );
  };
  const snapshot = (event: Event): unknown =>
    record(contracts.event, [
      ["kind", event.type],
      ["target", stringField((event as { target?: unknown }).target, "id")],
      ["value", stringField((event as { target?: unknown }).target, "value")],
      ["key", stringField(event, "key")],
    ]);
  return Object.freeze({
    async mount(root: unknown, _context?: AssertionContext) {
      const id = string(root);
      const document = host ?? defaultDocument();
      if (document === undefined) return missing(id);
      const element = document.getElementById(id);
      if (element === null) return missing(id);
      // The root anchor belongs to no view scope: it outlives every view,
      // is never detached by disposal, and parents children from any view
      // of its app. Its dedicated inert scope shares app liveness without
      // view disposal.
      const app: App = {
        document,
        root: element,
        scope: undefined as unknown as View,
        views: new Set(),
        disposed: false,
      };
      app.scope = {
        app,
        controller: new AbortController(),
        timers: new Set(),
        nodes: new Set(),
        states: new Set(),
        disposed: false,
      };
      return success(token(apps, app));
    },
    async root(app: unknown, _context?: AssertionContext) {
      if (isAssertScope(app)) return gone();
      const found = read(apps, app);
      if (found.disposed) return gone();
      return success(
        token(nodes, {
          dom: found.root,
          view: found.scope,
          text: false,
          root: true,
          disposed: false,
        }),
      );
    },
    async openView(app: unknown, _context?: AssertionContext) {
      if (isAssertScope(app)) return gone();
      const found = read(apps, app);
      if (found.disposed) return gone();
      const view: View = {
        app: found,
        controller: new AbortController(),
        timers: new Set(),
        nodes: new Set(),
        states: new Set(),
        disposed: false,
      };
      found.views.add(view);
      return success(token(views, view));
    },
    async disposeView(view: unknown, _context?: AssertionContext) {
      if (isAssertScope(view)) return success(undefined);
      destroyView(read(views, view));
      return success(undefined);
    },
    async disposeApp(app: unknown, _context?: AssertionContext) {
      if (isAssertScope(app)) return success(undefined);
      const found = read(apps, app);
      // destroyView deletes only the view under iteration, which Set iteration tolerates.
      for (const view of found.views) destroyView(view);
      found.disposed = true;
      return success(undefined);
    },
    async createElement(view: unknown, tag: unknown, _context?: AssertionContext) {
      if (isAssertScope(view)) return gone();
      const scope = read(views, view);
      if (!live(scope)) return gone();
      const name = string(tag);
      if (!admitTag(name)) return denied("tag");
      const dom = scope.app.document.createElement(asciiLower(name));
      const record: NodeRecord = { dom, view: scope, text: false, root: false, disposed: false };
      scope.nodes.add(record);
      return success(token(nodes, record));
    },
    async createText(view: unknown, value: unknown, _context?: AssertionContext) {
      if (isAssertScope(view)) return gone();
      const scope = read(views, view);
      if (!live(scope)) return gone();
      const dom = scope.app.document.createTextNode(string(value));
      const record: NodeRecord = { dom, view: scope, text: true, root: false, disposed: false };
      scope.nodes.add(record);
      return success(token(nodes, record));
    },
    async setText(node: unknown, value: unknown, _context?: AssertionContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      found.dom.textContent = string(value);
      return success(undefined);
    },
    async setAttribute(node: unknown, name: unknown, value: unknown, _context?: AssertionContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      if (found.text) return denied("text_node");
      const admitted = asciiLower(string(name));
      if (!admitAttribute(admitted)) return denied("attribute");
      const text = string(value);
      if (urlAttributes.has(admitted) && !admitURL(text)) return denied("url");
      (found.dom as BrowserElement).setAttribute(admitted, text);
      return success(undefined);
    },
    async removeAttribute(node: unknown, name: unknown, _context?: AssertionContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      if (found.text) return denied("text_node");
      const admitted = asciiLower(string(name));
      if (!admitAttribute(admitted)) return denied("attribute");
      (found.dom as BrowserElement).removeAttribute(admitted);
      return success(undefined);
    },
    async appendChild(parent: unknown, child: unknown, _context?: AssertionContext) {
      if (isAssertScope(parent) || isAssertScope(child)) return gone();
      const into = read(nodes, parent);
      const next = read(nodes, child);
      if (!liveNode(into) || !liveNode(next)) return gone();
      if (into.view !== next.view && !(into.root && into.view.app === next.view.app))
        return denied("scope");
      if (into.text) return denied("text_parent");
      if (next.root) return denied("root");
      if (covers(next.dom, into.dom)) return denied("cycle");
      (into.dom as BrowserElement).appendChild(next.dom);
      return success(undefined);
    },
    async removeNode(node: unknown, _context?: AssertionContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      if (found.root) return denied("root");
      found.dom.remove();
      return success(undefined);
    },
    async focus(node: unknown, _context?: AssertionContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      if (!found.text) (found.dom as BrowserElement).focus();
      return success(undefined);
    },
    async onEvent(
      view: unknown,
      node: unknown,
      kind: unknown,
      handler: unknown,
      context?: AssertionContext,
    ) {
      if (isAssertScope(view) || isAssertScope(node)) return gone();
      const scope = read(views, view);
      const found = read(nodes, node);
      if (!live(scope) || !liveNode(found)) return gone();
      if (found.view !== scope && !(found.root && found.view.app === scope.app))
        return denied("scope");
      const name = string(kind);
      if (!admitEvent(name)) return denied("event");
      const callback = checkHandler(handler);
      found.dom.addEventListener(
        asciiLower(name),
        (event) => settle(() => callback(snapshot(event), context)),
        { signal: scope.controller.signal },
      );
      return success(undefined);
    },
    async setTimeout(view: unknown, delay: unknown, handler: unknown, context?: AssertionContext) {
      if (isAssertScope(view)) return gone();
      const scope = read(views, view);
      if (!live(scope)) return gone();
      const wait = integer(delay);
      if (wait < 0n || wait > BigInt(maxDelayMs)) return denied("delay");
      const callback = checkHandler(handler);
      const timer = setTimeout(() => {
        scope.timers.delete(timer);
        if (live(scope)) settle(() => callback(context));
      }, Number(wait));
      scope.timers.add(timer);
      return success(undefined);
    },
  });
}
type StateContracts = Readonly<{
  disposed: string;
  stale: string;
  state: string;
  snapshot: string;
}>;
export function createBrowserState(
  domain: ReturnType<typeof createDomainRuntime>,
  contracts: StateContracts,
) {
  const gone = (): Completion<never> =>
    failure(domain.create(contracts.disposed, record(contracts.disposed, []), origin));
  const stale = (expected: bigint, actual: bigint): Completion<never> =>
    failure(
      domain.create(
        contracts.stale,
        record(contracts.stale, [
          ["expected", expected],
          ["actual", actual],
        ]),
        origin,
      ),
    );
  return Object.freeze({
    async createState(view: unknown, value: unknown, _context?: AssertionContext) {
      if (isAssertScope(view)) return gone();
      const scope = read(views, view);
      if (scope.disposed || scope.app.disposed) return gone();
      const record: StateRecord = {
        view: scope,
        identity: contracts.state,
        version: 0n,
        value,
        disposed: false,
      };
      scope.states.add(record);
      return success(token(states, record));
    },
    async readState(state: unknown, _context?: AssertionContext) {
      if (isAssertScope(state)) return gone();
      const found = read(states, state);
      if (found.disposed || found.view.disposed || found.view.app.disposed) return gone();
      return success(
        record(contracts.snapshot, [
          ["version", found.version],
          ["value", found.value],
        ]),
      );
    },
    async replaceState(
      state: unknown,
      expected: unknown,
      value: unknown,
      _context?: AssertionContext,
    ) {
      if (isAssertScope(state)) return gone();
      const found = read(states, state);
      if (found.disposed || found.view.disposed || found.view.app.disposed) return gone();
      const want = integer(expected);
      if (want !== found.version) return stale(want, found.version);
      found.version += 1n;
      found.value = value;
      return success(found.version);
    },
  });
}
