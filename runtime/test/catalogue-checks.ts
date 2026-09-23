import { strict as assert } from "node:assert";
import { catalogue, operation, requireConstructor, errorIdentity, validateErrorIdentity } from "../catalogue.ts";

export function checkCatalogue(): string[] {
  const passed: string[] = [];
  function check(name: string, body: () => void) { body(); passed.push(name); }
  check("complete operation lookup", () => {
    assert.equal(catalogue.operations.length, 254);
    for (const item of catalogue.operations) assert.equal(operation(item.name).identity, item.identity);
  });
  check("exact allocated error identity", () => {
    assert.deepEqual(errorIdentity("codec::invalid_data"), { name: "codec::invalid_data", identity: "can.std.codec@1::invalid_data", id: 1110, typeArguments: [] });
  });
  check("wrong target and revision rejected", () => {
    assert.throws(() => operation("append", "linux"));
    assert.throws(() => operation("append", catalogue.targetId, 2));
  });
  check("opaque constructors rejected", () => {
    for (const name of ["bytes::buffer", "html::safe", "sql::transaction", "standard_failure"]) assert.throws(() => requireConstructor(name));
    assert.equal(requireConstructor("choice_option").name, "choice_option");
  });
  check("unallocated and tampered errors rejected", () => {
    assert.throws(() => errorIdentity("codec::not_allocated"));
    assert.throws(() => validateErrorIdentity({ ...errorIdentity("codec::invalid_data"), id: 1199 }));
    assert.throws(() => validateErrorIdentity({ ...errorIdentity("codec::invalid_data"), identity: "user::invalid_data" }));
    assert.throws(() => errorIdentity("all_failed"));
  });
  check("kernel and protocol registration absent", () => {
    for (const name of ["kernel::register", "protocol::register", "sql::unsafe", "html::raw"]) assert.throws(() => operation(name));
  });
  check("metadata deeply immutable", () => {
    assert(Object.isFrozen(catalogue));
    assert(Object.isFrozen(catalogue.operations));
    assert(Object.isFrozen(catalogue.operations[0].lowering.native));
    assert.throws(() => Object.defineProperty(catalogue.operations[0], "emits", { value: [] }));
  });
  check("transaction callback has an empty bound", () => {
    const value = operation("sql::with_transaction");
    const callback = value.callbacks?.[0];
    assert(callback);
    assert.deepEqual(callback.emits, []);
    assert.equal(callback.deriveErrors, false);
  });
  check("thenable-safe sequential map recipe", () => {
    const value = operation("array.map");
    assert.deepEqual(value.callbackErrors, ["callback"]);
    const native: readonly string[] = value.lowering.native;
    assert(native.includes("Array.fromAsync"));
    assert(native.includes("Array.prototype.keys"));
  });
  return passed;
}
