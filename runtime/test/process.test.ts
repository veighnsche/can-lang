import { test, expect } from "bun:test";
import { createHash } from "node:crypto";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { mkdtemp, rm, mkdir, chmod, readFile, realpath } from "node:fs/promises";
import { catalogue } from "../catalogue.ts";
import { createDomainRuntime, domainFailureDiagnostics, type FailureShape } from "../domain.ts";
import { createProcesses } from "../platform/process/spawn.ts";
import { classifySpawnError, parseEnvEntry } from "../platform/process/errors.ts";
import { copyBytes, ownBytes } from "../bytes.ts";
import { array, record, recordIdentity, dataProperty } from "../data.ts";
import { success, invoke, type Completion } from "../completion.ts";
import { runAssertion } from "../assert/runner.ts";
import { runOwnedRoot, withScope } from "../owner.ts";
const hash = (kind: string, name: string) =>
  createHash("sha256")
    .update("can-concrete-type-v1\0" + JSON.stringify([kind, name]))
    .digest("hex");
const shape = (
  kind: string,
  declaration: string,
  fields: { name: string; type: string }[] = [],
): FailureShape => ({
  identity: hash(kind, declaration),
  kind,
  declaration,
  fields,
  arguments: [],
  leaves: [],
  inputs: [],
  errors: [],
});
const str = shape("primitive", "str"),
  int = shape("primitive", "int"),
  bool = shape("primitive", "bool");
const declarations = catalogue.errors.filter((e) =>
  [
    "files::not_found",
    "files::denied",
    "process::spawn_failed",
    "process::timeout",
    "process::output_limit",
    "process::nonzero",
    "process::invalid_config",
    "process::io_error",
  ].includes(e.name),
);
const errors = declarations.map((e) =>
  shape(
    "error",
    e.identity,
    e.fields.map((f) => ({ name: f.name, type: f.type === "int" ? int.identity : str.identity })),
  ),
);
const domain = createDomainRuntime({
  declarations: declarations.map((e) => ({ ...e, parameters: 0 })),
  shapes: [str, int, bool, ...errors],
});
const identity = (name: string) => errors.find((e) => e.declaration === name)!.identity;
const resultId = hash("record", "can.std.process@1::result");
const processes = createProcesses(domain, {
  notFound: identity("can.std.files@1::not_found"),
  denied: identity("can.std.files@1::denied"),
  spawnFailed: identity("can.std.process@1::spawn_failed"),
  timeout: identity("can.std.process@1::timeout"),
  outputLimit: identity("can.std.process@1::output_limit"),
  nonzero: identity("can.std.process@1::nonzero"),
  invalidConfig: identity("can.std.process@1::invalid_config"),
  ioError: identity("can.std.process@1::io_error"),
  result: resultId,
});
const origin = { source: "test", start: 0, end: 0, invocation: [] };
type Over = Partial<{
  cwd: string;
  inherit: boolean;
  env: string[];
  stdin: Uint8Array;
  out: bigint;
  err: bigint;
  deadline: bigint;
  grace: bigint;
}>;
const options = (over: Over = {}) =>
  record("test-options", [
    ["cwd", over.cwd ?? ""],
    ["inherit_env", over.inherit ?? false],
    ["env", array(over.env ?? [])],
    ["stdin", ownBytes(over.stdin ?? new Uint8Array())],
    ["stdout_limit", over.out ?? 1000000n],
    ["stderr_limit", over.err ?? 1000000n],
    ["deadline_ms", over.deadline ?? 0n],
    ["grace_ms", over.grace ?? 100n],
  ]);
