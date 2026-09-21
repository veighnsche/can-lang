package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Slice 1: named scalar-literal constants. Probes first: parse,
// naming, duplicates, literal-only initializers, uses/provides,
// lazy resolution in eval/emit, syntactic termination.

// hasError reports whether any diagnostic is an error: probes
// asserting clean builds ignore warnings (unused params still
// warn when a body ignores its row, const or not).
func hasError(diags []Diag) bool {
	for _, d := range diags {
		if d.Sev == "error" {
			return true
		}
	}
	return false
}

const constDeclSrc = `mod m
  provides [m__go, M__Out, m__COLON]
  uses []
  emits []

const m__COLON: int rev 1 = 58
`

// TestConstParses pins the declaration shape: legacyParsePaths yields a
// ConstDecl carrying name, type, revision, and literal value.
func TestConstParses(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "m.can")
	if err := os.WriteFile(fp, []byte(constDeclSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	mods, _, _, err := legacyParsePaths([]string{fp})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	var found *ConstDecl
	for _, d := range mods[0].Decls {
		if c, ok := d.(*ConstDecl); ok {
			found = c
		}
	}
	if found == nil {
		t.Fatalf("no ConstDecl parsed")
	}
	if found.Name != "m__COLON" || found.Type != "int" || found.Rev != 1 {
		t.Fatalf("bad decl: %+v", found)
	}
	if found.Value == nil || found.Value.Kind != "int" || found.Value.Num.String() != "58" {
		t.Fatalf("bad value: %+v", found.Value)
	}
}

// TestConstNaming pins CAN2003: const names are domain__SCREAMING.
func TestConstNaming(t *testing.T) {
	src := `mod m
  provides [m__go, Foo]
  uses []
  emits []

const Foo: int rev 1 = 1
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); !hasCode(diags, "CAN2003") {
		t.Fatalf("bad const name reported no CAN2003: %v", diags)
	}
}

// TestConstNamingMulti pins the decided stdlib convention: the
// domain is the first segment, so std__ascii__COLON is accepted
// (no CAN2003) while the SCREAMING tail stays mandatory.
func TestConstNamingMulti(t *testing.T) {
	src := `mod m
  provides [m__go, std__ascii__COLON]
  uses []
  emits []

const std__ascii__COLON: int rev 1 = 58
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); hasCode(diags, "CAN2003") {
		t.Fatalf("multi-segment const name reported CAN2003: %v", diags)
	}
}

// TestConstDuplicate pins CAN2205 for same-module duplicates.
func TestConstDuplicate(t *testing.T) {
	src := `mod m
  provides [m__go, m__A]
  uses []
  emits []

const m__A: int rev 1 = 1

const m__A: int rev 1 = 2
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); !hasCode(diags, "CAN2205") {
		t.Fatalf("duplicate const reported no CAN2205: %v", diags)
	}
}

// TestConstLiteralOnly pins CAN6016: initializers are literals,
// never computed, never aliases.
func TestConstLiteralOnly(t *testing.T) {
	for name, init := range map[string]string{"computed": "1 + 2", "alias": "m__A"} {
		src := `mod m
  provides [m__go, m__A, m__B]
  uses []
  emits []

const m__A: int rev 1 = 1

const m__B: int rev 1 = ` + init + `
`
		dir := writeLSPDir(t, map[string]string{"m.can": src})
		if diags := diagnose(dir, "m.can", src); !hasCode(diags, "CAN6016") {
			t.Fatalf("%s initializer reported no CAN6016: %v", name, diags)
		}
	}
}

// TestConstUseBeforeDecl pins order independence: a use above the
// declaration resolves by qualified name.
func TestConstUseBeforeDecl(t *testing.T) {
	src := `mod m
  provides [m__go, M__Out, m__N]
  uses []
  emits []

type M__Out rev 1 (
  value: int
)

fn m__go(x: int) -> M__Out rev 1
  emits []
  tests
    go(1) => Ok(58)
  Ok(m__N)

const m__N: int rev 1 = 58
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); hasError(diags) {
		t.Fatalf("use-before-decl reported: %v", diags)
	}
}

