package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const canonicalFactory = `mod acc
  provides [acc__a, acc__b, acc__factory]
  uses []
  emits []
fn acc__a(value: int, offset: int) -> int rev 1
  emits []
  tests
    one(1, 0) => Ok(1)
  Ok(value + offset)
fn acc__b(value: int, offset: int) -> int rev 1
  emits []
  tests
    one(1, 0) => Ok(2)
  Ok(value + offset + 1)
fn acc__factory() -> Fn<int, int, []> rev 1
  emits []
  tests
    one() => Ok(fnref acc__a(offset = 0)) pinned
  Ok(fnref acc__a(offset = 0))
`

func canonicalProgram(t *testing.T, source string) (*Program, map[string]string) {
	t.Helper()
	return revisionProg(t, map[string]string{"acc.can": source}, []string{"acc.can"})
}

// Both candidate programs pass their own tests. Updating the factory AND its
// pinned expectation must still warn against the independently accepted row.
func TestCanonicalPinnedCallbackChanges(t *testing.T) {
	old, _ := canonicalProgram(t, canonicalFactory)
	base := acceptanceBase(t, old)
	for name, source := range map[string]string{
		"target":   strings.ReplaceAll(canonicalFactory, "fnref acc__a(", "fnref acc__b("),
		"capture":  strings.ReplaceAll(canonicalFactory, "offset = 0", "offset = 1"),
		"revision": strings.Replace(canonicalFactory, "offset: int) -> int rev 1", "offset: int) -> int rev 2", 1),
	} {
		t.Run(name, func(t *testing.T) {
			prog, texts := canonicalProgram(t, source)
			if name != "revision" {
				if ds := CheckRevisionIdentity(prog, texts, base); len(ds) != 0 {
					t.Fatalf("body/expectation changes must reach the pin check: %+v", ds)
				}
				if ds := diagnoseWith(t.TempDir(), "acc.can", source, base); !hasCode(ds, CodePinnedWeakened) {
					t.Fatalf("LSP missed callback weakening: %+v", ds)
				}
			}
			ds := CheckPinnedRows(prog, texts, base)
			if len(ds) != 1 || ds[0].Code != CodePinnedWeakened || ds[0].Sev != "warning" {
				t.Fatalf("expected acceptance warning, got %+v", ds)
			}
			if ds[0].Expected == ds[0].Found {
				t.Fatal("diagnostic must show distinct evidence")
			}
		})
	}
}

func TestCanonicalIdentitySeparationAndFormatting(t *testing.T) {
	p, _ := canonicalProgram(t, canonicalFactory)
	base := acceptanceBase(t, p)
	formatted := strings.ReplaceAll(canonicalFactory, "Fn<int, int, []>", "Fn<int,int,[]>")
	formatted = strings.ReplaceAll(formatted, "offset = 0", "offset=0")
	formatted = strings.ReplaceAll(formatted, "  Ok(", "  // presentation only\n  Ok(")
	q, texts := canonicalProgram(t, formatted)
	if ds := CheckRevisionIdentity(q, texts, base); len(ds) != 0 {
		t.Fatal(ds)
	}
	if ds := CheckPinnedRows(q, texts, base); len(ds) != 0 {
		t.Fatal(ds)
	}
	if !reflect.DeepEqual(PinnedRows(p), PinnedRows(q)) {
		t.Fatal("formatting changed acceptance evidence")
	}
	// A behavior change with unchanged interface is executable churn, not an
	// interface change or an edit of the factory's promised callable identity.
	body := strings.Replace(canonicalFactory, "one(1, 0) => Ok(1)", "one(1, 0) => Ok(3)", 1)
	body = strings.Replace(body, "Ok(value + offset)", "Ok(value + offset + 2)", 1)
	r, _ := canonicalProgram(t, body)
	key := revisionKey("fn", "acc", "acc__a", 1, true)
	if FingerprintProgram(p)[key].Fingerprint != FingerprintProgram(r)[key].Fingerprint {
		t.Fatal("body entered interface fingerprint")
	}
	if canonNode(p.Fns["acc__a"].Body) == canonNode(r.Fns["acc__a"].Body) {
		t.Fatal("body change absent from executable structure")
	}
	if !reflect.DeepEqual(PinnedRows(p), PinnedRows(r)) {
		t.Fatal("implementation became acceptance expectation")
	}
}

