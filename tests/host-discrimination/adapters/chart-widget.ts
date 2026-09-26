// X-R01-1 (D01) UNREVIEWED PROTOTYPE — reviewed-adapter tier, Widget C.
//
// Invoice chart widget through a vendor-neutral third-party chart-SDK
// shape: create with spec+data, update data, select callback, dispose.
// No vendor is pinned: ChartHost is the SDK-shape class (init/update/
// callback/dispose lifecycle), so vendor choice cannot prejudge W2.
// The adapter injects its host so every leg runs under Bun; SDK-call
// fidelity (opaque handles, callback registration, explicit dispose)
// is the documented shape of chart-SDK families, with live-vendor
// verification deferred to D02 conformance.
//
// Audit-contract shape exercised here: declared I/O types, immutable
// copying (deep at every crossing), failure mapping, target
// availability, callback/reentrancy rules (single dispatch, sealed
// reporting, queued reentrant updates, deferred dispose), and
// view-scoped ownership/disposal mirroring the C04/C05 rules (root
// append only, late callbacks never mutate disposed views).
export const CHART_TITLE_MAX_CHARS = 128;
export const CHART_LABEL_MAX_CHARS = 64;
export const CHART_POINTS_MAX = 366;

export type ChartPoint = Readonly<{ label: string; value: number }>;
export type ChartSpec = Readonly<{ title: string; kind: "bar"; points: readonly ChartPoint[] }>;

export type ChartFailureCode = "chart::invalid_spec" | "chart::disposed" | "chart::unavailable";

export type ChartFailure = Readonly<{ code: ChartFailureCode; detail: string }>;

export type ChartResult<T> =
  | Readonly<{ ok: true; value: T }>
  | Readonly<{ ok: false; failure: ChartFailure }>;

function ok<T>(value: T): ChartResult<T> {
  return Object.freeze({ ok: true, value }) as ChartResult<T>;
}

function fail<T>(code: ChartFailureCode, detail: string): ChartResult<T> {
  return Object.freeze({ ok: false, failure: Object.freeze({ code, detail }) }) as ChartResult<T>;
}

export type ChartDiagnostic = Readonly<{ occurrence: number; detail: string }>;

export type ChartReporter = Readonly<{
  report: (detail: string) => void;
  diagnostics: () => readonly ChartDiagnostic[];
}>;

export function createChartReporter(): ChartReporter {
  const entries: ChartDiagnostic[] = [];
  return Object.freeze({
    report(detail: string): void {
      entries.push(Object.freeze({ occurrence: entries.length + 1, detail }));
    },
    diagnostics(): readonly ChartDiagnostic[] {
      return Object.freeze([...entries]);
    },
  });
}

// ChartHost is the SDK-shape surface the adapter needs. Handles are
// host-owned; the adapter never leaks them (it mints its own opaque
// tokens). `fireSelect` exists only on the fake for deterministic
// event legs; a live SDK invokes the registered callback itself.
export type ChartSdkHandle = object;

export type ChartHost = {
  readonly available: boolean;
  createChart(spec: ChartSpec): ChartSdkHandle;
  updateChart(handle: ChartSdkHandle, points: readonly ChartPoint[]): void;
  onSelect(handle: ChartSdkHandle, callback: (point: ChartPoint) => void): void;
  disposeChart(handle: ChartSdkHandle): void;
};

export type ChartWidget = unknown;

export type ChartAdapter = Readonly<{
  create: (spec: unknown, onSelect: (point: ChartPoint) => void) => ChartResult<ChartWidget>;
  update: (widget: unknown, points: unknown) => ChartResult<null>;
  dispose: (widget: unknown) => ChartResult<null>;
}>;

function checkPoint(point: unknown): ChartPoint | null {
  if (point === null || typeof point !== "object" || Array.isArray(point)) return null;
  const shape = point as Record<string, unknown>;
  if (typeof shape.label !== "string" || shape.label.length === 0 || shape.label.length > CHART_LABEL_MAX_CHARS || !shape.label.isWellFormed()) {
    return null;
  }
  if (typeof shape.value !== "number" || !Number.isFinite(shape.value)) return null;
  return Object.freeze({ label: shape.label, value: shape.value });
}

function checkPoints(points: unknown): readonly ChartPoint[] | null {
  if (!Array.isArray(points) || points.length === 0 || points.length > CHART_POINTS_MAX) return null;
  const checked: ChartPoint[] = [];
  for (const point of points) {
    const valid = checkPoint(point);
    if (valid === null) return null;
    checked.push(valid);
  }
  return Object.freeze(checked);
}

function checkSpec(spec: unknown): ChartSpec | null {
  if (spec === null || typeof spec !== "object" || Array.isArray(spec)) return null;
  const shape = spec as Record<string, unknown>;
  if (typeof shape.title !== "string" || shape.title.length === 0 || shape.title.length > CHART_TITLE_MAX_CHARS || !shape.title.isWellFormed()) {
    return null;
  }
  if (shape.kind !== "bar") return null;
  const points = checkPoints(shape.points);
  if (points === null) return null;
  return Object.freeze({ title: shape.title, kind: "bar", points });
}

type WidgetRecord = {
  sdk: ChartSdkHandle;
  callback: (point: ChartPoint) => void;
  disposed: boolean;
  dispatching: boolean;
  queue: ChartPoint[];
};

