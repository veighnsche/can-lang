// Compiler-private root supervisor. Generated entry modules await runEntry at
// top level and assign its status to process.exitCode only after it settles.
import {runOwnedRoot} from "./owner.ts";
import { invoke, success, type Completion } from "./completion.ts";
import { array } from "./data.ts";
import { domainFailureDiagnostics } from "./domain.ts";
import { standardFailureDiagnostics, type FailureOrigin } from "./failure.ts";

import { diagnosticFrames } from "./diagnostics.ts";

type Main = (args: readonly string[]) => Completion<void> | Promise<Completion<void>>;
type Reporter = (line: string) => void | Promise<void>;
const rootOrigin: FailureOrigin = Object.freeze({source: "can:entry", start: 0, end: 0, invocation: Object.freeze([])});

// Terminal completion reports select stable fields. Q10 late-owner reports
// separately include their specified C9 message projection. Error
// payloads can contain application secrets; native messages can contain paths,
// credentials, or response bodies. Neither is serialized by this terminal reporter.
function diagnostic(completion: Exclude<Completion<void>, {kind: "ok"}>, phase: "initialization" | "main"): string {
  const base = {schemaVersion: 1, kind: "can.runtime-failure", phase};
  if (completion.kind === "domain") {
    const details = domainFailureDiagnostics(completion.value);
    return JSON.stringify({...base, channel: "domain", id: details.declaration.id,
      error: details.declaration.name, typeIdentity: details.typeIdentity,
      occurrence: String(details.occurrenceID), payload: "<redacted>", frames: diagnosticFrames(undefined,details.origin)}) + "\n";
  }
  const details = standardFailureDiagnostics(completion.value);
  return JSON.stringify({...base, channel: "standard", category: details.kind,
    occurrence: String(details.occurrenceID), frames: diagnosticFrames(details.cause,details.boundaryOrigin ?? details.origin)}) + "\n";
}
async function reportToStderr(line: string): Promise<void> { await Bun.write(Bun.stderr, line); }

export async function runEntry(
  initialize: () => void,
  main: Main,
  applicationArgs: readonly string[],
  report: Reporter = reportToStderr,
): Promise<0 | 1> {
  let phase: "initialization" | "main" = "initialization";
  const owned=await runOwnedRoot(async()=>{
  let completion: Completion<void> = await invoke(() => {
    initialize();
    return success(undefined);
  }, rootOrigin);
  if (completion.kind === "ok") {
    phase = "main";
    completion = await invoke(async () => {
      // The CLI already removes its flags and project selector. Copy before
      // freezing so neither the caller nor the application retains a mutable alias.
      const result = await invoke(() => main(array([...applicationArgs])), rootOrigin);
      if (result.kind === "ok" && result.value !== undefined) throw new TypeError("main returned a non-void result");
      return result;
    }, rootOrigin);
  }
  return completion;
  }, diagnostic=>report(JSON.stringify({schemaVersion:1,...diagnostic})+"\n"));
  const completion=owned.completion;
  if (completion.kind === "ok") return owned.cleanupFailed?1:0;
  try { await report(diagnostic(completion, phase)); }
  catch {
    // A closed diagnostic pipe must not become an unhandled rejection or turn
    // a failed program into a successful exit. There is no fallback output sink.
  }
  return 1;
}
