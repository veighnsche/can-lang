// Compiler-owned HTMX guard: structural request/response/swap policy for
// action pages driven by the pinned htmx asset.
//
// The page-level `htmx-config` keeps every 4xx/5xx out of swaps by default
// and admits declared action cases only through exact `hx-status:<code>`
// attributes generated from the checked action case table. This adapter
// enforces the structural remainder that attributes cannot express:
//
// - `htmx:before:request`: the submitted target must be the resolved,
//   still-connected node; an absent or detached target cancels the
//   submission before any byte is sent.
// - `htmx:before:response`: fetch-level redirects and any `hx-*`
//   response-control header cancel processing before htmx applies them.
//   Application renderers have no API to set these headers, and the
//   server adapter rejects them; the guard is the browser backstop.
// - `htmx:before:swap`: a blocked failure (`swap === "none"`) cancels
//   every task when any task could mutate (out-of-band or partial
//   content rides along otherwise). An admitted swap proceeds only for
//   exactly one `main` task with `innerHTML` style against the same
//   still-connected node captured at submission.
//
// Every cancellation reports one frozen, sanitized occurrence: finite
// kind/phase/reason enums, the received HTTP status (never a body,
// URL, or header value), and the server-effect uncertainty. Requests
// canceled before sending report effect `none`; anything canceled
// after a response exists reports `uncertain`, since a committed
// write stays committed. Lone inert `main`/`none` tasks need no
// cancellation and report nothing: quiet statuses stay quiet.
//
// The module is portable: zero imports, structural host types, no
// Bun/Node/DOM-library dependency. Served bytes are the TypeScript
// transpilation of this file loaded as a module script after htmx; the
// same source imports directly into Bun tests. Loading self-installs
// exactly once in a DOM host; browser bundles may call
// `installHTMXGuard` with a reporter instead.

export type GuardNode = Readonly<{ isConnected: boolean }>;

export type GuardHeaders = Readonly<{
  has(name: string): boolean;
  keys(): Iterable<string>;
}>;

export type GuardResponse = Readonly<{
  status?: unknown;
  headers?: unknown;
  redirected?: unknown;
}>;

export type GuardContext = Readonly<{
  target?: unknown;
  swap?: unknown;
  response?: unknown;
}>;

export type GuardSwapSpec = Readonly<{ style?: unknown }>;

export type GuardTask = Readonly<{
  type?: unknown;
  target?: unknown;
  swapSpec?: unknown;
}>;

export type GuardEventDetail = Readonly<{ ctx?: unknown; tasks?: unknown }>;

export type GuardEvent = Readonly<{
  type: string;
  detail?: GuardEventDetail | null;
  preventDefault(): void;
}>;

export type GuardHost = Readonly<{
  addEventListener(type: string, listener: (event: GuardEvent) => void): void;
  removeEventListener(type: string, listener: (event: GuardEvent) => void): void;
  dispatchEvent(event: object): boolean;
}>;

export type GuardOccurrenceKind = "action::missing_target" | "action::protocol";

export type GuardOccurrence = Readonly<{
  kind: GuardOccurrenceKind;
  phase: "request" | "response" | "swap";
  status: number | null;
  effect: "none" | "uncertain";
  reason: string;
  header?: string;
}>;

export type GuardReporter = (occurrence: GuardOccurrence) => void;

export type GuardEventFactory = (type: string, detail: GuardOccurrence) => object;

export type GuardOptions = Readonly<{
  report?: GuardReporter;
  createEvent?: GuardEventFactory;
}>;

export const guardOccurrenceEvent = "can:action-occurrence";

const controlPrefix = "hx-";

const knownControlHeaders: Readonly<Record<string, string>> = Object.freeze({
  location: "location",
  "push-url": "push-url",
  redirect: "redirect",
  refresh: "refresh",
  "replace-url": "replace-url",
  reselect: "reselect",
  reswap: "reswap",
  retarget: "retarget",
  trigger: "trigger",
  "trigger-after-swap": "trigger-after-swap",
  "trigger-after-settle": "trigger-after-settle",
});

