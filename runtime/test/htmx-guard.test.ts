import { test, expect, afterAll } from "bun:test";
import { createHash } from "node:crypto";
import { chromium, type Browser, type Page } from "playwright";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, type FailureShape } from "../domain.ts";
import { createHTML, renderSafe } from "../platform/html.ts";
import { value } from "../completion.ts";
import {
  checkRequestTarget,
  checkResponseHeaders,
  checkSwapTasks,
  installHTMXGuard,
  guardOccurrenceEvent,
  type GuardHost,
  type GuardEvent,
  type GuardNode,
  type GuardOccurrence,
} from "../platform/htmx-guard.ts";

// Page-callback typing only: Playwright serializes the closures below
// into Chromium, so these ambient names emit nothing. The runtime
// tsconfig carries no DOM library; the structural shapes are exactly
// what the callbacks touch.
declare const window: {
  occurrences: GuardOccurrence[];
  stayed?: boolean;
};
declare const document: {
  querySelector(selector: string): { textContent: string | null; remove(): void } | null;
  addEventListener(type: string, listener: (event: { detail: unknown }) => void): void;
  activeElement: { id: string } | null;
};

// Pure decision tests: the guard verdicts run over plain data, so every
// rule is pinned without a browser. The pinned-asset section below
// proves the same verdicts against real htmx behavior in Chromium.

const node = (connected: boolean): GuardNode => ({ isConnected: connected });

test("before request admits a connected target and captures its identity", () => {
  const target = node(true);
  const verdict = checkRequestTarget({ target, swap: "innerHTML" });
  expect(verdict.admit).toBe(true);
  expect(verdict.capture).toBe(target);
  expect(verdict.occurrence).toBeUndefined();
});

test("before request reports an absent target with no HTTP outcome", () => {
  for (const target of [null, undefined]) {
    const verdict = checkRequestTarget({ target, swap: "innerHTML" });
    expect(verdict.admit).toBe(false);
    expect(verdict.capture).toBeUndefined();
    expect(verdict.occurrence).toEqual({
      kind: "action::missing_target",
      phase: "request",
      status: null,
      effect: "none",
      reason: "target_absent",
    });
  }
});

test("before request reports a detached target with no HTTP outcome", () => {
  const verdict = checkRequestTarget({ target: node(false), swap: "innerHTML" });
  expect(verdict.admit).toBe(false);
  expect(verdict.occurrence).toEqual({
    kind: "action::missing_target",
    phase: "request",
    status: null,
    effect: "none",
    reason: "target_detached",
  });
});

test("before request fails closed on malformed context", () => {
  for (const ctx of [null, undefined, 42, "ctx", { target: "#target" }]) {
    const verdict = checkRequestTarget(ctx);
    expect(verdict.admit).toBe(false);
    expect(verdict.occurrence?.kind).toBe("action::protocol");
    expect(verdict.occurrence?.phase).toBe("request");
    expect(verdict.occurrence?.effect).toBe("none");
    expect(verdict.occurrence?.reason).toBe("unexpected_shape");
  }
});

test("before response admits a clean response for a connected target", () => {
  const verdict = checkResponseHeaders({
    target: node(true),
    swap: "innerHTML",
    response: { status: 200, headers: new Headers({ "content-type": "text/html" }) },
  });
  expect(verdict).toEqual({ admit: true });
});

test("before response admits a missing headers object for the swap backstop", () => {
  for (const response of [{ status: 200 }, { status: 503 }, undefined]) {
    const verdict = checkResponseHeaders({ target: node(true), response });
    expect(verdict).toEqual({ admit: true });
  }
});

test("before response rejects every response-control header by finite name", () => {
  const cases: [string, string][] = [
    ["HX-Location", "location"],
    ["HX-Push-Url", "push-url"],
    ["HX-Redirect", "redirect"],
    ["HX-Refresh", "refresh"],
    ["HX-Replace-Url", "replace-url"],
    ["HX-Reselect", "reselect"],
    ["HX-Reswap", "reswap"],
    ["HX-Retarget", "retarget"],
    ["HX-Trigger", "trigger"],
    ["HX-Trigger-After-Swap", "trigger-after-swap"],
    ["HX-Trigger-After-Settle", "trigger-after-settle"],
    ["HX-Custom-Future", "hx-custom"],
  ];
  for (const [name, header] of cases) {
    const verdict = checkResponseHeaders({
      target: node(true),
      response: { status: 200, headers: new Headers([[name, "canary-value"]]) },
    });
    expect(verdict.admit).toBe(false);
    expect(verdict.occurrence).toEqual({
      kind: "action::protocol",
      phase: "response",
      status: 200,
      effect: "uncertain",
      reason: "control_header",
      header,
    });
    expect(JSON.stringify(verdict.occurrence)).not.toContain("canary");
  }
});

