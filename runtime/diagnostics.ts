// Private diagnostics: Bun host binding over the canonical core. The source
// index and module maps load from the generation directory; the browser
// profile overlays runtime/browser/diagnostics.ts with a sealed table.
import { readFileSync } from "node:fs";
import { configureFromData } from "./diagnostics-core.ts";

export type {
  DiagnosticFrame,
  DiagnosticSource,
  DiagnosticSourceIndex,
  DiagnosticSourceIndexModule,
  DiagnosticSpan,
} from "./diagnostics-core.ts";
export { configureFromData, diagnosticFrames, locateOrigin } from "./diagnostics-core.ts";

export function configureDiagnostics(entryURL: string): void {
  const base = new URL("./", entryURL);
  const index = JSON.parse(readFileSync(new URL("diagnostics/source-index.json", base), "utf8"));
  configureFromData(
    index,
    (modulePath) => JSON.parse(readFileSync(new URL(modulePath + ".map", base), "utf8")),
    (modulePath) => decodeURIComponent(new URL(modulePath, base).pathname),
  );
}
