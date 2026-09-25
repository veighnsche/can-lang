import { success, failure, invoke, type Completion, type AssertionContext } from "../completion.ts";
import type { OwnerContext } from "../owner-core.ts";
import { record } from "../data.ts";
import { createDomainRuntime } from "../domain.ts";
import { resourceStateFailure } from "../failure.ts";
import { isAssertScope } from "../assert/context.ts";
import { reportBrowserDiagnostic } from "../browser/diagnostics.ts";
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
// Query bootstrap, cancel-policy listeners and handler reporting are
// the UP13 surface: strict location.search budgets with
// malformed/duplicate rejection, synchronous registration-time
// cancellation with one immutable snapshot, and one sealed diagnostic
// per failed event/timer callback through the browser reporter.
const origin = Object.freeze({
  source: "can:browser",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});
// The native setTimeout range. Delays outside it fail instead of
// silently clamping to the native overflow behavior.
const maxDelayMs = 2147483647;
// Strict query budgets: the entire raw location.search and each decoded
// value, both in UTF-8 bytes. Literal keys are at most 64 ASCII bytes.
const maxQueryBytes = 8192;
const maxQueryValueBytes = 256;
const maxQueryKeyBytes = 64;
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
  readonly location?: { readonly search: string };
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
// Query keys mirror the checker literal rule: [a-z][a-z0-9_]*, at most
// 64 bytes. Non-ASCII input always fails the charset, so the UTF-16
// length check agrees with the byte bound on the admit path.
function admitQueryKey(key: string): boolean {
  if (key.length === 0 || key.length > maxQueryKeyBytes) return false;
  const first = key.charCodeAt(0);
  if (first < 97 || first > 122) return false;
  for (let i = 1; i < key.length; i++) {
    const code = key.charCodeAt(i);
    if ((code >= 97 && code <= 122) || (code >= 48 && code <= 57) || code === 95) continue;
    return false;
  }
  return true;
}
function utf8Bytes(value: string): number {
  return new TextEncoder().encode(value).byteLength;
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
// The raw query string for query_parameter: the host document location
// when one is bound, else the ambient location. A host without any
// location carries no query string, so lookups read empty and return
// none; only a non-string search (a broken host contract) throws.
function querySearch(host: BrowserDocument | undefined): string {
  const override = host?.location?.search;
  if (override !== undefined) {
    if (typeof override !== "string") throw new TypeError("invalid browser location");
    return override;
  }
  const candidate = (globalThis as { location?: unknown }).location;
  if (candidate === null || typeof candidate !== "object") return "";
  const search = (candidate as { search?: unknown }).search;
  if (search === undefined) return "";
  if (typeof search !== "string") throw new TypeError("invalid browser location");
  return search;
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
// Bun staging passes its assertion context in the trailing slot; browser
// production passes the explicit owner context. Listener and timer
// adapters forward the slot positionally into the handler's own trailing
// context, so both shapes stay assignable here.
type TrailingContext = OwnerContext | AssertionContext;
type EventHandler = (
  event: unknown,
  context: TrailingContext | undefined,
) => Completion<unknown> | Promise<Completion<unknown>>;
type TimerHandler = (
  context: TrailingContext | undefined,
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
  invalidQuery: string;
  some: string;
  none: string;
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
  const invalid = (key: string, reason: string): Completion<never> =>
    failure(
      domain.create(
        contracts.invalidQuery,
        record(contracts.invalidQuery, [
          ["key", key],
          ["reason", reason],
        ]),
        origin,
      ),
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
  // Settle reports one failed handler completion through the sealed
  // reporter and ends that dispatch: later events and timers still run,
  // disposal still removes listeners and timers, and successful handlers
  // report nothing. The record carries category, phase and Can
  // file/line/column only; raw input, native causes and secrets never
  // cross this boundary. Reporting itself never breaks dispatch.
  const settle = (call: () => Completion<unknown> | Promise<Completion<unknown>>): void => {
    void invoke(call, origin).then(
      (completion) => {
        if (completion.kind === "ok") return;
        try {
          reportBrowserDiagnostic(completion, "handler");
        } catch {
          // Diagnostics never replace the original occurrence.
        }
      },
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
    async mount(root: unknown, _context?: TrailingContext) {
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
    async root(app: unknown, _context?: TrailingContext) {
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
    async openView(app: unknown, _context?: TrailingContext) {
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
    async disposeView(view: unknown, _context?: TrailingContext) {
      if (isAssertScope(view)) return success(undefined);
      destroyView(read(views, view));
      return success(undefined);
    },
    async disposeApp(app: unknown, _context?: TrailingContext) {
      if (isAssertScope(app)) return success(undefined);
      const found = read(apps, app);
      // destroyView deletes only the view under iteration, which Set iteration tolerates.
      for (const view of found.views) destroyView(view);
      found.disposed = true;
      return success(undefined);
    },
    async createElement(view: unknown, tag: unknown, _context?: TrailingContext) {
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
    async createText(view: unknown, value: unknown, _context?: TrailingContext) {
      if (isAssertScope(view)) return gone();
      const scope = read(views, view);
      if (!live(scope)) return gone();
      const dom = scope.app.document.createTextNode(string(value));
      const record: NodeRecord = { dom, view: scope, text: true, root: false, disposed: false };
      scope.nodes.add(record);
      return success(token(nodes, record));
    },
    async setText(node: unknown, value: unknown, _context?: TrailingContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      found.dom.textContent = string(value);
      return success(undefined);
    },
    async setAttribute(node: unknown, name: unknown, value: unknown, _context?: TrailingContext) {
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
    async removeAttribute(node: unknown, name: unknown, _context?: TrailingContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      if (found.text) return denied("text_node");
      const admitted = asciiLower(string(name));
      if (!admitAttribute(admitted)) return denied("attribute");
      (found.dom as BrowserElement).removeAttribute(admitted);
      return success(undefined);
    },
    async appendChild(parent: unknown, child: unknown, _context?: TrailingContext) {
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
    async removeNode(node: unknown, _context?: TrailingContext) {
      if (isAssertScope(node)) return gone();
      const found = read(nodes, node);
      if (!liveNode(found)) return gone();
      if (found.root) return denied("root");
      found.dom.remove();
      return success(undefined);
    },
    async focus(node: unknown, _context?: TrailingContext) {
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
      context?: TrailingContext,
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
    async onCancelKey(
      view: unknown,
      node: unknown,
      kind: unknown,
      key: unknown,
      handler: unknown,
      context?: TrailingContext,
    ) {
      if (isAssertScope(view) || isAssertScope(node)) return gone();
      const scope = read(views, view);
      const found = read(nodes, node);
      if (!live(scope) || !liveNode(found)) return gone();
      if (found.view !== scope && !(found.root && found.view.app === scope.app))
        return denied("scope");
      const name = asciiLower(string(kind));
      if (name !== "keydown" && name !== "keyup") return denied("event");
      const wanted = string(key);
      if (wanted === "") return denied("key");
      const callback = checkHandler(handler);
      // Registration policy: the key match and cancelability decide
      // synchronously inside native dispatch, before the one immutable
      // snapshot dispatches. A different key or a noncancelable event
      // still dispatches once without cancellation. A view disposed
      // before dispatch can neither cancel nor dispatch.
      found.dom.addEventListener(
        name,
        (event) => {
          if (!live(scope)) return;
          if (stringField(event, "key") === wanted && event.cancelable) event.preventDefault();
          settle(() => callback(snapshot(event), context));
        },
        { signal: scope.controller.signal },
      );
      return success(undefined);
    },
    async onCancelEvent(
      view: unknown,
      node: unknown,
      kind: unknown,
      handler: unknown,
      context?: TrailingContext,
    ) {
      if (isAssertScope(view) || isAssertScope(node)) return gone();
      const scope = read(views, view);
      const found = read(nodes, node);
      if (!live(scope) || !liveNode(found)) return gone();
      if (found.view !== scope && !(found.root && found.view.app === scope.app))
        return denied("scope");
      const name = asciiLower(string(kind));
      if (name !== "submit") return denied("event");
      const callback = checkHandler(handler);
      found.dom.addEventListener(
        name,
        (event) => {
          if (!live(scope)) return;
          if (event.cancelable) event.preventDefault();
          settle(() => callback(snapshot(event), context));
        },
        { signal: scope.controller.signal },
      );
      return success(undefined);
    },
    async setTimeout(view: unknown, delay: unknown, handler: unknown, context?: TrailingContext) {
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
    async queryParameter(key: unknown, _context?: TrailingContext) {
      const name = string(key);
      if (!admitQueryKey(name)) return invalid(name, "key");
      // Read-only bootstrap input: the literal key looks up one value in
      // the raw location search. The strict precheck bounds the whole raw
      // string and every decoded value, and rejects malformed escapes,
      // invalid UTF-8 and ill-formed Unicode before native URLSearchParams
      // supplies pair semantics and getAll duplicate detection, including
      // encoded-key aliases. Zero occurrences return none, one returns
      // some, and two or more fail; budgets and duplicates report the
      // requested key, never raw query bytes.
      const search = querySearch(host);
      if (utf8Bytes(search) > maxQueryBytes) return invalid(name, "too_large");
      if (!search.isWellFormed()) return invalid(name, "malformed");
      const pairs = search.startsWith("?") ? search.slice(1) : search;
      if (pairs !== "") {
        for (const segment of pairs.split("&")) {
          if (segment === "") continue;
          const cut = segment.indexOf("=");
          const rawKey = cut === -1 ? segment : segment.slice(0, cut);
          const rawValue = cut === -1 ? "" : segment.slice(cut + 1);
          let decodedKey: string;
          let decodedValue: string;
          try {
            decodedKey = decodeURIComponent(rawKey.replace(/\+/g, " "));
            decodedValue = decodeURIComponent(rawValue.replace(/\+/g, " "));
          } catch {
            return invalid(name, "malformed");
          }
          if (!decodedKey.isWellFormed() || !decodedValue.isWellFormed())
            return invalid(name, "malformed");
          if (utf8Bytes(decodedValue) > maxQueryValueBytes) return invalid(name, "too_large");
        }
      }
      const values = new URLSearchParams(search).getAll(name);
      if (values.length === 0) return success(record(contracts.none, []));
      if (values.length > 1) return invalid(name, "duplicate");
      return success(record(contracts.some, [["value", values[0]]]));
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
    async createState(view: unknown, value: unknown, _context?: TrailingContext) {
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
    async readState(state: unknown, _context?: TrailingContext) {
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
      _context?: TrailingContext,
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
