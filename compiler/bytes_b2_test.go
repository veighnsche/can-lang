package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a46 S2 (B2): owner-authorized typed UTF-8 export. A grant
// (exports_utf8 Brand via fn@rev) authorizes one exact-shape function
// to disclose one brand as Bytes through the bytes__utf8__export
// kernel. E-rows are the scope-doc acceptance rows. Single-module
// rows reuse seqClean/seqCode; multi-module order rows run the real
// CLI pipeline (legacyParsePaths + checkProgram) in both input orders.

// checkTwo runs the real CLI pipeline over files written to a temp dir
// in paths order and returns every diagnostic with codes. Paths may
// name subdirectories (same-basename attacks); order is paths order.
func checkTwo(t *testing.T, paths []string, files map[string]string) []Diag {
	t.Helper()
	dir := t.TempDir()
	full := make([]string, 0, len(paths))
	for _, p := range paths {
		fp := filepath.Join(dir, p)
		if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fp, []byte(files[p]), 0o644); err != nil {
			t.Fatal(err)
		}
		full = append(full, fp)
	}
	mods, texts, collected, err := legacyParsePaths(full)
	if err != nil {
		t.Fatalf("legacyParsePaths: %v", err)
	}
	_, collected = checkProgram(mods, texts, collected, nil)
	return collected
}

func hasErrCode(diags []Diag, code string) bool {
	for _, d := range diags {
		if d.Sev == "error" && d.Code == code {
			return true
		}
	}
	return false
}

func onlyCodes(diags []Diag, codes ...string) bool {
	allowed := map[string]bool{}
	for _, c := range codes {
		allowed[c] = true
	}
	for _, d := range diags {
		if d.Sev == "error" && !allowed[d.Code] {
			return false
		}
	}
	return true
}

// bytesPub is the canonical granted exporter with byte-correctness
// rows: empty, ASCII, two-byte, astral, NUL, BOM. NULROW/BOMROW carry
// raw bytes (the only .can spelling) and are spliced in by tests.
const bytesPub = `mod pub
  provides [Pub__Doc, pub__export]
  uses []
  emits []

brand Pub__Doc is str rev 1

exports_utf8 Pub__Doc via pub__export@1

fn pub__export(document: Pub__Doc) -> Bytes__Value rev 1
  emits []
  tests
    empty(seal Pub__Doc("")) => Ok(Bytes(Seq<int>[]))
    ascii(seal Pub__Doc("A")) => Ok(Bytes(Seq<int>[65]))
    latin(seal Pub__Doc("é")) => Ok(Bytes(Seq<int>[195, 169]))
    astral(seal Pub__Doc("😀")) => Ok(Bytes(Seq<int>[240, 159, 152, 128]))
NULROW
BOMROW
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)
`

func bytesPubFull() string {
	nulRow := "    nul(seal Pub__Doc(\"a\x00b\")) => Ok(Bytes(Seq<int>[97, 0, 98]))\n"
	bomRow := "    bom(seal Pub__Doc(\"\uFEFFA\")) => Ok(Bytes(Seq<int>[239, 187, 191, 65]))\n"
	out := strings.Replace(bytesPub, "NULROW\n", nulRow, 1)
	return strings.Replace(out, "BOMROW\n", bomRow, 1)
}

// E0: the granted export is a value computation, byte-exact across
// the whole scalar range including preserved NUL and BOM.
func TestBytesE0GrantedExport(t *testing.T) {
	seqClean(t, map[string]string{"pub.can": bytesPubFull()}, "pub.can")
}

const bytesClientGood = `mod client
  provides [client__use]
  uses [pub__export@1, Pub__Doc@1]
  emits []

fn client__use(document: Pub__Doc) -> Bytes__Value rev 1
  emits []
  tests
    good(seal Pub__Doc("A")) => Ok(Bytes(Seq<int>[65]))
  match call pub__export(document)
    given
      good => [exchange args (document = seal Pub__Doc("A")) outcome Ok(Bytes(Seq<int>[65]))]
    on Ok r => Ok(r.value)
`

