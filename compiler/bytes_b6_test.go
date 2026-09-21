package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a50 B6: UTF-8 decode kernel. bytes__utf8__decode is the first
// fallible kernel: Bytes in, Encoding__Text on success,
// encoding.invalid_utf8 (original payload, unchanged) on malformed
// input. Both contracts are compiler-owned; fixtures must compile
// without declaring their own copies. D-rows are the acceptance
// rows (verdict v2: ignoreBOM-corrected TS, builtin error path,
// CAN3110 linkage).

const bytesDecodeBase = `mod m
  provides [m__go]
  uses []
  emits [encoding.invalid_utf8]

fn m__go(value: Bytes) -> Encoding__Text rev 1
  emits [encoding.invalid_utf8]
  tests
    empty(Bytes(Seq<int>[])) => Ok("")
    ascii(Bytes(Seq<int>[65])) => Ok("A")
    latin(Bytes(Seq<int>[195, 169])) => Ok("é")
    cjk(Bytes(Seq<int>[228, 184, 150])) => Ok("世")
    astral(Bytes(Seq<int>[240, 159, 152, 128])) => Ok("😀")
    ufffd(Bytes(Seq<int>[239, 191, 189])) => Ok("�")
    mark(Bytes(Seq<int>[38, 60, 62])) => Ok("&<>")
NULOUTROWS
BOMOUTROWS
BOUNDROWS
    overlong_nul(Bytes(Seq<int>[192, 128])) => encoding.invalid_utf8(value = Bytes(Seq<int>[192, 128]))
    overlong_3(Bytes(Seq<int>[224, 128, 128])) => encoding.invalid_utf8(value = Bytes(Seq<int>[224, 128, 128]))
    surrogate(Bytes(Seq<int>[237, 160, 128])) => encoding.invalid_utf8(value = Bytes(Seq<int>[237, 160, 128]))
    above_max(Bytes(Seq<int>[244, 144, 128, 128])) => encoding.invalid_utf8(value = Bytes(Seq<int>[244, 144, 128, 128]))
    stray_cont(Bytes(Seq<int>[128])) => encoding.invalid_utf8(value = Bytes(Seq<int>[128]))
    lead_f5(Bytes(Seq<int>[245])) => encoding.invalid_utf8(value = Bytes(Seq<int>[245]))
    lead_f6(Bytes(Seq<int>[246])) => encoding.invalid_utf8(value = Bytes(Seq<int>[246]))
    lead_f7(Bytes(Seq<int>[247])) => encoding.invalid_utf8(value = Bytes(Seq<int>[247]))
    lead_f8(Bytes(Seq<int>[248])) => encoding.invalid_utf8(value = Bytes(Seq<int>[248]))
    lead_ff(Bytes(Seq<int>[255])) => encoding.invalid_utf8(value = Bytes(Seq<int>[255]))
    trunc_e9a(Bytes(Seq<int>[195])) => encoding.invalid_utf8(value = Bytes(Seq<int>[195]))
    trunc_e2a(Bytes(Seq<int>[226])) => encoding.invalid_utf8(value = Bytes(Seq<int>[226]))
    trunc_e2b(Bytes(Seq<int>[226, 130])) => encoding.invalid_utf8(value = Bytes(Seq<int>[226, 130]))
    trunc_f0a(Bytes(Seq<int>[240])) => encoding.invalid_utf8(value = Bytes(Seq<int>[240]))
    trunc_f0b(Bytes(Seq<int>[240, 159])) => encoding.invalid_utf8(value = Bytes(Seq<int>[240, 159]))
    trunc_f0c(Bytes(Seq<int>[240, 159, 152])) => encoding.invalid_utf8(value = Bytes(Seq<int>[240, 159, 152]))
    trunc_after_ascii(Bytes(Seq<int>[65, 226, 130])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 226, 130]))
    trunc_after_a_e9a(Bytes(Seq<int>[65, 195])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 195]))
    trunc_after_a_e2a(Bytes(Seq<int>[65, 226])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 226]))
    trunc_after_a_f0a(Bytes(Seq<int>[65, 240])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 240]))
    trunc_after_a_f0b(Bytes(Seq<int>[65, 240, 159])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 240, 159]))
    trunc_after_a_f0c(Bytes(Seq<int>[65, 240, 159, 152])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 240, 159, 152]))
    bom_then_bad(Bytes(Seq<int>[239, 187, 191, 255])) => encoding.invalid_utf8(value = Bytes(Seq<int>[239, 187, 191, 255]))
    mid_bad(Bytes(Seq<int>[65, 255, 66])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 255, 66]))
  match call bytes__utf8__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_utf8 e => forward e
`

