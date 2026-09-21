package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// a59 B14: base64 decode kernel. bytes__base64__decode is the third
// fallible kernel: str in, Bytes__Value on success,
// encoding.invalid_base64 (original string, unchanged) on malformed
// input. F-rows are the acceptance rows (verdict v2: corrected
// 4-bit/2-bit masks, per-lie CAN3110, mixed probe module).

const bytesB64DecodeBase = `mod m
  provides [m__go]
  uses []
  emits [encoding.invalid_base64]

fn m__go(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_base64]
  tests
    empty("") => Ok(Bytes(Seq<int>[]))
    one_pad("QQ==") => Ok(Bytes(Seq<int>[65]))
    two_pad("QUI=") => Ok(Bytes(Seq<int>[65, 66]))
    full("QUJD") => Ok(Bytes(Seq<int>[65, 66, 67]))
    foo("Zm9v") => Ok(Bytes(Seq<int>[102, 111, 111]))
    stdpair("+/8=") => Ok(Bytes(Seq<int>[251, 255]))
    unpadded4("AAAB") => Ok(Bytes(Seq<int>[0, 0, 1]))
    slashes("////") => Ok(Bytes(Seq<int>[255, 255, 255]))
    hexchars("0x00") => Ok(Bytes(Seq<int>[211, 29, 52]))
    long("QUJDQUJD") => Ok(Bytes(Seq<int>[65, 66, 67, 65, 66, 67]))
NULROW
NONASCIIROWS
    odd_one("Q") => encoding.invalid_base64(value = "Q")
    odd_three("QUJDQ") => encoding.invalid_base64(value = "QUJDQ")
    badmod("ABCDE") => encoding.invalid_base64(value = "ABCDE")
    lead_pad("=QUI") => encoding.invalid_base64(value = "=QUI")
    mid_pad("Q=Q=") => encoding.invalid_base64(value = "Q=Q=")
    excess_pad("QUJD====") => encoding.invalid_base64(value = "QUJD====")
    pad_prefix("QQ==AAAA") => encoding.invalid_base64(value = "QQ==AAAA")
    pad_prefix2("QUI=AAAA") => encoding.invalid_base64(value = "QUI=AAAA")
    bad_four_bits("AE==") => encoding.invalid_base64(value = "AE==")
    bad_two_bits("QUJ=") => encoding.invalid_base64(value = "QUJ=")
    strict_lie("QUJDQUJ=") => encoding.invalid_base64(value = "QUJDQUJ=")
    sp_lead(" QUJDQUJ") => encoding.invalid_base64(value = " QUJDQUJ")
    sp_trail("QUJDQUJ ") => encoding.invalid_base64(value = "QUJDQUJ ")
    sp_mid("QUJD UJD") => encoding.invalid_base64(value = "QUJD UJD")
    pad_gap("QQ= ==") => encoding.invalid_base64(value = "QQ= ==")
    url_pair("-_8=") => encoding.invalid_base64(value = "-_8=")
    url_short("-_") => encoding.invalid_base64(value = "-_")
    at_sign("QU@D") => encoding.invalid_base64(value = "QU@D")
    bracket("QU[D") => encoding.invalid_base64(value = "QU[D")
    backtick("QU` + "`" + `D") => encoding.invalid_base64(value = "QU` + "`" + `D")
    brace("QU{D") => encoding.invalid_base64(value = "QU{D")
    fidelity("QUJD!!!") => encoding.invalid_base64(value = "QUJD!!!")
    fidelity8("QUJD!!!!") => encoding.invalid_base64(value = "QUJD!!!!")
    trunc("QQ") => encoding.invalid_base64(value = "QQ")
  match call bytes__base64__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_base64 e => forward e
`

func bytesB64DecodeFull() string {
	nul := "    nul_mid(\"A\x00I=\") => encoding.invalid_base64(value = \"A\x00I=\")\n"
	nonascii := "    e_acute(\"é0AA\") => encoding.invalid_base64(value = \"é0AA\")\n" +
		"    astral(\"AA😀\") => encoding.invalid_base64(value = \"AA😀\")\n" +
		"    bom(\"QUJ\uFEFF\") => encoding.invalid_base64(value = \"QUJ\uFEFF\")\n"
	out := strings.Replace(bytesB64DecodeBase, "NULROW\n", nul, 1)
	return strings.Replace(out, "NONASCIIROWS\n", nonascii, 1)
}