// E1: a correct scripted consumer passes under both module orders.
// Certificates issue before any linkage evaluation either way.
func TestBytesE1ClientBothOrders(t *testing.T) {
	files := map[string]string{"pub.can": bytesPubFull(), "client.can": bytesClientGood}
	for _, order := range [][]string{{"pub.can", "client.can"}, {"client.can", "pub.can"}} {
		diags := checkTwo(t, order, files)
		for _, d := range diags {
			if d.Sev == "error" {
				t.Fatalf("order %v: unexpected error %v", order, diags)
			}
		}
	}
}

const bytesClientLie = `mod client
  provides [client__use]
  uses [pub__export@1, Pub__Doc@1]
  emits []

fn client__use(document: Pub__Doc) -> Bytes__Value rev 1
  emits []
  tests
    wrong(seal Pub__Doc("A")) => Ok(Bytes(Seq<int>[66]))
  match call pub__export(document)
    given
      wrong => [exchange args (document = seal Pub__Doc("A")) outcome Ok(Bytes(Seq<int>[66]))]
    on Ok r => Ok(r.value)
`

// E2: an incorrect scripted export result contradicts (CAN3110) under
// both module orders. The exporter computes [65]; the script claims
// [66]. Order must not smuggle the lie through linkage trust.
func TestBytesE2ContradictionBothOrders(t *testing.T) {
	files := map[string]string{"pub.can": bytesPubFull(), "client.can": bytesClientLie}
	for _, order := range [][]string{{"pub.can", "client.can"}, {"client.can", "pub.can"}} {
		diags := checkTwo(t, order, files)
		if !hasErrCode(diags, CodeInconsistentScript) {
			t.Fatalf("order %v: expected CAN3110, got %v", order, diags)
		}
		if !hasDiag(diags, "error", "contradicts pub__export") {
			t.Fatalf("order %v: expected contradiction message, got %v", order, diags)
		}
	}
}

// E3: removing the grant fails the export call with CAN6010. Fresh
// programs carry no stale certificate.
func TestBytesE3GrantRemoval(t *testing.T) {
	body := strings.Replace(bytesPubFull(), "exports_utf8 Pub__Doc via pub__export@1\n\n", "", 1)
	seqCode(t, map[string]string{"pub.can": body}, "pub.can",
		CodeBytesExportAuthority, "not authorized")
}

// E5: grants that fail authority are CAN6010: revision mismatch,
// unknown targets, cross-module reach, and ambiguous brands.
func TestBytesE5GrantAuthority(t *testing.T) {
	grant := "exports_utf8 Pub__Doc via pub__export@1"
	rev := strings.Replace(bytesPubFull(), grant, "exports_utf8 Pub__Doc via pub__export@2", 1)
	seqCode(t, map[string]string{"pub.can": rev}, "pub.can",
		CodeBytesExportAuthority, "declares rev")
	brand := strings.Replace(bytesPubFull(), grant, "exports_utf8 Nope__X via pub__export@1", 1)
	seqCode(t, map[string]string{"pub.can": brand}, "pub.can",
		CodeBytesExportAuthority, "unknown brand")
	fn := strings.Replace(bytesPubFull(), grant, "exports_utf8 Pub__Doc via missing__fn@1", 1)
	// Two true facts here (the grant names nothing, and the export
	// call is therefore uncertified), so pin presence, not single.
	dir := writeLSPDir(t, map[string]string{"pub.can": fn})
	diags := diagnose(dir, "pub.can", fn)
	if !hasDiag(diags, "error", "unknown function missing__fn") {
		t.Fatalf("expected unknown-function rejection, got %v", diags)
	}
}

func TestBytesE5CrossModuleGrant(t *testing.T) {
	pub := strings.Replace(bytesPubFull(), "exports_utf8 Pub__Doc via pub__export@1\n\n", "", 1)
	client := `mod client
  provides [client__use]
  uses [pub__export@1, Pub__Doc@1]
  emits []

exports_utf8 Pub__Doc via pub__export@1

fn client__use(document: Pub__Doc) -> Bytes__Value rev 1
  emits []
  tests
    go(seal Pub__Doc("A")) => Ok(Bytes(Seq<int>[65]))
  match call pub__export(document)
    given
      go => [exchange args (document = seal Pub__Doc("A")) outcome Ok(Bytes(Seq<int>[65]))]
    on Ok r => Ok(r.value)
`
	diags := checkTwo(t, []string{"pub.can", "client.can"},
		map[string]string{"pub.can": pub, "client.can": client})
	if !hasErrCode(diags, CodeBytesExportAuthority) {
		t.Fatalf("expected CAN6010 cross-module rejection, got %v", diags)
	}
	if !hasDiag(diags, "error", "owner-local") {
		t.Fatalf("expected owner-local message, got %v", diags)
	}
}

