import { test, expect } from "bun:test";
import { runAssertion } from "../assert/runner.ts";
import { callContext } from "../assert/context.ts";
import { provideHTTP, providerHTTP } from "../assert/provider.ts";
import { success } from "../completion.ts";
const origin = { source: "can:test", start: 0, end: 0, invocation: [] };
test("raw provider fixtures remain root-local across generated child context views", async () => {
  const report = await runAssertion({
    root: { package: "p", declaration: "p::main", name: "wire" },
    expected: async () => success(201),
    actual: async (context) => {
      const body = new Uint8Array();
      provideHTTP(
        context,
        [200, 201].map((status) => ({
          request: { method: "POST", url: "https://fixture.invalid/", headers: [], body },
          response: { status, headers: [], body },
        })),
      );
      let status = 0;
      for (let i = 0; i < 2; i++)
        status = await callContext(context, "p::main#0", async (child) => {
          const exchange = providerHTTP(child, origin);
          expect(exchange).toBeDefined();
          return (await exchange!(new URL("https://fixture.invalid/"), { method: "POST", body }))
            .status;
        });
      return success(status);
    },
  });
  expect(report.passed).toBe(true);
  expect(report.evidence).toEqual(["raw-provider-fixture", "real-can"]);
});