// TestConstExecutes pins lazy resolution through execution: the
// test passes only if the const value flows into eval, and the
// emitted program inlines the literal.
func TestConstExecutes(t *testing.T) {
	src := `mod m
  provides [m__go, M__Out, m__N]
  uses []
  emits []

type M__Out rev 1 (
  value: int
)

const m__N: int rev 1 = 58

fn m__go(x: int) -> M__Out rev 1
  emits []
  tests
    go(1) => Ok(58)
  Ok(m__N)
`
	dir := t.TempDir()
	fp := filepath.Join(dir, "m.can")
	if err := os.WriteFile(fp, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := compile(out, []string{fp}); err != nil {
		t.Fatalf("const program failed to compile: %v", err)
	}
	ts, err := os.ReadFile(filepath.Join(out, "m.ts"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ts), "58n") {
		t.Fatalf("emitted program does not inline the literal: %s", ts)
	}
}

// TestConstExecutesValue pins the value, not just success: a wrong
// expectation fails, proving the const (not a default) flowed in.
func TestConstExecutesValue(t *testing.T) {
	src := `mod m
  provides [m__go, M__Out, m__N]
  uses []
  emits []

type M__Out rev 1 (
  value: int
)

const m__N: int rev 1 = 58

fn m__go(x: int) -> M__Out rev 1
  emits []
  tests
    go(1) => Ok(59)
  Ok(m__N)
`
	dir := t.TempDir()
	fp := filepath.Join(dir, "m.can")
	if err := os.WriteFile(fp, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := compile(filepath.Join(dir, "out"), []string{fp}); err == nil {
		t.Fatalf("wrong expectation compiled clean: const value did not flow into eval")
	}
}

// TestConstUnknown pins CAN2104 for references that resolve nowhere.
func TestConstUnknown(t *testing.T) {
	src := `mod m
  provides [m__go, M__Out]
  uses []
  emits []

type M__Out rev 1 (
  value: int
)

fn m__go(x: int) -> M__Out rev 1
  emits []
  tests
    go(1) => Ok(1)
  Ok(m__NOPE)
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); !hasCode(diags, "CAN2104") {
		t.Fatalf("unknown const reported no CAN2104: %v", diags)
	}
}

// TestConstMissingUses pins CAN2105: a foreign const without a pin.
func TestConstMissingUses(t *testing.T) {
	lib := `mod lib
  provides [lib__K]
  uses []
  emits []

const lib__K: int rev 1 = 7
`
	app := `mod app
  provides [app__go, A__Out]
  uses []
  emits []

type A__Out rev 1 (
  value: int
)

fn app__go(x: int) -> A__Out rev 1
  emits []
  tests
    go(1) => Ok(7)
  Ok(lib__K)
`
	dir := writeLSPDir(t, map[string]string{"lib.can": lib, "app.can": app})
	if diags := diagnose(dir, "app.can", app); !hasCode(diags, "CAN2105") {
		t.Fatalf("unpinnned foreign const reported no CAN2105: %v", diags)
	}
}

// TestConstUsesPin pins the open gate: a pinned foreign const
// resolves across files.
func TestConstUsesPin(t *testing.T) {
	lib := `mod lib
  provides [lib__K]
  uses []
  emits []

const lib__K: int rev 1 = 7
`
	app := `mod app
  provides [app__go, A__Out]
  uses [lib__K@1]
  emits []

type A__Out rev 1 (
  value: int
)

fn app__go(x: int) -> A__Out rev 1
  emits []
  tests
    go(1) => Ok(7)
  Ok(lib__K)
`
	dir := writeLSPDir(t, map[string]string{"lib.can": lib, "app.can": app})
	if diags := diagnose(dir, "app.can", app); hasError(diags) {
		t.Fatalf("pinned foreign const reported: %v", diags)
	}
}

// TestConstProvidesMiss pins provides coverage for the new decl.
func TestConstProvidesMiss(t *testing.T) {
	src := `mod m
  provides [m__go]
  uses []
  emits []

const m__K: int rev 1 = 7
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); !hasCode(diags, CodeProvidesMiss) {
		t.Fatalf("unprovided const reported no CAN2301: %v", diags)
	}
}

// TestConstPatternBoolStr pins const patterns in value matches: a
// declared bool/str constant behaves exactly like its literal.
func TestConstPatternBoolStr(t *testing.T) {
	src := `mod m
  provides [m__go, M__Out, m__FLAG, m__SEP]
  uses []
  emits []

type M__Out rev 1 (
  value: str
)

const m__FLAG: bool rev 1 = true

const m__SEP: str rev 1 = ", "

fn m__go(x: int, s: str) -> M__Out rev 1
  emits []
  tests
    one(1, ", ") => Ok("yes")
    two(2, ", ") => Ok("sep")
    three(2, ";") => Ok("no")
  match x <= 1
    m__FLAG => Ok("yes")
    false => match s
      m__SEP => Ok("sep")
      _ => Ok("no")
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); hasError(diags) {
		t.Fatalf("const patterns reported: %v", diags)
	}
}

// TestConstPatternInt pins slice-3 admission: integer constants
// are patterns now (the range slice landed), behaving exactly
// like their literal.
func TestConstPatternInt(t *testing.T) {
	src := `mod m
  provides [m__go, M__Out, m__N]
  uses []
  emits []

type M__Out rev 1 (
  value: int
)

const m__N: int rev 1 = 58

fn m__go(x: int) -> M__Out rev 1
  emits []
  tests
    colon(58) => Ok(2)
    other(1) => Ok(3)
  match x
    m__N => Ok(2)
    _ => Ok(3)
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); hasError(diags) {
		t.Fatalf("int const pattern reported: %v", diags)
	}
}