func TestBytesE5AmbiguousBrand(t *testing.T) {
	lib := `mod lib
  provides [Dup__B]
  uses []
  emits []

brand Dup__B is str rev 1
`
	sib := strings.Replace(lib, "mod lib", "mod sib", 1)
	app := `mod app
  provides [Dup__B, app__export]
  uses []
  emits []

brand Dup__B is str rev 1

exports_utf8 Dup__B via app__export@1

fn app__export(document: Dup__B) -> Bytes__Value rev 1
  emits []
  tests
    go(seal Dup__B("A")) => Ok(Bytes(Seq<int>[65]))
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)
`
	diags := checkTwo(t, []string{"lib.can", "sib.can", "app.can"},
		map[string]string{"lib.can": lib, "sib.can": sib, "app.can": app})
	if !hasErrCode(diags, CodeBytesExportAuthority) {
		t.Fatalf("expected CAN6010 ambiguity rejection, got %v", diags)
	}
	if !hasDiag(diags, "error", "ambiguous brand") {
		t.Fatalf("expected ambiguity message, got %v", diags)
	}
}

// E6: exporters that fail the exact shape are CAN6011. Each fixture
// keeps a valid grant, so the shape rule is the single voice.
func TestBytesE6ExporterShape(t *testing.T) {
	extraParam := `mod pub
  provides [Pub__Doc, pub__export]
  uses []
  emits []

brand Pub__Doc is str rev 1

exports_utf8 Pub__Doc via pub__export@1

fn pub__export(document: Pub__Doc, extra: int) -> Bytes__Value rev 1
  emits []
  tests
    go(document = seal Pub__Doc("A"), extra = 0) => Ok(Bytes(Seq<int>[65]))
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)
`
	seqCode(t, map[string]string{"pub.can": extraParam}, "pub.can",
		CodeBytesExportShape, "exactly one parameter")
	wrongParam := `mod pub
  provides [Pub__Doc, pub__export]
  uses []
  emits []

brand Pub__Doc is str rev 1

exports_utf8 Pub__Doc via pub__export@1

fn pub__export(document: str) -> Bytes__Value rev 1
  emits []
  tests
    go(document = "A") => Ok(Bytes(Seq<int>[65]))
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)
`
	seqCode(t, map[string]string{"pub.can": wrongParam}, "pub.can",
		CodeBytesExportShape, "must have type Pub__Doc")
	base := bytesPubFull()
	fnLine := "fn pub__export(document: Pub__Doc) -> Bytes__Value rev 1"
	body := "  match call bytes__utf8__export(document)\n    on Ok r => Ok(r.value)"
	for _, c := range []struct{ name, old, new, sub string }{
		{"wrong return", fnLine,
			"fn pub__export(document: Pub__Doc) -> M__Out rev 1",
			"must return Bytes__Value"},
		{"body not match", body,
			"  Ok(Bytes(Seq<int>[65]))",
			"single call match"},
		{"rhs literal", "on Ok r => Ok(r.value)",
			"on Ok r => Ok(Bytes(Seq<int>[65]))",
			"unchanged"},
		{"named arg", "match call bytes__utf8__export(document)",
			"match call bytes__utf8__export(document = document)",
			"bare parameter"},
		{"field arg", "match call bytes__utf8__export(document)",
			"match call bytes__utf8__export(document.value)",
			"bare parameter"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fx := strings.Replace(base, c.old, c.new, 1)
			if c.name == "wrong return" {
				fx = strings.Replace(fx, "brand Pub__Doc is str rev 1",
					"type M__Out rev 1 (\n  value: Bytes\n)\n\nbrand Pub__Doc is str rev 1", 1)
				fx = strings.Replace(fx, "provides [Pub__Doc, pub__export]",
					"provides [Pub__Doc, pub__export, M__Out]", 1)
			}
			seqCode(t, map[string]string{"pub.can": fx}, "pub.can",
				CodeBytesExportShape, c.sub)
		})
	}
}

