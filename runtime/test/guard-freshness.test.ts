import { test, expect } from "bun:test";

// The checked-in guard bytes are a generated mirror: they must stay
// byte-identical to the Bun transpiler output over
// runtime/platform/htmx-guard.ts, using the same settings as the pin test
// in html.test.ts. Regenerate with `bun tools/runtime/guardgen.ts`.
test("checked-in guard bytes match the transpiler output", async () => {
  const source = await Bun.file(new URL("../platform/htmx-guard.ts", import.meta.url)).text();
  const served = new Bun.Transpiler({ loader: "ts" }).transformSync(source);
  const checkedIn = await Bun.file(
    new URL("../../distribution/assets/htmx-guard.js", import.meta.url),
  ).bytes();
  expect(checkedIn).toEqual(new TextEncoder().encode(served));
});
