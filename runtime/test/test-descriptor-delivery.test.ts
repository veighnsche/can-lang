// NT-I11 descriptor-delivery contract: the runtime catalogue mirror carries
// the descriptor package with 7 owner-first operations over 1 opaque launch
// handle, owner-mirror records, K17 schema-verbatim wire records, and the
// descriptor_fault error. Markers descriptor:: and can.std.descriptor@1 keep
// the NT-I11 catalogue package referenced from runtime test evidence. No
// $canDescriptor behavior exists yet; this file pins the mirror shape only.
import { test, expect } from "bun:test";
import { catalogue } from "../catalogue.ts";

const DESCRIPTOR_OPS = [
  "descriptor::launch_child",
  "descriptor::collect_status",
  "descriptor::wait_child",
  "descriptor::kill_child",
  "descriptor::release_launch",
  "descriptor::read_facts",
  "descriptor::expected_ack",
] as const;

const DESCRIPTOR_EMITS: Record<string, string[]> = {
  "descriptor::launch_child": ["test::stale_handle", "descriptor::descriptor_fault"],
  "descriptor::collect_status": ["test::stale_handle", "descriptor::descriptor_fault"],
  "descriptor::wait_child": ["test::stale_handle", "descriptor::descriptor_fault"],
  "descriptor::kill_child": ["test::stale_handle"],
  "descriptor::release_launch": ["test::stale_handle", "descriptor::descriptor_fault"],
  "descriptor::read_facts": ["test::stale_handle"],
  "descriptor::expected_ack": ["test::stale_handle"],
};

test("descriptor package mirror carries the NT-I11 operation surface", () => {
  const pkg = catalogue.packages.find((entry) => entry.name === "descriptor");
  expect(pkg?.identity).toBe("can.std.descriptor@1");
  const ops = catalogue.operations.filter((op) => op.name.startsWith("descriptor::"));
  expect(ops.map((op) => op.name).sort()).toEqual([...DESCRIPTOR_OPS].sort());
  for (const op of ops) {
    expect(op.assertion).toBe("supplied");
    expect(op.lowering.task).toBe("NT-I11");
    expect(op.lowering.native.length).toBeGreaterThan(0);
    expect(op.emits).toEqual(DESCRIPTOR_EMITS[op.name]);
    expect(op.inputs[0]?.type).toBe("test::owner");
  }
});

test("descriptor facts mirror the owner observations verbatim", () => {
  const facts = catalogue.types.find((entry) => entry.name === "descriptor::facts");
  const fields = Object.fromEntries((facts?.fields ?? []).map((field) => [field.name, field.type]));
  expect(fields).toEqual({
    offered: "bool",
    offered_bytes: "int",
    writer_done: "bool",
    writer_err: "str",
    accepted: "bool",
    malformed: "bool",
    eof: "bool",
    reaped: "bool",
    child_exit: "option::value<descriptor::exit>",
  });
});

test("descriptor wire records mirror the K17 schemas verbatim", () => {
  const fd = catalogue.types.find((entry) => entry.name === "descriptor::fd_entry");
  expect((fd?.fields ?? []).map((field) => field.name)).toEqual([
    "fd",
    "owner",
    "direction",
    "byte_state",
    "eof_state",
    "lifetime",
    "bytes_offered",
    "bytes_accepted",
  ]);
  const env = catalogue.types.find((entry) => entry.name === "descriptor::environment");
  expect((env?.fields ?? []).map((field) => field.name)).toEqual([
    "schema_version",
    "run_id",
    "launch_id",
    "entry",
    "argv",
    "env_names",
    "env_digest",
    "log_sink",
  ]);
  const launch = catalogue.types.find((entry) => entry.name === "descriptor::launch");
  expect(launch?.kind).toBe("opaque");
  expect(launch?.constructible).toBe(false);
});