export function createChartAdapter(host: ChartHost, reporter: ChartReporter): ChartAdapter {
  const widgets = new WeakMap<object, WidgetRecord>();
  const mint = (record: WidgetRecord): ChartWidget => {
    const token = Object.freeze(Object.create(null));
    widgets.set(token, record);
    return token;
  };
  const read = (widget: unknown): WidgetRecord | null => {
    if (widget === null || (typeof widget !== "object" && typeof widget !== "function")) return null;
    return widgets.get(widget) ?? null;
  };
  const dispatch = (record: WidgetRecord): void => {
    if (record.dispatching) return;
    record.dispatching = true;
    try {
      while (record.queue.length > 0 && !record.disposed) {
        const point = record.queue.shift()!;
        try {
          record.callback(point);
        } catch {
          // Sealed reporting: one sanitized diagnostic per failed
          // callback; the SDK never sees the throw and dispatch ends
          // for this event only.
          reporter.report("select callback failed");
        }
      }
      // Disposal during dispatch drops the queue: late events never
      // mutate disposed widgets.
      record.queue.length = 0;
    } finally {
      record.dispatching = false;
    }
  };
  return Object.freeze({
    create(spec: unknown, onSelect: (point: ChartPoint) => void): ChartResult<ChartWidget> {
      const checked = checkSpec(spec);
      if (checked === null) return fail("chart::invalid_spec", "spec needs a title, kind bar, and 1..366 finite points");
      if (typeof onSelect !== "function") return fail("chart::invalid_spec", "select callback must be a function");
      if (!host.available) return fail("chart::unavailable", "chart SDK unavailable");
      // Deep copy at the boundary: later caller mutation cannot reach
      // the widget, and SDK-side retention cannot leak back.
      const owned: ChartSpec = Object.freeze({
        title: checked.title,
        kind: "bar",
        points: Object.freeze(checked.points.map((point) => Object.freeze({ label: point.label, value: point.value }))),
      });
      try {
        const record: WidgetRecord = { sdk: undefined as unknown as ChartSdkHandle, callback: onSelect, disposed: false, dispatching: false, queue: [] };
        record.sdk = host.createChart(owned);
        host.onSelect(record.sdk, (point: ChartPoint) => {
          record.queue.push(Object.freeze({ label: point.label, value: point.value }));
          dispatch(record);
        });
        return ok(mint(record));
      } catch {
        return fail("chart::unavailable", "chart SDK unavailable");
      }
    },
    update(widget: unknown, points: unknown): ChartResult<null> {
      const record = read(widget);
      if (record === null || record.disposed) return fail("chart::disposed", "widget disposed");
      const checked = checkPoints(points);
      if (checked === null) return fail("chart::invalid_spec", "update needs 1..366 finite points");
      const owned = Object.freeze(checked.map((point) => Object.freeze({ label: point.label, value: point.value })));
      try {
        host.updateChart(record.sdk, owned);
        return ok(null);
      } catch {
        return fail("chart::unavailable", "chart SDK unavailable");
      }
    },
    dispose(widget: unknown): ChartResult<null> {
      const record = read(widget);
      // Idempotent: double dispose is a no-op success, mirroring
      // dispose_view; unknown tokens are disposed, never a fault.
      if (record === null || record.disposed) return ok(null);
      record.disposed = true;
      record.queue.length = 0;
      try {
        host.disposeChart(record.sdk);
      } catch {
        // Poisoned regardless: a throwing SDK dispose still disposes.
      }
      return ok(null);
    },
  });
}

// Callback/reentrancy rules (pinned by legs below):
// 1. single dispatch: events queue while a callback runs;
// 2. callback throws are captured into the sealed reporter once each;
// 3. reentrant update during a callback applies immediately and
//    synchronously (no interleaving under a sync SDK); only event
//    callbacks serialize through the dispatch queue;
// 4. dispose during a callback poisons the widget and drops its queue;
//    the in-flight callback completes.

export type FakeChartHost = ChartHost & {
  calls: Array<Readonly<{ op: string }>>;
  setAvailable(available: boolean): void;
  injectCreateFailure(): void;
  clearFault(): void;
  fireSelect(handle: ChartSdkHandle, point: ChartPoint): void;
  liveHandles(): number;
  lastHandle(): ChartSdkHandle | null;
};

export function createFakeChartHost(): FakeChartHost {
  const calls: Array<Readonly<{ op: string }>> = [];
  const callbacks = new Map<object, (point: ChartPoint) => void>();
  let available = true;
  let failCreate = false;
  let last: ChartSdkHandle | null = null;
  return {
    calls,
    get available(): boolean {
      return available;
    },
    createChart(): ChartSdkHandle {
      calls.push({ op: "createChart" });
      if (failCreate) throw new Error("sdk exploded");
      const handle = {};
      callbacks.set(handle, () => {});
      last = handle;
      return handle;
    },
    updateChart(): void {
      calls.push({ op: "updateChart" });
    },
    onSelect(handle: ChartSdkHandle, callback: (point: ChartPoint) => void): void {
      calls.push({ op: "onSelect" });
      callbacks.set(handle, callback);
    },
    disposeChart(handle: ChartSdkHandle): void {
      calls.push({ op: "disposeChart" });
      callbacks.delete(handle);
    },
    setAvailable(next: boolean): void {
      available = next;
    },
    injectCreateFailure(): void {
      failCreate = true;
    },
    clearFault(): void {
      failCreate = false;
    },
    fireSelect(handle: ChartSdkHandle, point: ChartPoint): void {
      callbacks.get(handle)?.(point);
    },
    liveHandles(): number {
      return callbacks.size;
    },
    lastHandle(): ChartSdkHandle | null {
      return last;
    },
  };
}
