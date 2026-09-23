// Fixed development qualification tool. The fixture occurrence crosses the Go/TS
// boundary without inventing a second error registry or asserting I49 is done.
import "../../runtime/environment.ts";
import { strict as assert } from "node:assert";
import { catalogueSHA256, validateErrorIdentity } from "../../runtime/catalogue.ts";
import { checkCatalogue } from "../../runtime/test/catalogue-checks.ts";

const fixture = JSON.parse(await Bun.stdin.text());
assert.equal(fixture.kind, "domain");
assert.equal(typeof fixture.occurrenceId, "string");
assert.equal(fixture.error.name, "codec::invalid_data");
assert.deepEqual(Object.keys(fixture.payload).sort(), ["path", "reason"]);
assert.equal(typeof fixture.payload.path, "string");
assert.equal(typeof fixture.payload.reason, "string");
const identity = validateErrorIdentity(fixture.error);
console.log(
  JSON.stringify({
    schemaVersion: 1,
    kind: "can.catalogue-conformance",
    catalogueSHA256,
    checks: checkCatalogue(),
    occurrence: {
      kind: fixture.kind,
      occurrenceId: fixture.occurrenceId,
      error: identity,
      payload: fixture.payload,
    },
  }),
);