function check(result: Completion, name: string, payload: object) {
  expect(result.kind).toBe("domain");
  if (result.kind !== "domain") throw Error("expected domain");
  const d = domainFailureDiagnostics(result.value);
  expect(d.declaration.name).toBe(name);
  expect(d.payload).toMatchObject(payload);
}
async function owned<T>(body: () => Promise<Completion<T>>): Promise<Completion<T>> {
  const root = await runOwnedRoot(body);
  expect(root.cleanupFailed).toBe(false);
  return root.completion;
}
async function withTemp(run: (root: string) => Promise<void>): Promise<void> {
  const raw = await mkdtemp(join(tmpdir(), "can-process-test-"));
  const root = await realpath(raw);
  try {
    await run(root);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
}
function textOf(value: unknown, field: string): string {
  return new TextDecoder().decode(copyBytes(dataProperty(value, field) as never, origin));
}
test("echo round-trips stdout with exit zero", async () => {
  const result = await owned(() => processes.run("/bin/echo", ["hello"], options()));
  expect(result.kind).toBe("ok");
  if (result.kind !== "ok") throw Error();
  expect(recordIdentity(result.value)).toBe(resultId);
  expect(textOf(result.value, "stdout")).toBe("hello\n");
  expect(textOf(result.value, "stderr")).toBe("");
  expect(dataProperty(result.value, "code")).toBe(0n);
  expect(dataProperty(result.value, "signal")).toBe("");
  expect(await owned(() => processes.requireSuccess(result.value))).toEqual(success(result.value));
  const bare = await owned(() => processes.run("echo", ["via-path"], options()));
  expect(bare.kind).toBe("ok");
  if (bare.kind !== "ok") throw Error();
  expect(textOf(bare.value, "stdout")).toBe("via-path\n");
});
test("nonzero exits and signals report distinctly", async () => {
  const failed = await owned(() => processes.run("/bin/sh", ["-c", "exit 7"], options()));
  expect(failed.kind).toBe("ok");
  if (failed.kind !== "ok") throw Error();
  expect(dataProperty(failed.value, "code")).toBe(7n);
  expect(dataProperty(failed.value, "signal")).toBe("");
  check(await owned(() => processes.requireSuccess(failed.value)), "process::nonzero", {
    code: 7n,
    signal: "",
  });
  const signaled = await owned(() => processes.run("/bin/sh", ["-c", "kill -TERM $$"], options()));
  expect(signaled.kind).toBe("ok");
  if (signaled.kind !== "ok") throw Error();
  expect(dataProperty(signaled.value, "code")).toBe(-1n);
  expect(dataProperty(signaled.value, "signal")).toBe("SIGTERM");
  check(await owned(() => processes.requireSuccess(signaled.value)), "process::nonzero", {
    code: -1n,
    signal: "SIGTERM",
  });
});
test("stdin reaches the child; streams drain concurrently past pipe buffers", async () => {
  const upper = await owned(() =>
    processes.run(
      "/usr/bin/tr",
      ["a-z", "A-Z"],
      options({ stdin: new TextEncoder().encode("hello") }),
    ),
  );
  expect(upper.kind).toBe("ok");
  if (upper.kind !== "ok") throw Error();
  expect(textOf(upper.value, "stdout")).toBe("HELLO");
  const big = await owned(() =>
    processes.run(
      "/bin/sh",
      ["-c", "head -c 200000 /dev/zero | tr '\\0' 'x'; echo marker >&2"],
      options(),
    ),
  );
  expect(big.kind).toBe("ok");
  if (big.kind !== "ok") throw Error();
  expect(copyBytes(dataProperty(big.value, "stdout") as never, origin).byteLength).toBe(200000);
  expect(textOf(big.value, "stderr")).toBe("marker\n");
});
test("arguments stay literal without a shell", async () => {
  await withTemp(async (root) => {
    const sentinel = join(root, "pwned");
    const result = await owned(() =>
      processes.run("/bin/echo", ["$(touch " + sentinel + ")", "a;b", "`id`"], options()),
    );
    expect(result.kind).toBe("ok");
    if (result.kind !== "ok") throw Error();
    expect(textOf(result.value, "stdout")).toBe("$(touch " + sentinel + ") a;b `id`\n");
    expect(await Bun.file(sentinel).exists()).toBe(false);
  });
});
test("missing executables and denied directories are declared failures", async () => {
  await withTemp(async (root) => {
    check(
      await owned(() => processes.run("/nonexistent-bin-xyz", [], options())),
      "files::not_found",
      {
        path: "/nonexistent-bin-xyz",
      },
    );
    check(await owned(() => processes.which("definitely-not-a-binary-xyz")), "files::not_found", {
      path: "definitely-not-a-binary-xyz",
    });
    check(await owned(() => processes.which("")), "process::invalid_config", {
      field: "name",
      reason: "empty",
    });
    expect(await owned(() => processes.which("sh"))).toEqual(success("/bin/sh"));
    const locked = join(root, "locked");
    await mkdir(locked);
    await chmod(locked, 0o000);
    try {
      check(
        await owned(() => processes.run("/bin/echo", ["x"], options({ cwd: locked }))),
        "files::denied",
        {
          operation: "spawn",
        },
      );
    } finally {
      await chmod(locked, 0o700);
    }
    check(
      await owned(() => processes.run("/bin/echo", ["x"], options({ cwd: join(root, "nodir") }))),
      "files::not_found",
      { path: join(root, "nodir") },
    );
  });
});
test("invalid configs fail before spawning", async () => {
  await withTemp(async (root) => {
    const sentinel = join(root, "spawned");
    check(await owned(() => processes.run("", [], options())), "process::invalid_config", {
      field: "executable",
      reason: "empty",
    });
    check(await owned(() => processes.run("a\0b", [], options())), "process::invalid_config", {
      field: "executable",
      reason: "nul_byte",
    });
    check(
      await owned(() => processes.run("/bin/echo", ["a\0b"], options())),
      "process::invalid_config",
      {
        field: "args",
        reason: "nul_byte",
      },
    );
    check(
      await owned(() => processes.run("/bin/echo", [], options({ env: ["NOEQUALS"] }))),
      "process::invalid_config",
      {
        field: "env",
        reason: "entry",
      },
    );
    check(
      await owned(() => processes.run("/bin/echo", [], options({ env: ["1BAD=x"] }))),
      "process::invalid_config",
      {
        field: "env",
        reason: "name",
      },
    );
    check(
      await owned(() => processes.run("/bin/echo", [], options({ env: ["K=a\0b"] }))),
      "process::invalid_config",
      {
        field: "env",
        reason: "nul_byte",
      },
    );
    check(
      await owned(() => processes.run("/bin/echo", [], options({ out: -1n }))),
      "process::invalid_config",
      {
        field: "stdout_limit",
        reason: "negative",
      },
    );
    check(
      await owned(() => processes.run("/bin/echo", [], options({ deadline: -1n }))),
      "process::invalid_config",
      {
        field: "deadline_ms",
        reason: "negative",
      },
    );
    check(
      await owned(() => processes.run("/bin/echo", [], options({ deadline: 2147483648n }))),
      "process::invalid_config",
      { field: "deadline_ms", reason: "too_large" },
    );
    check(
      await owned(() => processes.run("/usr/bin/touch", [sentinel], options({ env: ["BAD"] }))),
      "process::invalid_config",
      { field: "env", reason: "entry" },
    );
    expect(await Bun.file(sentinel).exists()).toBe(false);
  });
});
test("output caps terminate runaway producers", async () => {
  const flood = await owned(() =>
    processes.run("/bin/sh", ["-c", "yes | head -c 1000000"], options({ out: 1000n, grace: 50n })),
  );
  check(flood, "process::output_limit", { stream: "stdout", limit: 1000n });
  const exact = await owned(() => processes.run("/bin/echo", ["hello"], options({ out: 6n })));
  expect(exact.kind).toBe("ok");
  const short = await owned(() => processes.run("/bin/echo", ["hello"], options({ out: 5n })));
  check(short, "process::output_limit", { stream: "stdout", limit: 5n });
});
test("deadlines terminate and reap, escalating past SIGTERM traps", async () => {
  const started = Date.now();
  const result = await owned(() =>
    processes.run("/bin/sleep", ["30"], options({ deadline: 300n, grace: 50n })),
  );
  check(result, "process::timeout", { deadline_ms: 300n });
  expect(Date.now() - started).toBeLessThan(10000);
  const trapped = await owned(() =>
    processes.run(
      "/bin/sh",
      ["-c", "trap '' TERM; sleep 30"],
      options({ deadline: 300n, grace: 100n }),
    ),
  );
  check(trapped, "process::timeout", { deadline_ms: 300n });
  expect(Date.now() - started).toBeLessThan(10000);
});
test("group termination reaches grandchildren", async () => {
  await withTemp(async (root) => {
    const pidfile = join(root, "grandchild.pid");
    const result = await owned(() =>
      processes.run(
        "/bin/sh",
        ["-c", "sleep 60 & echo $! > " + pidfile + "; wait"],
        options({ deadline: 500n, grace: 50n }),
      ),
    );
    check(result, "process::timeout", { deadline_ms: 500n });
    const grand = parseInt((await readFile(pidfile, "utf8")).trim(), 10);
    expect(Number.isSafeInteger(grand)).toBe(true);
    let alive = true;
    try {
      process.kill(grand, 0);
    } catch {
      alive = false;
    }
    expect(alive).toBe(false);
  });
});
test("environment inheritance is explicit; cwd applies", async () => {
  await withTemp(async (root) => {
    const bare = await owned(() =>
      processes.run("/usr/bin/env", [], options({ env: ["CAN_MARK=yes"] })),
    );
    expect(bare.kind).toBe("ok");
    if (bare.kind !== "ok") throw Error();
    const plain = textOf(bare.value, "stdout");
    expect(plain).toContain("CAN_MARK=yes\n");
    expect(plain).not.toContain("PATH=");
    const inherited = await owned(() =>
      processes.run("/usr/bin/env", [], options({ inherit: true, env: ["CAN_MARK=yes"] })),
    );
    expect(inherited.kind).toBe("ok");
    if (inherited.kind !== "ok") throw Error();
    const full = textOf(inherited.value, "stdout");
    expect(full).toContain("CAN_MARK=yes\n");
    expect(full).toContain("PATH=");
    const where = await owned(() => processes.run("/bin/pwd", [], options({ cwd: root })));
    expect(where.kind).toBe("ok");
    if (where.kind !== "ok") throw Error();
    expect(textOf(where.value, "stdout").trim()).toBe(root);
  });
});
test("unsupplied assertions cannot spawn", async () => {
  await withTemp(async (root) => {
    const ghost = join(root, "ghost");
    for (const [name, actual] of [
      ["run", (context?: never) => processes.run("/usr/bin/touch", [ghost], options(), context)],
      ["which", (context?: never) => processes.which("sh", context)],
    ] as const) {
      const result = await runAssertion({
        root: { package: "test", declaration: "boundary", name },
        actual: actual as never,
        expected: async () => success(undefined),
      });
      expect(result.passed).toBe(false);
    }
    expect(await Bun.file(ghost).exists()).toBe(false);
    const good = record(resultId, [
      ["stdout", ownBytes(new Uint8Array())],
      ["stderr", ownBytes(new Uint8Array())],
      ["code", 0n],
      ["signal", ""],
    ]);
    const pure = await runAssertion({
      root: { package: "test", declaration: "boundary", name: "require" },
      actual: () => processes.requireSuccess(good),
      expected: async () => success(good),
    });
    expect(pure.passed).toBe(true);
  });
});
test("unknown spawn failures stay standard; classification is exact", async () => {
  expect(classifySpawnError(new TypeError("defect"))).toBeUndefined();
  expect(classifySpawnError(Object.assign(new Error("x"), { code: "ENOENT" }))).toEqual({
    kind: "not_found",
    code: "ENOENT",
  });
  expect(classifySpawnError(Object.assign(new Error("x"), { code: "EACCES" }))).toEqual({
    kind: "denied",
    code: "EACCES",
  });
  expect(classifySpawnError(Object.assign(new Error("x"), { code: "ETXTBSY" }))).toEqual({
    kind: "failed",
    code: "ETXTBSY",
  });
  expect(classifySpawnError(Object.assign(new Error("x"), { code: "BOGUS" }))).toBeUndefined();
  expect(parseEnvEntry("A=b=c")).toEqual({ name: "A", value: "b=c" });
  const result = await invoke(() => processes.run(123 as unknown as string, [], options()), origin);
  expect(result.kind).toBe("standard");
});
test("abandoned runs die with their scope", async () => {
  let observed: Completion<unknown> | undefined;
  await runOwnedRoot(() =>
    withScope(async () => {
      const pending = processes.run("/bin/sleep", ["30"], options({ grace: 50n }));
      void pending.then(
        (completion) => {
          observed = completion;
        },
        (cause) => {
          observed = cause as Completion<unknown>;
        },
      );
      return success(undefined);
    }),
  );
  for (let i = 0; i < 100 && observed === undefined; i++) await Bun.sleep(50);
  expect(observed?.kind).toBe("ok");
  if (observed?.kind !== "ok") throw Error();
  expect(dataProperty(observed.value, "code")).toBe(-1n);
  expect(dataProperty(observed.value, "signal")).not.toBe("");
});