// TestConstEvidenceMarksUsed pins the unused-import interplay: a
// constant referenced only in evidence still marks its pin used.
func TestConstEvidenceMarksUsed(t *testing.T) {
	lib := `mod lib
  provides [lib__K]
  uses []
  emits []

const lib__K: int rev 1 = 7
`
	app := `mod app
  provides [app__go, A__Out]
  uses [lib__K@1]
  emits []

type A__Out rev 1 (
  value: int
)

fn app__go(x: int) -> A__Out rev 1
  emits []
  tests
    go(lib__K) => Ok(7)
  Ok(x)
`
	dir := writeLSPDir(t, map[string]string{"lib.can": lib, "app.can": app})
	if diags := diagnose(dir, "app.can", app); hasCode(diags, "CAN3401") {
		t.Fatalf("evidence-only const pin reported unused: %v", diags)
	}
}

// TestConstPatternForeignPin pins CAN2105 for patterns: a foreign
// constant in pattern position needs its rev pin like a body
// reference.
func TestConstPatternForeignPin(t *testing.T) {
	lib := `mod lib
  provides [lib__FLAG]
  uses []
  emits []

const lib__FLAG: bool rev 1 = true
`
	app := `mod app
  provides [app__go, A__Out]
  uses []
  emits []

type A__Out rev 1 (
  value: str
)

fn app__go(x: int) -> A__Out rev 1
  emits []
  tests
    one(1) => Ok("yes")
    two(2) => Ok("no")
  match x <= 1
    lib__FLAG => Ok("yes")
    false => Ok("no")
`
	dir := writeLSPDir(t, map[string]string{"lib.can": lib, "app.can": app})
	if diags := diagnose(dir, "app.can", app); !hasCode(diags, "CAN2105") {
		t.Fatalf("unpinned foreign const pattern reported no CAN2105: %v", diags)
	}
}

// TestConstPatternMarksUsed pins the pattern evidence interplay: a
// pinned foreign const referenced only in a pattern still marks
// its pin used (no CAN3401).
func TestConstPatternMarksUsed(t *testing.T) {
	lib := `mod lib
  provides [lib__FLAG]
  uses []
  emits []

const lib__FLAG: bool rev 1 = true
`
	app := `mod app
  provides [app__go, A__Out]
  uses [lib__FLAG@1]
  emits []

type A__Out rev 1 (
  value: str
)

fn app__go(x: int) -> A__Out rev 1
  emits []
  tests
    one(1) => Ok("yes")
    two(2) => Ok("no")
  match x <= 1
    lib__FLAG => Ok("yes")
    false => Ok("no")
`
	dir := writeLSPDir(t, map[string]string{"lib.can": lib, "app.can": app})
	if diags := diagnose(dir, "app.can", app); hasCode(diags, "CAN3401") {
		t.Fatalf("pattern-only const pin reported unused: %v", diags)
	}
}

