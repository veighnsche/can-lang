package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a55 B10: hex decode kernel. bytes__hex__decode is the second
// fallible kernel: str in, Bytes__Value on success,
// encoding.invalid_hex (original string, unchanged) on malformed
// input. X-rows are the acceptance rows (verdict v2: str payload,
// single kind, generalized lowering, parity-masking controls,
// CAN3110 prefix-attack linkage, mixed probe module).

const bytesHexDecodeBase = `mod m
  provides [m__go]
  uses []
  emits [encoding.invalid_hex]

fn m__go(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_hex]
  tests
    empty("") => Ok(Bytes(Seq<int>[]))
    hex00("00") => Ok(Bytes(Seq<int>[0]))
    lower("ff") => Ok(Bytes(Seq<int>[255]))
    upper("FF") => Ok(Bytes(Seq<int>[255]))
    mixed("aF") => Ok(Bytes(Seq<int>[175]))
    deadbeef("deadbeef") => Ok(Bytes(Seq<int>[222, 173, 190, 239]))
    upperlong("DEADBEEF") => Ok(Bytes(Seq<int>[222, 173, 190, 239]))
    long("0123456789abcdef") => Ok(Bytes(Seq<int>[1, 35, 69, 103, 137, 171, 205, 239]))
    eda080("eda080") => Ok(Bytes(Seq<int>[237, 160, 128]))
NULROW
NONASCIIROWS
    odd_f("f") => encoding.invalid_hex(value = "f")
    odd_abc("abc") => encoding.invalid_hex(value = "abc")
    odd_long("0123456789abc") => encoding.invalid_hex(value = "0123456789abc")
    prefix_0x("0x41") => encoding.invalid_hex(value = "0x41")
    prefix_0X("0X41") => encoding.invalid_hex(value = "0X41")
    sp_lead(" 142") => encoding.invalid_hex(value = " 142")
    sp_trail("414 ") => encoding.invalid_hex(value = "414 ")
    sp_mid("41 2") => encoding.invalid_hex(value = "41 2")
    slash_second("4/") => encoding.invalid_hex(value = "4/")
    slash_first("/4") => encoding.invalid_hex(value = "/4")
    colon("4:") => encoding.invalid_hex(value = "4:")
    at("4@") => encoding.invalid_hex(value = "4@")
    bigG("4G") => encoding.invalid_hex(value = "4G")
    backtick("4` + "`" + `") => encoding.invalid_hex(value = "4` + "`" + `")
    lowg("4g") => encoding.invalid_hex(value = "4g")
    fidelity("aFzz") => encoding.invalid_hex(value = "aFzz")
    prefix_attack("41zz42") => encoding.invalid_hex(value = "41zz42")
    prefix_trunc("00ffa") => encoding.invalid_hex(value = "00ffa")
    trunc_a("a") => encoding.invalid_hex(value = "a")
  match call bytes__hex__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_hex e => forward e
`

func bytesHexDecodeFull() string {
	nul := "    nul_mid(\"a\x00b\") => encoding.invalid_hex(value = \"a\x00b\")\n"
	nonascii := "    e_acute(\"é0\") => encoding.invalid_hex(value = \"é0\")\n" +
		"    cjk(\"0中\") => encoding.invalid_hex(value = \"0中\")\n" +
		"    astral(\"😀\") => encoding.invalid_hex(value = \"😀\")\n" +
		"    bom(\"0\uFEFF\") => encoding.invalid_hex(value = \"0\uFEFF\")\n"
	out := strings.Replace(bytesHexDecodeBase, "NULROW\n", nul, 1)
	return strings.Replace(out, "NONASCIIROWS\n", nonascii, 1)
}

