// Targeted runtime probe, not a language implementation change.
import { expect, test } from "bun:test";
import { registerResource, useResource, withScope, runOwnedRoot, registerCallableCaptures } from "../../../../../runtime/owner.ts";
import { invoke, success, value } from "../../../../../runtime/completion.ts";
import { standardFailureDiagnostics } from "../../../../../runtime/failure.ts";
import { record, dataProperty } from "../../../../../runtime/data.ts";

const origin = { source: "gap-verification", start: 0, end: 0, invocation: [] };
for (const shape of ["direct", "record", "array", "callable"] as const) {
  test(`escaped ${shape}: in-scope use succeeds, post-scope work is rejected`, async () => {
    let nativeUses = 0;
    const root = await runOwnedRoot(async () => {
      const escaped = await withScope(async () => {
        const handle = registerResource("probe", {}, async () => success(undefined), { scopeManaged: true });
        const use = () => useResource(handle, "probe", async () => { nativeUses++; return success(7n); });
        expect(value(await use())).toBe(7n);
        if (shape === "record") return success(record("probe::held", [["handle", handle]]));
        if (shape === "array") return success(Object.freeze([handle]));
        if (shape === "callable") return success(registerCallableCaptures(use, [handle]));
        return success(handle);
      });
      expect(escaped.kind).toBe("ok");
      const held = value(escaped);
      const attempt = await invoke(async () => {
        if (shape === "callable") return (held as Function)();
        const handle = shape === "record" ? dataProperty(held, "handle") : shape === "array" ? (held as unknown[])[0] : held;
        return useResource(handle, "probe", async () => { nativeUses++; return success(99n); });
      }, origin);
      expect(attempt.kind).toBe("standard");
      if (attempt.kind === "standard") expect(standardFailureDiagnostics(attempt.value).kind).toBe("resource_state");
      expect(nativeUses).toBe(1);
      return success(undefined);
    });
    expect(root.completion.kind).toBe("ok");
    expect(root.cleanupFailed).toBe(false);
  });
}