// TestConstIdentityDrift pins canonConst (R4 clarification): same
// name and rev with a changed value is baseline drift ("value
// changed") — a changed value cannot hide behind its spelling.
func TestConstIdentityDrift(t *testing.T) {
	base := `mod m
  provides [m__N]
  uses []
  emits []

const m__N: int rev 1 = 58
`
	files := map[string]string{"m.can": base}
	progB, _ := revisionProg(t, files, []string{"m.can"})
	bsln := revisionBaseline(t, progB, "review-base:B")
	changed := `mod m
  provides [m__N]
  uses []
  emits []

const m__N: int rev 1 = 59
`
	progC, textsC := revisionProg(t, map[string]string{"m.can": changed}, []string{"m.can"})
	diags := CheckRevisionIdentity(progC, textsC, bsln)
	if !hasDiag(diags, "error", "differs from accepted baseline") {
		t.Fatalf("expected value drift, got %v", diags)
	}
	if !hasFound(diags, "value changed") {
		t.Fatalf("expected value-changed detail, got %v", diags)
	}
}

// TestConstIdentityClean pins the quiet path: an unchanged const
// compares clean against its own baseline.
func TestConstIdentityClean(t *testing.T) {
	src := `mod m
  provides [m__N]
  uses []
  emits []

const m__N: int rev 1 = 58
`
	files := map[string]string{"m.can": src}
	prog, texts := revisionProg(t, files, []string{"m.can"})
	bsln := revisionBaseline(t, prog, "review-base:B")
	if diags := CheckRevisionIdentity(prog, texts, bsln); len(diags) != 0 {
		t.Fatalf("expected clean identity, got %v", diags)
	}
}

// TestConstStaysSyntactic pins the termination rule: `n - m__ONE`
// (const m__ONE = 1) is not another spelling of the certified
// `n - 1` step, so the guarded recursion must fail.
func TestConstStaysSyntactic(t *testing.T) {
	src := `mod m
  provides [m__poll, M__S, m__ONE]
  uses []
  emits []

type M__S rev 1 (
  n: int
)

const m__ONE: int rev 1 = 1

fn m__poll(n: int) -> M__S rev 1
  decreases n
  emits []
  tests
    now(0) => Ok(0)
    later(2) => Ok(0)
  match n <= 0
    true => Ok(0)
    false => match call m__poll(n - m__ONE)
      on Ok s => Ok(s.n)
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	diags := diagnose(dir, "m.can", src)
	if len(diags) == 0 {
		t.Fatalf("const-spelled step certified the recursion")
	}
}

// Composite constants (design3 closed data): records, sequences,
// and brand seals as named fixtures shared across test rows. A
// row naming a fixture evaluates exactly its literal, so rows
// stay short while outcomes stay identical.

// TestCompositeConstRecord pins the fixture shape: a record
// const referenced from test args checks clean, and the const
// row and the inline-literal row pass with the same outcome.
func TestCompositeConstRecord(t *testing.T) {
	src := `mod m
  provides [m__go, M__Point, M__Bool, m__ORIGIN]
  uses []
  emits []

type M__Point rev 1 (
  x: int,
  y: int
)

type M__Bool rev 1 (
  value: bool
)

const m__ORIGIN: M__Point rev 1 = M__Point(0, 0)

fn m__go(p: M__Point) -> M__Bool rev 1
  emits []
  tests
    via_const(m__ORIGIN) => Ok(true)
    via_inline(M__Point(0, 0)) => Ok(true)
    elsewhere(M__Point(1, 2)) => Ok(false)
  match p.x == 0, p.y == 0
    true, true => Ok(true)
    _, _ => Ok(false)
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); hasError(diags) {
		t.Fatalf("record fixture reported errors: %v", diags)
	}
}

