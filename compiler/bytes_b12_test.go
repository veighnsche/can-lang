package main

import (
	"strings"
	"testing"
)

// a58 B12: base64 encode kernel. bytes__base64__encode is total
// and deterministic: Bytes in, standard padded base64 in
// Encoding__Text, empty contract. G-rows are the acceptance rows.
// Padding shapes, byte order, and no-text-interpretation are the
// pinned properties.

const bytesB64Base = `mod m
  provides [m__go]
  uses []
  emits []

fn m__go(value: Bytes) -> Encoding__Text rev 1
  emits []
  tests
    empty(Bytes(Seq<int>[])) => Ok("")
    zero(Bytes(Seq<int>[0])) => Ok("AA==")
    ff(Bytes(Seq<int>[255])) => Ok("/w==")
    one(Bytes(Seq<int>[65])) => Ok("QQ==")
    two(Bytes(Seq<int>[65, 66])) => Ok("QUI=")
    three(Bytes(Seq<int>[65, 66, 67])) => Ok("QUJD")
    ordered(Bytes(Seq<int>[222, 173, 190, 239])) => Ok("3q2+7w==")
    notext(Bytes(Seq<int>[0, 65])) => Ok("AEE=")
    sweep(Bytes(Seq<int>[0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15])) => Ok("AAECAwQFBgcICQoLDA0ODw==")
  match call bytes__base64__encode(value)
    on Ok r => Ok(r.value)
`

// G0: base64 vectors, positional and named spellings. Padding
// per length mod 3, order kept, bytes never read as text.
func TestBytesG0B64Vectors(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesB64Base}, "m.can")
	named := strings.Replace(bytesB64Base,
		"match call bytes__base64__encode(value)",
		"match call bytes__base64__encode(value = value)", 1)
	// The named spelling evaluates identically but is a lint error
	// (CAN3410): exactly one finding, nothing else.
	seqCode(t, map[string]string{"m.can": named}, "m.can", CodeLintRedundant, "redundant argument name")
}

// G1: the deterministic kernel takes no given table.
func TestBytesG1NoGiven(t *testing.T) {
	body := strings.Replace(bytesB64Base,
		"  match call bytes__base64__encode(value)\n    on Ok r => Ok(r.value)",
		"  match call bytes__base64__encode(value)\n    given\n      empty => [exchange args (value = Bytes(Seq<int>[])) outcome Ok(\"\")]\n    on Ok r => Ok(r.value)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeGivenOnLocal, "no given table")
}

// G2: strict Bytes admission: str, int, and str-branded inputs
// refuse naming the wanted type.
func TestBytesG2Admission(t *testing.T) {
	mk := func(param, arg string) string {
		s := `mod m
  provides [m__go]
  uses []
  emits []

fn m__go(value: PARAM) -> Encoding__Text rev 1
  emits []
  tests
    go(ARG) => Ok("QQ==")
  match call bytes__base64__encode(value)
    on Ok r => Ok(r.value)
`
		s = strings.Replace(s, "PARAM", param, 1)
		return strings.Replace(s, "ARG", arg, 1)
	}
	seqCode(t, map[string]string{"m.can": mk("str", `"A"`)}, "m.can",
		CodeTypeMismatch, "want Bytes")
	seqCode(t, map[string]string{"m.can": mk("int", "3")}, "m.can",
		CodeTypeMismatch, "want Bytes")
	branded := strings.Replace(mk("M__Secret", `seal M__Secret("s")`),
		"fn m__go(value: M__Secret)", "brand M__Secret is str rev 1\n\nfn m__go(value: M__Secret)", 1)
	branded = strings.Replace(branded, "provides [m__go]", "provides [M__Secret, m__go]", 1)
	seqCode(t, map[string]string{"m.can": branded}, "m.can",
		CodeTypeMismatch, "want Bytes")
}

// G3: the kernel contract exists explicitly (never absent).
func TestBytesG3ContractsRegistered(t *testing.T) {
	dir := writeLSPDir(t, map[string]string{"m.can": bytesB64Base})
	mods, texts, _, err := legacyParsePaths([]string{dir + "/m.can"})
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := buildWorld(mods[0], mods, texts)
	emits, ok := prog.EmitsOf["bytes__base64__encode"]
	if !ok {
		t.Fatalf("missing EmitsOf entry for bytes__base64__encode")
	}
	if len(emits) != 0 {
		t.Fatalf("EmitsOf[b64__encode] = %v, want empty", emits)
	}
}

// G4: the encoder lowers through the base64 helper, not TextEncoder.
func TestBytesG4EmitPin(t *testing.T) {
	ts := compileEmit(t, bytesB64Base)
	if !strings.Contains(ts, "$canB64Encode(value)") {
		t.Fatalf("emit missing base64 helper lowering:\n%s", ts)
	}
	if strings.Contains(ts, "TextEncoder") {
		t.Fatalf("base64 emit must not touch TextEncoder:\n%s", ts)
	}
}

// G5: the empty contract is consulted: a stale error arm refuses.
func TestBytesG5StaleArm(t *testing.T) {
	body := strings.Replace(bytesB64Base,
		"    on Ok r => Ok(r.value)",
		"    on Ok r => Ok(r.value)\n    on m.boom e => Ok(\"\")", 1)
	body = strings.Replace(body, "fn m__go(value: Bytes)",
		"error m.boom(value: str)\n\nfn m__go(value: Bytes)", 1)
	dir := writeLSPDir(t, map[string]string{"m.can": body})
	diags := diagnose(dir, "m.can", body)
	if !hasErrCode(diags, CodeStaleArm) || !hasDiag(diags, "error", "stale match arm m.boom") {
		t.Fatalf("expected stale-arm rejection, got %v", diags)
	}
}
