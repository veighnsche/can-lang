# Audit evidence at 8312d85

**Current follow-up:** [Workstream A replacement audit and JEV review](workstream-a/README.md),
completed at `8bbcf13` on 2026-09-19. It reproduces F01–F05, adds F06
(named-argument fault-order mismatch), and preserves all thirteen new review
packets with explicit dispositions. The original evidence below remains history.

Start with the **[master TODO checklist](TODO.md)** for priorities, dependencies,
completion criteria, and coverage of every audit section.

These probes substantiate `../can-language-audit.md`. They are not normal
regression gates: the Go probe currently expects the documented defects to be
present. After a fix, a failed hypothesis is a reason to update the audit status,
not restore the bug. Turn each repaired finding into an ordinary negative
regression test in the repair commit.

## Compiler/evidence probes

From the repo root, use a temporary test filename, refusing to overwrite an
existing file and cleaning up even on failure:

```sh
test ! -e compiler/z_audit_probe_test.go && (
  trap 'rm -f compiler/z_audit_probe_test.go' EXIT
  cp docs/audit-probes/evidence_test.go.txt compiler/z_audit_probe_test.go
  go test ./compiler -count=1 -v \
    -run '^TestAudit(CanonicalEvidenceProbe|DuplicateRecordProbe)$'
)
```

Observed results:

```text
PIN_TARGET_COLLISION
  old=one() => ctor(Ok)[value=unknown-kind(fnref)]
  new=one() => ctor(Ok)[value=unknown-kind(fnref)]
  warnings=[]
TYPED_OK_CANON_EQUAL=true
TYPED_PATTERN_CANON_EQUAL=true
INVOKE_ARGUMENT_CANON_EQUAL=true
DUPLICATE_RECORD diagnostics=[] shape=[[value int]]
DUPLICATE_RECORD_REORDERED diagnostics=[] shape=[[value str]]
```

The callback probe checks two complete, green programs with different callable
targets and a pinned factory expectation, using the real accepted-baseline
comparison. Type-argument and invocation-argument comparisons are narrower
structural probes; they do not establish an exploited persisted proof-cache
bypass. The record probe checks the real program pipeline in both source orders.

## Host boundary

```sh
node --experimental-strip-types docs/audit-probes/host-boundary.mjs
```

Observed:

```text
ABI_ALIAS true 9
GETTER_INPUT { '$can_kind': 'ok', left: 7n, right: true } 2
HOST_CALLBACK_EFFECTS 1
```

This demonstrates the current direct/trusted JS boundary, not a violation of a
promised hostile-host sandbox. Readonly declarations alone would not eliminate
these behaviors. The callback probe demonstrates that a raw JS function can
enter an exported Fn parameter; source-level host-callback exclusions are not
an embedding-boundary purity validator.

## Whole-library and target integration

These commands intentionally expose additional baseline failures:

```sh
go run ./compiler --out /tmp/can-audit-stdlib $(find std -name '*.can' | sort)
# Fails: scalars.can:1016: math.nonterminating_decimal field numerator: got dec, want int

./tscheck/node_modules/.bin/tsc --noEmit -p tscheck/tsconfig.json
# Fails: missing relative provider artifacts plus JSON typing errors.

go run ./compiler --out /tmp/can-audit-json std/json/json.can std/scalars/scalars.can
# Succeeds: 785 tests passed, both modules emitted.
./tscheck/node_modules/.bin/tsc --noEmit --strict --target ES2020 \
  --module ESNext --moduleResolution bundler /tmp/can-audit-json/*.ts
# Fails independently of missing provider artifacts:
# TS2440 imported Bool__Value/Dec__Value/Int__Value/Str__Value conflict with local declarations.
# TS2322 JSON frame/tag constructions widen discriminants to string (json.ts:308 and more).
```

The all-stdlib failure has two incompatible definitions of the same global
`math.nonterminating_decimal`: ratio uses int numerator/denominator; scalars
uses dec dividend/divisor. Keep missing-artifact packaging failures distinct
from wrong types in a complete freshly emitted bundle.

## JEV records

**Withdrawn as decision support: insufficient Can context was supplied.** JEV
cannot research the repository, inspect linked files, or supply missing context.
The compressed summaries did not adequately support an independent design review.

`jev-decisions.json` and `jev-surface-decisions.json` retain the original
nonsecret requests and returned distributions (`jev-1.13.0`, Choice) for audit
history only. Do not use their scores to justify recommendations or approval.
The compiler reproductions above remain independent evidence; redesign
recommendations remain the audit author's proposals pending proper review.

See [TODO section A](TODO.md#a-redo-the-jev-review-correctly) for the replacement
review's context and completion requirements.

The replacement review is now complete: see its [results and dispositions](workstream-a/README.md)
and [method](workstream-a/method.md). The original two JSON files stay withdrawn;
new responses are in `workstream-a/responses/`. New scores are advice, not approval.