test("before response rejects fetch-level redirects with status intact", () => {
  for (const response of [
    { status: 200, redirected: true, headers: new Headers() },
    { status: 200, raw: { redirected: true }, headers: new Headers() },
  ]) {
    const verdict = checkResponseHeaders({ target: node(true), response });
    expect(verdict.admit).toBe(false);
    expect(verdict.occurrence).toEqual({
      kind: "action::protocol",
      phase: "response",
      status: 200,
      effect: "uncertain",
      reason: "redirect",
    });
  }
  const direct = checkResponseHeaders({
    target: node(true),
    response: { status: 200, raw: { redirected: false }, headers: new Headers() },
  });
  expect(direct).toEqual({ admit: true });
});

test("before response reports an in-flight target loss with status intact", () => {
  const gone = checkResponseHeaders({
    target: node(false),
    response: { status: 503, headers: new Headers() },
  });
  expect(gone.admit).toBe(false);
  expect(gone.occurrence).toEqual({
    kind: "action::missing_target",
    phase: "response",
    status: 503,
    effect: "uncertain",
    reason: "target_detached",
  });
  const absent = checkResponseHeaders({ target: null, response: { status: 422 } });
  expect(absent.occurrence).toEqual({
    kind: "action::missing_target",
    phase: "response",
    status: 422,
    effect: "uncertain",
    reason: "target_absent",
  });
});

test("before swap admits exactly one inner main task for the same node", () => {
  const target = node(true);
  const verdict = checkSwapTasks(
    { target, swap: "innerHTML", response: { status: 200 } },
    [{ type: "main", target, swapSpec: { style: "innerHTML" }, sourceElement: {} }],
    target,
  );
  expect(verdict).toEqual({ admit: true });
});

test("before swap leaves a lone inert blocked task silent", () => {
  for (const status of [204, 304, 422, 500]) {
    const target = node(true);
    const verdict = checkSwapTasks(
      { target, swap: "none", response: { status } },
      [{ type: "main", target, swapSpec: { style: "none" }, sourceElement: {} }],
      target,
    );
    expect(verdict).toEqual({ admit: true });
  }
});

test("before swap cancels blocked failures that carry mutable tasks", () => {
  const target = node(true);
  const other = node(true);
  const verdict = checkSwapTasks(
    { target, swap: "none", response: { status: 500 } },
    [
      { type: "main", target, swapSpec: { style: "none" }, sourceElement: {} },
      { type: "oob", target: other, swapSpec: { style: "innerHTML" }, sourceElement: {} },
    ],
    target,
  );
  expect(verdict.admit).toBe(false);
  expect(verdict.occurrence).toEqual({
    kind: "action::protocol",
    phase: "swap",
    status: 500,
    effect: "uncertain",
    reason: "blocked_tasks",
  });
});

test("before swap rejects out-of-band and partial tasks in admitted swaps", () => {
  const target = node(true);
  const other = node(true);
  for (const extra of ["oob", "partial"]) {
    const verdict = checkSwapTasks(
      { target, swap: "innerHTML", response: { status: 503 } },
      [
        { type: "main", target, swapSpec: { style: "innerHTML" }, sourceElement: {} },
        { type: extra, target: other, swapSpec: { style: "innerHTML" }, sourceElement: {} },
      ],
      target,
    );
    expect(verdict.admit).toBe(false);
    expect(verdict.occurrence).toEqual({
      kind: "action::protocol",
      phase: "swap",
      status: 503,
      effect: "uncertain",
      reason: "task_shape",
    });
  }
});

