export const guardOccurrenceEvent = "can:action-occurrence";
const controlPrefix = "hx-";
const knownControlHeaders = Object.freeze({
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
  "trigger-after-settle": "trigger-after-settle"
});
function isObject(value) {
  return value !== null && (typeof value === "object" || typeof value === "function");
}
function readStatus(response) {
  const status = isObject(response) ? response["status"] : undefined;
  return typeof status === "number" && Number.isInteger(status) && status >= 100 && status <= 599 ? status : null;
}
function readContext(detail) {
  return detail != null && isObject(detail.ctx) ? detail.ctx : undefined;
}
function controlHeaderName(name) {
  const lower = name.toLowerCase();
  if (!lower.startsWith(controlPrefix))
    return;
  return knownControlHeaders[lower.slice(controlPrefix.length)] ?? "hx-custom";
}
function taskStyle(task) {
  return isObject(task.swapSpec) ? task.swapSpec.style : undefined;
}
function occurrence(kind, phase, status, effect, reason, header) {
  return Object.freeze(header === undefined ? { kind, phase, status, effect, reason } : { kind, phase, status, effect, reason, header });
}
export function checkRequestTarget(ctx) {
  if (!isObject(ctx)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "request", null, "none", "unexpected_shape")
    };
  }
  const target = ctx.target;
  if (target === null || target === undefined) {
    return {
      admit: false,
      occurrence: occurrence("action::missing_target", "request", null, "none", "target_absent")
    };
  }
  if (!isObject(target)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "request", null, "none", "unexpected_shape")
    };
  }
  if (target.isConnected !== true) {
    return {
      admit: false,
      occurrence: occurrence("action::missing_target", "request", null, "none", "target_detached")
    };
  }
  return { admit: true, capture: target };
}
export function checkResponseHeaders(ctx) {
  if (!isObject(ctx)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "response", null, "uncertain", "unexpected_shape")
    };
  }
  const context = ctx;
  const status = readStatus(context.response);
  const target = context.target;
  if (target === null || target === undefined) {
    return {
      admit: false,
      occurrence: occurrence("action::missing_target", "response", status, "uncertain", "target_absent")
    };
  }
  if (!isObject(target)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "response", status, "uncertain", "unexpected_shape")
    };
  }
  if (target.isConnected !== true) {
    return {
      admit: false,
      occurrence: occurrence("action::missing_target", "response", status, "uncertain", "target_detached")
    };
  }
  const raw = isObject(context.response) ? context.response.raw : undefined;
  if (isObject(context.response) && context.response.redirected === true || isObject(raw) && raw.redirected === true) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "response", status, "uncertain", "redirect")
    };
  }
  const response = context.response;
  const jar = isObject(response) ? response.headers : undefined;
  if (jar === null || jar === undefined)
    return { admit: true };
  if (!isObject(jar)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "response", status, "uncertain", "unexpected_shape")
    };
  }
  const names = [];
  const keys = jar.keys;
  if (typeof keys === "function") {
    try {
      for (const name of keys.call(jar)) {
        if (typeof name === "string")
          names.push(name);
      }
    } catch {
      return {
        admit: false,
        occurrence: occurrence("action::protocol", "response", status, "uncertain", "unexpected_shape")
      };
    }
  } else if (typeof jar.has === "function") {
    for (const known of Object.keys(knownControlHeaders)) {
      try {
        if (jar.has(`${controlPrefix}${known}`) === true)
          names.push(`${controlPrefix}${known}`);
      } catch {
        return {
          admit: false,
          occurrence: occurrence("action::protocol", "response", status, "uncertain", "unexpected_shape")
        };
      }
    }
  } else {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "response", status, "uncertain", "unexpected_shape")
    };
  }
  for (const name of names) {
    const header = controlHeaderName(name);
    if (header !== undefined) {
      return {
        admit: false,
        occurrence: occurrence("action::protocol", "response", status, "uncertain", "control_header", header)
      };
    }
  }
  return { admit: true };
}
function blockedHazard(tasks) {
  for (const task of tasks) {
    if (!isObject(task))
      return true;
    const shaped = task;
    if (shaped.type !== "main" || taskStyle(shaped) !== "none")
      return true;
  }
  return false;
}
export function checkSwapTasks(ctx, tasks, captured) {
  const context = isObject(ctx) ? ctx : undefined;
  const status = context === undefined ? null : readStatus(context.response);
  if (context === undefined || !Array.isArray(tasks)) {
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "swap", status, "uncertain", "unexpected_shape")
    };
  }
  if (context.swap === "none") {
    if (!blockedHazard(tasks))
      return { admit: true };
    return {
      admit: false,
      occurrence: occurrence("action::protocol", "swap", status, "uncertain", "blocked_tasks")
    };
  }
  const main = tasks.length === 1 && isObject(tasks[0]) ? tasks[0] : undefined;
  const same = main !== undefined && main.type === "main" && taskStyle(main) === "innerHTML" && captured !== undefined && main.target === captured && captured.isConnected === true;
  if (context.swap === "innerHTML" && same)
    return { admit: true };
  if (main !== undefined && main.type === "main" && (main.target === null || main.target === undefined || isObject(main.target) && main.target.isConnected !== true)) {
    return {
      admit: false,
      occurrence: occurrence("action::missing_target", "swap", status, "uncertain", main.target === null || main.target === undefined ? "target_absent" : "target_detached")
    };
  }
  return {
    admit: false,
    occurrence: occurrence("action::protocol", "swap", status, "uncertain", "task_shape")
  };
}
function defaultEvent(type, detail) {
  const globals = globalThis;
  if (typeof globals.CustomEvent !== "function")
    return;
  return new globals.CustomEvent(type, { detail, bubbles: true });
}
export function installHTMXGuard(host, options = {}) {
  const globals = globalThis;
  if (globals.__canHtmxGuard === true)
    return () => {};
  globals.__canHtmxGuard = true;
  let targets = new WeakMap;
  const createEvent = options.createEvent;
  const report = options.report;
  const announce = (found) => {
    if (report !== undefined)
      report(found);
    const event = createEvent !== undefined ? createEvent(guardOccurrenceEvent, found) : defaultEvent(guardOccurrenceEvent, found);
    if (event !== undefined)
      host.dispatchEvent(event);
  };
  const guard = (verdict, event) => {
    let decided;
    try {
      decided = verdict();
    } catch {
      decided = {
        admit: false,
        occurrence: occurrence("action::protocol", event.type === "htmx:before:request" ? "request" : event.type === "htmx:before:response" ? "response" : "swap", null, event.type === "htmx:before:request" ? "none" : "uncertain", "unexpected_shape")
      };
    }
    if (decided.capture !== undefined && isObject(event.detail?.ctx))
      targets.set(event.detail.ctx, decided.capture);
    if (decided.admit)
      return;
    event.preventDefault();
    if (decided.occurrence !== undefined)
      announce(decided.occurrence);
  };
  const onRequest = (event) => {
    guard(() => checkRequestTarget(readContext(event.detail)), event);
  };
  const onResponse = (event) => {
    guard(() => checkResponseHeaders(readContext(event.detail)), event);
  };
  const onSwap = (event) => {
    guard(() => {
      const ctx = readContext(event.detail);
      const tasks = event.detail?.tasks;
      const captured = ctx !== undefined && isObject(ctx) ? targets.get(ctx) : undefined;
      return checkSwapTasks(ctx, tasks, captured);
    }, event);
  };
  host.addEventListener("htmx:before:request", onRequest);
  host.addEventListener("htmx:before:response", onResponse);
  host.addEventListener("htmx:before:swap", onSwap);
  let installed = true;
  return () => {
    if (!installed)
      return;
    installed = false;
    host.removeEventListener("htmx:before:request", onRequest);
    host.removeEventListener("htmx:before:response", onResponse);
    host.removeEventListener("htmx:before:swap", onSwap);
    targets = new WeakMap;
    globals.__canHtmxGuard = false;
  };
}
const autoGlobals = globalThis;
if (autoGlobals.document !== undefined && typeof autoGlobals.document.addEventListener === "function" && typeof autoGlobals.document.removeEventListener === "function" && typeof autoGlobals.document.dispatchEvent === "function") {
  installHTMXGuard(autoGlobals.document);
}
