import { test, expect } from "bun:test";
import { success } from "../completion.ts";
import { assertionContext, callContext } from "../assert/context.ts";
import { withFixture } from "../assert/fixtures.ts";
import { runAssertion } from "../assert/runner.ts";

const app = "can.project.root/app";
const helper = "can.project.root/helper";
const scenario = helper + "::checkout";
const origin = { source: "can:test", start: 0, end: 0, invocation: [] };

test("caller label renames never reselect another package's rows", async () => {
  const rows = [
    {
      selector: "customer",
      owner: helper,
      arguments: async () => success([7]),
      expected: async () => success("fixture"),
    },
  ];
  for (const name of ["customer", "renamed"]) {
    let ranReal = 0;
    const report = await runAssertion({
      root: { package: app, declaration: app + "::read_customer", name },
      expected: async () => success("real"),
      actual: async (context) =>
        withFixture(
          context,
          "table",
          rows,
          [7],
          async () => {
            ranReal++;
            return success("real");
          },
          origin,
        ),
    });
    expect(report.passed).toBe(true);
    expect(ranReal).toBe(1);
    expect(report.evidence).toEqual(["real-can"]);
    expect(report.violations).toEqual([]);
  }
});

test("same-owner rows still match same-package roots", async () => {
  let ranReal = 0;
  const report = await runAssertion({
    root: { package: helper, declaration: helper + "::read", name: "unit" },
    expected: async () => success("fixture"),
    actual: async (context) =>
      withFixture(
        context,
        "table",
        [
          {
            selector: "unit",
            owner: helper,
            arguments: async () => success([7]),
            expected: async () => success("fixture"),
          },
        ],
        [7],
        async () => {
          ranReal++;
          return success("real");
        },
        origin,
      ),
  });
  expect(report.passed).toBe(true);
  expect(ranReal).toBe(0);
  expect(report.evidence).toEqual(["real-can", "supplied-completion"]);
});

test("scenario rows activate only through an explicit link", async () => {
  const rows = [
    {
      selector: "checkout",
      owner: helper,
      scenario,
      arguments: async () => success([7]),
      expected: async () => success("fixture"),
    },
  ];
  for (const linked of [true, false]) {
    let ranReal = 0;
    const report = await runAssertion({
      root: {
        package: app,
        declaration: app + "::read_customer",
        name: "customer",
        ...(linked ? { links: [scenario] } : {}),
      },
      expected: async () => success(linked ? "fixture" : "real"),
      actual: async (context) =>
        withFixture(
          context,
          "table",
          rows,
          [7],
          async () => {
            ranReal++;
            return success("real");
          },
          origin,
        ),
    });
    expect(report.passed).toBe(true);
    expect(ranReal).toBe(linked ? 0 : 1);
    expect(report.violations).toEqual([]);
  }
});

test("whole-helper stubs stay available as caller-unit tests", async () => {
  let ranHelper = 0;
  const report = await runAssertion({
    root: { package: app, declaration: app + "::read_customer", name: "customer" },
    expected: async () => success("stubbed"),
    actual: async (context) =>
      withFixture(
        context,
        "table",
        [
          {
            selector: "customer",
            owner: app,
            arguments: async () => success([]),
            expected: async () => success("stubbed"),
          },
        ],
        [],
        async () => {
          ranHelper++;
          return success("real helper body");
        },
        origin,
      ),
  });
  expect(report.passed).toBe(true);
  expect(ranHelper).toBe(0);
  expect(report.evidence).toEqual(["real-can", "supplied-completion"]);
});

test("unused links fail the root as unused fixtures", async () => {
  const report = await runAssertion({
    root: {
      package: app,
      declaration: app + "::read_customer",
      name: "customer",
      links: [scenario],
    },
    expected: async () => success("real"),
    actual: async () => success("real"),
  });
  expect(report.passed).toBe(false);
  expect(report.violations).toEqual(["unused fixture"]);
  expect(report.fixturePaths).toHaveLength(1);
  expect(report.fixturePaths[0].reason).toBe("unused fixture");
  expect(report.fixturePaths[0].expected).toMatchObject({ table: scenario, row: 0 });
});