// X0: hex decode vectors, both outcomes, positional and named.
// Parity-masking controls (even-UTF-16 non-ASCII, neighbor
// nibbles) and the fidelity row pin the exact rejection set.
func TestBytesX0HexDecodeVectors(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesHexDecodeFull()}, "m.can")
	named := strings.Replace(bytesHexDecodeFull(),
		"match call bytes__hex__decode(value)",
		"match call bytes__hex__decode(value = value)", 1)
	// The named spelling evaluates identically but is a lint error
	// (CAN3410): exactly one finding, nothing else.
	seqCode(t, map[string]string{"m.can": named}, "m.can", CodeLintRedundant, "redundant argument name")
}

// X1: both arms are mandatory.
func TestBytesX1MissingArms(t *testing.T) {
	noErr := strings.Replace(bytesHexDecodeFull(),
		"\n    on encoding.invalid_hex e => forward e", "", 1)
	dir := writeLSPDir(t, map[string]string{"m.can": noErr})
	diags := diagnose(dir, "m.can", noErr)
	if !hasErrCode(diags, CodeMissingArm) || !hasDiag(diags, "error", "non-exhaustive match, missing") {
		t.Fatalf("expected missing-arm rejection without error arm, got %v", diags)
	}
	noOk := strings.Replace(bytesHexDecodeFull(),
		"    on Ok r => Ok(r.value)\n", "", 1)
	dir = writeLSPDir(t, map[string]string{"m.can": noOk})
	diags = diagnose(dir, "m.can", noOk)
	if !hasErrCode(diags, CodeMissingArm) || !hasDiag(diags, "error", "non-exhaustive match, missing") {
		t.Fatalf("expected missing-arm rejection without Ok arm, got %v", diags)
	}
}

// X2: a stale arm naming a declared unrelated error refuses.
func TestBytesX2StaleArm(t *testing.T) {
	body := strings.Replace(bytesHexDecodeFull(),
		"    on Ok r => Ok(r.value)",
		"    on Ok r => Ok(r.value)\n    on m.boom e2 => Ok(Bytes(Seq<int>[]))", 1)
	body = strings.Replace(body, "fn m__go(value: str)",
		"error m.boom(value: str)\n\nfn m__go(value: str)", 1)
	dir := writeLSPDir(t, map[string]string{"m.can": body})
	diags := diagnose(dir, "m.can", body)
	if !hasErrCode(diags, CodeStaleArm) || !hasDiag(diags, "error", "stale match arm m.boom") {
		t.Fatalf("expected stale-arm rejection, got %v", diags)
	}
}