function isObject(value: unknown): value is Record<string, unknown> {
  return value !== null && (typeof value === "object" || typeof value === "function");
}

function readStatus(response: unknown): number | null {
  const status = isObject(response) ? response["status"] : undefined;
  return typeof status === "number" &&
    Number.isInteger(status) &&
    status >= 100 &&
    status <= 599
    ? status
    : null;
}

function readContext(detail: GuardEventDetail | null | undefined): GuardContext | undefined {
  return detail != null && isObject(detail.ctx)
    ? (detail.ctx as GuardContext)
    : undefined;
}

function controlHeaderName(name: string): string | undefined {
  const lower = name.toLowerCase();
  if (!lower.startsWith(controlPrefix)) return undefined;
  return knownControlHeaders[lower.slice(controlPrefix.length)] ?? "hx-custom";
}

function taskStyle(task: GuardTask): unknown {
  return isObject(task.swapSpec)
    ? (task.swapSpec as GuardSwapSpec).style
    : undefined;
}

export type GuardVerdict = Readonly<{
  admit: boolean;
  occurrence?: GuardOccurrence;
  capture?: GuardNode;
}>;

function occurrence(
  kind: GuardOccurrenceKind,
  phase: GuardOccurrence["phase"],
  status: number | null,
  effect: GuardOccurrence["effect"],
  reason: string,
  header?: string,
): GuardOccurrence {
  return Object.freeze(
    header === undefined
      ? { kind, phase, status, effect, reason }
      : { kind, phase, status, effect, reason, header },
  );
}

// checkRequestTarget admits a submission only for a resolved, connected
// target node, capturing its identity for the later swap comparison. A
// replacement element with the same id is not the original target.
export function checkRequestTarget(ctx: unknown): GuardVerdict {
  if (!isObject(ctx)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "request", null, "none", "unexpected_shape"),
    };
  }
  const target = (ctx as GuardContext).target;
  if (target === null || target === undefined) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::missing_target",
        "request",
        null,
        "none",
        "target_absent",
      ),
    };
  }
  if (!isObject(target)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "request", null, "none", "unexpected_shape"),
    };
  }
  if ((target as GuardNode).isConnected !== true) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::missing_target",
        "request",
        null,
        "none",
        "target_detached",
      ),
    };
  }
  return { admit: true, capture: target as GuardNode };
}

