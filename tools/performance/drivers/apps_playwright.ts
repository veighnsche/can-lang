import { chromium, firefox, webkit } from "playwright";
import { readFileSync, writeFileSync } from "node:fs";
const [mode, work, profile, iterationsRaw, warmupsRaw, sizeRaw, resultPath] = process.argv.slice(2);
function emit(output: unknown) {
  const encoded = JSON.stringify(output);
  if (resultPath) writeFileSync(resultPath, encoded + "\n");
  else console.log(encoded);
}
const engines = { chromium, firefox, webkit };
if (mode === "probe") {
  const available: string[] = [];
  const unavailable: Record<string, string> = {};
  for (const [name, engine] of Object.entries(engines)) {
    let browser;
    try {
      browser = await engine.launch({ headless: true });
      available.push(name);
    } catch (e) {
      unavailable[name] = String(e);
    } finally {
      await browser?.close();
    }
  }
  emit({ available, unavailable });
} else {
  const manifest = JSON.parse(readFileSync(`${work}/apps-manifest.json`, "utf8"));
  const requested = process.env.CAN_PERF_BROWSER_ENGINES?.split(",") ?? manifest.browser.available;
  const iterations = Number(iterationsRaw),
    warmups = Number(warmupsRaw),
    size = Number(sizeRaw);
  const batchCount = profile === "standard" ? 7 : 3;
  const totalEvents = (warmups + batchCount) * iterations;
  const group = (raw: number[]) =>
    Array.from({ length: raw.length / iterations }, (_, i) =>
      raw.slice(i * iterations, (i + 1) * iterations),
    );
  const mean = (raw: number[]) =>
    group(raw).map((batch) => batch.reduce((a, b) => a + b, 0) / iterations);
  const server = Bun.serve({
    hostname: "127.0.0.1",
    port: 0,
    fetch(req) {
      return new URL(req.url).pathname === "/browser.js"
        ? new Response(Bun.file(`${work}/browser.js`), {
            headers: { "content-type": "text/javascript" },
          })
        : new Response('<div id="app"></div><script type="module" src="/browser.js"></script>', {
            headers: { "content-type": "text/html" },
          });
    },
  });
  const cases: any[] = [];
  try {
    for (const name of requested) {
      if (!manifest.browser.available.includes(name))
        throw Error(`requested browser engine unavailable: ${name}`);
      const browser = await engines[name as keyof typeof engines].launch({ headless: true });
      try {
        const page = await browser.newPage();
        const errors: string[] = [];
        page.on("pageerror", (e) => errors.push(String(e)));
        await page.goto(`http://127.0.0.1:${server.port}`);
        await page
          .waitForFunction(() => (window as any).ready === true, undefined, { timeout: 15000 })
          .catch((e) => {
            throw Error(`readiness failed: ${errors.join("; ")}; ${e}`);
          });
        for (const flow of [
          "input-property-snapshot",
          "keyboard-suffix",
          "state-reset",
          "selection-roundtrip",
        ]) {
          const durations: number[] = [];
          const records: any[] = [];
          for (let operation = 0; operation < totalEvents; operation++) {
            const base = `${operation}:` + "x".repeat(size);
            if (flow !== "input-property-snapshot") {
              const before = await page.evaluate(() => (window as any).bench.events.length);
              await page.locator("#field").fill(base);
              await page.waitForFunction((n) => (window as any).completed === n, before + 1);
            }
            const offsets = await page.evaluate(() => ({
              input: (window as any).bench.events.length,
              reset: (window as any).bench.resets.length,
            }));
            let raw: any[],
              expected = base;
            if (flow === "selection-roundtrip") {
              const record = await page.evaluate(async () => {
                const b = (window as any).bench,
                  start = performance.now();
                const set = await b.browser.setSelection(b.input, 1n, 3n, "forward");
                const read = await b.browser.readSelection(b.input);
                const live = await b.browser.readValue(b.input);
                return {
                  duration_ms: performance.now() - start,
                  correct:
                    set.kind === "ok" &&
                    read.kind === "ok" &&
                    read.value.start === 1n &&
                    read.value.end === 3n &&
                    read.value.direction === "forward",
                  value: live.value,
                };
              });
              if (!record.correct || record.value !== base)
                throw Error("live selection/value mismatch");
              raw = [record];
            } else if (flow === "state-reset") {
              await page.locator("#reset").click();
              await page.waitForFunction(
                (n) => (window as any).resetCompleted === n,
                offsets.reset + 1,
              );
              raw = await page.evaluate(
                (start) => (window as any).bench.resets.slice(start),
                offsets.reset,
              );
              expected = "";
              const correct = await page.evaluate(async () => {
                const b = (window as any).bench;
                const state = await b.stateRuntime.readState(b.state);
                const live = await b.browser.readValue(b.input);
                return state.value.value === "" && live.value === "";
              });
              if (
                !correct ||
                raw.length !== 1 ||
                raw[0].version_step !== "1" ||
                raw[0].old_snapshot_value !== base
              )
                throw Error("state reset mismatch");
            } else {
              if (flow === "keyboard-suffix") {
                await page.locator("#field").press("End");
                await page.locator("#field").pressSequentially("abc");
                expected = base + "abc";
              } else await page.locator("#field").fill(base);
              const count = flow === "keyboard-suffix" ? 3 : 1;
              await page.waitForFunction(
                (n) => (window as any).completed === n,
                offsets.input + count,
              );
              raw = await page.evaluate(
                (start) => (window as any).bench.events.slice(start),
                offsets.input,
              );
              if (
                raw.length !== count ||
                raw.some(
                  (r) => r.value !== r.snapshot_value || r.kind !== "input" || !r.frozen_snapshot,
                ) ||
                raw.at(-1).value !== expected
              )
                throw Error("C02 input snapshot mismatch");
            }
            if ((await page.locator("#result").textContent()) !== expected)
              throw Error("DOM end state mismatch");
            durations.push(raw.reduce((sum, r) => sum + r.duration_ms, 0));
            records.push({
              operation,
              batch: Math.floor(operation / iterations),
              operation_in_batch: operation % iterations,
              warmup: operation < warmups * iterations,
              observations: raw,
              output_length: expected.length,
            });
          }
          if (errors.length) throw Error(`browser page errors: ${errors}`);
          cases.push({
            name: `${name}.c02.${flow}`,
            unit: "ms/interaction",
            samples: mean(durations.slice(warmups * iterations)),
            warmup_samples: mean(durations.slice(0, warmups * iterations)),
            iterations_per_sample: iterations,
            timing_scope:
              flow === "selection-roundtrip"
                ? "In-page live setSelection/readSelection/readValue operation through completion; automation excluded; batch means of operations."
                : "Sum of in-page handler-entry to awaited C02 property/state/DOM completion durations in each interaction; keyboard suffix includes three handlers. Setup, event queue, automation, layout and paint excluded; batch means of operations.",
            parameters: {
              size,
              engine: name,
              engine_version: browser.version(),
              profile,
              flow,
              contract: "C02-state-controls-v2",
            },
            correctness: {
              passed: true,
              checks: [
                "ready flag",
                "every DOM end state",
                "exact event/property snapshots and immutable snapshot",
                "exact selection or state reset where applicable",
                "no page errors",
              ],
            },
            metrics: {
              actual_paint: "unavailable",
              INP: "unavailable",
              event_queue_delay: "unavailable",
              timer_precision:
                "Engine-dependent performance.now; observed durations may quantize to zero.",
              interaction_records: records,
            },
          });
        }
      } finally {
        await browser.close();
      }
    }
  } finally {
    server.stop(true);
  }
  emit({
    schema_version: 1,
    suite: "browser",
    status: "complete",
    cases,
    notes: [
      "Initial coverage: C02 input snapshots, native keyboard suffix, state reset and live selection per installed engine. No full UI or paint claims.",
    ],
    artifacts: [],
  });
}
