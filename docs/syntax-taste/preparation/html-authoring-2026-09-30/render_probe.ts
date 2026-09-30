// Research observation only: execute emitted Can, then inspect the genuine
// opaque safe token through the same repository HTML runtime instance.
import { record } from "../../../../runtime/data.ts";
import { renderSafe } from "../../../../runtime/platform/html.ts";
import { domainFailureDiagnostics } from "../../../../runtime/domain.ts";

const specifications = await Bun.file(process.argv[2]!).json();
const results = [];
for (const specification of specifications) {
  const state = await import(specification.state);
  state.$canInitialize();
  const authored = await import(specification.module);
  const observations = [];
  for (const sample of specification.cases) {
    const fields = record(specification.record_identity, Object.entries(sample.fields));
    const result = await authored[specification.function](fields, sample.csrf, sample.notice);
    if (result.kind === "ok") {
      observations.push({ case: sample.name, kind: "ok", html: renderSafe(result.value) });
    } else if (result.kind === "domain") {
      const failure = domainFailureDiagnostics(result.value);
      observations.push({ case: sample.name, kind: "domain", error: failure.declaration.name, payload: failure.payload, origin: failure.origin });
    } else {
      throw new Error(`unexpected standard failure in ${specification.name}/${sample.name}`);
    }
  }
  results.push({ name: specification.name, observations });
}
process.stdout.write(JSON.stringify(results));