test("before swap rejects retargeted, reswapped, and extra tasks", () => {
  const target = node(true);
  const other = node(true);
  const retargeted = checkSwapTasks(
    { target: undefined, swap: "innerHTML", response: { status: 200 } },
    [{ type: "main", target: other, swapSpec: { style: "innerHTML" }, sourceElement: {} }],
    target,
  );
  expect(retargeted.occurrence?.reason).toBe("task_shape");
  const reswapped = checkSwapTasks(
    { target, swap: "outerHTML", response: { status: 200 } },
    [{ type: "main", target, swapSpec: { style: "outerHTML" }, sourceElement: {} }],
    target,
  );
  expect(reswapped.admit).toBe(false);
  expect(reswapped.occurrence?.reason).toBe("task_shape");
  const doubled = checkSwapTasks(
    { target, swap: "innerHTML", response: { status: 200 } },
    [
      { type: "main", target, swapSpec: { style: "innerHTML" }, sourceElement: {} },
      { type: "main", target, swapSpec: { style: "innerHTML" }, sourceElement: {} },
    ],
    target,
  );
  expect(doubled.admit).toBe(false);
  expect(doubled.occurrence?.reason).toBe("task_shape");
});

test("before swap reports a detached swap node as a missing target", () => {
  const target = node(false);
  const verdict = checkSwapTasks(
    { target, swap: "innerHTML", response: { status: 200 } },
    [{ type: "main", target, swapSpec: { style: "innerHTML" }, sourceElement: {} }],
    target,
  );
  expect(verdict.admit).toBe(false);
  expect(verdict.occurrence).toEqual({
    kind: "action::missing_target",
    phase: "swap",
    status: 200,
    effect: "uncertain",
    reason: "target_detached",
  });
});

test("before swap fails closed without a captured submission node", () => {
  const target = node(true);
  const verdict = checkSwapTasks(
    { target, swap: "innerHTML", response: { status: 200 } },
    [{ type: "main", target, swapSpec: { style: "innerHTML" }, sourceElement: {} }],
    undefined,
  );
  expect(verdict.admit).toBe(false);
  expect(verdict.occurrence?.reason).toBe("task_shape");
});

test("before swap fails closed on malformed tasks", () => {
  const target = node(true);
  for (const tasks of [null, undefined, {}, "tasks"]) {
    const verdict = checkSwapTasks(
      { target, swap: "innerHTML", response: { status: 200 } },
      tasks,
      target,
    );
    expect(verdict.admit).toBe(false);
    expect(verdict.occurrence?.reason).toBe("unexpected_shape");
  }
  for (const tasks of [[null], [42], [{}]]) {
    const verdict = checkSwapTasks(
      { target, swap: "innerHTML", response: { status: 200 } },
      tasks,
      target,
    );
    expect(verdict.admit).toBe(false);
    expect(verdict.occurrence?.reason).toBe("task_shape");
  }
  const noContext = checkSwapTasks(null, [], target);
  expect(noContext.admit).toBe(false);
  expect(noContext.occurrence?.reason).toBe("unexpected_shape");
});

test("non-integer or out-of-range statuses read back as unknown", () => {
  for (const status of ["200", 20, 99, 600, NaN, 200.5, {}, null]) {
    const verdict = checkResponseHeaders({
      target: node(true),
      response: { status, redirected: true },
    });
    expect(verdict.occurrence?.status).toBeNull();
  }
});

test("occurrences are frozen and carry only finite vocabulary", () => {
  const verdict = checkSwapTasks(
    { target: node(true), swap: "innerHTML", response: { status: 200 } },
    [],
    node(true),
  );
  const found = verdict.occurrence!;
  expect(Object.isFrozen(found)).toBe(true);
  expect(Object.keys(found).sort()).toEqual(["effect", "kind", "phase", "reason", "status"]);
  expect(() => JSON.stringify(found)).not.toThrow();
});

function fakeHost(): GuardHost & { listeners(type: string): ((e: GuardEvent) => void)[] } {
  const table = new Map<string, ((event: GuardEvent) => void)[]>();
  const events: object[] = [];
  return {
    addEventListener(type, listener) {
      table.set(type, [...(table.get(type) ?? []), listener]);
    },
    removeEventListener(type, listener) {
      table.set(
        type,
        (table.get(type) ?? []).filter((entry) => entry !== listener),
      );
    },
    dispatchEvent: (event: object) => {
      events.push(event);
      return true;
    },
    listeners: (type: string) => table.get(type) ?? [],
  };
}

function fakeEvent(
  type: string,
  detail?: GuardEvent["detail"],
): GuardEvent & { canceled: boolean } {
  return {
    type,
    detail,
    canceled: false,
    preventDefault() {
      this.canceled = true;
    },
  };
}

