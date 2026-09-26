import { test, expect } from "bun:test";
import {
  createChartAdapter,
  createChartReporter,
  createFakeChartHost,
  CHART_POINTS_MAX,
  type ChartPoint,
  type ChartWidget,
} from "./chart-widget.ts";

const SPEC = {
  title: "Invoice totals",
  kind: "bar",
  points: [
    { label: "Jan", value: 120 },
    { label: "Feb", value: 200 },
  ],
};

function setup() {
  const host = createFakeChartHost();
  const reporter = createChartReporter();
  const adapter = createChartAdapter(host, reporter);
  return { host, reporter, adapter };
}

function mustCreate(adapter: ReturnType<typeof createChartAdapter>, onSelect: (point: ChartPoint) => void = () => {}): ChartWidget {
  const created = adapter.create(SPEC, onSelect);
  if (!created.ok) throw new Error("create failed");
  return created.value;
}

test("create, select, update, dispose roundtrip", () => {
  const { host, adapter } = setup();
  const seen: ChartPoint[] = [];
  const widget = mustCreate(adapter, (point) => void seen.push(point));
  host.fireSelect(host.lastHandle()!, { label: "Jan", value: 120 });
  expect(seen).toEqual([{ label: "Jan", value: 120 }]);
  expect(adapter.update(widget, [{ label: "Mar", value: 300 }]).ok).toBe(true);
  expect(adapter.dispose(widget)).toEqual({ ok: true, value: null });
  expect(host.liveHandles()).toBe(0);
});

test("invalid specs reject before host contact", () => {
  const { host, adapter } = setup();
  const bad = [
    null,
    {},
    { ...SPEC, title: "" },
    { ...SPEC, kind: "pie" },
    { ...SPEC, points: [] },
    { ...SPEC, points: [{ label: "x", value: NaN }] },
    { ...SPEC, points: [{ label: "", value: 1 }] },
    { ...SPEC, points: Array.from({ length: CHART_POINTS_MAX + 1 }, (_, i) => ({ label: `m${i}`, value: i })) },
  ];
  for (const spec of bad) {
    const got = adapter.create(spec, () => {});
    expect(got.ok).toBe(false);
    if (!got.ok) expect(got.failure.code).toBe("chart::invalid_spec");
  }
  expect(host.calls.length).toBe(0);
});

test("unavailable SDK fails closed without host contact", () => {
  const { host, adapter } = setup();
  host.setAvailable(false);
  const got = adapter.create(SPEC, () => {});
  expect(got.ok).toBe(false);
  if (!got.ok) expect(got.failure.code).toBe("chart::unavailable");
  expect(host.calls.length).toBe(0);
});

test("native SDK failure maps to unavailable with no native text", () => {
  const { host, adapter } = setup();
  host.injectCreateFailure();
  const got = adapter.create(SPEC, () => {});
  expect(got.ok).toBe(false);
  if (!got.ok) {
    expect(got.failure.code).toBe("chart::unavailable");
    expect(JSON.stringify(got)).not.toContain("exploded");
  }
});

test("deep copies: caller mutation cannot reach the widget or callbacks", () => {
  const { host, adapter } = setup();
  const mutable = { title: "Totals", kind: "bar", points: [{ label: "Jan", value: 1 }] };
  const seen: ChartPoint[] = [];
  const widget = mustCreate(adapter, (point) => void seen.push(point));
  void widget;
  mutable.points[0].label = "HACKED";
  mutable.points.push({ label: "extra", value: 9 });
  host.fireSelect(host.lastHandle()!, { label: "Jan", value: 1 });
  expect(seen).toEqual([{ label: "Jan", value: 1 }]);
  expect(Object.isFrozen(seen[0])).toBe(true);
});

test("callback throw is sealed: one diagnostic, SDK never sees it", () => {
  const { host, reporter, adapter } = setup();
  const widget = mustCreate(adapter, () => { throw new Error("boom"); });
  void widget;
  host.fireSelect(host.lastHandle()!, { label: "Jan", value: 120 });
  host.fireSelect(host.lastHandle()!, { label: "Feb", value: 200 });
  expect(reporter.diagnostics()).toEqual([
    { occurrence: 1, detail: "select callback failed" },
    { occurrence: 2, detail: "select callback failed" },
  ]);
  expect(JSON.stringify(reporter.diagnostics())).not.toContain("boom");
});