// X3: the deterministic kernel takes no given table.
func TestBytesX3NoGiven(t *testing.T) {
	body := strings.Replace(bytesHexDecodeFull(),
		"  match call bytes__hex__decode(value)\n    on Ok r => Ok(r.value)",
		"  match call bytes__hex__decode(value)\n    given\n      empty => [exchange args (value = \"\") outcome Ok(Bytes(Seq<int>[]))]\n    on Ok r => Ok(r.value)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeGivenOnLocal, "no given table")
}

// X4: strict str admission, through BOTH arms: a malformed
// branded input must fail at the argument (want str), never
// leak its representation through the error payload.
func TestBytesX4Admission(t *testing.T) {
	mk := func(param, arg string) string {
		s := `mod m
  provides [m__go]
  uses []
  emits [encoding.invalid_hex]

fn m__go(value: PARAM) -> Bytes__Value rev 1
  emits [encoding.invalid_hex]
  tests
    go(ARG) => Ok(Bytes(Seq<int>[65]))
  match call bytes__hex__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_hex e => encoding.invalid_hex(value = e.value)
`
		s = strings.Replace(s, "PARAM", param, 1)
		return strings.Replace(s, "ARG", arg, 1)
	}
	seqCode(t, map[string]string{"m.can": mk("Bytes", "Bytes(Seq<int>[65])")}, "m.can",
		CodeTypeMismatch, "want str")
	seqCode(t, map[string]string{"m.can": mk("int", "3")}, "m.can",
		CodeTypeMismatch, "want str")
	// Brands erase at runtime, so brand rows PASS execution (unlike
	// str/int, which fail and skip coverage): each brand fixture
	// needs a second row taking the error arm.
	goodBrand := strings.Replace(mk("M__Secret", `seal M__Secret("41")`),
		"fn m__go(value: M__Secret)", "brand M__Secret is str rev 1\n\nfn m__go(value: M__Secret)", 1)
	goodBrand = strings.Replace(goodBrand, "provides [m__go]", "provides [M__Secret, m__go]", 1)
	goodBrand = strings.Replace(goodBrand,
		`    go(seal M__Secret("41")) => Ok(Bytes(Seq<int>[65]))`,
		"    go(seal M__Secret(\"41\")) => Ok(Bytes(Seq<int>[65]))\n    bad(value = seal M__Secret(\"zz\")) => encoding.invalid_hex(value = \"zz\")", 1)
	seqCode(t, map[string]string{"m.can": goodBrand}, "m.can",
		CodeTypeMismatch, "want str")
	badBrand := strings.Replace(mk("M__Secret", `seal M__Secret("zz")`),
		"fn m__go(value: M__Secret)", "brand M__Secret is str rev 1\n\nfn m__go(value: M__Secret)", 1)
	badBrand = strings.Replace(badBrand, "provides [m__go]", "provides [M__Secret, m__go]", 1)
	badBrand = strings.Replace(badBrand,
		`    go(seal M__Secret("zz")) => Ok(Bytes(Seq<int>[65]))`,
		"    go(seal M__Secret(\"zz\")) => Ok(Bytes(Seq<int>[65]))\n    bad(value = seal M__Secret(\"41zz42\")) => encoding.invalid_hex(value = \"41zz42\")", 1)
	seqCode(t, map[string]string{"m.can": badBrand}, "m.can",
		CodeTypeMismatch, "want str")
}

// X5: the decode contract exists explicitly: EmitsOf entry plus
// the compiler-owned error registration (fields ["value"]).
func TestBytesX5ContractsRegistered(t *testing.T) {
	dir := writeLSPDir(t, map[string]string{"m.can": bytesHexDecodeFull()})
	mods, texts, _, err := legacyParsePaths([]string{dir + "/m.can"})
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := buildWorld(mods[0], mods, texts)
	emits, ok := prog.EmitsOf["bytes__hex__decode"]
	if !ok {
		t.Fatalf("missing EmitsOf entry for bytes__hex__decode")
	}
	if len(emits) != 1 || emits[0] != "encoding.invalid_hex" {
		t.Fatalf("EmitsOf[hex__decode] = %v, want [encoding.invalid_hex]", emits)
	}
	fs, ok := prog.Errors["encoding.invalid_hex"]
	if !ok || len(fs) != 1 || fs[0] != "value" {
		t.Fatalf("prog.Errors[invalid_hex] = %v, want [value]", fs)
	}
}

// X6: the hex decoder lowers through its own helper and union;
// a hex-only fixture touches neither TextEncoder nor TextDecoder.
func TestBytesX6EmitPins(t *testing.T) {
	ts := compileEmit(t, bytesHexDecodeFull())
	for _, want := range []string{
		"$canHexDecode(",
		`{ $can_kind: "ok"; value: Uint8Array } | { $can_kind: "encoding.invalid_hex"; value: string }`,
	} {
		if !strings.Contains(ts, want) {
			t.Fatalf("emit missing %q:\n%s", want, ts)
		}
	}
	for _, banned := range []string{"TextEncoder", "TextDecoder", "$canUtf8Decode("} {
		if strings.Contains(ts, banned) {
			t.Fatalf("hex-only emit must not contain %q:\n%s", banned, ts)
		}
	}
}

// X7: source cannot redeclare the compiler-owned error, even
// identically.
func TestBytesX7ShadowRejection(t *testing.T) {
	body := strings.Replace(bytesHexDecodeFull(), "fn m__go(value: str)",
		"error encoding.invalid_hex(value: str)\n\nfn m__go(value: str)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodePrimitiveShadow, "shadows a compiler-owned error")
}

// X8: the builtin error path is typed end-to-end: a Bytes payload
// construction refuses, and a wrong-payload expectation fails.
func TestBytesX8ErrorPathTyped(t *testing.T) {
	wrongCtor := strings.Replace(bytesHexDecodeFull(),
		"on encoding.invalid_hex e => forward e",
		`on encoding.invalid_hex e => encoding.invalid_hex(value = Bytes(Seq<int>[65]))`, 1)
	seqCode(t, map[string]string{"m.can": wrongCtor}, "m.can",
		CodeTypeMismatch, "want str")
	wrongPay := strings.Replace(bytesHexDecodeFull(),
		`fidelity("aFzz") => encoding.invalid_hex(value = "aFzz")`,
		`fidelity("aFzz") => encoding.invalid_hex(value = "aFZZ")`, 1)
	seqCode(t, map[string]string{"m.can": wrongPay}, "m.can",
		CodeTestFailed, "fidelity")
}

const bytesHexDecodeProv = `mod prov
  provides [prov__go]
  uses []
  emits [encoding.invalid_hex]

fn prov__go(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_hex]
  tests
    good("41") => Ok(Bytes(Seq<int>[65]))
    bad("41zz42") => encoding.invalid_hex(value = "41zz42")
  match call bytes__hex__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_hex e => encoding.invalid_hex(value = e.value)
`

const bytesHexDecodeLie = `mod client
  provides [client__use]
  uses [prov__go@1]
  emits [encoding.invalid_hex]

fn client__use(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_hex]
  tests
    prefixlie("41zz42") => Ok(Bytes(Seq<int>[65]))
    suffixlie("ffzz") => Ok(Bytes(Seq<int>[255]))
  match call prov__go(value)
    given
      prefixlie => [exchange args (value = "41zz42") outcome Ok(Bytes(Seq<int>[65]))]
      suffixlie => [exchange args (value = "ffzz") outcome Ok(Bytes(Seq<int>[255]))]
    on Ok r => Ok(r.value)
    on encoding.invalid_hex e => encoding.invalid_hex(value = e.value)
`

// X9: partial results never escape as success: the prefix-attack
// lie ("41zz42" scripted as Ok([65])) and the suffix lie ("ffzz"
// as Ok([255])) both contradict (CAN3110) under both orders.
func TestBytesX9ContradictionBothOrders(t *testing.T) {
	files := map[string]string{"prov.can": bytesHexDecodeProv, "client.can": bytesHexDecodeLie}
	for _, order := range [][]string{{"prov.can", "client.can"}, {"client.can", "prov.can"}} {
		diags := checkTwo(t, order, files)
		if !hasErrCode(diags, CodeInconsistentScript) {
			t.Fatalf("order %v: expected CAN3110, got %v", order, diags)
		}
		if !hasDiag(diags, "error", "contradicts prov__go") {
			t.Fatalf("order %v: expected contradiction message, got %v", order, diags)
		}
	}
}

// X10: the catalog attributes the intrinsic's declared failure
// to the compiler kernel, with the owned field list.
func TestBytesX10CatalogAttribution(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "m.can")
	if err := os.WriteFile(srcPath, []byte(bytesHexDecodeProv), 0o644); err != nil {
		t.Fatal(err)
	}
	out := t.TempDir()
	if err := compileEx(out, []string{srcPath}, true); err != nil {
		t.Fatalf("decode fixture must compile: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(out, "errors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var entries []catalogEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatal(err)
	}
	byKind := map[string]catalogEntry{}
	for _, e := range entries {
		byKind[e.Kind] = e
	}
	got, ok := byKind["encoding.invalid_hex"]
	if !ok {
		t.Fatalf("catalog omits encoding.invalid_hex, got %v", entries)
	}
	if len(got.Fields) != 1 || got.Fields[0] != "value" {
		t.Fatalf("catalog fields = %v, want [value]", got.Fields)
	}
	found := false
	for _, r := range got.RaisedBy {
		if r == "kernel.bytes__hex__decode" {
			found = true
		}
	}
	if !found {
		t.Fatalf("catalog raised_by lacks kernel attribution, got %v", got.RaisedBy)
	}
}

// X11: the mixed probe chains hex decode into UTF-8 decode
// (verdict's exact module, test-only — never stdlib). Every arm
// has a witness: "ff" must survive hex and fail UTF-8 after;
// "41zz42" must fail hex without forwarding its prefix.
const bytesHexMixedProbe = `mod probe
  provides [probe__decode_text]
  uses []
  emits [encoding.invalid_hex, encoding.invalid_utf8]

fn probe__decode_text(value: str) -> Encoding__Text rev 1
  emits [encoding.invalid_hex, encoding.invalid_utf8]
  tests
    ascii("41") => Ok("A")
    invalid_text("ff") => encoding.invalid_utf8(value = Bytes(Seq<int>[255]))
    invalid_hex("41zz42") => encoding.invalid_hex(value = "41zz42")
    incomplete_pair("00ffa") => encoding.invalid_hex(value = "00ffa")
  match call bytes__hex__decode(value)
    on Ok b => match call bytes__utf8__decode(b.value)
      on Ok t => Ok(t.value)
      on encoding.invalid_utf8 e => forward e
    on encoding.invalid_hex e => forward e
`

// X11a: the mixed probe is clean: every arm witnessed, both
// kernels' contracts live.
func TestBytesX11aMixedProbeClean(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesHexMixedProbe}, "m.can")
}