// Cases where companion rules legitimately co-fire: the shape rule is
// pinned by presence, not single-primary exactness.
func TestBytesE6CompanionRules(t *testing.T) {
	twoArms := strings.Replace(bytesPubFull(),
		"    on Ok r => Ok(r.value)",
		"    on Ok r => Ok(r.value)\n    on Ok r2 => Ok(r2.value)", 1)
	dir := writeLSPDir(t, map[string]string{"pub.can": twoArms})
	if diags := diagnose(dir, "pub.can", twoArms); !hasErrCode(diags, CodeBytesExportShape) {
		t.Fatalf("two arms: expected CAN6011, got %v", diags)
	}
	given := strings.Replace(bytesPubFull(),
		"  match call bytes__utf8__export(document)\n    on Ok r => Ok(r.value)",
		"  match call bytes__utf8__export(document)\n    given\n      empty => [exchange args (document = seal Pub__Doc(\"\")) outcome Ok(Bytes(Seq<int>[]))]\n    on Ok r => Ok(r.value)", 1)
	dir = writeLSPDir(t, map[string]string{"pub.can": given})
	diags := diagnose(dir, "pub.can", given)
	if !hasErrCode(diags, CodeBytesExportShape) {
		t.Fatalf("given table: expected CAN6011, got %v", diags)
	}
	if !hasErrCode(diags, CodeGivenOnLocal) {
		t.Fatalf("given table: expected deterministic-call rejection, got %v", diags)
	}
	decreases := strings.Replace(bytesPubFull(),
		"fn pub__export(document: Pub__Doc) -> Bytes__Value rev 1\n  emits []",
		"fn pub__export(document: Pub__Doc) -> Bytes__Value rev 1\n  decreases document\n  emits []", 1)
	dir = writeLSPDir(t, map[string]string{"pub.can": decreases})
	if diags := diagnose(dir, "pub.can", decreases); !hasErrCode(diags, CodeBytesExportShape) {
		t.Fatalf("decreases: expected CAN6011, got %v", diags)
	}
}

func TestBytesE6ContractClauses(t *testing.T) {
	emits := strings.Replace(bytesPubFull(),
		"fn pub__export(document: Pub__Doc) -> Bytes__Value rev 1\n  emits []",
		"error pub.boom(value: str)\n\nfn pub__export(document: Pub__Doc) -> Bytes__Value rev 1\n  emits [pub.boom]", 1)
	seqCode(t, map[string]string{"pub.can": emits}, "pub.can",
		CodeBytesExportShape, "emits []")
	effects := strings.Replace(bytesPubFull(),
		"fn pub__export(document: Pub__Doc) -> Bytes__Value rev 1\n  emits []",
		"state M__C: int = 0\n\nfn pub__export(document: Pub__Doc) -> Bytes__Value rev 1\n  effects [M__C.read]\n  emits []", 1)
	// Companion CAN3108 (declared-but-unused effect) co-fires; pin
	// the shape rule by presence.
	dir := writeLSPDir(t, map[string]string{"pub.can": effects})
	if diags := diagnose(dir, "pub.can", effects); !hasErrCode(diags, CodeBytesExportShape) {
		t.Fatalf("effects: expected CAN6011, got %v", diags)
	}
}

// E7: an export call with no grant anywhere is unauthorized.
func TestBytesE7UngrantedCall(t *testing.T) {
	body := `mod m
  provides [M__Doc, m__go]
  uses []
  emits []

brand M__Doc is str rev 1

fn m__go(document: M__Doc) -> Bytes__Value rev 1
  emits []
  tests
    go(seal M__Doc("A")) => Ok(Bytes(Seq<int>[65]))
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)
`
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeBytesExportAuthority, "not authorized")
}