func bytesDecodeFull() string {
	nul := "    nul_first(Bytes(Seq<int>[0, 104, 105])) => Ok(\"\x00hi\")\n" +
		"    nul_middle(Bytes(Seq<int>[97, 0, 98])) => Ok(\"a\x00b\")\n" +
		"    nul_last(Bytes(Seq<int>[97, 98, 0])) => Ok(\"ab\x00\")\n"
	bom := "    bom_only(Bytes(Seq<int>[239, 187, 191])) => Ok(\"\uFEFF\")\n" +
		"    bom_text(Bytes(Seq<int>[239, 187, 191, 65])) => Ok(\"\uFEFFA\")\n" +
		"    bom_bom(Bytes(Seq<int>[239, 187, 191, 239, 187, 191])) => Ok(\"\uFEFF\uFEFF\")\n" +
		"    bom_interior(Bytes(Seq<int>[65, 239, 187, 191, 66])) => Ok(\"A\uFEFFB\")\n"
	bound := "    u007f(Bytes(Seq<int>[127])) => Ok(\"\u007f\")\n" +
		"    u0080(Bytes(Seq<int>[194, 128])) => Ok(\"\u0080\")\n" +
		"    u07ff(Bytes(Seq<int>[223, 191])) => Ok(\"\u07ff\")\n" +
		"    u0800(Bytes(Seq<int>[224, 160, 128])) => Ok(\"\u0800\")\n" +
		"    ud7ff(Bytes(Seq<int>[237, 159, 191])) => Ok(\"\ud7ff\")\n" +
		"    ue000(Bytes(Seq<int>[238, 128, 128])) => Ok(\"\ue000\")\n" +
		"    uffff(Bytes(Seq<int>[239, 191, 191])) => Ok(\"\uffff\")\n" +
		"    u10000(Bytes(Seq<int>[240, 144, 128, 128])) => Ok(\"\U00010000\")\n" +
		"    u10ffff(Bytes(Seq<int>[244, 143, 191, 191])) => Ok(\"\U0010ffff\")\n"
	out := strings.Replace(bytesDecodeBase, "NULOUTROWS\n", nul, 1)
	out = strings.Replace(out, "BOMOUTROWS\n", bom, 1)
	return strings.Replace(out, "BOUNDROWS\n", bound, 1)
}

// D0: decode vectors, both outcomes. Valid incl. boundaries,
// surrogate gap, U+FFFD, noncharacter, BOM variants, NUL
// positions, empty; invalid per class incl. truncation at each
// length alone and after valid text. No fixture-local contract
// copies: this compiles on compiler-owned declarations alone.
func TestBytesD0DecodeVectors(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesDecodeFull()}, "m.can")
	named := strings.Replace(bytesDecodeFull(),
		"match call bytes__utf8__decode(value)",
		"match call bytes__utf8__decode(value = value)", 1)
	// The named spelling evaluates identically but is a lint error
	// (CAN3410): exactly one finding, nothing else.
	seqCode(t, map[string]string{"m.can": named}, "m.can", CodeLintRedundant, "redundant argument name")
}

// D1: both arms are mandatory: a missing error arm and a missing
// Ok arm each refuse with the missing-arm rule.
func TestBytesD1MissingArms(t *testing.T) {
	noErr := strings.Replace(bytesDecodeFull(),
		"\n    on encoding.invalid_utf8 e => forward e", "", 1)
	dir := writeLSPDir(t, map[string]string{"m.can": noErr})
	diags := diagnose(dir, "m.can", noErr)
	if !hasErrCode(diags, CodeMissingArm) || !hasDiag(diags, "error", "non-exhaustive match, missing") {
		t.Fatalf("expected missing-arm rejection without error arm, got %v", diags)
	}
	noOk := strings.Replace(bytesDecodeFull(),
		"    on Ok r => Ok(r.value)\n", "", 1)
	dir = writeLSPDir(t, map[string]string{"m.can": noOk})
	diags = diagnose(dir, "m.can", noOk)
	if !hasErrCode(diags, CodeMissingArm) || !hasDiag(diags, "error", "non-exhaustive match, missing") {
		t.Fatalf("expected missing-arm rejection without Ok arm, got %v", diags)
	}
}

