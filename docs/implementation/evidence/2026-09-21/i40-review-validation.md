# I40 independent review corrections

Two confirmed report paths lacked Can locations: assertion suite initialization
failure, and sticky missing-fixture violations created inside the CLI adapter
without a native cause. Both now use the existing sanitized map boundary.

Suite setup reports mapped frames. Runtime-created synthetic standard failures
retain their original occurrence, origin, category and message; the first checked
invocation origin is separate private metadata. Later callers cannot replace it.
Immediate, awaited and rejected carrier paths preserve the same occurrence.
Assertion contexts retain violation tokens, so source handling cannot discard
the location or turn the harness result into success. Unused fixture rows also
retain their table origin. Neither raw stack text nor payload values are exposed.

Validation:

- `TestAssertionFailureLocations` builds a fresh release and invokes its absolute
  compiler/private Bun with networking denied. It checks exact byte spans and
  line/columns for top-level `1 / 0` and a caught `io::stdout_write(payload)`
  missing-fixture failure. The report contains no host paths, native message,
  synthetic adapter source or application text.
- [Offline staged suites](i40-review-offline-tests.txt): integration, driver and
  emitter, including the two exact-location regressions.
- [Full Go suite](i40-review-go-tests.txt).
- [Native runtime tests](i40-review-runtime-tests.txt): 42 pass, 1,240 expectations.
- [Strict TypeScript](i40-review-typescript-tests.txt): all runtime modules/tests.
- [Three fresh design consultations](i40-review-jev/README.md).

This correction does not expand I18's fixture admission or queue semantics.
I08 was inspected but no callable implementation changes preceded these fixes.