func TestCanonicalCallbackDependencyClosure(t *testing.T) {
	source := strings.Replace(canonicalFactory, "  provides [", "  provides [Acc__Value, ", 1)
	source = strings.Replace(source, "fn acc__a", "type Acc__Value rev 1 (\n  value: int\n)\nfn acc__a", 1)
	source = strings.Replace(source, "offset: int) -> int rev 1", "offset: int) -> Acc__Value rev 1", 1)
	source = strings.Replace(source, "Fn<int, int, []>", "Fn<int, Acc__Value, []>", 1)
	p, _ := canonicalProgram(t, source)
	base := acceptanceBase(t, p)
	q, texts := canonicalProgram(t, strings.Replace(source, "type Acc__Value rev 1", "type Acc__Value rev 2", 1))
	key := revisionKey("fn", "acc", "acc__a", 1, true)
	a, b := FingerprintProgram(p)[key], FingerprintProgram(q)[key]
	if a.Own != b.Own || reflect.DeepEqual(a.DepPrints, b.DepPrints) {
		t.Fatal("exact dependency identity must change independently of own interface")
	}
	if ds := CheckPinnedRows(q, texts, base); len(ds) != 1 || ds[0].Code != CodePinnedWeakened {
		t.Fatalf("referenced payload dependency changed silently: %+v", ds)
	}
	shape := strings.Replace(source, "  value: int\n)", "  value: int\n  flag: bool\n)", 1)
	shape = strings.Replace(shape, "one(1, 0) => Ok(1)", "one(1, 0) => Ok(1, true)", 1)
	shape = strings.Replace(shape, "Ok(value + offset)", "Ok(value + offset, true)", 1)
	r, rt := canonicalProgram(t, shape)
	c := FingerprintProgram(r)[key]
	if a.Own != c.Own || a.Fingerprint == c.Fingerprint {
		t.Fatal("payload shape must affect closure but not own interface")
	}
	if ds := CheckPinnedRows(r, rt, base); !hasCode(ds, CodePinnedWeakened) {
		t.Fatalf("payload schema change lost from acceptance: %+v", ds)
	}
}

func TestCanonicalStampedCallbackEvidence(t *testing.T) {
	source := strings.Replace(canonicalFactory, `fn acc__a(value: int, offset: int) -> int rev 1
  emits []
  tests
    one(1, 0) => Ok(1)
  Ok(value + offset)`, `fn acc__a<T>(value: int, offset: T) -> int rev 1
  emits []
  tests
    one<T=int>(1, 0) => Ok(1)
    flag<T=bool>(1, false) => Ok(1)
  Ok(value)`, 1)
	source = strings.ReplaceAll(source, "fnref acc__a(offset = 0)", "fnref acc__a<int>(offset = 0)")
	p, _ := canonicalProgram(t, source)
	base := acceptanceBase(t, p)
	candidate := strings.ReplaceAll(source, "fnref acc__a<int>(offset = 0)", "fnref acc__a<bool>(offset = false)")
	q, texts := canonicalProgram(t, candidate)
	ds := CheckPinnedRows(q, texts, base)
	if len(ds) != 1 || !strings.Contains(ds[0].Expected, "acc__a$T$int") || !strings.Contains(ds[0].Found, "acc__a$T$bool") {
		t.Fatalf("specialized identities missing: %+v", ds)
	}
}

