package main

import (
	"strings"
	"testing"
)

// a52 B8: hex encode kernel. bytes__hex__encode is total and
// deterministic: Bytes in, lowercase hex in Encoding__Text, empty
// contract. H-rows are the acceptance rows. Lowercase, byte order,
// and no-text-interpretation are the pinned properties.

const bytesHexBase = `mod m
  provides [m__go]
  uses []
  emits []

fn m__go(value: Bytes) -> Encoding__Text rev 1
  emits []
  tests
    empty(Bytes(Seq<int>[])) => Ok("")
    zero(Bytes(Seq<int>[0])) => Ok("00")
    ff(Bytes(Seq<int>[255])) => Ok("ff")
    lower(Bytes(Seq<int>[171])) => Ok("ab")
    leadzero(Bytes(Seq<int>[1])) => Ok("01")
    sixteen(Bytes(Seq<int>[16])) => Ok("10")
    ordered(Bytes(Seq<int>[222, 173, 190, 239])) => Ok("deadbeef")
    notext(Bytes(Seq<int>[65, 66])) => Ok("4142")
    nulbyte(Bytes(Seq<int>[0, 65])) => Ok("0041")
    high(Bytes(Seq<int>[128, 200])) => Ok("80c8")
    nibbles(Bytes(Seq<int>[0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15])) => Ok("000102030405060708090a0b0c0d0e0f")
  match call bytes__hex__encode(value)
    on Ok r => Ok(r.value)
`

// H0: hex vectors, positional and named spellings. Lowercase by
// row (ab, deadbeef), order kept, ASCII-looking bytes never pass
// through as text, leading zeros kept.
func TestBytesH0HexVectors(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesHexBase}, "m.can")
	named := strings.Replace(bytesHexBase,
		"match call bytes__hex__encode(value)",
		"match call bytes__hex__encode(value = value)", 1)
	// The named spelling evaluates identically but is a lint error
	// (CAN3410): exactly one finding, nothing else.
	seqCode(t, map[string]string{"m.can": named}, "m.can", CodeLintRedundant, "redundant argument name")
}

// H1: the deterministic kernel takes no given table.
func TestBytesH1NoGiven(t *testing.T) {
	body := strings.Replace(bytesHexBase,
		"  match call bytes__hex__encode(value)\n    on Ok r => Ok(r.value)",
		"  match call bytes__hex__encode(value)\n    given\n      empty => [exchange args (value = Bytes(Seq<int>[])) outcome Ok(\"\")]\n    on Ok r => Ok(r.value)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeGivenOnLocal, "no given table")
}

// H2: strict Bytes admission: str, int, and str-branded inputs
// refuse naming the wanted type.
func TestBytesH2Admission(t *testing.T) {
	mk := func(param, arg string) string {
		s := `mod m
  provides [m__go]
  uses []
  emits []

fn m__go(value: PARAM) -> Encoding__Text rev 1
  emits []
  tests
    go(ARG) => Ok("41")
  match call bytes__hex__encode(value)
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

// H3: the kernel contract exists explicitly (never absent).
func TestBytesH3ContractsRegistered(t *testing.T) {
	dir := writeLSPDir(t, map[string]string{"m.can": bytesHexBase})
	mods, texts, _, err := legacyParsePaths([]string{dir + "/m.can"})
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := buildWorld(mods[0], mods, texts)
	emits, ok := prog.EmitsOf["bytes__hex__encode"]
	if !ok {
		t.Fatalf("missing EmitsOf entry for bytes__hex__encode")
	}
	if len(emits) != 0 {
		t.Fatalf("EmitsOf[hex__encode] = %v, want empty", emits)
	}
}

// H4: the encoder lowers through the hex helper, not TextEncoder.
func TestBytesH4EmitPin(t *testing.T) {
	ts := compileEmit(t, bytesHexBase)
	if !strings.Contains(ts, "$canHexEncode(value)") {
		t.Fatalf("emit missing hex helper lowering:\n%s", ts)
	}
	if strings.Contains(ts, "TextEncoder") {
		t.Fatalf("hex emit must not touch TextEncoder:\n%s", ts)
	}
}

// H5: the empty contract is consulted: a stale error arm refuses.
func TestBytesH5StaleArm(t *testing.T) {
	body := strings.Replace(bytesHexBase,
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