// checkResponseHeaders admits a received response only without
// fetch-level redirection, a still-connected target, and no `hx-*`
// response-control header. Cancellation here precedes any mutation.
export function checkResponseHeaders(ctx: unknown): GuardVerdict {
  if (!isObject(ctx)) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::protocol",
        "response",
        null,
        "uncertain",
        "unexpected_shape",
      ),
    };
  }
  const context = ctx as GuardContext;
  const status = readStatus(context.response);
  const target = context.target;
  if (target === null || target === undefined) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::missing_target",
        "response",
        status,
        "uncertain",
        "target_absent",
      ),
    };
  }
  if (!isObject(target)) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::protocol",
        "response",
        status,
        "uncertain",
        "unexpected_shape",
      ),
    };
  }
  if ((target as GuardNode).isConnected !== true) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::missing_target",
        "response",
        status,
        "uncertain",
        "target_detached",
      ),
    };
  }
  // The pinned asset wraps the fetch response as { raw, status,
  // headers }; the followed-redirect flag lives on the raw response.
  // Either shape carrying it rejects the landing content.
  const raw = isObject(context.response)
    ? (context.response as GuardResponse & { raw?: unknown }).raw
    : undefined;
  if (
    (isObject(context.response) &&
      (context.response as GuardResponse).redirected === true) ||
    (isObject(raw) && (raw as { redirected?: unknown }).redirected === true)
  ) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::protocol",
        "response",
        status,
        "uncertain",
        "redirect",
      ),
    };
  }
  const response = context.response;
  const jar = isObject(response) ? (response as GuardResponse).headers : undefined;
  if (jar === null || jar === undefined) return { admit: true };
  if (!isObject(jar)) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::protocol",
        "response",
        status,
        "uncertain",
        "unexpected_shape",
      ),
    };
  }
  const names: string[] = [];
  const keys = (jar as GuardHeaders).keys;
  if (typeof keys === "function") {
    try {
      for (const name of keys.call(jar) as Iterable<unknown>) {
        if (typeof name === "string") names.push(name);
      }
    } catch {
      return {
        admit: false,
        occurrence: occurrence(
          "action::protocol",
          "response",
          status,
          "uncertain",
          "unexpected_shape",
        ),
      };
    }
  } else if (typeof (jar as GuardHeaders).has === "function") {
    for (const known of Object.keys(knownControlHeaders)) {
      try {
        if ((jar as GuardHeaders).has(`${controlPrefix}${known}`) === true)
          names.push(`${controlPrefix}${known}`);
      } catch {
        return {
          admit: false,
          occurrence: occurrence(
            "action::protocol",
            "response",
            status,
            "uncertain",
            "unexpected_shape",
          ),
        };
      }
    }
  } else {
    return {
      admit: false,
      occurrence: occurrence(
        "action::protocol",
        "response",
        status,
        "uncertain",
        "unexpected_shape",
      ),
    };
  }
  for (const name of names) {
    const header = controlHeaderName(name);
    if (header !== undefined) {
      return {
        admit: false,
        occurrence: occurrence(
          "action::protocol",
          "response",
          status,
          "uncertain",
          "control_header",
          header,
        ),
      };
    }
  }
  return { admit: true };
}

function blockedHazard(tasks: readonly unknown[]): boolean {
  for (const task of tasks) {
    if (!isObject(task)) return true;
    const shaped = task as GuardTask;
    if (shaped.type !== "main" || taskStyle(shaped) !== "none") return true;
  }
  return false;
}

// checkSwapTasks admits exactly one inner `main` task for the same
// still-connected node captured at submission. Blocked failures cancel
// every task when any task could mutate; a lone inert `main`/`none`
// task stays silent because quiet statuses report nothing.
export function checkSwapTasks(
  ctx: unknown,
  tasks: unknown,
  captured: GuardNode | undefined,
): GuardVerdict {
  const context = isObject(ctx) ? (ctx as GuardContext) : undefined;
  const status = context === undefined ? null : readStatus(context.response);
  if (context === undefined || !Array.isArray(tasks)) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::protocol",
        "swap",
        status,
        "uncertain",
        "unexpected_shape",
      ),
    };
  }
  if (context.swap === "none") {
    if (!blockedHazard(tasks)) return { admit: true };
    return {
      admit: false,
      occurrence: occurrence(
        "action::protocol",
        "swap",
        status,
        "uncertain",
        "blocked_tasks",
      ),
    };
  }
  const main =
    tasks.length === 1 && isObject(tasks[0]) ? (tasks[0] as GuardTask) : undefined;
  const same =
    main !== undefined &&
    main.type === "main" &&
    taskStyle(main) === "innerHTML" &&
    captured !== undefined &&
    main.target === captured &&
    captured.isConnected === true;
  if (context.swap === "innerHTML" && same) return { admit: true };
  if (
    main !== undefined &&
    main.type === "main" &&
    (main.target === null ||
      main.target === undefined ||
      (isObject(main.target) && (main.target as GuardNode).isConnected !== true))
  ) {
    return {
      admit: false,
      occurrence: occurrence(
        "action::missing_target",
        "swap",
        status,
        "uncertain",
        main.target === null || main.target === undefined
          ? "target_absent"
          : "target_detached",
      ),
    };
  }
  return {
    admit: false,
    occurrence: occurrence("action::protocol", "swap", status, "uncertain", "task_shape"),
  };
}