func TestCanonicalSemanticInputs(t *testing.T) {
	parse := func(s string) *Small {
		t.Helper()
		v, err := parseSmall(s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	for _, pair := range [][2]string{
		{"fnref f<int>(x = 1)", "fnref f<str>(x = 1)"},
		{"call f<int>(1)", "call f<str>(1)"},
		{"Ok<int>(value)", "Ok<str>(value)"},
		{"Thing(pos = 1)", "Thing(1)"},
		{"fnref f(x = 1, y = 2)", "fnref f(y = 2, x = 1)"},
	} {
		if canonSmall(parse(pair[0])) == canonSmall(parse(pair[1])) {
			t.Fatalf("collision: %v", pair)
		}
	}
	a, _ := parsePattern("Ok<int> r")
	b, _ := parsePattern("Ok<str> r")
	if canonPattern(a) == canonPattern(b) {
		t.Fatal("pattern annotations omitted")
	}
	n := &Node{IsMatch: true, Kind: MatchInvoke, Scruts: []*Small{parse("cb")}, InvokeArg: parse("1")}
	first := canonNode(n)
	n.InvokeArg = parse("2")
	if first == canonNode(n) {
		t.Fatal("invocation argument omitted")
	}
	// Checked/derived annotations and positions do not become source identity.
	x := parse("fnref f<int>(x = 1)")
	first = canonSmall(x)
	x.T, x.ExportBrand = "derived", "certificate"
	if first != canonSmall(x) {
		t.Fatal("derived annotations entered canonical source")
	}
}

func TestCanonicalRefusesUnsupportedStructure(t *testing.T) {
	for name, run := range map[string]func(){
		"expression": func() { canonSmall(&Small{Kind: "future-node"}) },
		"pattern":    func() { canonPattern(Pattern{Kind: "future-pattern"}) },
		"match":      func() { canonNode(&Node{IsMatch: true, Kind: MatchKind(255)}) },
		"chain":      func() { canonNode(&Node{IsMatch: true, Kind: MatchChain}) },
		"integer":    func() { canonSmall(&Small{Kind: "int"}) },
		"range":      func() { canonPattern(Pattern{Kind: "range", LoS: "unresolved"}) },
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if _, ok := recover().(canonicalError); !ok {
					t.Fatal("expected fail-closed canonical error")
				}
			}()
			run()
		})
	}
	p, texts := canonicalProgram(t, canonicalFactory)
	base := acceptanceBase(t, p)
	p.Fns["acc__factory"].Body.Small.Kind = "future-node"
	path := filepath.Join(t.TempDir(), "baseline.json")
	if err := WriteBaseline(path, p, "candidate"); err == nil {
		t.Fatal("unsupported body written")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("partial baseline file created")
	}
	if ds := CheckRevisionIdentity(p, texts, base); len(ds) != 1 || ds[0].Sev != "error" {
		t.Fatal(ds)
	}
	p.Fns["acc__factory"].Tests[0].Expected.Kind = "future-node"
	if ds := CheckPinnedRows(p, texts, base); len(ds) != 1 || ds[0].Sev != "error" {
		t.Fatal(ds)
	}
}

func TestCanonicalVersionAndAcceptanceAuthority(t *testing.T) {
	p, texts := canonicalProgram(t, canonicalFactory)
	base := acceptanceBase(t, p)
	base.Format = 1
	if ds := CheckRevisionIdentity(p, texts, base); len(ds) != 1 || !strings.Contains(ds[0].Msg, "unsupported revision format") {
		t.Fatal(ds)
	}
	base.Format = RevisionFormat
	base.Pinned = nil
	if ds := CheckRevisionIdentity(p, texts, base); len(ds) != 1 || !strings.Contains(ds[0].Msg, "missing pinned") {
		t.Fatal(ds)
	}
	path := filepath.Join(t.TempDir(), "candidate.json")
	if err := WriteBaseline(path, p, "review-required"); err != nil {
		t.Fatal(err)
	}
	candidate, err := LoadBaseline(path)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Accepted || candidate.Format != RevisionFormat || len(candidate.Pinned) != 1 {
		t.Fatalf("bad candidate: %+v", candidate)
	}
	if ds := CheckRevisionIdentity(p, texts, candidate); len(ds) != 1 || !strings.Contains(ds[0].Msg, "not an accepted") {
		t.Fatal(ds)
	}
	candidate.Accepted = true // deliberate review authority; never generation behavior
	if ds := CheckRevisionIdentity(p, texts, candidate); len(ds) != 0 {
		t.Fatal(ds)
	}
	if ds := CheckPinnedRows(p, texts, candidate); len(ds) != 0 {
		t.Fatal(ds)
	}
}