test("two sequential links consume rows in source FIFO order", async () => {
  const retry = helper + "::retry";
  const rows = [
    {
      selector: "checkout",
      owner: helper,
      scenario,
      arguments: async () => success([7]),
      expected: async () => success("first"),
    },
    {
      selector: "retry",
      owner: helper,
      scenario: retry,
      arguments: async () => success([7]),
      expected: async () => success("second"),
    },
  ];
  const consumed: unknown[] = [];
  const report = await runAssertion({
    root: {
      package: app,
      declaration: app + "::read_customer",
      name: "customer",
      links: [scenario, retry],
    },
    expected: async () => success(["first", "second"]),
    actual: async (context) => {
      for (let index = 0; index < 2; index++) {
        const next = await callContext(context, `${app}::read_customer#0`, (child) =>
          withFixture(child, "table", rows, [7], async () => success("real"), origin),
        );
        if (next.kind !== "ok") return next;
        consumed.push(next.value);
      }
      return success(consumed);
    },
  });
  expect(report.passed).toBe(true);
  expect(consumed).toEqual(["first", "second"]);
});

test("concurrent roots keep link selection isolated", async () => {
  const rows = [
    {
      selector: "checkout",
      owner: helper,
      scenario,
      arguments: async () => success([7]),
      expected: async () => success("fixture"),
    },
  ];
  const run = (name: string, links: readonly string[] | undefined, want: string) =>
    runAssertion({
      root: {
        package: app,
        declaration: app + "::read_customer",
        name,
        ...(links === undefined ? {} : { links }),
      },
      expected: async () => success(want),
      actual: async (context) =>
        withFixture(context, "table", rows, [7], async () => success("real"), origin),
    });
  const [linked, plain] = await Promise.all([
    run("customer", [scenario], "fixture"),
    run("other", undefined, "real"),
  ]);
  expect(linked.passed).toBe(true);
  expect(plain.passed).toBe(true);
  expect(linked.evidence).toEqual(["real-can", "supplied-completion"]);
  expect(plain.evidence).toEqual(["real-can"]);
});

test("scenario rows in raw mode execute the real target and check the outcome", async () => {
  const row = (want: string) => ({
    selector: "checkout",
    owner: helper,
    scenario,
    arguments: async () => success([7]),
    expected: async () => success(want),
    raw: {
      operation: helper + "::load",
      spec: { environment: {}, exchange: null } as const,
    },
  });
  let ranReal = 0;
  const passing = await runAssertion({
    root: {
      package: app,
      declaration: app + "::read_customer",
      name: "customer",
      links: [scenario],
    },
    expected: async () => success("real"),
    actual: async (context) =>
      withFixture(
        context,
        "table",
        [row("real")],
        [7],
        async () => {
          ranReal++;
          return success("real");
        },
        origin,
      ),
  });
  expect(passing.passed).toBe(true);
  expect(ranReal).toBe(1);
  const failing = await runAssertion({
    root: {
      package: app,
      declaration: app + "::read_customer",
      name: "customer",
      links: [scenario],
    },
    expected: async () => success("real"),
    actual: async (context) =>
      withFixture(context, "table", [row("other")], [7], async () => success("real"), origin),
  });
  expect(failing.passed).toBe(false);
  expect(failing.violations).toEqual(["outcome mismatch"]);
});

test("scenario links are validated and duplicate-free", () => {
  const root = { package: app, declaration: app + "::read_customer", name: "customer" };
  expect(() => assertionContext({ ...root, links: [scenario, scenario] })).toThrow();
  expect(() => assertionContext({ ...root, links: [""] })).toThrow();
  expect(() => assertionContext({ ...root, links: "nope" as unknown as string[] })).toThrow();
  expect(() => assertionContext({ ...root, links: [] })).not.toThrow();
});
