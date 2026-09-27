import { createBrowser, createBrowserState } from "../../../runtime/platform/browser.ts";
import {
  createDomainRuntime,
  verifyErrorPlan,
  domainFailureDiagnostics,
} from "../../../runtime/browser/domain.ts";
import { catalogue } from "../../../runtime/catalogue.ts";
import { success, value as completionValue } from "../../../runtime/completion.ts";
async function hash(kind: string, declaration: string) {
  const bytes = await crypto.subtle.digest(
    "SHA-256",
    new TextEncoder().encode("can-concrete-type-v1\0" + JSON.stringify([kind, declaration])),
  );
  return Array.from(new Uint8Array(bytes), (n) => n.toString(16).padStart(2, "0")).join("");
}
const primitives = await Promise.all(
  ["str", "int"].map(async (declaration) => ({
    identity: await hash("primitive", declaration),
    kind: "primitive",
    declaration,
    arguments: [],
    fields: [],
    leaves: [],
    inputs: [],
    errors: [],
  })),
);
const declarations = catalogue.errors.filter((e) => e.name.startsWith("browser::"));
const shapes = await Promise.all(
  declarations.map(async (e) => ({
    identity: await hash("error", e.identity),
    kind: "error",
    declaration: e.identity,
    arguments: [],
    fields: e.fields.map((f) => ({
      name: f.name,
      type: primitives[f.type === "int" ? 1 : 0]!.identity,
    })),
    leaves: [],
    inputs: [],
    errors: [],
  })),
);
const domain = createDomainRuntime(
  await verifyErrorPlan({
    declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
    shapes: [...primitives, ...shapes],
  }),
);
function value(completion: any): any {
  if (completion.kind === "domain") {
    const diagnostics = domainFailureDiagnostics(completion.value);
    throw Error(JSON.stringify(diagnostics, (_key, v) => (typeof v === "bigint" ? String(v) : v)));
  }
  return completionValue(completion);
}
const errorId = (name: string) =>
  shapes[declarations.findIndex((e) => e.name === `browser::${name}`)]!.identity;
const browser = createBrowser(domain, {
  missingRoot: errorId("missing_root"),
  disposed: errorId("disposed"),
  rejected: errorId("rejected"),
  invalidQuery: errorId("invalid_query"),
  event: "perf:event",
  some: "perf:some",
  none: "perf:none",
  modifiers: "perf:modifiers",
  selection: "perf:selection",
  file: "perf:file",
});
const app = value(await browser.mount("app"));
const view = value(await browser.openView(app));
const root = value(await browser.root(app));
const input = value(await browser.createElement(view, "input"));
value(await browser.setAttribute(input, "id", "field"));
value(await browser.setAttribute(input, "type", "text"));
value(await browser.appendChild(root, input));
const output = value(await browser.createElement(view, "span"));
value(await browser.setAttribute(output, "id", "result"));
value(await browser.appendChild(root, output));
const stateRuntime = createBrowserState(domain, {
  disposed: errorId("disposed"),
  stale: errorId("stale_version"),
  state: "perf:state",
  snapshot: "perf:snapshot",
});
const state = value(await stateRuntime.createState(view, ""));
const reset = value(await browser.createElement(view, "button"));
value(await browser.setAttribute(reset, "id", "reset"));
value(await browser.setText(reset, "Reset"));
value(await browser.appendChild(root, reset));
const events: any[] = [];
const resets: any[] = [];
value(
  await browser.onEvent(view, input, "input", async (event: any) => {
    const start = performance.now();
    const text = value(await browser.readValue(input));
    const previous = value(await stateRuntime.readState(state));
    value(await stateRuntime.replaceState(state, previous.version, text));
    value(await browser.setText(output, text));
    events.push({
      duration_ms: performance.now() - start,
      value: text,
      snapshot_value: event.value,
      kind: event.kind,
      composing: event.composing,
      selection_start: String(event.selection.start),
      selection_end: String(event.selection.end),
      frozen_snapshot: Object.isFrozen(event) && Object.isFrozen(event.selection),
    });
    (window as any).completed = events.length;
    return success(undefined);
  }),
);
value(
  await browser.onEvent(view, reset, "click", async () => {
    const start = performance.now();
    const previous = value(await stateRuntime.readState(state));
    value(await stateRuntime.replaceState(state, previous.version, ""));
    value(await browser.setValue(input, ""));
    value(await browser.setSelection(input, 0n, 0n, "none"));
    value(await browser.setText(output, ""));
    const current = value(await stateRuntime.readState(state));
    resets.push({
      duration_ms: performance.now() - start,
      version_step: String(current.version - previous.version),
      state_value: current.value,
      old_snapshot_value: previous.value,
    });
    (window as any).resetCompleted = resets.length;
    return success(undefined);
  }),
);
(window as any).bench = { events, resets, browser, input, view, state, stateRuntime };
(window as any).ready = true;