type GuardGlobals = Readonly<{
  document?: GuardHost;
  CustomEvent?: new (type: string, init: { detail: GuardOccurrence; bubbles: boolean }) => object;
  __canHtmxGuard?: boolean;
}>;

function defaultEvent(type: string, detail: GuardOccurrence): object | undefined {
  const globals = globalThis as unknown as GuardGlobals;
  if (typeof globals.CustomEvent !== "function") return undefined;
  return new globals.CustomEvent(type, { detail, bubbles: true });
}

// installHTMXGuard wires the three guard positions on the host
// document. It reports every cancellation once through the optional
// reporter and always announces the frozen occurrence as a
// `can:action-occurrence` event when the host can construct one. The
// returned uninstaller removes the listeners; installing twice keeps
// the first installation.
export function installHTMXGuard(host: GuardHost, options: GuardOptions = {}): () => void {
  const globals = globalThis as unknown as { __canHtmxGuard?: boolean };
  if (globals.__canHtmxGuard === true) return () => {};
  globals.__canHtmxGuard = true;
  let targets = new WeakMap<object, GuardNode>();
  const createEvent = options.createEvent;
  const report = options.report;
  const announce = (found: GuardOccurrence): void => {
    if (report !== undefined) report(found);
    const event =
      createEvent !== undefined
        ? createEvent(guardOccurrenceEvent, found)
        : defaultEvent(guardOccurrenceEvent, found);
    if (event !== undefined) host.dispatchEvent(event);
  };
  const guard = (verdict: () => GuardVerdict, event: GuardEvent): void => {
    let decided: GuardVerdict;
    try {
      decided = verdict();
    } catch {
      decided = {
        admit: false,
        occurrence: occurrence(
          "action::protocol",
          event.type === "htmx:before:request"
            ? "request"
            : event.type === "htmx:before:response"
              ? "response"
              : "swap",
          null,
          event.type === "htmx:before:request" ? "none" : "uncertain",
          "unexpected_shape",
        ),
      };
    }
    if (decided.capture !== undefined && isObject(event.detail?.ctx))
      targets.set(event.detail.ctx as object, decided.capture);
    if (decided.admit) return;
    event.preventDefault();
    if (decided.occurrence !== undefined) announce(decided.occurrence);
  };
  const onRequest = (event: GuardEvent): void => {
    guard(() => checkRequestTarget(readContext(event.detail)), event);
  };
  const onResponse = (event: GuardEvent): void => {
    guard(() => checkResponseHeaders(readContext(event.detail)), event);
  };
  const onSwap = (event: GuardEvent): void => {
    guard(() => {
      const ctx = readContext(event.detail);
      const tasks = event.detail?.tasks;
      const captured =
        ctx !== undefined && isObject(ctx) ? targets.get(ctx as object) : undefined;
      return checkSwapTasks(ctx, tasks, captured);
    }, event);
  };
  host.addEventListener("htmx:before:request", onRequest);
  host.addEventListener("htmx:before:response", onResponse);
  host.addEventListener("htmx:before:swap", onSwap);
  let installed = true;
  return () => {
    if (!installed) return;
    installed = false;
    host.removeEventListener("htmx:before:request", onRequest);
    host.removeEventListener("htmx:before:response", onResponse);
    host.removeEventListener("htmx:before:swap", onSwap);
    targets = new WeakMap<object, GuardNode>();
    globals.__canHtmxGuard = false;
  };
}

const autoGlobals = globalThis as unknown as GuardGlobals;
if (
  autoGlobals.document !== undefined &&
  typeof autoGlobals.document.addEventListener === "function" &&
  typeof autoGlobals.document.removeEventListener === "function" &&
  typeof autoGlobals.document.dispatchEvent === "function"
) {
  installHTMXGuard(autoGlobals.document);
}