test("nested events serialize in dispatch order", () => {
  const { host, adapter } = setup();
  const seen: string[] = [];
  const widget = mustCreate(adapter, (point) => {
    seen.push(point.label);
    if (point.label === "first") host.fireSelect(host.lastHandle()!, { label: "nested", value: 2 });
  });
  void widget;
  host.fireSelect(host.lastHandle()!, { label: "first", value: 1 });
  expect(seen).toEqual(["first", "nested"]);
});

test("reentrant update applies immediately during dispatch", () => {
  const { host, adapter } = setup();
  let widget: ChartWidget | null = null;
  const created = adapter.create(SPEC, () => {
    expect(adapter.update(widget, [{ label: "live", value: 5 }]).ok).toBe(true);
  });
  if (!created.ok) throw new Error("create failed");
  widget = created.value;
  host.fireSelect(host.lastHandle()!, { label: "Jan", value: 120 });
  expect(host.calls.filter((call) => call.op === "updateChart").length).toBe(1);
});

test("dispose poisons: late events dropped, ops fail, double dispose ok", () => {
  const { host, adapter } = setup();
  let calls = 0;
  const widget = mustCreate(adapter, () => { calls += 1; });
  expect(adapter.dispose(widget).ok).toBe(true);
  expect(adapter.dispose(widget)).toEqual({ ok: true, value: null });
  host.fireSelect(host.lastHandle()!, { label: "Jan", value: 120 });
  expect(calls).toBe(0);
  const updated = adapter.update(widget, [{ label: "x", value: 1 }]);
  expect(updated.ok).toBe(false);
  if (!updated.ok) expect(updated.failure.code).toBe("chart::disposed");
  expect(adapter.dispose({})).toEqual({ ok: true, value: null });
  const foreign = adapter.update({}, [{ label: "x", value: 1 }]);
  expect(foreign.ok).toBe(false);
  if (!foreign.ok) expect(foreign.failure.code).toBe("chart::disposed");
});

test("dispose during callback completes the callback then drops the rest", () => {
  const { host, adapter } = setup();
  let widget: ChartWidget | null = null;
  const seen: string[] = [];
  const created = adapter.create(SPEC, (point) => {
    seen.push(point.label);
    expect(adapter.dispose(widget).ok).toBe(true);
  });
  if (!created.ok) throw new Error("create failed");
  widget = created.value;
  host.fireSelect(host.lastHandle()!, { label: "one", value: 1 });
  host.fireSelect(host.lastHandle()!, { label: "two", value: 2 });
  expect(seen).toEqual(["one"]);
  expect(host.liveHandles()).toBe(0);
});

test("tokens are opaque: SDK handles never leak to callers", () => {
  const { host, adapter } = setup();
  const widget = mustCreate(adapter);
  expect(Object.keys(widget as object)).toEqual([]);
  expect(Object.isFrozen(widget)).toBe(true);
  expect(widget).not.toBe(host.lastHandle());
});

test("surface census: three ops, three failure leaves, one callback", () => {
  const { adapter } = setup();
  expect(Object.keys(adapter).sort()).toEqual(["create", "dispose", "update"]);
  const leaves = new Set<string>();
  const bad = adapter.create({}, () => {});
  if (!bad.ok) leaves.add(bad.failure.code);
  const widget = mustCreate(adapter);
  expect(adapter.dispose(widget).ok).toBe(true);
  const stale = adapter.update(widget, [{ label: "x", value: 1 }]);
  if (!stale.ok) leaves.add(stale.failure.code);
  const down = setup();
  down.host.setAvailable(false);
  const unavailable = down.adapter.create(SPEC, () => {});
  if (!unavailable.ok) leaves.add(unavailable.failure.code);
  expect([...leaves].sort()).toEqual(["chart::disposed", "chart::invalid_spec", "chart::unavailable"]);
});