// F0: base64 decode vectors, both outcomes, positional + named.
// Mask matrix (AE/QUI vs QUJ, AAAB ////), fidelity pair, even
// shapes throughout so length never masks alphabet checks.
func TestBytesF0B64DecodeVectors(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesB64DecodeFull()}, "m.can")
	named := strings.Replace(bytesB64DecodeFull(),
		"match call bytes__base64__decode(value)",
		"match call bytes__base64__decode(value = value)", 1)
	// The named spelling evaluates identically but is a lint error
	// (CAN3410): exactly one finding, nothing else.
	seqCode(t, map[string]string{"m.can": named}, "m.can", CodeLintRedundant, "redundant argument name")
}

// F1: both arms are mandatory.
func TestBytesF1MissingArms(t *testing.T) {
	noErr := strings.Replace(bytesB64DecodeFull(),
		"\n    on encoding.invalid_base64 e => forward e", "", 1)
	dir := writeLSPDir(t, map[string]string{"m.can": noErr})
	diags := diagnose(dir, "m.can", noErr)
	if !hasErrCode(diags, CodeMissingArm) || !hasDiag(diags, "error", "non-exhaustive match, missing") {
		t.Fatalf("expected missing-arm rejection without error arm, got %v", diags)
	}
	noOk := strings.Replace(bytesB64DecodeFull(),
		"    on Ok r => Ok(r.value)\n", "", 1)
	dir = writeLSPDir(t, map[string]string{"m.can": noOk})
	diags = diagnose(dir, "m.can", noOk)
	if !hasErrCode(diags, CodeMissingArm) || !hasDiag(diags, "error", "non-exhaustive match, missing") {
		t.Fatalf("expected missing-arm rejection without Ok arm, got %v", diags)
	}
}