// E8: the kernel name and the compiler-owned record are reserved.
func TestBytesE8ReservedNames(t *testing.T) {
	shadowFn := `mod m
  provides [m__go, M__Out, bytes__utf8__export]
  uses []
  emits []

type M__Out rev 1 (
  flag: bool
)

fn m__go() -> M__Out rev 1
  emits []
  tests
    go() => Ok(true)
  Ok(true)

fn bytes__utf8__export(x: int) -> M__Out rev 1
  emits []
  tests
    go(1) => Ok(true)
  Ok(true)
`
	seqCode(t, map[string]string{"m.can": shadowFn}, "m.can",
		CodePrimitiveShadow, "shadows")
	shadowEx := `mod m
  provides [m__go, M__Out, bytes__utf8__export]
  uses []
  emits []

type M__Out rev 1 (
  flag: bool
)

extern bytes__utf8__export(x: int) -> M__Out rev 1

fn m__go() -> M__Out rev 1
  emits []
  tests
    go() => Ok(true)
  Ok(true)
`
	seqCode(t, map[string]string{"m.can": shadowEx}, "m.can",
		CodePrimitiveShadow, "shadows")
	rec := strings.Replace(bytesPubFull(), `brand Pub__Doc is str rev 1`,
		"type Bytes__Value rev 1 (\n  x: int\n)\n\nbrand Pub__Doc is str rev 1", 1)
	rec = strings.Replace(rec, "provides [Pub__Doc, pub__export]",
		"provides [Pub__Doc, pub__export, Bytes__Value]", 1)
	seqCode(t, map[string]string{"pub.can": rec}, "pub.can",
		CodePrimitiveShadow, "shadows")
	brand := strings.Replace(bytesPubFull(), `brand Pub__Doc is str rev 1`,
		"brand Bytes__Value is str rev 1\n\nbrand Pub__Doc is str rev 1", 1)
	brand = strings.Replace(brand, "provides [Pub__Doc, pub__export]",
		"provides [Pub__Doc, pub__export, Bytes__Value]", 1)
	seqCode(t, map[string]string{"pub.can": brand}, "pub.can",
		CodePrimitiveShadow, "shadows")
}

// E9: two independent grants coexist in both module orders while a
// third ungranted brand stays rejected.
func TestBytesE9TwoGrants(t *testing.T) {
	a := `mod a
  provides [A__Doc, a__export]
  uses []
  emits []

brand A__Doc is str rev 1

exports_utf8 A__Doc via a__export@1

fn a__export(document: A__Doc) -> Bytes__Value rev 1
  emits []
  tests
    go(seal A__Doc("A")) => Ok(Bytes(Seq<int>[65]))
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)
`
	b := strings.Replace(strings.Replace(a, "mod a", "mod b", 1), "A__Doc", "B__Doc", -1)
	b = strings.Replace(b, "a__export", "b__export", -1)
	c := `mod c
  provides [C__Doc, c__try]
  uses []
  emits []

brand C__Doc is str rev 1

fn c__try(document: C__Doc) -> Bytes__Value rev 1
  emits []
  tests
    go(seal C__Doc("A")) => Ok(Bytes(Seq<int>[65]))
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)
`
	files := map[string]string{"a.can": a, "b.can": b, "c.can": c}
	for _, order := range [][]string{{"a.can", "b.can", "c.can"}, {"c.can", "b.can", "a.can"}} {
		diags := checkTwo(t, order, files)
		if !hasErrCode(diags, CodeBytesExportAuthority) {
			t.Fatalf("order %v: expected CAN6010 for the ungranted brand, got %v", order, diags)
		}
		if !onlyCodes(diags, CodeBytesExportAuthority, CodeTestFailed, CodeInconsistentScript) {
			t.Fatalf("order %v: unexpected companion errors, got %v", order, diags)
		}
	}
}

// E10: sink controls. The allowed typed route (export then sink)
// checks and runs; the denied route stops at authority.
func TestBytesE10SinkControls(t *testing.T) {
	allowed := `mod m
  provides [M__Doc, m__export, t__sink, T__Text, m__route]
  uses []
  emits []

brand M__Doc is str rev 1

exports_utf8 M__Doc via m__export@1

type T__Text rev 1 (
  value: str
)

extern t__sink(b: Bytes) -> T__Text rev 1

fn m__export(document: M__Doc) -> Bytes__Value rev 1
  emits []
  tests
    go(seal M__Doc("A")) => Ok(Bytes(Seq<int>[65]))
  match call bytes__utf8__export(document)
    on Ok r => Ok(r.value)

fn m__route(document: M__Doc) -> T__Text rev 1
  emits []
  tests
    go(seal M__Doc("A")) => Ok("A")
  match call m__export(document)
    on Ok e => match call t__sink(e.value)
      given
        go => [exchange args (b = Bytes(Seq<int>[65])) outcome Ok("A")]
      on Ok s => Ok(s.value)
`
	seqClean(t, map[string]string{"m.can": allowed}, "m.can")
	denied := `mod m
  provides [M__Secret, m__route, T__Text]
  uses []
  emits []

brand M__Secret is str rev 1

type T__Text rev 1 (
  value: str
)

fn m__route(secret: M__Secret) -> T__Text rev 1
  emits []
  tests
    go(secret = seal M__Secret("s")) => Ok("s")
  match call bytes__utf8__export(secret)
    on Ok e => Ok("s")
`
	seqCode(t, map[string]string{"m.can": denied}, "m.can",
		CodeBytesExportAuthority, "not authorized")
}

