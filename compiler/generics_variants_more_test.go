package main

import (
	"strings"
	"testing"
)

func TestGenericVariantMultipleParameters(t *testing.T) {
	src := `mod either
  provides [Either__Value, Either__Box, either__swap]
  uses []
  emits []

variant Either__Value<L,R> rev 1 (
  case Left(value: L)
  case Right(value: R)
)

type Either__Box<L,R> rev 1 (
  value: Either__Value<L,R>
)

fn either__swap<L,R>(value: Either__Value<L,R>) -> Either__Box<R,L> rev 1
  emits []
  tests
    left_num<L=int,R=str>(Either__Left<int,str>(7)) => Ok(Either__Right<str,int>(7))
    right_text<L=int,R=str>(Either__Right<int,str>("a")) => Ok(Either__Left<str,int>("a"))
    left_text<L=str,R=int>(Either__Left<str,int>("b")) => Ok(Either__Right<int,str>("b"))
    right_num<L=str,R=int>(Either__Right<str,int>(9)) => Ok(Either__Left<int,str>(9))
  match value
    on Either__Left<L,R> l => Ok(Either__Right<R,L>(l.value))
    on Either__Right<L,R> r => Ok(Either__Left<R,L>(r.value))
`
	_, _, ds := genericProgram(t, map[string]string{"either.can": src})
	if errs := genericErrs(ds); len(errs) != 0 {
		t.Fatal(errs)
	}
	dir := writeLSPDir(t, map[string]string{"either.can": src})
	if ds := diagnose(dir, "either.can", src); len(genericErrs(ds)) != 0 {
		t.Fatal(ds)
	}
	// Parsing alternatives must also protect the comma inside <L,R>.
	p, err := parseValuePattern("Either__Left<L,R> l | Either__Right<L,R> r")
	if err != nil || len(p.Alts) != 2 || len(p.Alts[1].TypeArgs) != 2 {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestGenericVariantClosedSequenceArgument(t *testing.T) {
	src := `mod m
  provides [m__choose, Opt__Value, M__Out]
  uses []
  emits []

variant Opt__Value<T> rev 1 (
  case Some(value: T)
  case None()
)

type M__Out rev 1 (
  value: Seq<int>
)

fn m__choose(value: Opt__Value<Seq<int>>) -> M__Out rev 1
  emits []
  tests
    some(Opt__Some<Seq<int>>(Seq<int>[1,2])) => Ok(Seq<int>[1,2])
    none(Opt__None<Seq<int>>()) => Ok(Seq<int>[])
  match value
    on Opt__Some<Seq<int>> s => Ok(s.value)
    on Opt__None<Seq<int>> _ => Ok(Seq<int>[])
`
	_, _, ds := genericProgram(t, map[string]string{"m.can": src})
	if errs := genericErrs(ds); len(errs) != 0 {
		t.Fatal(errs)
	}
}

func TestGenericVariantScopeLimits(t *testing.T) {
	src := genericOptionSource(t)
	for _, tt := range []struct{ old, new, want string }{
		{"variant Option__Value<T>", "variant Option__Value<T,U>", "never used in its fields"},
		{"variant Option__Value<T>", "variant Option<T>", "must match Domain__Name"},
	} {
		_, _, ds := genericProgram(t, map[string]string{"option.can": strings.Replace(src, tt.old, tt.new, 1)})
		if !hasDiag(ds, "error", tt.want) {
			t.Fatalf("want %s: %v", tt.want, ds)
		}
	}
	unused := `mod m
  provides [Opt__Value]
  uses []
  emits []

variant Opt__Value<T> rev 1 (
  case Some(value: T)
  case None()
)
`
	_, _, ds := genericProgram(t, map[string]string{"m.can": unused})
	if !hasDiag(ds, "error", "never instantiated") {
		t.Fatal(ds)
	}
}

func TestGenericVariantRevisionIdentity(t *testing.T) {
	src := genericOptionSource(t)
	old, _ := revisionProg(t, map[string]string{"option.can": src}, []string{"option.can"})
	base := &RevisionBaseline{Format: RevisionFormat, Origin: "generic-variant-test", Accepted: true,
		Scope: []string{"option"}, Entries: FingerprintProgram(old), Pinned: PinnedRows(old)}
	// Rename one case consistently so the new program is valid on its
	// own, but changes both stamped variant contracts at the same rev.
	changed := strings.ReplaceAll(src, "case None()", "case Absent()")
	changed = strings.ReplaceAll(changed, "Option__None", "Option__Absent")
	cur, texts := revisionProg(t, map[string]string{"option.can": changed}, []string{"option.can"})
	ds := CheckRevisionIdentity(cur, texts, base)
	for _, arg := range []string{"int", "str"} {
		if !hasFound(ds, "Option__Value$T$"+arg) {
			t.Fatalf("missing variant revision drift for %s: %v", arg, ds)
		}
	}
}

func TestGenericVariantStrayPatternArgs(t *testing.T) {
	// With no generic declarations, a stray pattern argument must still
	// enter expansion and fail, never silently become a monomorphic case.
	src := variantMatchMod(variantMatchTests, strings.Replace(variantMatchBody, "Login__Anonymous _", "Login__Anonymous<int> _", 1))
	_, _, ds := genericProgram(t, map[string]string{"m.can": src})
	if !hasDiag(ds, "error", "unknown type Login__Anonymous") {
		t.Fatal(ds)
	}
	for _, p := range []string{"bad.Error__Some<int> e", "Opt__Some<int>e", "Opt__Some<int> x.y"} {
		if _, err := parsePattern(p); err == nil {
			t.Fatalf("accepted %s", p)
		}
	}
}
