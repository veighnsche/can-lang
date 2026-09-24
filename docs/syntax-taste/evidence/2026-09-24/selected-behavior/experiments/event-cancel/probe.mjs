import { createRequire } from "node:module";
import { strict as assert } from "node:assert";

const require = createRequire(
  new URL("../../../../../../../tests/integration/browser/package.json", import.meta.url),
);
const { chromium } = require("playwright");
const browser = await chromium.launch({ timeout: 120000 });
try {
  const page = await browser.newPage();
  await page.goto("about:blank");
  const observed = await page.evaluate(async () => {
    const lateInput = document.createElement("input");
    document.body.append(lateInput);
    lateInput.addEventListener("keydown", (event) => {
      void Promise.resolve().then(() => event.preventDefault());
    });
    const lateEvent = new KeyboardEvent("keydown", {
      key: "Enter",
      bubbles: true,
      cancelable: true,
    });
    lateInput.dispatchEvent(lateEvent);
    const lateImmediate = lateEvent.defaultPrevented;
    await Promise.resolve();
    const lateAfterMicrotask = lateEvent.defaultPrevented;

    const input = document.createElement("input");
    document.body.append(input);
    const owner = new AbortController();
    const handled = [];
    input.addEventListener(
      "keydown",
      (event) => {
        if (event.key === "Enter" && event.cancelable) event.preventDefault();
        handled.push(event.key);
        void Promise.resolve().then(() => undefined);
      },
      { signal: owner.signal },
    );
    const send = (key, cancelable) => {
      const event = new KeyboardEvent("keydown", {
        key,
        bubbles: true,
        cancelable,
      });
      input.dispatchEvent(event);
      return { key, cancelable, defaultPrevented: event.defaultPrevented, handled: handled.length };
    };
    const matched = send("Enter", true);
    const unmatched = send("Tab", true);
    const notCancelable = send("Enter", false);
    owner.abort();
    const disposed = send("Enter", true);
    return {
      delayedHandler: { immediate: lateImmediate, afterMicrotask: lateAfterMicrotask },
      registrationPolicy: { matched, unmatched, notCancelable, disposed, handled },
    };
  });
  assert.deepEqual(observed.delayedHandler, { immediate: false, afterMicrotask: true });
  assert.deepEqual(observed.registrationPolicy.matched, {
    key: "Enter", cancelable: true, defaultPrevented: true, handled: 1,
  });
  assert.deepEqual(observed.registrationPolicy.unmatched, {
    key: "Tab", cancelable: true, defaultPrevented: false, handled: 2,
  });
  assert.deepEqual(observed.registrationPolicy.notCancelable, {
    key: "Enter", cancelable: false, defaultPrevented: false, handled: 3,
  });
  assert.deepEqual(observed.registrationPolicy.disposed, {
    key: "Enter", cancelable: true, defaultPrevented: false, handled: 3,
  });
  console.log(JSON.stringify({ browser: browser.version(), observed }, null, 2));
} finally {
  await browser.close();
}
