package main

import (
	"strings"
	"testing"
)

// a47 S3 (B3): generic UTF-8 encode kernel. bytes__utf8__encode is a
// public, total, deterministic kernel: strict str admission through
// the ordinary checker, explicit empty contract, TextEncoder
// lowering. N-rows are the scope-doc acceptance rows.

const bytesEncodeBase = `mod m
  provides [m__go, M__Out]
  uses []
  emits []

type M__Out rev 1 (
  vals: Bytes
)

fn m__go(text: str) -> M__Out rev 1
  emits []
  tests
    empty("") => Ok(Bytes(Seq<int>[]))
    ascii("A") => Ok(Bytes(Seq<int>[65]))
    latin("é") => Ok(Bytes(Seq<int>[195, 169]))
    astral("😀") => Ok(Bytes(Seq<int>[240, 159, 152, 128]))
NULROW
BOMROW
  match call bytes__utf8__encode(text)
    on Ok r => Ok(r.value)
`

func bytesEncodeFull() string {
	nulRow := "    nul(\"a\x00b\") => Ok(Bytes(Seq<int>[97, 0, 98]))\n"
	bomRow := "    bom(\"\uFEFFA\") => Ok(Bytes(Seq<int>[239, 187, 191, 65]))\n"
	out := strings.Replace(bytesEncodeBase, "NULROW\n", nulRow, 1)
	return strings.Replace(out, "BOMROW\n", bomRow, 1)
}

// N0: encode vectors, positional and named spellings. Total over CAN
// scalars; NUL and BOM preserved; no normalization.
func TestBytesN0EncodeVectors(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesEncodeFull()}, "m.can")
	named := strings.Replace(bytesEncodeFull(),
		"match call bytes__utf8__encode(text)",
		"match call bytes__utf8__encode(value = text)", 1)
	// The named spelling evaluates identically but is a lint error
	// (CAN3410): exactly one finding, nothing else.
	seqCode(t, map[string]string{"m.can": named}, "m.can", CodeLintRedundant, "redundant argument name")
}

// N1: brands are not strings at the encoder edge. The nominal
// boundary is static by design (brands erase at runtime): this row
// pins the CAN6003, and the runtime passing is not the evidence.
func TestBytesN1BrandRejected(t *testing.T) {
	body := `mod m
  provides [M__Secret, m__go, M__Out]
  uses []
  emits []

brand M__Secret is str rev 1

type M__Out rev 1 (
  vals: Bytes
)

fn m__go(secret: M__Secret) -> M__Out rev 1
  emits []
  tests
    go(seal M__Secret("s")) => Ok(Bytes(Seq<int>[115]))
  match call bytes__utf8__encode(secret)
    on Ok r => Ok(r.value)
`
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeTypeMismatch, "want str")
}

// N2: non-str arguments refuse with the wanted type named.
func TestBytesN2NonStrRejected(t *testing.T) {
	clean := `mod m
  provides [m__go, M__Out]
  uses []
  emits []

type M__Out rev 1 (
  vals: Bytes
)

fn m__go(x: int) -> M__Out rev 1
  emits []
  tests
    go(3) => Ok(Bytes(Seq<int>[51]))
  match call bytes__utf8__encode(x)
    on Ok r => Ok(r.value)
`
	seqCode(t, map[string]string{"m.can": clean}, "m.can",
		CodeTypeMismatch, "want str")
}

// N3: arity failures fail binding, naming the missing/extra shape.
func TestBytesN3Arity(t *testing.T) {
	none := strings.Replace(bytesEncodeFull(),
		"match call bytes__utf8__encode(text)",
		"match call bytes__utf8__encode()", 1)
	seqCode(t, map[string]string{"m.can": none}, "m.can",
		CodeBadBinding, "missing arg value")
	two := strings.Replace(bytesEncodeFull(),
		"match call bytes__utf8__encode(text)",
		"match call bytes__utf8__encode(text, text)", 1)
	seqCode(t, map[string]string{"m.can": two}, "m.can",
		CodeBadBinding, "takes 2 args for 1 params")
}

// N4: the deterministic kernel takes no given table.
func TestBytesN4NoGiven(t *testing.T) {
	body := strings.Replace(bytesEncodeFull(),
		"  match call bytes__utf8__encode(text)\n    on Ok r => Ok(r.value)",
		"  match call bytes__utf8__encode(text)\n    given\n      empty => [exchange args (text = \"\") outcome Ok(Bytes(Seq<int>[]))]\n    on Ok r => Ok(r.value)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeGivenOnLocal, "no given table")
}

// N5: both kernel contracts exist explicitly (never absent entries).
func TestBytesN5ContractsRegistered(t *testing.T) {
	dir := writeLSPDir(t, map[string]string{"m.can": bytesEncodeFull()})
	mods, texts, _, err := legacyParsePaths([]string{dir + "/m.can"})
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := buildWorld(mods[0], mods, texts)
	for _, k := range []string{"bytes__utf8__export", "bytes__utf8__encode"} {
		emits, ok := prog.EmitsOf[k]
		if !ok {
			t.Fatalf("missing EmitsOf entry for %s", k)
		}
		if len(emits) != 0 {
			t.Fatalf("EmitsOf[%s] = %v, want empty", k, emits)
		}
	}
}

// N6: the encoder lowers through TextEncoder like the exporter.
func TestBytesN6EmitPin(t *testing.T) {
	ts := compileEmit(t, bytesEncodeFull())
	if !strings.Contains(ts, "new TextEncoder().encode(text)") {
		t.Fatalf("emit missing TextEncoder lowering:\n%s", ts)
	}
}

// N7: the empty contract is consulted: a stale error arm refuses.
func TestBytesN7StaleArm(t *testing.T) {
	body := strings.Replace(bytesEncodeFull(),
		"    on Ok r => Ok(r.value)",
		"    on Ok r => Ok(r.value)\n    on m.boom e => Ok(Bytes(Seq<int>[]))", 1)
	body = strings.Replace(body, "type M__Out rev 1 (",
		"error m.boom(value: str)\n\ntype M__Out rev 1 (", 1)
	// A stale arm is untaken by definition, so coverage co-fires;
	// pin the stale-arm rule by presence.
	dir := writeLSPDir(t, map[string]string{"m.can": body})
	diags := diagnose(dir, "m.can", body)
	if !hasDiag(diags, "error", "stale match arm m.boom") {
		t.Fatalf("expected stale-arm rejection, got %v", diags)
	}
}