// E11: the exporter lowers to TextEncoder().encode, and the
// Bytes__Value definition appears exactly when referenced.
func TestBytesE11EmitPins(t *testing.T) {
	ts := compileEmit(t, bytesPubFull())
	if !strings.Contains(ts, "new TextEncoder().encode(document)") {
		t.Fatalf("emit missing TextEncoder lowering:\n%s", ts)
	}
	if strings.Contains(ts, "Bytes__Value") {
		t.Fatalf("emit leaked an unreferenced builtin type:\n%s", ts)
	}
	withField := strings.Replace(bytesPubFull(), "brand Pub__Doc is str rev 1",
		"type M__Wrap rev 1 (\n  inner: Bytes__Value\n)\n\nbrand Pub__Doc is str rev 1", 1)
	withField = strings.Replace(withField, "provides [Pub__Doc, pub__export]",
		"provides [Pub__Doc, pub__export, M__Wrap]", 1)
	ts2 := compileEmit(t, withField)
	if !strings.Contains(ts2, "export type Bytes__Value = { value: Uint8Array };") {
		t.Fatalf("emit missing builtin record definition:\n%s", ts2)
	}
}

// E12: the same-basename two-directory attack fails owner-locality in
// both input orders. The attacker never redeclares the brand.
func TestBytesE12SameBasenameAttack(t *testing.T) {
	trusted := `mod trusted
  provides [Vault__Secret]
  uses []
  emits []

brand Vault__Secret is str rev 1
`
	attacker := `mod attacker
  provides [attacker__export]
  uses [Vault__Secret@1]
  emits []

exports_utf8 Vault__Secret via attacker__export@1

fn attacker__export(secret: Vault__Secret) -> Bytes__Value rev 1
  emits []
  tests
    go(seal Vault__Secret("s")) => Ok(Bytes(Seq<int>[115]))
  match call bytes__utf8__export(secret)
    on Ok r => Ok(r.value)
`
	files := map[string]string{"trusted/common.can": trusted, "attacker/common.can": attacker}
	for _, order := range [][]string{
		{"trusted/common.can", "attacker/common.can"},
		{"attacker/common.can", "trusted/common.can"},
	} {
		diags := checkTwo(t, order, files)
		if !hasErrCode(diags, CodeBytesExportAuthority) {
			t.Fatalf("order %v: expected CAN6010, got %v", order, diags)
		}
		if !hasDiag(diags, "error", "owner-local") {
			t.Fatalf("order %v: expected owner-local message, got %v", order, diags)
		}
	}
}

// E4: malformed grants are parse errors, never partial grants.
func TestBytesE4GrantParseShapes(t *testing.T) {
	grant := "exports_utf8 Pub__Doc via pub__export@1"
	for _, c := range []struct{ name, frag string }{
		{"missing rev", "exports_utf8 Pub__Doc via pub__export"},
		{"missing via", "exports_utf8 Pub__Doc pub__export@1"},
		{"surplus", "exports_utf8 Pub__Doc via pub__export@1 extra"},
		{"negative rev", "exports_utf8 Pub__Doc via pub__export@-1"},
		{"overflow rev", "exports_utf8 Pub__Doc via pub__export@99999999999999999999999"},
		{"missing brand", "exports_utf8 via pub__export@1"},
	} {
		t.Run(c.name, func(t *testing.T) {
			body := strings.Replace(bytesPubFull(), grant, c.frag, 1)
			dir := writeLSPDir(t, map[string]string{"pub.can": body})
			diags := diagnose(dir, "pub.can", body)
			found := false
			for _, d := range diags {
				if d.Sev == "error" && d.Code == CodeParse {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected parse error, got %v", diags)
			}
		})
	}
}