// TestCompositeConstNested pins fixtures nesting Seq literals
// and brand seals: the snapshot-shaped const (entries plus an
// empty revoked list plus a sealed mark) checks clean and its
// row selects through projection exactly like inline data.
func TestCompositeConstNested(t *testing.T) {
	src := `mod m
  provides [m__go, M__Entry, M__Snap, M__Tag, M__Bool, m__SNAP, m__REVL0]
  uses []
  emits []

type M__Entry rev 1 (
  id: str,
  n: int
)

type M__Snap rev 1 (
  entries: Seq<M__Entry>,
  revoked: Seq<M__Entry>,
  mark: M__Tag
)

brand M__Tag is str rev 1

type M__Bool rev 1 (
  value: bool
)

const m__SNAP: M__Snap rev 1 = M__Snap(Seq<M__Entry>[M__Entry(id = "a", n = 1)], Seq<M__Entry>[], seal M__Tag("ok"))

const m__REVL0: Seq<M__Entry> rev 1 = Seq<M__Entry>[]

fn m__go(s: M__Snap, r: Seq<M__Entry>) -> M__Bool rev 1
  emits []
  tests
    nested(m__SNAP, m__REVL0) => Ok(true)
    renamed(M__Snap(Seq<M__Entry>[], Seq<M__Entry>[], seal M__Tag("no")), m__REVL0) => Ok(false)
  match s.mark == seal M__Tag("ok"), #s.entries == 1, #r == 0
    true, true, true => Ok(true)
    _, _, _ => Ok(false)
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); hasError(diags) {
		t.Fatalf("nested fixture reported errors: %v", diags)
	}
}

// TestCompositeConstShapes pins CAN6016 on every non-data
// initializer and every excluded sort: calls, arithmetic,
// aliases, cross-shape heads, unknown sorts, Bytes, variants.
func TestCompositeConstShapes(t *testing.T) {
	for name, decl := range map[string]string{
		"call":    `const m__B: M__Point rev 1 = m__go(p = m__ORIGIN)`,
		"arith":   `const m__B: M__Point rev 1 = M__Point(x = 1 + 2, y = 0)`,
		"alias":   `const m__B: M__Point rev 1 = m__ORIGIN`,
		"seqhead": `const m__B: M__Point rev 1 = Seq<M__Point>[]`,
		"sealhead": `const m__B: M__Tag rev 1 = M__Point(x = 0, y = 0)`,
		"ctorhead": `const m__B: Seq<M__Point> rev 1 = M__Point(x = 0, y = 0)`,
		"unknown": `const m__B: Nope rev 1 = 1`,
		"bytes":   `const m__B: Bytes rev 1 = Bytes(Seq<int>[65])`,
	} {
		src := `mod m
  provides [m__go, M__Point, M__Bool, M__Tag, m__ORIGIN, m__B]
  uses []
  emits []

type M__Point rev 1 (
  x: int,
  y: int
)

type M__Bool rev 1 (
  value: bool
)

brand M__Tag is str rev 1

const m__ORIGIN: M__Point rev 1 = M__Point(x = 0, y = 0)

` + decl + `

fn m__go(p: M__Point) -> M__Bool rev 1
  emits []
  tests
    origin(m__ORIGIN) => Ok(true)
  Ok(p.x == 0)
`
		dir := writeLSPDir(t, map[string]string{"m.can": src})
		if diags := diagnose(dir, "m.can", src); !hasCode(diags, "CAN6016") {
			t.Fatalf("%s initializer reported no CAN6016: %v", name, diags)
		}
	}
}

// TestCompositeConstVariant pins the design3 exclusion: a
// variant-typed constant stays inline with CAN6016.
func TestCompositeConstVariant(t *testing.T) {
	src := `mod m
  provides [m__go, M__State, m__B]
  uses []
  emits []

variant M__State rev 1 (
  case Ready()
  case Waiting()
)

const m__B: M__State rev 1 = M__State.Ready()
`
	dir := writeLSPDir(t, map[string]string{"m.can": src})
	if diags := diagnose(dir, "m.can", src); !hasCode(diags, "CAN6016") {
		t.Fatalf("variant initializer reported no CAN6016: %v", diags)
	}
}

// TestCompositeConstFields pins shared field checking: unknown,
// repeated, missing, and mistyped constructor fields fail inside
// const values exactly as in handwritten rows (CAN6003).
func TestCompositeConstFields(t *testing.T) {
	for name, init := range map[string]string{
		"unknown":  `M__Point(x = 0, z = 0)`,
		"repeated": `M__Point(x = 0, x = 1, y = 0)`,
		"missing":  `M__Point(0)`,
		"mistyped": `M__Point(x = "s", y = 0)`,
	} {
		src := `mod m
  provides [m__go, M__Point, M__Bool, m__B]
  uses []
  emits []

type M__Point rev 1 (
  x: int,
  y: int
)

type M__Bool rev 1 (
  value: bool
)

const m__B: M__Point rev 1 = ` + init + `

fn m__go(p: M__Point) -> M__Bool rev 1
  emits []
  tests
    origin(m__B) => Ok(true)
  Ok(p.x == 0)
`
		dir := writeLSPDir(t, map[string]string{"m.can": src})
		if diags := diagnose(dir, "m.can", src); !hasCode(diags, "CAN6003") {
			t.Fatalf("%s field reported no CAN6003: %v", name, diags)
		}
	}
}