// D2: a stale arm naming a declared unrelated error refuses.
func TestBytesD2StaleArm(t *testing.T) {
	body := strings.Replace(bytesDecodeFull(),
		"    on Ok r => Ok(r.value)",
		"    on Ok r => Ok(r.value)\n    on m.boom e2 => Ok(\"\")", 1)
	body = strings.Replace(body, "fn m__go(value: Bytes)",
		"error m.boom(value: str)\n\nfn m__go(value: Bytes)", 1)
	dir := writeLSPDir(t, map[string]string{"m.can": body})
	diags := diagnose(dir, "m.can", body)
	if !hasErrCode(diags, CodeStaleArm) || !hasDiag(diags, "error", "stale match arm m.boom") {
		t.Fatalf("expected stale-arm rejection, got %v", diags)
	}
}

// D3: the deterministic kernel takes no given table.
func TestBytesD3NoGiven(t *testing.T) {
	body := strings.Replace(bytesDecodeFull(),
		"  match call bytes__utf8__decode(value)\n    on Ok r => Ok(r.value)",
		"  match call bytes__utf8__decode(value)\n    given\n      empty => [exchange args (value = Bytes(Seq<int>[])) outcome Ok(\"\")]\n    on Ok r => Ok(r.value)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeGivenOnLocal, "no given table")
}

// D4: strict Bytes admission: str, int, and str-branded inputs
// refuse naming the wanted type (no new admission path). Rows
// match their params; only the kernel call mismatches.
func TestBytesD4Admission(t *testing.T) {
	mk := func(param, arg string) string {
		s := `mod m
  provides [m__go]
  uses []
  emits [encoding.invalid_utf8]

fn m__go(value: PARAM) -> Encoding__Text rev 1
  emits [encoding.invalid_utf8]
  tests
    go(ARG) => Ok("A")
  match call bytes__utf8__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_utf8 e => encoding.invalid_utf8(value = e.value)
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

// D5: the decode contract exists explicitly: EmitsOf entry plus
// the compiler-owned error registration (fields ["value"]).
func TestBytesD5ContractsRegistered(t *testing.T) {
	dir := writeLSPDir(t, map[string]string{"m.can": bytesDecodeFull()})
	mods, texts, _, err := legacyParsePaths([]string{dir + "/m.can"})
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := buildWorld(mods[0], mods, texts)
	emits, ok := prog.EmitsOf["bytes__utf8__decode"]
	if !ok {
		t.Fatalf("missing EmitsOf entry for bytes__utf8__decode")
	}
	if len(emits) != 1 || emits[0] != "encoding.invalid_utf8" {
		t.Fatalf("EmitsOf[decode] = %v, want [encoding.invalid_utf8]", emits)
	}
	fs, ok := prog.Errors["encoding.invalid_utf8"]
	if !ok || len(fs) != 1 || fs[0] != "value" {
		t.Fatalf("prog.Errors[invalid_utf8] = %v, want [value]", fs)
	}
}

// D6: the decoder lowers through fatal TextDecoder with BOM
// preservation, and the error member carries Uint8Array.
func TestBytesD6EmitPins(t *testing.T) {
	ts := compileEmit(t, bytesDecodeFull())
	for _, want := range []string{
		"fatal: true",
		"ignoreBOM: true",
		`{ $can_kind: "encoding.invalid_utf8"; value: Uint8Array }`,
	} {
		if !strings.Contains(ts, want) {
			t.Fatalf("emit missing %q:\n%s", want, ts)
		}
	}
}

// D7: source cannot redefine compiler-owned contracts, even
// identically: record, error, and brand shadows all refuse.
func TestBytesD7ShadowRejections(t *testing.T) {
	trec := strings.Replace(bytesDecodeFull(), "fn m__go(value: Bytes)",
		"type Encoding__Text rev 1 (\n  value: str\n)\n\nfn m__go(value: Bytes)", 1)
	trec = strings.Replace(trec, "provides [m__go]", "provides [m__go, Encoding__Text]", 1)
	seqCode(t, map[string]string{"m.can": trec}, "m.can",
		CodePrimitiveShadow, "shadows a compiler-owned record")
	terr := strings.Replace(bytesDecodeFull(), "fn m__go(value: Bytes)",
		"error encoding.invalid_utf8(value: Bytes)\n\nfn m__go(value: Bytes)", 1)
	seqCode(t, map[string]string{"m.can": terr}, "m.can",
		CodePrimitiveShadow, "shadows a compiler-owned error")
	tbrand := strings.Replace(bytesDecodeFull(), "fn m__go(value: Bytes)",
		"brand Encoding__Text is str rev 1\n\nfn m__go(value: Bytes)", 1)
	tbrand = strings.Replace(tbrand, "provides [m__go]", "provides [m__go, Encoding__Text]", 1)
	seqCode(t, map[string]string{"m.can": tbrand}, "m.can",
		CodePrimitiveShadow, "shadows a compiler-owned record")
}

// D8: the builtin error path is typed end-to-end: a str payload
// construction refuses, and a wrong-payload expectation fails.
func TestBytesD8ErrorPathTyped(t *testing.T) {
	wrongCtor := strings.Replace(bytesDecodeFull(),
		"on encoding.invalid_utf8 e => forward e",
		`on encoding.invalid_utf8 e => encoding.invalid_utf8(value = "nope")`, 1)
	seqCode(t, map[string]string{"m.can": wrongCtor}, "m.can",
		CodeTypeMismatch, "want Bytes")
	wrongPay := strings.Replace(bytesDecodeFull(),
		"mid_bad(Bytes(Seq<int>[65, 255, 66])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 255, 66]))",
		"mid_bad(Bytes(Seq<int>[65, 255, 66])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 255]))", 1)
	seqCode(t, map[string]string{"m.can": wrongPay}, "m.can",
		CodeTestFailed, "mid_bad")
}

const bytesDecodeProv = `mod prov
  provides [prov__go]
  uses []
  emits [encoding.invalid_utf8]

fn prov__go(value: Bytes) -> Encoding__Text rev 1
  emits [encoding.invalid_utf8]
  tests
    good(Bytes(Seq<int>[65])) => Ok("A")
    bad(Bytes(Seq<int>[65, 226, 130])) => encoding.invalid_utf8(value = Bytes(Seq<int>[65, 226, 130]))
  match call bytes__utf8__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_utf8 e => encoding.invalid_utf8(value = e.value)
`

const bytesDecodeLie = `mod client
  provides [client__use]
  uses [prov__go@1]
  emits [encoding.invalid_utf8]

fn client__use() -> Encoding__Text rev 1
  emits [encoding.invalid_utf8]
  tests
    lie() => Ok("A")
  match call prov__go(Bytes(Seq<int>[65, 226, 130]))
    given
      lie => [exchange args (value = Bytes(Seq<int>[65, 226, 130])) outcome Ok("A")]
    on Ok r => Ok(r.value)
    on encoding.invalid_utf8 e => encoding.invalid_utf8(value = e.value)
`

// D9: invalid UTF-8 is a computed comparable result, not a
// modeling gap: a false scripted success contradicts (CAN3110)
// under both module orders. A Go-error implementation would let
// this lie pass as "not contradicted".
func TestBytesD9ContradictionBothOrders(t *testing.T) {
	files := map[string]string{"prov.can": bytesDecodeProv, "client.can": bytesDecodeLie}
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

// D10: the catalog attributes the intrinsic's declared failure
// to the compiler kernel, with the owned field list.
func TestBytesD10CatalogAttribution(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "m.can")
	if err := os.WriteFile(srcPath, []byte(bytesDecodeProv), 0o644); err != nil {
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
	got, ok := byKind["encoding.invalid_utf8"]
	if !ok {
		t.Fatalf("catalog omits encoding.invalid_utf8, got %v", entries)
	}
	if len(got.Fields) != 1 || got.Fields[0] != "value" {
		t.Fatalf("catalog fields = %v, want [value]", got.Fields)
	}
	found := false
	for _, r := range got.RaisedBy {
		if r == "kernel.bytes__utf8__decode" {
			found = true
		}
	}
	if !found {
		t.Fatalf("catalog raised_by lacks kernel attribution, got %v", got.RaisedBy)
	}
}