// X11b: per-call unions and helpers are exact in a both-decoders
// module: each call site carries its own kernel's union and
// helper, never a merged payload type.
func TestBytesX11bMixedEmitPins(t *testing.T) {
	ts := compileEmit(t, bytesHexMixedProbe)
	for _, want := range []string{
		"$canHexDecode(",
		"$canUtf8Decode(",
		`{ $can_kind: "ok"; value: Uint8Array } | { $can_kind: "encoding.invalid_hex"; value: string }`,
		`{ $can_kind: "ok"; value: string } | { $can_kind: "encoding.invalid_utf8"; value: Uint8Array }`,
	} {
		if !strings.Contains(ts, want) {
			t.Fatalf("mixed emit missing %q:\n%s", want, ts)
		}
	}
	if strings.Contains(ts, "value: string | Uint8Array") {
		t.Fatalf("mixed emit must not merge payload types:\n%s", ts)
	}
}

const bytesHexMixedLie = `mod client
  provides [client__use]
  uses [probe__decode_text@1]
  emits [encoding.invalid_hex, encoding.invalid_utf8]

fn client__use() -> Encoding__Text rev 1
  emits [encoding.invalid_hex, encoding.invalid_utf8]
  tests
    lie() => Ok("A")
  match call probe__decode_text("41zz42")
    given
      lie => [exchange args (value = "41zz42") outcome Ok("A")]
    on Ok r => Ok(r.value)
    on encoding.invalid_hex e => encoding.invalid_hex(value = e.value)
    on encoding.invalid_utf8 e2 => encoding.invalid_utf8(value = e2.value)
`

// X11c: the mixed probe's false script contradicts (CAN3110)
// under both module orders.
func TestBytesX11cMixedContradiction(t *testing.T) {
	files := map[string]string{"probe.can": bytesHexMixedProbe, "client.can": bytesHexMixedLie}
	for _, order := range [][]string{{"probe.can", "client.can"}, {"client.can", "probe.can"}} {
		diags := checkTwo(t, order, files)
		if !hasErrCode(diags, CodeInconsistentScript) {
			t.Fatalf("order %v: expected CAN3110, got %v", order, diags)
		}
		if !hasDiag(diags, "error", "contradicts probe__decode_text") {
			t.Fatalf("order %v: expected contradiction message, got %v", order, diags)
		}
	}
}