test("installing wires all three positions and uninstalling removes them", () => {
  const host = fakeHost();
  const reports: GuardOccurrence[] = [];
  const created: { type: string; detail: GuardOccurrence }[] = [];
  const uninstall = installHTMXGuard(host, {
    report: (found) => reports.push(found),
    createEvent: (type, detail) => {
      created.push({ type, detail });
      return { type, detail };
    },
  });
  try {
    expect(host.listeners("htmx:before:request")).toHaveLength(1);
    expect(host.listeners("htmx:before:response")).toHaveLength(1);
    expect(host.listeners("htmx:before:swap")).toHaveLength(1);
    const ctx = { target: node(true), swap: "innerHTML" };
    const request = fakeEvent("htmx:before:request", { ctx });
    host.listeners("htmx:before:request")[0]!(request);
    expect(request.canceled).toBe(false);
    const swap = fakeEvent("htmx:before:swap", {
      ctx: { ...ctx, response: { status: 200 } },
      tasks: [
        {
          type: "main",
          target: (ctx as { target: GuardNode }).target,
          swapSpec: { style: "innerHTML" },
        },
      ],
    });
    // A fresh context object is not the captured submission: fail closed.
    host.listeners("htmx:before:swap")[0]!(swap);
    expect(swap.canceled).toBe(true);
    expect(reports).toHaveLength(1);
    expect(created).toEqual([{ type: guardOccurrenceEvent, detail: reports[0] }]);
  } finally {
    uninstall();
  }
  expect(host.listeners("htmx:before:request")).toHaveLength(0);
  expect(host.listeners("htmx:before:response")).toHaveLength(0);
  expect(host.listeners("htmx:before:swap")).toHaveLength(0);
});

test("a second installation keeps the first and reports once", () => {
  const first = fakeHost();
  const second = fakeHost();
  const reports: GuardOccurrence[] = [];
  const uninstall = installHTMXGuard(first, { report: (found) => reports.push(found) });
  try {
    const extra = installHTMXGuard(second, { report: (found) => reports.push(found) });
    try {
      expect(second.listeners("htmx:before:request")).toHaveLength(0);
      const event = fakeEvent("htmx:before:request", { ctx: { target: null } });
      first.listeners("htmx:before:request")[0]!(event);
      expect(event.canceled).toBe(true);
      expect(reports).toHaveLength(1);
    } finally {
      extra();
    }
  } finally {
    uninstall();
  }
});

test("a throwing host read cancels fail-closed with one occurrence", () => {
  const host = fakeHost();
  const reports: GuardOccurrence[] = [];
  const uninstall = installHTMXGuard(host, { report: (found) => reports.push(found) });
  try {
    const bomb = {
      get target(): unknown {
        throw new Error("host read failed");
      },
    };
    const event = fakeEvent("htmx:before:request", { ctx: bomb });
    host.listeners("htmx:before:request")[0]!(event);
    expect(event.canceled).toBe(true);
    expect(reports).toHaveLength(1);
    expect(reports[0]?.reason).toBe("unexpected_shape");
  } finally {
    uninstall();
  }
});

// Pinned-asset browser proof: the real page head markup from
// html::runtime_head (config policy plus both script tags) drives the
// pinned htmx bytes and the served guard bytes over real HTTP in
// Chromium. Pages below are fixtures; only the head markup is under
// test here. The hx-status attributes stand in for checked action
// emission, which no integrated task supplies yet.

let pinnedBrowser: Browser | undefined;
try {
  pinnedBrowser = await chromium.launch();
} catch {
  console.log(
    "SKIP pinned htmx guard browser tests: chromium is unavailable " +
      "(run `bunx playwright install chromium` to qualify the guard)",
  );
}
const pinned = pinnedBrowser === undefined ? test.skip : test;
if (pinnedBrowser !== undefined)
  console.log(`pinned htmx guard tests on ${pinnedBrowser.version()}`);

