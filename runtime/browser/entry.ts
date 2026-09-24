// Browser-profile root supervisor. The compiler-owned generated root awaits
// runBrowserEntry after DOM readiness: it configures the sealed diagnostic
// table, runs the checked zero-argument main exactly once under an explicit
// owner root, and reports a startup fault once through the sealed reporter.
// Success resolves; a startup fault rejects with the frozen diagnostic
// record. There is no process exit code and no test-authored boot module.
import { runExplicitRoot, type OwnerContext } from "../owner-core.ts";
import { invoke, type Completion } from "../completion.ts";
import {
  configureBrowserDiagnostics,
  reportBrowserDiagnostic,
  type BrowserDiagnostic,
  type SealedDiagnosticTable,
} from "./diagnostics.ts";
import type { FailureOrigin } from "../failure.ts";

export type BrowserMain = (ctx: OwnerContext) => Completion<void> | Promise<Completion<void>>;
export type BrowserReporter = (record: BrowserDiagnostic) => void | Promise<void>;
export type BrowserDocument = Readonly<{
  readyState: string;
  addEventListener(type: string, listener: () => void): void;
}>;

const rootOrigin: FailureOrigin = Object.freeze({
  source: "can:entry",
  start: 0,
  end: 0,
  invocation: Object.freeze([]),
});

function defaultDocument(): BrowserDocument {
  const candidate = (globalThis as { document?: unknown }).document;
  if (
    candidate === null ||
    typeof candidate !== "object" ||
    typeof (candidate as BrowserDocument).addEventListener !== "function" ||
    typeof (candidate as BrowserDocument).readyState !== "string"
  )
    throw new TypeError("browser entry requires a document");
  return candidate as BrowserDocument;
}

async function afterReady(document: BrowserDocument): Promise<void> {
  if (document.readyState !== "loading") return;
  await new Promise<void>((resolve) => document.addEventListener("DOMContentLoaded", resolve));
}

export async function runBrowserEntry(
  options: Readonly<{
    table: SealedDiagnosticTable;
    main: BrowserMain;
    report?: BrowserReporter;
  }>,
  host?: BrowserDocument,
): Promise<void> {
  const document = host ?? defaultDocument();
  await afterReady(document);
  const report = options.report ?? (async () => {});
  const startupFault = async (completion: Completion<void>): Promise<never> => {
    const record = reportBrowserDiagnostic(completion, "startup");
    if (!record) throw new TypeError("startup fault produced no diagnostic");
    await report(record);
    throw record;
  };
  try {
    configureBrowserDiagnostics(options.table);
  } catch (cause) {
    // A broken sealed table still reports once through the channel as an
    // unlocated startup fault instead of escaping unreported.
    const fault = await invoke<void>(() => {
      throw cause;
    }, rootOrigin);
    return startupFault(fault);
  }
  const owned = await runExplicitRoot(
    async (ctx) =>
      invoke(async () => {
        const result = await invoke(() => options.main(ctx), rootOrigin);
        if (result.kind === "ok" && result.value !== undefined)
          throw new TypeError("main returned a non-void result");
        return result;
      }, rootOrigin),
    async (diagnostic) => {
      // Late-owner diagnostics during startup share the sealed record shape
      // through the injected reporter; occurrence identity stays owner-held.
      const record: BrowserDiagnostic = Object.freeze({
        kind: "can.runtime-diagnostic",
        phase: "startup",
        occurrence: diagnostic.occurrence,
        category: diagnostic.category,
        file: "",
        line: 0,
        column: 0,
      });
      console.error(record);
      await report(record);
    },
  );
  const completion = owned.completion;
  if (completion.kind === "ok") {
    if (owned.cleanupFailed) {
      const fault = await invoke<void>(() => {
        throw new TypeError("browser startup cleanup failed");
      }, rootOrigin);
      return startupFault(fault);
    }
    return;
  }
  return startupFault(completion);
}