// F2: a stale arm naming a declared unrelated error refuses.
func TestBytesF2StaleArm(t *testing.T) {
	body := strings.Replace(bytesB64DecodeFull(),
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

// F3: the deterministic kernel takes no given table.
func TestBytesF3NoGiven(t *testing.T) {
	body := strings.Replace(bytesB64DecodeFull(),
		"  match call bytes__base64__decode(value)\n    on Ok r => Ok(r.value)",
		"  match call bytes__base64__decode(value)\n    given\n      empty => [exchange args (value = \"\") outcome Ok(Bytes(Seq<int>[]))]\n    on Ok r => Ok(r.value)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodeGivenOnLocal, "no given table")
}

// F4: strict str admission, through BOTH arms (brands erase at
// runtime, so brand fixtures need an error-arm row each).
func TestBytesF4Admission(t *testing.T) {
	mk := func(param, arg string) string {
		s := `mod m
  provides [m__go]
  uses []
  emits [encoding.invalid_base64]

fn m__go(value: PARAM) -> Bytes__Value rev 1
  emits [encoding.invalid_base64]
  tests
    go(ARG) => Ok(Bytes(Seq<int>[65, 66, 67]))
  match call bytes__base64__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_base64 e => encoding.invalid_base64(value = e.value)
`
		s = strings.Replace(s, "PARAM", param, 1)
		return strings.Replace(s, "ARG", arg, 1)
	}
	seqCode(t, map[string]string{"m.can": mk("Bytes", "Bytes(Seq<int>[65])")}, "m.can",
		CodeTypeMismatch, "want str")
	seqCode(t, map[string]string{"m.can": mk("int", "3")}, "m.can",
		CodeTypeMismatch, "want str")
	goodBrand := strings.Replace(mk("M__Secret", `seal M__Secret("QUJD")`),
		"fn m__go(value: M__Secret)", "brand M__Secret is str rev 1\n\nfn m__go(value: M__Secret)", 1)
	goodBrand = strings.Replace(goodBrand, "provides [m__go]", "provides [M__Secret, m__go]", 1)
	goodBrand = strings.Replace(goodBrand,
		`    go(seal M__Secret("QUJD")) => Ok(Bytes(Seq<int>[65, 66, 67]))`,
		"    go(seal M__Secret(\"QUJD\")) => Ok(Bytes(Seq<int>[65, 66, 67]))\n    bad(value = seal M__Secret(\"!!!\")) => encoding.invalid_base64(value = \"!!!\")", 1)
	seqCode(t, map[string]string{"m.can": goodBrand}, "m.can",
		CodeTypeMismatch, "want str")
	badBrand := strings.Replace(mk("M__Secret", `seal M__Secret("!!!")`),
		"fn m__go(value: M__Secret)", "brand M__Secret is str rev 1\n\nfn m__go(value: M__Secret)", 1)
	badBrand = strings.Replace(badBrand, "provides [m__go]", "provides [M__Secret, m__go]", 1)
	badBrand = strings.Replace(badBrand,
		`    go(seal M__Secret("!!!")) => Ok(Bytes(Seq<int>[65, 66, 67]))`,
		"    go(seal M__Secret(\"!!!\")) => Ok(Bytes(Seq<int>[65, 66, 67]))\n    bad(value = seal M__Secret(\"QUJD\")) => Ok(Bytes(Seq<int>[65, 66, 67]))", 1)
	seqCode(t, map[string]string{"m.can": badBrand}, "m.can",
		CodeTypeMismatch, "want str")
}

// F5: the decode contract exists explicitly: EmitsOf entry plus
// the compiler-owned error registration (fields ["value"]).
func TestBytesF5ContractsRegistered(t *testing.T) {
	dir := writeLSPDir(t, map[string]string{"m.can": bytesB64DecodeFull()})
	mods, texts, _, err := legacyParsePaths([]string{dir + "/m.can"})
	if err != nil {
		t.Fatal(err)
	}
	prog, _ := buildWorld(mods[0], mods, texts)
	emits, ok := prog.EmitsOf["bytes__base64__decode"]
	if !ok {
		t.Fatalf("missing EmitsOf entry for bytes__base64__decode")
	}
	if len(emits) != 1 || emits[0] != "encoding.invalid_base64" {
		t.Fatalf("EmitsOf[b64__decode] = %v, want [encoding.invalid_base64]", emits)
	}
	fs, ok := prog.Errors["encoding.invalid_base64"]
	if !ok || len(fs) != 1 || fs[0] != "value" {
		t.Fatalf("prog.Errors[invalid_base64] = %v, want [value]", fs)
	}
}

// F6: the base64 decoder lowers through its own helper and union;
// a base64-only fixture touches no text codec, no atob, no btoa.
func TestBytesF6EmitPins(t *testing.T) {
	ts := compileEmit(t, bytesB64DecodeFull())
	for _, want := range []string{
		"$canB64Decode(",
		`{ $can_kind: "ok"; value: Uint8Array } | { $can_kind: "encoding.invalid_base64"; value: string }`,
	} {
		if !strings.Contains(ts, want) {
			t.Fatalf("emit missing %q:\n%s", want, ts)
		}
	}
	for _, banned := range []string{"TextEncoder", "TextDecoder", "$canUtf8Decode(", "$canHexDecode(", "atob(", "btoa("} {
		if strings.Contains(ts, banned) {
			t.Fatalf("base64-only emit must not contain %q:\n%s", banned, ts)
		}
	}
}

// F7: source cannot redeclare the compiler-owned error, even
// identically.
func TestBytesF7ShadowRejection(t *testing.T) {
	body := strings.Replace(bytesB64DecodeFull(), "fn m__go(value: str)",
		"error encoding.invalid_base64(value: str)\n\nfn m__go(value: str)", 1)
	seqCode(t, map[string]string{"m.can": body}, "m.can",
		CodePrimitiveShadow, "shadows a compiler-owned error")
}

// F8: the builtin error path is typed end-to-end: a Bytes payload
// construction refuses, and a wrong-payload expectation fails.
func TestBytesF8ErrorPathTyped(t *testing.T) {
	wrongCtor := strings.Replace(bytesB64DecodeFull(),
		"on encoding.invalid_base64 e => forward e",
		`on encoding.invalid_base64 e => encoding.invalid_base64(value = Bytes(Seq<int>[65]))`, 1)
	seqCode(t, map[string]string{"m.can": wrongCtor}, "m.can",
		CodeTypeMismatch, "want str")
	wrongPay := strings.Replace(bytesB64DecodeFull(),
		`fidelity("QUJD!!!") => encoding.invalid_base64(value = "QUJD!!!")`,
		`fidelity("QUJD!!!") => encoding.invalid_base64(value = "QUJD!!!!")`, 1)
	seqCode(t, map[string]string{"m.can": wrongPay}, "m.can",
		CodeTestFailed, "fidelity")
}

const bytesB64DecodeProv = `mod prov
  provides [prov__go]
  uses []
  emits [encoding.invalid_base64]

fn prov__go(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_base64]
  tests
    good("QUJD") => Ok(Bytes(Seq<int>[65, 66, 67]))
    bad("QUJDQUJ=") => encoding.invalid_base64(value = "QUJDQUJ=")
  match call bytes__base64__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_base64 e => encoding.invalid_base64(value = e.value)
`

const bytesB64DecodeLie = `mod client
  provides [client__use]
  uses [prov__go@1]
  emits [encoding.invalid_base64]

fn client__use(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_base64]
  tests
    prefixlie("QUJD!!!") => Ok(Bytes(Seq<int>[65, 66, 67]))
    padlie("QUJDQUJ=") => Ok(Bytes(Seq<int>[65, 66, 67, 65, 66]))
  match call prov__go(value)
    given
      prefixlie => [exchange args (value = "QUJD!!!") outcome Ok(Bytes(Seq<int>[65, 66, 67]))]
      padlie => [exchange args (value = "QUJDQUJ=") outcome Ok(Bytes(Seq<int>[65, 66, 67, 65, 66]))]
    on Ok r => Ok(r.value)
    on encoding.invalid_base64 e => encoding.invalid_base64(value = e.value)
`

// F9: partial results never escape as success, and unchecked pad
// bits never pass: EACH lie asserts its OWN CAN3110 in both
// module orders (no aggregate check).
func TestBytesF9ContradictionBothOrders(t *testing.T) {
	files := map[string]string{"prov.can": bytesB64DecodeProv, "client.can": bytesB64DecodeLie}
	for _, order := range [][]string{{"prov.can", "client.can"}, {"client.can", "prov.can"}} {
		diags := checkTwo(t, order, files)
		if !hasDiag(diags, "error", "script prefixlie contradicts prov__go") {
			t.Fatalf("order %v: expected prefixlie CAN3110, got %v", order, diags)
		}
		if !hasDiag(diags, "error", "script padlie contradicts prov__go") {
			t.Fatalf("order %v: expected padlie CAN3110, got %v", order, diags)
		}
	}
}

// F10: the catalog attributes the intrinsic's declared failure
// to the compiler kernel, with the owned field list.
func TestBytesF10CatalogAttribution(t *testing.T) {
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "m.can")
	if err := os.WriteFile(srcPath, []byte(bytesB64DecodeProv), 0o644); err != nil {
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
	got, ok := byKind["encoding.invalid_base64"]
	if !ok {
		t.Fatalf("catalog omits encoding.invalid_base64, got %v", entries)
	}
	if len(got.Fields) != 1 || got.Fields[0] != "value" {
		t.Fatalf("catalog fields = %v, want [value]", got.Fields)
	}
	found := false
	for _, r := range got.RaisedBy {
		if r == "kernel.bytes__base64__decode" {
			found = true
		}
	}
	if !found {
		t.Fatalf("catalog raised_by lacks kernel attribution, got %v", got.RaisedBy)
	}
}

// F11: the mixed probe chains base64 decode into UTF-8 decode
// (verdict's exact module, test-only — never stdlib). "/w=="
// must survive base64 as [255] and fail UTF-8 after;
// "QUJD!!!" must fail base64 without forwarding.
const bytesB64MixedProbe = `mod probe
  provides [probe__base64]
  uses []
  emits [encoding.invalid_base64]

fn probe__base64(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_base64]
  tests
    empty("") => Ok(Bytes(Seq<int>[]))
    one_byte("AA==") => Ok(Bytes(Seq<int>[0]))
    two_bytes("QUI=") => Ok(Bytes(Seq<int>[65, 66]))
    bad_four_bits("AE==") => encoding.invalid_base64(value = "AE==")
    bad_two_bits("QUJ=") => encoding.invalid_base64(value = "QUJ=")
    bad_final_quartet("QUJDQUJ=") => encoding.invalid_base64(value = "QUJDQUJ=")
  match call bytes__base64__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_base64 e => forward e
`

const bytesB64ChainProbe = `mod chain
  provides [chain__text]
  uses []
  emits [encoding.invalid_base64, encoding.invalid_utf8]

fn chain__text(value: str) -> Encoding__Text rev 1
  emits [encoding.invalid_base64, encoding.invalid_utf8]
  tests
    ascii("QQ==") => Ok("A")
    bad_bytes("/w==") => encoding.invalid_utf8(value = Bytes(Seq<int>[255]))
    bad_b64("QUJD!!!") => encoding.invalid_base64(value = "QUJD!!!")
    bad_pad("QUJDQUJ=") => encoding.invalid_base64(value = "QUJDQUJ=")
  match call bytes__base64__decode(value)
    on Ok b => match call bytes__utf8__decode(b.value)
      on Ok t => Ok(t.value)
      on encoding.invalid_utf8 e => forward e
    on encoding.invalid_base64 e => forward e
`

// F11a: both probe modules are clean: every arm witnessed.
func TestBytesF11aMixedProbesClean(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesB64MixedProbe}, "m.can")
	seqClean(t, map[string]string{"m.can": bytesB64ChainProbe}, "m.can")
}

// F11b: per-call unions and helpers are exact where base64 and
// hex decoders coexist: the "4142" discriminator (hex [65,66]
// vs base64 [227,94,54]) proves correct helper selection beyond
// co-presence.
const bytesB64HexCoexist = `mod m
  provides [m__hex, m__b64]
  uses []
  emits [encoding.invalid_base64, encoding.invalid_hex]

fn m__hex(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_hex]
  tests
    disc("4142") => Ok(Bytes(Seq<int>[65, 66]))
    bad("zz") => encoding.invalid_hex(value = "zz")
  match call bytes__hex__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_hex e => forward e

fn m__b64(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_base64]
  tests
    disc("4142") => Ok(Bytes(Seq<int>[227, 94, 54]))
    bad("!!!") => encoding.invalid_base64(value = "!!!")
  match call bytes__base64__decode(value)
    on Ok r => Ok(r.value)
    on encoding.invalid_base64 e => forward e
`

func TestBytesF11bCoexistEmitPins(t *testing.T) {
	seqClean(t, map[string]string{"m.can": bytesB64HexCoexist}, "m.can")
	ts := compileEmit(t, bytesB64HexCoexist)
	for _, want := range []string{
		"$canHexDecode(",
		"$canB64Decode(",
		`{ $can_kind: "ok"; value: Uint8Array } | { $can_kind: "encoding.invalid_hex"; value: string }`,
		`{ $can_kind: "ok"; value: Uint8Array } | { $can_kind: "encoding.invalid_base64"; value: string }`,
	} {
		if !strings.Contains(ts, want) {
			t.Fatalf("coexist emit missing %q:\n%s", want, ts)
		}
	}
	if strings.Contains(ts, "value: string | Uint8Array") {
		t.Fatalf("coexist emit must not merge payload types:\n%s", ts)
	}
}

const bytesB64MixedLie = `mod client
  provides [client__use]
  uses [probe__base64@1]
  emits [encoding.invalid_base64]

fn client__use(value: str) -> Bytes__Value rev 1
  emits [encoding.invalid_base64]
  tests
    strictlie("QUJDQUJ=") => Ok(Bytes(Seq<int>[65, 66, 67, 65, 66]))
  match call probe__base64(value)
    given
      strictlie => [exchange args (value = "QUJDQUJ=") outcome Ok(Bytes(Seq<int>[65, 66, 67, 65, 66]))]
    on Ok r => Ok(r.value)
    on encoding.invalid_base64 e => forward e
`

// F11c: the strict lie (plausible bytes, rejected string)
// contradicts under both module orders.
func TestBytesF11cMixedContradiction(t *testing.T) {
	files := map[string]string{"probe.can": bytesB64MixedProbe, "client.can": bytesB64MixedLie}
	for _, order := range [][]string{{"probe.can", "client.can"}, {"client.can", "probe.can"}} {
		diags := checkTwo(t, order, files)
		if !hasDiag(diags, "error", "script strictlie contradicts probe__base64") {
			t.Fatalf("order %v: expected strictlie CAN3110, got %v", order, diags)
		}
	}
}