type BrowserShape = FailureShape;
const typeHash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const typeShape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): BrowserShape => ({
  identity: typeHash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const primitiveStr = typeShape("primitive", "str");
const primitiveInt = typeShape("primitive", "int");
const guardDeclarations = catalogue.errors.filter((e) =>
  [
    "html::invalid_structure",
    "html::invalid_url",
    "htmx::invalid_target",
    "htmx::invalid_interval",
  ].includes(e.name),
);
const guardErrors = guardDeclarations.map((e) =>
  typeShape(
    "error",
    e.identity,
    e.fields.map((f) => ({
      name: f.name,
      type: f.type === "int" ? primitiveInt.identity : primitiveStr.identity,
    })),
  ),
);
const guardDomain = createDomainRuntime({
  declarations: guardDeclarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [primitiveStr, primitiveInt, ...guardErrors],
});
const guardHTML = createHTML(guardDomain, {
  structure: guardErrors[0]!.identity,
  url: guardErrors[1]!.identity,
  target: guardErrors[2]!.identity,
  interval: guardErrors[3]!.identity,
});

const htmxBytes = await Bun.file(
  new URL("../../distribution/assets/htmx-4.0.0.min.js", import.meta.url),
).bytes();
const htmxLock = (await Bun.file(
  new URL("../../distribution/assets/htmx.lock.json", import.meta.url),
).json()) as { sha256: string; integrity: string };
const guardSource = await Bun.file(new URL("../platform/htmx-guard.ts", import.meta.url)).text();
const guardBytes = new TextEncoder().encode(
  new Bun.Transpiler({ loader: "ts" }).transformSync(guardSource),
);
const suiteHead =
  renderSafe(
    value(
      await guardHTML.document(
        "guard",
        [value(await guardHTML.metaViewport()), value(await guardHTML.runtimeHead())],
        [],
      ),
    ),
  ).match(/<head>(.*)<\/head>/)?.[1] ?? "";

type Reply = Readonly<{
  status: number;
  headers?: Record<string, string>;
  body: string;
  gate?: Promise<void>;
  journalEntry?: string;
}>;
type SeenRequest = Readonly<{ method: string; path: string }>;
const seen: SeenRequest[] = [];
const journal: string[] = [];
let pageBody = "";
let reply: Reply = { status: 200, body: "" };
const suiteServer = Bun.serve({
  hostname: "127.0.0.1",
  port: 0,
  async fetch(request) {
    const url = new URL(request.url);
    seen.push({ method: request.method, path: url.pathname + url.search });
    if (url.pathname === "/__can/assets/htmx-4.0.0.min.js")
      return new Response(htmxBytes, { headers: { "content-type": "text/javascript" } });
    if (url.pathname === "/__can/assets/htmx-guard.js")
      return new Response(guardBytes, { headers: { "content-type": "text/javascript" } });
    if (url.pathname === "/page")
      return new Response(
        `<!doctype html><html><head>${suiteHead}</head><body>${pageBody}</body></html>`,
        { headers: { "content-type": "text/html" } },
      );
    if (url.pathname === "/elsewhere") return new Response("redirect landing");
    if (url.pathname === "/reply") {
      const current = reply;
      if (current.journalEntry !== undefined) journal.push(current.journalEntry);
      if (current.gate !== undefined) await current.gate;
      return new Response(current.body, { status: current.status, headers: current.headers });
    }
    return new Response("missing", { status: 404 });
  },
});
const suiteBase = `http://127.0.0.1:${suiteServer.port}`;

afterAll(async () => {
  await suiteServer.stop(true);
  await pinnedBrowser?.close();
});

type PinnedPage = Readonly<{
  page: Page;
  occurrences: () => Promise<GuardOccurrence[]>;
  errors: string[];
}>;
async function openPinnedPage(body: string): Promise<PinnedPage> {
  pageBody = body;
  const page = await pinnedBrowser!.newPage();
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(String(error?.message ?? error)));
  await page.goto(`${suiteBase}/page`, { waitUntil: "load" });
  await page.evaluate(() => {
    window.occurrences = [];
    document.addEventListener("can:action-occurrence", (event) =>
      window.occurrences.push(event.detail as GuardOccurrence),
    );
  });
  return {
    page,
    errors,
    occurrences: () => page.evaluate(() => window.occurrences),
  };
}

async function awaitOccurrences(pinned: PinnedPage, count: number): Promise<GuardOccurrence[]> {
  await pinned.page.waitForFunction((want) => window.occurrences.length >= want, count, {
    timeout: 10000,
  });
  return pinned.occurrences();
}

const declaredSource = (id: string) =>
  `<button id="${id}" hx-post="/reply" hx-target="#target" hx-swap="innerHTML" hx-status:422='{"swap":"innerHTML"}' hx-status:409='{"swap":"innerHTML"}' hx-status:403='{"swap":"innerHTML"}' hx-status:503='{"swap":"innerHTML"}'>go</button>`;
const plainSource = (id: string) =>
  `<button id="${id}" hx-post="/reply" hx-target="#target" hx-swap="innerHTML">go</button>`;
const targets = `<div id="target">old target</div><div id="other">old other</div>`;
const oobBody = (main: string) =>
  `<p>${main}</p><div id="other" hx-swap-oob="innerHTML">oob intruder</div>`;
const partialBody = (main: string) =>
  `<p>${main}</p><template hx type="partial" hx-target="#other"><p>partial intruder</p></template>`;

pinned(
  "all five admitted statuses swap inner into the admitted target",
  async () => {
    for (const status of [200, 422, 409, 403, 503]) {
      reply = { status, body: `<p>fresh ${status}</p>` };
      const screen = await openPinnedPage(`${declaredSource("source")}${targets}`);
      try {
        await screen.page.locator("#source").click();
        await screen.page.waitForFunction(
          (want) => document.querySelector("#target")?.textContent === want,
          `fresh ${status}`,
          { timeout: 10000 },
        );
        expect(await screen.page.locator("#other").textContent()).toBe("old other");
        expect(await screen.occurrences()).toEqual([]);
        expect(screen.errors).toEqual([]);
      } finally {
        await screen.page.close();
      }
    }
  },
  { timeout: 60000 },
);

pinned(
  "undeclared failures on plain and declared sources swap nothing and stay silent",
  async () => {
    const matrix: [string, string, number[]][] = [
      ["plain", plainSource("source"), [422, 409, 403, 503, 500]],
      ["declared", declaredSource("source"), [500, 502]],
    ];
    for (const [name, source, statuses] of matrix) {
      for (const status of statuses) {
        reply = { status, body: `<p>${name} ${status}</p>` };
        const screen = await openPinnedPage(`${source}${targets}`);
        try {
          await screen.page.locator("#source").click();
          await screen.page.waitForTimeout(400);
          expect(await screen.page.locator("#target").textContent()).toBe("old target");
          expect(await screen.page.locator("#other").textContent()).toBe("old other");
          expect(await screen.occurrences()).toEqual([]);
          expect(screen.errors).toEqual([]);
        } finally {
          await screen.page.close();
        }
      }
    }
  },
  { timeout: 60000 },
);

pinned(
  "a missing target before request sends nothing and reports effect none",
  async () => {
    seen.length = 0;
    reply = { status: 200, body: "<p>fresh</p>" };
    const screen = await openPinnedPage(
      `${declaredSource("source")}<div id="other">old other</div>`,
    );
    try {
      await screen.page.locator("#source").click();
      const found = await awaitOccurrences(screen, 1);
      expect(found).toEqual([
        {
          kind: "action::missing_target",
          phase: "request",
          status: null,
          effect: "none",
          reason: "target_absent",
        },
      ]);
      expect(seen.filter((entry) => entry.path === "/reply")).toEqual([]);
      expect(seen.some((entry) => entry.path === "/page")).toBe(true);
      expect(await screen.page.locator("#other").textContent()).toBe("old other");
      expect(screen.errors).toEqual([]);
    } finally {
      await screen.page.close();
    }
  },
  { timeout: 30000 },
);

pinned(
  "a target lost in flight reports its status with uncertain effect",
  async () => {
    let release!: () => void;
    const gate = new Promise<void>((done) => {
      release = done;
    });
    reply = { status: 200, body: "<p>fresh</p>", gate };
    const screen = await openPinnedPage(`${declaredSource("source")}${targets}`);
    try {
      seen.length = 0;
      await screen.page.locator("#source").click();
      const replies = () => seen.filter((entry) => entry.path === "/reply");
      for (let waited = 0; replies().length === 0 && waited < 10000; waited += 50)
        await Bun.sleep(50);
      expect(replies()).toHaveLength(1);
      await screen.page.evaluate(() => document.querySelector("#target")?.remove());
      release();
      const found = await awaitOccurrences(screen, 1);
      expect(found).toHaveLength(1);
      expect(found[0]?.kind).toBe("action::missing_target");
      expect(found[0]?.status).toBe(200);
      expect(found[0]?.effect).toBe("uncertain");
      expect(await screen.page.locator("#other").textContent()).toBe("old other");
      expect(screen.errors).toEqual([]);
    } finally {
      await screen.page.close();
    }
  },
  { timeout: 30000 },
);

pinned(
  "response-control headers cancel before any mutation",
  async () => {
    const cases: [string, Record<string, string>, string][] = [
      ["redirect", { "HX-Redirect": "/elsewhere" }, "redirect"],
      ["location", { "HX-Location": "/elsewhere" }, "location"],
      ["refresh", { "HX-Refresh": "true" }, "refresh"],
      ["retarget", { "HX-Retarget": "#other" }, "retarget"],
      ["reswap", { "HX-Reswap": "outerHTML" }, "reswap"],
      ["reselect", { "HX-Reselect": "#other" }, "reselect"],
      ["push-url", { "HX-Push-Url": "/elsewhere" }, "push-url"],
      ["replace-url", { "HX-Replace-Url": "/elsewhere" }, "replace-url"],
      ["trigger", { "HX-Trigger": "intruder" }, "trigger"],
    ];
    for (const [name, headers, header] of cases) {
      seen.length = 0;
      reply = { status: 200, headers, body: "<p>fresh</p>" };
      const screen = await openPinnedPage(`${declaredSource("source")}${targets}`);
      try {
        await screen.page.evaluate(() => {
          window.stayed = true;
        });
        await screen.page.locator("#source").click();
        const found = await awaitOccurrences(screen, 1);
        expect(found).toEqual([
          {
            kind: "action::protocol",
            phase: "response",
            status: 200,
            effect: "uncertain",
            reason: "control_header",
            header,
          },
        ]);
        expect(await screen.page.locator("#target").textContent(), name).toBe("old target");
        expect(await screen.page.locator("#other").textContent(), name).toBe("old other");
        expect(screen.page.url(), name).toBe(`${suiteBase}/page`);
        expect(await screen.page.evaluate(() => window.stayed), name).toBe(true);
        expect(
          seen.some((entry) => entry.path === "/elsewhere"),
          `${name} requested elsewhere`,
        ).toBe(false);
        expect(screen.errors).toEqual([]);
      } finally {
        await screen.page.close();
      }
    }
  },
  { timeout: 90000 },
);

pinned(
  "a fetch-level redirect never swaps its landing content",
  async () => {
    seen.length = 0;
    reply = { status: 302, headers: { location: "/elsewhere" }, body: "" };
    const screen = await openPinnedPage(`${declaredSource("source")}${targets}`);
    try {
      await screen.page.locator("#source").click();
      const found = await awaitOccurrences(screen, 1);
      expect(found).toEqual([
        {
          kind: "action::protocol",
          phase: "response",
          status: 200,
          effect: "uncertain",
          reason: "redirect",
        },
      ]);
      expect(seen.some((entry) => entry.path === "/elsewhere")).toBe(true);
      expect(await screen.page.locator("#target").textContent()).toBe("old target");
      expect(await screen.page.locator("#other").textContent()).toBe("old other");
      expect(screen.page.url()).toBe(`${suiteBase}/page`);
      expect(screen.errors).toEqual([]);
    } finally {
      await screen.page.close();
    }
  },
  { timeout: 30000 },
);

pinned(
  "out-of-band and partial tasks cancel in every admitted status",
  async () => {
    const shapes: [string, (main: string) => string][] = [
      ["oob", oobBody],
      ["partial", partialBody],
    ];
    for (const status of [200, 422, 409, 403, 503]) {
      for (const [shape, shapeBody] of shapes) {
        reply = { status, body: shapeBody(`fresh ${status}`) };
        const screen = await openPinnedPage(`${declaredSource("source")}${targets}`);
        try {
          await screen.page.locator("#source").click();
          const found = await awaitOccurrences(screen, 1);
          expect(found).toEqual([
            {
              kind: "action::protocol",
              phase: "swap",
              status,
              effect: "uncertain",
              reason: "task_shape",
            },
          ]);
          expect(await screen.page.locator("#target").textContent(), `${status} ${shape}`).toBe(
            "old target",
          );
          expect(await screen.page.locator("#other").textContent(), `${status} ${shape}`).toBe(
            "old other",
          );
          expect(
            await screen.page.evaluate(() => document.activeElement?.id),
            `${status} ${shape} focus`,
          ).toBe("source");
          expect(screen.errors).toEqual([]);
        } finally {
          await screen.page.close();
        }
      }
    }
  },
  { timeout: 120000 },
);

pinned(
  "an undeclared failure with out-of-band content cancels every task",
  async () => {
    reply = { status: 500, body: oobBody("fresh") };
    const screen = await openPinnedPage(`${plainSource("source")}${targets}`);
    try {
      await screen.page.locator("#source").click();
      const found = await awaitOccurrences(screen, 1);
      expect(found).toEqual([
        {
          kind: "action::protocol",
          phase: "swap",
          status: 500,
          effect: "uncertain",
          reason: "blocked_tasks",
        },
      ]);
      expect(await screen.page.locator("#target").textContent()).toBe("old target");
      expect(await screen.page.locator("#other").textContent()).toBe("old other");
      expect(screen.errors).toEqual([]);
    } finally {
      await screen.page.close();
    }
  },
  { timeout: 30000 },
);

pinned(
  "a committed write stays committed through render and swap failures",
  async () => {
    journal.length = 0;
    reply = { status: 500, headers: {}, body: oobBody("render fault"), journalEntry: "render" };
    const render = await openPinnedPage(`${plainSource("source")}${targets}`);
    try {
      await render.page.locator("#source").click();
      const found = await awaitOccurrences(render, 1);
      expect(found[0]?.effect).toBe("uncertain");
      expect(found[0]?.status).toBe(500);
      expect(journal).toEqual(["render"]);
      expect(await render.page.locator("#target").textContent()).toBe("old target");
      expect(await render.page.locator("#other").textContent()).toBe("old other");
      expect(render.errors).toEqual([]);
    } finally {
      await render.page.close();
    }
    reply = {
      status: 200,
      body: partialBody("swap fault"),
      journalEntry: "swap",
    };
    const swap = await openPinnedPage(`${declaredSource("source")}${targets}`);
    try {
      await swap.page.locator("#source").click();
      const found = await awaitOccurrences(swap, 1);
      expect(found).toEqual([
        {
          kind: "action::protocol",
          phase: "swap",
          status: 200,
          effect: "uncertain",
          reason: "task_shape",
        },
      ]);
      expect(journal).toEqual(["render", "swap"]);
      expect(await swap.page.locator("#target").textContent()).toBe("old target");
      expect(await swap.page.locator("#other").textContent()).toBe("old other");
      expect(swap.errors).toEqual([]);
    } finally {
      await swap.page.close();
    }
  },
  { timeout: 60000 },
);

pinned(
  "occurrences never carry bodies, header values, or urls",
  async () => {
    reply = {
      status: 200,
      headers: { "HX-Retarget": "#canary-selector-value" },
      body: "<p>canary-body-value</p>",
    };
    const screen = await openPinnedPage(`${declaredSource("source")}${targets}`);
    try {
      await screen.page.locator("#source").click();
      const found = await awaitOccurrences(screen, 1);
      expect(found).toEqual([
        {
          kind: "action::protocol",
          phase: "response",
          status: 200,
          effect: "uncertain",
          reason: "control_header",
          header: "retarget",
        },
      ]);
      const serialized = JSON.stringify(found);
      expect(serialized).not.toContain("canary");
      expect(serialized).not.toContain("#target");
      expect(serialized).not.toContain("/reply");
      expect(screen.errors).toEqual([]);
    } finally {
      await screen.page.close();
    }
  },
  { timeout: 30000 },
);

pinned(
  "served vendor and guard bytes match their pins",
  async () => {
    const servedHtmx = new Uint8Array(
      await (await fetch(`${suiteBase}/__can/assets/htmx-4.0.0.min.js`)).arrayBuffer(),
    );
    expect(servedHtmx).toEqual(htmxBytes);
    expect(createHash("sha256").update(servedHtmx).digest("hex")).toBe(htmxLock.sha256);
    const servedGuard = new Uint8Array(
      await (await fetch(`${suiteBase}/__can/assets/htmx-guard.js`)).arrayBuffer(),
    );
    expect(servedGuard).toEqual(guardBytes);
    const digest = await crypto.subtle.digest("SHA-384", servedGuard);
    let binary = "";
    for (const byte of new Uint8Array(digest)) binary += String.fromCharCode(byte);
    const guardTag = suiteHead.match(
      /<script type="module" src="\/__can\/assets\/htmx-guard\.js" integrity="(.*?)"><\/script>/,
    );
    expect(guardTag?.[1]).toBeString();
    expect(`sha384-${btoa(binary)}`).toBe(guardTag![1]);
    const htmxIndex = suiteHead.indexOf("/__can/assets/htmx-4.0.0.min.js");
    const guardIndex = suiteHead.indexOf("/__can/assets/htmx-guard.js");
    expect(htmxIndex).toBeGreaterThanOrEqual(0);
    expect(guardIndex).toBeGreaterThan(htmxIndex);
  },
  { timeout: 30000 },
);
